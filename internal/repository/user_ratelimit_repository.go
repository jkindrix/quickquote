package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// UserRateLimitRepository provides database persistence for user rate limits.
type UserRateLimitRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewUserRateLimitRepository creates a new repository.
func NewUserRateLimitRepository(db *pgxpool.Pool, logger *zap.Logger) *UserRateLimitRepository {
	return &UserRateLimitRepository{
		db:     db,
		logger: logger,
	}
}

// windowDuration returns the duration for a window type.
func windowDuration(window string) time.Duration {
	switch window {
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	default:
		return time.Minute
	}
}

// IncrementRequestCount atomically increments the request count for a user.
// Uses PostgreSQL's INSERT ... ON CONFLICT to handle concurrent updates.
func (r *UserRateLimitRepository) IncrementRequestCount(ctx context.Context, userID uuid.UUID, window string) (int, error) {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	duration := windowDuration(window)
	now := time.Now().UTC()
	windowEnd := now.Truncate(duration).Add(duration)

	// Upsert: insert new record or increment existing
	// The window_end column determines when the window expires
	query := `
		INSERT INTO user_rate_limits (id, user_id, window_type, request_count, window_end, created_at, updated_at)
		VALUES ($1, $2, $3, 1, $4, $5, $5)
		ON CONFLICT (user_id, window_type) DO UPDATE
		SET
			request_count = CASE
				WHEN user_rate_limits.window_end <= $5 THEN 1
				ELSE user_rate_limits.request_count + 1
			END,
			window_end = CASE
				WHEN user_rate_limits.window_end <= $5 THEN $4
				ELSE user_rate_limits.window_end
			END,
			updated_at = $5
		RETURNING request_count
	`

	var count int
	err := r.db.QueryRow(ctx, query, uuid.New(), userID, window, windowEnd, now).Scan(&count)
	if err != nil {
		r.logger.Error("failed to increment rate limit count",
			zap.String("user_id", userID.String()),
			zap.String("window", window),
			zap.Error(err),
		)
		return 0, err
	}

	return count, nil
}

// GetRequestCount returns the current count for a user in a window.
func (r *UserRateLimitRepository) GetRequestCount(ctx context.Context, userID uuid.UUID, window string) (int, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	now := time.Now().UTC()

	query := `
		SELECT request_count
		FROM user_rate_limits
		WHERE user_id = $1 AND window_type = $2 AND window_end > $3
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID, window, now).Scan(&count)
	if err != nil {
		// No record means zero count
		return 0, nil
	}

	return count, nil
}

// ResetExpiredWindows deletes expired rate limit records.
func (r *UserRateLimitRepository) ResetExpiredWindows(ctx context.Context) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	now := time.Now().UTC()

	query := `DELETE FROM user_rate_limits WHERE window_end <= $1`

	result, err := r.db.Exec(ctx, query, now)
	if err != nil {
		r.logger.Error("failed to reset expired windows", zap.Error(err))
		return err
	}

	if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
		r.logger.Debug("cleaned up expired rate limit records",
			zap.Int64("rows_deleted", rowsAffected),
		)
	}

	return nil
}
