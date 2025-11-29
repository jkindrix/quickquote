package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/repository"
)

const (
	csrfTokenLength  = 32
	csrfCookieName   = "csrf_token"
	csrfHeaderName   = "X-CSRF-Token"
	csrfFormField    = "csrf_token"
	csrfTokenExpiry  = 24 * time.Hour
)

// CSRFRepository interface for CSRF token persistence.
type CSRFRepository interface {
	GetByToken(ctx context.Context, token string) (*repository.CSRFToken, error)
	GetOrCreate(ctx context.Context, sessionID *uuid.UUID, token string, expiry time.Duration) (*repository.CSRFToken, error)
	MarkUsed(ctx context.Context, token string) error
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// CSRFProtection provides CSRF protection middleware.
type CSRFProtection struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // fallback in-memory store
	repo   CSRFRepository       // optional persistent store
	logger *zap.Logger
}

// NewCSRFProtection creates a new CSRF protection middleware (in-memory fallback).
func NewCSRFProtection(logger *zap.Logger) *CSRFProtection {
	csrf := &CSRFProtection{
		tokens: make(map[string]time.Time),
		logger: logger,
	}

	// Start cleanup goroutine
	go csrf.cleanup()

	return csrf
}

// NewCSRFProtectionWithRepo creates CSRF protection with database persistence.
func NewCSRFProtectionWithRepo(repo CSRFRepository, logger *zap.Logger) *CSRFProtection {
	csrf := &CSRFProtection{
		tokens: make(map[string]time.Time),
		repo:   repo,
		logger: logger,
	}

	// Start cleanup goroutine
	go csrf.cleanup()

	return csrf
}

// cleanup removes expired tokens periodically.
func (c *CSRFProtection) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Clean up in-memory tokens
		c.mu.Lock()
		now := time.Now()
		for token, expiry := range c.tokens {
			if now.After(expiry) {
				delete(c.tokens, token)
			}
		}
		c.mu.Unlock()

		// Clean up database tokens if repo is available
		if c.repo != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.repo.DeleteExpired(ctx); err != nil {
				c.logger.Error("failed to delete expired CSRF tokens from database", zap.Error(err))
			}
			cancel()
		}
	}
}

// GenerateToken generates a new CSRF token.
func (c *CSRFProtection) GenerateToken() (string, error) {
	return c.GenerateTokenWithContext(context.Background(), nil)
}

// GenerateTokenForSession generates a new CSRF token, optionally tied to a session.
func (c *CSRFProtection) GenerateTokenForSession(sessionID *uuid.UUID) (string, error) {
	return c.GenerateTokenWithContext(context.Background(), sessionID)
}

// GenerateTokenWithContext generates a new CSRF token using the provided context.
func (c *CSRFProtection) GenerateTokenWithContext(ctx context.Context, sessionID *uuid.UUID) (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(bytes)

	// If we have a repo, persist to database
	if c.repo != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		csrfToken, err := c.repo.GetOrCreate(dbCtx, sessionID, token, csrfTokenExpiry)
		if err != nil {
			c.logger.Error("failed to persist CSRF token", zap.Error(err))
			// Fall back to in-memory storage
			c.mu.Lock()
			c.tokens[token] = time.Now().Add(csrfTokenExpiry)
			c.mu.Unlock()
			return token, nil
		}
		// Return the token from repo (may be existing token for same session)
		return csrfToken.Token, nil
	}

	// Use in-memory storage
	c.mu.Lock()
	c.tokens[token] = time.Now().Add(csrfTokenExpiry)
	c.mu.Unlock()

	return token, nil
}

// ValidateToken checks if a token is valid.
func (c *CSRFProtection) ValidateToken(token string) bool {
	return c.ValidateTokenWithContext(context.Background(), token)
}

// ValidateTokenWithContext checks if a token is valid using the provided context.
func (c *CSRFProtection) ValidateTokenWithContext(ctx context.Context, token string) bool {
	// Try database first if available
	if c.repo != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, err := c.repo.GetByToken(dbCtx, token)
		if err == nil {
			return true
		}
		if !errors.Is(err, repository.ErrNotFound) {
			c.logger.Error("failed to validate CSRF token from database", zap.Error(err))
		}
		// Fall through to check in-memory store
	}

	// Check in-memory store
	c.mu.RLock()
	expiry, exists := c.tokens[token]
	c.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		c.mu.Lock()
		delete(c.tokens, token)
		c.mu.Unlock()
		return false
	}

	return true
}

// InvalidateToken removes a token after use (for one-time tokens).
func (c *CSRFProtection) InvalidateToken(token string) {
	c.InvalidateTokenWithContext(context.Background(), token)
}

// InvalidateTokenWithContext removes a token after use using the provided context.
func (c *CSRFProtection) InvalidateTokenWithContext(ctx context.Context, token string) {
	// Remove from database if available
	if c.repo != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := c.repo.MarkUsed(dbCtx, token); err != nil && !errors.Is(err, repository.ErrNotFound) {
			c.logger.Error("failed to mark CSRF token as used", zap.Error(err))
		}
	}

	// Always remove from in-memory store
	c.mu.Lock()
	delete(c.tokens, token)
	c.mu.Unlock()
}

// Middleware returns the CSRF protection middleware.
// It sets a CSRF cookie and validates it on state-changing requests.
func (c *CSRFProtection) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Skip for safe methods
		if isSafeMethod(r.Method) {
			// Ensure a token exists in cookie for forms to use
			c.ensureTokenCookie(ctx, w, r)
			next.ServeHTTP(w, r)
			return
		}

		// For state-changing methods, validate CSRF token
		cookieToken := c.getTokenFromCookie(r)
		if cookieToken == "" {
			c.logger.Warn("CSRF: missing cookie token",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
			http.Error(w, "Forbidden - CSRF token missing", http.StatusForbidden)
			return
		}

		// Get token from request (header or form)
		requestToken := c.getTokenFromRequest(r)
		if requestToken == "" {
			c.logger.Warn("CSRF: missing request token",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
			http.Error(w, "Forbidden - CSRF token missing", http.StatusForbidden)
			return
		}

		// Compare tokens
		if !c.compareTokens(cookieToken, requestToken) {
			c.logger.Warn("CSRF: token mismatch",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
			http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
			return
		}

		// Validate token is known to us (use request context)
		if !c.ValidateTokenWithContext(ctx, cookieToken) {
			c.logger.Warn("CSRF: token not in store",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
			http.Error(w, "Forbidden - CSRF token expired", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureTokenCookie ensures a CSRF token cookie is set.
func (c *CSRFProtection) ensureTokenCookie(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Check if cookie already exists
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		// Validate it's still valid (use request context)
		if c.ValidateTokenWithContext(ctx, cookie.Value) {
			return
		}
	}

	// Generate new token (use request context)
	token, err := c.GenerateTokenWithContext(ctx, nil)
	if err != nil {
		c.logger.Error("failed to generate CSRF token", zap.Error(err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JavaScript needs to read this for AJAX
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(csrfTokenExpiry.Seconds()),
	})
}

// getTokenFromCookie extracts the CSRF token from the cookie.
func (c *CSRFProtection) getTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getTokenFromRequest extracts the CSRF token from the request.
func (c *CSRFProtection) getTokenFromRequest(r *http.Request) string {
	// Check header first
	if token := r.Header.Get(csrfHeaderName); token != "" {
		return token
	}

	// Check form field
	if err := r.ParseForm(); err == nil {
		if token := r.FormValue(csrfFormField); token != "" {
			return token
		}
	}

	return ""
}

// compareTokens performs a constant-time comparison of tokens.
func (c *CSRFProtection) compareTokens(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// isSafeMethod returns true for methods that don't change state.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// GetToken returns the current CSRF token for a request.
// Use this in templates to get the token value.
func (c *CSRFProtection) GetToken(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SkipPath wraps the middleware to skip CSRF checks for specific paths.
// Paths ending with "/" are treated as prefixes (e.g., "/api/" matches "/api/v1/foo").
func (c *CSRFProtection) SkipPath(paths ...string) func(http.Handler) http.Handler {
	exactPaths := make(map[string]bool)
	var prefixPaths []string

	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			prefixPaths = append(prefixPaths, p)
		} else {
			exactPaths[p] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF for exact path matches
			if exactPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip CSRF for prefix matches
			for _, prefix := range prefixPaths {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			c.Middleware(next).ServeHTTP(w, r)
		})
	}
}
