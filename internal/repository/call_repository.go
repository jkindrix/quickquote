// Package repository implements data persistence using PostgreSQL.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
	apperrors "github.com/jkindrix/quickquote/internal/errors"
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
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	transcriptJSON, err := json.Marshal(call.TranscriptJSON)
	if err != nil {
		return apperrors.Wrap(err, "CallRepository.Create", apperrors.CodeInternal, "failed to marshal transcript")
	}

	extractedDataJSON, err := json.Marshal(call.ExtractedData)
	if err != nil {
		return apperrors.Wrap(err, "CallRepository.Create", apperrors.CodeInternal, "failed to marshal extracted data")
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
		return apperrors.DatabaseError("CallRepository.Create", err)
	}

	return nil
}

// GetByID retrieves a call by its internal ID (excludes soft-deleted calls).
func (r *CallRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Call, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at, deleted_at
		FROM calls
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanCall(ctx, query, id)
}

// GetByProviderCallID retrieves a call by the voice provider's call ID (excludes soft-deleted calls).
func (r *CallRepository) GetByProviderCallID(ctx context.Context, providerCallID string) (*domain.Call, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at, deleted_at
		FROM calls
		WHERE provider_call_id = $1 AND deleted_at IS NULL`

	return r.scanCall(ctx, query, providerCallID)
}

// Update updates an existing call record (excludes soft-deleted calls).
func (r *CallRepository) Update(ctx context.Context, call *domain.Call) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	call.UpdatedAt = time.Now().UTC()

	transcriptJSON, err := json.Marshal(call.TranscriptJSON)
	if err != nil {
		return apperrors.Wrap(err, "CallRepository.Update", apperrors.CodeInternal, "failed to marshal transcript")
	}

	extractedDataJSON, err := json.Marshal(call.ExtractedData)
	if err != nil {
		return apperrors.Wrap(err, "CallRepository.Update", apperrors.CodeInternal, "failed to marshal extracted data")
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
			updated_at = $16,
			deleted_at = $17
		WHERE id = $1 AND deleted_at IS NULL`

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
		call.DeletedAt,
	)
	if err != nil {
		return apperrors.DatabaseError("CallRepository.Update", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("call")
	}

	return nil
}

// Delete soft-deletes a call by setting deleted_at.
func (r *CallRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()

	now := time.Now().UTC()
	query := `
		UPDATE calls SET
			deleted_at = $2,
			updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return apperrors.DatabaseError("CallRepository.Delete", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("call")
	}

	return nil
}

// List retrieves calls with pagination, ordered by creation time descending (excludes soft-deleted).
func (r *CallRepository) List(ctx context.Context, filter *domain.CallListFilter, limit, offset int) ([]*domain.Call, error) {
	ctx, cancel := WithListQueryTimeout(ctx)
	defer cancel()

	baseQuery := `
		SELECT
			id, provider_call_id, provider, phone_number, from_number, caller_name,
			status, started_at, ended_at, duration_seconds, transcript,
			transcript_json, recording_url, quote_summary, extracted_data,
			error_message, created_at, updated_at, deleted_at
		FROM calls`

	whereClause, args := buildCallFilter(filter)
	paramIndex := len(args) + 1

	query := fmt.Sprintf(`%s %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, baseQuery, whereClause, paramIndex, paramIndex+1)

	args = append(args, limit, offset)

	return r.scanCalls(ctx, query, args...)
}

// Count returns the total number of active (non-deleted) calls.
func (r *CallRepository) Count(ctx context.Context, filter *domain.CallListFilter) (int, error) {
	ctx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	whereClause, args := buildCallFilter(filter)

	query := fmt.Sprintf(`SELECT COUNT(*) FROM calls %s`, whereClause)

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, apperrors.DatabaseError("CallRepository.Count", err)
	}
	return count, nil
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
		&call.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("call")
		}
		return nil, apperrors.DatabaseError("CallRepository.scanCall", err)
	}

	if len(transcriptJSON) > 0 {
		if err := json.Unmarshal(transcriptJSON, &call.TranscriptJSON); err != nil {
			return nil, apperrors.Wrap(err, "CallRepository.scanCall", apperrors.CodeInternal, "failed to unmarshal transcript")
		}
	}

	if len(extractedDataJSON) > 0 {
		call.ExtractedData = &domain.ExtractedData{}
		if err := json.Unmarshal(extractedDataJSON, call.ExtractedData); err != nil {
			return nil, apperrors.Wrap(err, "CallRepository.scanCall", apperrors.CodeInternal, "failed to unmarshal extracted data")
		}
	}

	return call, nil
}

// scanCalls scans multiple calls from a query.
func (r *CallRepository) scanCalls(ctx context.Context, query string, args ...interface{}) ([]*domain.Call, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.DatabaseError("CallRepository.scanCalls", err)
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
			&call.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.DatabaseError("CallRepository.scanCalls", err)
		}

		if len(transcriptJSON) > 0 {
			if err := json.Unmarshal(transcriptJSON, &call.TranscriptJSON); err != nil {
				return nil, apperrors.Wrap(err, "CallRepository.scanCalls", apperrors.CodeInternal, "failed to unmarshal transcript")
			}
		}

		if len(extractedDataJSON) > 0 {
			call.ExtractedData = &domain.ExtractedData{}
			if err := json.Unmarshal(extractedDataJSON, call.ExtractedData); err != nil {
				return nil, apperrors.Wrap(err, "CallRepository.scanCalls", apperrors.CodeInternal, "failed to unmarshal extracted data")
			}
		}

		calls = append(calls, call)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.DatabaseError("CallRepository.scanCalls", err)
	}

	return calls, nil
}

// buildCallFilter builds the WHERE clause and arguments for call listing/counting.
func buildCallFilter(filter *domain.CallListFilter) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := make([]interface{}, 0, 2)
	paramIndex := 1

	if filter != nil {
		if filter.Status != nil {
			conditions = append(conditions, fmt.Sprintf("status = $%d", paramIndex))
			args = append(args, *filter.Status)
			paramIndex++
		}
		if search := strings.TrimSpace(filter.Search); search != "" {
			conditions = append(conditions, fmt.Sprintf("(COALESCE(caller_name, '') ILIKE $%d OR phone_number ILIKE $%d OR from_number ILIKE $%d OR provider_call_id ILIKE $%d)", paramIndex, paramIndex, paramIndex, paramIndex))
			args = append(args, "%"+search+"%")
			paramIndex++
		}
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
