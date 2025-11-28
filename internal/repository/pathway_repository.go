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

// PathwayRepository implements domain.PathwayRepository.
type PathwayRepository struct {
	pool *pgxpool.Pool
}

// NewPathwayRepository creates a new pathway repository.
func NewPathwayRepository(pool *pgxpool.Pool) *PathwayRepository {
	return &PathwayRepository{pool: pool}
}

// Create creates a new pathway.
func (r *PathwayRepository) Create(ctx context.Context, pathway *domain.Pathway) error {
	// Marshal nodes and edges to JSON
	if err := pathway.MarshalNodes(); err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}
	if err := pathway.MarshalEdges(); err != nil {
		return fmt.Errorf("failed to marshal edges: %w", err)
	}

	query := `
		INSERT INTO pathways (id, bland_id, name, description, version, status, nodes,
			edges, start_node_id, last_synced_at, sync_error, is_published, published_at,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		pathway.ID, pathway.BlandID, pathway.Name, pathway.Description, pathway.Version,
		pathway.Status, pathway.NodesJSON, pathway.EdgesJSON, pathway.StartNodeID,
		pathway.LastSyncedAt, pathway.SyncError, pathway.IsPublished, pathway.PublishedAt,
		pathway.CreatedAt, pathway.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create pathway: %w", err)
	}
	return nil
}

// GetByID retrieves a pathway by ID.
func (r *PathwayRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Pathway, error) {
	query := `
		SELECT id, bland_id, name, description, version, status, nodes, edges,
			start_node_id, last_synced_at, sync_error, is_published, published_at,
			created_at, updated_at
		FROM pathways
		WHERE id = $1
	`
	return r.scanPathway(ctx, query, id)
}

// GetByBlandID retrieves a pathway by Bland ID.
func (r *PathwayRepository) GetByBlandID(ctx context.Context, blandID string) (*domain.Pathway, error) {
	query := `
		SELECT id, bland_id, name, description, version, status, nodes, edges,
			start_node_id, last_synced_at, sync_error, is_published, published_at,
			created_at, updated_at
		FROM pathways
		WHERE bland_id = $1
	`
	return r.scanPathway(ctx, query, blandID)
}

// List retrieves pathways with optional filtering.
func (r *PathwayRepository) List(ctx context.Context, filter *domain.PathwayFilter) ([]*domain.Pathway, error) {
	query := `
		SELECT id, bland_id, name, description, version, status, nodes, edges,
			start_node_id, last_synced_at, sync_error, is_published, published_at,
			created_at, updated_at
		FROM pathways
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
		if filter.IsPublished != nil {
			query += fmt.Sprintf(" AND is_published = $%d", argNum)
			args = append(args, *filter.IsPublished)
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
		return nil, fmt.Errorf("failed to list pathways: %w", err)
	}
	defer rows.Close()

	var pathways []*domain.Pathway
	for rows.Next() {
		pathway, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		pathways = append(pathways, pathway)
	}
	return pathways, rows.Err()
}

// Update updates a pathway.
func (r *PathwayRepository) Update(ctx context.Context, pathway *domain.Pathway) error {
	if err := pathway.MarshalNodes(); err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}
	if err := pathway.MarshalEdges(); err != nil {
		return fmt.Errorf("failed to marshal edges: %w", err)
	}

	query := `
		UPDATE pathways
		SET bland_id = $2, name = $3, description = $4, version = $5, status = $6,
			nodes = $7, edges = $8, start_node_id = $9, last_synced_at = $10,
			sync_error = $11, is_published = $12, published_at = $13, updated_at = $14
		WHERE id = $1
	`
	pathway.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, query,
		pathway.ID, pathway.BlandID, pathway.Name, pathway.Description, pathway.Version,
		pathway.Status, pathway.NodesJSON, pathway.EdgesJSON, pathway.StartNodeID,
		pathway.LastSyncedAt, pathway.SyncError, pathway.IsPublished, pathway.PublishedAt,
		pathway.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update pathway: %w", err)
	}
	return nil
}

// Delete deletes a pathway.
func (r *PathwayRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM pathways WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pathway: %w", err)
	}
	return nil
}

// SaveVersion saves a version of a pathway.
func (r *PathwayRepository) SaveVersion(ctx context.Context, version *domain.PathwayVersion) error {
	query := `
		INSERT INTO pathway_versions (id, pathway_id, version, nodes, edges, change_notes,
			created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		version.ID, version.PathwayID, version.Version, version.NodesJSON,
		version.EdgesJSON, version.ChangeNotes, version.CreatedAt, version.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to save pathway version: %w", err)
	}
	return nil
}

// GetVersion retrieves a specific version of a pathway.
func (r *PathwayRepository) GetVersion(ctx context.Context, pathwayID uuid.UUID, version int) (*domain.PathwayVersion, error) {
	query := `
		SELECT id, pathway_id, version, nodes, edges, change_notes, created_at, created_by
		FROM pathway_versions
		WHERE pathway_id = $1 AND version = $2
	`
	row := r.pool.QueryRow(ctx, query, pathwayID, version)
	v := &domain.PathwayVersion{}
	err := row.Scan(
		&v.ID, &v.PathwayID, &v.Version, &v.NodesJSON, &v.EdgesJSON,
		&v.ChangeNotes, &v.CreatedAt, &v.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pathway version: %w", err)
	}
	return v, nil
}

// ListVersions lists all versions of a pathway.
func (r *PathwayRepository) ListVersions(ctx context.Context, pathwayID uuid.UUID) ([]*domain.PathwayVersion, error) {
	query := `
		SELECT id, pathway_id, version, nodes, edges, change_notes, created_at, created_by
		FROM pathway_versions
		WHERE pathway_id = $1
		ORDER BY version DESC
	`
	rows, err := r.pool.Query(ctx, query, pathwayID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pathway versions: %w", err)
	}
	defer rows.Close()

	var versions []*domain.PathwayVersion
	for rows.Next() {
		v := &domain.PathwayVersion{}
		err := rows.Scan(
			&v.ID, &v.PathwayID, &v.Version, &v.NodesJSON, &v.EdgesJSON,
			&v.ChangeNotes, &v.CreatedAt, &v.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// RestoreVersion restores a pathway to a specific version.
func (r *PathwayRepository) RestoreVersion(ctx context.Context, pathwayID uuid.UUID, version int) error {
	// Get the version
	v, err := r.GetVersion(ctx, pathwayID, version)
	if err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("version %d not found", version)
	}

	// Get current pathway
	pathway, err := r.GetByID(ctx, pathwayID)
	if err != nil {
		return err
	}
	if pathway == nil {
		return fmt.Errorf("pathway not found")
	}

	// Save current state as new version first
	newVersion := &domain.PathwayVersion{
		ID:          uuid.New(),
		PathwayID:   pathwayID,
		Version:     pathway.Version,
		NodesJSON:   pathway.NodesJSON,
		EdgesJSON:   pathway.EdgesJSON,
		ChangeNotes: fmt.Sprintf("Auto-saved before restoring to version %d", version),
		CreatedAt:   time.Now(),
	}
	if err := r.SaveVersion(ctx, newVersion); err != nil {
		return fmt.Errorf("failed to save current version: %w", err)
	}

	// Update pathway with restored version
	pathway.NodesJSON = v.NodesJSON
	pathway.EdgesJSON = v.EdgesJSON
	pathway.Version = pathway.Version + 1
	pathway.UpdatedAt = time.Now()

	// Unmarshal for the struct
	if err := pathway.UnmarshalNodes(); err != nil {
		return err
	}
	if err := pathway.UnmarshalEdges(); err != nil {
		return err
	}

	return r.Update(ctx, pathway)
}

// MarkSyncing marks a pathway as syncing.
func (r *PathwayRepository) MarkSyncing(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE pathways SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PathwayStatusSyncing, time.Now())
	return err
}

// MarkSynced marks a pathway as synced.
func (r *PathwayRepository) MarkSynced(ctx context.Context, id uuid.UUID, blandID string) error {
	now := time.Now()
	query := `UPDATE pathways SET status = $2, bland_id = $3, last_synced_at = $4, sync_error = '', updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PathwayStatusActive, blandID, now)
	return err
}

// MarkSyncError marks a pathway as having a sync error.
func (r *PathwayRepository) MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE pathways SET status = $2, sync_error = $3, updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.PathwayStatusError, errMsg, time.Now())
	return err
}

// Publish marks a pathway as published.
func (r *PathwayRepository) Publish(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE pathways SET is_published = true, published_at = $2, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, now)
	return err
}

// Unpublish marks a pathway as unpublished.
func (r *PathwayRepository) Unpublish(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE pathways SET is_published = false, published_at = NULL, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

// Helper methods

func (r *PathwayRepository) scanPathway(ctx context.Context, query string, arg interface{}) (*domain.Pathway, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	pathway := &domain.Pathway{}
	err := row.Scan(
		&pathway.ID, &pathway.BlandID, &pathway.Name, &pathway.Description,
		&pathway.Version, &pathway.Status, &pathway.NodesJSON, &pathway.EdgesJSON,
		&pathway.StartNodeID, &pathway.LastSyncedAt, &pathway.SyncError,
		&pathway.IsPublished, &pathway.PublishedAt, &pathway.CreatedAt, &pathway.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan pathway: %w", err)
	}

	if err := pathway.UnmarshalNodes(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}
	if err := pathway.UnmarshalEdges(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
	}
	return pathway, nil
}

func (r *PathwayRepository) scanRow(rows pgx.Rows) (*domain.Pathway, error) {
	pathway := &domain.Pathway{}
	err := rows.Scan(
		&pathway.ID, &pathway.BlandID, &pathway.Name, &pathway.Description,
		&pathway.Version, &pathway.Status, &pathway.NodesJSON, &pathway.EdgesJSON,
		&pathway.StartNodeID, &pathway.LastSyncedAt, &pathway.SyncError,
		&pathway.IsPublished, &pathway.PublishedAt, &pathway.CreatedAt, &pathway.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pathway row: %w", err)
	}

	if err := pathway.UnmarshalNodes(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}
	if err := pathway.UnmarshalEdges(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
	}
	return pathway, nil
}
