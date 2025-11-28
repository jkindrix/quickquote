package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jkindrix/quickquote/internal/domain"
)

// KnowledgeBaseRepository implements domain.KnowledgeBaseRepository.
type KnowledgeBaseRepository struct {
	pool *pgxpool.Pool
}

// NewKnowledgeBaseRepository creates a new knowledge base repository.
func NewKnowledgeBaseRepository(pool *pgxpool.Pool) *KnowledgeBaseRepository {
	return &KnowledgeBaseRepository{pool: pool}
}

// Create creates a new knowledge base.
func (r *KnowledgeBaseRepository) Create(ctx context.Context, kb *domain.KnowledgeBase) error {
	metadataJSON, err := json.Marshal(kb.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO knowledge_bases (id, bland_id, name, description, vector_db_id, status,
			document_count, last_synced_at, sync_error, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = r.pool.Exec(ctx, query,
		kb.ID, kb.BlandID, kb.Name, kb.Description, kb.VectorDBID, kb.Status,
		kb.DocumentCount, kb.LastSyncedAt, kb.SyncError, string(metadataJSON),
		kb.CreatedAt, kb.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create knowledge base: %w", err)
	}
	return nil
}

// GetByID retrieves a knowledge base by ID.
func (r *KnowledgeBaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.KnowledgeBase, error) {
	query := `
		SELECT id, bland_id, name, description, vector_db_id, status, document_count,
			last_synced_at, sync_error, metadata, created_at, updated_at
		FROM knowledge_bases
		WHERE id = $1
	`
	return r.scanKnowledgeBase(ctx, query, id)
}

// GetByBlandID retrieves a knowledge base by Bland ID.
func (r *KnowledgeBaseRepository) GetByBlandID(ctx context.Context, blandID string) (*domain.KnowledgeBase, error) {
	query := `
		SELECT id, bland_id, name, description, vector_db_id, status, document_count,
			last_synced_at, sync_error, metadata, created_at, updated_at
		FROM knowledge_bases
		WHERE bland_id = $1
	`
	return r.scanKnowledgeBase(ctx, query, blandID)
}

// List retrieves knowledge bases with optional filtering.
func (r *KnowledgeBaseRepository) List(ctx context.Context, filter *domain.KnowledgeBaseFilter) ([]*domain.KnowledgeBase, error) {
	query := `
		SELECT id, bland_id, name, description, vector_db_id, status, document_count,
			last_synced_at, sync_error, metadata, created_at, updated_at
		FROM knowledge_bases
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
		return nil, fmt.Errorf("failed to list knowledge bases: %w", err)
	}
	defer rows.Close()

	var kbs []*domain.KnowledgeBase
	for rows.Next() {
		kb, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		kbs = append(kbs, kb)
	}
	return kbs, rows.Err()
}

// Update updates a knowledge base.
func (r *KnowledgeBaseRepository) Update(ctx context.Context, kb *domain.KnowledgeBase) error {
	metadataJSON, err := json.Marshal(kb.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE knowledge_bases
		SET bland_id = $2, name = $3, description = $4, vector_db_id = $5, status = $6,
			document_count = $7, last_synced_at = $8, sync_error = $9, metadata = $10,
			updated_at = $11
		WHERE id = $1
	`
	kb.UpdatedAt = time.Now()
	_, err = r.pool.Exec(ctx, query,
		kb.ID, kb.BlandID, kb.Name, kb.Description, kb.VectorDBID, kb.Status,
		kb.DocumentCount, kb.LastSyncedAt, kb.SyncError, string(metadataJSON),
		kb.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update knowledge base: %w", err)
	}
	return nil
}

// Delete deletes a knowledge base.
func (r *KnowledgeBaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM knowledge_bases WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge base: %w", err)
	}
	return nil
}

// MarkSyncing marks a knowledge base as syncing.
func (r *KnowledgeBaseRepository) MarkSyncing(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE knowledge_bases SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.KnowledgeBaseStatusSyncing, time.Now())
	return err
}

// MarkSynced marks a knowledge base as synced.
func (r *KnowledgeBaseRepository) MarkSynced(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE knowledge_bases SET status = $2, last_synced_at = $3, sync_error = '', updated_at = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.KnowledgeBaseStatusActive, now)
	return err
}

// MarkSyncError marks a knowledge base as having a sync error.
func (r *KnowledgeBaseRepository) MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE knowledge_bases SET status = $2, sync_error = $3, updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, domain.KnowledgeBaseStatusError, errMsg, time.Now())
	return err
}

// AddDocument adds a document to a knowledge base.
func (r *KnowledgeBaseRepository) AddDocument(ctx context.Context, doc *domain.KnowledgeBaseDocument) error {
	query := `
		INSERT INTO knowledge_base_documents (id, knowledge_base_id, bland_doc_id, name,
			content_type, content_hash, size_bytes, chunk_count, status, error_message,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		doc.ID, doc.KnowledgeBaseID, doc.BlandDocID, doc.Name,
		doc.ContentType, doc.ContentHash, doc.SizeBytes, doc.ChunkCount,
		doc.Status, doc.ErrorMessage, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	// Update document count
	_, err = r.pool.Exec(ctx,
		`UPDATE knowledge_bases SET document_count = document_count + 1 WHERE id = $1`,
		doc.KnowledgeBaseID,
	)
	return err
}

// GetDocument retrieves a document by ID.
func (r *KnowledgeBaseRepository) GetDocument(ctx context.Context, id uuid.UUID) (*domain.KnowledgeBaseDocument, error) {
	query := `
		SELECT id, knowledge_base_id, bland_doc_id, name, content_type, content_hash,
			size_bytes, chunk_count, status, error_message, created_at, updated_at
		FROM knowledge_base_documents
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	doc := &domain.KnowledgeBaseDocument{}
	err := row.Scan(
		&doc.ID, &doc.KnowledgeBaseID, &doc.BlandDocID, &doc.Name, &doc.ContentType,
		&doc.ContentHash, &doc.SizeBytes, &doc.ChunkCount, &doc.Status,
		&doc.ErrorMessage, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return doc, nil
}

// ListDocuments lists documents in a knowledge base.
func (r *KnowledgeBaseRepository) ListDocuments(ctx context.Context, kbID uuid.UUID) ([]*domain.KnowledgeBaseDocument, error) {
	query := `
		SELECT id, knowledge_base_id, bland_doc_id, name, content_type, content_hash,
			size_bytes, chunk_count, status, error_message, created_at, updated_at
		FROM knowledge_base_documents
		WHERE knowledge_base_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, kbID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var docs []*domain.KnowledgeBaseDocument
	for rows.Next() {
		doc := &domain.KnowledgeBaseDocument{}
		err := rows.Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.BlandDocID, &doc.Name, &doc.ContentType,
			&doc.ContentHash, &doc.SizeBytes, &doc.ChunkCount, &doc.Status,
			&doc.ErrorMessage, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// UpdateDocumentStatus updates a document's status.
func (r *KnowledgeBaseRepository) UpdateDocumentStatus(ctx context.Context, docID uuid.UUID, status, errMsg string) error {
	query := `UPDATE knowledge_base_documents SET status = $2, error_message = $3, updated_at = $4 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, docID, status, errMsg, time.Now())
	return err
}

// DeleteDocument deletes a document.
func (r *KnowledgeBaseRepository) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	// Get the document to find its KB ID
	doc, err := r.GetDocument(ctx, id)
	if err != nil {
		return err
	}
	if doc == nil {
		return nil
	}

	query := `DELETE FROM knowledge_base_documents WHERE id = $1`
	_, err = r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Update document count
	_, err = r.pool.Exec(ctx,
		`UPDATE knowledge_bases SET document_count = GREATEST(document_count - 1, 0) WHERE id = $1`,
		doc.KnowledgeBaseID,
	)
	return err
}

// GetDocumentCount returns the number of documents in a knowledge base.
func (r *KnowledgeBaseRepository) GetDocumentCount(ctx context.Context, kbID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM knowledge_base_documents WHERE knowledge_base_id = $1`
	err := r.pool.QueryRow(ctx, query, kbID).Scan(&count)
	return count, err
}

// Helper methods

func (r *KnowledgeBaseRepository) scanKnowledgeBase(ctx context.Context, query string, arg interface{}) (*domain.KnowledgeBase, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	kb := &domain.KnowledgeBase{}
	var metadataJSON string
	err := row.Scan(
		&kb.ID, &kb.BlandID, &kb.Name, &kb.Description, &kb.VectorDBID, &kb.Status,
		&kb.DocumentCount, &kb.LastSyncedAt, &kb.SyncError, &metadataJSON,
		&kb.CreatedAt, &kb.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan knowledge base: %w", err)
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &kb.Metadata); err != nil {
			kb.Metadata = make(map[string]string)
		}
	}
	return kb, nil
}

func (r *KnowledgeBaseRepository) scanRow(rows pgx.Rows) (*domain.KnowledgeBase, error) {
	kb := &domain.KnowledgeBase{}
	var metadataJSON string
	err := rows.Scan(
		&kb.ID, &kb.BlandID, &kb.Name, &kb.Description, &kb.VectorDBID, &kb.Status,
		&kb.DocumentCount, &kb.LastSyncedAt, &kb.SyncError, &metadataJSON,
		&kb.CreatedAt, &kb.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan knowledge base row: %w", err)
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &kb.Metadata); err != nil {
			kb.Metadata = make(map[string]string)
		}
	}
	return kb, nil
}
