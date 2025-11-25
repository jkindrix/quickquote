package domain

import (
	"context"

	"github.com/google/uuid"
)

// CallRepository defines the interface for call data persistence.
type CallRepository interface {
	// Create inserts a new call record.
	Create(ctx context.Context, call *Call) error

	// GetByID retrieves a call by its internal ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Call, error)

	// GetByBlandCallID retrieves a call by the Bland AI call ID.
	GetByBlandCallID(ctx context.Context, blandCallID string) (*Call, error)

	// Update updates an existing call record.
	Update(ctx context.Context, call *Call) error

	// List retrieves calls with pagination.
	List(ctx context.Context, limit, offset int) ([]*Call, error)

	// Count returns the total number of calls.
	Count(ctx context.Context) (int, error)

	// ListByStatus retrieves calls filtered by status.
	ListByStatus(ctx context.Context, status CallStatus, limit, offset int) ([]*Call, error)
}

// UserRepository defines the interface for user data persistence.
type UserRepository interface {
	// Create inserts a new user.
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmail retrieves a user by email address.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *User) error
}

// SessionRepository defines the interface for session data persistence.
type SessionRepository interface {
	// Create inserts a new session.
	Create(ctx context.Context, session *Session) error

	// GetByToken retrieves a session by its token.
	GetByToken(ctx context.Context, token string) (*Session, error)

	// Delete removes a session.
	Delete(ctx context.Context, token string) error

	// DeleteExpired removes all expired sessions.
	DeleteExpired(ctx context.Context) error

	// DeleteByUserID removes all sessions for a user.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
