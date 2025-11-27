// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter implements a token bucket rate limiter per IP address.
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	rate     int           // requests per window
	window   time.Duration // time window
	logger   *zap.Logger
}

type visitor struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rate int, window time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
		logger:   logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes stale visitors periodically.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastReset) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset tokens if window has passed
	if now.Sub(v.lastReset) >= rl.window {
		v.tokens = rl.rate - 1
		v.lastReset = now
		return true
	}

	// Check if tokens available
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// remaining returns the number of remaining requests for an IP.
func (rl *RateLimiter) remaining(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	v, exists := rl.visitors[ip]
	if !exists {
		return rl.rate
	}

	now := time.Now()
	if now.Sub(v.lastReset) >= rl.window {
		return rl.rate
	}

	return v.tokens
}

// RateLimit returns HTTP middleware that rate limits requests.
func RateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !rl.allow(ip) {
				rl.logger.Warn("rate limit exceeded",
					zap.String("ip", ip),
					zap.String("path", r.URL.Path),
				)
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rl.remaining(ip)))

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from a request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr - use net.SplitHostPort for proper IPv4/IPv6 handling
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, return RemoteAddr as-is (may not have port)
		return r.RemoteAddr
	}
	return host
}

// LoginRateLimiter provides stricter rate limiting for login attempts.
type LoginRateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*loginAttempts
	logger   *zap.Logger
}

type loginAttempts struct {
	count     int
	firstTry  time.Time
	blockedAt time.Time
}

const (
	maxLoginAttempts = 5
	loginWindow      = 15 * time.Minute
	blockDuration    = 30 * time.Minute
)

// NewLoginRateLimiter creates a new login rate limiter.
func NewLoginRateLimiter(logger *zap.Logger) *LoginRateLimiter {
	lrl := &LoginRateLimiter{
		attempts: make(map[string]*loginAttempts),
		logger:   logger,
	}

	// Start cleanup goroutine
	go lrl.cleanup()

	return lrl
}

// cleanup removes stale entries periodically.
func (lrl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(loginWindow)
	defer ticker.Stop()

	for range ticker.C {
		lrl.mu.Lock()
		now := time.Now()
		for key, a := range lrl.attempts {
			// Remove if blocked and block expired, or if window expired
			if (!a.blockedAt.IsZero() && now.Sub(a.blockedAt) > blockDuration) ||
				(a.blockedAt.IsZero() && now.Sub(a.firstTry) > loginWindow) {
				delete(lrl.attempts, key)
			}
		}
		lrl.mu.Unlock()
	}
}

// Check checks if a login attempt is allowed and records it.
// Returns true if allowed, false if blocked.
func (lrl *LoginRateLimiter) Check(ip, email string) bool {
	key := ip + ":" + email

	lrl.mu.Lock()
	defer lrl.mu.Unlock()

	now := time.Now()

	a, exists := lrl.attempts[key]
	if !exists {
		lrl.attempts[key] = &loginAttempts{
			count:    1,
			firstTry: now,
		}
		return true
	}

	// Check if currently blocked
	if !a.blockedAt.IsZero() {
		if now.Sub(a.blockedAt) < blockDuration {
			lrl.logger.Warn("login blocked",
				zap.String("ip", ip),
				zap.String("email", email),
				zap.Duration("remaining", blockDuration-now.Sub(a.blockedAt)),
			)
			return false
		}
		// Block expired, reset
		a.count = 1
		a.firstTry = now
		a.blockedAt = time.Time{}
		return true
	}

	// Check if window expired
	if now.Sub(a.firstTry) > loginWindow {
		a.count = 1
		a.firstTry = now
		return true
	}

	// Increment attempts
	a.count++

	// Check if should block
	if a.count > maxLoginAttempts {
		a.blockedAt = now
		lrl.logger.Warn("login rate limit exceeded, blocking",
			zap.String("ip", ip),
			zap.String("email", email),
			zap.Int("attempts", a.count),
		)
		return false
	}

	return true
}

// RecordSuccess records a successful login and resets the counter.
func (lrl *LoginRateLimiter) RecordSuccess(ip, email string) {
	key := ip + ":" + email

	lrl.mu.Lock()
	defer lrl.mu.Unlock()

	delete(lrl.attempts, key)
}

// RemainingAttempts returns the number of remaining login attempts.
func (lrl *LoginRateLimiter) RemainingAttempts(ip, email string) int {
	key := ip + ":" + email

	lrl.mu.RLock()
	defer lrl.mu.RUnlock()

	a, exists := lrl.attempts[key]
	if !exists {
		return maxLoginAttempts
	}

	if !a.blockedAt.IsZero() {
		return 0
	}

	now := time.Now()
	if now.Sub(a.firstTry) > loginWindow {
		return maxLoginAttempts
	}

	remaining := maxLoginAttempts - a.count
	if remaining < 0 {
		return 0
	}
	return remaining
}
