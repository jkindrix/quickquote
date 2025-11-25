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

// Login authenticates a user and creates a session.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.Session, error) {
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
	token, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	session := domain.NewSession(user.ID, token, s.sessionDuration)

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

// ValidateSession validates a session token and returns the associated user.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*domain.User, error) {
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

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
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
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
