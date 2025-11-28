package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
)

// SettingsRepository implements domain.SettingsRepository using PostgreSQL.
type SettingsRepository struct {
	db *pgxpool.Pool
}

// NewSettingsRepository creates a new settings repository.
func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a single setting by key.
func (r *SettingsRepository) Get(ctx context.Context, key string) (*domain.Setting, error) {
	query := `
		SELECT id, key, value, value_type, category, description, created_at, updated_at
		FROM settings
		WHERE key = $1
	`

	var s domain.Setting
	err := r.db.QueryRow(ctx, query, key).Scan(
		&s.ID, &s.Key, &s.Value, &s.ValueType, &s.Category,
		&s.Description, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	return &s, nil
}

// GetByCategory retrieves all settings in a category.
func (r *SettingsRepository) GetByCategory(ctx context.Context, category string) ([]*domain.Setting, error) {
	query := `
		SELECT id, key, value, value_type, category, description, created_at, updated_at
		FROM settings
		WHERE category = $1
		ORDER BY key
	`

	rows, err := r.db.Query(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings by category: %w", err)
	}
	defer rows.Close()

	var settings []*domain.Setting
	for rows.Next() {
		var s domain.Setting
		if err := rows.Scan(
			&s.ID, &s.Key, &s.Value, &s.ValueType, &s.Category,
			&s.Description, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, &s)
	}

	return settings, rows.Err()
}

// GetAll retrieves all settings.
func (r *SettingsRepository) GetAll(ctx context.Context) ([]*domain.Setting, error) {
	query := `
		SELECT id, key, value, value_type, category, description, created_at, updated_at
		FROM settings
		ORDER BY category, key
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all settings: %w", err)
	}
	defer rows.Close()

	var settings []*domain.Setting
	for rows.Next() {
		var s domain.Setting
		if err := rows.Scan(
			&s.ID, &s.Key, &s.Value, &s.ValueType, &s.Category,
			&s.Description, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, &s)
	}

	return settings, rows.Err()
}

// Set updates or inserts a setting value.
func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	query := `
		UPDATE settings SET value = $2, updated_at = NOW()
		WHERE key = $1
	`

	result, err := r.db.Exec(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to update setting %s: %w", key, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("setting %s not found", key)
	}

	return nil
}

// SetMany updates multiple settings in a transaction.
func (r *SettingsRepository) SetMany(ctx context.Context, settings map[string]string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `UPDATE settings SET value = $2, updated_at = NOW() WHERE key = $1`

	for key, value := range settings {
		_, err := tx.Exec(ctx, query, key, value)
		if err != nil {
			return fmt.Errorf("failed to update setting %s: %w", key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit settings: %w", err)
	}

	return nil
}

// Delete removes a setting.
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM settings WHERE key = $1`

	_, err := r.db.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting %s: %w", key, err)
	}

	return nil
}

// GetAsMap returns all settings as a key->value map for easy consumption.
func (r *SettingsRepository) GetAsMap(ctx context.Context) (map[string]string, error) {
	settings, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return result, nil
}
