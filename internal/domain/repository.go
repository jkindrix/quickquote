package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CallRepository defines the interface for call data persistence.
type CallRepository interface {
	// Create inserts a new call record.
	Create(ctx context.Context, call *Call) error

	// GetByID retrieves a call by its internal ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Call, error)

	// GetByProviderCallID retrieves a call by the voice provider's call ID.
	GetByProviderCallID(ctx context.Context, providerCallID string) (*Call, error)

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

	// Count returns the total number of users.
	Count(ctx context.Context) (int64, error)
}

// SessionRepository defines the interface for session data persistence.
type SessionRepository interface {
	// Create inserts a new session.
	Create(ctx context.Context, session *Session) error

	// GetByToken retrieves a session by its token.
	GetByToken(ctx context.Context, token string) (*Session, error)

	// Update updates an existing session (for token rotation, activity tracking).
	Update(ctx context.Context, session *Session) error

	// Delete removes a session.
	Delete(ctx context.Context, token string) error

	// DeleteExpired removes all expired sessions.
	DeleteExpired(ctx context.Context) error

	// DeleteByUserID removes all sessions for a user.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// QuoteJobRepository defines the interface for quote job persistence.
type QuoteJobRepository interface {
	// Create inserts a new quote job.
	Create(ctx context.Context, job *QuoteJob) error

	// GetByID retrieves a job by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*QuoteJob, error)

	// GetByCallID retrieves the job for a specific call.
	GetByCallID(ctx context.Context, callID uuid.UUID) (*QuoteJob, error)

	// Update updates an existing job.
	Update(ctx context.Context, job *QuoteJob) error

	// GetPendingJobs retrieves jobs ready to be processed.
	// Returns jobs where status='pending' and scheduled_at <= now.
	GetPendingJobs(ctx context.Context, limit int) ([]*QuoteJob, error)

	// GetProcessingJobs retrieves jobs currently being processed.
	// Useful for detecting stuck jobs on startup.
	GetProcessingJobs(ctx context.Context, olderThan time.Duration) ([]*QuoteJob, error)

	// CountByStatus returns counts of jobs by status.
	CountByStatus(ctx context.Context) (map[QuoteJobStatus]int, error)
}
