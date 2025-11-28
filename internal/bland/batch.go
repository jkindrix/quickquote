package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Batch represents a batch of calls.
type Batch struct {
	ID            string       `json:"batch_id"`
	Name          string       `json:"name,omitempty"`
	Status        string       `json:"status"` // created, in_progress, completed, paused, cancelled
	TotalCalls    int          `json:"total_calls"`
	CompletedCalls int         `json:"completed_calls"`
	FailedCalls   int          `json:"failed_calls"`
	InProgressCalls int        `json:"in_progress_calls,omitempty"`
	PendingCalls  int          `json:"pending_calls,omitempty"`
	BasePrompt    string       `json:"base_prompt,omitempty"`
	CallParams    *SendCallRequest `json:"call_params,omitempty"`
	CreatedAt     time.Time    `json:"created_at,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at,omitempty"`
	CompletedAt   *time.Time   `json:"completed_at,omitempty"`
}

// BatchCall represents a single call within a batch.
type BatchCall struct {
	CallID      string                 `json:"call_id"`
	PhoneNumber string                 `json:"phone_number"`
	Status      string                 `json:"status"` // pending, in_progress, completed, failed
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	EndedAt     *time.Time             `json:"ended_at,omitempty"`
	Duration    float64                `json:"duration,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// BatchCallTarget represents a target for a batch call.
type BatchCallTarget struct {
	PhoneNumber string                 `json:"phone_number"`
	Variables   map[string]interface{} `json:"variables,omitempty"` // Per-call variable substitution
}

// CreateBatchRequest contains parameters for creating a batch.
type CreateBatchRequest struct {
	// Name: Optional name for the batch
	Name string `json:"name,omitempty"`

	// BasePrompt: The base task/prompt (can use {{variable}} substitution)
	BasePrompt string `json:"base_prompt,omitempty"`

	// Calls: List of call targets with optional per-call variables
	Calls []BatchCallTarget `json:"calls"`

	// CallParams: Shared call parameters for all calls in batch
	// These are the same parameters as SendCallRequest
	Voice             string  `json:"voice,omitempty"`
	PathwayID         string  `json:"pathway_id,omitempty"`
	Model             string  `json:"model,omitempty"`
	Language          string  `json:"language,omitempty"`
	MaxDuration       int     `json:"max_duration,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	WaitForGreeting   bool    `json:"wait_for_greeting,omitempty"`
	Record            bool    `json:"record,omitempty"`
	WebhookURL        string  `json:"webhook,omitempty"`
	WebhookEvents     []string `json:"webhook_events,omitempty"`
	AnalyzeAfter      bool    `json:"analyze,omitempty"`
	SummaryPrompt     string  `json:"summary_prompt,omitempty"`

	// Scheduling
	ScheduledTime     *time.Time `json:"scheduled_time,omitempty"`
	CallsPerMinute    int        `json:"calls_per_minute,omitempty"` // Rate limiting
	MaxConcurrentCalls int       `json:"max_concurrent_calls,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateBatchResponse contains the response from creating a batch.
type CreateBatchResponse struct {
	BatchID     string `json:"batch_id"`
	Status      string `json:"status"`
	TotalCalls  int    `json:"total_calls"`
	Message     string `json:"message,omitempty"`
}

// UpdateBatchRequest contains parameters for updating a batch.
type UpdateBatchRequest struct {
	Name           *string    `json:"name,omitempty"`
	Status         *string    `json:"status,omitempty"` // pause, resume, cancel
	ScheduledTime  *time.Time `json:"scheduled_time,omitempty"`
	CallsPerMinute *int       `json:"calls_per_minute,omitempty"`
}

// ListBatchesResponse contains the response from listing batches.
type ListBatchesResponse struct {
	Batches []Batch `json:"batches"`
	Total   int     `json:"total,omitempty"`
}

// BatchAnalytics contains analytics for a batch.
type BatchAnalytics struct {
	BatchID          string  `json:"batch_id"`
	TotalCalls       int     `json:"total_calls"`
	CompletedCalls   int     `json:"completed_calls"`
	FailedCalls      int     `json:"failed_calls"`
	AverageDuration  float64 `json:"average_duration"`
	TotalDuration    float64 `json:"total_duration"`
	SuccessRate      float64 `json:"success_rate"`
	CompletionRate   float64 `json:"completion_rate"`
	// Call outcome breakdowns
	AnsweredCalls    int     `json:"answered_calls,omitempty"`
	VoicemailCalls   int     `json:"voicemail_calls,omitempty"`
	NoAnswerCalls    int     `json:"no_answer_calls,omitempty"`
	BusyCalls        int     `json:"busy_calls,omitempty"`
}

// CreateBatch creates a new batch of calls.
func (c *Client) CreateBatch(ctx context.Context, req *CreateBatchRequest) (*CreateBatchResponse, error) {
	if len(req.Calls) == 0 {
		return nil, fmt.Errorf("at least one call target is required")
	}

	var resp CreateBatchResponse
	if err := c.request(ctx, "POST", "/batches", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("batch created",
		zap.String("batch_id", resp.BatchID),
		zap.Int("total_calls", resp.TotalCalls),
	)

	return &resp, nil
}

// GetBatch retrieves details for a specific batch.
func (c *Client) GetBatch(ctx context.Context, batchID string) (*Batch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	var batch Batch
	if err := c.request(ctx, "GET", "/batches/"+batchID, nil, &batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

// ListBatches retrieves all batches.
func (c *Client) ListBatches(ctx context.Context, limit, offset int) (*ListBatchesResponse, error) {
	path := fmt.Sprintf("/batches?limit=%d&offset=%d", limit, offset)

	var resp ListBatchesResponse
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateBatch updates a batch (pause, resume, cancel).
func (c *Client) UpdateBatch(ctx context.Context, batchID string, req *UpdateBatchRequest) (*Batch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	var batch Batch
	if err := c.request(ctx, "PATCH", "/batches/"+batchID, req, &batch); err != nil {
		return nil, err
	}

	c.logger.Info("batch updated", zap.String("batch_id", batchID))
	return &batch, nil
}

// PauseBatch pauses a running batch.
func (c *Client) PauseBatch(ctx context.Context, batchID string) error {
	status := "paused"
	_, err := c.UpdateBatch(ctx, batchID, &UpdateBatchRequest{Status: &status})
	if err != nil {
		return err
	}

	c.logger.Info("batch paused", zap.String("batch_id", batchID))
	return nil
}

// ResumeBatch resumes a paused batch.
func (c *Client) ResumeBatch(ctx context.Context, batchID string) error {
	status := "in_progress"
	_, err := c.UpdateBatch(ctx, batchID, &UpdateBatchRequest{Status: &status})
	if err != nil {
		return err
	}

	c.logger.Info("batch resumed", zap.String("batch_id", batchID))
	return nil
}

// CancelBatch cancels a batch.
func (c *Client) CancelBatch(ctx context.Context, batchID string) error {
	status := "cancelled"
	_, err := c.UpdateBatch(ctx, batchID, &UpdateBatchRequest{Status: &status})
	if err != nil {
		return err
	}

	c.logger.Info("batch cancelled", zap.String("batch_id", batchID))
	return nil
}

// DeleteBatch deletes a batch.
func (c *Client) DeleteBatch(ctx context.Context, batchID string) error {
	if batchID == "" {
		return fmt.Errorf("batch_id is required")
	}

	if err := c.request(ctx, "DELETE", "/batches/"+batchID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("batch deleted", zap.String("batch_id", batchID))
	return nil
}

// GetBatchCalls retrieves all calls in a batch.
func (c *Client) GetBatchCalls(ctx context.Context, batchID string, limit, offset int) ([]BatchCall, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	path := fmt.Sprintf("/batches/%s/calls?limit=%d&offset=%d", batchID, limit, offset)

	var resp struct {
		Calls []BatchCall `json:"calls"`
	}
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Calls, nil
}

// GetBatchAnalytics retrieves analytics for a batch.
func (c *Client) GetBatchAnalytics(ctx context.Context, batchID string) (*BatchAnalytics, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	var analytics BatchAnalytics
	if err := c.request(ctx, "GET", "/batches/"+batchID+"/analytics", nil, &analytics); err != nil {
		return nil, err
	}

	return &analytics, nil
}

// AddCallsToBatch adds more calls to an existing batch.
func (c *Client) AddCallsToBatch(ctx context.Context, batchID string, calls []BatchCallTarget) error {
	if batchID == "" {
		return fmt.Errorf("batch_id is required")
	}
	if len(calls) == 0 {
		return fmt.Errorf("at least one call target is required")
	}

	req := map[string]interface{}{
		"calls": calls,
	}

	if err := c.request(ctx, "POST", "/batches/"+batchID+"/calls", req, nil); err != nil {
		return err
	}

	c.logger.Info("calls added to batch",
		zap.String("batch_id", batchID),
		zap.Int("count", len(calls)),
	)
	return nil
}

// Helper functions for batch operations

// CreateQuoteBatch creates a batch of calls for quote follow-ups.
func (c *Client) CreateQuoteBatch(ctx context.Context, name string, targets []BatchCallTarget, basePrompt string, opts *SendCallRequest) (*CreateBatchResponse, error) {
	req := &CreateBatchRequest{
		Name:       name,
		BasePrompt: basePrompt,
		Calls:      targets,
		Record:     true,
		AnalyzeAfter: true,
	}

	// Apply optional call parameters
	if opts != nil {
		req.Voice = opts.Voice
		req.PathwayID = opts.PathwayID
		req.Model = opts.Model
		req.Language = opts.Language
		if opts.MaxDuration != nil {
			req.MaxDuration = *opts.MaxDuration
		}
		if opts.Temperature != nil {
			req.Temperature = *opts.Temperature
		}
		req.WaitForGreeting = opts.WaitForGreeting
		req.WebhookURL = opts.Webhook
		req.WebhookEvents = opts.WebhookEvents
		req.SummaryPrompt = opts.SummaryPrompt
		req.Metadata = opts.Metadata
	}

	return c.CreateBatch(ctx, req)
}

// BuildBatchTargets converts a list of phone numbers to batch targets.
func BuildBatchTargets(phoneNumbers []string, sharedVars map[string]interface{}) []BatchCallTarget {
	targets := make([]BatchCallTarget, len(phoneNumbers))
	for i, phone := range phoneNumbers {
		targets[i] = BatchCallTarget{
			PhoneNumber: phone,
			Variables:   sharedVars,
		}
	}
	return targets
}

// BuildPersonalizedTargets creates targets with per-call personalization.
func BuildPersonalizedTargets(contacts []map[string]interface{}, phoneField string) []BatchCallTarget {
	targets := make([]BatchCallTarget, 0, len(contacts))
	for _, contact := range contacts {
		phone, ok := contact[phoneField].(string)
		if !ok || phone == "" {
			continue
		}
		targets = append(targets, BatchCallTarget{
			PhoneNumber: phone,
			Variables:   contact,
		})
	}
	return targets
}
