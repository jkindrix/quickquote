package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/repository"
)

func TestCSRFProtection_GenerateToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	token, err := csrf.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// Tokens should be unique
	token2, _ := csrf.GenerateToken()
	if token == token2 {
		t.Error("GenerateToken() returned duplicate tokens")
	}
}

func TestCSRFProtection_ValidateToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	token, _ := csrf.GenerateToken()

	// Valid token should be accepted
	if !csrf.ValidateToken(token) {
		t.Error("ValidateToken() should return true for valid token")
	}

	// Invalid token should be rejected
	if csrf.ValidateToken("invalid-token") {
		t.Error("ValidateToken() should return false for invalid token")
	}
}

func TestCSRFProtection_InvalidateToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	token, _ := csrf.GenerateToken()

	// Token should be valid initially
	if !csrf.ValidateToken(token) {
		t.Error("token should be valid before invalidation")
	}

	// Invalidate it
	csrf.InvalidateToken(token)

	// Token should be invalid now
	if csrf.ValidateToken(token) {
		t.Error("token should be invalid after invalidation")
	}
}

func TestCSRFProtection_Middleware_SafeMethods(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Safe methods should pass without CSRF token
			if rr.Code != http.StatusOK {
				t.Errorf("expected status %d for %s, got %d", http.StatusOK, method, rr.Code)
			}
		})
	}
}

func TestCSRFProtection_Middleware_UnsafeMethodsBlocked(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	unsafeMethods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Unsafe methods without token should be blocked
			if rr.Code != http.StatusForbidden {
				t.Errorf("expected status %d for %s without token, got %d", http.StatusForbidden, method, rr.Code)
			}
		})
	}
}

func TestCSRFProtection_Middleware_ValidToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Generate token
	token, _ := csrf.GenerateToken()

	// Create POST request with token in form
	form := url.Values{}
	form.Add("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d with valid token, got %d", http.StatusOK, rr.Code)
	}
}

func TestCSRFProtection_Middleware_TokenInHeader(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Generate token
	token, _ := csrf.GenerateToken()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-CSRF-Token", token)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d with valid header token, got %d", http.StatusOK, rr.Code)
	}
}

func TestCSRFProtection_Middleware_MismatchedToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Generate two different tokens
	token1, _ := csrf.GenerateToken()
	token2, _ := csrf.GenerateToken()

	form := url.Values{}
	form.Add("csrf_token", token1)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token2}) // Different token in cookie
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d with mismatched tokens, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestCSRFProtection_SkipPath(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	handler := csrf.SkipPath("/webhook/bland")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Webhook path should be skipped
	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d for skipped path, got %d", http.StatusOK, rr.Code)
	}

	// Other paths should require CSRF
	req2 := httptest.NewRequest(http.MethodPost, "/login", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Errorf("expected status %d for non-skipped path, got %d", http.StatusForbidden, rr2.Code)
	}
}

func TestCSRFProtection_GetToken(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	token, _ := csrf.GenerateToken()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	if got := csrf.GetToken(req); got != token {
		t.Errorf("GetToken() = %q, expected %q", got, token)
	}
}

func TestCSRFProtection_GetToken_NoCookie(t *testing.T) {
	logger := zap.NewNop()
	csrf := NewCSRFProtection(logger)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	if got := csrf.GetToken(req); got != "" {
		t.Errorf("GetToken() = %q, expected empty string", got)
	}
}

func TestIsSafeMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodOptions, true},
		{http.MethodPost, false},
		{http.MethodPut, false},
		{http.MethodDelete, false},
		{http.MethodPatch, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isSafeMethod(tt.method); got != tt.expected {
				t.Errorf("isSafeMethod(%q) = %v, expected %v", tt.method, got, tt.expected)
			}
		})
	}
}

// MockCSRFRepository is a mock implementation of CSRFRepository for testing.
type MockCSRFRepository struct {
	mu     sync.RWMutex
	tokens map[string]*repository.CSRFToken
}

func NewMockCSRFRepository() *MockCSRFRepository {
	return &MockCSRFRepository{
		tokens: make(map[string]*repository.CSRFToken),
	}
}

func (m *MockCSRFRepository) GetByToken(ctx context.Context, token string) (*repository.CSRFToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if t, ok := m.tokens[token]; ok && !t.Used && t.ExpiresAt.After(time.Now()) {
		return t, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockCSRFRepository) GetOrCreate(ctx context.Context, sessionID *uuid.UUID, token string, expiry time.Duration) (*repository.CSRFToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing token for session
	if sessionID != nil {
		for _, t := range m.tokens {
			if t.SessionID != nil && *t.SessionID == *sessionID && !t.Used && t.ExpiresAt.After(time.Now()) {
				return t, nil
			}
		}
	}

	// Create new token
	csrfToken := &repository.CSRFToken{
		ID:        uuid.New(),
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
		Used:      false,
	}
	m.tokens[token] = csrfToken
	return csrfToken, nil
}

func (m *MockCSRFRepository) MarkUsed(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tokens[token]; ok {
		t.Used = true
		return nil
	}
	return repository.ErrNotFound
}

func (m *MockCSRFRepository) Delete(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, token)
	return nil
}

func (m *MockCSRFRepository) DeleteExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for token, t := range m.tokens {
		if t.ExpiresAt.Before(now) {
			delete(m.tokens, token)
		}
	}
	return nil
}

func TestCSRFProtectionWithRepo_GenerateToken(t *testing.T) {
	logger := zap.NewNop()
	repo := NewMockCSRFRepository()
	csrf := NewCSRFProtectionWithRepo(repo, logger)

	token, err := csrf.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// Token should be persisted in repo
	if len(repo.tokens) != 1 {
		t.Errorf("expected 1 token in repo, got %d", len(repo.tokens))
	}
}

func TestCSRFProtectionWithRepo_ValidateToken(t *testing.T) {
	logger := zap.NewNop()
	repo := NewMockCSRFRepository()
	csrf := NewCSRFProtectionWithRepo(repo, logger)

	token, _ := csrf.GenerateToken()

	// Valid token should be accepted
	if !csrf.ValidateToken(token) {
		t.Error("ValidateToken() should return true for valid token")
	}

	// Invalid token should be rejected
	if csrf.ValidateToken("invalid-token") {
		t.Error("ValidateToken() should return false for invalid token")
	}
}

func TestCSRFProtectionWithRepo_InvalidateToken(t *testing.T) {
	logger := zap.NewNop()
	repo := NewMockCSRFRepository()
	csrf := NewCSRFProtectionWithRepo(repo, logger)

	token, _ := csrf.GenerateToken()

	// Token should be valid initially
	if !csrf.ValidateToken(token) {
		t.Error("token should be valid before invalidation")
	}

	// Invalidate it
	csrf.InvalidateToken(token)

	// Token should be invalid now (marked as used)
	if csrf.ValidateToken(token) {
		t.Error("token should be invalid after invalidation")
	}
}

func TestCSRFProtectionWithRepo_GenerateTokenForSession(t *testing.T) {
	logger := zap.NewNop()
	repo := NewMockCSRFRepository()
	csrf := NewCSRFProtectionWithRepo(repo, logger)

	sessionID := uuid.New()

	token1, err := csrf.GenerateTokenForSession(&sessionID)
	if err != nil {
		t.Fatalf("GenerateTokenForSession() error = %v", err)
	}

	// Second call with same session should return same token
	token2, err := csrf.GenerateTokenForSession(&sessionID)
	if err != nil {
		t.Fatalf("GenerateTokenForSession() second call error = %v", err)
	}

	if token1 != token2 {
		t.Errorf("expected same token for same session, got %q and %q", token1, token2)
	}

	// Different session should get different token
	differentSessionID := uuid.New()
	token3, err := csrf.GenerateTokenForSession(&differentSessionID)
	if err != nil {
		t.Fatalf("GenerateTokenForSession() different session error = %v", err)
	}

	if token1 == token3 {
		t.Error("expected different token for different session")
	}
}
