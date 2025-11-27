package ratelimit

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestLimiter() *QuoteLimiter {
	logger := zap.NewNop()
	cfg := &QuoteLimiterConfig{
		MaxRequestsPerMinute: 5,
		MaxRequestsPerHour:   20,
		MaxRequestsPerDay:    50,
		MaxConcurrent:        2,
	}
	return NewQuoteLimiter(cfg, logger)
}

func TestQuoteLimiter_Acquire_Success(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	err := limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("Acquire() error = %v, want nil", err)
	}

	stats := limiter.Stats()
	if stats.CurrentActive != 1 {
		t.Errorf("CurrentActive = %d, want 1", stats.CurrentActive)
	}
}

func TestQuoteLimiter_Release(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	limiter.Acquire(ctx)
	limiter.Release()

	stats := limiter.Stats()
	if stats.CurrentActive != 0 {
		t.Errorf("CurrentActive = %d, want 0", stats.CurrentActive)
	}
}

func TestQuoteLimiter_ConcurrentLimit(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	// Acquire up to max concurrent
	for i := 0; i < 2; i++ {
		if err := limiter.Acquire(ctx); err != nil {
			t.Fatalf("Acquire() %d error = %v", i, err)
		}
	}

	// Next should fail
	err := limiter.Acquire(ctx)
	if err != ErrConcurrentLimitExceeded {
		t.Errorf("Acquire() error = %v, want %v", err, ErrConcurrentLimitExceeded)
	}

	// Release one and try again
	limiter.Release()
	err = limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("Acquire() after release error = %v, want nil", err)
	}
}

func TestQuoteLimiter_MinuteLimit(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	// Acquire and release to stay within concurrent limit
	for i := 0; i < 5; i++ {
		if err := limiter.Acquire(ctx); err != nil {
			t.Fatalf("Acquire() %d error = %v", i, err)
		}
		limiter.Release()
	}

	// Next should fail due to minute limit
	err := limiter.Acquire(ctx)
	if err != ErrMinuteLimitExceeded {
		t.Errorf("Acquire() error = %v, want %v", err, ErrMinuteLimitExceeded)
	}
}

func TestQuoteLimiter_Stats(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	// Initial stats
	stats := limiter.Stats()
	if stats.MinuteRemaining != 5 {
		t.Errorf("MinuteRemaining = %d, want 5", stats.MinuteRemaining)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}

	// After one acquisition
	limiter.Acquire(ctx)
	stats = limiter.Stats()
	if stats.MinuteRemaining != 4 {
		t.Errorf("MinuteRemaining = %d, want 4", stats.MinuteRemaining)
	}
	if stats.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", stats.TotalRequests)
	}
}

func TestQuoteLimiter_RejectionStats(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	// Exhaust limits
	for i := 0; i < 5; i++ {
		limiter.Acquire(ctx)
		limiter.Release()
	}

	// This should be rejected
	limiter.Acquire(ctx)

	stats := limiter.Stats()
	if stats.TotalRejected != 1 {
		t.Errorf("TotalRejected = %d, want 1", stats.TotalRejected)
	}
	if stats.LastRejectionReason != "minute limit" {
		t.Errorf("LastRejectionReason = %q, want %q", stats.LastRejectionReason, "minute limit")
	}
}

func TestQuoteLimiter_Wait_Immediate(t *testing.T) {
	limiter := newTestLimiter()
	ctx := context.Background()

	// Should succeed immediately
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
}

func TestQuoteLimiter_Wait_ContextCancel(t *testing.T) {
	limiter := newTestLimiter()

	// Exhaust the minute limit
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		limiter.Acquire(ctx)
		limiter.Release()
	}

	// Wait with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := limiter.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want %v", err, context.Canceled)
	}
}

func TestQuoteLimiter_Wait_Timeout(t *testing.T) {
	limiter := newTestLimiter()

	// Exhaust the minute limit
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		limiter.Acquire(ctx)
		limiter.Release()
	}

	// Wait with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	now := time.Now()
	bucket := newTokenBucket(5, time.Minute, now)

	// Use all tokens
	for i := 0; i < 5; i++ {
		if !bucket.tryAcquire(now) {
			t.Fatalf("tryAcquire() failed at %d", i)
		}
	}

	// No more tokens
	if bucket.tryAcquire(now) {
		t.Error("tryAcquire() should fail when empty")
	}

	// After period, should refill
	future := now.Add(time.Minute + time.Second)
	if !bucket.tryAcquire(future) {
		t.Error("tryAcquire() should succeed after refill")
	}
	if bucket.remaining() != 4 {
		t.Errorf("remaining() = %d, want 4", bucket.remaining())
	}
}

func TestTokenBucket_Release(t *testing.T) {
	now := time.Now()
	bucket := newTokenBucket(5, time.Minute, now)

	bucket.tryAcquire(now)
	if bucket.remaining() != 4 {
		t.Errorf("remaining() = %d, want 4", bucket.remaining())
	}

	bucket.release()
	if bucket.remaining() != 5 {
		t.Errorf("remaining() after release = %d, want 5", bucket.remaining())
	}

	// Release when at max should not exceed max
	bucket.release()
	if bucket.remaining() != 5 {
		t.Errorf("remaining() should not exceed max, got %d", bucket.remaining())
	}
}

func TestTokenBucket_ResetIn(t *testing.T) {
	now := time.Now()
	bucket := newTokenBucket(5, time.Minute, now)

	resetIn := bucket.resetIn(now)
	if resetIn != time.Minute {
		t.Errorf("resetIn() = %v, want %v", resetIn, time.Minute)
	}

	// Halfway through
	halfWay := now.Add(30 * time.Second)
	resetIn = bucket.resetIn(halfWay)
	if resetIn != 30*time.Second {
		t.Errorf("resetIn() = %v, want %v", resetIn, 30*time.Second)
	}

	// Past reset
	past := now.Add(2 * time.Minute)
	resetIn = bucket.resetIn(past)
	if resetIn != 0 {
		t.Errorf("resetIn() = %v, want 0", resetIn)
	}
}

func TestDefaultQuoteLimiterConfig(t *testing.T) {
	cfg := DefaultQuoteLimiterConfig()

	if cfg.MaxRequestsPerMinute <= 0 {
		t.Error("MaxRequestsPerMinute should be positive")
	}
	if cfg.MaxRequestsPerHour <= 0 {
		t.Error("MaxRequestsPerHour should be positive")
	}
	if cfg.MaxRequestsPerDay <= 0 {
		t.Error("MaxRequestsPerDay should be positive")
	}
	if cfg.MaxConcurrent <= 0 {
		t.Error("MaxConcurrent should be positive")
	}
}
