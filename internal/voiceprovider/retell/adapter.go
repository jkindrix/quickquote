// Package retell implements the VoiceProvider interface for Retell AI.
// See: https://docs.retellai.com/api-references
package retell

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

// Config holds Retell AI provider configuration.
type Config struct {
	APIKey        string
	WebhookSecret string
	APIURL        string
}

// Provider implements the voiceprovider.Provider interface for Retell AI.
type Provider struct {
	config *Config
	logger *zap.Logger
}

// New creates a new Retell AI provider.
func New(cfg *Config, logger *zap.Logger) *Provider {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.retellai.com"
	}
	return &Provider{
		config: cfg,
		logger: logger,
	}
}

// GetName returns the provider type identifier.
func (p *Provider) GetName() voiceprovider.ProviderType {
	return voiceprovider.ProviderRetell
}

// GetWebhookPath returns the path for Retell webhooks.
func (p *Provider) GetWebhookPath() string {
	return "/webhook/retell"
}

// ValidateWebhook verifies the webhook signature.
// Retell uses HMAC-SHA256 for webhook authentication.
func (p *Provider) ValidateWebhook(r *http.Request) bool {
	// If no webhook secret is configured, skip validation
	// NOTE: In production, webhook secrets should always be configured
	if p.config.WebhookSecret == "" {
		p.logger.Warn("webhook secret not configured, skipping signature validation")
		return true
	}

	signature := r.Header.Get("X-Retell-Signature")
	if signature == "" {
		p.logger.Warn("webhook missing X-Retell-Signature header",
			zap.String("provider", "retell"),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

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

	// Use constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		p.logger.Warn("webhook signature mismatch",
			zap.String("provider", "retell"),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	p.logger.Debug("webhook signature validated successfully",
		zap.String("provider", "retell"),
	)
	return true
}

// ParseWebhook parses a Retell AI webhook into a normalized CallEvent.
func (p *Provider) ParseWebhook(r *http.Request) (*voiceprovider.CallEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload RetellWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event, err := p.toCallEvent(&payload)
	if err != nil {
		return nil, err
	}

	// Store raw payload for debugging
	var rawMetadata map[string]interface{}
	if err := json.Unmarshal(body, &rawMetadata); err == nil {
		event.RawMetadata = rawMetadata
	}

	p.logger.Debug("parsed retell webhook",
		zap.String("event", payload.Event),
		zap.String("call_id", event.ProviderCallID),
	)

	return event, nil
}

// validatePayload validates and sanitizes the webhook payload.
func (p *Provider) validatePayload(payload *RetellWebhookPayload) error {
	v := validation.NewCallEventValidator()
	call := payload.Call

	// Validate call ID (required)
	v.ValidateCallID(call.CallID)

	// Validate phone numbers
	v.ValidatePhoneNumbers(call.ToNumber, call.FromNumber)

	// Validate transcript content
	v.ValidateTranscript(call.Transcript)

	// Validate recording URL
	v.ValidateRecordingURL(call.RecordingURL)

	// Check for validation errors
	if !v.IsValid() {
		errs := v.Errors()
		p.logger.Warn("webhook payload validation failed",
			zap.String("provider", "retell"),
			zap.String("call_id", call.CallID),
			zap.Int("error_count", len(errs)),
			zap.String("errors", errs.Error()),
		)
		return errs
	}

	// Sanitize strings that passed validation
	payload.Call.Transcript = validation.SanitizeString(payload.Call.Transcript)
	payload.Call.DisconnectionReason = validation.SanitizeString(payload.Call.DisconnectionReason)

	return nil
}

// toCallEvent converts a Retell-specific payload to a normalized CallEvent.
func (p *Provider) toCallEvent(payload *RetellWebhookPayload) (*voiceprovider.CallEvent, error) {
	call := payload.Call
	if call.CallID == "" {
		return nil, fmt.Errorf("missing call_id in webhook")
	}

	// Validate and sanitize payload
	if err := p.validatePayload(payload); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	event := &voiceprovider.CallEvent{
		Provider:       voiceprovider.ProviderRetell,
		ProviderCallID: call.CallID,
		ToNumber:       call.ToNumber,
		FromNumber:     call.FromNumber,
		Status:         p.normalizeStatus(payload.Event, call.CallStatus),
		Transcript:     call.Transcript,
		RecordingURL:   call.RecordingURL,
	}

	// Parse timestamps
	if call.StartTimestamp > 0 {
		t := time.Unix(call.StartTimestamp/1000, 0) // Retell uses milliseconds
		event.StartedAt = &t
	}
	if call.EndTimestamp > 0 {
		t := time.Unix(call.EndTimestamp/1000, 0)
		event.EndedAt = &t
	}

	// Calculate duration
	if call.StartTimestamp > 0 && call.EndTimestamp > 0 {
		event.DurationSecs = int((call.EndTimestamp - call.StartTimestamp) / 1000)
	}

	// Parse structured transcript
	if len(call.TranscriptObject) > 0 {
		event.TranscriptEntries = make([]voiceprovider.TranscriptEntry, len(call.TranscriptObject))
		for i, t := range call.TranscriptObject {
			event.TranscriptEntries[i] = voiceprovider.TranscriptEntry{
				Role:    t.Role,
				Content: t.Content,
			}
		}
	}

	// Extract custom data from call analysis
	if call.CallAnalysis != nil {
		event.ExtractedData = p.extractData(call.CallAnalysis)
		event.Summary = call.CallAnalysis.CallSummary
	}

	// Set disposition if available
	event.Disposition = call.DisconnectionReason

	return event, nil
}

// normalizeStatus converts Retell-specific status to normalized CallStatus.
func (p *Provider) normalizeStatus(event, callStatus string) voiceprovider.CallStatus {
	// Event-based status takes precedence
	switch event {
	case "call_ended":
		return p.normalizeCallStatus(callStatus)
	case "call_started":
		return voiceprovider.CallStatusInProgress
	case "call_analyzed":
		return voiceprovider.CallStatusCompleted
	}

	// Fall back to call status
	return p.normalizeCallStatus(callStatus)
}

// normalizeCallStatus converts Retell's call_status to normalized CallStatus.
func (p *Provider) normalizeCallStatus(status string) voiceprovider.CallStatus {
	switch status {
	case "ended", "registered":
		return voiceprovider.CallStatusCompleted
	case "error":
		return voiceprovider.CallStatusFailed
	case "ongoing":
		return voiceprovider.CallStatusInProgress
	default:
		return voiceprovider.CallStatusPending
	}
}

// extractData extracts structured data from Retell's call analysis.
func (p *Provider) extractData(analysis *RetellCallAnalysis) *voiceprovider.ExtractedData {
	if analysis == nil {
		return nil
	}

	data := &voiceprovider.ExtractedData{
		Custom: make(map[string]interface{}),
	}

	// Map custom analysis data to our structure
	if analysis.CustomAnalysisData != nil {
		if v, ok := analysis.CustomAnalysisData["project_type"].(string); ok {
			data.ProjectType = v
		}
		if v, ok := analysis.CustomAnalysisData["requirements"].(string); ok {
			data.Requirements = v
		}
		if v, ok := analysis.CustomAnalysisData["timeline"].(string); ok {
			data.Timeline = v
		}
		if v, ok := analysis.CustomAnalysisData["budget_range"].(string); ok {
			data.BudgetRange = v
		}
		if v, ok := analysis.CustomAnalysisData["contact_preference"].(string); ok {
			data.ContactPreference = v
		}
		if v, ok := analysis.CustomAnalysisData["caller_name"].(string); ok {
			data.CallerName = v
		}

		// Copy all to custom
		for k, v := range analysis.CustomAnalysisData {
			data.Custom[k] = v
		}
	}

	return data
}

// RetellWebhookPayload represents the data sent by Retell AI webhooks.
type RetellWebhookPayload struct {
	Event string     `json:"event"` // "call_started", "call_ended", "call_analyzed"
	Call  RetellCall `json:"call"`
}

// RetellCall represents a call object in Retell AI.
type RetellCall struct {
	CallID               string                   `json:"call_id"`
	AgentID              string                   `json:"agent_id,omitempty"`
	CallType             string                   `json:"call_type,omitempty"` // "inbound", "outbound", "web_call"
	CallStatus           string                   `json:"call_status,omitempty"`
	FromNumber           string                   `json:"from_number,omitempty"`
	ToNumber             string                   `json:"to_number,omitempty"`
	StartTimestamp       int64                    `json:"start_timestamp,omitempty"` // milliseconds
	EndTimestamp         int64                    `json:"end_timestamp,omitempty"`   // milliseconds
	Transcript           string                   `json:"transcript,omitempty"`
	TranscriptObject     []RetellTranscriptEntry  `json:"transcript_object,omitempty"`
	RecordingURL         string                   `json:"recording_url,omitempty"`
	PublicLogURL         string                   `json:"public_log_url,omitempty"`
	DisconnectionReason  string                   `json:"disconnection_reason,omitempty"`
	CallAnalysis         *RetellCallAnalysis      `json:"call_analysis,omitempty"`
	Metadata             map[string]interface{}   `json:"metadata,omitempty"`
}

// RetellTranscriptEntry represents a single message in the conversation.
type RetellTranscriptEntry struct {
	Role    string `json:"role"`    // "agent", "user"
	Content string `json:"content"`
	Words   []RetellWord `json:"words,omitempty"`
}

// RetellWord represents a single word with timing information.
type RetellWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// RetellCallAnalysis represents the post-call analysis from Retell.
type RetellCallAnalysis struct {
	CallSummary        string                 `json:"call_summary,omitempty"`
	CallSentiment      string                 `json:"call_sentiment,omitempty"` // "positive", "negative", "neutral"
	InVoicemailDetected bool                  `json:"in_voicemail_detected,omitempty"`
	UserSentiment      string                 `json:"user_sentiment,omitempty"`
	CustomAnalysisData map[string]interface{} `json:"custom_analysis_data,omitempty"`
}
