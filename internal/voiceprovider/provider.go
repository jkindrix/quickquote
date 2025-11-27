// Package voiceprovider defines the interface and types for voice AI providers.
// This abstraction allows QuickQuote to work with multiple voice providers
// (Bland AI, Vapi, Retell, self-hosted) without changing core business logic.
package voiceprovider

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// ProviderType identifies which voice provider is being used.
type ProviderType string

const (
	ProviderBland    ProviderType = "bland"
	ProviderVapi     ProviderType = "vapi"
	ProviderRetell   ProviderType = "retell"
	ProviderLiveKit  ProviderType = "livekit"
	ProviderCustom   ProviderType = "custom"
)

// CallStatus represents the normalized status of a call across all providers.
type CallStatus string

const (
	CallStatusPending    CallStatus = "pending"
	CallStatusInProgress CallStatus = "in_progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusNoAnswer   CallStatus = "no_answer"
	CallStatusVoicemail  CallStatus = "voicemail"
	CallStatusTransferred CallStatus = "transferred"
)

// TranscriptEntry represents a single message in a conversation transcript.
type TranscriptEntry struct {
	Role      string    `json:"role"`       // "assistant", "user", "system"
	Content   string    `json:"content"`    // The spoken text
	Timestamp float64   `json:"timestamp"`  // Seconds from call start
	StartTime *float64  `json:"start_time,omitempty"` // Start time if available
	EndTime   *float64  `json:"end_time,omitempty"`   // End time if available
}

// ExtractedData holds structured data extracted from the call.
// This is provider-agnostic - each provider adapter normalizes to this format.
type ExtractedData struct {
	// Contact information
	Name           string `json:"name,omitempty"`
	Email          string `json:"email,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Company        string `json:"company,omitempty"`
	CallerName     string `json:"caller_name,omitempty"` // Alias for Name

	// Project details
	ProjectType       string `json:"project_type,omitempty"`
	Requirements      string `json:"requirements,omitempty"`
	Timeline          string `json:"timeline,omitempty"`
	Budget            string `json:"budget,omitempty"`
	BudgetRange       string `json:"budget_range,omitempty"` // Alias for Budget
	ContactPreference string `json:"contact_preference,omitempty"`

	// Additional info
	AdditionalInfo string `json:"additional_info,omitempty"`

	// Flexible key-value store for provider-specific or custom fields
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// CallEvent represents a normalized call event from any voice provider.
// This is the core abstraction that decouples business logic from providers.
type CallEvent struct {
	// Provider identification
	Provider     ProviderType `json:"provider"`
	ProviderCallID string     `json:"provider_call_id"` // ID from the voice provider

	// Call participants
	ToNumber     string `json:"to_number"`     // Number that received the call
	FromNumber   string `json:"from_number"`   // Caller's number
	CallerName   string `json:"caller_name,omitempty"`

	// Call lifecycle
	Status       CallStatus `json:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	DurationSecs int        `json:"duration_secs,omitempty"`

	// Conversation content
	Transcript          string            `json:"transcript,omitempty"`           // Full concatenated transcript
	TranscriptEntries   []TranscriptEntry `json:"transcript_entries,omitempty"`   // Structured transcript

	// Extracted information
	ExtractedData *ExtractedData `json:"extracted_data,omitempty"`

	// Recording
	RecordingURL string `json:"recording_url,omitempty"`

	// Error information
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`

	// Provider-specific metadata (for debugging/advanced use)
	RawMetadata map[string]interface{} `json:"raw_metadata,omitempty"`

	// Call disposition/outcome (if provider supports it)
	Disposition string `json:"disposition,omitempty"`
	Summary     string `json:"summary,omitempty"` // Provider-generated summary if available
}

// HasTranscript returns true if the call event has a non-empty transcript.
func (e *CallEvent) HasTranscript() bool {
	return strings.TrimSpace(e.Transcript) != ""
}

// IsComplete returns true if the call is in a terminal state.
func (e *CallEvent) IsComplete() bool {
	switch e.Status {
	case CallStatusCompleted, CallStatusFailed, CallStatusNoAnswer:
		return true
	default:
		return false
	}
}

// Provider defines the interface that all voice providers must implement.
type Provider interface {
	// GetName returns the provider type identifier.
	GetName() ProviderType

	// ParseWebhook parses an incoming webhook request into a normalized CallEvent.
	// Returns an error if the payload is invalid or cannot be parsed.
	ParseWebhook(r *http.Request) (*CallEvent, error)

	// ValidateWebhook verifies the webhook signature/authenticity.
	// Returns true if the webhook is authentic, false otherwise.
	// If the provider doesn't support webhook validation, always returns true.
	ValidateWebhook(r *http.Request) bool

	// GetWebhookPath returns the path this provider's webhooks should be sent to.
	// Example: "/webhook/bland", "/webhook/vapi"
	GetWebhookPath() string
}

// OutboundProvider extends Provider with the ability to initiate calls.
// Not all providers need to implement this - it's optional for inbound-only setups.
type OutboundProvider interface {
	Provider

	// InitiateCall starts an outbound call to the given number.
	InitiateCall(ctx context.Context, req OutboundCallRequest) (*OutboundCallResponse, error)

	// GetCallStatus retrieves the current status of a call by provider ID.
	GetCallStatus(ctx context.Context, providerCallID string) (*CallEvent, error)
}

// OutboundCallRequest contains parameters for initiating an outbound call.
type OutboundCallRequest struct {
	ToNumber     string                 `json:"to_number"`
	FromNumber   string                 `json:"from_number,omitempty"` // If blank, use default
	Task         string                 `json:"task,omitempty"`        // Prompt/task for the AI
	PathwayID    string                 `json:"pathway_id,omitempty"`  // For pathway-based calls
	Voice        string                 `json:"voice,omitempty"`       // Voice ID to use
	FirstMessage string                 `json:"first_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// OutboundCallResponse contains the result of initiating an outbound call.
type OutboundCallResponse struct {
	ProviderCallID string `json:"provider_call_id"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
}

// ConfigurableProvider extends Provider with runtime configuration capabilities.
type ConfigurableProvider interface {
	Provider

	// UpdateAgentConfig updates the agent/prompt configuration.
	UpdateAgentConfig(ctx context.Context, cfg AgentConfig) error

	// GetAgentConfig retrieves the current agent configuration.
	GetAgentConfig(ctx context.Context) (*AgentConfig, error)
}

// AgentConfig holds configuration for the voice AI agent.
type AgentConfig struct {
	Prompt             string                 `json:"prompt,omitempty"`
	PathwayID          string                 `json:"pathway_id,omitempty"`
	Voice              string                 `json:"voice,omitempty"`
	FirstMessage       string                 `json:"first_message,omitempty"`
	InterruptThreshold int                    `json:"interrupt_threshold,omitempty"` // ms
	BackgroundTrack    string                 `json:"background_track,omitempty"`
	NoiseCancellation  bool                   `json:"noise_cancellation,omitempty"`
	MaxDuration        int                    `json:"max_duration,omitempty"` // minutes
	Language           string                 `json:"language,omitempty"`
	Temperature        float64                `json:"temperature,omitempty"`
	Keywords           []string               `json:"keywords,omitempty"`
	TransferNumber     string                 `json:"transfer_number,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}
