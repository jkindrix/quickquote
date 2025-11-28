package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// PhoneNumber represents a phone number in the Bland system.
type PhoneNumber struct {
	ID                string            `json:"id"`
	PhoneNumber       string            `json:"phone_number"`
	CountryCode       string            `json:"country_code,omitempty"`
	AreaCode          string            `json:"area_code,omitempty"`
	Type              string            `json:"type,omitempty"` // local, toll-free
	Status            string            `json:"status,omitempty"`
	Capabilities      []string          `json:"capabilities,omitempty"` // voice, sms
	InboundAgentID    string            `json:"inbound_agent_id,omitempty"`
	InboundPathwayID  string            `json:"inbound_pathway_id,omitempty"`
	InboundPrompt     string            `json:"inbound_prompt,omitempty"`
	InboundVoice      string            `json:"inbound_voice,omitempty"`
	InboundWebhookURL string            `json:"inbound_webhook_url,omitempty"`
	InboundConfig     *InboundConfig    `json:"inbound_config,omitempty"`
	MonthlyCost       float64           `json:"monthly_cost,omitempty"`
	Provider          string            `json:"provider,omitempty"`
	Region            string            `json:"region,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	CreatedAt         time.Time         `json:"created_at,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at,omitempty"`
}

// InboundConfig contains configuration for inbound call handling.
type InboundConfig struct {
	// Agent configuration
	Task             string                 `json:"task,omitempty"`
	PathwayID        string                 `json:"pathway_id,omitempty"`
	Voice            string                 `json:"voice,omitempty"`
	VoiceSettings    *VoiceSettings         `json:"voice_settings,omitempty"`
	Language         string                 `json:"language,omitempty"`
	Model            string                 `json:"model,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	FirstSentence    string                 `json:"first_sentence,omitempty"`
	WaitForGreeting  bool                   `json:"wait_for_greeting,omitempty"`
	InterruptionThreshold int              `json:"interruption_threshold,omitempty"`

	// Knowledge and context
	KnowledgeBases   []string               `json:"knowledge_base_ids,omitempty"`
	Tools            []string               `json:"tool_ids,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`

	// Recording and analysis
	Record            bool                   `json:"record,omitempty"`
	AnalysisSchema    map[string]interface{} `json:"analysis_schema,omitempty"`
	SummaryPrompt     string                 `json:"summary_prompt,omitempty"`
	Keywords          []string               `json:"keywords,omitempty"`
	BackgroundTrack   string                 `json:"background_track,omitempty"`
	NoiseCancellation bool                   `json:"noise_cancellation,omitempty"`

	// Webhooks
	WebhookURL       string                 `json:"webhook,omitempty"`
	WebhookEvents    []string               `json:"webhook_events,omitempty"`

	// Call limits
	MaxDuration      int                    `json:"max_duration,omitempty"`
}

// ListPhoneNumbersRequest contains parameters for listing phone numbers.
type ListPhoneNumbersRequest struct {
	Status      string `json:"status,omitempty"`
	Type        string `json:"type,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// ListPhoneNumbersResponse contains the response from listing phone numbers.
type ListPhoneNumbersResponse struct {
	PhoneNumbers []PhoneNumber `json:"phone_numbers"`
	Total        int           `json:"total,omitempty"`
}

// AvailablePhoneNumber represents a phone number available for purchase.
type AvailablePhoneNumber struct {
	PhoneNumber  string   `json:"phone_number"`
	CountryCode  string   `json:"country_code"`
	AreaCode     string   `json:"area_code,omitempty"`
	Type         string   `json:"type"` // local, toll-free
	Capabilities []string `json:"capabilities"`
	MonthlyCost  float64  `json:"monthly_cost"`
	Region       string   `json:"region,omitempty"`
	City         string   `json:"city,omitempty"`
	State        string   `json:"state,omitempty"`
}

// SearchAvailableNumbersRequest contains parameters for searching available numbers.
type SearchAvailableNumbersRequest struct {
	CountryCode  string `json:"country_code"`
	AreaCode     string `json:"area_code,omitempty"`
	Type         string `json:"type,omitempty"` // local, toll-free
	Contains     string `json:"contains,omitempty"` // Pattern matching
	Capabilities []string `json:"capabilities,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

// SearchAvailableNumbersResponse contains available numbers.
type SearchAvailableNumbersResponse struct {
	Numbers []AvailablePhoneNumber `json:"numbers"`
	Total   int                    `json:"total,omitempty"`
}

// PurchaseNumberRequest contains parameters for purchasing a phone number.
type PurchaseNumberRequest struct {
	PhoneNumber   string         `json:"phone_number"`
	InboundConfig *InboundConfig `json:"inbound_config,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// PurchaseNumberResponse contains the result of purchasing a number.
type PurchaseNumberResponse struct {
	PhoneNumber PhoneNumber `json:"phone_number"`
	Success     bool        `json:"success"`
	Message     string      `json:"message,omitempty"`
}

// UpdatePhoneNumberRequest contains parameters for updating a phone number.
type UpdatePhoneNumberRequest struct {
	InboundAgentID    *string           `json:"inbound_agent_id,omitempty"`
	InboundPathwayID  *string           `json:"inbound_pathway_id,omitempty"`
	InboundPrompt     *string           `json:"inbound_prompt,omitempty"`
	InboundVoice      *string           `json:"inbound_voice,omitempty"`
	InboundWebhookURL *string           `json:"inbound_webhook_url,omitempty"`
	InboundConfig     *InboundConfig    `json:"inbound_config,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// BlockedNumber represents a blocked phone number.
type BlockedNumber struct {
	ID          string    `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	Reason      string    `json:"reason,omitempty"`
	Direction   string    `json:"direction,omitempty"` // inbound, outbound, both
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// ListBlockedNumbersResponse contains blocked numbers.
type ListBlockedNumbersResponse struct {
	Numbers []BlockedNumber `json:"numbers"`
	Total   int             `json:"total,omitempty"`
}

// BlockNumberRequest contains parameters for blocking a number.
type BlockNumberRequest struct {
	PhoneNumber string `json:"phone_number"`
	Reason      string `json:"reason,omitempty"`
	Direction   string `json:"direction,omitempty"` // inbound, outbound, both
}

// ListPhoneNumbers retrieves all phone numbers in the account.
func (c *Client) ListPhoneNumbers(ctx context.Context, req *ListPhoneNumbersRequest) ([]PhoneNumber, error) {
	path := "/numbers"
	if req != nil {
		params := ""
		if req.Status != "" {
			params += fmt.Sprintf("status=%s&", req.Status)
		}
		if req.Type != "" {
			params += fmt.Sprintf("type=%s&", req.Type)
		}
		if req.CountryCode != "" {
			params += fmt.Sprintf("country_code=%s&", req.CountryCode)
		}
		if req.Limit > 0 {
			params += fmt.Sprintf("limit=%d&", req.Limit)
		}
		if req.Offset > 0 {
			params += fmt.Sprintf("offset=%d&", req.Offset)
		}
		if params != "" {
			path += "?" + params[:len(params)-1]
		}
	}

	var resp ListPhoneNumbersResponse
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.PhoneNumbers, nil
}

// GetPhoneNumber retrieves details for a specific phone number.
func (c *Client) GetPhoneNumber(ctx context.Context, phoneNumberID string) (*PhoneNumber, error) {
	if phoneNumberID == "" {
		return nil, fmt.Errorf("phone_number_id is required")
	}

	var number PhoneNumber
	if err := c.request(ctx, "GET", "/numbers/"+phoneNumberID, nil, &number); err != nil {
		return nil, err
	}

	return &number, nil
}

// SearchAvailableNumbers searches for phone numbers available for purchase.
func (c *Client) SearchAvailableNumbers(ctx context.Context, req *SearchAvailableNumbersRequest) ([]AvailablePhoneNumber, error) {
	if req == nil || req.CountryCode == "" {
		return nil, fmt.Errorf("country_code is required")
	}

	path := fmt.Sprintf("/numbers/available?country_code=%s", req.CountryCode)
	if req.AreaCode != "" {
		path += fmt.Sprintf("&area_code=%s", req.AreaCode)
	}
	if req.Type != "" {
		path += fmt.Sprintf("&type=%s", req.Type)
	}
	if req.Contains != "" {
		path += fmt.Sprintf("&contains=%s", req.Contains)
	}
	if req.Limit > 0 {
		path += fmt.Sprintf("&limit=%d", req.Limit)
	}

	var resp SearchAvailableNumbersResponse
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Numbers, nil
}

// PurchaseNumber purchases a phone number.
func (c *Client) PurchaseNumber(ctx context.Context, req *PurchaseNumberRequest) (*PhoneNumber, error) {
	if req == nil || req.PhoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}

	var resp PurchaseNumberResponse
	if err := c.request(ctx, "POST", "/numbers/purchase", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("phone number purchased",
		zap.String("phone_number", resp.PhoneNumber.PhoneNumber),
		zap.String("id", resp.PhoneNumber.ID),
	)

	return &resp.PhoneNumber, nil
}

// UpdatePhoneNumber updates a phone number's configuration.
func (c *Client) UpdatePhoneNumber(ctx context.Context, phoneNumberID string, req *UpdatePhoneNumberRequest) (*PhoneNumber, error) {
	if phoneNumberID == "" {
		return nil, fmt.Errorf("phone_number_id is required")
	}

	var number PhoneNumber
	if err := c.request(ctx, "PATCH", "/numbers/"+phoneNumberID, req, &number); err != nil {
		return nil, err
	}

	c.logger.Info("phone number updated", zap.String("id", phoneNumberID))
	return &number, nil
}

// ReleasePhoneNumber releases (cancels) a phone number.
func (c *Client) ReleasePhoneNumber(ctx context.Context, phoneNumberID string) error {
	if phoneNumberID == "" {
		return fmt.Errorf("phone_number_id is required")
	}

	if err := c.request(ctx, "DELETE", "/numbers/"+phoneNumberID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("phone number released", zap.String("id", phoneNumberID))
	return nil
}

// ConfigureInboundAgent configures the AI agent for inbound calls on a number.
// This uses the POST /v1/inbound/{phone_number} endpoint.
func (c *Client) ConfigureInboundAgent(ctx context.Context, phoneNumber string, config *InboundConfig) (*PhoneNumber, error) {
	if phoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	var number PhoneNumber
	if err := c.request(ctx, "POST", "/inbound/"+phoneNumber, config, &number); err != nil {
		return nil, err
	}

	c.logger.Info("inbound agent configured",
		zap.String("phone_number", phoneNumber),
		zap.String("voice", config.Voice),
	)

	return &number, nil
}

// SetInboundPathway sets the conversational pathway for inbound calls.
func (c *Client) SetInboundPathway(ctx context.Context, phoneNumberID, pathwayID string) (*PhoneNumber, error) {
	return c.UpdatePhoneNumber(ctx, phoneNumberID, &UpdatePhoneNumberRequest{
		InboundPathwayID: &pathwayID,
	})
}

// SetInboundPrompt sets the task/prompt for inbound calls.
func (c *Client) SetInboundPrompt(ctx context.Context, phoneNumberID, prompt string) (*PhoneNumber, error) {
	return c.UpdatePhoneNumber(ctx, phoneNumberID, &UpdatePhoneNumberRequest{
		InboundPrompt: &prompt,
	})
}

// SetInboundVoice sets the voice for inbound calls.
func (c *Client) SetInboundVoice(ctx context.Context, phoneNumberID, voice string) (*PhoneNumber, error) {
	return c.UpdatePhoneNumber(ctx, phoneNumberID, &UpdatePhoneNumberRequest{
		InboundVoice: &voice,
	})
}

// SetInboundWebhook sets the webhook URL for inbound call events.
func (c *Client) SetInboundWebhook(ctx context.Context, phoneNumberID, webhookURL string) (*PhoneNumber, error) {
	return c.UpdatePhoneNumber(ctx, phoneNumberID, &UpdatePhoneNumberRequest{
		InboundWebhookURL: &webhookURL,
	})
}

// ListBlockedNumbers retrieves all blocked phone numbers.
func (c *Client) ListBlockedNumbers(ctx context.Context) ([]BlockedNumber, error) {
	var resp ListBlockedNumbersResponse
	if err := c.request(ctx, "GET", "/numbers/blocked", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Numbers, nil
}

// BlockNumber blocks a phone number.
func (c *Client) BlockNumber(ctx context.Context, req *BlockNumberRequest) (*BlockedNumber, error) {
	if req == nil || req.PhoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}

	var blocked BlockedNumber
	if err := c.request(ctx, "POST", "/numbers/blocked", req, &blocked); err != nil {
		return nil, err
	}

	c.logger.Info("phone number blocked",
		zap.String("phone_number", req.PhoneNumber),
		zap.String("reason", req.Reason),
	)

	return &blocked, nil
}

// UnblockNumber unblocks a phone number.
func (c *Client) UnblockNumber(ctx context.Context, blockedID string) error {
	if blockedID == "" {
		return fmt.Errorf("blocked_id is required")
	}

	if err := c.request(ctx, "DELETE", "/numbers/blocked/"+blockedID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("phone number unblocked", zap.String("id", blockedID))
	return nil
}

// Helper functions for common inbound configurations

// NewQuoteAgentInboundConfig creates an inbound config optimized for project quote collection.
// The businessName and greeting parameters allow customization.
func NewQuoteAgentInboundConfig(webhookURL, businessName, greeting string) *InboundConfig {
	if businessName == "" {
		businessName = "our company"
	}
	if greeting == "" {
		greeting = "Hello! Thanks for calling. I'm here to help you get a quote for your software project. What kind of project are you looking to build?"
	}
	return &InboundConfig{
		Task: `You are a software project consultant. Your job is to:
1. Greet the caller warmly
2. Ask what type of software project they need (web app, mobile app, API, etc.)
3. Collect key requirements (features, timeline, budget)
4. Thank them and let them know they'll receive their quote shortly

Be professional, friendly, and efficient. Ask one question at a time.`,
		Voice:                 "maya",
		Language:              "en-US",
		Model:                 "enhanced",
		Temperature:           0.7,
		FirstSentence:         greeting,
		WaitForGreeting:       true,
		InterruptionThreshold: 50,
		Record:                true,
		WebhookURL:            webhookURL,
		WebhookEvents:         []string{"call.completed", "call.analyzed"},
		MaxDuration:           600, // 10 minutes
		AnalysisSchema: map[string]interface{}{
			"project_type":  "The type of project requested (web_app, mobile_app, api, etc.)",
			"customer_name": "The customer's full name",
			"contact_info":  "Any contact information provided",
			"key_details":   "Important details for generating the quote",
			"follow_up":     "Whether follow-up is needed and why",
		},
	}
}

// NewSupportAgentInboundConfig creates an inbound config for customer support.
func NewSupportAgentInboundConfig(webhookURL string, knowledgeBaseIDs []string) *InboundConfig {
	return &InboundConfig{
		Task: `You are a customer support agent. Your job is to:
1. Greet the caller and ask how you can help
2. Listen carefully to their issue
3. Use the knowledge base to provide accurate information
4. If you can't resolve the issue, offer to transfer or schedule a callback
5. Always be empathetic and professional`,
		Voice:                 "matt",
		Language:              "en-US",
		Model:                 "enhanced",
		Temperature:           0.5,
		FirstSentence:         "Hello, thank you for calling support. How can I help you today?",
		WaitForGreeting:       true,
		InterruptionThreshold: 40,
		KnowledgeBases:        knowledgeBaseIDs,
		Record:                true,
		WebhookURL:            webhookURL,
		WebhookEvents:         []string{"call.completed"},
		MaxDuration:           900, // 15 minutes
	}
}

// NewAppointmentAgentInboundConfig creates an inbound config for appointment scheduling.
func NewAppointmentAgentInboundConfig(webhookURL string, toolIDs []string) *InboundConfig {
	return &InboundConfig{
		Task: `You are an appointment scheduling assistant. Your job is to:
1. Greet the caller
2. Ask what type of appointment they need
3. Check availability using the scheduling tool
4. Book the appointment and confirm details
5. Send confirmation information`,
		Voice:                 "evelyn",
		Language:              "en-US",
		Model:                 "enhanced",
		Temperature:           0.3,
		FirstSentence:         "Hello! I can help you schedule an appointment. What type of appointment would you like to book?",
		WaitForGreeting:       true,
		InterruptionThreshold: 50,
		Tools:                 toolIDs,
		Record:                true,
		WebhookURL:            webhookURL,
		WebhookEvents:         []string{"call.completed", "tool.executed"},
		MaxDuration:           600,
	}
}
