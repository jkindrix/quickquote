package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Pathway represents a conversational pathway in Bland.
// Pathways are node-based conversation flows that guide AI agent behavior.
type Pathway struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Nodes       []PathwayNode `json:"nodes,omitempty"`
	Edges       []PathwayEdge `json:"edges,omitempty"`

	// Version control
	Version      int       `json:"version,omitempty"`
	IsProduction bool      `json:"is_production,omitempty"`
	IsDraft      bool      `json:"is_draft,omitempty"`

	// Metadata
	FolderID  string    `json:"folder_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// PathwayNode represents a node in a conversational pathway.
type PathwayNode struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Type string `json:"type"` // default, webhook, knowledge_base, transfer, end_call, wait_for_response

	// Position for UI rendering
	Position *NodePosition `json:"position,omitempty"`

	// Node configuration
	Data *NodeData `json:"data,omitempty"`
}

// NodePosition represents the visual position of a node.
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeData contains the configuration for a pathway node.
type NodeData struct {
	// Prompt/instructions for this node
	Prompt string `json:"prompt,omitempty"`

	// Static text (alternative to prompt-generated responses)
	StaticText string `json:"static_text,omitempty"`

	// Conditions for staying on this node
	Conditions []NodeCondition `json:"conditions,omitempty"`

	// Variables to extract at this node
	Variables []NodeVariable `json:"variables,omitempty"`

	// Decision guide for choosing paths
	DecisionGuide string `json:"decision_guide,omitempty"`

	// Whether this is a global node (accessible from any node)
	IsGlobal bool `json:"is_global,omitempty"`

	// For webhook nodes
	WebhookURL     string            `json:"webhook_url,omitempty"`
	WebhookMethod  string            `json:"webhook_method,omitempty"`
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty"`
	WebhookBody    interface{}       `json:"webhook_body,omitempty"`
	PreWebhookText string            `json:"pre_webhook_text,omitempty"`
	PostWebhookText string           `json:"post_webhook_text,omitempty"`

	// For knowledge base nodes
	KnowledgeBaseID string `json:"knowledge_base_id,omitempty"`

	// For transfer nodes
	TransferNumber string `json:"transfer_number,omitempty"`
	TransferMessage string `json:"transfer_message,omitempty"`

	// For end call nodes
	EndMessage string `json:"end_message,omitempty"`
}

// NodeCondition defines when to stay on or leave a node.
type NodeCondition struct {
	Description string `json:"description"`
	Type        string `json:"type,omitempty"` // required, optional
}

// NodeVariable defines data to extract at a node.
type NodeVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean, date, etc.
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PathwayEdge represents a connection between nodes.
type PathwayEdge struct {
	ID           string `json:"id,omitempty"`
	SourceNodeID string `json:"source"` // From node
	TargetNodeID string `json:"target"` // To node
	Label        string `json:"label,omitempty"` // Condition label
	Condition    string `json:"condition,omitempty"` // When to take this path
}

// PathwayVersion represents a specific version of a pathway.
type PathwayVersion struct {
	PathwayID    string    `json:"pathway_id"`
	Version      int       `json:"version"`
	IsProduction bool      `json:"is_production"`
	CreatedAt    time.Time `json:"created_at"`
	Nodes        []PathwayNode `json:"nodes,omitempty"`
	Edges        []PathwayEdge `json:"edges,omitempty"`
}

// PathwayFolder represents a folder for organizing pathways.
type PathwayFolder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  string    `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// CreatePathwayRequest contains parameters for creating a pathway.
type CreatePathwayRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Nodes       []PathwayNode `json:"nodes,omitempty"`
	Edges       []PathwayEdge `json:"edges,omitempty"`
	FolderID    string        `json:"folder_id,omitempty"`
}

// UpdatePathwayRequest contains parameters for updating a pathway.
type UpdatePathwayRequest struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	Nodes       []PathwayNode `json:"nodes,omitempty"`
	Edges       []PathwayEdge `json:"edges,omitempty"`
	FolderID    *string       `json:"folder_id,omitempty"`
}

// ListPathwaysResponse contains the response from listing pathways.
type ListPathwaysResponse struct {
	Pathways []Pathway `json:"pathways"`
}

// ListPathways retrieves all pathways.
func (c *Client) ListPathways(ctx context.Context) ([]Pathway, error) {
	var resp ListPathwaysResponse
	if err := c.request(ctx, "GET", "/pathways", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Pathways, nil
}

// GetPathway retrieves a specific pathway by ID.
func (c *Client) GetPathway(ctx context.Context, pathwayID string) (*Pathway, error) {
	if pathwayID == "" {
		return nil, fmt.Errorf("pathway_id is required")
	}

	var pathway Pathway
	if err := c.request(ctx, "GET", "/pathways/"+pathwayID, nil, &pathway); err != nil {
		return nil, err
	}

	return &pathway, nil
}

// CreatePathway creates a new pathway.
func (c *Client) CreatePathway(ctx context.Context, req *CreatePathwayRequest) (*Pathway, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	var pathway Pathway
	if err := c.request(ctx, "POST", "/pathways", req, &pathway); err != nil {
		return nil, err
	}

	c.logger.Info("pathway created",
		zap.String("id", pathway.ID),
		zap.String("name", pathway.Name),
	)

	return &pathway, nil
}

// UpdatePathway updates an existing pathway.
func (c *Client) UpdatePathway(ctx context.Context, pathwayID string, req *UpdatePathwayRequest) (*Pathway, error) {
	if pathwayID == "" {
		return nil, fmt.Errorf("pathway_id is required")
	}

	var pathway Pathway
	if err := c.request(ctx, "PATCH", "/pathways/"+pathwayID, req, &pathway); err != nil {
		return nil, err
	}

	c.logger.Info("pathway updated", zap.String("id", pathwayID))

	return &pathway, nil
}

// DeletePathway deletes a pathway.
func (c *Client) DeletePathway(ctx context.Context, pathwayID string) error {
	if pathwayID == "" {
		return fmt.Errorf("pathway_id is required")
	}

	if err := c.request(ctx, "DELETE", "/pathways/"+pathwayID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("pathway deleted", zap.String("id", pathwayID))
	return nil
}

// PublishPathway publishes the draft version of a pathway to production.
func (c *Client) PublishPathway(ctx context.Context, pathwayID string) error {
	if pathwayID == "" {
		return fmt.Errorf("pathway_id is required")
	}

	if err := c.request(ctx, "POST", "/pathways/"+pathwayID+"/publish", nil, nil); err != nil {
		return err
	}

	c.logger.Info("pathway published", zap.String("id", pathwayID))
	return nil
}

// GetPathwayVersions retrieves version history for a pathway.
func (c *Client) GetPathwayVersions(ctx context.Context, pathwayID string) ([]PathwayVersion, error) {
	if pathwayID == "" {
		return nil, fmt.Errorf("pathway_id is required")
	}

	var resp struct {
		Versions []PathwayVersion `json:"versions"`
	}
	if err := c.request(ctx, "GET", "/pathways/"+pathwayID+"/versions", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Versions, nil
}

// GetPathwayVersion retrieves a specific version of a pathway.
func (c *Client) GetPathwayVersion(ctx context.Context, pathwayID string, version int) (*PathwayVersion, error) {
	if pathwayID == "" {
		return nil, fmt.Errorf("pathway_id is required")
	}

	var pv PathwayVersion
	path := fmt.Sprintf("/pathways/%s/versions/%d", pathwayID, version)
	if err := c.request(ctx, "GET", path, nil, &pv); err != nil {
		return nil, err
	}

	return &pv, nil
}

// RevertPathway reverts a pathway to a previous version.
func (c *Client) RevertPathway(ctx context.Context, pathwayID string, version int) error {
	if pathwayID == "" {
		return fmt.Errorf("pathway_id is required")
	}

	req := map[string]int{"version": version}
	path := fmt.Sprintf("/pathways/%s/revert", pathwayID)
	if err := c.request(ctx, "POST", path, req, nil); err != nil {
		return err
	}

	c.logger.Info("pathway reverted",
		zap.String("id", pathwayID),
		zap.Int("version", version),
	)
	return nil
}

// DuplicatePathway creates a copy of a pathway.
func (c *Client) DuplicatePathway(ctx context.Context, pathwayID, newName string) (*Pathway, error) {
	if pathwayID == "" {
		return nil, fmt.Errorf("pathway_id is required")
	}
	if newName == "" {
		return nil, fmt.Errorf("new_name is required")
	}

	req := map[string]string{"name": newName}
	var pathway Pathway
	if err := c.request(ctx, "POST", "/pathways/"+pathwayID+"/duplicate", req, &pathway); err != nil {
		return nil, err
	}

	c.logger.Info("pathway duplicated",
		zap.String("original_id", pathwayID),
		zap.String("new_id", pathway.ID),
		zap.String("new_name", newName),
	)

	return &pathway, nil
}

// Folder Operations

// ListPathwayFolders retrieves all pathway folders.
func (c *Client) ListPathwayFolders(ctx context.Context) ([]PathwayFolder, error) {
	var resp struct {
		Folders []PathwayFolder `json:"folders"`
	}
	if err := c.request(ctx, "GET", "/pathways/folders", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Folders, nil
}

// CreatePathwayFolder creates a new folder for organizing pathways.
func (c *Client) CreatePathwayFolder(ctx context.Context, name, parentID string) (*PathwayFolder, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	req := map[string]string{
		"name": name,
	}
	if parentID != "" {
		req["parent_id"] = parentID
	}

	var folder PathwayFolder
	if err := c.request(ctx, "POST", "/pathways/folders", req, &folder); err != nil {
		return nil, err
	}

	return &folder, nil
}

// DeletePathwayFolder deletes an empty folder.
func (c *Client) DeletePathwayFolder(ctx context.Context, folderID string) error {
	if folderID == "" {
		return fmt.Errorf("folder_id is required")
	}

	return c.request(ctx, "DELETE", "/pathways/folders/"+folderID, nil, nil)
}

// Helper functions for building pathways

// NewDefaultNode creates a new default/base node.
func NewDefaultNode(id, name, prompt string) PathwayNode {
	return PathwayNode{
		ID:   id,
		Name: name,
		Type: "default",
		Data: &NodeData{
			Prompt: prompt,
		},
	}
}

// NewWebhookNode creates a new webhook node.
func NewWebhookNode(id, name, webhookURL, method string) PathwayNode {
	return PathwayNode{
		ID:   id,
		Name: name,
		Type: "webhook",
		Data: &NodeData{
			WebhookURL:    webhookURL,
			WebhookMethod: method,
		},
	}
}

// NewKnowledgeBaseNode creates a new knowledge base node.
func NewKnowledgeBaseNode(id, name, kbID string) PathwayNode {
	return PathwayNode{
		ID:   id,
		Name: name,
		Type: "knowledge_base",
		Data: &NodeData{
			KnowledgeBaseID: kbID,
		},
	}
}

// NewTransferNode creates a new transfer node.
func NewTransferNode(id, name, transferNumber, message string) PathwayNode {
	return PathwayNode{
		ID:   id,
		Name: name,
		Type: "transfer",
		Data: &NodeData{
			TransferNumber:  transferNumber,
			TransferMessage: message,
		},
	}
}

// NewEndCallNode creates a new end call node.
func NewEndCallNode(id, name, endMessage string) PathwayNode {
	return PathwayNode{
		ID:   id,
		Name: name,
		Type: "end_call",
		Data: &NodeData{
			EndMessage: endMessage,
		},
	}
}

// NewEdge creates a new edge connecting two nodes.
func NewEdge(sourceID, targetID, label, condition string) PathwayEdge {
	return PathwayEdge{
		SourceNodeID: sourceID,
		TargetNodeID: targetID,
		Label:        label,
		Condition:    condition,
	}
}
