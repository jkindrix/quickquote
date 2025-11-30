package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestUserRateLimiter_AllowInMemory(t *testing.T) {
	logger := zap.NewNop()
	config := UserRateLimitConfig{
		MaxRequestsPerMinute: 5,
		MaxRequestsPerHour:   10,
		MaxRequestsPerDay:    20,
		CleanupInterval:      time.Hour, // Long interval to prevent cleanup during test
		StaleUserThreshold:   time.Hour,
	}

	limiter := NewUserRateLimiter(config, nil, logger)
	userID := uuid.New()
	ctx := context.Background()

	// Should allow first 5 requests (minute limit)
	for i := 0; i < 5; i++ {
		if err := limiter.Allow(ctx, userID); err != nil {
			t.Errorf("request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// 6th request should exceed minute limit
	if err := limiter.Allow(ctx, userID); err != ErrUserMinuteLimitExceeded {
		t.Errorf("expected ErrUserMinuteLimitExceeded, got %v", err)
	}
}

func TestUserRateLimiter_DifferentUsers(t *testing.T) {
	logger := zap.NewNop()
	config := UserRateLimitConfig{
		MaxRequestsPerMinute: 3,
		MaxRequestsPerHour:   10,
		MaxRequestsPerDay:    20,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}

	limiter := NewUserRateLimiter(config, nil, logger)
	user1 := uuid.New()
	user2 := uuid.New()
	ctx := context.Background()

	// Use up user1's minute limit
	for i := 0; i < 3; i++ {
		if err := limiter.Allow(ctx, user1); err != nil {
			t.Errorf("user1 request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// user1 should be blocked
	if err := limiter.Allow(ctx, user1); err != ErrUserMinuteLimitExceeded {
		t.Errorf("user1 should be rate limited, got %v", err)
	}

	// user2 should still be allowed
	for i := 0; i < 3; i++ {
		if err := limiter.Allow(ctx, user2); err != nil {
			t.Errorf("user2 request %d should be allowed, got error: %v", i+1, err)
		}
	}
}

func TestUserRateLimiter_Stats(t *testing.T) {
	logger := zap.NewNop()
	config := UserRateLimitConfig{
		MaxRequestsPerMinute: 10,
		MaxRequestsPerHour:   100,
		MaxRequestsPerDay:    1000,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}

	limiter := NewUserRateLimiter(config, nil, logger)
	userID := uuid.New()
	ctx := context.Background()

	// Check initial stats
	stats := limiter.Stats(ctx, userID)
	if stats.MinuteRemaining != 10 {
		t.Errorf("expected 10 minute remaining, got %d", stats.MinuteRemaining)
	}
	if stats.HourRemaining != 100 {
		t.Errorf("expected 100 hour remaining, got %d", stats.HourRemaining)
	}
	if stats.DayRemaining != 1000 {
		t.Errorf("expected 1000 day remaining, got %d", stats.DayRemaining)
	}

	// Make 3 requests
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, userID)
	}

	// Check updated stats
	stats = limiter.Stats(ctx, userID)
	if stats.MinuteRemaining != 7 {
		t.Errorf("expected 7 minute remaining, got %d", stats.MinuteRemaining)
	}
	if stats.HourRemaining != 97 {
		t.Errorf("expected 97 hour remaining, got %d", stats.HourRemaining)
	}
	if stats.DayRemaining != 997 {
		t.Errorf("expected 997 day remaining, got %d", stats.DayRemaining)
	}
}

func TestUserRateLimiter_HourLimit(t *testing.T) {
	logger := zap.NewNop()
	config := UserRateLimitConfig{
		MaxRequestsPerMinute: 100, // High minute limit
		MaxRequestsPerHour:   5,   // Low hour limit
		MaxRequestsPerDay:    100,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}

	limiter := NewUserRateLimiter(config, nil, logger)
	userID := uuid.New()
	ctx := context.Background()

	// Should allow first 5 requests (hour limit)
	for i := 0; i < 5; i++ {
		if err := limiter.Allow(ctx, userID); err != nil {
			t.Errorf("request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// 6th request should exceed hour limit
	if err := limiter.Allow(ctx, userID); err != ErrUserHourLimitExceeded {
		t.Errorf("expected ErrUserHourLimitExceeded, got %v", err)
	}
}

func TestUserRateLimiter_DayLimit(t *testing.T) {
	logger := zap.NewNop()
	config := UserRateLimitConfig{
		MaxRequestsPerMinute: 100, // High minute limit
		MaxRequestsPerHour:   100, // High hour limit
		MaxRequestsPerDay:    5,   // Low day limit
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}

	limiter := NewUserRateLimiter(config, nil, logger)
	userID := uuid.New()
	ctx := context.Background()

	// Should allow first 5 requests (day limit)
	for i := 0; i < 5; i++ {
		if err := limiter.Allow(ctx, userID); err != nil {
			t.Errorf("request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// 6th request should exceed day limit
	if err := limiter.Allow(ctx, userID); err != ErrUserDayLimitExceeded {
		t.Errorf("expected ErrUserDayLimitExceeded, got %v", err)
	}
}

func TestDefaultUserRateLimitConfig(t *testing.T) {
	config := DefaultUserRateLimitConfig()

	if config.MaxRequestsPerMinute != 60 {
		t.Errorf("expected MaxRequestsPerMinute=60, got %d", config.MaxRequestsPerMinute)
	}
	if config.MaxRequestsPerHour != 300 {
		t.Errorf("expected MaxRequestsPerHour=300, got %d", config.MaxRequestsPerHour)
	}
	if config.MaxRequestsPerDay != 1000 {
		t.Errorf("expected MaxRequestsPerDay=1000, got %d", config.MaxRequestsPerDay)
	}
	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("expected CleanupInterval=5m, got %v", config.CleanupInterval)
	}
	if config.StaleUserThreshold != 30*time.Minute {
		t.Errorf("expected StaleUserThreshold=30m, got %v", config.StaleUserThreshold)
	}
}
