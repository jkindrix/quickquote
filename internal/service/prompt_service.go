package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
)

// PromptService handles prompt management business logic.
type PromptService struct {
	promptRepo domain.PromptRepository
	logger     *zap.Logger
}

// NewPromptService creates a new PromptService.
func NewPromptService(promptRepo domain.PromptRepository, logger *zap.Logger) *PromptService {
	return &PromptService{
		promptRepo: promptRepo,
		logger:     logger,
	}
}

// CreatePromptRequest contains parameters for creating a prompt.
type CreatePromptRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Task        string `json:"task"`

	// Voice settings
	Voice    string `json:"voice,omitempty"`
	Language string `json:"language,omitempty"`

	// Model settings
	Model                 string   `json:"model,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	InterruptionThreshold *int     `json:"interruption_threshold,omitempty"`
	MaxDuration           *int     `json:"max_duration,omitempty"`

	// Opening behavior
	FirstSentence   string `json:"first_sentence,omitempty"`
	WaitForGreeting bool   `json:"wait_for_greeting,omitempty"`

	// Transfer settings
	TransferPhoneNumber string            `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string `json:"transfer_list,omitempty"`

	// Voicemail
	VoicemailAction  string `json:"voicemail_action,omitempty"`
	VoicemailMessage string `json:"voicemail_message,omitempty"`

	// Recording
	Record            bool    `json:"record,omitempty"`
	BackgroundTrack   *string `json:"background_track,omitempty"`
	NoiseCancellation bool    `json:"noise_cancellation,omitempty"`

	// Knowledge and tools
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
	CustomToolIDs    []string `json:"custom_tool_ids,omitempty"`

	// Analysis
	SummaryPrompt string   `json:"summary_prompt,omitempty"`
	Dispositions  []string `json:"dispositions,omitempty"`

	// Organization
	IsDefault bool `json:"is_default,omitempty"`
}

// UpdatePromptRequest contains parameters for updating a prompt.
type UpdatePromptRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Task        *string `json:"task,omitempty"`

	Voice    *string `json:"voice,omitempty"`
	Language *string `json:"language,omitempty"`

	Model                 *string  `json:"model,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	InterruptionThreshold *int     `json:"interruption_threshold,omitempty"`
	MaxDuration           *int     `json:"max_duration,omitempty"`

	FirstSentence   *string `json:"first_sentence,omitempty"`
	WaitForGreeting *bool   `json:"wait_for_greeting,omitempty"`

	TransferPhoneNumber *string            `json:"transfer_phone_number,omitempty"`
	TransferList        map[string]string  `json:"transfer_list,omitempty"`

	VoicemailAction  *string `json:"voicemail_action,omitempty"`
	VoicemailMessage *string `json:"voicemail_message,omitempty"`

	Record            *bool   `json:"record,omitempty"`
	BackgroundTrack   *string `json:"background_track,omitempty"`
	NoiseCancellation *bool   `json:"noise_cancellation,omitempty"`

	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
	CustomToolIDs    []string `json:"custom_tool_ids,omitempty"`

	SummaryPrompt *string  `json:"summary_prompt,omitempty"`
	Dispositions  []string `json:"dispositions,omitempty"`

	IsDefault *bool `json:"is_default,omitempty"`
	IsActive  *bool `json:"is_active,omitempty"`
}

// CreatePrompt creates a new prompt.
func (s *PromptService) CreatePrompt(ctx context.Context, req *CreatePromptRequest) (*domain.Prompt, error) {
	prompt := domain.NewPrompt(req.Name, req.Task)

	// Apply optional fields
	if req.Description != "" {
		prompt.Description = req.Description
	}
	if req.Voice != "" {
		prompt.Voice = req.Voice
	}
	if req.Language != "" {
		prompt.Language = req.Language
	}
	if req.Model != "" {
		prompt.Model = req.Model
	}
	if req.Temperature != nil {
		prompt.Temperature = req.Temperature
	}
	if req.InterruptionThreshold != nil {
		prompt.InterruptionThreshold = req.InterruptionThreshold
	}
	if req.MaxDuration != nil {
		prompt.MaxDuration = req.MaxDuration
	}
	if req.FirstSentence != "" {
		prompt.FirstSentence = req.FirstSentence
	}
	prompt.WaitForGreeting = req.WaitForGreeting
	if req.TransferPhoneNumber != "" {
		prompt.TransferPhoneNumber = req.TransferPhoneNumber
	}
	if req.TransferList != nil {
		prompt.TransferList = req.TransferList
	}
	if req.VoicemailAction != "" {
		prompt.VoicemailAction = req.VoicemailAction
	}
	if req.VoicemailMessage != "" {
		prompt.VoicemailMessage = req.VoicemailMessage
	}
	prompt.Record = req.Record
	if req.BackgroundTrack != nil {
		prompt.BackgroundTrack = req.BackgroundTrack
	}
	prompt.NoiseCancellation = req.NoiseCancellation
	if req.KnowledgeBaseIDs != nil {
		prompt.KnowledgeBaseIDs = req.KnowledgeBaseIDs
	}
	if req.CustomToolIDs != nil {
		prompt.CustomToolIDs = req.CustomToolIDs
	}
	if req.SummaryPrompt != "" {
		prompt.SummaryPrompt = req.SummaryPrompt
	}
	if req.Dispositions != nil {
		prompt.Dispositions = req.Dispositions
	}
	prompt.IsDefault = req.IsDefault

	// Validate
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	// Create in database
	if err := s.promptRepo.Create(ctx, prompt); err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	// If this is set as default, update default status
	if prompt.IsDefault {
		if err := s.promptRepo.SetDefault(ctx, prompt.ID); err != nil {
			s.logger.Warn("failed to set prompt as default", zap.Error(err))
		}
	}

	s.logger.Info("prompt created",
		zap.String("id", prompt.ID.String()),
		zap.String("name", prompt.Name),
	)

	return prompt, nil
}

// GetPrompt retrieves a prompt by ID.
func (s *PromptService) GetPrompt(ctx context.Context, id uuid.UUID) (*domain.Prompt, error) {
	return s.promptRepo.GetByID(ctx, id)
}

// GetPromptByName retrieves a prompt by name.
func (s *PromptService) GetPromptByName(ctx context.Context, name string) (*domain.Prompt, error) {
	return s.promptRepo.GetByName(ctx, name)
}

// GetDefaultPrompt retrieves the default prompt.
func (s *PromptService) GetDefaultPrompt(ctx context.Context) (*domain.Prompt, error) {
	return s.promptRepo.GetDefault(ctx)
}

// ListPrompts retrieves prompts with pagination.
func (s *PromptService) ListPrompts(ctx context.Context, page, pageSize int, activeOnly bool) ([]*domain.Prompt, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	prompts, err := s.promptRepo.List(ctx, pageSize, offset, activeOnly)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.promptRepo.Count(ctx, activeOnly)
	if err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}

// UpdatePrompt updates an existing prompt.
func (s *PromptService) UpdatePrompt(ctx context.Context, id uuid.UUID, req *UpdatePromptRequest) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		prompt.Name = *req.Name
	}
	if req.Description != nil {
		prompt.Description = *req.Description
	}
	if req.Task != nil {
		prompt.Task = *req.Task
	}
	if req.Voice != nil {
		prompt.Voice = *req.Voice
	}
	if req.Language != nil {
		prompt.Language = *req.Language
	}
	if req.Model != nil {
		prompt.Model = *req.Model
	}
	if req.Temperature != nil {
		prompt.Temperature = req.Temperature
	}
	if req.InterruptionThreshold != nil {
		prompt.InterruptionThreshold = req.InterruptionThreshold
	}
	if req.MaxDuration != nil {
		prompt.MaxDuration = req.MaxDuration
	}
	if req.FirstSentence != nil {
		prompt.FirstSentence = *req.FirstSentence
	}
	if req.WaitForGreeting != nil {
		prompt.WaitForGreeting = *req.WaitForGreeting
	}
	if req.TransferPhoneNumber != nil {
		prompt.TransferPhoneNumber = *req.TransferPhoneNumber
	}
	if req.TransferList != nil {
		prompt.TransferList = req.TransferList
	}
	if req.VoicemailAction != nil {
		prompt.VoicemailAction = *req.VoicemailAction
	}
	if req.VoicemailMessage != nil {
		prompt.VoicemailMessage = *req.VoicemailMessage
	}
	if req.Record != nil {
		prompt.Record = *req.Record
	}
	if req.BackgroundTrack != nil {
		prompt.BackgroundTrack = req.BackgroundTrack
	}
	if req.NoiseCancellation != nil {
		prompt.NoiseCancellation = *req.NoiseCancellation
	}
	if req.KnowledgeBaseIDs != nil {
		prompt.KnowledgeBaseIDs = req.KnowledgeBaseIDs
	}
	if req.CustomToolIDs != nil {
		prompt.CustomToolIDs = req.CustomToolIDs
	}
	if req.SummaryPrompt != nil {
		prompt.SummaryPrompt = *req.SummaryPrompt
	}
	if req.Dispositions != nil {
		prompt.Dispositions = req.Dispositions
	}
	if req.IsActive != nil {
		prompt.IsActive = *req.IsActive
	}

	// Handle default status change
	if req.IsDefault != nil && *req.IsDefault && !prompt.IsDefault {
		if err := s.promptRepo.SetDefault(ctx, prompt.ID); err != nil {
			return nil, fmt.Errorf("failed to set as default: %w", err)
		}
		prompt.IsDefault = true
	}

	// Validate
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	// Update in database
	if err := s.promptRepo.Update(ctx, prompt); err != nil {
		return nil, fmt.Errorf("failed to update prompt: %w", err)
	}

	s.logger.Info("prompt updated",
		zap.String("id", prompt.ID.String()),
		zap.String("name", prompt.Name),
	)

	return prompt, nil
}

// DeletePrompt soft-deletes a prompt.
func (s *PromptService) DeletePrompt(ctx context.Context, id uuid.UUID) error {
	if err := s.promptRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	s.logger.Info("prompt deleted", zap.String("id", id.String()))
	return nil
}

// SetDefaultPrompt sets a prompt as the default.
func (s *PromptService) SetDefaultPrompt(ctx context.Context, id uuid.UUID) error {
	if err := s.promptRepo.SetDefault(ctx, id); err != nil {
		return fmt.Errorf("failed to set default prompt: %w", err)
	}

	s.logger.Info("default prompt set", zap.String("id", id.String()))
	return nil
}

// DuplicatePrompt creates a copy of an existing prompt.
func (s *PromptService) DuplicatePrompt(ctx context.Context, id uuid.UUID, newName string) (*domain.Prompt, error) {
	original, err := s.promptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create a copy with a new ID and name
	copy := *original
	copy.ID = uuid.New()
	copy.Name = newName
	copy.IsDefault = false
	copy.CreatedAt = copy.UpdatedAt

	if err := s.promptRepo.Create(ctx, &copy); err != nil {
		return nil, fmt.Errorf("failed to duplicate prompt: %w", err)
	}

	s.logger.Info("prompt duplicated",
		zap.String("original_id", id.String()),
		zap.String("new_id", copy.ID.String()),
		zap.String("new_name", newName),
	)

	return &copy, nil
}
