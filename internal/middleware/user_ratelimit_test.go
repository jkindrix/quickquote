package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/ratelimit"
)

func TestUserRateLimit_AllowsAuthenticatedRequests(t *testing.T) {
	logger := zap.NewNop()
	config := ratelimit.UserRateLimitConfig{
		MaxRequestsPerMinute: 10,
		MaxRequestsPerHour:   100,
		MaxRequestsPerDay:    1000,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}
	limiter := ratelimit.NewUserRateLimiter(config, nil, logger)

	handler := UserRateLimit(limiter, logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req = req.WithContext(WithUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Check rate limit headers are set
	if rec.Header().Get("X-RateLimit-Remaining-Minute") == "" {
		t.Error("expected X-RateLimit-Remaining-Minute header to be set")
	}
}

func TestUserRateLimit_BlocksExceededLimit(t *testing.T) {
	logger := zap.NewNop()
	config := ratelimit.UserRateLimitConfig{
		MaxRequestsPerMinute: 2,
		MaxRequestsPerHour:   100,
		MaxRequestsPerDay:    1000,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}
	limiter := ratelimit.NewUserRateLimiter(config, nil, logger)

	handler := UserRateLimit(limiter, logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userID := uuid.New()

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req = req.WithContext(WithUserID(req.Context(), userID))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req = req.WithContext(WithUserID(req.Context(), userID))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	// Check Retry-After header
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestUserRateLimit_SkipsUnauthenticatedRequests(t *testing.T) {
	logger := zap.NewNop()
	config := ratelimit.UserRateLimitConfig{
		MaxRequestsPerMinute: 1, // Very low limit
		MaxRequestsPerHour:   1,
		MaxRequestsPerDay:    1,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}
	limiter := ratelimit.NewUserRateLimiter(config, nil, logger)

	callCount := 0
	handler := UserRateLimit(limiter, logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	// Make multiple requests without user ID
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200 for unauthenticated request, got %d", i+1, rec.Code)
		}
	}

	if callCount != 5 {
		t.Errorf("expected all 5 requests to pass through, got %d", callCount)
	}
}

func TestUserRateLimit_DifferentUsersIndependent(t *testing.T) {
	logger := zap.NewNop()
	config := ratelimit.UserRateLimitConfig{
		MaxRequestsPerMinute: 2,
		MaxRequestsPerHour:   100,
		MaxRequestsPerDay:    1000,
		CleanupInterval:      time.Hour,
		StaleUserThreshold:   time.Hour,
	}
	limiter := ratelimit.NewUserRateLimiter(config, nil, logger)

	handler := UserRateLimit(limiter, logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	user1 := uuid.New()
	user2 := uuid.New()

	// Exhaust user1's limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req = req.WithContext(WithUserID(req.Context(), user1))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// User1 should be blocked
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req = req.WithContext(WithUserID(req.Context(), user1))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("user1 should be rate limited, got status %d", rec.Code)
	}

	// User2 should still be allowed
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req = req.WithContext(WithUserID(req.Context(), user2))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("user2 should be allowed, got status %d", rec.Code)
	}
}

func TestUserIDFromContext(t *testing.T) {
	userID := uuid.New()

	// Test with user ID set
	ctx := WithUserID(context.Background(), userID)
	gotID, ok := UserIDFromContext(ctx)
	if !ok {
		t.Error("expected user ID to be found in context")
	}
	if gotID != userID {
		t.Errorf("expected user ID %s, got %s", userID, gotID)
	}

	// Test without user ID
	ctx = context.Background()
	_, ok = UserIDFromContext(ctx)
	if ok {
		t.Error("expected no user ID in empty context")
	}
}
