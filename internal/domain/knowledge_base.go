package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// KnowledgeBase represents a local cache of a Bland knowledge base.
// This allows for local management, versioning, and synchronization
// with the Bland API.
type KnowledgeBase struct {
	ID              uuid.UUID         `json:"id" db:"id"`
	BlandID         string            `json:"bland_id" db:"bland_id"`           // Bland's KB ID
	Name            string            `json:"name" db:"name"`
	Description     string            `json:"description,omitempty" db:"description"`
	VectorDBID      string            `json:"vector_db_id,omitempty" db:"vector_db_id"`
	Status          string            `json:"status" db:"status"`               // active, syncing, error
	DocumentCount   int               `json:"document_count" db:"document_count"`
	LastSyncedAt    *time.Time        `json:"last_synced_at,omitempty" db:"last_synced_at"`
	SyncError       string            `json:"sync_error,omitempty" db:"sync_error"`
	Metadata        map[string]string `json:"metadata,omitempty" db:"-"`        // Stored as JSON
	MetadataJSON    string            `json:"-" db:"metadata"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the knowledge base has been soft-deleted.
func (kb *KnowledgeBase) IsDeleted() bool {
	return kb.DeletedAt != nil
}

// MarkDeleted soft-deletes the knowledge base by setting DeletedAt.
func (kb *KnowledgeBase) MarkDeleted() {
	now := time.Now().UTC()
	kb.DeletedAt = &now
	kb.UpdatedAt = now
}

// KnowledgeBaseStatus constants.
const (
	KnowledgeBaseStatusActive   = "active"
	KnowledgeBaseStatusSyncing  = "syncing"
	KnowledgeBaseStatusError    = "error"
	KnowledgeBaseStatusPending  = "pending"
	KnowledgeBaseStatusArchived = "archived"
)

// KnowledgeBaseDocument represents a document within a knowledge base.
type KnowledgeBaseDocument struct {
	ID             uuid.UUID `json:"id" db:"id"`
	KnowledgeBaseID uuid.UUID `json:"knowledge_base_id" db:"knowledge_base_id"`
	BlandDocID     string    `json:"bland_doc_id,omitempty" db:"bland_doc_id"`
	Name           string    `json:"name" db:"name"`
	ContentType    string    `json:"content_type" db:"content_type"` // text, pdf, html, markdown
	ContentHash    string    `json:"content_hash" db:"content_hash"`
	SizeBytes      int64     `json:"size_bytes" db:"size_bytes"`
	ChunkCount     int       `json:"chunk_count,omitempty" db:"chunk_count"`
	Status         string    `json:"status" db:"status"`
	ErrorMessage   string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// DocumentStatus constants.
const (
	DocumentStatusPending    = "pending"
	DocumentStatusProcessing = "processing"
	DocumentStatusReady      = "ready"
	DocumentStatusError      = "error"
)

// KnowledgeBaseFilter contains filtering options for listing knowledge bases.
type KnowledgeBaseFilter struct {
	Status string
	Name   string
	Limit  int
	Offset int
}

// KnowledgeBaseRepository defines the interface for knowledge base persistence.
type KnowledgeBaseRepository interface {
	// Core CRUD
	Create(ctx context.Context, kb *KnowledgeBase) error
	GetByID(ctx context.Context, id uuid.UUID) (*KnowledgeBase, error)
	GetByBlandID(ctx context.Context, blandID string) (*KnowledgeBase, error)
	List(ctx context.Context, filter *KnowledgeBaseFilter) ([]*KnowledgeBase, error)
	Update(ctx context.Context, kb *KnowledgeBase) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Sync operations
	MarkSyncing(ctx context.Context, id uuid.UUID) error
	MarkSynced(ctx context.Context, id uuid.UUID) error
	MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error

	// Document operations
	AddDocument(ctx context.Context, doc *KnowledgeBaseDocument) error
	GetDocument(ctx context.Context, id uuid.UUID) (*KnowledgeBaseDocument, error)
	ListDocuments(ctx context.Context, kbID uuid.UUID) ([]*KnowledgeBaseDocument, error)
	UpdateDocumentStatus(ctx context.Context, docID uuid.UUID, status, errMsg string) error
	DeleteDocument(ctx context.Context, id uuid.UUID) error

	// Stats
	GetDocumentCount(ctx context.Context, kbID uuid.UUID) (int, error)
}

// NewKnowledgeBase creates a new knowledge base with generated ID.
func NewKnowledgeBase(name, description string) *KnowledgeBase {
	now := time.Now()
	return &KnowledgeBase{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Status:      KnowledgeBaseStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsActive checks if the knowledge base is in active status.
func (kb *KnowledgeBase) IsActive() bool {
	return kb.Status == KnowledgeBaseStatusActive
}

// IsSyncing checks if the knowledge base is currently syncing.
func (kb *KnowledgeBase) IsSyncing() bool {
	return kb.Status == KnowledgeBaseStatusSyncing
}

// HasError checks if the knowledge base has a sync error.
func (kb *KnowledgeBase) HasError() bool {
	return kb.Status == KnowledgeBaseStatusError
}

// NeedsSync checks if the knowledge base needs to be synced to Bland.
func (kb *KnowledgeBase) NeedsSync() bool {
	// No Bland ID means never synced
	if kb.BlandID == "" {
		return true
	}

	// Updated since last sync
	if kb.LastSyncedAt == nil {
		return true
	}

	return kb.UpdatedAt.After(*kb.LastSyncedAt)
}

// SetSynced marks the knowledge base as successfully synced.
func (kb *KnowledgeBase) SetSynced(blandID string) {
	now := time.Now()
	kb.BlandID = blandID
	kb.Status = KnowledgeBaseStatusActive
	kb.LastSyncedAt = &now
	kb.SyncError = ""
	kb.UpdatedAt = now
}

// SetSyncError marks the knowledge base as having a sync error.
func (kb *KnowledgeBase) SetSyncError(err string) {
	kb.Status = KnowledgeBaseStatusError
	kb.SyncError = err
	kb.UpdatedAt = time.Now()
}

// NewKnowledgeBaseDocument creates a new document.
func NewKnowledgeBaseDocument(kbID uuid.UUID, name, contentType string) *KnowledgeBaseDocument {
	now := time.Now()
	return &KnowledgeBaseDocument{
		ID:             uuid.New(),
		KnowledgeBaseID: kbID,
		Name:           name,
		ContentType:    contentType,
		Status:         DocumentStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
