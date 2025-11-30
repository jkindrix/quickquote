// Package domain contains the core business entities and interfaces.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CallStatus represents the status of a call.
type CallStatus string

const (
	CallStatusPending    CallStatus = "pending"
	CallStatusInProgress CallStatus = "in_progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusNoAnswer   CallStatus = "no_answer"
)

// Call represents a phone call record.
type Call struct {
	ID              uuid.UUID         `json:"id"`
	ProviderCallID  string            `json:"provider_call_id"` // ID from voice provider (Bland, Vapi, Retell, etc.)
	Provider        string            `json:"provider"`         // Provider type: "bland", "vapi", "retell", etc.
	PhoneNumber     string            `json:"phone_number"`     // Number that received the call (to)
	FromNumber      string            `json:"from_number"`      // Caller's number
	CallerName      *string           `json:"caller_name,omitempty"`
	Status          CallStatus        `json:"status"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	EndedAt         *time.Time        `json:"ended_at,omitempty"`
	DurationSeconds *int              `json:"duration_seconds,omitempty"`
	Transcript      *string           `json:"transcript,omitempty"`
	TranscriptJSON  []TranscriptEntry `json:"transcript_json,omitempty"`
	RecordingURL    *string           `json:"recording_url,omitempty"`
	QuoteSummary    *string           `json:"quote_summary,omitempty"`
	ExtractedData   *ExtractedData    `json:"extracted_data,omitempty"`
	ErrorMessage    *string           `json:"error_message,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       *time.Time        `json:"deleted_at,omitempty"`
}

// IsDeleted returns true if the call has been soft-deleted.
func (c *Call) IsDeleted() bool {
	return c.DeletedAt != nil
}

// MarkDeleted soft-deletes the call by setting DeletedAt.
func (c *Call) MarkDeleted() {
	now := time.Now().UTC()
	c.DeletedAt = &now
	c.UpdatedAt = now
}

// TranscriptEntry represents a single message in the call transcript.
type TranscriptEntry struct {
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Timestamp float64 `json:"timestamp"`
}

// ExtractedData holds structured data extracted from the call.
type ExtractedData struct {
	ProjectType       string `json:"project_type,omitempty"`
	Requirements      string `json:"requirements,omitempty"`
	Timeline          string `json:"timeline,omitempty"`
	BudgetRange       string `json:"budget_range,omitempty"`
	ContactPreference string `json:"contact_preference,omitempty"`
	CallerName        string `json:"caller_name,omitempty"`
}

// NewCall creates a new Call with default values.
func NewCall(providerCallID, provider, phoneNumber, fromNumber string) *Call {
	now := time.Now().UTC()
	return &Call{
		ID:             uuid.New(),
		ProviderCallID: providerCallID,
		Provider:       provider,
		PhoneNumber:    phoneNumber,
		FromNumber:     fromNumber,
		Status:         CallStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsComplete returns true if the call has ended.
func (c *Call) IsComplete() bool {
	return c.Status == CallStatusCompleted || c.Status == CallStatusFailed || c.Status == CallStatusNoAnswer
}

// HasQuote returns true if a quote has been generated.
func (c *Call) HasQuote() bool {
	return c.QuoteSummary != nil && *c.QuoteSummary != ""
}

// Duration returns the call duration as a time.Duration.
func (c *Call) Duration() time.Duration {
	if c.DurationSeconds == nil {
		return 0
	}
	return time.Duration(*c.DurationSeconds) * time.Second
}

// FormattedDuration returns the duration as a human-readable string.
func (c *Call) FormattedDuration() string {
	d := c.Duration()
	if d == 0 {
		return "-"
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// CallListFilter defines optional filters for listing calls.
type CallListFilter struct {
	Status *CallStatus
	Search string
}

// HasFilters returns true if any filter fields are set.
func (f *CallListFilter) HasFilters() bool {
	if f == nil {
		return false
	}
	if f.Status != nil {
		return true
	}
	return strings.TrimSpace(f.Search) != ""
}
