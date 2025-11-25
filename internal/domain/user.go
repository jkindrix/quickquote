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
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// NewSession creates a new session for a user.
func NewSession(userID uuid.UUID, token string, duration time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(duration),
		CreatedAt: now,
	}
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}
