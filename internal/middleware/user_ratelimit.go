package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/metrics"
	"github.com/jkindrix/quickquote/internal/ratelimit"
)

// userContextKey is the context key for the authenticated user ID.
type userContextKey struct{}

// UserIDFromContext extracts the user ID from the request context.
// This should be set by the authentication middleware.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userContextKey{}).(uuid.UUID)
	return id, ok
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userContextKey{}, userID)
}

// UserRateLimit returns HTTP middleware that enforces per-user rate limits.
// It requires the user to be authenticated (user ID in context).
// Unauthenticated requests pass through without rate limiting.
func UserRateLimit(limiter *ratelimit.UserRateLimiter, logger *zap.Logger, metricsCollector *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by auth middleware)
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				// No user ID - not authenticated, skip rate limiting
				// (unauthenticated endpoints should use IP-based limiting)
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			if err := limiter.Allow(r.Context(), userID); err != nil {
				logger.Warn("user rate limit exceeded",
					zap.String("user_id", userID.String()),
					zap.String("path", r.URL.Path),
					zap.Error(err),
				)
				if metricsCollector != nil {
					metricsCollector.RateLimitHitsTotal.WithLabelValues("user").Inc()
				}

				// Get stats for headers
				stats := limiter.Stats(r.Context(), userID)

				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit-Minute", strconv.Itoa(stats.MinuteMax))
				w.Header().Set("X-RateLimit-Remaining-Minute", strconv.Itoa(stats.MinuteRemaining))
				w.Header().Set("X-RateLimit-Limit-Hour", strconv.Itoa(stats.HourMax))
				w.Header().Set("X-RateLimit-Remaining-Hour", strconv.Itoa(stats.HourRemaining))
				w.Header().Set("X-RateLimit-Limit-Day", strconv.Itoa(stats.DayMax))
				w.Header().Set("X-RateLimit-Remaining-Day", strconv.Itoa(stats.DayRemaining))
				w.Header().Set("Retry-After", "60")

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Get stats for response headers
			stats := limiter.Stats(r.Context(), userID)
			w.Header().Set("X-RateLimit-Remaining-Minute", strconv.Itoa(stats.MinuteRemaining))
			w.Header().Set("X-RateLimit-Remaining-Hour", strconv.Itoa(stats.HourRemaining))
			w.Header().Set("X-RateLimit-Remaining-Day", strconv.Itoa(stats.DayRemaining))

			next.ServeHTTP(w, r)
		})
	}
}
