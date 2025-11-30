package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/jkindrix/quickquote/internal/errors"
)

// CSRFToken represents a CSRF token in the database.
type CSRFToken struct {
	ID        uuid.UUID
	Token     string
	SessionID *uuid.UUID // Optional: tied to a session
	ExpiresAt time.Time
	CreatedAt time.Time
	Used      bool
}

// CSRFRepository handles CSRF token persistence.
type CSRFRepository struct {
	pool *pgxpool.Pool
}

// NewCSRFRepository creates a new CSRF repository.
func NewCSRFRepository(pool *pgxpool.Pool) *CSRFRepository {
	return &CSRFRepository{pool: pool}
}

// Create stores a new CSRF token.
func (r *CSRFRepository) Create(ctx context.Context, token *CSRFToken) error {
	query := `
		INSERT INTO csrf_tokens (id, token, session_id, expires_at, created_at, used)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.Token,
		token.SessionID,
		token.ExpiresAt,
		token.CreatedAt,
		token.Used,
	)
	if err != nil {
		return apperrors.DatabaseError("CSRFRepository.Create", err)
	}

	return nil
}

// GetByToken retrieves a CSRF token by its value.
func (r *CSRFRepository) GetByToken(ctx context.Context, token string) (*CSRFToken, error) {
	query := `
		SELECT id, token, session_id, expires_at, created_at, used
		FROM csrf_tokens
		WHERE token = $1 AND expires_at > NOW() AND NOT used`

	csrfToken := &CSRFToken{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&csrfToken.ID,
		&csrfToken.Token,
		&csrfToken.SessionID,
		&csrfToken.ExpiresAt,
		&csrfToken.CreatedAt,
		&csrfToken.Used,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("csrf_token")
		}
		return nil, apperrors.DatabaseError("CSRFRepository.GetByToken", err)
	}

	return csrfToken, nil
}

// MarkUsed marks a token as used (for one-time tokens).
func (r *CSRFRepository) MarkUsed(ctx context.Context, token string) error {
	query := `UPDATE csrf_tokens SET used = true WHERE token = $1`

	result, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return apperrors.DatabaseError("CSRFRepository.MarkUsed", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("csrf_token")
	}

	return nil
}

// Delete removes a specific token.
func (r *CSRFRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM csrf_tokens WHERE token = $1`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return apperrors.DatabaseError("CSRFRepository.Delete", err)
	}

	return nil
}

// DeleteBySessionID removes all tokens for a session.
func (r *CSRFRepository) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM csrf_tokens WHERE session_id = $1`

	_, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return apperrors.DatabaseError("CSRFRepository.DeleteBySessionID", err)
	}

	return nil
}

// DeleteExpired removes all expired tokens.
func (r *CSRFRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM csrf_tokens WHERE expires_at < NOW()`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return apperrors.DatabaseError("CSRFRepository.DeleteExpired", err)
	}

	return nil
}

// GetOrCreate gets an existing valid token for a session or creates a new one.
func (r *CSRFRepository) GetOrCreate(ctx context.Context, sessionID *uuid.UUID, token string, expiry time.Duration) (*CSRFToken, error) {
	// If we have a session, try to get existing token
	if sessionID != nil {
		query := `
			SELECT id, token, session_id, expires_at, created_at, used
			FROM csrf_tokens
			WHERE session_id = $1 AND expires_at > NOW() AND NOT used
			ORDER BY created_at DESC
			LIMIT 1`

		existing := &CSRFToken{}
		err := r.pool.QueryRow(ctx, query, sessionID).Scan(
			&existing.ID,
			&existing.Token,
			&existing.SessionID,
			&existing.ExpiresAt,
			&existing.CreatedAt,
			&existing.Used,
		)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.DatabaseError("CSRFRepository.GetOrCreate", err)
		}
	}

	// Create new token
	now := time.Now().UTC()
	csrfToken := &CSRFToken{
		ID:        uuid.New(),
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: now.Add(expiry),
		CreatedAt: now,
		Used:      false,
	}

	if err := r.Create(ctx, csrfToken); err != nil {
		return nil, err
	}

	return csrfToken, nil
}
