// Package bland implements the VoiceProvider interface for Bland AI.
package bland

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

// Config holds Bland AI provider configuration.
type Config struct {
	APIKey        string
	WebhookSecret string
	APIURL        string
}

// Provider implements the voiceprovider.Provider interface for Bland AI.
type Provider struct {
	config *Config
	logger *zap.Logger
}

// New creates a new Bland AI provider.
func New(cfg *Config, logger *zap.Logger) *Provider {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.bland.ai/v1"
	}
	return &Provider{
		config: cfg,
		logger: logger,
	}
}

// GetName returns the provider type identifier.
func (p *Provider) GetName() voiceprovider.ProviderType {
	return voiceprovider.ProviderBland
}

// GetWebhookPath returns the path for Bland webhooks.
func (p *Provider) GetWebhookPath() string {
	return "/webhook/bland"
}

// ValidateWebhook verifies the webhook signature if a secret is configured.
func (p *Provider) ValidateWebhook(r *http.Request) bool {
	// If no webhook secret is configured, skip validation
	// NOTE: In production, webhook secrets should always be configured
	if p.config.WebhookSecret == "" {
		p.logger.Warn("webhook secret not configured, skipping signature validation")
		return true
	}

	// Bland AI uses X-Webhook-Secret header for signature validation
	// The signature is an HMAC-SHA256 of the request body
	signature := r.Header.Get("X-Webhook-Secret")
	if signature == "" {
		// Also check alternative header names Bland might use
		signature = r.Header.Get("X-Bland-Signature")
	}
	if signature == "" {
		p.logger.Warn("webhook missing signature header",
			zap.String("provider", "bland"),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	// Read the body for signature computation
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
			zap.String("provider", "bland"),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	p.logger.Debug("webhook signature validated successfully",
		zap.String("provider", "bland"),
	)
	return true
}

// ParseWebhook parses a Bland AI webhook into a normalized CallEvent.
func (p *Provider) ParseWebhook(r *http.Request) (*voiceprovider.CallEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload BlandWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Validate required fields and sanitize input
	if err := p.validatePayload(&payload); err != nil {
		return nil, fmt.Errorf("webhook validation failed: %w", err)
	}

	// Convert to normalized CallEvent
	event := p.toCallEvent(&payload)

	// Store raw payload for debugging
	var rawMetadata map[string]interface{}
	if err := json.Unmarshal(body, &rawMetadata); err == nil {
		event.RawMetadata = rawMetadata
	}

	p.logger.Debug("parsed bland webhook",
		zap.String("call_id", payload.CallID),
		zap.String("status", string(event.Status)),
	)

	return event, nil
}

// validatePayload validates and sanitizes the webhook payload.
func (p *Provider) validatePayload(payload *BlandWebhookPayload) error {
	v := validation.NewCallEventValidator()

	// Validate call ID (required)
	v.ValidateCallID(payload.CallID)

	// Validate phone numbers
	v.ValidatePhoneNumbers(payload.getToNumber(), payload.getFromNumber())

	// Validate transcript content
	v.ValidateTranscript(payload.ConcatenatedTranscript)

	// Validate recording URL
	v.ValidateRecordingURL(payload.RecordingURL)

	// Validate duration
	v.ValidateDuration(int(payload.Duration))

	// Check for validation errors
	if !v.IsValid() {
		errs := v.Errors()
		p.logger.Warn("webhook payload validation failed",
			zap.String("provider", "bland"),
			zap.String("call_id", payload.CallID),
			zap.Int("error_count", len(errs)),
			zap.String("errors", errs.Error()),
		)
		return errs
	}

	// Sanitize strings that passed validation
	payload.ConcatenatedTranscript = validation.SanitizeString(payload.ConcatenatedTranscript)
	payload.ErrorMessage = validation.SanitizeString(payload.ErrorMessage)
	payload.Summary = validation.SanitizeString(payload.Summary)
	payload.Disposition = validation.SanitizeString(payload.Disposition)

	return nil
}

// toCallEvent converts a Bland-specific payload to a normalized CallEvent.
func (p *Provider) toCallEvent(payload *BlandWebhookPayload) *voiceprovider.CallEvent {
	event := &voiceprovider.CallEvent{
		Provider:       voiceprovider.ProviderBland,
		ProviderCallID: payload.CallID,
		ToNumber:       payload.getToNumber(),
		FromNumber:     payload.getFromNumber(),
		Status:         p.normalizeStatus(payload),
		DurationSecs:   int(payload.Duration),
		Transcript:     payload.ConcatenatedTranscript,
		RecordingURL:   payload.RecordingURL,
		ErrorMessage:   payload.ErrorMessage,
		Disposition:    payload.Disposition,
		Summary:        payload.Summary,
	}

	// Convert timestamps
	if payload.StartTime != nil {
		event.StartedAt = payload.StartTime
	}
	if payload.EndTime != nil {
		event.EndedAt = payload.EndTime
	}

	// Convert transcript entries
	if len(payload.Transcripts) > 0 {
		event.TranscriptEntries = make([]voiceprovider.TranscriptEntry, len(payload.Transcripts))
		for i, t := range payload.Transcripts {
			event.TranscriptEntries[i] = voiceprovider.TranscriptEntry{
				Role:      t.Role,
				Content:   t.Content,
				Timestamp: t.Timestamp,
			}
		}
	}

	// Extract structured data from variables
	event.ExtractedData = p.extractData(payload)

	// Set caller name if available
	if event.ExtractedData != nil && event.ExtractedData.CallerName != "" {
		event.CallerName = event.ExtractedData.CallerName
	}

	return event
}

// normalizeStatus converts Bland-specific status to normalized CallStatus.
func (p *Provider) normalizeStatus(payload *BlandWebhookPayload) voiceprovider.CallStatus {
	switch payload.Status {
	case "completed", "success":
		return voiceprovider.CallStatusCompleted
	case "failed", "error":
		return voiceprovider.CallStatusFailed
	case "no_answer", "no-answer":
		return voiceprovider.CallStatusNoAnswer
	case "in_progress", "in-progress", "active":
		return voiceprovider.CallStatusInProgress
	case "voicemail":
		return voiceprovider.CallStatusVoicemail
	case "transferred":
		return voiceprovider.CallStatusTransferred
	default:
		// Check answered_by for voicemail detection
		if payload.AnsweredBy == "voicemail" {
			return voiceprovider.CallStatusVoicemail
		}
		if payload.Status == "" {
			return voiceprovider.CallStatusPending
		}
		// Log unknown status for debugging
		p.logger.Warn("unknown bland status", zap.String("status", payload.Status))
		return voiceprovider.CallStatusPending
	}
}

// extractData extracts structured data from Bland's variables map.
func (p *Provider) extractData(payload *BlandWebhookPayload) *voiceprovider.ExtractedData {
	if payload.Variables == nil {
		return nil
	}

	data := &voiceprovider.ExtractedData{
		Custom: make(map[string]interface{}),
	}

	// Extract known fields
	data.ProjectType = getStringFromMap(payload.Variables, "project_type")
	data.Requirements = getStringFromMap(payload.Variables, "requirements")
	data.Timeline = getStringFromMap(payload.Variables, "timeline")
	data.BudgetRange = getStringFromMap(payload.Variables, "budget_range")
	data.ContactPreference = getStringFromMap(payload.Variables, "contact_preference")
	data.CallerName = getStringFromMap(payload.Variables, "caller_name")

	// Copy all variables to custom for flexibility
	for k, v := range payload.Variables {
		data.Custom[k] = v
	}

	return data
}

// getStringFromMap safely extracts a string value from a map.
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		// Try JSON marshaling for complex types
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes)
		}
	}
	return ""
}

// BlandWebhookPayload represents the data sent by Bland AI after a call.
type BlandWebhookPayload struct {
	CallID               string                 `json:"call_id"`
	BatchID              string                 `json:"batch_id,omitempty"`
	PhoneNumber          string                 `json:"phone_number"`
	FromNumber           string                 `json:"from_number"`
	To                   string                 `json:"to,omitempty"`
	From                 string                 `json:"from,omitempty"`
	Status               string                 `json:"status"`
	AnsweredBy           string                 `json:"answered_by,omitempty"`
	Duration             float64                `json:"duration,omitempty"`
	StartTime            *time.Time             `json:"start_time,omitempty"`
	EndTime              *time.Time             `json:"end_time,omitempty"`
	RecordingURL         string                 `json:"recording_url,omitempty"`
	ConcatenatedTranscript string               `json:"concatenated_transcript,omitempty"`
	Transcripts          []TranscriptMessage    `json:"transcripts,omitempty"`
	Variables            map[string]interface{} `json:"variables,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	ErrorMessage         string                 `json:"error_message,omitempty"`
	CallEndedBy          string                 `json:"call_ended_by,omitempty"`
	Disposition          string                 `json:"disposition,omitempty"`
	Summary              string                 `json:"summary,omitempty"`
	Price                float64                `json:"price,omitempty"`
}

// TranscriptMessage represents a single message in the conversation.
type TranscriptMessage struct {
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Timestamp float64 `json:"timestamp,omitempty"`
}

// getToNumber returns the phone number, handling different field names.
func (p *BlandWebhookPayload) getToNumber() string {
	if p.PhoneNumber != "" {
		return p.PhoneNumber
	}
	return p.To
}

// getFromNumber returns the caller's number, handling different field names.
func (p *BlandWebhookPayload) getFromNumber() string {
	if p.FromNumber != "" {
		return p.FromNumber
	}
	return p.From
}
