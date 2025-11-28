package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
)

// PersonaRepository implements domain.PersonaRepository.
type PersonaRepository struct {
	pool *pgxpool.Pool
}

// NewPersonaRepository creates a new persona repository.
func NewPersonaRepository(pool *pgxpool.Pool) *PersonaRepository {
	return &PersonaRepository{pool: pool}
}

// Create creates a new persona.
func (r *PersonaRepository) Create(ctx context.Context, persona *domain.Persona) error {
	if err := persona.MarshalAll(); err != nil {
		return fmt.Errorf("failed to marshal persona fields: %w", err)
	}

	query := `
		INSERT INTO personas (id, bland_id, name, description, voice, language,
			voice_settings, personality, background_story, system_prompt, behavior,
			knowledge_bases, tools, status, is_default, last_synced_at, sync_error,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.pool.Exec(ctx, query,
		persona.ID, persona.BlandID, persona.Name, persona.Description, persona.Voice,
		persona.Language, persona.VoiceSettingsJSON, persona.Personality, persona.BackgroundStory,
		persona.SystemPrompt, persona.BehaviorJSON, persona.KBsJSON, persona.ToolsJSON,
		persona.Status, persona.IsDefault, persona.LastSyncedAt, persona.SyncError,
		persona.CreatedAt, persona.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create persona: %w", err)
	}
	return nil
}

// GetByID retrieves a persona by ID.
func (r *PersonaRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Persona, error) {
	query := `
		SELECT id, bland_id, name, description, voice, language, voice_settings,
			personality, background_story, system_prompt, behavior, knowledge_bases,
			tools, status, is_default, last_synced_at, sync_error, created_at, updated_at
		FROM personas
		WHERE id = $1
	`
	return r.scanPersona(ctx, query, id)
}

// GetByBlandID retrieves a persona by Bland ID.
func (r *PersonaRepository) GetByBlandID(ctx context.Context, blandID string) (*domain.Persona, error) {
	query := `
		SELECT id, bland_id, name, description, voice, language, voice_settings,
			personality, background_story, system_prompt, behavior, knowledge_bases,
			tools, status, is_default, last_synced_at, sync_error, created_at, updated_at
		FROM personas
		WHERE bland_id = $1
	`
	return r.scanPersona(ctx, query, blandID)
}

// GetDefault retrieves the default persona.
func (r *PersonaRepository) GetDefault(ctx context.Context) (*domain.Persona, error) {
	query := `
		SELECT id, bland_id, name, description, voice, language, voice_settings,
			personality, background_story, system_prompt, behavior, knowledge_bases,
			tools, status, is_default, last_synced_at, sync_error, created_at, updated_at
		FROM personas
		WHERE is_default = true
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query)
	persona := &domain.Persona{}
	err := row.Scan(
		&persona.ID, &persona.BlandID, &persona.Name, &persona.Description, &persona.Voice,
		&persona.Language, &persona.VoiceSettingsJSON, &persona.Personality,
		&persona.BackgroundStory, &persona.SystemPrompt, &persona.BehaviorJSON,
		&persona.KBsJSON, &persona.ToolsJSON, &persona.Status, &persona.IsDefault,
		&persona.LastSyncedAt, &persona.SyncError, &persona.CreatedAt, &persona.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get default persona: %w", err)
	}

	if err := persona.UnmarshalAll(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal persona fields: %w", err)
	}
	return persona, nil
}

// List retrieves personas with optional filtering.
func (r *PersonaRepository) List(ctx context.Context, filter *domain.PersonaFilter) ([]*domain.Persona, error) {
	query := `
		SELECT id, bland_id, name, description, voice, language, voice_settings,
			personality, background_story, system_prompt, behavior, knowledge_bases,
			tools, status, is_default, last_synced_at, sync_error, created_at, updated_at
		FROM personas
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter != nil {
		if filter.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argNum)
			args = append(args, filter.Status)
			argNum++
		}
		if filter.Name != "" {
			query += fmt.Sprintf(" AND name ILIKE $%d", argNum)
			args = append(args, "%"+filter.Name+"%")
			argNum++
		}
		if filter.IsDefault != nil {
			query += fmt.Sprintf(" AND is_default = $%d", argNum)
			args = append(args, *filter.IsDefault)
			argNum++
		}
		query += " ORDER BY created_at DESC"
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argNum)
			args = append(args, filter.Limit)
			argNum++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argNum)
			args = append(args, filter.Offset)
		}
	} else {
		query += " ORDER BY created_at DESC"
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}
	defer rows.Close()

	var personas []*domain.Persona
	for rows.Next() {
		persona, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		personas = append(personas, persona)
	}
	return personas, rows.Err()
}

// Update updates a persona.
func (r *PersonaRepository) Update(ctx context.Context, persona *domain.Persona) error {
	if err := persona.MarshalAll(); err != nil {
		return fmt.Errorf("failed to marshal persona fields: %w", err)
	}

	query := `
		UPDATE personas
		SET bland_id = $2, name = $3, description = $4, voice = $5, language = $6,
			voice_settings = $7, personality = $8, background_story = $9, system_prompt = $10,
			behavior = $11, knowledge_bases = $12, tools = $13, status = $14,
			is_default = $15, last_synced_at = $16, sync_error = $17, updated_at = $18
		WHERE id = $1
	`
	persona.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, query,
		persona.ID, persona.BlandID, persona.Name, persona.Description, persona.Voice,
		persona.Language, persona.VoiceSettingsJSON, persona.Personality, persona.BackgroundStory,
		persona.SystemPrompt, persona.BehaviorJSON, persona.KBsJSON, persona.ToolsJSON,
		persona.Status, persona.IsDefault, persona.LastSyncedAt, persona.SyncError,
		persona.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update persona: %w", err)
	}
	return nil
}

// Delete deletes a persona.
func (r *PersonaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM personas WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete persona: %w", err)
	}
	return nil
}

// SetDefault sets a persona as the default.
func (r *PersonaRepository) SetDefault(ctx context.Context, id uuid.UUID) error {
	// First, clear any existing default
	if err := r.ClearDefault(ctx); err != nil {
		return err
	}

	query := `UPDATE personas SET is_default = true, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

// ClearDefault clears the default persona flag.
func (r *PersonaRepository) ClearDefault(ctx context.Context) error {
	query := `UPDATE personas SET is_default = false, updated_at = $1 WHERE is_default = true`
	_, err := r.pool.Exec(ctx, query, time.Now())
	return err
}

// MarkSyncing marks a persona as syncing.
func (r *PersonaRepository) MarkSyncing(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE personas SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PersonaStatusSyncing, time.Now())
	return err
}

// MarkSynced marks a persona as synced.
func (r *PersonaRepository) MarkSynced(ctx context.Context, id uuid.UUID, blandID string) error {
	now := time.Now()
	query := `UPDATE personas SET status = $2, bland_id = $3, last_synced_at = $4, sync_error = '', updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PersonaStatusActive, blandID, now)
	return err
}

// MarkSyncError marks a persona as having a sync error.
func (r *PersonaRepository) MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE personas SET status = $2, sync_error = $3, updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PersonaStatusError, errMsg, time.Now())
	return err
}

// Helper methods

func (r *PersonaRepository) scanPersona(ctx context.Context, query string, arg interface{}) (*domain.Persona, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	persona := &domain.Persona{}
	err := row.Scan(
		&persona.ID, &persona.BlandID, &persona.Name, &persona.Description, &persona.Voice,
		&persona.Language, &persona.VoiceSettingsJSON, &persona.Personality,
		&persona.BackgroundStory, &persona.SystemPrompt, &persona.BehaviorJSON,
		&persona.KBsJSON, &persona.ToolsJSON, &persona.Status, &persona.IsDefault,
		&persona.LastSyncedAt, &persona.SyncError, &persona.CreatedAt, &persona.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan persona: %w", err)
	}

	if err := persona.UnmarshalAll(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal persona fields: %w", err)
	}
	return persona, nil
}

func (r *PersonaRepository) scanRow(rows pgx.Rows) (*domain.Persona, error) {
	persona := &domain.Persona{}
	err := rows.Scan(
		&persona.ID, &persona.BlandID, &persona.Name, &persona.Description, &persona.Voice,
		&persona.Language, &persona.VoiceSettingsJSON, &persona.Personality,
		&persona.BackgroundStory, &persona.SystemPrompt, &persona.BehaviorJSON,
		&persona.KBsJSON, &persona.ToolsJSON, &persona.Status, &persona.IsDefault,
		&persona.LastSyncedAt, &persona.SyncError, &persona.CreatedAt, &persona.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan persona row: %w", err)
	}

	if err := persona.UnmarshalAll(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal persona fields: %w", err)
	}
	return persona, nil
}
