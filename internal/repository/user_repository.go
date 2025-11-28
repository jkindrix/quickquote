package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
)

// UserRepository implements domain.UserRepository using PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates an existing user.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users SET
			email = $2,
			password_hash = $3,
			updated_at = $4
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// SessionRepository implements domain.SessionRepository using PostgreSQL.
type SessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Create inserts a new session.
func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at, last_active_at, ip_address, user_agent, previous_token, rotated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
		session.LastActiveAt,
		session.IPAddress,
		session.UserAgent,
		session.PreviousToken,
		session.RotatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	return nil
}

// GetByToken retrieves a session by its token (current or previous within grace period).
func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	// Query for current token OR previous token within grace period
	query := `
		SELECT id, user_id, token, expires_at, created_at,
		       COALESCE(last_active_at, created_at) as last_active_at,
		       COALESCE(ip_address, '') as ip_address,
		       COALESCE(user_agent, '') as user_agent,
		       previous_token,
		       rotated_at
		FROM sessions
		WHERE expires_at > NOW() AND (
			token = $1 OR
			(previous_token = $1 AND rotated_at > NOW() - INTERVAL '30 seconds')
		)`

	session := &domain.Session{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastActiveAt,
		&session.IPAddress,
		&session.UserAgent,
		&session.PreviousToken,
		&session.RotatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return session, nil
}

// Update updates an existing session.
func (r *SessionRepository) Update(ctx context.Context, session *domain.Session) error {
	query := `
		UPDATE sessions SET
			token = $2,
			expires_at = $3,
			last_active_at = $4,
			ip_address = $5,
			user_agent = $6,
			previous_token = $7,
			rotated_at = $8
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		session.ID,
		session.Token,
		session.ExpiresAt,
		session.LastActiveAt,
		session.IPAddress,
		session.UserAgent,
		session.PreviousToken,
		session.RotatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a session.
func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions.
func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a user.
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// ClearExpiredPreviousTokens clears previous_token for sessions past the grace period.
func (r *SessionRepository) ClearExpiredPreviousTokens(ctx context.Context) error {
	query := `
		UPDATE sessions SET
			previous_token = NULL,
			rotated_at = NULL
		WHERE previous_token IS NOT NULL AND rotated_at < NOW() - INTERVAL '30 seconds'`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear expired previous tokens: %w", err)
	}

	return nil
}

// InvalidatePreviousToken explicitly invalidates a previous token for a session.
func (r *SessionRepository) InvalidatePreviousToken(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE sessions SET
			previous_token = NULL,
			rotated_at = NULL
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to invalidate previous token: %w", err)
	}

	return nil
}
