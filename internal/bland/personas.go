package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Persona represents a reusable AI agent configuration in Bland.
type Persona struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`

	// Agent configuration
	Prompt             string                 `json:"prompt,omitempty"`
	Voice              string                 `json:"voice,omitempty"`
	Language           string                 `json:"language,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Temperature        float64                `json:"temperature,omitempty"`
	FirstSentence      string                 `json:"first_sentence,omitempty"`
	WaitForGreeting    bool                   `json:"wait_for_greeting,omitempty"`
	InterruptThreshold int                    `json:"interruption_threshold,omitempty"`

	// Call settings
	MaxDuration           int                    `json:"max_duration,omitempty"`
	Record                bool                   `json:"record,omitempty"`
	BackgroundTrack       string                 `json:"background_track,omitempty"`
	NoiseCancellation     bool                   `json:"noise_cancellation,omitempty"`

	// Transfer settings
	TransferPhoneNumber string            `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string `json:"transfer_list,omitempty"`

	// Voicemail
	VoicemailAction  string `json:"voicemail_action,omitempty"`
	VoicemailMessage string `json:"voicemail_message,omitempty"`

	// Tools and knowledge
	Tools            []string               `json:"tools,omitempty"`
	KnowledgeBaseIDs []string               `json:"knowledge_base_ids,omitempty"`

	// Analysis
	SummaryPrompt     string                 `json:"summary_prompt,omitempty"`
	CitationSchemaIDs []string               `json:"citation_schema_ids,omitempty"`
	Dispositions      []string               `json:"dispositions,omitempty"`

	// Metadata
	Metadata         map[string]interface{} `json:"metadata,omitempty"`

	// Version control
	Version          int       `json:"version,omitempty"`
	IsProduction     bool      `json:"is_production,omitempty"`
	IsDraft          bool      `json:"is_draft,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// CreatePersonaRequest contains parameters for creating a persona.
type CreatePersonaRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Agent configuration
	Prompt             string  `json:"prompt,omitempty"`
	Voice              string  `json:"voice,omitempty"`
	Language           string  `json:"language,omitempty"`
	Model              string  `json:"model,omitempty"`
	Temperature        float64 `json:"temperature,omitempty"`
	FirstSentence      string  `json:"first_sentence,omitempty"`
	WaitForGreeting    bool    `json:"wait_for_greeting,omitempty"`
	InterruptThreshold int     `json:"interruption_threshold,omitempty"`

	// Call settings
	MaxDuration       int    `json:"max_duration,omitempty"`
	Record            bool   `json:"record,omitempty"`
	BackgroundTrack   string `json:"background_track,omitempty"`
	NoiseCancellation bool   `json:"noise_cancellation,omitempty"`

	// Transfer
	TransferPhoneNumber string            `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string `json:"transfer_list,omitempty"`

	// Voicemail
	VoicemailAction  string `json:"voicemail_action,omitempty"`
	VoicemailMessage string `json:"voicemail_message,omitempty"`

	// Tools
	Tools            []string `json:"tools,omitempty"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`

	// Analysis
	SummaryPrompt     string   `json:"summary_prompt,omitempty"`
	CitationSchemaIDs []string `json:"citation_schema_ids,omitempty"`
	Dispositions      []string `json:"dispositions,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdatePersonaRequest contains parameters for updating a persona.
type UpdatePersonaRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	Prompt             *string  `json:"prompt,omitempty"`
	Voice              *string  `json:"voice,omitempty"`
	Language           *string  `json:"language,omitempty"`
	Model              *string  `json:"model,omitempty"`
	Temperature        *float64 `json:"temperature,omitempty"`
	FirstSentence      *string  `json:"first_sentence,omitempty"`
	WaitForGreeting    *bool    `json:"wait_for_greeting,omitempty"`
	InterruptThreshold *int     `json:"interruption_threshold,omitempty"`

	MaxDuration       *int    `json:"max_duration,omitempty"`
	Record            *bool   `json:"record,omitempty"`
	BackgroundTrack   *string `json:"background_track,omitempty"`
	NoiseCancellation *bool   `json:"noise_cancellation,omitempty"`

	TransferPhoneNumber *string           `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string `json:"transfer_list,omitempty"`

	VoicemailAction  *string `json:"voicemail_action,omitempty"`
	VoicemailMessage *string `json:"voicemail_message,omitempty"`

	Tools            []string `json:"tools,omitempty"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`

	SummaryPrompt     *string  `json:"summary_prompt,omitempty"`
	CitationSchemaIDs []string `json:"citation_schema_ids,omitempty"`
	Dispositions      []string `json:"dispositions,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListPersonasResponse contains the response from listing personas.
type ListPersonasResponse struct {
	Personas []Persona `json:"personas"`
}

// ListPersonas retrieves all personas.
func (c *Client) ListPersonas(ctx context.Context) ([]Persona, error) {
	var resp ListPersonasResponse
	if err := c.request(ctx, "GET", "/personas", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Personas, nil
}

// GetPersona retrieves a specific persona by ID.
func (c *Client) GetPersona(ctx context.Context, personaID string) (*Persona, error) {
	if personaID == "" {
		return nil, fmt.Errorf("persona_id is required")
	}

	var persona Persona
	if err := c.request(ctx, "GET", "/personas/"+personaID, nil, &persona); err != nil {
		return nil, err
	}

	return &persona, nil
}

// CreatePersona creates a new persona.
func (c *Client) CreatePersona(ctx context.Context, req *CreatePersonaRequest) (*Persona, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	var persona Persona
	if err := c.request(ctx, "POST", "/personas", req, &persona); err != nil {
		return nil, err
	}

	c.logger.Info("persona created",
		zap.String("id", persona.ID),
		zap.String("name", persona.Name),
	)

	return &persona, nil
}

// UpdatePersona updates an existing persona.
func (c *Client) UpdatePersona(ctx context.Context, personaID string, req *UpdatePersonaRequest) (*Persona, error) {
	if personaID == "" {
		return nil, fmt.Errorf("persona_id is required")
	}

	var persona Persona
	if err := c.request(ctx, "PATCH", "/personas/"+personaID, req, &persona); err != nil {
		return nil, err
	}

	c.logger.Info("persona updated", zap.String("id", personaID))

	return &persona, nil
}

// DeletePersona deletes a persona.
func (c *Client) DeletePersona(ctx context.Context, personaID string) error {
	if personaID == "" {
		return fmt.Errorf("persona_id is required")
	}

	if err := c.request(ctx, "DELETE", "/personas/"+personaID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("persona deleted", zap.String("id", personaID))
	return nil
}

// PromotePersona promotes a persona draft to production.
func (c *Client) PromotePersona(ctx context.Context, personaID string) error {
	if personaID == "" {
		return fmt.Errorf("persona_id is required")
	}

	if err := c.request(ctx, "POST", "/personas/"+personaID+"/promote", nil, nil); err != nil {
		return err
	}

	c.logger.Info("persona promoted to production", zap.String("id", personaID))
	return nil
}

// GetPersonaVersions retrieves version history for a persona.
func (c *Client) GetPersonaVersions(ctx context.Context, personaID string) ([]Persona, error) {
	if personaID == "" {
		return nil, fmt.Errorf("persona_id is required")
	}

	var resp struct {
		Versions []Persona `json:"versions"`
	}
	if err := c.request(ctx, "GET", "/personas/"+personaID+"/versions", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Versions, nil
}
