package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a dashboard user.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize password hash
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewUser creates a new user with a hashed password.
func NewUser(email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CheckPassword verifies a password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// Session represents an authenticated user session.
type Session struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	Token        string     `json:"token"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	// Token rotation tracking
	PreviousToken *string    `json:"-"` // Previous token (for grace period)
	RotatedAt     *time.Time `json:"-"` // When the token was last rotated
}

// NewSession creates a new session for a user.
func NewSession(userID uuid.UUID, token string, duration time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:           uuid.New(),
		UserID:       userID,
		Token:        token,
		ExpiresAt:    now.Add(duration),
		CreatedAt:    now,
		LastActiveAt: now,
	}
}

// NewSessionWithContext creates a new session with IP and user agent info.
func NewSessionWithContext(userID uuid.UUID, token string, duration time.Duration, ipAddress, userAgent string) *Session {
	session := NewSession(userID, token, duration)
	session.IPAddress = ipAddress
	session.UserAgent = userAgent
	return session
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// Touch updates the last active timestamp.
func (s *Session) Touch() {
	s.LastActiveAt = time.Now().UTC()
}

// ShouldRotate returns true if the session token should be rotated.
// Tokens should be rotated every 15 minutes to limit exposure.
func (s *Session) ShouldRotate() bool {
	return time.Since(s.LastActiveAt) > 15*time.Minute
}

// Refresh extends the session expiration time.
func (s *Session) Refresh(duration time.Duration) {
	s.ExpiresAt = time.Now().UTC().Add(duration)
	s.Touch()
}

// RotateToken rotates the session token, keeping track of the previous token.
func (s *Session) RotateToken(newToken string) {
	// Make a copy of the current token before overwriting
	oldToken := s.Token
	s.PreviousToken = &oldToken
	now := time.Now().UTC()
	s.RotatedAt = &now
	s.Token = newToken
	s.Touch()
}

// TokenGracePeriod is the duration during which the old token is still valid after rotation.
const TokenGracePeriod = 30 * time.Second

// IsWithinGracePeriod returns true if the previous token is still within grace period.
func (s *Session) IsWithinGracePeriod() bool {
	if s.RotatedAt == nil {
		return false
	}
	return time.Since(*s.RotatedAt) < TokenGracePeriod
}

// MatchesToken returns true if the given token matches either current or previous (within grace).
func (s *Session) MatchesToken(token string) bool {
	if s.Token == token {
		return true
	}
	// Check previous token within grace period
	if s.PreviousToken != nil && *s.PreviousToken == token && s.IsWithinGracePeriod() {
		return true
	}
	return false
}

// InvalidatePreviousToken clears the previous token (e.g., after grace period).
func (s *Session) InvalidatePreviousToken() {
	s.PreviousToken = nil
	s.RotatedAt = nil
}
