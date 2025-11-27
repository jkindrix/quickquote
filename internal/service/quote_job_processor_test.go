package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/ratelimit"
	"github.com/jkindrix/quickquote/internal/repository"
)

// MockQuoteJobRepository is a mock implementation of domain.QuoteJobRepository.
type MockQuoteJobRepository struct {
	mu   sync.RWMutex
	jobs map[uuid.UUID]*domain.QuoteJob

	CreateError error
	UpdateError error
}

func NewMockQuoteJobRepository() *MockQuoteJobRepository {
	return &MockQuoteJobRepository{
		jobs: make(map[uuid.UUID]*domain.QuoteJob),
	}
}

func (m *MockQuoteJobRepository) Create(ctx context.Context, job *domain.QuoteJob) error {
	if m.CreateError != nil {
		return m.CreateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *MockQuoteJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuoteJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if job, ok := m.jobs[id]; ok {
		return job, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockQuoteJobRepository) GetByCallID(ctx context.Context, callID uuid.UUID) (*domain.QuoteJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, job := range m.jobs {
		if job.CallID == callID {
			return job, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *MockQuoteJobRepository) Update(ctx context.Context, job *domain.QuoteJob) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[job.ID]; !ok {
		return repository.ErrNotFound
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *MockQuoteJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*domain.QuoteJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var pending []*domain.QuoteJob
	now := time.Now()
	for _, job := range m.jobs {
		if job.Status == domain.QuoteJobStatusPending && job.ScheduledAt.Before(now) {
			pending = append(pending, job)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *MockQuoteJobRepository) GetProcessingJobs(ctx context.Context, olderThan time.Duration) ([]*domain.QuoteJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().Add(-olderThan)
	var stuck []*domain.QuoteJob
	for _, job := range m.jobs {
		if job.Status == domain.QuoteJobStatusProcessing && job.StartedAt != nil && job.StartedAt.Before(cutoff) {
			stuck = append(stuck, job)
		}
	}
	return stuck, nil
}

func (m *MockQuoteJobRepository) CountByStatus(ctx context.Context) (map[domain.QuoteJobStatus]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[domain.QuoteJobStatus]int)
	for _, job := range m.jobs {
		counts[job.Status]++
	}
	return counts, nil
}

func newTestProcessor() (*QuoteJobProcessor, *MockQuoteJobRepository, *MockCallRepository, *MockQuoteGenerator) {
	logger := zap.NewNop()
	jobRepo := NewMockQuoteJobRepository()
	callRepo := NewMockCallRepository()
	quoteGen := NewMockQuoteGenerator()

	config := &QuoteJobProcessorConfig{
		PollInterval:    100 * time.Millisecond,
		BatchSize:       10,
		StuckJobTimeout: 1 * time.Minute,
	}

	// Pass nil for limiter - tests don't need rate limiting
	processor := NewQuoteJobProcessor(jobRepo, callRepo, quoteGen, nil, logger, config)
	return processor, jobRepo, callRepo, quoteGen
}

func TestQuoteJobProcessor_EnqueueJob(t *testing.T) {
	processor, jobRepo, _, _ := newTestProcessor()
	ctx := context.Background()

	callID := uuid.New()
	job, err := processor.EnqueueJob(ctx, callID)
	if err != nil {
		t.Fatalf("EnqueueJob() error = %v", err)
	}

	if job.CallID != callID {
		t.Errorf("expected CallID %s, got %s", callID, job.CallID)
	}
	if job.Status != domain.QuoteJobStatusPending {
		t.Errorf("expected status %s, got %s", domain.QuoteJobStatusPending, job.Status)
	}

	// Verify job was persisted
	stored, err := jobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.ID != job.ID {
		t.Errorf("stored job ID mismatch")
	}
}

func TestQuoteJobProcessor_EnqueueJob_DuplicatePrevented(t *testing.T) {
	processor, _, _, _ := newTestProcessor()
	ctx := context.Background()

	callID := uuid.New()

	// First enqueue
	job1, err := processor.EnqueueJob(ctx, callID)
	if err != nil {
		t.Fatalf("first EnqueueJob() error = %v", err)
	}

	// Second enqueue should return existing job
	job2, err := processor.EnqueueJob(ctx, callID)
	if err != nil {
		t.Fatalf("second EnqueueJob() error = %v", err)
	}

	if job1.ID != job2.ID {
		t.Errorf("expected same job ID, got %s and %s", job1.ID, job2.ID)
	}
}

func TestQuoteJobProcessor_EnqueueJob_AllowsNewAfterTerminal(t *testing.T) {
	processor, jobRepo, _, _ := newTestProcessor()
	ctx := context.Background()

	callID := uuid.New()

	// First enqueue and complete
	job1, _ := processor.EnqueueJob(ctx, callID)
	job1.MarkCompleted()
	jobRepo.Update(ctx, job1)

	// Second enqueue should create new job
	job2, err := processor.EnqueueJob(ctx, callID)
	if err != nil {
		t.Fatalf("second EnqueueJob() error = %v", err)
	}

	if job1.ID == job2.ID {
		t.Errorf("expected different job ID after terminal state")
	}
}

func TestQuoteJobProcessor_ProcessJob_Success(t *testing.T) {
	processor, jobRepo, callRepo, quoteGen := newTestProcessor()
	ctx := context.Background()

	// Create a call with transcript
	transcript := "Test transcript"
	call := domain.NewCall("provider-123", "bland", "+1234567890", "+19876543210")
	call.Transcript = &transcript
	call.Status = domain.CallStatusCompleted
	callRepo.Create(ctx, call)

	// Create a job
	job := domain.NewQuoteJob(call.ID)
	jobRepo.Create(ctx, job)

	// Process the job
	processor.processJob(ctx, job)

	// Verify job completed
	updatedJob, _ := jobRepo.GetByID(ctx, job.ID)
	if updatedJob.Status != domain.QuoteJobStatusCompleted {
		t.Errorf("expected status %s, got %s", domain.QuoteJobStatusCompleted, updatedJob.Status)
	}

	// Verify call got quote
	updatedCall, _ := callRepo.GetByID(ctx, call.ID)
	if updatedCall.QuoteSummary == nil {
		t.Error("expected QuoteSummary to be set")
	}
	if *updatedCall.QuoteSummary != quoteGen.GeneratedQuote {
		t.Errorf("expected quote %q, got %q", quoteGen.GeneratedQuote, *updatedCall.QuoteSummary)
	}
}

func TestQuoteJobProcessor_ProcessJob_RetryOnFailure(t *testing.T) {
	processor, jobRepo, callRepo, quoteGen := newTestProcessor()
	ctx := context.Background()

	// Create a call with transcript
	transcript := "Test transcript"
	call := domain.NewCall("provider-123", "bland", "+1234567890", "+19876543210")
	call.Transcript = &transcript
	call.Status = domain.CallStatusCompleted
	callRepo.Create(ctx, call)

	// Create a job
	job := domain.NewQuoteJob(call.ID)
	jobRepo.Create(ctx, job)

	// Make quote generation fail
	quoteGen.GenerateQuoteError = errors.New("AI service unavailable")

	// Process the job
	processor.processJob(ctx, job)

	// Verify job scheduled for retry
	updatedJob, _ := jobRepo.GetByID(ctx, job.ID)
	if updatedJob.Status != domain.QuoteJobStatusPending {
		t.Errorf("expected status %s for retry, got %s", domain.QuoteJobStatusPending, updatedJob.Status)
	}
	if updatedJob.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", updatedJob.Attempts)
	}
	if updatedJob.LastError == nil || *updatedJob.LastError == "" {
		t.Error("expected LastError to be set")
	}
}

func TestQuoteJobProcessor_ProcessJob_FailsAfterMaxRetries(t *testing.T) {
	processor, jobRepo, callRepo, quoteGen := newTestProcessor()
	ctx := context.Background()

	// Create a call with transcript
	transcript := "Test transcript"
	call := domain.NewCall("provider-123", "bland", "+1234567890", "+19876543210")
	call.Transcript = &transcript
	call.Status = domain.CallStatusCompleted
	callRepo.Create(ctx, call)

	// Create a job at max attempts
	job := domain.NewQuoteJob(call.ID)
	job.Attempts = 2 // Already tried twice
	jobRepo.Create(ctx, job)

	// Make quote generation fail
	quoteGen.GenerateQuoteError = errors.New("persistent failure")

	// Process the job (3rd attempt)
	processor.processJob(ctx, job)

	// Verify job permanently failed
	updatedJob, _ := jobRepo.GetByID(ctx, job.ID)
	if updatedJob.Status != domain.QuoteJobStatusFailed {
		t.Errorf("expected status %s after max retries, got %s", domain.QuoteJobStatusFailed, updatedJob.Status)
	}
	if updatedJob.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", updatedJob.Attempts)
	}
}

func TestQuoteJobProcessor_ProcessJob_NoTranscript(t *testing.T) {
	processor, jobRepo, callRepo, _ := newTestProcessor()
	ctx := context.Background()

	// Create a call WITHOUT transcript
	call := domain.NewCall("provider-123", "bland", "+1234567890", "+19876543210")
	call.Status = domain.CallStatusCompleted
	callRepo.Create(ctx, call)

	// Create a job
	job := domain.NewQuoteJob(call.ID)
	jobRepo.Create(ctx, job)

	// Process the job
	processor.processJob(ctx, job)

	// Verify job failed (no transcript to process)
	updatedJob, _ := jobRepo.GetByID(ctx, job.ID)
	if updatedJob.LastError == nil {
		t.Error("expected error for missing transcript")
	}
}

func TestQuoteJobProcessor_GetStats(t *testing.T) {
	processor, jobRepo, _, _ := newTestProcessor()
	ctx := context.Background()

	// Create jobs in different states
	job1 := domain.NewQuoteJob(uuid.New())
	job1.Status = domain.QuoteJobStatusPending
	jobRepo.Create(ctx, job1)

	job2 := domain.NewQuoteJob(uuid.New())
	job2.Status = domain.QuoteJobStatusCompleted
	jobRepo.Create(ctx, job2)

	job3 := domain.NewQuoteJob(uuid.New())
	job3.Status = domain.QuoteJobStatusFailed
	jobRepo.Create(ctx, job3)

	stats, err := processor.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats[domain.QuoteJobStatusPending] != 1 {
		t.Errorf("expected 1 pending, got %d", stats[domain.QuoteJobStatusPending])
	}
	if stats[domain.QuoteJobStatusCompleted] != 1 {
		t.Errorf("expected 1 completed, got %d", stats[domain.QuoteJobStatusCompleted])
	}
	if stats[domain.QuoteJobStatusFailed] != 1 {
		t.Errorf("expected 1 failed, got %d", stats[domain.QuoteJobStatusFailed])
	}
}

func TestQuoteJobProcessor_StartStop(t *testing.T) {
	processor, _, _, _ := newTestProcessor()
	ctx := context.Background()

	// Start processor
	err := processor.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify it's running
	processor.mu.RLock()
	running := processor.running
	processor.mu.RUnlock()
	if !running {
		t.Error("processor should be running")
	}

	// Cannot start twice
	err = processor.Start(ctx)
	if err == nil {
		t.Error("expected error when starting twice")
	}

	// Stop processor
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = processor.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify it stopped
	processor.mu.RLock()
	running = processor.running
	processor.mu.RUnlock()
	if running {
		t.Error("processor should not be running")
	}
}

func TestQuoteJobProcessor_RecoverStuckJobs(t *testing.T) {
	processor, jobRepo, _, _ := newTestProcessor()
	ctx := context.Background()

	// Create a stuck job (processing for too long)
	stuckJob := domain.NewQuoteJob(uuid.New())
	stuckJob.Status = domain.QuoteJobStatusProcessing
	startedAt := time.Now().Add(-10 * time.Minute) // Started 10 minutes ago
	stuckJob.StartedAt = &startedAt
	stuckJob.Attempts = 1
	jobRepo.Create(ctx, stuckJob)

	// Recover stuck jobs
	err := processor.recoverStuckJobs(ctx)
	if err != nil {
		t.Fatalf("recoverStuckJobs() error = %v", err)
	}

	// Verify job was rescheduled for retry
	recovered, _ := jobRepo.GetByID(ctx, stuckJob.ID)
	if recovered.Status != domain.QuoteJobStatusPending {
		t.Errorf("expected status %s after recovery, got %s", domain.QuoteJobStatusPending, recovered.Status)
	}
	if recovered.LastError == nil {
		t.Error("expected LastError to be set after recovery")
	}
}

func TestQuoteJob_ExponentialBackoff(t *testing.T) {
	// Test backoff by observing MarkFailed behavior
	// Note: MarkProcessing increments Attempts, MarkFailed checks CanRetry
	// MaxAttempts is 3 by default, so job can retry while Attempts < 3
	tests := []struct {
		name         string
		setupAttempts int
		expectRetry  bool
	}{
		{"after first attempt can retry", 1, true},
		{"after second attempt can retry", 2, true},
		{"after third attempt fails permanently", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := domain.NewQuoteJob(uuid.New())
			job.Attempts = tt.setupAttempts

			job.MarkFailed(errors.New("test error"))

			if tt.expectRetry {
				if job.Status != domain.QuoteJobStatusPending {
					t.Errorf("expected pending status for retry, got %s", job.Status)
				}
				if job.ScheduledAt.Before(time.Now()) {
					t.Error("expected scheduled_at to be in the future")
				}
			} else {
				if job.Status != domain.QuoteJobStatusFailed {
					t.Errorf("expected failed status, got %s", job.Status)
				}
			}
		})
	}
}

func TestQuoteJob_IsReadyToProcess(t *testing.T) {
	job := domain.NewQuoteJob(uuid.New())
	job.Status = domain.QuoteJobStatusPending

	// Just created, should be ready
	if !job.IsReadyToProcess() {
		t.Error("new job should be ready to process")
	}

	// Schedule for future
	job.ScheduledAt = time.Now().Add(1 * time.Hour)
	if job.IsReadyToProcess() {
		t.Error("job scheduled for future should not be ready")
	}

	// Different status
	job.ScheduledAt = time.Now().Add(-1 * time.Second)
	job.Status = domain.QuoteJobStatusProcessing
	if job.IsReadyToProcess() {
		t.Error("processing job should not be ready")
	}
}

func TestQuoteJob_CanRetry(t *testing.T) {
	job := domain.NewQuoteJob(uuid.New())
	job.MaxAttempts = 3

	// Fresh job
	if !job.CanRetry() {
		t.Error("fresh job should be able to retry")
	}

	// After some attempts
	job.Attempts = 2
	if !job.CanRetry() {
		t.Error("job with 2/3 attempts should be able to retry")
	}

	// At max attempts
	job.Attempts = 3
	if job.CanRetry() {
		t.Error("job at max attempts should not be able to retry")
	}

	// Completed job
	job.Attempts = 1
	job.Status = domain.QuoteJobStatusCompleted
	if job.CanRetry() {
		t.Error("completed job should not be able to retry")
	}
}

func TestQuoteJob_IsTerminal(t *testing.T) {
	job := domain.NewQuoteJob(uuid.New())

	// Pending is not terminal
	job.Status = domain.QuoteJobStatusPending
	if job.IsTerminal() {
		t.Error("pending job should not be terminal")
	}

	// Processing is not terminal
	job.Status = domain.QuoteJobStatusProcessing
	if job.IsTerminal() {
		t.Error("processing job should not be terminal")
	}

	// Completed is terminal
	job.Status = domain.QuoteJobStatusCompleted
	if !job.IsTerminal() {
		t.Error("completed job should be terminal")
	}

	// Failed is terminal
	job.Status = domain.QuoteJobStatusFailed
	if !job.IsTerminal() {
		t.Error("failed job should be terminal")
	}
}

func TestQuoteJobProcessor_WithRateLimiter(t *testing.T) {
	logger := zap.NewNop()
	jobRepo := NewMockQuoteJobRepository()
	callRepo := NewMockCallRepository()
	quoteGen := NewMockQuoteGenerator()

	// Create a limiter with low limits for testing
	limiterConfig := &ratelimit.QuoteLimiterConfig{
		MaxRequestsPerMinute: 2,
		MaxRequestsPerHour:   10,
		MaxRequestsPerDay:    20,
		MaxConcurrent:        1,
	}
	limiter := ratelimit.NewQuoteLimiter(limiterConfig, logger)

	config := &QuoteJobProcessorConfig{
		PollInterval:    100 * time.Millisecond,
		BatchSize:       10,
		StuckJobTimeout: 1 * time.Minute,
	}

	processor := NewQuoteJobProcessor(jobRepo, callRepo, quoteGen, limiter, logger, config)
	ctx := context.Background()

	// Create calls with transcripts
	var calls []*domain.Call
	for i := 0; i < 3; i++ {
		transcript := "Test transcript"
		call := domain.NewCall("provider-123", "bland", "+1234567890", "+19876543210")
		call.Transcript = &transcript
		call.Status = domain.CallStatusCompleted
		callRepo.Create(ctx, call)
		calls = append(calls, call)
	}

	// Create jobs for all calls
	var jobs []*domain.QuoteJob
	for _, call := range calls {
		job := domain.NewQuoteJob(call.ID)
		jobRepo.Create(ctx, job)
		jobs = append(jobs, job)
	}

	// Process first job - should succeed
	processor.processJob(ctx, jobs[0])
	updatedJob0, _ := jobRepo.GetByID(ctx, jobs[0].ID)
	if updatedJob0.Status != domain.QuoteJobStatusCompleted {
		t.Errorf("first job should complete, got status %s", updatedJob0.Status)
	}

	// Process second job - should succeed (2nd of 2 per minute)
	processor.processJob(ctx, jobs[1])
	updatedJob1, _ := jobRepo.GetByID(ctx, jobs[1].ID)
	if updatedJob1.Status != domain.QuoteJobStatusCompleted {
		t.Errorf("second job should complete, got status %s", updatedJob1.Status)
	}

	// Process third job - should be rate limited (3rd request exceeds 2/minute)
	processor.processJob(ctx, jobs[2])
	updatedJob2, _ := jobRepo.GetByID(ctx, jobs[2].ID)
	// Job should still be pending because rate limiter blocked it
	if updatedJob2.Status != domain.QuoteJobStatusPending {
		t.Errorf("third job should be deferred due to rate limit, got status %s", updatedJob2.Status)
	}

	// Verify limiter stats
	stats := processor.GetRateLimiterStats()
	if stats == nil {
		t.Fatal("expected rate limiter stats")
	}
	if stats.MinuteRemaining != 0 {
		t.Errorf("expected 0 minute remaining, got %d", stats.MinuteRemaining)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
}

func TestQuoteJobProcessor_GetRateLimiterStats_NoLimiter(t *testing.T) {
	processor, _, _, _ := newTestProcessor()

	// Processor without limiter should return nil stats
	stats := processor.GetRateLimiterStats()
	if stats != nil {
		t.Error("expected nil stats when no limiter configured")
	}
}
