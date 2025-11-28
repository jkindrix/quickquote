package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Pathway represents a local cache of a Bland conversational pathway.
// Pathways define the flow and logic of AI-driven conversations.
type Pathway struct {
	ID           uuid.UUID `json:"id" db:"id"`
	BlandID      string    `json:"bland_id,omitempty" db:"bland_id"` // Bland's pathway ID
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description,omitempty" db:"description"`
	Version      int       `json:"version" db:"version"`
	Status       string    `json:"status" db:"status"`
	NodesJSON    string    `json:"-" db:"nodes"`                    // Stored as JSON
	EdgesJSON    string    `json:"-" db:"edges"`                    // Stored as JSON
	Nodes        []PathwayNode `json:"nodes,omitempty" db:"-"`
	Edges        []PathwayEdge `json:"edges,omitempty" db:"-"`
	StartNodeID  string    `json:"start_node_id,omitempty" db:"start_node_id"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty" db:"last_synced_at"`
	SyncError    string    `json:"sync_error,omitempty" db:"sync_error"`
	IsPublished  bool      `json:"is_published" db:"is_published"`
	PublishedAt  *time.Time `json:"published_at,omitempty" db:"published_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// PathwayStatus constants.
const (
	PathwayStatusDraft     = "draft"
	PathwayStatusActive    = "active"
	PathwayStatusSyncing   = "syncing"
	PathwayStatusError     = "error"
	PathwayStatusArchived  = "archived"
)

// PathwayNode represents a node in a pathway.
type PathwayNode struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`                    // default, webhook, knowledge_base, transfer, end
	Prompt      string                 `json:"prompt,omitempty"`
	Condition   string                 `json:"condition,omitempty"`     // Condition for entering this node
	Tools       []string               `json:"tools,omitempty"`
	KnowledgeBases []string            `json:"knowledge_bases,omitempty"`
	WebhookURL  string                 `json:"webhook_url,omitempty"`
	TransferTo  string                 `json:"transfer_to,omitempty"`   // Phone number for transfer
	IsTerminal  bool                   `json:"is_terminal,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Position    *NodePosition          `json:"position,omitempty"`      // For visual editor
}

// NodePosition defines the visual position of a node in an editor.
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PathwayEdge represents a connection between nodes.
type PathwayEdge struct {
	ID         string `json:"id"`
	SourceID   string `json:"source"`
	TargetID   string `json:"target"`
	Label      string `json:"label,omitempty"`
	Condition  string `json:"condition,omitempty"`
	Priority   int    `json:"priority,omitempty"`
}

// PathwayNodeType constants.
const (
	NodeTypeDefault       = "default"
	NodeTypeWebhook       = "webhook"
	NodeTypeKnowledgeBase = "knowledge_base"
	NodeTypeTransfer      = "transfer"
	NodeTypeEnd           = "end"
	NodeTypeConditional   = "conditional"
	NodeTypeInput         = "input"
)

// PathwayFilter contains filtering options for listing pathways.
type PathwayFilter struct {
	Status      string
	Name        string
	IsPublished *bool
	Limit       int
	Offset      int
}

// PathwayVersion represents a version history entry.
type PathwayVersion struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PathwayID   uuid.UUID `json:"pathway_id" db:"pathway_id"`
	Version     int       `json:"version" db:"version"`
	NodesJSON   string    `json:"-" db:"nodes"`
	EdgesJSON   string    `json:"-" db:"edges"`
	ChangeNotes string    `json:"change_notes,omitempty" db:"change_notes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   string    `json:"created_by,omitempty" db:"created_by"`
}

// PathwayRepository defines the interface for pathway persistence.
type PathwayRepository interface {
	// Core CRUD
	Create(ctx context.Context, pathway *Pathway) error
	GetByID(ctx context.Context, id uuid.UUID) (*Pathway, error)
	GetByBlandID(ctx context.Context, blandID string) (*Pathway, error)
	List(ctx context.Context, filter *PathwayFilter) ([]*Pathway, error)
	Update(ctx context.Context, pathway *Pathway) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Versioning
	SaveVersion(ctx context.Context, version *PathwayVersion) error
	GetVersion(ctx context.Context, pathwayID uuid.UUID, version int) (*PathwayVersion, error)
	ListVersions(ctx context.Context, pathwayID uuid.UUID) ([]*PathwayVersion, error)
	RestoreVersion(ctx context.Context, pathwayID uuid.UUID, version int) error

	// Sync operations
	MarkSyncing(ctx context.Context, id uuid.UUID) error
	MarkSynced(ctx context.Context, id uuid.UUID, blandID string) error
	MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error

	// Publishing
	Publish(ctx context.Context, id uuid.UUID) error
	Unpublish(ctx context.Context, id uuid.UUID) error
}

// NewPathway creates a new pathway with generated ID.
func NewPathway(name, description string) *Pathway {
	now := time.Now()
	return &Pathway{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Version:     1,
		Status:      PathwayStatusDraft,
		Nodes:       []PathwayNode{},
		Edges:       []PathwayEdge{},
		IsPublished: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddNode adds a node to the pathway.
func (p *Pathway) AddNode(node PathwayNode) {
	p.Nodes = append(p.Nodes, node)
	p.UpdatedAt = time.Now()
}

// AddEdge adds an edge between nodes.
func (p *Pathway) AddEdge(edge PathwayEdge) {
	p.Edges = append(p.Edges, edge)
	p.UpdatedAt = time.Now()
}

// RemoveNode removes a node and its connected edges.
func (p *Pathway) RemoveNode(nodeID string) {
	// Remove the node
	var newNodes []PathwayNode
	for _, n := range p.Nodes {
		if n.ID != nodeID {
			newNodes = append(newNodes, n)
		}
	}
	p.Nodes = newNodes

	// Remove connected edges
	var newEdges []PathwayEdge
	for _, e := range p.Edges {
		if e.SourceID != nodeID && e.TargetID != nodeID {
			newEdges = append(newEdges, e)
		}
	}
	p.Edges = newEdges
	p.UpdatedAt = time.Now()
}

// GetNode retrieves a node by ID.
func (p *Pathway) GetNode(nodeID string) *PathwayNode {
	for i := range p.Nodes {
		if p.Nodes[i].ID == nodeID {
			return &p.Nodes[i]
		}
	}
	return nil
}

// GetStartNode returns the starting node of the pathway.
func (p *Pathway) GetStartNode() *PathwayNode {
	if p.StartNodeID != "" {
		return p.GetNode(p.StartNodeID)
	}
	// Default to first node if no start node specified
	if len(p.Nodes) > 0 {
		return &p.Nodes[0]
	}
	return nil
}

// GetOutgoingEdges returns edges originating from a node.
func (p *Pathway) GetOutgoingEdges(nodeID string) []PathwayEdge {
	var edges []PathwayEdge
	for _, e := range p.Edges {
		if e.SourceID == nodeID {
			edges = append(edges, e)
		}
	}
	return edges
}

// GetIncomingEdges returns edges terminating at a node.
func (p *Pathway) GetIncomingEdges(nodeID string) []PathwayEdge {
	var edges []PathwayEdge
	for _, e := range p.Edges {
		if e.TargetID == nodeID {
			edges = append(edges, e)
		}
	}
	return edges
}

// Validate checks if the pathway is valid.
func (p *Pathway) Validate() []string {
	var errors []string

	if p.Name == "" {
		errors = append(errors, "pathway name is required")
	}

	if len(p.Nodes) == 0 {
		errors = append(errors, "pathway must have at least one node")
	}

	// Check for orphan nodes (no incoming or outgoing edges, not start node)
	for _, node := range p.Nodes {
		if node.ID == p.StartNodeID {
			continue
		}
		hasIncoming := false
		hasOutgoing := false
		for _, edge := range p.Edges {
			if edge.TargetID == node.ID {
				hasIncoming = true
			}
			if edge.SourceID == node.ID {
				hasOutgoing = true
			}
		}
		if !hasIncoming && !node.IsTerminal {
			errors = append(errors, "node "+node.ID+" has no incoming edges")
		}
		if !hasOutgoing && !node.IsTerminal {
			errors = append(errors, "node "+node.ID+" has no outgoing edges")
		}
	}

	// Check edge references
	nodeIDs := make(map[string]bool)
	for _, node := range p.Nodes {
		nodeIDs[node.ID] = true
	}
	for _, edge := range p.Edges {
		if !nodeIDs[edge.SourceID] {
			errors = append(errors, "edge references non-existent source node: "+edge.SourceID)
		}
		if !nodeIDs[edge.TargetID] {
			errors = append(errors, "edge references non-existent target node: "+edge.TargetID)
		}
	}

	return errors
}

// IsDraft checks if the pathway is in draft status.
func (p *Pathway) IsDraft() bool {
	return p.Status == PathwayStatusDraft
}

// IsActive checks if the pathway is active.
func (p *Pathway) IsActive() bool {
	return p.Status == PathwayStatusActive
}

// NeedsSync checks if the pathway needs to be synced to Bland.
func (p *Pathway) NeedsSync() bool {
	if p.BlandID == "" {
		return true
	}
	if p.LastSyncedAt == nil {
		return true
	}
	return p.UpdatedAt.After(*p.LastSyncedAt)
}

// SetSynced marks the pathway as successfully synced.
func (p *Pathway) SetSynced(blandID string) {
	now := time.Now()
	p.BlandID = blandID
	p.Status = PathwayStatusActive
	p.LastSyncedAt = &now
	p.SyncError = ""
	p.UpdatedAt = now
}

// MarshalNodes converts nodes to JSON for storage.
func (p *Pathway) MarshalNodes() error {
	data, err := json.Marshal(p.Nodes)
	if err != nil {
		return err
	}
	p.NodesJSON = string(data)
	return nil
}

// UnmarshalNodes parses nodes from JSON storage.
func (p *Pathway) UnmarshalNodes() error {
	if p.NodesJSON == "" {
		p.Nodes = []PathwayNode{}
		return nil
	}
	return json.Unmarshal([]byte(p.NodesJSON), &p.Nodes)
}

// MarshalEdges converts edges to JSON for storage.
func (p *Pathway) MarshalEdges() error {
	data, err := json.Marshal(p.Edges)
	if err != nil {
		return err
	}
	p.EdgesJSON = string(data)
	return nil
}

// UnmarshalEdges parses edges from JSON storage.
func (p *Pathway) UnmarshalEdges() error {
	if p.EdgesJSON == "" {
		p.Edges = []PathwayEdge{}
		return nil
	}
	return json.Unmarshal([]byte(p.EdgesJSON), &p.Edges)
}

// Helper functions for creating common node types

// NewDefaultNode creates a standard conversation node.
func NewDefaultNode(id, name, prompt string) PathwayNode {
	return PathwayNode{
		ID:     id,
		Name:   name,
		Type:   NodeTypeDefault,
		Prompt: prompt,
	}
}

// NewWebhookNode creates a node that calls a webhook.
func NewWebhookNode(id, name, webhookURL string) PathwayNode {
	return PathwayNode{
		ID:         id,
		Name:       name,
		Type:       NodeTypeWebhook,
		WebhookURL: webhookURL,
	}
}

// NewKnowledgeBaseNode creates a node that uses knowledge bases.
func NewKnowledgeBaseNode(id, name string, kbIDs []string) PathwayNode {
	return PathwayNode{
		ID:             id,
		Name:           name,
		Type:           NodeTypeKnowledgeBase,
		KnowledgeBases: kbIDs,
	}
}

// NewTransferNode creates a node that transfers the call.
func NewTransferNode(id, name, transferTo string) PathwayNode {
	return PathwayNode{
		ID:         id,
		Name:       name,
		Type:       NodeTypeTransfer,
		TransferTo: transferTo,
		IsTerminal: true,
	}
}

// NewEndNode creates a terminal node.
func NewEndNode(id, name string) PathwayNode {
	return PathwayNode{
		ID:         id,
		Name:       name,
		Type:       NodeTypeEnd,
		IsTerminal: true,
	}
}
