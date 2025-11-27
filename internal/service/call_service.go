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
	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

// CallService handles call-related business logic.
type CallService struct {
	callRepo     domain.CallRepository
	quoteGen     QuoteGenerator
	jobProcessor *QuoteJobProcessor
	logger       *zap.Logger
}

// QuoteGenerator defines the interface for generating quotes from transcripts.
type QuoteGenerator interface {
	GenerateQuote(ctx context.Context, transcript string, extractedData *domain.ExtractedData) (string, error)
}

// NewCallService creates a new CallService.
func NewCallService(
	callRepo domain.CallRepository,
	quoteGen QuoteGenerator,
	jobProcessor *QuoteJobProcessor,
	logger *zap.Logger,
) *CallService {
	return &CallService{
		callRepo:     callRepo,
		quoteGen:     quoteGen,
		jobProcessor: jobProcessor,
		logger:       logger,
	}
}

// ProcessCallEvent processes a normalized call event from any voice provider.
// This is the provider-agnostic entry point for call processing.
func (s *CallService) ProcessCallEvent(ctx context.Context, event *voiceprovider.CallEvent) (*domain.Call, error) {
	s.logger.Info("processing call event",
		zap.String("provider", string(event.Provider)),
		zap.String("provider_call_id", event.ProviderCallID),
		zap.String("status", string(event.Status)),
	)

	// Check if call already exists
	call, err := s.callRepo.GetByProviderCallID(ctx, event.ProviderCallID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing call: %w", err)
	}

	if call == nil {
		// Create new call record
		call = domain.NewCall(
			event.ProviderCallID,
			string(event.Provider),
			event.ToNumber,
			event.FromNumber,
		)
		if err := s.callRepo.Create(ctx, call); err != nil {
			return nil, fmt.Errorf("failed to create call: %w", err)
		}
		s.logger.Info("created new call record", zap.String("id", call.ID.String()))
	}

	// Update call with event data
	s.updateCallFromEvent(call, event)

	if err := s.callRepo.Update(ctx, call); err != nil {
		return nil, fmt.Errorf("failed to update call: %w", err)
	}

	s.logger.Info("call updated",
		zap.String("id", call.ID.String()),
		zap.String("status", string(call.Status)),
	)

	// Enqueue quote generation job if call completed successfully with transcript
	if call.Status == domain.CallStatusCompleted && call.Transcript != nil && *call.Transcript != "" {
		if s.jobProcessor != nil {
			if _, err := s.jobProcessor.EnqueueJob(ctx, call.ID); err != nil {
				s.logger.Error("failed to enqueue quote job",
					zap.String("call_id", call.ID.String()),
					zap.Error(err),
				)
				// Don't fail the whole request, quote will need manual retry
			}
		} else {
			// Fallback to synchronous generation if no job processor (for backwards compatibility)
			go s.generateQuoteAsync(call.ID)
		}
	}

	return call, nil
}

// updateCallFromEvent updates a call record with data from a normalized CallEvent.
func (s *CallService) updateCallFromEvent(call *domain.Call, event *voiceprovider.CallEvent) {
	// Update phone numbers if provided
	if event.ToNumber != "" {
		call.PhoneNumber = event.ToNumber
	}
	if event.FromNumber != "" {
		call.FromNumber = event.FromNumber
	}

	// Update caller name
	if event.CallerName != "" {
		call.CallerName = &event.CallerName
	}

	// Update timestamps
	if event.StartedAt != nil {
		call.StartedAt = event.StartedAt
	}
	if event.EndedAt != nil {
		call.EndedAt = event.EndedAt
	}

	// Update duration
	if event.DurationSecs > 0 {
		call.DurationSeconds = &event.DurationSecs
	}

	// Update transcript
	if event.Transcript != "" {
		call.Transcript = &event.Transcript
	}

	// Update transcript JSON from entries
	if len(event.TranscriptEntries) > 0 {
		call.TranscriptJSON = make([]domain.TranscriptEntry, len(event.TranscriptEntries))
		for i, t := range event.TranscriptEntries {
			call.TranscriptJSON[i] = domain.TranscriptEntry{
				Role:      t.Role,
				Content:   t.Content,
				Timestamp: t.Timestamp,
			}
		}
	}

	// Update recording URL
	if event.RecordingURL != "" {
		call.RecordingURL = &event.RecordingURL
	}

	// Update extracted data
	if event.ExtractedData != nil {
		call.ExtractedData = &domain.ExtractedData{
			ProjectType:       event.ExtractedData.ProjectType,
			Requirements:      event.ExtractedData.Requirements,
			Timeline:          event.ExtractedData.Timeline,
			BudgetRange:       event.ExtractedData.BudgetRange,
			ContactPreference: event.ExtractedData.ContactPreference,
			CallerName:        event.ExtractedData.CallerName,
		}

		// Update caller name from extracted data if not already set
		if call.CallerName == nil && event.ExtractedData.CallerName != "" {
			call.CallerName = &event.ExtractedData.CallerName
		}
	}

	// Update status
	call.Status = s.mapProviderStatus(event.Status)

	// Update error message if present
	if event.ErrorMessage != "" {
		call.ErrorMessage = &event.ErrorMessage
	}
}

// mapProviderStatus converts provider status to domain status.
func (s *CallService) mapProviderStatus(status voiceprovider.CallStatus) domain.CallStatus {
	switch status {
	case voiceprovider.CallStatusCompleted:
		return domain.CallStatusCompleted
	case voiceprovider.CallStatusFailed:
		return domain.CallStatusFailed
	case voiceprovider.CallStatusNoAnswer, voiceprovider.CallStatusVoicemail:
		return domain.CallStatusNoAnswer
	case voiceprovider.CallStatusInProgress:
		return domain.CallStatusInProgress
	default:
		return domain.CallStatusPending
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
