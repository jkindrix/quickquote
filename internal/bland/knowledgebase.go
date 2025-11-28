package bland

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"go.uber.org/zap"
)

// KnowledgeBase represents a vectorized knowledge store in Bland.
type KnowledgeBase struct {
	VectorID    string    `json:"vector_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Text        string    `json:"text,omitempty"` // Only included if include-text=true
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// CreateKnowledgeBaseRequest contains parameters for creating a knowledge base.
type CreateKnowledgeBaseRequest struct {
	// Name: A clear name that describes the contents
	Name string `json:"name"`

	// Description: Visible to AI, helps it understand when to use this KB
	Description string `json:"description"`

	// Text: The full text document to be vectorized
	Text string `json:"text"`
}

// CreateKnowledgeBaseResponse contains the response from creating a KB.
type CreateKnowledgeBaseResponse struct {
	VectorID string `json:"vector_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// UpdateKnowledgeBaseRequest contains parameters for updating a knowledge base.
type UpdateKnowledgeBaseRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Text        *string `json:"text,omitempty"`
}

// ListKnowledgeBasesResponse contains the response from listing KBs.
type ListKnowledgeBasesResponse struct {
	Vectors []KnowledgeBase `json:"vectors"`
}

// ListKnowledgeBases retrieves all knowledge bases.
func (c *Client) ListKnowledgeBases(ctx context.Context) ([]KnowledgeBase, error) {
	var resp ListKnowledgeBasesResponse
	if err := c.request(ctx, "GET", "/knowledgebases", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Vectors, nil
}

// ListKnowledgeBasesWithText retrieves all KBs including full text content.
func (c *Client) ListKnowledgeBasesWithText(ctx context.Context) ([]KnowledgeBase, error) {
	var resp ListKnowledgeBasesResponse
	if err := c.request(ctx, "GET", "/knowledgebases?include-text=true", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Vectors, nil
}

// GetKnowledgeBase retrieves a specific knowledge base by ID.
func (c *Client) GetKnowledgeBase(ctx context.Context, vectorID string) (*KnowledgeBase, error) {
	if vectorID == "" {
		return nil, fmt.Errorf("vector_id is required")
	}

	var kb KnowledgeBase
	if err := c.request(ctx, "GET", "/knowledgebases/"+vectorID, nil, &kb); err != nil {
		return nil, err
	}

	return &kb, nil
}

// CreateKnowledgeBase creates a new knowledge base from text.
func (c *Client) CreateKnowledgeBase(ctx context.Context, req *CreateKnowledgeBaseRequest) (*CreateKnowledgeBaseResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	var resp CreateKnowledgeBaseResponse
	if err := c.request(ctx, "POST", "/knowledgebases", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("knowledge base created",
		zap.String("vector_id", resp.VectorID),
		zap.String("name", req.Name),
	)

	return &resp, nil
}

// CreateKnowledgeBaseFromFile creates a knowledge base from an audio/video file.
func (c *Client) CreateKnowledgeBaseFromFile(ctx context.Context, name, description string, file io.Reader, filename string) (*CreateKnowledgeBaseResponse, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if file == nil {
		return nil, fmt.Errorf("file is required")
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add name field
	if err := writer.WriteField("name", name); err != nil {
		return nil, fmt.Errorf("failed to write name field: %w", err)
	}

	// Add description if provided
	if description != "" {
		if err := writer.WriteField("description", description); err != nil {
			return nil, fmt.Errorf("failed to write description field: %w", err)
		}
	}

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	var resp CreateKnowledgeBaseResponse
	if err := c.requestMultipart(ctx, "/knowledgebases/upload-media", &buf, writer.FormDataContentType(), &resp); err != nil {
		return nil, err
	}

	c.logger.Info("knowledge base created from file",
		zap.String("vector_id", resp.VectorID),
		zap.String("name", name),
		zap.String("filename", filename),
	)

	return &resp, nil
}

// UpdateKnowledgeBase updates an existing knowledge base.
func (c *Client) UpdateKnowledgeBase(ctx context.Context, vectorID string, req *UpdateKnowledgeBaseRequest) error {
	if vectorID == "" {
		return fmt.Errorf("vector_id is required")
	}

	if err := c.request(ctx, "PATCH", "/knowledgebases/"+vectorID, req, nil); err != nil {
		return err
	}

	c.logger.Info("knowledge base updated", zap.String("vector_id", vectorID))
	return nil
}

// DeleteKnowledgeBase soft-deletes a knowledge base.
func (c *Client) DeleteKnowledgeBase(ctx context.Context, vectorID string) error {
	if vectorID == "" {
		return fmt.Errorf("vector_id is required")
	}

	if err := c.request(ctx, "DELETE", "/knowledgebases/"+vectorID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("knowledge base deleted", zap.String("vector_id", vectorID))
	return nil
}

// AppendToKnowledgeBase adds more text to an existing knowledge base.
func (c *Client) AppendToKnowledgeBase(ctx context.Context, vectorID, additionalText string) error {
	if vectorID == "" {
		return fmt.Errorf("vector_id is required")
	}
	if additionalText == "" {
		return fmt.Errorf("additional_text is required")
	}

	// Get existing KB
	kb, err := c.GetKnowledgeBase(ctx, vectorID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge base: %w", err)
	}

	// Append text (if we have existing text)
	newText := kb.Text + "\n\n" + additionalText
	if kb.Text == "" {
		newText = additionalText
	}

	// Update with combined text
	return c.UpdateKnowledgeBase(ctx, vectorID, &UpdateKnowledgeBaseRequest{
		Text: &newText,
	})
}

// SearchKnowledgeBase queries a knowledge base (if Bland supports this directly).
// Note: Typically the AI agent searches during calls, but this can be useful for testing.
func (c *Client) SearchKnowledgeBase(ctx context.Context, vectorID, query string) ([]string, error) {
	if vectorID == "" {
		return nil, fmt.Errorf("vector_id is required")
	}
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	req := map[string]string{
		"query": query,
	}

	var resp struct {
		Results []string `json:"results"`
	}
	if err := c.request(ctx, "POST", "/knowledgebases/"+vectorID+"/search", req, &resp); err != nil {
		// This endpoint may not exist - return empty results
		c.logger.Debug("knowledge base search not available", zap.Error(err))
		return nil, nil
	}

	return resp.Results, nil
}
