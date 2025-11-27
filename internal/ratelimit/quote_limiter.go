// Package ratelimit provides rate limiting functionality for cost control.
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QuoteLimiter provides rate limiting for quote generation to control API costs.
type QuoteLimiter struct {
	mu sync.RWMutex

	// Configuration
	maxRequestsPerMinute int
	maxRequestsPerHour   int
	maxRequestsPerDay    int
	maxConcurrent        int

	// State
	minuteBucket  *tokenBucket
	hourBucket    *tokenBucket
	dayBucket     *tokenBucket
	currentActive int

	// Metrics
	totalRequests   int64
	totalRejected   int64
	lastRejectedAt  time.Time
	rejectionReason string

	logger *zap.Logger
}

// QuoteLimiterConfig holds configuration for the quote limiter.
type QuoteLimiterConfig struct {
	MaxRequestsPerMinute int
	MaxRequestsPerHour   int
	MaxRequestsPerDay    int
	MaxConcurrent        int
}

// DefaultQuoteLimiterConfig returns sensible defaults for cost control.
func DefaultQuoteLimiterConfig() *QuoteLimiterConfig {
	return &QuoteLimiterConfig{
		MaxRequestsPerMinute: 10,   // 10 quotes per minute
		MaxRequestsPerHour:   100,  // 100 quotes per hour
		MaxRequestsPerDay:    500,  // 500 quotes per day
		MaxConcurrent:        5,    // 5 concurrent generations
	}
}

// NewQuoteLimiter creates a new quote rate limiter.
func NewQuoteLimiter(cfg *QuoteLimiterConfig, logger *zap.Logger) *QuoteLimiter {
	if cfg == nil {
		cfg = DefaultQuoteLimiterConfig()
	}

	now := time.Now()
	return &QuoteLimiter{
		maxRequestsPerMinute: cfg.MaxRequestsPerMinute,
		maxRequestsPerHour:   cfg.MaxRequestsPerHour,
		maxRequestsPerDay:    cfg.MaxRequestsPerDay,
		maxConcurrent:        cfg.MaxConcurrent,
		minuteBucket:         newTokenBucket(cfg.MaxRequestsPerMinute, time.Minute, now),
		hourBucket:           newTokenBucket(cfg.MaxRequestsPerHour, time.Hour, now),
		dayBucket:            newTokenBucket(cfg.MaxRequestsPerDay, 24*time.Hour, now),
		logger:               logger,
	}
}

// Errors for rate limiting.
var (
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrMinuteLimitExceeded   = errors.New("minute rate limit exceeded")
	ErrHourLimitExceeded     = errors.New("hour rate limit exceeded")
	ErrDayLimitExceeded      = errors.New("day rate limit exceeded")
	ErrConcurrentLimitExceeded = errors.New("concurrent request limit exceeded")
)

// Acquire attempts to acquire a rate limit slot for quote generation.
// Returns nil if successful, or an error if rate limited.
func (ql *QuoteLimiter) Acquire(ctx context.Context) error {
	ql.mu.Lock()
	defer ql.mu.Unlock()

	ql.totalRequests++
	now := time.Now()

	// Check concurrent limit
	if ql.currentActive >= ql.maxConcurrent {
		ql.reject("concurrent limit", now)
		return ErrConcurrentLimitExceeded
	}

	// Check minute limit
	if !ql.minuteBucket.tryAcquire(now) {
		ql.reject("minute limit", now)
		return ErrMinuteLimitExceeded
	}

	// Check hour limit
	if !ql.hourBucket.tryAcquire(now) {
		// Rollback minute bucket
		ql.minuteBucket.release()
		ql.reject("hour limit", now)
		return ErrHourLimitExceeded
	}

	// Check day limit
	if !ql.dayBucket.tryAcquire(now) {
		// Rollback minute and hour buckets
		ql.minuteBucket.release()
		ql.hourBucket.release()
		ql.reject("day limit", now)
		return ErrDayLimitExceeded
	}

	// All checks passed
	ql.currentActive++

	ql.logger.Debug("quote rate limit acquired",
		zap.Int("active", ql.currentActive),
		zap.Int("minute_remaining", ql.minuteBucket.remaining()),
		zap.Int("hour_remaining", ql.hourBucket.remaining()),
		zap.Int("day_remaining", ql.dayBucket.remaining()),
	)

	return nil
}

// Release releases a rate limit slot after quote generation completes.
func (ql *QuoteLimiter) Release() {
	ql.mu.Lock()
	defer ql.mu.Unlock()

	if ql.currentActive > 0 {
		ql.currentActive--
	}

	ql.logger.Debug("quote rate limit released",
		zap.Int("active", ql.currentActive),
	)
}

// Wait blocks until a rate limit slot is available or context is canceled.
func (ql *QuoteLimiter) Wait(ctx context.Context) error {
	// Try to acquire immediately
	if err := ql.Acquire(ctx); err == nil {
		return nil
	}

	// Poll for availability
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := ql.Acquire(ctx); err == nil {
				return nil
			}
		}
	}
}

// reject records a rejection.
func (ql *QuoteLimiter) reject(reason string, t time.Time) {
	ql.totalRejected++
	ql.lastRejectedAt = t
	ql.rejectionReason = reason

	ql.logger.Warn("quote rate limit exceeded",
		zap.String("reason", reason),
		zap.Int64("total_rejected", ql.totalRejected),
	)
}

// Stats returns current rate limiter statistics.
func (ql *QuoteLimiter) Stats() QuoteLimiterStats {
	ql.mu.RLock()
	defer ql.mu.RUnlock()

	now := time.Now()
	return QuoteLimiterStats{
		CurrentActive:     ql.currentActive,
		MaxConcurrent:     ql.maxConcurrent,
		MinuteRemaining:   ql.minuteBucket.remaining(),
		MinuteMax:         ql.maxRequestsPerMinute,
		HourRemaining:     ql.hourBucket.remaining(),
		HourMax:           ql.maxRequestsPerHour,
		DayRemaining:      ql.dayBucket.remaining(),
		DayMax:            ql.maxRequestsPerDay,
		TotalRequests:     ql.totalRequests,
		TotalRejected:     ql.totalRejected,
		LastRejectedAt:    ql.lastRejectedAt,
		LastRejectionReason: ql.rejectionReason,
		MinuteResetIn:     ql.minuteBucket.resetIn(now),
		HourResetIn:       ql.hourBucket.resetIn(now),
		DayResetIn:        ql.dayBucket.resetIn(now),
	}
}

// QuoteLimiterStats holds statistics about the rate limiter.
type QuoteLimiterStats struct {
	CurrentActive       int           `json:"current_active"`
	MaxConcurrent       int           `json:"max_concurrent"`
	MinuteRemaining     int           `json:"minute_remaining"`
	MinuteMax           int           `json:"minute_max"`
	HourRemaining       int           `json:"hour_remaining"`
	HourMax             int           `json:"hour_max"`
	DayRemaining        int           `json:"day_remaining"`
	DayMax              int           `json:"day_max"`
	TotalRequests       int64         `json:"total_requests"`
	TotalRejected       int64         `json:"total_rejected"`
	LastRejectedAt      time.Time     `json:"last_rejected_at,omitempty"`
	LastRejectionReason string        `json:"last_rejection_reason,omitempty"`
	MinuteResetIn       time.Duration `json:"minute_reset_in"`
	HourResetIn         time.Duration `json:"hour_reset_in"`
	DayResetIn          time.Duration `json:"day_reset_in"`
}

// tokenBucket is a simple sliding window token bucket implementation.
type tokenBucket struct {
	max       int
	period    time.Duration
	tokens    int
	lastReset time.Time
}

func newTokenBucket(maxTokens int, period time.Duration, now time.Time) *tokenBucket {
	return &tokenBucket{
		max:       maxTokens,
		period:    period,
		tokens:    maxTokens,
		lastReset: now,
	}
}

func (b *tokenBucket) tryAcquire(now time.Time) bool {
	b.refill(now)
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (b *tokenBucket) release() {
	if b.tokens < b.max {
		b.tokens++
	}
}

func (b *tokenBucket) remaining() int {
	return b.tokens
}

func (b *tokenBucket) resetIn(now time.Time) time.Duration {
	elapsed := now.Sub(b.lastReset)
	remaining := b.period - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (b *tokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastReset)
	if elapsed >= b.period {
		// Reset the bucket
		b.tokens = b.max
		b.lastReset = now
	}
}
