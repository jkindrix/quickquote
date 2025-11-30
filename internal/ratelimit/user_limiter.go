// Package ratelimit provides rate limiting functionality for cost control.
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserRateLimiter provides per-user rate limiting for API operations.
// Unlike IP-based limiting, this tracks authenticated users by their ID.
type UserRateLimiter struct {
	mu sync.RWMutex

	// Configuration
	config UserRateLimitConfig

	// In-memory tracking (fallback when no repository)
	buckets map[uuid.UUID]*userBuckets

	// Optional persistence
	repo UserRateLimitRepository

	logger *zap.Logger
}

// userBuckets holds rate limit state for a single user.
type userBuckets struct {
	minuteBucket *tokenBucket
	hourBucket   *tokenBucket
	dayBucket    *tokenBucket
	lastAccess   time.Time
}

// UserRateLimitConfig holds configuration for per-user rate limiting.
type UserRateLimitConfig struct {
	MaxRequestsPerMinute int           `json:"max_requests_per_minute"`
	MaxRequestsPerHour   int           `json:"max_requests_per_hour"`
	MaxRequestsPerDay    int           `json:"max_requests_per_day"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
	StaleUserThreshold   time.Duration `json:"stale_user_threshold"`
}

// DefaultUserRateLimitConfig returns sensible defaults.
func DefaultUserRateLimitConfig() UserRateLimitConfig {
	return UserRateLimitConfig{
		MaxRequestsPerMinute: 60,               // 1 request per second average
		MaxRequestsPerHour:   300,              // 5 requests per minute average
		MaxRequestsPerDay:    1000,             // ~42 requests per hour average
		CleanupInterval:      5 * time.Minute,  // Clean stale entries every 5 min
		StaleUserThreshold:   30 * time.Minute, // Remove users inactive for 30 min
	}
}

// UserRateLimitRepository provides persistence for rate limit state.
type UserRateLimitRepository interface {
	// IncrementRequestCount atomically increments the request count for a user.
	// Returns the new count for the specified window.
	IncrementRequestCount(ctx context.Context, userID uuid.UUID, window string) (int, error)

	// GetRequestCount returns the current count for a user in a window.
	GetRequestCount(ctx context.Context, userID uuid.UUID, window string) (int, error)

	// ResetExpiredWindows resets counts for windows that have expired.
	ResetExpiredWindows(ctx context.Context) error
}

// UserRateLimitEntry represents a rate limit record in the database.
type UserRateLimitEntry struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Window    string    `json:"window"` // "minute", "hour", "day"
	Count     int       `json:"count"`
	WindowEnd time.Time `json:"window_end"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUserRateLimiter creates a new per-user rate limiter.
func NewUserRateLimiter(config UserRateLimitConfig, repo UserRateLimitRepository, logger *zap.Logger) *UserRateLimiter {
	rl := &UserRateLimiter{
		config:  config,
		buckets: make(map[uuid.UUID]*userBuckets),
		repo:    repo,
		logger:  logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Errors for user rate limiting.
var (
	ErrUserRateLimitExceeded = errors.New("user rate limit exceeded")
	ErrUserMinuteLimitExceeded = errors.New("user minute rate limit exceeded")
	ErrUserHourLimitExceeded = errors.New("user hour rate limit exceeded")
	ErrUserDayLimitExceeded = errors.New("user day rate limit exceeded")
)

// Allow checks if a request from the user is allowed.
// Returns nil if allowed, or an error describing which limit was exceeded.
func (rl *UserRateLimiter) Allow(ctx context.Context, userID uuid.UUID) error {
	// If we have a repository, use it for distributed rate limiting
	if rl.repo != nil {
		return rl.allowWithRepo(ctx, userID)
	}

	// Fall back to in-memory rate limiting
	return rl.allowInMemory(userID)
}

// allowWithRepo uses the database for rate limit tracking.
func (rl *UserRateLimiter) allowWithRepo(ctx context.Context, userID uuid.UUID) error {
	// Check minute limit
	minuteCount, err := rl.repo.IncrementRequestCount(ctx, userID, "minute")
	if err != nil {
		rl.logger.Error("failed to increment minute count", zap.Error(err))
		// Fall back to allowing the request on error
		return nil
	}
	if minuteCount > rl.config.MaxRequestsPerMinute {
		rl.logger.Warn("user minute rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("count", minuteCount),
			zap.Int("limit", rl.config.MaxRequestsPerMinute),
		)
		return ErrUserMinuteLimitExceeded
	}

	// Check hour limit
	hourCount, err := rl.repo.IncrementRequestCount(ctx, userID, "hour")
	if err != nil {
		rl.logger.Error("failed to increment hour count", zap.Error(err))
		return nil
	}
	if hourCount > rl.config.MaxRequestsPerHour {
		rl.logger.Warn("user hour rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("count", hourCount),
			zap.Int("limit", rl.config.MaxRequestsPerHour),
		)
		return ErrUserHourLimitExceeded
	}

	// Check day limit
	dayCount, err := rl.repo.IncrementRequestCount(ctx, userID, "day")
	if err != nil {
		rl.logger.Error("failed to increment day count", zap.Error(err))
		return nil
	}
	if dayCount > rl.config.MaxRequestsPerDay {
		rl.logger.Warn("user day rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("count", dayCount),
			zap.Int("limit", rl.config.MaxRequestsPerDay),
		)
		return ErrUserDayLimitExceeded
	}

	return nil
}

// allowInMemory uses in-memory tracking when no repository is available.
func (rl *UserRateLimiter) allowInMemory(userID uuid.UUID) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	buckets, exists := rl.buckets[userID]
	if !exists {
		// Create new buckets for this user
		buckets = &userBuckets{
			minuteBucket: newTokenBucket(rl.config.MaxRequestsPerMinute, time.Minute, now),
			hourBucket:   newTokenBucket(rl.config.MaxRequestsPerHour, time.Hour, now),
			dayBucket:    newTokenBucket(rl.config.MaxRequestsPerDay, 24*time.Hour, now),
			lastAccess:   now,
		}
		rl.buckets[userID] = buckets
	}

	buckets.lastAccess = now

	// Check minute limit
	if !buckets.minuteBucket.tryAcquire(now) {
		rl.logger.Warn("user minute rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("limit", rl.config.MaxRequestsPerMinute),
		)
		return ErrUserMinuteLimitExceeded
	}

	// Check hour limit
	if !buckets.hourBucket.tryAcquire(now) {
		buckets.minuteBucket.release()
		rl.logger.Warn("user hour rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("limit", rl.config.MaxRequestsPerHour),
		)
		return ErrUserHourLimitExceeded
	}

	// Check day limit
	if !buckets.dayBucket.tryAcquire(now) {
		buckets.minuteBucket.release()
		buckets.hourBucket.release()
		rl.logger.Warn("user day rate limit exceeded",
			zap.String("user_id", userID.String()),
			zap.Int("limit", rl.config.MaxRequestsPerDay),
		)
		return ErrUserDayLimitExceeded
	}

	return nil
}

// Stats returns rate limit statistics for a user.
func (rl *UserRateLimiter) Stats(ctx context.Context, userID uuid.UUID) UserRateLimitStats {
	// If we have a repository, get counts from there
	if rl.repo != nil {
		minuteCount, _ := rl.repo.GetRequestCount(ctx, userID, "minute")
		hourCount, _ := rl.repo.GetRequestCount(ctx, userID, "hour")
		dayCount, _ := rl.repo.GetRequestCount(ctx, userID, "day")

		return UserRateLimitStats{
			UserID:            userID,
			MinuteRemaining:   max(0, rl.config.MaxRequestsPerMinute-minuteCount),
			MinuteMax:         rl.config.MaxRequestsPerMinute,
			HourRemaining:     max(0, rl.config.MaxRequestsPerHour-hourCount),
			HourMax:           rl.config.MaxRequestsPerHour,
			DayRemaining:      max(0, rl.config.MaxRequestsPerDay-dayCount),
			DayMax:            rl.config.MaxRequestsPerDay,
		}
	}

	// Fall back to in-memory stats
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	buckets, exists := rl.buckets[userID]
	if !exists {
		return UserRateLimitStats{
			UserID:          userID,
			MinuteRemaining: rl.config.MaxRequestsPerMinute,
			MinuteMax:       rl.config.MaxRequestsPerMinute,
			HourRemaining:   rl.config.MaxRequestsPerHour,
			HourMax:         rl.config.MaxRequestsPerHour,
			DayRemaining:    rl.config.MaxRequestsPerDay,
			DayMax:          rl.config.MaxRequestsPerDay,
		}
	}

	return UserRateLimitStats{
		UserID:          userID,
		MinuteRemaining: buckets.minuteBucket.remaining(),
		MinuteMax:       rl.config.MaxRequestsPerMinute,
		HourRemaining:   buckets.hourBucket.remaining(),
		HourMax:         rl.config.MaxRequestsPerHour,
		DayRemaining:    buckets.dayBucket.remaining(),
		DayMax:          rl.config.MaxRequestsPerDay,
	}
}

// UserRateLimitStats holds rate limit statistics for a user.
type UserRateLimitStats struct {
	UserID          uuid.UUID `json:"user_id"`
	MinuteRemaining int       `json:"minute_remaining"`
	MinuteMax       int       `json:"minute_max"`
	HourRemaining   int       `json:"hour_remaining"`
	HourMax         int       `json:"hour_max"`
	DayRemaining    int       `json:"day_remaining"`
	DayMax          int       `json:"day_max"`
}

// cleanup removes stale user entries periodically.
func (rl *UserRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanupStaleUsers()

		// Also clean up database if we have a repository
		if rl.repo != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := rl.repo.ResetExpiredWindows(ctx); err != nil {
				rl.logger.Error("failed to reset expired windows", zap.Error(err))
			}
			cancel()
		}
	}
}

// cleanupStaleUsers removes users who haven't made requests recently.
func (rl *UserRateLimiter) cleanupStaleUsers() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, buckets := range rl.buckets {
		if now.Sub(buckets.lastAccess) > rl.config.StaleUserThreshold {
			delete(rl.buckets, userID)
		}
	}
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
