package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Tool represents a custom tool that AI agents can use during calls.
// Tools allow mid-call API integrations for real-time data fetching or actions.
type Tool struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"` // webhook, function
	URL         string          `json:"url,omitempty"`
	Method      string          `json:"method,omitempty"` // GET, POST, etc.
	Headers     map[string]string `json:"headers,omitempty"`
	Parameters  []ToolParameter `json:"parameters,omitempty"`
	ResponseMap *ResponseMapping `json:"response_map,omitempty"`
	SpeechConfig *ToolSpeechConfig `json:"speech,omitempty"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty"`
}

// ToolParameter defines an input parameter for a tool.
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean, array, object
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"` // Allowed values
	Example     interface{} `json:"example,omitempty"`
}

// ResponseMapping defines how to map tool response to AI context.
type ResponseMapping struct {
	// SuccessPath: JSONPath to extract success indicator
	SuccessPath string `json:"success_path,omitempty"`

	// DataPath: JSONPath to extract main data
	DataPath string `json:"data_path,omitempty"`

	// ErrorPath: JSONPath to extract error message
	ErrorPath string `json:"error_path,omitempty"`

	// FieldMappings: Map response fields to AI-friendly names
	FieldMappings map[string]string `json:"field_mappings,omitempty"`

	// SummaryTemplate: Template for creating a summary from response
	SummaryTemplate string `json:"summary_template,omitempty"`
}

// ToolSpeechConfig configures what the AI says during tool execution.
type ToolSpeechConfig struct {
	// BeforeExecution: What to say while fetching data
	BeforeExecution string `json:"before,omitempty"`

	// OnSuccess: What to say (template) when successful
	OnSuccess string `json:"on_success,omitempty"`

	// OnError: What to say when tool fails
	OnError string `json:"on_error,omitempty"`

	// WaitForResult: Whether AI should wait silently
	WaitForResult bool `json:"wait_for_result,omitempty"`
}

// CreateToolRequest contains parameters for creating a custom tool.
type CreateToolRequest struct {
	// Name: A clear name for the tool (AI uses this to decide when to call)
	Name string `json:"name"`

	// Description: Explains to AI when/why to use this tool
	Description string `json:"description"`

	// Type: "webhook" for HTTP calls, "function" for built-in functions
	Type string `json:"type"`

	// URL: The endpoint to call (for webhook type)
	URL string `json:"url,omitempty"`

	// Method: HTTP method (default: POST)
	Method string `json:"method,omitempty"`

	// Headers: Additional HTTP headers
	Headers map[string]string `json:"headers,omitempty"`

	// Parameters: Input parameters the AI should collect/provide
	Parameters []ToolParameter `json:"parameters,omitempty"`

	// ResponseMap: How to interpret the response
	ResponseMap *ResponseMapping `json:"response_map,omitempty"`

	// SpeechConfig: What to say during tool execution
	SpeechConfig *ToolSpeechConfig `json:"speech,omitempty"`

	// Timeout: Maximum time to wait for response (seconds)
	Timeout int `json:"timeout,omitempty"`

	// RetryConfig: Retry behavior on failure
	RetryCount int `json:"retry_count,omitempty"`
	RetryDelay int `json:"retry_delay,omitempty"` // seconds
}

// UpdateToolRequest contains parameters for updating a tool.
type UpdateToolRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	URL         *string           `json:"url,omitempty"`
	Method      *string           `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Parameters  []ToolParameter   `json:"parameters,omitempty"`
	ResponseMap *ResponseMapping  `json:"response_map,omitempty"`
	SpeechConfig *ToolSpeechConfig `json:"speech,omitempty"`
	IsActive    *bool             `json:"is_active,omitempty"`
	Timeout     *int              `json:"timeout,omitempty"`
}

// ListToolsResponse contains the response from listing tools.
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
	Total int    `json:"total,omitempty"`
}

// ToolExecutionLog represents a record of a tool being called.
type ToolExecutionLog struct {
	ID           string                 `json:"id"`
	ToolID       string                 `json:"tool_id"`
	ToolName     string                 `json:"tool_name"`
	CallID       string                 `json:"call_id"`
	Input        map[string]interface{} `json:"input"`
	Output       interface{}            `json:"output,omitempty"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	DurationMs   int                    `json:"duration_ms"`
	ExecutedAt   time.Time              `json:"executed_at"`
}

// CreateTool creates a new custom tool.
func (c *Client) CreateTool(ctx context.Context, req *CreateToolRequest) (*Tool, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if req.Type == "" {
		req.Type = "webhook"
	}
	if req.Type == "webhook" && req.URL == "" {
		return nil, fmt.Errorf("url is required for webhook tools")
	}

	var tool Tool
	if err := c.request(ctx, "POST", "/tools", req, &tool); err != nil {
		return nil, err
	}

	c.logger.Info("tool created",
		zap.String("id", tool.ID),
		zap.String("name", tool.Name),
	)

	return &tool, nil
}

// GetTool retrieves a specific tool by ID.
func (c *Client) GetTool(ctx context.Context, toolID string) (*Tool, error) {
	if toolID == "" {
		return nil, fmt.Errorf("tool_id is required")
	}

	var tool Tool
	if err := c.request(ctx, "GET", "/tools/"+toolID, nil, &tool); err != nil {
		return nil, err
	}

	return &tool, nil
}

// ListTools retrieves all custom tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var resp ListToolsResponse
	if err := c.request(ctx, "GET", "/tools", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Tools, nil
}

// UpdateTool updates an existing tool.
func (c *Client) UpdateTool(ctx context.Context, toolID string, req *UpdateToolRequest) (*Tool, error) {
	if toolID == "" {
		return nil, fmt.Errorf("tool_id is required")
	}

	var tool Tool
	if err := c.request(ctx, "PATCH", "/tools/"+toolID, req, &tool); err != nil {
		return nil, err
	}

	c.logger.Info("tool updated", zap.String("id", toolID))
	return &tool, nil
}

// DeleteTool deletes a custom tool.
func (c *Client) DeleteTool(ctx context.Context, toolID string) error {
	if toolID == "" {
		return fmt.Errorf("tool_id is required")
	}

	if err := c.request(ctx, "DELETE", "/tools/"+toolID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("tool deleted", zap.String("id", toolID))
	return nil
}

// EnableTool activates a tool so AI agents can use it.
func (c *Client) EnableTool(ctx context.Context, toolID string) error {
	active := true
	_, err := c.UpdateTool(ctx, toolID, &UpdateToolRequest{IsActive: &active})
	return err
}

// DisableTool deactivates a tool.
func (c *Client) DisableTool(ctx context.Context, toolID string) error {
	active := false
	_, err := c.UpdateTool(ctx, toolID, &UpdateToolRequest{IsActive: &active})
	return err
}

// GetToolExecutions retrieves execution logs for a tool.
func (c *Client) GetToolExecutions(ctx context.Context, toolID string, limit, offset int) ([]ToolExecutionLog, error) {
	if toolID == "" {
		return nil, fmt.Errorf("tool_id is required")
	}

	path := fmt.Sprintf("/tools/%s/executions?limit=%d&offset=%d", toolID, limit, offset)

	var resp struct {
		Executions []ToolExecutionLog `json:"executions"`
	}
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Executions, nil
}

// TestTool tests a tool with sample input.
func (c *Client) TestTool(ctx context.Context, toolID string, input map[string]interface{}) (*ToolExecutionLog, error) {
	if toolID == "" {
		return nil, fmt.Errorf("tool_id is required")
	}

	req := map[string]interface{}{
		"input": input,
	}

	var log ToolExecutionLog
	if err := c.request(ctx, "POST", "/tools/"+toolID+"/test", req, &log); err != nil {
		return nil, err
	}

	return &log, nil
}

// Helper functions for common tool patterns

// NewQuoteLookupTool creates a tool for looking up quotes during calls.
func NewQuoteLookupTool(webhookBaseURL string) *CreateToolRequest {
	return &CreateToolRequest{
		Name:        "lookup_quote",
		Description: "Look up an existing quote by quote ID or customer phone number. Use when customer asks about a previous quote.",
		Type:        "webhook",
		URL:         webhookBaseURL + "/api/v1/tools/quote-lookup",
		Method:      "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Parameters: []ToolParameter{
			{
				Name:        "quote_id",
				Type:        "string",
				Description: "The quote ID (if customer provides it)",
				Required:    false,
			},
			{
				Name:        "phone_number",
				Type:        "string",
				Description: "The customer's phone number to look up quotes",
				Required:    false,
			},
		},
		ResponseMap: &ResponseMapping{
			SuccessPath:     "$.success",
			DataPath:        "$.quote",
			ErrorPath:       "$.error",
			SummaryTemplate: "Quote {{quote_id}}: {{description}} - ${{amount}}",
		},
		SpeechConfig: &ToolSpeechConfig{
			BeforeExecution: "Let me look that up for you.",
			OnSuccess:       "I found your quote. {{summary}}",
			OnError:         "I couldn't find that quote. Can you verify the information?",
		},
		Timeout: 10,
	}
}

// NewScheduleCallbackTool creates a tool for scheduling callback appointments.
func NewScheduleCallbackTool(webhookBaseURL string) *CreateToolRequest {
	return &CreateToolRequest{
		Name:        "schedule_callback",
		Description: "Schedule a callback appointment for the customer. Use when they want to speak with a representative later.",
		Type:        "webhook",
		URL:         webhookBaseURL + "/api/v1/tools/schedule-callback",
		Method:      "POST",
		Parameters: []ToolParameter{
			{
				Name:        "preferred_date",
				Type:        "string",
				Description: "The customer's preferred date (e.g., 'tomorrow', 'Monday', 'next week')",
				Required:    true,
			},
			{
				Name:        "preferred_time",
				Type:        "string",
				Description: "The customer's preferred time (e.g., 'morning', '2pm', 'afternoon')",
				Required:    true,
			},
			{
				Name:        "reason",
				Type:        "string",
				Description: "The reason for the callback",
				Required:    false,
			},
		},
		SpeechConfig: &ToolSpeechConfig{
			BeforeExecution: "I'm scheduling that callback for you now.",
			OnSuccess:       "I've scheduled your callback for {{date}} at {{time}}. You'll receive a confirmation.",
			OnError:         "I had trouble scheduling that time. Can we try a different time?",
		},
		Timeout: 15,
	}
}

// NewPricingLookupTool creates a tool for real-time pricing lookup.
func NewPricingLookupTool(webhookBaseURL string) *CreateToolRequest {
	return &CreateToolRequest{
		Name:        "get_pricing",
		Description: "Get real-time pricing for a service or product. Use when customer asks about costs.",
		Type:        "webhook",
		URL:         webhookBaseURL + "/api/v1/tools/pricing",
		Method:      "POST",
		Parameters: []ToolParameter{
			{
				Name:        "service_type",
				Type:        "string",
				Description: "The type of service (e.g., 'basic', 'premium', 'enterprise')",
				Required:    true,
				Enum:        []string{"basic", "standard", "premium", "enterprise"},
			},
			{
				Name:        "quantity",
				Type:        "number",
				Description: "Quantity or volume for pricing",
				Required:    false,
				Default:     1,
			},
		},
		ResponseMap: &ResponseMapping{
			DataPath: "$.pricing",
			FieldMappings: map[string]string{
				"base_price":    "price",
				"discount_pct":  "discount",
				"final_price":   "total",
			},
		},
		SpeechConfig: &ToolSpeechConfig{
			BeforeExecution: "Let me check our current pricing.",
			OnSuccess:       "The {{service_type}} plan is ${{price}} per month.",
			OnError:         "I'm having trouble accessing pricing. Let me transfer you to someone who can help.",
		},
	}
}

// NewCustomerVerificationTool creates a tool for verifying customer identity.
func NewCustomerVerificationTool(webhookBaseURL string) *CreateToolRequest {
	return &CreateToolRequest{
		Name:        "verify_customer",
		Description: "Verify customer identity using their account details. Use before accessing sensitive information.",
		Type:        "webhook",
		URL:         webhookBaseURL + "/api/v1/tools/verify-customer",
		Method:      "POST",
		Parameters: []ToolParameter{
			{
				Name:        "phone_number",
				Type:        "string",
				Description: "Customer's phone number on file",
				Required:    true,
			},
			{
				Name:        "verification_code",
				Type:        "string",
				Description: "Last 4 digits of account or ZIP code",
				Required:    true,
			},
		},
		SpeechConfig: &ToolSpeechConfig{
			BeforeExecution: "Let me verify that for you.",
			OnSuccess:       "Thank you, I've verified your account. How can I help you today?",
			OnError:         "I couldn't verify those details. Can you double-check and try again?",
		},
		Timeout: 10,
	}
}

// BuildToolsList returns tool IDs for use in call parameters.
func BuildToolsList(tools ...*Tool) []string {
	ids := make([]string, len(tools))
	for i, t := range tools {
		ids[i] = t.ID
	}
	return ids
}
