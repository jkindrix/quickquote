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

// PromptRepository implements domain.PromptRepository using PostgreSQL.
type PromptRepository struct {
	pool *pgxpool.Pool
}

// NewPromptRepository creates a new PromptRepository.
func NewPromptRepository(pool *pgxpool.Pool) *PromptRepository {
	return &PromptRepository{pool: pool}
}

// Create inserts a new prompt record.
func (r *PromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	query := `
		INSERT INTO prompts (
			id, name, description, task, voice, language, model,
			temperature, interruption_threshold, max_duration,
			first_sentence, wait_for_greeting,
			transfer_phone_number, transfer_list,
			voicemail_action, voicemail_message,
			record, background_track, noise_cancellation,
			knowledge_base_ids, custom_tool_ids,
			summary_prompt, dispositions, analysis_schema, keywords,
			is_default, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12,
			$13, $14,
			$15, $16,
			$17, $18, $19,
			$20, $21,
			$22, $23, $24, $25,
			$26, $27, $28, $29
		)`

	_, err := r.pool.Exec(ctx, query,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		prompt.Task,
		prompt.Voice,
		prompt.Language,
		prompt.Model,
		prompt.Temperature,
		prompt.InterruptionThreshold,
		prompt.MaxDuration,
		prompt.FirstSentence,
		prompt.WaitForGreeting,
		prompt.TransferPhoneNumber,
		prompt.TransferList,
		prompt.VoicemailAction,
		prompt.VoicemailMessage,
		prompt.Record,
		prompt.BackgroundTrack,
		prompt.NoiseCancellation,
		prompt.KnowledgeBaseIDs,
		prompt.CustomToolIDs,
		prompt.SummaryPrompt,
		prompt.Dispositions,
		prompt.AnalysisSchema,
		prompt.Keywords,
		prompt.IsDefault,
		prompt.IsActive,
		prompt.CreatedAt,
		prompt.UpdatedAt,
	)
	if err != nil {
		return apperrors.DatabaseError("PromptRepository.Create", err)
	}

	return nil
}

// GetByID retrieves a prompt by its ID.
func (r *PromptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Prompt, error) {
	query := `
		SELECT id, name, description, task, voice, language, model,
			temperature, interruption_threshold, max_duration,
			first_sentence, wait_for_greeting,
			transfer_phone_number, transfer_list,
			voicemail_action, voicemail_message,
			record, background_track, noise_cancellation,
			knowledge_base_ids, custom_tool_ids,
			summary_prompt, dispositions, analysis_schema, keywords,
			is_default, is_active, created_at, updated_at, deleted_at
		FROM prompts
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanPrompt(r.pool.QueryRow(ctx, query, id))
}

// GetByName retrieves a prompt by its name.
func (r *PromptRepository) GetByName(ctx context.Context, name string) (*domain.Prompt, error) {
	query := `
		SELECT id, name, description, task, voice, language, model,
			temperature, interruption_threshold, max_duration,
			first_sentence, wait_for_greeting,
			transfer_phone_number, transfer_list,
			voicemail_action, voicemail_message,
			record, background_track, noise_cancellation,
			knowledge_base_ids, custom_tool_ids,
			summary_prompt, dispositions, analysis_schema, keywords,
			is_default, is_active, created_at, updated_at, deleted_at
		FROM prompts
		WHERE name = $1 AND deleted_at IS NULL`

	return r.scanPrompt(r.pool.QueryRow(ctx, query, name))
}

// GetDefault retrieves the default prompt.
func (r *PromptRepository) GetDefault(ctx context.Context) (*domain.Prompt, error) {
	query := `
		SELECT id, name, description, task, voice, language, model,
			temperature, interruption_threshold, max_duration,
			first_sentence, wait_for_greeting,
			transfer_phone_number, transfer_list,
			voicemail_action, voicemail_message,
			record, background_track, noise_cancellation,
			knowledge_base_ids, custom_tool_ids,
			summary_prompt, dispositions, analysis_schema, keywords,
			is_default, is_active, created_at, updated_at, deleted_at
		FROM prompts
		WHERE is_default = true AND is_active = true AND deleted_at IS NULL
		LIMIT 1`

	return r.scanPrompt(r.pool.QueryRow(ctx, query))
}

// List retrieves prompts with pagination.
func (r *PromptRepository) List(ctx context.Context, limit, offset int, activeOnly bool) ([]*domain.Prompt, error) {
	query := `
		SELECT id, name, description, task, voice, language, model,
			temperature, interruption_threshold, max_duration,
			first_sentence, wait_for_greeting,
			transfer_phone_number, transfer_list,
			voicemail_action, voicemail_message,
			record, background_track, noise_cancellation,
			knowledge_base_ids, custom_tool_ids,
			summary_prompt, dispositions, analysis_schema, keywords,
			is_default, is_active, created_at, updated_at, deleted_at
		FROM prompts
		WHERE deleted_at IS NULL`

	if activeOnly {
		query += " AND is_active = true"
	}

	query += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, apperrors.DatabaseError("PromptRepository.List", err)
	}
	defer rows.Close()

	var prompts []*domain.Prompt
	for rows.Next() {
		prompt, err := r.scanPromptFromRows(rows)
		if err != nil {
			return nil, apperrors.DatabaseError("PromptRepository.List", err)
		}
		prompts = append(prompts, prompt)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.DatabaseError("PromptRepository.List", err)
	}

	return prompts, nil
}

// Count returns the total number of prompts.
func (r *PromptRepository) Count(ctx context.Context, activeOnly bool) (int, error) {
	query := "SELECT COUNT(*) FROM prompts WHERE deleted_at IS NULL"
	if activeOnly {
		query += " AND is_active = true"
	}

	var count int
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, apperrors.DatabaseError("PromptRepository.Count", err)
	}
	return count, nil
}

// Update updates an existing prompt record.
func (r *PromptRepository) Update(ctx context.Context, prompt *domain.Prompt) error {
	prompt.UpdatedAt = time.Now()

	query := `
		UPDATE prompts SET
			name = $2,
			description = $3,
			task = $4,
			voice = $5,
			language = $6,
			model = $7,
			temperature = $8,
			interruption_threshold = $9,
			max_duration = $10,
			first_sentence = $11,
			wait_for_greeting = $12,
			transfer_phone_number = $13,
			transfer_list = $14,
			voicemail_action = $15,
			voicemail_message = $16,
			record = $17,
			background_track = $18,
			noise_cancellation = $19,
			knowledge_base_ids = $20,
			custom_tool_ids = $21,
			summary_prompt = $22,
			dispositions = $23,
			analysis_schema = $24,
			keywords = $25,
			is_default = $26,
			is_active = $27,
			updated_at = $28
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		prompt.Task,
		prompt.Voice,
		prompt.Language,
		prompt.Model,
		prompt.Temperature,
		prompt.InterruptionThreshold,
		prompt.MaxDuration,
		prompt.FirstSentence,
		prompt.WaitForGreeting,
		prompt.TransferPhoneNumber,
		prompt.TransferList,
		prompt.VoicemailAction,
		prompt.VoicemailMessage,
		prompt.Record,
		prompt.BackgroundTrack,
		prompt.NoiseCancellation,
		prompt.KnowledgeBaseIDs,
		prompt.CustomToolIDs,
		prompt.SummaryPrompt,
		prompt.Dispositions,
		prompt.AnalysisSchema,
		prompt.Keywords,
		prompt.IsDefault,
		prompt.IsActive,
		prompt.UpdatedAt,
	)

	if err != nil {
		return apperrors.DatabaseError("PromptRepository.Update", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("prompt")
	}

	return nil
}

// Delete performs a soft delete on a prompt.
func (r *PromptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE prompts SET
			deleted_at = $2,
			is_active = false,
			is_default = false
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return apperrors.DatabaseError("PromptRepository.Delete", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("prompt")
	}

	return nil
}

// SetDefault sets a prompt as the default, unsetting any previous default.
func (r *PromptRepository) SetDefault(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperrors.DatabaseError("PromptRepository.SetDefault", err)
	}
	defer tx.Rollback(ctx)

	// Unset any existing default
	_, err = tx.Exec(ctx, "UPDATE prompts SET is_default = false WHERE is_default = true")
	if err != nil {
		return apperrors.DatabaseError("PromptRepository.SetDefault", err)
	}

	// Set the new default
	result, err := tx.Exec(ctx,
		"UPDATE prompts SET is_default = true, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL",
		id, time.Now())
	if err != nil {
		return apperrors.DatabaseError("PromptRepository.SetDefault", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound("prompt")
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.DatabaseError("PromptRepository.SetDefault", err)
	}
	return nil
}

// scanPrompt scans a single row into a Prompt struct.
func (r *PromptRepository) scanPrompt(row pgx.Row) (*domain.Prompt, error) {
	var p domain.Prompt
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Task,
		&p.Voice,
		&p.Language,
		&p.Model,
		&p.Temperature,
		&p.InterruptionThreshold,
		&p.MaxDuration,
		&p.FirstSentence,
		&p.WaitForGreeting,
		&p.TransferPhoneNumber,
		&p.TransferList,
		&p.VoicemailAction,
		&p.VoicemailMessage,
		&p.Record,
		&p.BackgroundTrack,
		&p.NoiseCancellation,
		&p.KnowledgeBaseIDs,
		&p.CustomToolIDs,
		&p.SummaryPrompt,
		&p.Dispositions,
		&p.AnalysisSchema,
		&p.Keywords,
		&p.IsDefault,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("prompt")
	}
	if err != nil {
		return nil, apperrors.DatabaseError("PromptRepository.scanPrompt", err)
	}

	return &p, nil
}

// scanPromptFromRows scans a row from pgx.Rows into a Prompt struct.
func (r *PromptRepository) scanPromptFromRows(rows pgx.Rows) (*domain.Prompt, error) {
	var p domain.Prompt
	err := rows.Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Task,
		&p.Voice,
		&p.Language,
		&p.Model,
		&p.Temperature,
		&p.InterruptionThreshold,
		&p.MaxDuration,
		&p.FirstSentence,
		&p.WaitForGreeting,
		&p.TransferPhoneNumber,
		&p.TransferList,
		&p.VoicemailAction,
		&p.VoicemailMessage,
		&p.Record,
		&p.BackgroundTrack,
		&p.NoiseCancellation,
		&p.KnowledgeBaseIDs,
		&p.CustomToolIDs,
		&p.SummaryPrompt,
		&p.Dispositions,
		&p.AnalysisSchema,
		&p.Keywords,
		&p.IsDefault,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)

	return &p, err
}
