package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BackoffConfig configures exponential backoff behavior.
type BackoffConfig struct {
	// InitialDelay is the first delay duration
	InitialDelay time.Duration

	// MaxDelay is the maximum delay duration
	MaxDelay time.Duration

	// Multiplier is the factor to multiply delay by after each retry
	Multiplier float64

	// MaxRetries is the maximum number of retry attempts (0 = unlimited)
	MaxRetries int

	// Jitter adds randomness to delays to prevent thundering herd
	Jitter float64 // 0.0 to 1.0, e.g., 0.1 = +/- 10%

	// RetryableStatusCodes defines which HTTP status codes should trigger retry
	RetryableStatusCodes []int

	// RetryOn429 specifically handles rate limit responses (429 Too Many Requests)
	RetryOn429 bool

	// RespectRetryAfter honors the Retry-After header if present
	RespectRetryAfter bool
}

// DefaultBackoffConfig returns sensible defaults for API requests.
func DefaultBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		Multiplier:        2.0,
		MaxRetries:        5,
		Jitter:            0.2,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,     // 429
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
			http.StatusBadGateway,          // 502
			http.StatusRequestTimeout,      // 408
		},
		RetryOn429:        true,
		RespectRetryAfter: true,
	}
}

// AggressiveBackoffConfig returns config for high-priority requests.
func AggressiveBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		InitialDelay:      50 * time.Millisecond,
		MaxDelay:          10 * time.Second,
		Multiplier:        1.5,
		MaxRetries:        10,
		Jitter:            0.3,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
		RetryOn429:        true,
		RespectRetryAfter: true,
	}
}

// ConservativeBackoffConfig returns config that backs off more slowly.
func ConservativeBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          60 * time.Second,
		Multiplier:        3.0,
		MaxRetries:        3,
		Jitter:            0.1,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,
			http.StatusServiceUnavailable,
		},
		RetryOn429:        true,
		RespectRetryAfter: true,
	}
}

// Backoff provides exponential backoff retry logic.
type Backoff struct {
	config *BackoffConfig
	logger *zap.Logger

	mu         sync.RWMutex
	stats      BackoffStats
	lastError  error
	lastDelay  time.Duration
}

// BackoffStats tracks retry statistics.
type BackoffStats struct {
	TotalAttempts     int64         `json:"total_attempts"`
	TotalRetries      int64         `json:"total_retries"`
	SuccessfulRetries int64         `json:"successful_retries"`
	ExhaustedRetries  int64         `json:"exhausted_retries"`
	TotalDelayTime    time.Duration `json:"total_delay_time"`
	AvgRetryDelay     time.Duration `json:"avg_retry_delay"`
	MaxDelayUsed      time.Duration `json:"max_delay_used"`
}

// NewBackoff creates a new Backoff instance.
func NewBackoff(config *BackoffConfig, logger *zap.Logger) *Backoff {
	if config == nil {
		config = DefaultBackoffConfig()
	}
	return &Backoff{
		config: config,
		logger: logger,
	}
}

// Errors for backoff.
var (
	ErrMaxRetriesExhausted = errors.New("maximum retries exhausted")
	ErrNotRetryable        = errors.New("error is not retryable")
	ErrContextCanceled     = errors.New("context canceled during backoff")
)

// RetryableError wraps an error with retry information.
type RetryableError struct {
	Err           error
	StatusCode    int
	RetryAfter    time.Duration
	Attempt       int
	TotalAttempts int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d/%d): %v", e.Attempt, e.TotalAttempts, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// Operation is a function that can be retried.
type Operation func(ctx context.Context) error

// OperationWithResult is a function that returns a result and can be retried.
type OperationWithResult[T any] func(ctx context.Context) (T, error)

// Execute runs an operation with exponential backoff retry logic.
func (b *Backoff) Execute(ctx context.Context, op Operation) error {
	for attempt := 0; ; attempt++ {
		b.mu.Lock()
		b.stats.TotalAttempts++
		b.mu.Unlock()

		// Check context before attempt
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, err)
		}

		// Execute the operation
		err := op(ctx)
		if err == nil {
			if attempt > 0 {
				b.mu.Lock()
				b.stats.SuccessfulRetries++
				b.mu.Unlock()

				b.logger.Info("operation succeeded after retry",
					zap.Int("attempts", attempt+1),
				)
			}
			return nil
		}

		// Check if we should retry
		if !b.shouldRetry(err, attempt) {
			b.mu.Lock()
			b.stats.ExhaustedRetries++
			b.lastError = err
			b.mu.Unlock()

			if attempt >= b.config.MaxRetries && b.config.MaxRetries > 0 {
				return fmt.Errorf("%w after %d attempts: %v", ErrMaxRetriesExhausted, attempt+1, err)
			}
			return fmt.Errorf("%w: %v", ErrNotRetryable, err)
		}

		// Calculate delay
		delay := b.calculateDelay(err, attempt)

		b.mu.Lock()
		b.stats.TotalRetries++
		b.stats.TotalDelayTime += delay
		if delay > b.stats.MaxDelayUsed {
			b.stats.MaxDelayUsed = delay
		}
		b.lastDelay = delay
		b.mu.Unlock()

		b.logger.Warn("operation failed, retrying with backoff",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Duration("delay", delay),
		)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
}

// ExecuteWithResult runs an operation that returns a result with retry logic.
func ExecuteWithResult[T any](ctx context.Context, b *Backoff, op OperationWithResult[T]) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; ; attempt++ {
		b.mu.Lock()
		b.stats.TotalAttempts++
		b.mu.Unlock()

		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("%w: %v", ErrContextCanceled, err)
		}

		var err error
		result, err = op(ctx)
		if err == nil {
			if attempt > 0 {
				b.mu.Lock()
				b.stats.SuccessfulRetries++
				b.mu.Unlock()
			}
			return result, nil
		}

		lastErr = err

		if !b.shouldRetry(err, attempt) {
			b.mu.Lock()
			b.stats.ExhaustedRetries++
			b.lastError = err
			b.mu.Unlock()

			if attempt >= b.config.MaxRetries && b.config.MaxRetries > 0 {
				return result, fmt.Errorf("%w after %d attempts: %v", ErrMaxRetriesExhausted, attempt+1, lastErr)
			}
			return result, fmt.Errorf("%w: %v", ErrNotRetryable, lastErr)
		}

		delay := b.calculateDelay(err, attempt)

		b.mu.Lock()
		b.stats.TotalRetries++
		b.stats.TotalDelayTime += delay
		if delay > b.stats.MaxDelayUsed {
			b.stats.MaxDelayUsed = delay
		}
		b.lastDelay = delay
		b.mu.Unlock()

		b.logger.Warn("operation failed, retrying with backoff",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Duration("delay", delay),
		)

		select {
		case <-ctx.Done():
			return result, fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		case <-time.After(delay):
		}
	}
}

// shouldRetry determines if an operation should be retried.
func (b *Backoff) shouldRetry(err error, attempt int) bool {
	// Check max retries
	if b.config.MaxRetries > 0 && attempt >= b.config.MaxRetries {
		return false
	}

	// Check for RetryableError with status code
	var retryErr *RetryableError
	if errors.As(err, &retryErr) {
		// Check if status code is retryable
		for _, code := range b.config.RetryableStatusCodes {
			if retryErr.StatusCode == code {
				return true
			}
		}
		// 429 special handling
		if b.config.RetryOn429 && retryErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Default: retry on all errors
	return true
}

// calculateDelay calculates the delay for the next retry attempt.
func (b *Backoff) calculateDelay(err error, attempt int) time.Duration {
	// Check for Retry-After header
	var retryErr *RetryableError
	if errors.As(err, &retryErr) && b.config.RespectRetryAfter && retryErr.RetryAfter > 0 {
		// Use Retry-After but cap at MaxDelay
		if retryErr.RetryAfter > b.config.MaxDelay {
			return b.config.MaxDelay
		}
		return retryErr.RetryAfter
	}

	// Calculate exponential delay
	delay := float64(b.config.InitialDelay) * math.Pow(b.config.Multiplier, float64(attempt))

	// Apply jitter
	if b.config.Jitter > 0 {
		jitterRange := delay * b.config.Jitter
		jitter := (rand.Float64()*2 - 1) * jitterRange // -jitter to +jitter
		delay = delay + jitter
	}

	// Cap at max delay
	if delay > float64(b.config.MaxDelay) {
		delay = float64(b.config.MaxDelay)
	}

	return time.Duration(delay)
}

// Stats returns current backoff statistics.
func (b *Backoff) Stats() BackoffStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := b.stats
	if stats.TotalRetries > 0 {
		stats.AvgRetryDelay = stats.TotalDelayTime / time.Duration(stats.TotalRetries)
	}
	return stats
}

// Reset resets the backoff statistics.
func (b *Backoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats = BackoffStats{}
	b.lastError = nil
	b.lastDelay = 0
}

// LastError returns the last error encountered.
func (b *Backoff) LastError() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastError
}

// LastDelay returns the last delay used.
func (b *Backoff) LastDelay() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastDelay
}

// AdaptiveBackoff provides adaptive rate limiting that adjusts based on error rates.
type AdaptiveBackoff struct {
	*Backoff

	mu sync.RWMutex

	// Adaptive parameters
	successStreak    int
	failureStreak    int
	currentMultiplier float64

	// Thresholds
	successThreshold int     // Successes needed to reduce delay
	failureThreshold int     // Failures needed to increase delay
	minMultiplier    float64 // Minimum multiplier
	maxMultiplier    float64 // Maximum multiplier
	adaptRate        float64 // How fast to adapt (0.0-1.0)
}

// AdaptiveBackoffConfig extends BackoffConfig with adaptive parameters.
type AdaptiveBackoffConfig struct {
	*BackoffConfig

	SuccessThreshold int
	FailureThreshold int
	MinMultiplier    float64
	MaxMultiplier    float64
	AdaptRate        float64
}

// DefaultAdaptiveBackoffConfig returns sensible defaults for adaptive backoff.
func DefaultAdaptiveBackoffConfig() *AdaptiveBackoffConfig {
	return &AdaptiveBackoffConfig{
		BackoffConfig:    DefaultBackoffConfig(),
		SuccessThreshold: 5,
		FailureThreshold: 3,
		MinMultiplier:    1.5,
		MaxMultiplier:    4.0,
		AdaptRate:        0.2,
	}
}

// NewAdaptiveBackoff creates a new adaptive backoff instance.
func NewAdaptiveBackoff(config *AdaptiveBackoffConfig, logger *zap.Logger) *AdaptiveBackoff {
	if config == nil {
		config = DefaultAdaptiveBackoffConfig()
	}

	return &AdaptiveBackoff{
		Backoff:           NewBackoff(config.BackoffConfig, logger),
		successThreshold:  config.SuccessThreshold,
		failureThreshold:  config.FailureThreshold,
		minMultiplier:     config.MinMultiplier,
		maxMultiplier:     config.MaxMultiplier,
		adaptRate:         config.AdaptRate,
		currentMultiplier: config.BackoffConfig.Multiplier,
	}
}

// RecordSuccess records a successful operation and potentially reduces backoff.
func (ab *AdaptiveBackoff) RecordSuccess() {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.successStreak++
	ab.failureStreak = 0

	if ab.successStreak >= ab.successThreshold {
		// Reduce multiplier
		newMultiplier := ab.currentMultiplier * (1 - ab.adaptRate)
		if newMultiplier < ab.minMultiplier {
			newMultiplier = ab.minMultiplier
		}
		if newMultiplier != ab.currentMultiplier {
			ab.logger.Debug("adaptive backoff reduced",
				zap.Float64("old_multiplier", ab.currentMultiplier),
				zap.Float64("new_multiplier", newMultiplier),
			)
			ab.currentMultiplier = newMultiplier
			ab.config.Multiplier = newMultiplier
		}
		ab.successStreak = 0
	}
}

// RecordFailure records a failed operation and potentially increases backoff.
func (ab *AdaptiveBackoff) RecordFailure() {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.failureStreak++
	ab.successStreak = 0

	if ab.failureStreak >= ab.failureThreshold {
		// Increase multiplier
		newMultiplier := ab.currentMultiplier * (1 + ab.adaptRate)
		if newMultiplier > ab.maxMultiplier {
			newMultiplier = ab.maxMultiplier
		}
		if newMultiplier != ab.currentMultiplier {
			ab.logger.Debug("adaptive backoff increased",
				zap.Float64("old_multiplier", ab.currentMultiplier),
				zap.Float64("new_multiplier", newMultiplier),
			)
			ab.currentMultiplier = newMultiplier
			ab.config.Multiplier = newMultiplier
		}
		ab.failureStreak = 0
	}
}

// CurrentMultiplier returns the current adaptive multiplier.
func (ab *AdaptiveBackoff) CurrentMultiplier() float64 {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.currentMultiplier
}
