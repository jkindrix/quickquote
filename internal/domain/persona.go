package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Persona represents a local cache of a Bland AI persona.
// Personas define the personality, behavior, and characteristics
// of the AI agent during calls.
type Persona struct {
	ID              uuid.UUID `json:"id" db:"id"`
	BlandID         string    `json:"bland_id,omitempty" db:"bland_id"` // Bland's persona ID
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description,omitempty" db:"description"`

	// Voice and Speech
	Voice           string    `json:"voice,omitempty" db:"voice"`
	Language        string    `json:"language,omitempty" db:"language"`
	VoiceSettingsJSON string  `json:"-" db:"voice_settings"`
	VoiceSettings   *PersonaVoiceSettings `json:"voice_settings,omitempty" db:"-"`

	// Personality
	Personality     string    `json:"personality,omitempty" db:"personality"`
	BackgroundStory string    `json:"background_story,omitempty" db:"background_story"`
	SystemPrompt    string    `json:"system_prompt,omitempty" db:"system_prompt"`

	// Behavior
	BehaviorJSON    string    `json:"-" db:"behavior"`
	Behavior        *PersonaBehavior `json:"behavior,omitempty" db:"-"`

	// Knowledge and Tools
	KnowledgeBases  []string  `json:"knowledge_bases,omitempty" db:"-"`
	KBsJSON         string    `json:"-" db:"knowledge_bases"`
	Tools           []string  `json:"tools,omitempty" db:"-"`
	ToolsJSON       string    `json:"-" db:"tools"`

	// Status
	Status          string    `json:"status" db:"status"`
	IsDefault       bool      `json:"is_default" db:"is_default"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty" db:"last_synced_at"`
	SyncError       string    `json:"sync_error,omitempty" db:"sync_error"`

	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the persona has been soft-deleted.
func (p *Persona) IsDeleted() bool {
	return p.DeletedAt != nil
}

// MarkDeleted soft-deletes the persona by setting DeletedAt.
func (p *Persona) MarkDeleted() {
	now := time.Now().UTC()
	p.DeletedAt = &now
	p.UpdatedAt = now
}

// PersonaVoiceSettings contains voice customization parameters.
type PersonaVoiceSettings struct {
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
	SpeechRate      float64 `json:"speech_rate,omitempty"`
	Pitch           float64 `json:"pitch,omitempty"`
}

// PersonaBehavior defines behavioral characteristics.
type PersonaBehavior struct {
	// Response style
	ResponseStyle    string  `json:"response_style,omitempty"`    // concise, detailed, conversational
	Formality        string  `json:"formality,omitempty"`         // casual, professional, formal
	Enthusiasm       float64 `json:"enthusiasm,omitempty"`        // 0.0-1.0

	// Interaction patterns
	InterruptionThreshold int     `json:"interruption_threshold,omitempty"` // 0-100
	WaitForGreeting       bool    `json:"wait_for_greeting,omitempty"`
	AllowInterruptions    bool    `json:"allow_interruptions,omitempty"`

	// Intelligence settings
	Model           string  `json:"model,omitempty"`              // basic, enhanced, turbo
	Temperature     float64 `json:"temperature,omitempty"`
	MaxTokens       int     `json:"max_tokens,omitempty"`

	// Emotional intelligence
	EmpathyLevel    float64 `json:"empathy_level,omitempty"`      // 0.0-1.0
	PatienceLevel   float64 `json:"patience_level,omitempty"`     // 0.0-1.0

	// Boundaries
	MaxCallDuration int     `json:"max_call_duration,omitempty"`
	ProhibitedTopics []string `json:"prohibited_topics,omitempty"`
	EscalationTriggers []string `json:"escalation_triggers,omitempty"`
}

// PersonaStatus constants.
const (
	PersonaStatusActive   = "active"
	PersonaStatusDraft    = "draft"
	PersonaStatusSyncing  = "syncing"
	PersonaStatusError    = "error"
	PersonaStatusArchived = "archived"
)

// PersonaFilter contains filtering options for listing personas.
type PersonaFilter struct {
	Status    string
	Name      string
	IsDefault *bool
	Limit     int
	Offset    int
}

// PersonaRepository defines the interface for persona persistence.
type PersonaRepository interface {
	// Core CRUD
	Create(ctx context.Context, persona *Persona) error
	GetByID(ctx context.Context, id uuid.UUID) (*Persona, error)
	GetByBlandID(ctx context.Context, blandID string) (*Persona, error)
	GetDefault(ctx context.Context) (*Persona, error)
	List(ctx context.Context, filter *PersonaFilter) ([]*Persona, error)
	Update(ctx context.Context, persona *Persona) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Default management
	SetDefault(ctx context.Context, id uuid.UUID) error
	ClearDefault(ctx context.Context) error

	// Sync operations
	MarkSyncing(ctx context.Context, id uuid.UUID) error
	MarkSynced(ctx context.Context, id uuid.UUID, blandID string) error
	MarkSyncError(ctx context.Context, id uuid.UUID, errMsg string) error
}

// NewPersona creates a new persona with generated ID.
func NewPersona(name, description string) *Persona {
	now := time.Now()
	return &Persona{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Status:      PersonaStatusDraft,
		Language:    "en-US",
		Voice:       "maya",
		Behavior:    DefaultBehavior(),
		VoiceSettings: DefaultVoiceSettings(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// DefaultBehavior returns sensible default behavior settings.
func DefaultBehavior() *PersonaBehavior {
	return &PersonaBehavior{
		ResponseStyle:         "conversational",
		Formality:             "professional",
		Enthusiasm:            0.7,
		InterruptionThreshold: 50,
		WaitForGreeting:       true,
		AllowInterruptions:    true,
		Model:                 "enhanced",
		Temperature:           0.7,
		EmpathyLevel:          0.8,
		PatienceLevel:         0.8,
		MaxCallDuration:       600,
	}
}

// DefaultVoiceSettings returns sensible default voice settings.
func DefaultVoiceSettings() *PersonaVoiceSettings {
	return &PersonaVoiceSettings{
		Stability:       0.5,
		SimilarityBoost: 0.75,
		Style:           0.5,
		UseSpeakerBoost: true,
		SpeechRate:      1.0,
		Pitch:           1.0,
	}
}

// IsActive checks if the persona is in active status.
func (p *Persona) IsActive() bool {
	return p.Status == PersonaStatusActive
}

// IsDraft checks if the persona is in draft status.
func (p *Persona) IsDraft() bool {
	return p.Status == PersonaStatusDraft
}

// NeedsSync checks if the persona needs to be synced to Bland.
func (p *Persona) NeedsSync() bool {
	if p.BlandID == "" {
		return true
	}
	if p.LastSyncedAt == nil {
		return true
	}
	return p.UpdatedAt.After(*p.LastSyncedAt)
}

// SetSynced marks the persona as successfully synced.
func (p *Persona) SetSynced(blandID string) {
	now := time.Now()
	p.BlandID = blandID
	p.Status = PersonaStatusActive
	p.LastSyncedAt = &now
	p.SyncError = ""
	p.UpdatedAt = now
}

// SetSyncError marks the persona as having a sync error.
func (p *Persona) SetSyncError(err string) {
	p.Status = PersonaStatusError
	p.SyncError = err
	p.UpdatedAt = time.Now()
}

// MarshalVoiceSettings converts voice settings to JSON for storage.
func (p *Persona) MarshalVoiceSettings() error {
	if p.VoiceSettings == nil {
		p.VoiceSettingsJSON = ""
		return nil
	}
	data, err := json.Marshal(p.VoiceSettings)
	if err != nil {
		return err
	}
	p.VoiceSettingsJSON = string(data)
	return nil
}

// UnmarshalVoiceSettings parses voice settings from JSON storage.
func (p *Persona) UnmarshalVoiceSettings() error {
	if p.VoiceSettingsJSON == "" {
		p.VoiceSettings = nil
		return nil
	}
	p.VoiceSettings = &PersonaVoiceSettings{}
	return json.Unmarshal([]byte(p.VoiceSettingsJSON), p.VoiceSettings)
}

// MarshalBehavior converts behavior to JSON for storage.
func (p *Persona) MarshalBehavior() error {
	if p.Behavior == nil {
		p.BehaviorJSON = ""
		return nil
	}
	data, err := json.Marshal(p.Behavior)
	if err != nil {
		return err
	}
	p.BehaviorJSON = string(data)
	return nil
}

// UnmarshalBehavior parses behavior from JSON storage.
func (p *Persona) UnmarshalBehavior() error {
	if p.BehaviorJSON == "" {
		p.Behavior = nil
		return nil
	}
	p.Behavior = &PersonaBehavior{}
	return json.Unmarshal([]byte(p.BehaviorJSON), p.Behavior)
}

// MarshalKnowledgeBases converts knowledge bases to JSON for storage.
func (p *Persona) MarshalKnowledgeBases() error {
	if len(p.KnowledgeBases) == 0 {
		p.KBsJSON = "[]"
		return nil
	}
	data, err := json.Marshal(p.KnowledgeBases)
	if err != nil {
		return err
	}
	p.KBsJSON = string(data)
	return nil
}

// UnmarshalKnowledgeBases parses knowledge bases from JSON storage.
func (p *Persona) UnmarshalKnowledgeBases() error {
	if p.KBsJSON == "" || p.KBsJSON == "[]" {
		p.KnowledgeBases = []string{}
		return nil
	}
	return json.Unmarshal([]byte(p.KBsJSON), &p.KnowledgeBases)
}

// MarshalTools converts tools to JSON for storage.
func (p *Persona) MarshalTools() error {
	if len(p.Tools) == 0 {
		p.ToolsJSON = "[]"
		return nil
	}
	data, err := json.Marshal(p.Tools)
	if err != nil {
		return err
	}
	p.ToolsJSON = string(data)
	return nil
}

// UnmarshalTools parses tools from JSON storage.
func (p *Persona) UnmarshalTools() error {
	if p.ToolsJSON == "" || p.ToolsJSON == "[]" {
		p.Tools = []string{}
		return nil
	}
	return json.Unmarshal([]byte(p.ToolsJSON), &p.Tools)
}

// MarshalAll marshals all JSON fields for storage.
func (p *Persona) MarshalAll() error {
	if err := p.MarshalVoiceSettings(); err != nil {
		return err
	}
	if err := p.MarshalBehavior(); err != nil {
		return err
	}
	if err := p.MarshalKnowledgeBases(); err != nil {
		return err
	}
	return p.MarshalTools()
}

// UnmarshalAll unmarshals all JSON fields from storage.
func (p *Persona) UnmarshalAll() error {
	if err := p.UnmarshalVoiceSettings(); err != nil {
		return err
	}
	if err := p.UnmarshalBehavior(); err != nil {
		return err
	}
	if err := p.UnmarshalKnowledgeBases(); err != nil {
		return err
	}
	return p.UnmarshalTools()
}

// Preset personas for common use cases

// QuoteAgentPersona creates a persona optimized for collecting software project quotes.
func QuoteAgentPersona() *Persona {
	persona := NewPersona("Project Consultant", "Friendly agent for collecting software project requirements")
	persona.Voice = "maya"
	persona.Personality = "friendly, patient, professional, knowledgeable"
	persona.BackgroundStory = "You are an experienced software project consultant who helps clients define their project needs and get accurate quotes."
	persona.SystemPrompt = `You are a helpful project consultant. Your primary goals are:
1. Warmly greet the caller
2. Understand what type of software project they need
3. Collect key requirements (features, timeline, budget)
4. Be patient and answer any questions they have
5. Thank them for their time and explain next steps

Always be friendly, professional, and thorough. Ask one question at a time.`
	persona.Behavior = &PersonaBehavior{
		ResponseStyle:         "conversational",
		Formality:             "professional",
		Enthusiasm:            0.7,
		InterruptionThreshold: 50,
		WaitForGreeting:       true,
		AllowInterruptions:    true,
		Model:                 "enhanced",
		Temperature:           0.7,
		EmpathyLevel:          0.8,
		PatienceLevel:         0.9,
		MaxCallDuration:       600,
	}
	return persona
}

// SupportAgentPersona creates a persona for customer support calls.
func SupportAgentPersona() *Persona {
	persona := NewPersona("Support Agent", "Professional agent for handling customer support inquiries")
	persona.Voice = "matt"
	persona.Personality = "helpful, empathetic, solution-oriented"
	persona.BackgroundStory = "You are a skilled customer support specialist with years of experience resolving customer issues efficiently and compassionately."
	persona.SystemPrompt = `You are a customer support specialist. Your primary goals are:
1. Listen carefully to the customer's issue
2. Show empathy and understanding
3. Use available resources to find solutions
4. Provide clear, helpful guidance
5. Escalate when necessary

Always remain calm, professional, and focused on resolving the customer's issue.`
	persona.Behavior = &PersonaBehavior{
		ResponseStyle:         "detailed",
		Formality:             "professional",
		Enthusiasm:            0.6,
		InterruptionThreshold: 40,
		WaitForGreeting:       true,
		AllowInterruptions:    true,
		Model:                 "enhanced",
		Temperature:           0.5,
		EmpathyLevel:          0.9,
		PatienceLevel:         0.95,
		MaxCallDuration:       900,
	}
	return persona
}

// AppointmentAgentPersona creates a persona for scheduling appointments.
func AppointmentAgentPersona() *Persona {
	persona := NewPersona("Appointment Scheduler", "Efficient agent for booking appointments")
	persona.Voice = "evelyn"
	persona.Personality = "efficient, organized, friendly"
	persona.BackgroundStory = "You are an experienced scheduling coordinator who helps customers book appointments quickly and accurately."
	persona.SystemPrompt = `You are an appointment scheduling assistant. Your primary goals are:
1. Understand what type of appointment the customer needs
2. Check available time slots
3. Book the appointment efficiently
4. Confirm all details
5. Send confirmation information

Be efficient but friendly. Make the booking process as smooth as possible.`
	persona.Behavior = &PersonaBehavior{
		ResponseStyle:         "concise",
		Formality:             "professional",
		Enthusiasm:            0.6,
		InterruptionThreshold: 60,
		WaitForGreeting:       true,
		AllowInterruptions:    true,
		Model:                 "enhanced",
		Temperature:           0.3,
		EmpathyLevel:          0.6,
		PatienceLevel:         0.7,
		MaxCallDuration:       300,
	}
	return persona
}
