package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodySizeLimiter_AllowsSmallBody(t *testing.T) {
	handler := BodySizeLimiter(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected read error: %v", err)
			return
		}
		w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small body"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "small body" {
		t.Errorf("expected body 'small body', got %q", rec.Body.String())
	}
}

func TestBodySizeLimiter_RejectsLargeContentLength(t *testing.T) {
	handler := BodySizeLimiter(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for oversized request")
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small"))
	req.ContentLength = 200 // Claim larger size
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}

func TestBodySizeLimiter_MaxBytesReaderEnforces(t *testing.T) {
	var readErr error
	handler := BodySizeLimiter(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			// Handler properly detected the oversized body
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Send body larger than limit
	// Note: httptest.NewRequest sets ContentLength automatically,
	// so this will be caught by the Content-Length check first.
	// To test MaxBytesReader specifically, we simulate chunked encoding
	// by setting ContentLength to -1 (unknown).
	largeBody := strings.Repeat("x", 20)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))
	req.ContentLength = -1 // Simulate chunked/unknown length
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// MaxBytesReader should have caused a read error
	if readErr == nil {
		t.Error("expected error reading oversized body, got nil")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}

func TestBodySizeLimiter_AllowsNoBody(t *testing.T) {
	called := false
	handler := BodySizeLimiter(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestBodySizeLimiter_AllowsEmptyBody(t *testing.T) {
	called := false
	handler := BodySizeLimiter(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestBodySizeLimiterJSON(t *testing.T) {
	handler := BodySizeLimiterJSON()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with body under limit
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"test": true}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestBodySizeLimiterForm(t *testing.T) {
	handler := BodySizeLimiterForm()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with body under limit
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("field=value"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestBodySizeLimiterWebhook(t *testing.T) {
	handler := BodySizeLimiterWebhook()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with body under limit
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"webhook": "payload"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify constants are set correctly
	if DefaultMaxBodySize != 1<<20 {
		t.Errorf("DefaultMaxBodySize should be 1MB, got %d", DefaultMaxBodySize)
	}
	if MaxWebhookBodySize != 10<<20 {
		t.Errorf("MaxWebhookBodySize should be 10MB, got %d", MaxWebhookBodySize)
	}
	if MaxFormBodySize != 100<<10 {
		t.Errorf("MaxFormBodySize should be 100KB, got %d", MaxFormBodySize)
	}
	if MaxJSONBodySize != 1<<20 {
		t.Errorf("MaxJSONBodySize should be 1MB, got %d", MaxJSONBodySize)
	}
}
