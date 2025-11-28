package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Memory represents persistent context for a phone number or call.
// Memory allows AI agents to remember information across multiple calls.
type Memory struct {
	ID          string                 `json:"id"`
	PhoneNumber string                 `json:"phone_number,omitempty"`
	CallID      string                 `json:"call_id,omitempty"`
	Key         string                 `json:"key,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// MemoryEntry represents a single key-value memory entry.
type MemoryEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Type      string      `json:"type,omitempty"` // string, number, boolean, json
	CreatedAt time.Time   `json:"created_at,omitempty"`
	UpdatedAt time.Time   `json:"updated_at,omitempty"`
}

// CreateMemoryRequest contains parameters for storing memory.
type CreateMemoryRequest struct {
	// PhoneNumber: Store memory associated with this phone number
	PhoneNumber string `json:"phone_number,omitempty"`

	// CallID: Store memory associated with a specific call
	CallID string `json:"call_id,omitempty"`

	// Key: The memory key (for key-value storage)
	Key string `json:"key,omitempty"`

	// Value: The memory value (for key-value storage)
	Value interface{} `json:"value,omitempty"`

	// Data: Bulk key-value pairs to store
	Data map[string]interface{} `json:"data,omitempty"`

	// ExpiresIn: Optional TTL in seconds
	ExpiresIn *int `json:"expires_in,omitempty"`
}

// UpdateMemoryRequest contains parameters for updating memory.
type UpdateMemoryRequest struct {
	// Value: New value for the key
	Value interface{} `json:"value,omitempty"`

	// Data: Bulk update of key-value pairs (merges with existing)
	Data map[string]interface{} `json:"data,omitempty"`

	// ExpiresIn: Update TTL in seconds
	ExpiresIn *int `json:"expires_in,omitempty"`
}

// ListMemoryResponse contains the response from listing memory.
type ListMemoryResponse struct {
	Memories []Memory `json:"memories"`
	Total    int      `json:"total,omitempty"`
}

// GetMemoryByPhoneResponse contains memory entries for a phone number.
type GetMemoryByPhoneResponse struct {
	PhoneNumber string                 `json:"phone_number"`
	Data        map[string]interface{} `json:"data"`
	Entries     []MemoryEntry          `json:"entries,omitempty"`
}

// GetMemoryByPhone retrieves all memory associated with a phone number.
func (c *Client) GetMemoryByPhone(ctx context.Context, phoneNumber string) (*GetMemoryByPhoneResponse, error) {
	if phoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}

	var resp GetMemoryByPhoneResponse
	path := fmt.Sprintf("/memory?phone_number=%s", phoneNumber)
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMemoryByCall retrieves all memory associated with a specific call.
func (c *Client) GetMemoryByCall(ctx context.Context, callID string) (*GetMemoryByPhoneResponse, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var resp GetMemoryByPhoneResponse
	path := fmt.Sprintf("/memory?call_id=%s", callID)
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMemoryValue retrieves a specific memory value by key.
func (c *Client) GetMemoryValue(ctx context.Context, phoneNumber, key string) (interface{}, error) {
	if phoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	var resp struct {
		Value interface{} `json:"value"`
	}
	path := fmt.Sprintf("/memory/%s?phone_number=%s", key, phoneNumber)
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// StoreMemory stores one or more key-value pairs in memory.
func (c *Client) StoreMemory(ctx context.Context, req *CreateMemoryRequest) error {
	if req.PhoneNumber == "" && req.CallID == "" {
		return fmt.Errorf("either phone_number or call_id is required")
	}

	// Single key-value or bulk data
	if req.Key == "" && len(req.Data) == 0 {
		return fmt.Errorf("either key/value or data is required")
	}

	if err := c.request(ctx, "POST", "/memory", req, nil); err != nil {
		return err
	}

	c.logger.Info("memory stored",
		zap.String("phone_number", req.PhoneNumber),
		zap.String("call_id", req.CallID),
		zap.String("key", req.Key),
	)

	return nil
}

// UpdateMemory updates existing memory values.
func (c *Client) UpdateMemory(ctx context.Context, phoneNumber string, req *UpdateMemoryRequest) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}

	body := map[string]interface{}{
		"phone_number": phoneNumber,
	}
	if req.Value != nil {
		body["value"] = req.Value
	}
	if req.Data != nil {
		body["data"] = req.Data
	}
	if req.ExpiresIn != nil {
		body["expires_in"] = *req.ExpiresIn
	}

	if err := c.request(ctx, "PATCH", "/memory", body, nil); err != nil {
		return err
	}

	c.logger.Info("memory updated", zap.String("phone_number", phoneNumber))
	return nil
}

// DeleteMemory deletes all memory for a phone number.
func (c *Client) DeleteMemory(ctx context.Context, phoneNumber string) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}

	path := fmt.Sprintf("/memory?phone_number=%s", phoneNumber)
	if err := c.request(ctx, "DELETE", path, nil, nil); err != nil {
		return err
	}

	c.logger.Info("memory deleted", zap.String("phone_number", phoneNumber))
	return nil
}

// DeleteMemoryKey deletes a specific memory key for a phone number.
func (c *Client) DeleteMemoryKey(ctx context.Context, phoneNumber, key string) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	path := fmt.Sprintf("/memory/%s?phone_number=%s", key, phoneNumber)
	if err := c.request(ctx, "DELETE", path, nil, nil); err != nil {
		return err
	}

	c.logger.Info("memory key deleted",
		zap.String("phone_number", phoneNumber),
		zap.String("key", key),
	)
	return nil
}

// ListAllMemory retrieves all memory entries (admin function).
func (c *Client) ListAllMemory(ctx context.Context, limit, offset int) (*ListMemoryResponse, error) {
	path := fmt.Sprintf("/memory/all?limit=%d&offset=%d", limit, offset)

	var resp ListMemoryResponse
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Helper functions for common memory operations

// RememberCustomer stores customer-specific data that persists across calls.
func (c *Client) RememberCustomer(ctx context.Context, phoneNumber string, data map[string]interface{}) error {
	return c.StoreMemory(ctx, &CreateMemoryRequest{
		PhoneNumber: phoneNumber,
		Data:        data,
	})
}

// GetCustomerContext retrieves stored context for a customer.
func (c *Client) GetCustomerContext(ctx context.Context, phoneNumber string) (map[string]interface{}, error) {
	resp, err := c.GetMemoryByPhone(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// StoreQuoteContext stores quote-related context for a customer.
// This helps the AI agent remember previous quote requests and preferences.
func (c *Client) StoreQuoteContext(ctx context.Context, phoneNumber string, quoteData map[string]interface{}) error {
	// Wrap quote data under a "quotes" key for organization
	data := map[string]interface{}{
		"last_quote_request": quoteData,
		"quote_history":      true,
	}
	return c.RememberCustomer(ctx, phoneNumber, data)
}

// GetCallHistory returns memory data structured as call history.
func (c *Client) GetCallHistory(ctx context.Context, phoneNumber string) ([]map[string]interface{}, error) {
	resp, err := c.GetMemoryByPhone(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	// Extract call history if it exists
	if history, ok := resp.Data["call_history"].([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(history))
		for _, item := range history {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result, nil
	}

	return nil, nil
}

// AppendCallToHistory adds a call record to the customer's history.
func (c *Client) AppendCallToHistory(ctx context.Context, phoneNumber string, callRecord map[string]interface{}) error {
	// Get existing history
	history, err := c.GetCallHistory(ctx, phoneNumber)
	if err != nil {
		// If no history exists, start fresh
		history = []map[string]interface{}{}
	}

	// Append new record
	history = append(history, callRecord)

	// Keep only last 10 calls to prevent unbounded growth
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	// Store updated history
	return c.StoreMemory(ctx, &CreateMemoryRequest{
		PhoneNumber: phoneNumber,
		Data: map[string]interface{}{
			"call_history": history,
		},
	})
}

// ClearCustomerMemory removes all stored context for a customer.
// Use carefully - this is irreversible.
func (c *Client) ClearCustomerMemory(ctx context.Context, phoneNumber string) error {
	return c.DeleteMemory(ctx, phoneNumber)
}
