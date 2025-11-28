package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// SMS represents an SMS message.
type SMS struct {
	ID            string    `json:"id"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Body          string    `json:"body"`
	Direction     string    `json:"direction"` // inbound, outbound
	Status        string    `json:"status"`    // queued, sent, delivered, failed
	ErrorCode     string    `json:"error_code,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	MediaURLs     []string  `json:"media_urls,omitempty"` // MMS attachments
	NumSegments   int       `json:"num_segments,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
}

// SMSConversation represents an ongoing SMS conversation with AI.
type SMSConversation struct {
	ID            string    `json:"id"`
	PhoneNumber   string    `json:"phone_number"`
	Status        string    `json:"status"` // active, ended
	Messages      []SMS     `json:"messages,omitempty"`
	Task          string    `json:"task,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

// SendSMSRequest contains parameters for sending an SMS.
type SendSMSRequest struct {
	// To: The recipient phone number (E.164 format)
	To string `json:"to"`

	// From: The sender phone number (must be a Bland number)
	From string `json:"from,omitempty"`

	// Body: The message content
	Body string `json:"body"`

	// MediaURLs: URLs for MMS attachments (images, etc.)
	MediaURLs []string `json:"media_urls,omitempty"`

	// AI-powered SMS conversation
	Task string `json:"task,omitempty"` // AI task for responding to replies

	// Webhook for delivery status updates
	WebhookURL string `json:"webhook_url,omitempty"`

	// Scheduling
	ScheduledTime *time.Time `json:"scheduled_time,omitempty"`

	// Metadata for tracking
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendSMSResponse contains the response from sending an SMS.
type SendSMSResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	To        string `json:"to"`
	From      string `json:"from"`
	Body      string `json:"body,omitempty"`
	Message   string `json:"message,omitempty"` // Success/error message
}

// StartSMSConversationRequest contains parameters for starting an AI SMS conversation.
type StartSMSConversationRequest struct {
	// To: The recipient phone number
	To string `json:"to"`

	// From: The sender phone number (Bland number)
	From string `json:"from,omitempty"`

	// Task: The AI task/prompt for managing the conversation
	Task string `json:"task"`

	// FirstMessage: The initial message to send
	FirstMessage string `json:"first_message,omitempty"`

	// Model: The AI model to use
	Model string `json:"model,omitempty"`

	// Temperature: AI response creativity
	Temperature float64 `json:"temperature,omitempty"`

	// PathwayID: Use a conversational pathway
	PathwayID string `json:"pathway_id,omitempty"`

	// KnowledgeBaseIDs: Knowledge bases for AI reference
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`

	// WebhookURL: Webhook for conversation events
	WebhookURL string `json:"webhook_url,omitempty"`

	// MaxMessages: Maximum messages in conversation
	MaxMessages int `json:"max_messages,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StartSMSConversationResponse contains the response from starting a conversation.
type StartSMSConversationResponse struct {
	ConversationID string `json:"conversation_id"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
}

// ListSMSResponse contains the response from listing SMS messages.
type ListSMSResponse struct {
	Messages []SMS `json:"messages"`
	Total    int   `json:"total,omitempty"`
}

// SendSMS sends a single SMS message.
func (c *Client) SendSMS(ctx context.Context, req *SendSMSRequest) (*SendSMSResponse, error) {
	if req.To == "" {
		return nil, fmt.Errorf("to phone number is required")
	}
	if req.Body == "" {
		return nil, fmt.Errorf("message body is required")
	}

	var resp SendSMSResponse
	if err := c.request(ctx, "POST", "/sms", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("SMS sent",
		zap.String("message_id", resp.MessageID),
		zap.String("to", req.To),
		zap.String("status", resp.Status),
	)

	return &resp, nil
}

// GetSMS retrieves details for a specific SMS message.
func (c *Client) GetSMS(ctx context.Context, messageID string) (*SMS, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message_id is required")
	}

	var sms SMS
	if err := c.request(ctx, "GET", "/sms/"+messageID, nil, &sms); err != nil {
		return nil, err
	}

	return &sms, nil
}

// ListSMS retrieves SMS messages with optional filters.
func (c *Client) ListSMS(ctx context.Context, phoneNumber string, limit, offset int) (*ListSMSResponse, error) {
	path := fmt.Sprintf("/sms?limit=%d&offset=%d", limit, offset)
	if phoneNumber != "" {
		path += "&phone_number=" + phoneNumber
	}

	var resp ListSMSResponse
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// StartSMSConversation starts an AI-powered SMS conversation.
func (c *Client) StartSMSConversation(ctx context.Context, req *StartSMSConversationRequest) (*StartSMSConversationResponse, error) {
	if req.To == "" {
		return nil, fmt.Errorf("to phone number is required")
	}
	if req.Task == "" {
		return nil, fmt.Errorf("task is required for AI conversation")
	}

	var resp StartSMSConversationResponse
	if err := c.request(ctx, "POST", "/sms/conversation", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("SMS conversation started",
		zap.String("conversation_id", resp.ConversationID),
		zap.String("to", req.To),
	)

	return &resp, nil
}

// GetSMSConversation retrieves an SMS conversation.
func (c *Client) GetSMSConversation(ctx context.Context, conversationID string) (*SMSConversation, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}

	var conv SMSConversation
	if err := c.request(ctx, "GET", "/sms/conversation/"+conversationID, nil, &conv); err != nil {
		return nil, err
	}

	return &conv, nil
}

// EndSMSConversation ends an active SMS conversation.
func (c *Client) EndSMSConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}

	if err := c.request(ctx, "POST", "/sms/conversation/"+conversationID+"/end", nil, nil); err != nil {
		return err
	}

	c.logger.Info("SMS conversation ended", zap.String("conversation_id", conversationID))
	return nil
}

// ListSMSConversations retrieves all SMS conversations.
func (c *Client) ListSMSConversations(ctx context.Context, status string, limit, offset int) ([]SMSConversation, error) {
	path := fmt.Sprintf("/sms/conversations?limit=%d&offset=%d", limit, offset)
	if status != "" {
		path += "&status=" + status
	}

	var resp struct {
		Conversations []SMSConversation `json:"conversations"`
	}
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Conversations, nil
}

// SendBulkSMS sends multiple SMS messages at once.
func (c *Client) SendBulkSMS(ctx context.Context, from, body string, toNumbers []string) ([]SendSMSResponse, error) {
	if len(toNumbers) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}
	if body == "" {
		return nil, fmt.Errorf("message body is required")
	}

	req := map[string]interface{}{
		"from":   from,
		"body":   body,
		"to":     toNumbers,
	}

	var resp struct {
		Messages []SendSMSResponse `json:"messages"`
	}
	if err := c.request(ctx, "POST", "/sms/bulk", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("bulk SMS sent",
		zap.Int("count", len(toNumbers)),
	)

	return resp.Messages, nil
}

// Helper functions for SMS operations

// SendQuoteFollowUp sends a follow-up SMS after a quote call.
func (c *Client) SendQuoteFollowUp(ctx context.Context, phoneNumber, customerName, quoteID string) (*SendSMSResponse, error) {
	body := fmt.Sprintf("Hi %s! Thanks for your recent quote request. Your quote ID is %s. Reply with any questions!", customerName, quoteID)
	return c.SendSMS(ctx, &SendSMSRequest{
		To:   phoneNumber,
		Body: body,
		Metadata: map[string]interface{}{
			"type":     "quote_followup",
			"quote_id": quoteID,
		},
	})
}

// SendQuoteReadySMS notifies customer their quote is ready.
func (c *Client) SendQuoteReadySMS(ctx context.Context, phoneNumber, quoteID string, amount float64) (*SendSMSResponse, error) {
	body := fmt.Sprintf("Great news! Your quote is ready. Quote ID: %s, Estimated: $%.2f. Reply YES to accept or call us to discuss.", quoteID, amount)
	return c.SendSMS(ctx, &SendSMSRequest{
		To:   phoneNumber,
		Body: body,
		Metadata: map[string]interface{}{
			"type":      "quote_ready",
			"quote_id":  quoteID,
			"amount":    amount,
		},
	})
}

// StartQuoteSMSConversation starts an AI conversation for quote questions.
func (c *Client) StartQuoteSMSConversation(ctx context.Context, phoneNumber, task string, knowledgeBases []string) (*StartSMSConversationResponse, error) {
	return c.StartSMSConversation(ctx, &StartSMSConversationRequest{
		To:               phoneNumber,
		Task:             task,
		KnowledgeBaseIDs: knowledgeBases,
		FirstMessage:     "Hi! I'm here to help with any questions about your quote. What would you like to know?",
		MaxMessages:      20,
	})
}

// GetConversationHistory retrieves the full message history for a phone number.
func (c *Client) GetConversationHistory(ctx context.Context, phoneNumber string) ([]SMS, error) {
	resp, err := c.ListSMS(ctx, phoneNumber, 100, 0)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}
