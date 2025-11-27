package domain

import (
	"time"

	"github.com/google/uuid"
)

// QuoteJobStatus represents the state of a quote generation job.
type QuoteJobStatus string

const (
	QuoteJobStatusPending    QuoteJobStatus = "pending"
	QuoteJobStatusProcessing QuoteJobStatus = "processing"
	QuoteJobStatusCompleted  QuoteJobStatus = "completed"
	QuoteJobStatusFailed     QuoteJobStatus = "failed"
)

// QuoteJob represents an async quote generation job with retry support.
type QuoteJob struct {
	ID          uuid.UUID      `json:"id"`
	CallID      uuid.UUID      `json:"call_id"`
	Status      QuoteJobStatus `json:"status"`
	Attempts    int            `json:"attempts"`
	MaxAttempts int            `json:"max_attempts"`

	// Timing
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error tracking
	LastError  *string `json:"last_error,omitempty"`
	ErrorCount int     `json:"error_count"`

	// Metadata for extensibility
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewQuoteJob creates a new quote generation job for a call.
func NewQuoteJob(callID uuid.UUID) *QuoteJob {
	now := time.Now()
	return &QuoteJob{
		ID:          uuid.New(),
		CallID:      callID,
		Status:      QuoteJobStatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: now,
		ErrorCount:  0,
		Metadata:    make(map[string]interface{}),
	}
}

// CanRetry returns true if the job can be retried.
func (j *QuoteJob) CanRetry() bool {
	return j.Attempts < j.MaxAttempts && j.Status != QuoteJobStatusCompleted
}

// IsTerminal returns true if the job is in a final state.
func (j *QuoteJob) IsTerminal() bool {
	return j.Status == QuoteJobStatusCompleted || j.Status == QuoteJobStatusFailed
}

// MarkProcessing marks the job as currently being processed.
func (j *QuoteJob) MarkProcessing() {
	now := time.Now()
	j.Status = QuoteJobStatusProcessing
	j.Attempts++
	j.StartedAt = &now
	j.UpdatedAt = now
}

// MarkCompleted marks the job as successfully completed.
func (j *QuoteJob) MarkCompleted() {
	now := time.Now()
	j.Status = QuoteJobStatusCompleted
	j.CompletedAt = &now
	j.UpdatedAt = now
}

// MarkFailed marks the job as failed with an error message.
// If retries are available, schedules for retry with exponential backoff.
func (j *QuoteJob) MarkFailed(err error) {
	now := time.Now()
	j.UpdatedAt = now
	j.ErrorCount++

	errMsg := err.Error()
	j.LastError = &errMsg

	if j.CanRetry() {
		// Schedule retry with exponential backoff: 5s, 15s, 60s
		backoff := j.calculateBackoff()
		j.ScheduledAt = now.Add(backoff)
		j.Status = QuoteJobStatusPending
	} else {
		// No more retries - mark as permanently failed
		j.Status = QuoteJobStatusFailed
		j.CompletedAt = &now
	}
}

// calculateBackoff returns the backoff duration for the next retry attempt.
// Uses exponential backoff: 5s, 15s, 60s
func (j *QuoteJob) calculateBackoff() time.Duration {
	switch j.Attempts {
	case 1:
		return 5 * time.Second
	case 2:
		return 15 * time.Second
	default:
		return 60 * time.Second
	}
}

// NextRetryAt returns when the job should next be attempted, or nil if not applicable.
func (j *QuoteJob) NextRetryAt() *time.Time {
	if j.Status == QuoteJobStatusPending && j.Attempts > 0 {
		return &j.ScheduledAt
	}
	return nil
}

// TimeUntilRetry returns the duration until the next retry, or 0 if ready now.
func (j *QuoteJob) TimeUntilRetry() time.Duration {
	if j.Status != QuoteJobStatusPending {
		return 0
	}
	remaining := time.Until(j.ScheduledAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsReadyToProcess returns true if the job is ready to be processed.
func (j *QuoteJob) IsReadyToProcess() bool {
	return j.Status == QuoteJobStatusPending && time.Now().After(j.ScheduledAt)
}
