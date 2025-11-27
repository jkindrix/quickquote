package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestRateLimiter_Allow(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(5, time.Minute, logger)

	ip := "192.168.1.1"

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		if !rl.allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be blocked
	if rl.allow(ip) {
		t.Error("6th request should be blocked")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(5, time.Minute, logger)

	ip := "192.168.1.1"

	// Initial remaining should be max rate
	if r := rl.remaining(ip); r != 5 {
		t.Errorf("expected 5 remaining, got %d", r)
	}

	// After one request
	rl.allow(ip)
	if r := rl.remaining(ip); r != 4 {
		t.Errorf("expected 4 remaining, got %d", r)
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(2, time.Minute, logger)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Exhaust ip1's rate limit
	rl.allow(ip1)
	rl.allow(ip1)
	if rl.allow(ip1) {
		t.Error("ip1 should be blocked")
	}

	// ip2 should still have full quota
	if !rl.allow(ip2) {
		t.Error("ip2 should be allowed")
	}
}

func TestRateLimit_Middleware(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(2, time.Minute, logger)

	handler := RateLimit(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, rr.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}

	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "10.0.0.2"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := getClientIP(req); got != tt.expected {
				t.Errorf("getClientIP() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestLoginRateLimiter_Check(t *testing.T) {
	logger := zap.NewNop()
	lrl := NewLoginRateLimiter(logger)

	ip := "192.168.1.1"
	email := "test@example.com"

	// First 5 attempts should be allowed
	for i := 0; i < 5; i++ {
		if !lrl.Check(ip, email) {
			t.Errorf("attempt %d should be allowed", i+1)
		}
	}

	// 6th attempt should be blocked
	if lrl.Check(ip, email) {
		t.Error("6th attempt should be blocked")
	}
}

func TestLoginRateLimiter_RemainingAttempts(t *testing.T) {
	logger := zap.NewNop()
	lrl := NewLoginRateLimiter(logger)

	ip := "192.168.1.1"
	email := "test@example.com"

	// Initial should be max
	if r := lrl.RemainingAttempts(ip, email); r != 5 {
		t.Errorf("expected 5 remaining, got %d", r)
	}

	// After one attempt
	lrl.Check(ip, email)
	if r := lrl.RemainingAttempts(ip, email); r != 4 {
		t.Errorf("expected 4 remaining, got %d", r)
	}
}

func TestLoginRateLimiter_RecordSuccess(t *testing.T) {
	logger := zap.NewNop()
	lrl := NewLoginRateLimiter(logger)

	ip := "192.168.1.1"
	email := "test@example.com"

	// Make some attempts
	lrl.Check(ip, email)
	lrl.Check(ip, email)

	if r := lrl.RemainingAttempts(ip, email); r != 3 {
		t.Errorf("expected 3 remaining after 2 attempts, got %d", r)
	}

	// Record success
	lrl.RecordSuccess(ip, email)

	// Should be reset
	if r := lrl.RemainingAttempts(ip, email); r != 5 {
		t.Errorf("expected 5 remaining after success, got %d", r)
	}
}

func TestLoginRateLimiter_DifferentUsers(t *testing.T) {
	logger := zap.NewNop()
	lrl := NewLoginRateLimiter(logger)

	ip := "192.168.1.1"
	email1 := "user1@example.com"
	email2 := "user2@example.com"

	// Exhaust attempts for email1
	for i := 0; i < 6; i++ {
		lrl.Check(ip, email1)
	}

	// email1 should be blocked
	if lrl.Check(ip, email1) {
		t.Error("email1 should be blocked")
	}

	// email2 should still have full quota
	if !lrl.Check(ip, email2) {
		t.Error("email2 should be allowed")
	}
}
