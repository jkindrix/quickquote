package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
)

// QuoteJobRepository implements domain.QuoteJobRepository using PostgreSQL.
type QuoteJobRepository struct {
	pool *pgxpool.Pool
}

// NewQuoteJobRepository creates a new QuoteJobRepository.
func NewQuoteJobRepository(pool *pgxpool.Pool) *QuoteJobRepository {
	return &QuoteJobRepository{pool: pool}
}

// Create inserts a new quote job.
func (r *QuoteJobRepository) Create(ctx context.Context, job *domain.QuoteJob) error {
	metadataJSON, err := json.Marshal(job.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO quote_jobs (
			id, call_id, status, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at,
			last_error, error_count, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err = r.pool.Exec(ctx, query,
		job.ID,
		job.CallID,
		job.Status,
		job.Attempts,
		job.MaxAttempts,
		job.CreatedAt,
		job.UpdatedAt,
		job.ScheduledAt,
		job.StartedAt,
		job.CompletedAt,
		job.LastError,
		job.ErrorCount,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert quote job: %w", err)
	}

	return nil
}

// GetByID retrieves a job by ID.
func (r *QuoteJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuoteJob, error) {
	query := `
		SELECT
			id, call_id, status, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at,
			last_error, error_count, metadata
		FROM quote_jobs
		WHERE id = $1`

	return r.scanJob(ctx, query, id)
}

// GetByCallID retrieves the job for a specific call.
func (r *QuoteJobRepository) GetByCallID(ctx context.Context, callID uuid.UUID) (*domain.QuoteJob, error) {
	query := `
		SELECT
			id, call_id, status, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at,
			last_error, error_count, metadata
		FROM quote_jobs
		WHERE call_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	return r.scanJob(ctx, query, callID)
}

// Update updates an existing job.
func (r *QuoteJobRepository) Update(ctx context.Context, job *domain.QuoteJob) error {
	metadataJSON, err := json.Marshal(job.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE quote_jobs SET
			status = $2,
			attempts = $3,
			max_attempts = $4,
			updated_at = $5,
			scheduled_at = $6,
			started_at = $7,
			completed_at = $8,
			last_error = $9,
			error_count = $10,
			metadata = $11
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		job.ID,
		job.Status,
		job.Attempts,
		job.MaxAttempts,
		job.UpdatedAt,
		job.ScheduledAt,
		job.StartedAt,
		job.CompletedAt,
		job.LastError,
		job.ErrorCount,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update quote job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// GetPendingJobs retrieves jobs ready to be processed.
// Returns jobs where status='pending' and scheduled_at <= now.
func (r *QuoteJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*domain.QuoteJob, error) {
	query := `
		SELECT
			id, call_id, status, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at,
			last_error, error_count, metadata
		FROM quote_jobs
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT $1`

	return r.scanJobs(ctx, query, limit)
}

// GetProcessingJobs retrieves jobs currently being processed.
// Useful for detecting stuck jobs on startup.
func (r *QuoteJobRepository) GetProcessingJobs(ctx context.Context, olderThan time.Duration) ([]*domain.QuoteJob, error) {
	cutoff := time.Now().Add(-olderThan)

	query := `
		SELECT
			id, call_id, status, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at,
			last_error, error_count, metadata
		FROM quote_jobs
		WHERE status = 'processing' AND started_at < $1
		ORDER BY started_at ASC`

	return r.scanJobs(ctx, query, cutoff)
}

// CountByStatus returns counts of jobs by status.
func (r *QuoteJobRepository) CountByStatus(ctx context.Context) (map[domain.QuoteJobStatus]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM quote_jobs
		GROUP BY status`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[domain.QuoteJobStatus]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		counts[domain.QuoteJobStatus(status)] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status counts: %w", err)
	}

	return counts, nil
}

// scanJob scans a single job from a query.
func (r *QuoteJobRepository) scanJob(ctx context.Context, query string, args ...interface{}) (*domain.QuoteJob, error) {
	job := &domain.QuoteJob{}
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID,
		&job.CallID,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.ScheduledAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.LastError,
		&job.ErrorCount,
		&metadataJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan quote job: %w", err)
	}

	if len(metadataJSON) > 0 {
		job.Metadata = make(map[string]interface{})
		if err := json.Unmarshal(metadataJSON, &job.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return job, nil
}

// scanJobs scans multiple jobs from a query.
func (r *QuoteJobRepository) scanJobs(ctx context.Context, query string, args ...interface{}) ([]*domain.QuoteJob, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query quote jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*domain.QuoteJob
	for rows.Next() {
		job := &domain.QuoteJob{}
		var metadataJSON []byte

		err := rows.Scan(
			&job.ID,
			&job.CallID,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.ScheduledAt,
			&job.StartedAt,
			&job.CompletedAt,
			&job.LastError,
			&job.ErrorCount,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quote job row: %w", err)
		}

		if len(metadataJSON) > 0 {
			job.Metadata = make(map[string]interface{})
			if err := json.Unmarshal(metadataJSON, &job.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quote job rows: %w", err)
	}

	return jobs, nil
}
