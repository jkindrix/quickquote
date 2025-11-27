// Package repository implements data persistence using PostgreSQL.
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

// CallRepository implements domain.CallRepository using PostgreSQL.
type CallRepository struct {
	pool *pgxpool.Pool
}

// NewCallRepository creates a new CallRepository.
func NewCallRepository(pool *pgxpool.Pool) *CallRepository {
	return &CallRepository{pool: pool}
}

// Create inserts a new call record.
func (r *CallRepository) Create(ctx context.Context, call *domain.Call) error {
	transcriptJSON, err := json.Marshal(call.TranscriptJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal transcript: %w", err)
	}

	extractedDataJSON, err := json.Marshal(call.ExtractedData)
	if err != nil {
		return fmt.Errorf("failed to marshal extracted data: %w", err)
	}

	query := `
		INSERT INTO calls (
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)`

	_, err = r.pool.Exec(ctx, query,
		call.ID,
		call.ProviderCallID,
		call.Provider,
		call.PhoneNumber,
		call.FromNumber,
		call.CallerName,
		call.Status,
		call.StartedAt,
		call.EndedAt,
		call.DurationSeconds,
		call.Transcript,
		transcriptJSON,
		call.RecordingURL,
		call.QuoteSummary,
		extractedDataJSON,
		call.ErrorMessage,
		call.CreatedAt,
		call.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert call: %w", err)
	}

	return nil
}

// GetByID retrieves a call by its internal ID.
func (r *CallRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Call, error) {
	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at
		FROM calls
		WHERE id = $1`

	return r.scanCall(ctx, query, id)
}

// GetByProviderCallID retrieves a call by the voice provider's call ID.
func (r *CallRepository) GetByProviderCallID(ctx context.Context, providerCallID string) (*domain.Call, error) {
	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at
		FROM calls
		WHERE provider_call_id = $1`

	return r.scanCall(ctx, query, providerCallID)
}

// Update updates an existing call record.
func (r *CallRepository) Update(ctx context.Context, call *domain.Call) error {
	call.UpdatedAt = time.Now().UTC()

	transcriptJSON, err := json.Marshal(call.TranscriptJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal transcript: %w", err)
	}

	extractedDataJSON, err := json.Marshal(call.ExtractedData)
	if err != nil {
		return fmt.Errorf("failed to marshal extracted data: %w", err)
	}

	query := `
		UPDATE calls SET
			provider = $2,
			phone_number = $3,
			from_number = $4,
			caller_name = $5,
			status = $6,
			started_at = $7,
			ended_at = $8,
			duration_seconds = $9,
			transcript = $10,
			transcript_json = $11,
			recording_url = $12,
			quote_summary = $13,
			extracted_data = $14,
			error_message = $15,
			updated_at = $16
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		call.ID,
		call.Provider,
		call.PhoneNumber,
		call.FromNumber,
		call.CallerName,
		call.Status,
		call.StartedAt,
		call.EndedAt,
		call.DurationSeconds,
		call.Transcript,
		transcriptJSON,
		call.RecordingURL,
		call.QuoteSummary,
		extractedDataJSON,
		call.ErrorMessage,
		call.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update call: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// List retrieves calls with pagination, ordered by creation time descending.
func (r *CallRepository) List(ctx context.Context, limit, offset int) ([]*domain.Call, error) {
	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at
		FROM calls
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	return r.scanCalls(ctx, query, limit, offset)
}

// Count returns the total number of calls.
func (r *CallRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM calls").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count calls: %w", err)
	}
	return count, nil
}

// ListByStatus retrieves calls filtered by status.
func (r *CallRepository) ListByStatus(ctx context.Context, status domain.CallStatus, limit, offset int) ([]*domain.Call, error) {
	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at
		FROM calls
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.scanCalls(ctx, query, status, limit, offset)
}

// scanCall scans a single call from a query.
func (r *CallRepository) scanCall(ctx context.Context, query string, args ...interface{}) (*domain.Call, error) {
	call := &domain.Call{}
	var transcriptJSON, extractedDataJSON []byte

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&call.ID,
		&call.ProviderCallID,
		&call.Provider,
		&call.PhoneNumber,
		&call.FromNumber,
		&call.CallerName,
		&call.Status,
		&call.StartedAt,
		&call.EndedAt,
		&call.DurationSeconds,
		&call.Transcript,
		&transcriptJSON,
		&call.RecordingURL,
		&call.QuoteSummary,
		&extractedDataJSON,
		&call.ErrorMessage,
		&call.CreatedAt,
		&call.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan call: %w", err)
	}

	if len(transcriptJSON) > 0 {
		if err := json.Unmarshal(transcriptJSON, &call.TranscriptJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transcript: %w", err)
		}
	}

	if len(extractedDataJSON) > 0 {
		call.ExtractedData = &domain.ExtractedData{}
		if err := json.Unmarshal(extractedDataJSON, call.ExtractedData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal extracted data: %w", err)
		}
	}

	return call, nil
}

// scanCalls scans multiple calls from a query.
func (r *CallRepository) scanCalls(ctx context.Context, query string, args ...interface{}) ([]*domain.Call, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query calls: %w", err)
	}
	defer rows.Close()

	var calls []*domain.Call
	for rows.Next() {
		call := &domain.Call{}
		var transcriptJSON, extractedDataJSON []byte

		err := rows.Scan(
			&call.ID,
			&call.ProviderCallID,
			&call.Provider,
			&call.PhoneNumber,
			&call.FromNumber,
			&call.CallerName,
			&call.Status,
			&call.StartedAt,
			&call.EndedAt,
			&call.DurationSeconds,
			&call.Transcript,
			&transcriptJSON,
			&call.RecordingURL,
			&call.QuoteSummary,
			&extractedDataJSON,
			&call.ErrorMessage,
			&call.CreatedAt,
			&call.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan call row: %w", err)
		}

		if len(transcriptJSON) > 0 {
			if err := json.Unmarshal(transcriptJSON, &call.TranscriptJSON); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transcript: %w", err)
			}
		}

		if len(extractedDataJSON) > 0 {
			call.ExtractedData = &domain.ExtractedData{}
			if err := json.Unmarshal(extractedDataJSON, call.ExtractedData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal extracted data: %w", err)
			}
		}

		calls = append(calls, call)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating call rows: %w", err)
	}

	return calls, nil
}
