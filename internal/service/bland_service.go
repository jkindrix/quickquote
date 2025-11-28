// Package service contains business logic implementations.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/domain"
)

// BlandService handles voice call initiation and management via Bland AI.
type BlandService struct {
	blandClient     *bland.Client
	callRepo        domain.CallRepository
	promptRepo      domain.PromptRepository
	settingsService *SettingsService
	webhookURL      string
	logger          *zap.Logger
}

// NewBlandService creates a new BlandService.
func NewBlandService(
	blandClient *bland.Client,
	callRepo domain.CallRepository,
	promptRepo domain.PromptRepository,
	webhookURL string,
	logger *zap.Logger,
) *BlandService {
	return &BlandService{
		blandClient: blandClient,
		callRepo:    callRepo,
		promptRepo:  promptRepo,
		webhookURL:  webhookURL,
		logger:      logger,
	}
}

// SetSettingsService sets the settings service for retrieving call configuration.
func (s *BlandService) SetSettingsService(ss *SettingsService) {
	s.settingsService = ss
}

// InitiateCallRequest contains parameters for initiating a call.
type InitiateCallRequest struct {
	// Required: Phone number to call (E.164 format)
	PhoneNumber string `json:"phone_number"`

	// PromptID: Use a saved prompt (optional if Task provided)
	PromptID *uuid.UUID `json:"prompt_id,omitempty"`

	// Task: Direct prompt text (optional if PromptID provided)
	Task string `json:"task,omitempty"`

	// Voice: Override prompt's voice setting
	Voice string `json:"voice,omitempty"`

	// FirstSentence: Override prompt's opening
	FirstSentence string `json:"first_sentence,omitempty"`

	// RequestData: Custom variables for the call (accessible as {{variable}})
	RequestData map[string]interface{} `json:"request_data,omitempty"`

	// Metadata: Custom tracking data
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// PathwayID: Use a conversation pathway instead of task
	PathwayID string `json:"pathway_id,omitempty"`

	// PersonaID: Use a Bland persona
	PersonaID string `json:"persona_id,omitempty"`

	// MaxDuration: Override max call duration (minutes)
	MaxDuration *int `json:"max_duration,omitempty"`

	// Record: Override recording setting
	Record *bool `json:"record,omitempty"`

	// ScheduledTime: Schedule call for later (RFC3339 format)
	ScheduledTime string `json:"scheduled_time,omitempty"`
}

// InitiateCallResponse contains the result of initiating a call.
type InitiateCallResponse struct {
	CallID         uuid.UUID `json:"call_id"`
	BlandCallID    string    `json:"bland_call_id"`
	Status         string    `json:"status"`
	PhoneNumber    string    `json:"phone_number"`
	PromptID       *uuid.UUID `json:"prompt_id,omitempty"`
	PromptName     string    `json:"prompt_name,omitempty"`
}

// InitiateCall starts a new outbound call via Bland AI.
func (s *BlandService) InitiateCall(ctx context.Context, req *InitiateCallRequest) (*InitiateCallResponse, error) {
	// Validate request
	if req.PhoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}

	// Build the Bland API request
	blandReq, prompt, err := s.buildBlandRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Set webhook URL
	blandReq.Webhook = s.webhookURL

	// Log the parameters we're sending (for debugging)
	paramsJSON, _ := json.Marshal(blandReq)
	s.logger.Info("initiating call",
		zap.String("phone_number", req.PhoneNumber),
		zap.String("webhook", blandReq.Webhook),
	)

	// Send the call via Bland API
	blandResp, err := s.blandClient.SendCall(ctx, blandReq)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate call: %w", err)
	}

	// Create call record in our database
	call := &domain.Call{
		ID:             uuid.New(),
		ProviderCallID: blandResp.CallID,
		Provider:       "bland",
		PhoneNumber:    req.PhoneNumber,
		Status:         domain.CallStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Store prompt reference if used
	var promptID *uuid.UUID
	var promptName string
	if prompt != nil {
		promptID = &prompt.ID
		promptName = prompt.Name
	}

	// Create the call record
	if err := s.callRepo.Create(ctx, call); err != nil {
		s.logger.Error("failed to create call record",
			zap.String("bland_call_id", blandResp.CallID),
			zap.Error(err),
		)
		// Don't fail - the call was already initiated
	}

	// Store the full parameters in call metadata for debugging
	s.logger.Debug("call initiated successfully",
		zap.String("call_id", call.ID.String()),
		zap.String("bland_call_id", blandResp.CallID),
		zap.ByteString("params", paramsJSON),
	)

	return &InitiateCallResponse{
		CallID:      call.ID,
		BlandCallID: blandResp.CallID,
		Status:      blandResp.Status,
		PhoneNumber: req.PhoneNumber,
		PromptID:    promptID,
		PromptName:  promptName,
	}, nil
}

// buildBlandRequest constructs the Bland API request from our request.
func (s *BlandService) buildBlandRequest(ctx context.Context, req *InitiateCallRequest) (*bland.SendCallRequest, *domain.Prompt, error) {
	blandReq := &bland.SendCallRequest{
		PhoneNumber: req.PhoneNumber,
		RequestData: req.RequestData,
		Metadata:    req.Metadata,
	}

	var prompt *domain.Prompt

	// Load prompt if ID provided
	if req.PromptID != nil {
		var err error
		prompt, err = s.promptRepo.GetByID(ctx, *req.PromptID)
		if err != nil {
			return nil, nil, fmt.Errorf("prompt not found: %w", err)
		}

		// Apply prompt settings
		s.applyPromptToRequest(blandReq, prompt)
	}

	// Use default prompt if no task, pathway, or persona specified
	if req.Task == "" && req.PathwayID == "" && req.PersonaID == "" && prompt == nil {
		var err error
		prompt, err = s.promptRepo.GetDefault(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("no default prompt configured and no task provided: %w", err)
		}
		s.applyPromptToRequest(blandReq, prompt)
	}

	// Override with direct request parameters
	if req.Task != "" {
		blandReq.Task = req.Task
	}
	if req.PathwayID != "" {
		blandReq.PathwayID = req.PathwayID
	}
	if req.PersonaID != "" {
		blandReq.PersonaID = req.PersonaID
	}
	if req.Voice != "" {
		blandReq.Voice = req.Voice
	}
	if req.FirstSentence != "" {
		blandReq.FirstSentence = req.FirstSentence
	}
	if req.MaxDuration != nil {
		blandReq.MaxDuration = req.MaxDuration
	}
	if req.Record != nil {
		blandReq.Record = *req.Record
	}
	if req.ScheduledTime != "" {
		blandReq.StartTime = req.ScheduledTime
	}

	return blandReq, prompt, nil
}

// applyPromptToRequest applies a prompt's settings to a Bland request.
func (s *BlandService) applyPromptToRequest(req *bland.SendCallRequest, prompt *domain.Prompt) {
	req.Task = prompt.Task
	req.Voice = prompt.Voice
	req.Language = prompt.Language
	req.Model = prompt.Model
	req.Temperature = prompt.Temperature
	req.InterruptionThreshold = prompt.InterruptionThreshold
	req.MaxDuration = prompt.MaxDuration
	req.FirstSentence = prompt.FirstSentence
	req.WaitForGreeting = prompt.WaitForGreeting
	req.TransferPhoneNumber = prompt.TransferPhoneNumber
	req.TransferList = prompt.TransferList
	req.Record = prompt.Record
	req.BackgroundTrack = prompt.BackgroundTrack
	req.NoiseCancellation = prompt.NoiseCancellation
	req.SummaryPrompt = prompt.SummaryPrompt
	req.Dispositions = prompt.Dispositions
	req.Tools = append(prompt.KnowledgeBaseIDs, prompt.CustomToolIDs...)

	// Configure voicemail if specified
	if prompt.VoicemailAction != "" {
		req.Voicemail = &bland.VoicemailConfig{
			Action:  prompt.VoicemailAction,
			Message: prompt.VoicemailMessage,
		}
	}
}

// GetCallStatus retrieves the current status of a call from Bland.
func (s *BlandService) GetCallStatus(ctx context.Context, blandCallID string) (*bland.CallDetails, error) {
	return s.blandClient.GetCall(ctx, blandCallID)
}

// EndCall terminates an active call.
func (s *BlandService) EndCall(ctx context.Context, blandCallID string) error {
	return s.blandClient.EndCall(ctx, blandCallID)
}

// GetCallTranscript retrieves the transcript for a completed call.
func (s *BlandService) GetCallTranscript(ctx context.Context, blandCallID string) (*bland.TranscriptResponse, error) {
	return s.blandClient.GetCallTranscript(ctx, blandCallID)
}

// AnalyzeCall performs post-call analysis on a completed call.
func (s *BlandService) AnalyzeCall(ctx context.Context, blandCallID string, goal string, questions []string) (*bland.AnalyzeCallResponse, error) {
	return s.blandClient.AnalyzeCall(ctx, blandCallID, &bland.AnalyzeCallRequest{
		Goal:      goal,
		Questions: questions,
	})
}

// GetActiveCalls returns all currently active calls.
func (s *BlandService) GetActiveCalls(ctx context.Context) (*bland.ActiveCallsResponse, error) {
	return s.blandClient.GetActiveCalls(ctx)
}

// CircuitBreakerStats returns the Bland API circuit breaker statistics.
func (s *BlandService) CircuitBreakerStats() interface{} {
	return s.blandClient.CircuitBreakerStats()
}

// ===============================================
// Voice Management
// ===============================================

// ListVoices returns all available voices.
func (s *BlandService) ListVoices(ctx context.Context) ([]bland.Voice, error) {
	return s.blandClient.ListVoices(ctx)
}

// GetVoice retrieves details for a specific voice.
func (s *BlandService) GetVoice(ctx context.Context, voiceID string) (*bland.Voice, error) {
	return s.blandClient.GetVoice(ctx, voiceID)
}

// CloneVoice creates a new voice from audio samples.
func (s *BlandService) CloneVoice(ctx context.Context, req *bland.CloneVoiceRequest) (*bland.CloneVoiceResponse, error) {
	return s.blandClient.CloneVoice(ctx, req)
}

// GenerateVoiceSample generates an audio sample with a voice.
func (s *BlandService) GenerateVoiceSample(ctx context.Context, voiceID string, req *bland.GenerateSampleRequest) (*bland.GenerateSampleResponse, error) {
	return s.blandClient.GenerateVoiceSample(ctx, voiceID, req)
}

// DeleteVoice removes a custom voice.
func (s *BlandService) DeleteVoice(ctx context.Context, voiceID string) error {
	return s.blandClient.DeleteVoice(ctx, voiceID)
}

// ===============================================
// Persona Management
// ===============================================

// ListPersonas returns all personas.
func (s *BlandService) ListPersonas(ctx context.Context) ([]bland.Persona, error) {
	return s.blandClient.ListPersonas(ctx)
}

// GetPersona retrieves a specific persona.
func (s *BlandService) GetPersona(ctx context.Context, personaID string) (*bland.Persona, error) {
	return s.blandClient.GetPersona(ctx, personaID)
}

// CreatePersona creates a new persona.
func (s *BlandService) CreatePersona(ctx context.Context, req *bland.CreatePersonaRequest) (*bland.Persona, error) {
	return s.blandClient.CreatePersona(ctx, req)
}

// UpdatePersona updates an existing persona.
func (s *BlandService) UpdatePersona(ctx context.Context, personaID string, req *bland.UpdatePersonaRequest) (*bland.Persona, error) {
	return s.blandClient.UpdatePersona(ctx, personaID, req)
}

// DeletePersona removes a persona.
func (s *BlandService) DeletePersona(ctx context.Context, personaID string) error {
	return s.blandClient.DeletePersona(ctx, personaID)
}

// ===============================================
// Knowledge Base Management
// ===============================================

// ListKnowledgeBases returns all knowledge bases.
func (s *BlandService) ListKnowledgeBases(ctx context.Context) ([]bland.KnowledgeBase, error) {
	return s.blandClient.ListKnowledgeBases(ctx)
}

// GetKnowledgeBase retrieves a specific knowledge base.
func (s *BlandService) GetKnowledgeBase(ctx context.Context, vectorID string) (*bland.KnowledgeBase, error) {
	return s.blandClient.GetKnowledgeBase(ctx, vectorID)
}

// CreateKnowledgeBase creates a new knowledge base from text.
func (s *BlandService) CreateKnowledgeBase(ctx context.Context, req *bland.CreateKnowledgeBaseRequest) (*bland.CreateKnowledgeBaseResponse, error) {
	return s.blandClient.CreateKnowledgeBase(ctx, req)
}

// UpdateKnowledgeBase updates an existing knowledge base.
func (s *BlandService) UpdateKnowledgeBase(ctx context.Context, vectorID string, req *bland.UpdateKnowledgeBaseRequest) error {
	return s.blandClient.UpdateKnowledgeBase(ctx, vectorID, req)
}

// DeleteKnowledgeBase removes a knowledge base.
func (s *BlandService) DeleteKnowledgeBase(ctx context.Context, vectorID string) error {
	return s.blandClient.DeleteKnowledgeBase(ctx, vectorID)
}

// ===============================================
// Pathway Management
// ===============================================

// ListPathways returns all pathways.
func (s *BlandService) ListPathways(ctx context.Context) ([]bland.Pathway, error) {
	return s.blandClient.ListPathways(ctx)
}

// GetPathway retrieves a specific pathway.
func (s *BlandService) GetPathway(ctx context.Context, pathwayID string) (*bland.Pathway, error) {
	return s.blandClient.GetPathway(ctx, pathwayID)
}

// CreatePathway creates a new pathway.
func (s *BlandService) CreatePathway(ctx context.Context, req *bland.CreatePathwayRequest) (*bland.Pathway, error) {
	return s.blandClient.CreatePathway(ctx, req)
}

// UpdatePathway updates an existing pathway.
func (s *BlandService) UpdatePathway(ctx context.Context, pathwayID string, req *bland.UpdatePathwayRequest) (*bland.Pathway, error) {
	return s.blandClient.UpdatePathway(ctx, pathwayID, req)
}

// DeletePathway removes a pathway.
func (s *BlandService) DeletePathway(ctx context.Context, pathwayID string) error {
	return s.blandClient.DeletePathway(ctx, pathwayID)
}

// PublishPathway publishes a pathway to production.
func (s *BlandService) PublishPathway(ctx context.Context, pathwayID string) error {
	return s.blandClient.PublishPathway(ctx, pathwayID)
}

// ===============================================
// Customer Memory Management
// ===============================================

// GetCustomerMemory retrieves stored context for a phone number.
func (s *BlandService) GetCustomerMemory(ctx context.Context, phoneNumber string) (map[string]interface{}, error) {
	return s.blandClient.GetCustomerContext(ctx, phoneNumber)
}

// StoreCustomerMemory saves context for a phone number.
func (s *BlandService) StoreCustomerMemory(ctx context.Context, phoneNumber string, data map[string]interface{}) error {
	return s.blandClient.RememberCustomer(ctx, phoneNumber, data)
}

// ClearCustomerMemory removes all stored context for a phone number.
func (s *BlandService) ClearCustomerMemory(ctx context.Context, phoneNumber string) error {
	return s.blandClient.ClearCustomerMemory(ctx, phoneNumber)
}

// StoreQuoteContext saves quote-related context for follow-up calls.
func (s *BlandService) StoreQuoteContext(ctx context.Context, phoneNumber string, quoteData map[string]interface{}) error {
	return s.blandClient.StoreQuoteContext(ctx, phoneNumber, quoteData)
}

// ===============================================
// Batch Call Management
// ===============================================

// CreateBatch creates a batch of calls.
func (s *BlandService) CreateBatch(ctx context.Context, req *bland.CreateBatchRequest) (*bland.CreateBatchResponse, error) {
	// Add webhook URL if not specified
	if req.WebhookURL == "" {
		req.WebhookURL = s.webhookURL
	}
	return s.blandClient.CreateBatch(ctx, req)
}

// GetBatch retrieves batch details.
func (s *BlandService) GetBatch(ctx context.Context, batchID string) (*bland.Batch, error) {
	return s.blandClient.GetBatch(ctx, batchID)
}

// ListBatches returns all batches.
func (s *BlandService) ListBatches(ctx context.Context, limit, offset int) (*bland.ListBatchesResponse, error) {
	return s.blandClient.ListBatches(ctx, limit, offset)
}

// PauseBatch pauses a running batch.
func (s *BlandService) PauseBatch(ctx context.Context, batchID string) error {
	return s.blandClient.PauseBatch(ctx, batchID)
}

// ResumeBatch resumes a paused batch.
func (s *BlandService) ResumeBatch(ctx context.Context, batchID string) error {
	return s.blandClient.ResumeBatch(ctx, batchID)
}

// CancelBatch cancels a batch.
func (s *BlandService) CancelBatch(ctx context.Context, batchID string) error {
	return s.blandClient.CancelBatch(ctx, batchID)
}

// GetBatchAnalytics retrieves analytics for a batch.
func (s *BlandService) GetBatchAnalytics(ctx context.Context, batchID string) (*bland.BatchAnalytics, error) {
	return s.blandClient.GetBatchAnalytics(ctx, batchID)
}

// ===============================================
// SMS Management
// ===============================================

// SendSMS sends an SMS message.
func (s *BlandService) SendSMS(ctx context.Context, req *bland.SendSMSRequest) (*bland.SendSMSResponse, error) {
	return s.blandClient.SendSMS(ctx, req)
}

// StartSMSConversation starts an AI-powered SMS conversation.
func (s *BlandService) StartSMSConversation(ctx context.Context, req *bland.StartSMSConversationRequest) (*bland.StartSMSConversationResponse, error) {
	// Add webhook URL if not specified
	if req.WebhookURL == "" {
		req.WebhookURL = s.webhookURL
	}
	return s.blandClient.StartSMSConversation(ctx, req)
}

// GetSMSConversation retrieves an SMS conversation.
func (s *BlandService) GetSMSConversation(ctx context.Context, conversationID string) (*bland.SMSConversation, error) {
	return s.blandClient.GetSMSConversation(ctx, conversationID)
}

// EndSMSConversation ends an active conversation.
func (s *BlandService) EndSMSConversation(ctx context.Context, conversationID string) error {
	return s.blandClient.EndSMSConversation(ctx, conversationID)
}

// SendQuoteReadySMS sends a quote-ready notification.
func (s *BlandService) SendQuoteReadySMS(ctx context.Context, phoneNumber, quoteID string, amount float64) (*bland.SendSMSResponse, error) {
	return s.blandClient.SendQuoteReadySMS(ctx, phoneNumber, quoteID, amount)
}

// ===============================================
// Custom Tools Management
// ===============================================

// ListTools returns all custom tools.
func (s *BlandService) ListTools(ctx context.Context) ([]bland.Tool, error) {
	return s.blandClient.ListTools(ctx)
}

// GetTool retrieves a specific tool.
func (s *BlandService) GetTool(ctx context.Context, toolID string) (*bland.Tool, error) {
	return s.blandClient.GetTool(ctx, toolID)
}

// CreateTool creates a new custom tool.
func (s *BlandService) CreateTool(ctx context.Context, req *bland.CreateToolRequest) (*bland.Tool, error) {
	return s.blandClient.CreateTool(ctx, req)
}

// UpdateTool updates an existing tool.
func (s *BlandService) UpdateTool(ctx context.Context, toolID string, req *bland.UpdateToolRequest) (*bland.Tool, error) {
	return s.blandClient.UpdateTool(ctx, toolID, req)
}

// DeleteTool removes a custom tool.
func (s *BlandService) DeleteTool(ctx context.Context, toolID string) error {
	return s.blandClient.DeleteTool(ctx, toolID)
}

// TestTool tests a tool with sample input.
func (s *BlandService) TestTool(ctx context.Context, toolID string, input map[string]interface{}) (*bland.ToolExecutionLog, error) {
	return s.blandClient.TestTool(ctx, toolID, input)
}

// ===============================================
// Helper Methods for Common Workflows
// ===============================================

// InitiateQuoteFollowUp calls a customer back about their quote.
func (s *BlandService) InitiateQuoteFollowUp(ctx context.Context, phoneNumber, quoteID string, quoteSummary string) (*InitiateCallResponse, error) {
	task := fmt.Sprintf(`You are calling to follow up on quote %s. The quote details are: %s

Be friendly and professional. Answer any questions about the quote. If they want to proceed,
ask when they'd like to schedule the service. Collect any additional information needed.`, quoteID, quoteSummary)

	return s.InitiateCall(ctx, &InitiateCallRequest{
		PhoneNumber: phoneNumber,
		Task:        task,
		Metadata: map[string]interface{}{
			"type":     "quote_followup",
			"quote_id": quoteID,
		},
	})
}

// SetupQuoteLookupTool creates the quote lookup tool in Bland.
func (s *BlandService) SetupQuoteLookupTool(ctx context.Context) (*bland.Tool, error) {
	toolReq := bland.NewQuoteLookupTool(s.webhookURL)
	return s.blandClient.CreateTool(ctx, toolReq)
}

// SetupScheduleCallbackTool creates the schedule callback tool in Bland.
func (s *BlandService) SetupScheduleCallbackTool(ctx context.Context) (*bland.Tool, error) {
	toolReq := bland.NewScheduleCallbackTool(s.webhookURL)
	return s.blandClient.CreateTool(ctx, toolReq)
}

// ===============================================
// Phone Number Management
// ===============================================

// ListPhoneNumbers returns all phone numbers.
func (s *BlandService) ListPhoneNumbers(ctx context.Context, req *bland.ListPhoneNumbersRequest) ([]bland.PhoneNumber, error) {
	return s.blandClient.ListPhoneNumbers(ctx, req)
}

// GetPhoneNumber retrieves a specific phone number.
func (s *BlandService) GetPhoneNumber(ctx context.Context, numberID string) (*bland.PhoneNumber, error) {
	return s.blandClient.GetPhoneNumber(ctx, numberID)
}

// SearchAvailableNumbers searches for available phone numbers.
func (s *BlandService) SearchAvailableNumbers(ctx context.Context, req *bland.SearchAvailableNumbersRequest) ([]bland.AvailablePhoneNumber, error) {
	return s.blandClient.SearchAvailableNumbers(ctx, req)
}

// PurchaseNumber purchases a phone number.
func (s *BlandService) PurchaseNumber(ctx context.Context, req *bland.PurchaseNumberRequest) (*bland.PhoneNumber, error) {
	return s.blandClient.PurchaseNumber(ctx, req)
}

// UpdatePhoneNumber updates a phone number.
func (s *BlandService) UpdatePhoneNumber(ctx context.Context, numberID string, req *bland.UpdatePhoneNumberRequest) (*bland.PhoneNumber, error) {
	return s.blandClient.UpdatePhoneNumber(ctx, numberID, req)
}

// ReleasePhoneNumber releases a phone number.
func (s *BlandService) ReleasePhoneNumber(ctx context.Context, numberID string) error {
	return s.blandClient.ReleasePhoneNumber(ctx, numberID)
}

// ConfigureInboundAgent configures an inbound agent for a phone number.
func (s *BlandService) ConfigureInboundAgent(ctx context.Context, phoneNumberID string, config *bland.InboundConfig) (*bland.PhoneNumber, error) {
	return s.blandClient.ConfigureInboundAgent(ctx, phoneNumberID, config)
}

// ListBlockedNumbers returns all blocked numbers.
func (s *BlandService) ListBlockedNumbers(ctx context.Context) ([]bland.BlockedNumber, error) {
	return s.blandClient.ListBlockedNumbers(ctx)
}

// BlockNumber blocks a phone number.
func (s *BlandService) BlockNumber(ctx context.Context, req *bland.BlockNumberRequest) (*bland.BlockedNumber, error) {
	return s.blandClient.BlockNumber(ctx, req)
}

// UnblockNumber unblocks a phone number.
func (s *BlandService) UnblockNumber(ctx context.Context, blockedID string) error {
	return s.blandClient.UnblockNumber(ctx, blockedID)
}

// ===============================================
// Citation Schema Management
// ===============================================

// ListCitationSchemas returns all citation schemas.
func (s *BlandService) ListCitationSchemas(ctx context.Context) ([]bland.CitationSchema, error) {
	return s.blandClient.ListCitationSchemas(ctx)
}

// GetCitationSchema retrieves a specific citation schema.
func (s *BlandService) GetCitationSchema(ctx context.Context, schemaID string) (*bland.CitationSchema, error) {
	return s.blandClient.GetCitationSchema(ctx, schemaID)
}

// CreateCitationSchema creates a new citation schema.
func (s *BlandService) CreateCitationSchema(ctx context.Context, req *bland.CreateCitationSchemaRequest) (*bland.CitationSchema, error) {
	return s.blandClient.CreateCitationSchema(ctx, req)
}

// UpdateCitationSchema updates a citation schema.
func (s *BlandService) UpdateCitationSchema(ctx context.Context, schemaID string, req *bland.UpdateCitationSchemaRequest) (*bland.CitationSchema, error) {
	return s.blandClient.UpdateCitationSchema(ctx, schemaID, req)
}

// DeleteCitationSchema deletes a citation schema.
func (s *BlandService) DeleteCitationSchema(ctx context.Context, schemaID string) error {
	return s.blandClient.DeleteCitationSchema(ctx, schemaID)
}

// GetCallCitations retrieves citations from a call.
func (s *BlandService) GetCallCitations(ctx context.Context, callID string) ([]bland.CitationResult, error) {
	return s.blandClient.GetCallCitations(ctx, callID)
}

// ExtractCitations extracts citations from a call using specified schemas.
func (s *BlandService) ExtractCitations(ctx context.Context, callID string, schemaIDs []string) ([]bland.CitationResult, error) {
	return s.blandClient.ExtractCitations(ctx, callID, schemaIDs)
}

// ===============================================
// Dynamic Data Source Management
// ===============================================

// ListDynamicDataSources returns all dynamic data sources.
func (s *BlandService) ListDynamicDataSources(ctx context.Context) ([]bland.DynamicDataSource, error) {
	return s.blandClient.ListDynamicDataSources(ctx)
}

// GetDynamicDataSource retrieves a specific dynamic data source.
func (s *BlandService) GetDynamicDataSource(ctx context.Context, sourceID string) (*bland.DynamicDataSource, error) {
	return s.blandClient.GetDynamicDataSource(ctx, sourceID)
}

// CreateDynamicDataSource creates a new dynamic data source.
func (s *BlandService) CreateDynamicDataSource(ctx context.Context, req *bland.CreateDynamicDataSourceRequest) (*bland.DynamicDataSource, error) {
	return s.blandClient.CreateDynamicDataSource(ctx, req)
}

// UpdateDynamicDataSource updates a dynamic data source.
func (s *BlandService) UpdateDynamicDataSource(ctx context.Context, sourceID string, req *bland.UpdateDynamicDataSourceRequest) (*bland.DynamicDataSource, error) {
	return s.blandClient.UpdateDynamicDataSource(ctx, sourceID, req)
}

// DeleteDynamicDataSource deletes a dynamic data source.
func (s *BlandService) DeleteDynamicDataSource(ctx context.Context, sourceID string) error {
	return s.blandClient.DeleteDynamicDataSource(ctx, sourceID)
}

// TestDynamicDataSource tests a dynamic data source.
func (s *BlandService) TestDynamicDataSource(ctx context.Context, sourceID string, params map[string]interface{}) (*bland.DynamicDataTestResult, error) {
	return s.blandClient.TestDynamicDataSource(ctx, sourceID, params)
}

// RefreshDynamicDataSource refreshes a dynamic data source.
func (s *BlandService) RefreshDynamicDataSource(ctx context.Context, sourceID string) error {
	return s.blandClient.RefreshDynamicDataSource(ctx, sourceID)
}

// ===============================================
// Enterprise - Twilio BYOT
// ===============================================

// ListTwilioAccounts returns all Twilio accounts.
func (s *BlandService) ListTwilioAccounts(ctx context.Context) ([]bland.TwilioAccount, error) {
	return s.blandClient.ListTwilioAccounts(ctx)
}

// GetTwilioAccount retrieves a specific Twilio account.
func (s *BlandService) GetTwilioAccount(ctx context.Context, accountID string) (*bland.TwilioAccount, error) {
	return s.blandClient.GetTwilioAccount(ctx, accountID)
}

// CreateTwilioAccount creates a new Twilio account.
func (s *BlandService) CreateTwilioAccount(ctx context.Context, req *bland.CreateTwilioAccountRequest) (*bland.TwilioAccount, error) {
	return s.blandClient.CreateTwilioAccount(ctx, req)
}

// UpdateTwilioAccount updates a Twilio account.
func (s *BlandService) UpdateTwilioAccount(ctx context.Context, accountID string, req *bland.UpdateTwilioAccountRequest) (*bland.TwilioAccount, error) {
	return s.blandClient.UpdateTwilioAccount(ctx, accountID, req)
}

// DeleteTwilioAccount deletes a Twilio account.
func (s *BlandService) DeleteTwilioAccount(ctx context.Context, accountID string) error {
	return s.blandClient.DeleteTwilioAccount(ctx, accountID)
}

// VerifyTwilioAccount verifies a Twilio account.
func (s *BlandService) VerifyTwilioAccount(ctx context.Context, accountID string) (bool, error) {
	return s.blandClient.VerifyTwilioAccount(ctx, accountID)
}

// ===============================================
// Enterprise - SIP Integration
// ===============================================

// ListSIPTrunks returns all SIP trunks.
func (s *BlandService) ListSIPTrunks(ctx context.Context) ([]bland.SIPTrunk, error) {
	return s.blandClient.ListSIPTrunks(ctx)
}

// GetSIPTrunk retrieves a specific SIP trunk.
func (s *BlandService) GetSIPTrunk(ctx context.Context, trunkID string) (*bland.SIPTrunk, error) {
	return s.blandClient.GetSIPTrunk(ctx, trunkID)
}

// CreateSIPTrunk creates a new SIP trunk.
func (s *BlandService) CreateSIPTrunk(ctx context.Context, req *bland.CreateSIPTrunkRequest) (*bland.SIPTrunk, error) {
	return s.blandClient.CreateSIPTrunk(ctx, req)
}

// UpdateSIPTrunk updates a SIP trunk.
func (s *BlandService) UpdateSIPTrunk(ctx context.Context, trunkID string, req *bland.UpdateSIPTrunkRequest) (*bland.SIPTrunk, error) {
	return s.blandClient.UpdateSIPTrunk(ctx, trunkID, req)
}

// DeleteSIPTrunk deletes a SIP trunk.
func (s *BlandService) DeleteSIPTrunk(ctx context.Context, trunkID string) error {
	return s.blandClient.DeleteSIPTrunk(ctx, trunkID)
}

// TestSIPTrunk tests a SIP trunk connection.
func (s *BlandService) TestSIPTrunk(ctx context.Context, trunkID string) (bool, error) {
	return s.blandClient.TestSIPTrunk(ctx, trunkID)
}

// GetSIPTrunkStats retrieves statistics for a SIP trunk.
func (s *BlandService) GetSIPTrunkStats(ctx context.Context, trunkID string, period string) (*bland.SIPTrunkStats, error) {
	return s.blandClient.GetSIPTrunkStats(ctx, trunkID, period)
}

// ===============================================
// Enterprise - Custom Dialing Pools
// ===============================================

// ListDialingPools returns all dialing pools.
func (s *BlandService) ListDialingPools(ctx context.Context) ([]bland.DialingPool, error) {
	return s.blandClient.ListDialingPools(ctx)
}

// GetDialingPool retrieves a specific dialing pool.
func (s *BlandService) GetDialingPool(ctx context.Context, poolID string) (*bland.DialingPool, error) {
	return s.blandClient.GetDialingPool(ctx, poolID)
}

// CreateDialingPool creates a new dialing pool.
func (s *BlandService) CreateDialingPool(ctx context.Context, req *bland.CreateDialingPoolRequest) (*bland.DialingPool, error) {
	return s.blandClient.CreateDialingPool(ctx, req)
}

// UpdateDialingPool updates a dialing pool.
func (s *BlandService) UpdateDialingPool(ctx context.Context, poolID string, req *bland.UpdateDialingPoolRequest) (*bland.DialingPool, error) {
	return s.blandClient.UpdateDialingPool(ctx, poolID, req)
}

// DeleteDialingPool deletes a dialing pool.
func (s *BlandService) DeleteDialingPool(ctx context.Context, poolID string) error {
	return s.blandClient.DeleteDialingPool(ctx, poolID)
}

// AddNumberToPool adds a phone number to a dialing pool.
func (s *BlandService) AddNumberToPool(ctx context.Context, poolID string, number *bland.PoolNumber) error {
	return s.blandClient.AddNumberToPool(ctx, poolID, number)
}

// RemoveNumberFromPool removes a phone number from a dialing pool.
func (s *BlandService) RemoveNumberFromPool(ctx context.Context, poolID string, phoneNumber string) error {
	return s.blandClient.RemoveNumberFromPool(ctx, poolID, phoneNumber)
}

// GetDialingPoolStats retrieves statistics for a dialing pool.
func (s *BlandService) GetDialingPoolStats(ctx context.Context, poolID string) (*bland.DialingPoolStats, error) {
	return s.blandClient.GetDialingPoolStats(ctx, poolID)
}

// ===============================================
// Usage & Cost Tracking
// ===============================================

// GetUsageSummary retrieves usage summary.
func (s *BlandService) GetUsageSummary(ctx context.Context, req *bland.GetUsageSummaryRequest) (*bland.UsageSummary, error) {
	return s.blandClient.GetUsageSummary(ctx, req)
}

// GetDailyUsage retrieves daily usage data for the specified number of days.
func (s *BlandService) GetDailyUsage(ctx context.Context, days int) ([]bland.DailyUsage, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	return s.blandClient.GetDailyUsage(ctx, startDate, endDate)
}

// GetUsageLimits retrieves current usage limits.
func (s *BlandService) GetUsageLimits(ctx context.Context) (*bland.UsageLimits, error) {
	return s.blandClient.GetUsageLimits(ctx)
}

// SetUsageLimit sets a usage limit.
func (s *BlandService) SetUsageLimit(ctx context.Context, limitType string, value float64) error {
	return s.blandClient.SetUsageLimit(ctx, limitType, value)
}

// GetPricing retrieves pricing information.
func (s *BlandService) GetPricing(ctx context.Context) (*bland.PricingInfo, error) {
	return s.blandClient.GetPricing(ctx)
}

// GetUsageAlerts retrieves usage alerts.
func (s *BlandService) GetUsageAlerts(ctx context.Context) ([]bland.UsageAlert, error) {
	return s.blandClient.GetUsageAlerts(ctx)
}

// SetAlertThreshold sets an alert threshold.
func (s *BlandService) SetAlertThreshold(ctx context.Context, alertType string, threshold float64, thresholdType string) error {
	return s.blandClient.SetAlertThreshold(ctx, alertType, threshold, thresholdType)
}

// AcknowledgeAlert acknowledges an alert.
func (s *BlandService) AcknowledgeAlert(ctx context.Context, alertID string) error {
	return s.blandClient.AcknowledgeAlert(ctx, alertID)
}

// EstimateCallCost estimates the cost of a call.
func (s *BlandService) EstimateCallCost(ctx context.Context, durationMinutes float64, direction, numberType string, includeTranscription, includeAnalysis bool) (float64, error) {
	return s.blandClient.EstimateCallCost(ctx, durationMinutes, direction, numberType, includeTranscription, includeAnalysis)
}

// ===============================================
// Organization Management
// ===============================================

// GetOrganization retrieves the organization.
func (s *BlandService) GetOrganization(ctx context.Context) (*bland.Organization, error) {
	return s.blandClient.GetOrganization(ctx)
}

// ListOrganizationMembers lists organization members.
func (s *BlandService) ListOrganizationMembers(ctx context.Context) ([]bland.OrganizationMember, error) {
	return s.blandClient.ListOrganizationMembers(ctx)
}

// InviteOrganizationMember invites a member to the organization.
func (s *BlandService) InviteOrganizationMember(ctx context.Context, email, role string) error {
	return s.blandClient.InviteOrganizationMember(ctx, email, role)
}

// RemoveOrganizationMember removes a member from the organization.
func (s *BlandService) RemoveOrganizationMember(ctx context.Context, memberID string) error {
	return s.blandClient.RemoveOrganizationMember(ctx, memberID)
}

// UpdateMemberRole updates a member's role.
func (s *BlandService) UpdateMemberRole(ctx context.Context, memberID, role string) error {
	return s.blandClient.UpdateMemberRole(ctx, memberID, role)
}

// ===============================================
// Settings-Driven Configuration
// ===============================================

// GetQuickQuoteConfig builds a QuickQuoteConfig from database settings.
// This is the primary method for getting call configuration.
func (s *BlandService) GetQuickQuoteConfig(ctx context.Context) (*bland.QuickQuoteConfig, error) {
	if s.settingsService == nil {
		// Fallback to defaults if settings service not configured
		return bland.DefaultQuickQuoteConfig(s.webhookURL, "QuickQuote"), nil
	}

	callSettings, err := s.settingsService.GetCallSettings(ctx)
	if err != nil {
		s.logger.Warn("failed to load settings, using defaults", zap.Error(err))
		return bland.DefaultQuickQuoteConfig(s.webhookURL, "QuickQuote"), nil
	}

	// Convert domain.CallSettings to bland.CallSettings
	blandSettings := &bland.CallSettings{
		BusinessName:          callSettings.BusinessName,
		Voice:                 callSettings.Voice,
		VoiceStability:        callSettings.VoiceStability,
		VoiceSimilarityBoost:  callSettings.VoiceSimilarityBoost,
		VoiceStyle:            callSettings.VoiceStyle,
		VoiceSpeakerBoost:     callSettings.VoiceSpeakerBoost,
		Model:                 callSettings.Model,
		Language:              callSettings.Language,
		Temperature:           callSettings.Temperature,
		InterruptionThreshold: callSettings.InterruptionThreshold,
		WaitForGreeting:       callSettings.WaitForGreeting,
		NoiseCancellation:     callSettings.NoiseCancellation,
		BackgroundTrack:       callSettings.BackgroundTrack,
		MaxDurationMinutes:    callSettings.MaxDurationMinutes,
		RecordCalls:           callSettings.RecordCalls,
		QualityPreset:         callSettings.QualityPreset,
		CustomGreeting:        callSettings.CustomGreeting,
		ProjectTypes:          callSettings.ProjectTypes,
	}

	return bland.NewQuickQuoteConfigFromSettings(blandSettings, s.webhookURL), nil
}

// GetInboundConfig builds an InboundConfig from database settings.
// Use this when configuring inbound agents on phone numbers.
func (s *BlandService) GetInboundConfig(ctx context.Context) (*bland.InboundConfig, error) {
	cfg, err := s.GetQuickQuoteConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.BuildInboundConfig(), nil
}

// ConfigureInboundAgentFromSettings configures an inbound agent using database settings.
// This is the recommended method for setting up phone numbers with current configuration.
func (s *BlandService) ConfigureInboundAgentFromSettings(ctx context.Context, phoneNumber string) (*bland.PhoneNumber, error) {
	config, err := s.GetInboundConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbound config: %w", err)
	}

	return s.blandClient.ConfigureInboundAgent(ctx, phoneNumber, config)
}
