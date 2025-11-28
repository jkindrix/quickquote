package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Prompt represents an AI agent prompt/task configuration for voice calls.
// Prompts define how the AI agent should behave during a call.
type Prompt struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`

	// Task is the actual prompt text - instructions for the AI agent
	Task string `json:"task"`

	// Voice settings
	Voice    string `json:"voice,omitempty"`    // Voice ID or preset name
	Language string `json:"language,omitempty"` // Language code (en-US, es, etc.)

	// Model and behavior settings
	Model                 string   `json:"model,omitempty"`       // "base" or "turbo"
	Temperature           *float64 `json:"temperature,omitempty"` // 0-1, controls creativity
	InterruptionThreshold *int     `json:"interruption_threshold,omitempty"`
	MaxDuration           *int     `json:"max_duration,omitempty"` // Minutes

	// Opening and closing
	FirstSentence string `json:"first_sentence,omitempty"`
	WaitForGreeting bool   `json:"wait_for_greeting,omitempty"`

	// Transfer settings
	TransferPhoneNumber string            `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string `json:"transfer_list,omitempty"`

	// Voicemail handling
	VoicemailAction  string `json:"voicemail_action,omitempty"` // hangup, leave_message, ignore
	VoicemailMessage string `json:"voicemail_message,omitempty"`

	// Recording and audio
	Record            bool    `json:"record,omitempty"`
	BackgroundTrack   *string `json:"background_track,omitempty"`
	NoiseCancellation bool    `json:"noise_cancellation,omitempty"`

	// Knowledge and tools
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
	CustomToolIDs    []string `json:"custom_tool_ids,omitempty"`

	// Post-call analysis
	SummaryPrompt  string                 `json:"summary_prompt,omitempty"`
	Dispositions   []string               `json:"dispositions,omitempty"`
	AnalysisSchema map[string]interface{} `json:"analysis_schema,omitempty"` // JSON schema for data extraction
	Keywords       []string               `json:"keywords,omitempty"`        // Boost transcription accuracy

	// Organization
	IsDefault bool `json:"is_default,omitempty"` // Default prompt for new calls
	IsActive  bool `json:"is_active"`            // Whether prompt can be used

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// NewPrompt creates a new prompt with sensible defaults.
func NewPrompt(name, task string) *Prompt {
	now := time.Now()
	temp := 0.7

	return &Prompt{
		ID:          uuid.New(),
		Name:        name,
		Task:        task,
		Voice:       "maya",      // Default voice
		Language:    "en-US",     // Default language
		Model:       "base",      // Default model
		Temperature: &temp,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate validates the prompt fields.
func (p *Prompt) Validate() error {
	if p.Name == "" {
		return ErrPromptNameRequired
	}
	if p.Task == "" {
		return ErrPromptTaskRequired
	}
	if p.Temperature != nil && (*p.Temperature < 0 || *p.Temperature > 1) {
		return ErrPromptTemperatureInvalid
	}
	if p.MaxDuration != nil && *p.MaxDuration < 1 {
		return ErrPromptMaxDurationInvalid
	}
	return nil
}

// PromptRepository defines the interface for prompt persistence.
type PromptRepository interface {
	Create(ctx context.Context, prompt *Prompt) error
	GetByID(ctx context.Context, id uuid.UUID) (*Prompt, error)
	GetByName(ctx context.Context, name string) (*Prompt, error)
	GetDefault(ctx context.Context) (*Prompt, error)
	List(ctx context.Context, limit, offset int, activeOnly bool) ([]*Prompt, error)
	Count(ctx context.Context, activeOnly bool) (int, error)
	Update(ctx context.Context, prompt *Prompt) error
	Delete(ctx context.Context, id uuid.UUID) error // Soft delete
	SetDefault(ctx context.Context, id uuid.UUID) error
}

// Prompt errors
var (
	ErrPromptNameRequired       = NewValidationError("name", "prompt name is required")
	ErrPromptTaskRequired       = NewValidationError("task", "prompt task is required")
	ErrPromptTemperatureInvalid = NewValidationError("temperature", "temperature must be between 0 and 1")
	ErrPromptMaxDurationInvalid = NewValidationError("max_duration", "max duration must be at least 1 minute")
	ErrPromptNotFound           = NewNotFoundError("prompt", "prompt not found")
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// NotFoundError represents a not found error.
type NotFoundError struct {
	Resource string `json:"resource"`
	Message  string `json:"message"`
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(resource, message string) *NotFoundError {
	return &NotFoundError{Resource: resource, Message: message}
}
