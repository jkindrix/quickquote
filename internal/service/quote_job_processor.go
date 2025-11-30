package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/ratelimit"
)

// QuoteJobProcessor handles async quote generation with retry support.
type QuoteJobProcessor struct {
	jobRepo   domain.QuoteJobRepository
	callRepo  domain.CallRepository
	quoteGen  QuoteGenerator
	limiter   *ratelimit.QuoteLimiter
	logger    *zap.Logger

	// Configuration
	pollInterval    time.Duration
	batchSize       int
	stuckJobTimeout time.Duration
	workerCount     int

	// Lifecycle
	stopCh   chan struct{}
	jobCh    chan *domain.QuoteJob
	wg       sync.WaitGroup
	workerWg sync.WaitGroup
	mu       sync.RWMutex
	running  bool
}

// QuoteJobProcessorConfig holds configuration for the processor.
type QuoteJobProcessorConfig struct {
	PollInterval    time.Duration
	BatchSize       int
	StuckJobTimeout time.Duration
	WorkerCount     int
}

// DefaultQuoteJobProcessorConfig returns sensible defaults.
func DefaultQuoteJobProcessorConfig() *QuoteJobProcessorConfig {
	return &QuoteJobProcessorConfig{
		PollInterval:    5 * time.Second,
		BatchSize:       10,
		StuckJobTimeout: 5 * time.Minute,
		WorkerCount:     3,
	}
}

// NewQuoteJobProcessor creates a new job processor.
func NewQuoteJobProcessor(
	jobRepo domain.QuoteJobRepository,
	callRepo domain.CallRepository,
	quoteGen QuoteGenerator,
	limiter *ratelimit.QuoteLimiter,
	logger *zap.Logger,
	config *QuoteJobProcessorConfig,
) *QuoteJobProcessor {
	if config == nil {
		config = DefaultQuoteJobProcessorConfig()
	}

	workerCount := config.WorkerCount
	if workerCount < 1 {
		workerCount = 1
	}

	return &QuoteJobProcessor{
		jobRepo:         jobRepo,
		callRepo:        callRepo,
		quoteGen:        quoteGen,
		limiter:         limiter,
		logger:          logger,
		pollInterval:    config.PollInterval,
		batchSize:       config.BatchSize,
		stuckJobTimeout: config.StuckJobTimeout,
		workerCount:     workerCount,
		stopCh:          make(chan struct{}),
		jobCh:           make(chan *domain.QuoteJob, config.BatchSize),
	}
}

// Start begins the job processing loop.
func (p *QuoteJobProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.New("processor already running")
	}
	p.running = true
	p.mu.Unlock()

	p.logger.Info("starting quote job processor",
		zap.Duration("poll_interval", p.pollInterval),
		zap.Int("batch_size", p.batchSize),
		zap.Int("worker_count", p.workerCount),
	)

	// Recover any stuck jobs from previous runs
	if err := p.recoverStuckJobs(ctx); err != nil {
		p.logger.Error("failed to recover stuck jobs", zap.Error(err))
	}

	// Start worker pool
	for i := 0; i < p.workerCount; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	// Start the dispatcher
	p.wg.Add(1)
	go p.runLoop()

	return nil
}

// Stop gracefully stops the processor.
func (p *QuoteJobProcessor) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	p.logger.Info("stopping quote job processor")

	// Signal stop to dispatcher
	close(p.stopCh)

	// Wait for dispatcher to finish (which closes jobCh)
	dispatcherDone := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(p.jobCh) // Signal workers to stop
		close(dispatcherDone)
	}()

	select {
	case <-dispatcherDone:
	case <-ctx.Done():
		p.logger.Warn("dispatcher stop timed out")
		return ctx.Err()
	}

	// Wait for all workers to finish
	workersDone := make(chan struct{})
	go func() {
		p.workerWg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		p.logger.Info("quote job processor stopped gracefully")
		return nil
	case <-ctx.Done():
		p.logger.Warn("workers stop timed out")
		return ctx.Err()
	}
}

// EnqueueJob creates a new quote generation job for a call.
func (p *QuoteJobProcessor) EnqueueJob(ctx context.Context, callID uuid.UUID) (*domain.QuoteJob, error) {
	// Check if job already exists for this call
	existing, err := p.jobRepo.GetByCallID(ctx, callID)
	if err == nil && !existing.IsTerminal() {
		p.logger.Debug("job already exists for call",
			zap.String("call_id", callID.String()),
			zap.String("job_id", existing.ID.String()),
		)
		return existing, nil
	}

	job := domain.NewQuoteJob(callID)
	if err := p.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	p.logger.Info("enqueued quote job",
		zap.String("job_id", job.ID.String()),
		zap.String("call_id", callID.String()),
	)

	return job, nil
}

// GetJobStatus retrieves the status of a job.
func (p *QuoteJobProcessor) GetJobStatus(ctx context.Context, jobID uuid.UUID) (*domain.QuoteJob, error) {
	return p.jobRepo.GetByID(ctx, jobID)
}

// GetJobByCallID retrieves the job for a specific call.
func (p *QuoteJobProcessor) GetJobByCallID(ctx context.Context, callID uuid.UUID) (*domain.QuoteJob, error) {
	return p.jobRepo.GetByCallID(ctx, callID)
}

// GetStats returns job queue statistics.
func (p *QuoteJobProcessor) GetStats(ctx context.Context) (map[domain.QuoteJobStatus]int, error) {
	return p.jobRepo.CountByStatus(ctx)
}

// GetRateLimiterStats returns rate limiter statistics.
func (p *QuoteJobProcessor) GetRateLimiterStats() *ratelimit.QuoteLimiterStats {
	if p.limiter == nil {
		return nil
	}
	stats := p.limiter.Stats()
	return &stats
}

// runLoop is the main processing loop.
func (p *QuoteJobProcessor) runLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.processBatch()
		}
	}
}

// processBatch fetches pending jobs and dispatches them to workers.
func (p *QuoteJobProcessor) processBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	jobs, err := p.jobRepo.GetPendingJobs(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("failed to get pending jobs", zap.Error(err))
		return
	}

	if len(jobs) == 0 {
		return
	}

	p.logger.Debug("dispatching job batch to workers", zap.Int("count", len(jobs)))

	for _, job := range jobs {
		select {
		case <-p.stopCh:
			return
		case p.jobCh <- job:
			// Job dispatched to worker
		}
	}
}

// worker processes jobs from the job channel.
func (p *QuoteJobProcessor) worker(id int) {
	defer p.workerWg.Done()

	logger := p.logger.With(zap.Int("worker_id", id))
	logger.Debug("worker started")

	for job := range p.jobCh {
		p.processJob(context.Background(), job)
	}

	logger.Debug("worker stopped")
}

// processJob processes a single job.
func (p *QuoteJobProcessor) processJob(ctx context.Context, job *domain.QuoteJob) {
	logger := p.logger.With(
		zap.String("job_id", job.ID.String()),
		zap.String("call_id", job.CallID.String()),
		zap.Int("attempt", job.Attempts+1),
	)

	logger.Info("processing job")

	// Acquire rate limit slot if limiter is configured
	if p.limiter != nil {
		if err := p.limiter.Acquire(ctx); err != nil {
			// Rate limited - don't mark as failed, just skip for now
			// The job will be picked up in the next batch
			logger.Warn("rate limited, deferring job",
				zap.Error(err),
				zap.String("limiter_stats", fmt.Sprintf("%+v", p.limiter.Stats())),
			)
			return
		}
		// Ensure we release the slot when done
		defer p.limiter.Release()
	}

	// Mark as processing
	job.MarkProcessing()
	if err := p.jobRepo.Update(ctx, job); err != nil {
		logger.Error("failed to mark job as processing", zap.Error(err))
		return
	}

	// Get the call
	call, err := p.callRepo.GetByID(ctx, job.CallID)
	if err != nil {
		logger.Error("failed to get call", zap.Error(err))
		p.failJob(ctx, job, fmt.Errorf("failed to get call: %w", err))
		return
	}

	// Validate call has transcript
	if call.Transcript == nil || *call.Transcript == "" {
		logger.Warn("call has no transcript")
		p.failJob(ctx, job, errors.New("call has no transcript"))
		return
	}

	// Generate quote
	quote, err := p.quoteGen.GenerateQuote(ctx, *call.Transcript, call.ExtractedData)
	if err != nil {
		logger.Error("quote generation failed", zap.Error(err))
		p.failJob(ctx, job, err)
		return
	}

	// Update call with quote
	call.QuoteSummary = &quote
	if err := p.callRepo.Update(ctx, call); err != nil {
		logger.Error("failed to update call with quote", zap.Error(err))
		p.failJob(ctx, job, fmt.Errorf("failed to update call: %w", err))
		return
	}

	// Mark job as completed
	job.MarkCompleted()
	if err := p.jobRepo.Update(ctx, job); err != nil {
		logger.Error("failed to mark job as completed", zap.Error(err))
		return
	}

	logger.Info("job completed successfully")
}

// failJob handles job failure with retry logic.
func (p *QuoteJobProcessor) failJob(ctx context.Context, job *domain.QuoteJob, err error) {
	logger := p.logger.With(
		zap.String("job_id", job.ID.String()),
		zap.String("call_id", job.CallID.String()),
	)

	job.MarkFailed(err)

	if job.Status == domain.QuoteJobStatusPending {
		// Scheduled for retry
		logger.Info("job scheduled for retry",
			zap.Int("attempts", job.Attempts),
			zap.Time("next_retry", job.ScheduledAt),
		)
	} else {
		// Permanently failed
		logger.Warn("job permanently failed",
			zap.Int("attempts", job.Attempts),
			zap.String("error", *job.LastError),
		)
	}

	if updateErr := p.jobRepo.Update(ctx, job); updateErr != nil {
		logger.Error("failed to update failed job", zap.Error(updateErr))
	}
}

// recoverStuckJobs handles jobs that were processing when the service stopped.
func (p *QuoteJobProcessor) recoverStuckJobs(ctx context.Context) error {
	stuckJobs, err := p.jobRepo.GetProcessingJobs(ctx, p.stuckJobTimeout)
	if err != nil {
		return fmt.Errorf("failed to get stuck jobs: %w", err)
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	p.logger.Info("recovering stuck jobs", zap.Int("count", len(stuckJobs)))

	for _, job := range stuckJobs {
		// Mark as failed to trigger retry logic
		job.MarkFailed(errors.New("job interrupted - process restarted"))

		if err := p.jobRepo.Update(ctx, job); err != nil {
			p.logger.Error("failed to recover stuck job",
				zap.String("job_id", job.ID.String()),
				zap.Error(err),
			)
			continue
		}

		p.logger.Info("recovered stuck job",
			zap.String("job_id", job.ID.String()),
			zap.String("status", string(job.Status)),
		)
	}

	return nil
}
