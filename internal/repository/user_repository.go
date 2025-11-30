package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
	apperrors "github.com/jkindrix/quickquote/internal/errors"
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
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

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
		return apperrors.DatabaseError("UserRepository.Create", err)
	}

	return nil
}

// GetByID retrieves a user by ID (excludes soft-deleted users).
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, email, password_hash, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user")
		}
		return nil, apperrors.DatabaseError("UserRepository.GetByID", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email address (excludes soft-deleted users).
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, email, password_hash, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user")
		}
		return nil, apperrors.DatabaseError("UserRepository.GetByEmail", err)
	}

	return user, nil
}

// Update updates an existing user (excludes soft-deleted users).
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users SET
			email = $2,
			password_hash = $3,
			updated_at = $4,
			deleted_at = $5
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.UpdatedAt,
		user.DeletedAt,
	)
	if err != nil {
		return apperrors.DatabaseError("UserRepository.Update", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("user")
	}

	return nil
}

// Delete soft-deletes a user by setting deleted_at.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	now := time.Now().UTC()
	query := `
		UPDATE users SET
			deleted_at = $2,
			updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return apperrors.DatabaseError("UserRepository.Delete", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("user")
	}

	return nil
}

// Count returns the total number of active (non-deleted) users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&count)
	if err != nil {
		return 0, apperrors.DatabaseError("UserRepository.Count", err)
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
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

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
		return apperrors.DatabaseError("SessionRepository.Create", err)
	}

	return nil
}

// GetByToken retrieves a session by its token (current or previous within grace period).
func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

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
			return nil, apperrors.NotFound("session")
		}
		return nil, apperrors.DatabaseError("SessionRepository.GetByToken", err)
	}

	return session, nil
}

// Update updates an existing session.
func (r *SessionRepository) Update(ctx context.Context, session *domain.Session) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

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
		return apperrors.DatabaseError("SessionRepository.Update", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("session")
	}

	return nil
}

// Delete removes a session.
func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `DELETE FROM sessions WHERE token = $1`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return apperrors.DatabaseError("SessionRepository.Delete", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions.
func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `DELETE FROM sessions WHERE expires_at < NOW()`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return apperrors.DatabaseError("SessionRepository.DeleteExpired", err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a user.
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `DELETE FROM sessions WHERE user_id = $1`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return apperrors.DatabaseError("SessionRepository.DeleteByUserID", err)
	}

	return nil
}

// ClearExpiredPreviousTokens clears previous_token for sessions past the grace period.
func (r *SessionRepository) ClearExpiredPreviousTokens(ctx context.Context) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `
		UPDATE sessions SET
			previous_token = NULL,
			rotated_at = NULL
		WHERE previous_token IS NOT NULL AND rotated_at < NOW() - INTERVAL '30 seconds'`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return apperrors.DatabaseError("SessionRepository.ClearExpiredPreviousTokens", err)
	}

	return nil
}

// InvalidatePreviousToken explicitly invalidates a previous token for a session.
func (r *SessionRepository) InvalidatePreviousToken(ctx context.Context, sessionID uuid.UUID) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `
		UPDATE sessions SET
			previous_token = NULL,
			rotated_at = NULL
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return apperrors.DatabaseError("SessionRepository.InvalidatePreviousToken", err)
	}

	return nil
}
