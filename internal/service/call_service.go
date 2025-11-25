// Package service contains business logic implementations.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/repository"
	"github.com/jkindrix/quickquote/internal/webhook"
)

// CallService handles call-related business logic.
type CallService struct {
	callRepo   domain.CallRepository
	quoteGen   QuoteGenerator
	logger     *zap.Logger
}

// QuoteGenerator defines the interface for generating quotes from transcripts.
type QuoteGenerator interface {
	GenerateQuote(ctx context.Context, transcript string, extractedData *domain.ExtractedData) (string, error)
}

// NewCallService creates a new CallService.
func NewCallService(
	callRepo domain.CallRepository,
	quoteGen QuoteGenerator,
	logger *zap.Logger,
) *CallService {
	return &CallService{
		callRepo:   callRepo,
		quoteGen:   quoteGen,
		logger:     logger,
	}
}

// ProcessWebhook processes an incoming Bland AI webhook payload.
func (s *CallService) ProcessWebhook(ctx context.Context, payload *webhook.BlandWebhookPayload) (*domain.Call, error) {
	s.logger.Info("processing webhook",
		zap.String("call_id", payload.CallID),
		zap.String("status", payload.Status),
	)

	// Check if call already exists
	call, err := s.callRepo.GetByBlandCallID(ctx, payload.CallID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing call: %w", err)
	}

	if call == nil {
		// Create new call record
		call = domain.NewCall(
			payload.CallID,
			payload.GetPhoneNumber(),
			payload.GetFromNumber(),
		)
		if err := s.callRepo.Create(ctx, call); err != nil {
			return nil, fmt.Errorf("failed to create call: %w", err)
		}
		s.logger.Info("created new call record", zap.String("id", call.ID.String()))
	}

	// Update call with webhook data
	s.updateCallFromPayload(call, payload)

	if err := s.callRepo.Update(ctx, call); err != nil {
		return nil, fmt.Errorf("failed to update call: %w", err)
	}

	s.logger.Info("call updated",
		zap.String("id", call.ID.String()),
		zap.String("status", string(call.Status)),
	)

	// Generate quote if call completed successfully with transcript
	if call.Status == domain.CallStatusCompleted && call.Transcript != nil && *call.Transcript != "" {
		go s.generateQuoteAsync(call.ID)
	}

	return call, nil
}

// updateCallFromPayload updates a call record with data from the webhook payload.
func (s *CallService) updateCallFromPayload(call *domain.Call, payload *webhook.BlandWebhookPayload) {
	// Update phone numbers if provided
	if phone := payload.GetPhoneNumber(); phone != "" {
		call.PhoneNumber = phone
	}
	if from := payload.GetFromNumber(); from != "" {
		call.FromNumber = from
	}

	// Update timestamps
	if payload.StartTime != nil {
		call.StartedAt = payload.StartTime
	}
	if payload.EndTime != nil {
		call.EndedAt = payload.EndTime
	}

	// Update duration
	if payload.Duration > 0 {
		duration := payload.GetDurationSeconds()
		call.DurationSeconds = &duration
	}

	// Update transcript
	if transcript := payload.GetTranscript(); transcript != "" {
		call.Transcript = &transcript
	}

	// Update transcript JSON
	if len(payload.Transcripts) > 0 {
		call.TranscriptJSON = make([]domain.TranscriptEntry, len(payload.Transcripts))
		for i, t := range payload.Transcripts {
			call.TranscriptJSON[i] = domain.TranscriptEntry{
				Role:      t.Role,
				Content:   t.Content,
				Timestamp: t.Timestamp,
			}
		}
	}

	// Update recording URL
	if payload.RecordingURL != "" {
		call.RecordingURL = &payload.RecordingURL
	}

	// Update extracted data
	extracted := payload.ExtractedVariables()
	call.ExtractedData = &domain.ExtractedData{
		ProjectType:       extracted.ProjectType,
		Requirements:      extracted.Requirements,
		Timeline:          extracted.Timeline,
		BudgetRange:       extracted.BudgetRange,
		ContactPreference: extracted.ContactPreference,
		CallerName:        extracted.CallerName,
	}

	// Update caller name if extracted
	if extracted.CallerName != "" {
		call.CallerName = &extracted.CallerName
	}

	// Update status
	switch {
	case payload.IsCompleted():
		call.Status = domain.CallStatusCompleted
	case payload.IsFailed():
		call.Status = domain.CallStatusFailed
		if payload.ErrorMessage != "" {
			call.ErrorMessage = &payload.ErrorMessage
		}
	case payload.IsNoAnswer():
		call.Status = domain.CallStatusNoAnswer
	default:
		call.Status = domain.CallStatusInProgress
	}

	// Update error message if present
	if payload.ErrorMessage != "" {
		call.ErrorMessage = &payload.ErrorMessage
	}
}

// generateQuoteAsync generates a quote for a call in the background.
func (s *CallService) generateQuoteAsync(callID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := s.GenerateQuote(ctx, callID); err != nil {
		s.logger.Error("failed to generate quote",
			zap.String("call_id", callID.String()),
			zap.Error(err),
		)
	}
}

// GenerateQuote generates a quote summary for a call.
func (s *CallService) GenerateQuote(ctx context.Context, callID uuid.UUID) (*domain.Call, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	if call.Transcript == nil || *call.Transcript == "" {
		return nil, errors.New("call has no transcript")
	}

	s.logger.Info("generating quote", zap.String("call_id", callID.String()))

	quote, err := s.quoteGen.GenerateQuote(ctx, *call.Transcript, call.ExtractedData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate quote: %w", err)
	}

	call.QuoteSummary = &quote

	if err := s.callRepo.Update(ctx, call); err != nil {
		return nil, fmt.Errorf("failed to update call with quote: %w", err)
	}

	s.logger.Info("quote generated successfully", zap.String("call_id", callID.String()))

	return call, nil
}

// GetCall retrieves a call by ID.
func (s *CallService) GetCall(ctx context.Context, id uuid.UUID) (*domain.Call, error) {
	return s.callRepo.GetByID(ctx, id)
}

// ListCalls retrieves calls with pagination.
func (s *CallService) ListCalls(ctx context.Context, page, pageSize int) ([]*domain.Call, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	calls, err := s.callRepo.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.callRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return calls, total, nil
}
