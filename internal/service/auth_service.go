package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/repository"
)

// tokenLength is the length of session tokens in bytes.
const tokenLength = 32

// AuthService handles authentication-related business logic.
type AuthService struct {
	userRepo       domain.UserRepository
	sessionRepo    domain.SessionRepository
	sessionDuration time.Duration
	logger         *zap.Logger
}

// AuthError represents an authentication error.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common auth errors
var (
	ErrInvalidCredentials = &AuthError{Message: "invalid email or password"}
	ErrSessionExpired     = &AuthError{Message: "session expired"}
	ErrUserNotFound       = &AuthError{Message: "user not found"}
)

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	sessionDuration time.Duration,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		sessionDuration: sessionDuration,
		logger:          logger,
	}
}

// LoginContext holds contextual information for login.
type LoginContext struct {
	IPAddress string
	UserAgent string
}

// Login authenticates a user and creates a session.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.Session, error) {
	return s.LoginWithContext(ctx, email, password, nil)
}

// LoginWithContext authenticates a user and creates a session with context info.
func (s *AuthService) LoginWithContext(ctx context.Context, email, password string, loginCtx *LoginContext) (*domain.Session, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("login attempt for non-existent user", zap.String("email", email))
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.CheckPassword(password) {
		s.logger.Warn("invalid password attempt", zap.String("email", email))
		return nil, ErrInvalidCredentials
	}

	// Generate session token
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	var session *domain.Session
	if loginCtx != nil {
		session = domain.NewSessionWithContext(user.ID, token, s.sessionDuration, loginCtx.IPAddress, loginCtx.UserAgent)
	} else {
		session = domain.NewSession(user.ID, token, s.sessionDuration)
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
	)

	return session, nil
}

// Logout invalidates a session.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if err := s.sessionRepo.Delete(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.logger.Info("user logged out")
	return nil
}

// SessionValidationResult contains the user and optionally a new session token.
type SessionValidationResult struct {
	User     *domain.User
	NewToken string // Set if token was rotated
}

// ValidateSession validates a session token and returns the associated user.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*domain.User, error) {
	result, err := s.ValidateAndRefreshSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return result.User, nil
}

// ValidateAndRefreshSession validates a session and refreshes/rotates if needed.
func (s *AuthService) ValidateAndRefreshSession(ctx context.Context, token string) (*SessionValidationResult, error) {
	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSessionExpired
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.IsExpired() {
		// Clean up expired session
		_ = s.sessionRepo.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	// Check if this is an old token being used during grace period
	usingOldToken := session.PreviousToken != nil && *session.PreviousToken == token && session.IsWithinGracePeriod()

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	result := &SessionValidationResult{User: user}

	// If using old token during grace period, return current token for client to update
	if usingOldToken {
		result.NewToken = session.Token
		s.logger.Debug("client using old token during grace period, returning new token",
			zap.String("user_id", user.ID.String()),
		)
		return result, nil
	}

	// Check if token should be rotated (every 15 minutes)
	if session.ShouldRotate() {
		newToken, err := generateToken()
		if err != nil {
			s.logger.Warn("failed to generate new token for rotation", zap.Error(err))
			// Continue without rotation
			session.Touch()
			_ = s.sessionRepo.Update(ctx, session)
			return result, nil
		}

		// Use the new RotateToken method which tracks the old token
		session.RotateToken(newToken)
		session.Refresh(s.sessionDuration)

		if err := s.sessionRepo.Update(ctx, session); err != nil {
			s.logger.Warn("failed to update session for rotation", zap.Error(err))
			// Revert token change and continue
			session.InvalidatePreviousToken()
			return result, nil
		}

		result.NewToken = newToken
		s.logger.Debug("session token rotated with grace period",
			zap.String("user_id", user.ID.String()),
			zap.Duration("grace_period", domain.TokenGracePeriod),
		)
	} else {
		// Just update last active time
		session.Touch()
		_ = s.sessionRepo.Update(ctx, session)
	}

	return result, nil
}

// CreateUser creates a new user account.
func (s *AuthService) CreateUser(ctx context.Context, email, password string) (*domain.User, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	user, err := domain.NewUser(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("user created",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
	)

	return user, nil
}

// CleanupExpiredSessions removes all expired sessions.
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessionRepo.DeleteExpired(ctx)
}

// generateToken generates a cryptographically secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
