// Package webhook handles incoming webhooks from external services.
package webhook

import (
	"encoding/json"
	"time"
)

// BlandWebhookPayload represents the data sent by Bland AI after a call completes.
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

// GetPhoneNumber returns the phone number, handling different field names.
func (p *BlandWebhookPayload) GetPhoneNumber() string {
	if p.PhoneNumber != "" {
		return p.PhoneNumber
	}
	return p.To
}

// GetFromNumber returns the caller's number, handling different field names.
func (p *BlandWebhookPayload) GetFromNumber() string {
	if p.FromNumber != "" {
		return p.FromNumber
	}
	return p.From
}

// GetDurationSeconds returns the duration as an integer number of seconds.
func (p *BlandWebhookPayload) GetDurationSeconds() int {
	return int(p.Duration)
}

// GetTranscript returns the concatenated transcript.
func (p *BlandWebhookPayload) GetTranscript() string {
	return p.ConcatenatedTranscript
}

// IsCompleted returns true if the call was completed successfully.
func (p *BlandWebhookPayload) IsCompleted() bool {
	return p.Status == "completed" || p.Status == "success"
}

// IsFailed returns true if the call failed.
func (p *BlandWebhookPayload) IsFailed() bool {
	return p.Status == "failed" || p.Status == "error"
}

// IsNoAnswer returns true if the call was not answered.
func (p *BlandWebhookPayload) IsNoAnswer() bool {
	return p.Status == "no_answer" || p.Status == "no-answer" || p.AnsweredBy == "voicemail"
}

// ExtractVariable extracts a string variable from the variables map.
func (p *BlandWebhookPayload) ExtractVariable(key string) string {
	if p.Variables == nil {
		return ""
	}
	if val, ok := p.Variables[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		// Try to convert to string via JSON
		if bytes, err := json.Marshal(val); err == nil {
			return string(bytes)
		}
	}
	return ""
}

// ExtractedVariables extracts all known variables into a struct.
func (p *BlandWebhookPayload) ExtractedVariables() ExtractedVariables {
	return ExtractedVariables{
		ProjectType:       p.ExtractVariable("project_type"),
		Requirements:      p.ExtractVariable("requirements"),
		Timeline:          p.ExtractVariable("timeline"),
		BudgetRange:       p.ExtractVariable("budget_range"),
		ContactPreference: p.ExtractVariable("contact_preference"),
		CallerName:        p.ExtractVariable("caller_name"),
	}
}

// ExtractedVariables holds the structured data extracted from a call.
type ExtractedVariables struct {
	ProjectType       string `json:"project_type,omitempty"`
	Requirements      string `json:"requirements,omitempty"`
	Timeline          string `json:"timeline,omitempty"`
	BudgetRange       string `json:"budget_range,omitempty"`
	ContactPreference string `json:"contact_preference,omitempty"`
	CallerName        string `json:"caller_name,omitempty"`
}
