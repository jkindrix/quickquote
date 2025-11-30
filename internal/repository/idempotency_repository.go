package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	apperrors "github.com/jkindrix/quickquote/internal/errors"
)

// IdempotencyRepository persists outbound call idempotency responses.
type IdempotencyRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewIdempotencyRepository creates a new repository instance.
func NewIdempotencyRepository(db *pgxpool.Pool, logger *zap.Logger) *IdempotencyRepository {
	return &IdempotencyRepository{
		db:     db,
		logger: logger,
	}
}

// Get retrieves a cached response payload if it exists and hasn't expired.
func (r *IdempotencyRepository) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	query := `
		SELECT response
		FROM idempotency_keys
		WHERE key = $1 AND expires_at > NOW()`

	var response []byte
	err := r.db.QueryRow(ctx, query, key).Scan(&response)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return response, nil
}

// Save upserts the response payload for a key.
func (r *IdempotencyRepository) Save(ctx context.Context, key string, response []byte, expiresAt time.Time) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO idempotency_keys (key, response, created_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (key) DO UPDATE
		SET response = EXCLUDED.response,
		    expires_at = EXCLUDED.expires_at,
		    created_at = NOW()`

	if _, err := r.db.Exec(ctx, query, key, response, expiresAt); err != nil {
		return apperrors.DatabaseError("IdempotencyRepository.Save", err)
	}
	return nil
}

// CleanupExpired removes expired rows. It's safe to call periodically.
func (r *IdempotencyRepository) CleanupExpired(ctx context.Context) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	if _, err := r.db.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at <= NOW()`); err != nil {
		return apperrors.DatabaseError("IdempotencyRepository.CleanupExpired", err)
	}
	return nil
}
