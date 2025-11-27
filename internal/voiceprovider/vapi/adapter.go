// Package vapi implements the VoiceProvider interface for Vapi.ai.
// This is a reference implementation based on Vapi's webhook structure.
// See: https://docs.vapi.ai/server-url
package vapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/validation"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

// Config holds Vapi provider configuration.
type Config struct {
	APIKey        string
	WebhookSecret string
	APIURL        string
}

// Provider implements the voiceprovider.Provider interface for Vapi.
type Provider struct {
	config *Config
	logger *zap.Logger
}

// New creates a new Vapi provider.
func New(cfg *Config, logger *zap.Logger) *Provider {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.vapi.ai"
	}
	return &Provider{
		config: cfg,
		logger: logger,
	}
}

// GetName returns the provider type identifier.
func (p *Provider) GetName() voiceprovider.ProviderType {
	return voiceprovider.ProviderVapi
}

// GetWebhookPath returns the path for Vapi webhooks.
func (p *Provider) GetWebhookPath() string {
	return "/webhook/vapi"
}

// ValidateWebhook verifies the webhook authenticity.
// Vapi supports multiple authentication methods - we implement HMAC-SHA256.
func (p *Provider) ValidateWebhook(r *http.Request) bool {
	// If no webhook secret is configured, skip validation
	// NOTE: In production, webhook secrets should always be configured
	if p.config.WebhookSecret == "" {
		p.logger.Warn("webhook secret not configured, skipping signature validation")
		return true
	}

	// Vapi can use multiple auth methods:
	// 1. X-Vapi-Signature header (HMAC-SHA256)
	// 2. Authorization: Bearer <secret>
	// 3. Custom X-Vapi-Secret header

	// Check HMAC signature first (preferred method)
	signature := r.Header.Get("X-Vapi-Signature")
	if signature != "" {
		// Read body for signature verification
		body, err := io.ReadAll(r.Body)
		if err != nil {
			p.logger.Error("failed to read webhook body for validation", zap.Error(err))
			return false
		}

		// CRITICAL: Restore the body so ParseWebhook can read it
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Compute expected HMAC-SHA256 signature
		mac := hmac.New(sha256.New, []byte(p.config.WebhookSecret))
		mac.Write(body)
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		if hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			p.logger.Debug("webhook signature validated successfully",
				zap.String("provider", "vapi"),
			)
			return true
		}

		p.logger.Warn("webhook signature mismatch",
			zap.String("provider", "vapi"),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	// Fallback: Check for authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "Bearer "+p.config.WebhookSecret {
		p.logger.Debug("webhook validated via Authorization header",
			zap.String("provider", "vapi"),
		)
		return true
	}

	// Fallback: Check for custom secret header
	secretHeader := r.Header.Get("X-Vapi-Secret")
	if secretHeader == p.config.WebhookSecret {
		p.logger.Debug("webhook validated via X-Vapi-Secret header",
			zap.String("provider", "vapi"),
		)
		return true
	}

	p.logger.Warn("vapi webhook failed authentication",
		zap.String("remote_addr", r.RemoteAddr),
	)
	return false
}

// ParseWebhook parses a Vapi webhook into a normalized CallEvent.
func (p *Provider) ParseWebhook(r *http.Request) (*voiceprovider.CallEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload VapiWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Vapi sends different message types - handle accordingly
	// For call completion, we're interested in "end-of-call-report"
	event, err := p.toCallEvent(&payload)
	if err != nil {
		return nil, err
	}

	// Store raw payload for debugging
	var rawMetadata map[string]interface{}
	if err := json.Unmarshal(body, &rawMetadata); err == nil {
		event.RawMetadata = rawMetadata
	}

	p.logger.Debug("parsed vapi webhook",
		zap.String("type", payload.Message.Type),
		zap.String("call_id", event.ProviderCallID),
	)

	return event, nil
}

// validateEndOfCallReport validates and sanitizes the end-of-call report payload.
func (p *Provider) validateEndOfCallReport(payload *VapiWebhookPayload) error {
	v := validation.NewCallEventValidator()

	// Validate call ID (required)
	v.ValidateCallID(payload.Message.Call.ID)

	// Validate phone numbers
	v.ValidatePhoneNumbers(payload.Message.Call.Customer.Number, payload.Message.Call.PhoneNumber.Number)

	// Validate transcript content
	v.ValidateTranscript(payload.Message.Transcript)

	// Validate recording URL
	v.ValidateRecordingURL(payload.Message.RecordingURL)

	// Check for validation errors
	if !v.IsValid() {
		errs := v.Errors()
		p.logger.Warn("webhook payload validation failed",
			zap.String("provider", "vapi"),
			zap.String("call_id", payload.Message.Call.ID),
			zap.Int("error_count", len(errs)),
			zap.String("errors", errs.Error()),
		)
		return errs
	}

	// Sanitize strings that passed validation
	payload.Message.Transcript = validation.SanitizeString(payload.Message.Transcript)
	payload.Message.Summary = validation.SanitizeString(payload.Message.Summary)

	return nil
}

// toCallEvent converts a Vapi-specific payload to a normalized CallEvent.
func (p *Provider) toCallEvent(payload *VapiWebhookPayload) (*voiceprovider.CallEvent, error) {
	// Vapi sends various message types; handle based on type
	switch payload.Message.Type {
	case "end-of-call-report":
		return p.parseEndOfCallReport(payload)
	case "status-update":
		return p.parseStatusUpdate(payload)
	case "transcript":
		return p.parseTranscript(payload)
	default:
		// For other message types, return a minimal event
		return &voiceprovider.CallEvent{
			Provider:       voiceprovider.ProviderVapi,
			ProviderCallID: payload.Message.Call.ID,
			Status:         voiceprovider.CallStatusInProgress,
		}, nil
	}
}

// parseEndOfCallReport handles the end-of-call-report message type.
func (p *Provider) parseEndOfCallReport(payload *VapiWebhookPayload) (*voiceprovider.CallEvent, error) {
	call := payload.Message.Call
	if call.ID == "" {
		return nil, fmt.Errorf("missing call ID in end-of-call-report")
	}

	// Validate and sanitize payload
	if err := p.validateEndOfCallReport(payload); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	event := &voiceprovider.CallEvent{
		Provider:       voiceprovider.ProviderVapi,
		ProviderCallID: call.ID,
		ToNumber:       call.Customer.Number,
		FromNumber:     call.PhoneNumber.Number,
		Status:         p.normalizeStatus(call.Status),
		Transcript:     payload.Message.Transcript,
		Summary:        payload.Message.Summary,
		RecordingURL:   payload.Message.RecordingURL,
	}

	// Parse timestamps
	if call.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, call.StartedAt); err == nil {
			event.StartedAt = &t
		}
	}
	if call.EndedAt != "" {
		if t, err := time.Parse(time.RFC3339, call.EndedAt); err == nil {
			event.EndedAt = &t
		}
	}

	// Calculate duration if timestamps available
	if event.StartedAt != nil && event.EndedAt != nil {
		event.DurationSecs = int(event.EndedAt.Sub(*event.StartedAt).Seconds())
	}

	// Extract data from analysis if available
	if payload.Message.Analysis != nil {
		event.ExtractedData = p.extractData(payload.Message.Analysis)
	}

	// Parse structured transcript if available
	if len(payload.Message.Messages) > 0 {
		event.TranscriptEntries = make([]voiceprovider.TranscriptEntry, len(payload.Message.Messages))
		for i, msg := range payload.Message.Messages {
			event.TranscriptEntries[i] = voiceprovider.TranscriptEntry{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
	}

	return event, nil
}

// parseStatusUpdate handles the status-update message type.
func (p *Provider) parseStatusUpdate(payload *VapiWebhookPayload) (*voiceprovider.CallEvent, error) {
	call := payload.Message.Call
	return &voiceprovider.CallEvent{
		Provider:       voiceprovider.ProviderVapi,
		ProviderCallID: call.ID,
		ToNumber:       call.Customer.Number,
		FromNumber:     call.PhoneNumber.Number,
		Status:         p.normalizeStatus(payload.Message.Status),
	}, nil
}

// parseTranscript handles the transcript message type.
func (p *Provider) parseTranscript(payload *VapiWebhookPayload) (*voiceprovider.CallEvent, error) {
	call := payload.Message.Call
	return &voiceprovider.CallEvent{
		Provider:       voiceprovider.ProviderVapi,
		ProviderCallID: call.ID,
		Status:         voiceprovider.CallStatusInProgress,
		Transcript:     payload.Message.Transcript,
	}, nil
}

// normalizeStatus converts Vapi-specific status to normalized CallStatus.
func (p *Provider) normalizeStatus(status string) voiceprovider.CallStatus {
	switch status {
	case "ended", "completed":
		return voiceprovider.CallStatusCompleted
	case "failed", "error":
		return voiceprovider.CallStatusFailed
	case "no-answer":
		return voiceprovider.CallStatusNoAnswer
	case "in-progress", "ringing", "queued":
		return voiceprovider.CallStatusInProgress
	case "forwarding":
		return voiceprovider.CallStatusTransferred
	default:
		return voiceprovider.CallStatusPending
	}
}

// extractData extracts structured data from Vapi's analysis.
func (p *Provider) extractData(analysis *VapiAnalysis) *voiceprovider.ExtractedData {
	if analysis == nil {
		return nil
	}

	data := &voiceprovider.ExtractedData{
		Custom: make(map[string]interface{}),
	}

	// Vapi's analysis is more flexible - map to our structure
	if analysis.StructuredData != nil {
		if v, ok := analysis.StructuredData["project_type"].(string); ok {
			data.ProjectType = v
		}
		if v, ok := analysis.StructuredData["requirements"].(string); ok {
			data.Requirements = v
		}
		if v, ok := analysis.StructuredData["timeline"].(string); ok {
			data.Timeline = v
		}
		if v, ok := analysis.StructuredData["budget_range"].(string); ok {
			data.BudgetRange = v
		}
		if v, ok := analysis.StructuredData["contact_preference"].(string); ok {
			data.ContactPreference = v
		}
		if v, ok := analysis.StructuredData["caller_name"].(string); ok {
			data.CallerName = v
		}

		// Copy all to custom
		for k, v := range analysis.StructuredData {
			data.Custom[k] = v
		}
	}

	return data
}

// VapiWebhookPayload represents the data sent by Vapi webhooks.
// This structure accommodates multiple message types.
type VapiWebhookPayload struct {
	Message VapiMessage `json:"message"`
}

// VapiMessage represents a Vapi webhook message.
type VapiMessage struct {
	Type         string         `json:"type"`
	Call         VapiCall       `json:"call"`
	Status       string         `json:"status,omitempty"`
	Transcript   string         `json:"transcript,omitempty"`
	Summary      string         `json:"summary,omitempty"`
	RecordingURL string         `json:"recordingUrl,omitempty"`
	Messages     []VapiTranscriptMessage `json:"messages,omitempty"`
	Analysis     *VapiAnalysis  `json:"analysis,omitempty"`
	EndedReason  string         `json:"endedReason,omitempty"`
}

// VapiCall represents a call object in Vapi.
type VapiCall struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"orgId,omitempty"`
	Type        string          `json:"type,omitempty"` // "inboundPhoneCall", "outboundPhoneCall", "webCall"
	Status      string          `json:"status,omitempty"`
	StartedAt   string          `json:"startedAt,omitempty"`
	EndedAt     string          `json:"endedAt,omitempty"`
	Customer    VapiCustomer    `json:"customer,omitempty"`
	PhoneNumber VapiPhoneNumber `json:"phoneNumber,omitempty"`
}

// VapiCustomer represents the customer in a Vapi call.
type VapiCustomer struct {
	Number string `json:"number,omitempty"`
	Name   string `json:"name,omitempty"`
}

// VapiPhoneNumber represents the phone number configuration.
type VapiPhoneNumber struct {
	ID     string `json:"id,omitempty"`
	Number string `json:"number,omitempty"`
}

// VapiTranscriptMessage represents a message in the conversation.
type VapiTranscriptMessage struct {
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	StartTime float64 `json:"startTime,omitempty"`
	EndTime   float64 `json:"endTime,omitempty"`
}

// VapiAnalysis represents the post-call analysis from Vapi.
type VapiAnalysis struct {
	Summary        string                 `json:"summary,omitempty"`
	StructuredData map[string]interface{} `json:"structuredData,omitempty"`
	SuccessScore   float64                `json:"successScore,omitempty"`
}
