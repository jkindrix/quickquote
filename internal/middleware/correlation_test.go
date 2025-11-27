package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestRequestCorrelation_GeneratesIDs(t *testing.T) {
	logger := zap.NewNop()
	middleware := NewRequestCorrelation(logger)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Verify all IDs are set
		if GetCorrelationID(ctx) == "" {
			t.Error("correlation ID not set")
		}
		if GetRequestID(ctx) == "" {
			t.Error("request ID not set")
		}
		if GetTraceID(ctx) == "" {
			t.Error("trace ID not set")
		}
		if GetSpanID(ctx) == "" {
			t.Error("span ID not set")
		}
		if GetRequestStartTime(ctx).IsZero() {
			t.Error("request start time not set")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response headers
	if rec.Header().Get(CorrelationIDHeader) == "" {
		t.Error("correlation ID header not set in response")
	}
	if rec.Header().Get(RequestIDHeader) == "" {
		t.Error("request ID header not set in response")
	}
	if rec.Header().Get(TraceIDHeader) == "" {
		t.Error("trace ID header not set in response")
	}
	if rec.Header().Get(SpanIDHeader) == "" {
		t.Error("span ID header not set in response")
	}
}

func TestRequestCorrelation_PreservesIncomingIDs(t *testing.T) {
	logger := zap.NewNop()
	middleware := NewRequestCorrelation(logger)

	incomingCorrelationID := "test-correlation-123"
	incomingRequestID := "test-request-456"
	incomingTraceID := "test-trace-789"

	var capturedCorrelationID, capturedRequestID, capturedTraceID string

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		capturedCorrelationID = GetCorrelationID(ctx)
		capturedRequestID = GetRequestID(ctx)
		capturedTraceID = GetTraceID(ctx)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(CorrelationIDHeader, incomingCorrelationID)
	req.Header.Set(RequestIDHeader, incomingRequestID)
	req.Header.Set(TraceIDHeader, incomingTraceID)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedCorrelationID != incomingCorrelationID {
		t.Errorf("correlation ID not preserved: got %s, want %s", capturedCorrelationID, incomingCorrelationID)
	}
	if capturedRequestID != incomingRequestID {
		t.Errorf("request ID not preserved: got %s, want %s", capturedRequestID, incomingRequestID)
	}
	if capturedTraceID != incomingTraceID {
		t.Errorf("trace ID not preserved: got %s, want %s", capturedTraceID, incomingTraceID)
	}

	// Verify response headers match
	if rec.Header().Get(CorrelationIDHeader) != incomingCorrelationID {
		t.Errorf("correlation ID not in response headers")
	}
}

func TestRequestCorrelation_AlwaysGeneratesNewSpanID(t *testing.T) {
	logger := zap.NewNop()
	middleware := NewRequestCorrelation(logger)

	incomingSpanID := "old-span-id"
	var capturedSpanID string

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSpanID = GetSpanID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(SpanIDHeader, incomingSpanID)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Span ID should be different from incoming (new span for each request)
	if capturedSpanID == incomingSpanID {
		t.Error("span ID should be newly generated, not preserved from incoming")
	}
	if capturedSpanID == "" {
		t.Error("span ID should be generated")
	}
}

func TestGetCorrelationID_NoContext(t *testing.T) {
	ctx := context.Background()
	if id := GetCorrelationID(ctx); id != "" {
		t.Errorf("expected empty string for context without correlation ID, got %s", id)
	}
}

func TestWithCorrelationID(t *testing.T) {
	ctx := context.Background()
	id := "test-id-123"

	ctx = WithCorrelationID(ctx, id)

	if got := GetCorrelationID(ctx); got != id {
		t.Errorf("WithCorrelationID: got %s, want %s", got, id)
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	id := "test-trace-456"

	ctx = WithTraceID(ctx, id)

	if got := GetTraceID(ctx); got != id {
		t.Errorf("WithTraceID: got %s, want %s", got, id)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID returned empty string")
	}
	if id1 == id2 {
		t.Error("generateID should return unique IDs")
	}
	if len(id1) != 32 { // 16 bytes * 2 for hex encoding
		t.Errorf("expected 32 char ID, got %d", len(id1))
	}
}

func TestLoggerWithCorrelation(t *testing.T) {
	logger := zap.NewNop()

	// Without correlation context
	result := LoggerWithCorrelation(context.Background(), logger)
	if result == nil {
		t.Error("LoggerWithCorrelation returned nil")
	}

	// With correlation context
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "test-correlation")
	ctx = WithTraceID(ctx, "test-trace")

	result = LoggerWithCorrelation(ctx, logger)
	if result == nil {
		t.Error("LoggerWithCorrelation returned nil with context")
	}
}

func TestPropagateHeaders(t *testing.T) {
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "test-correlation")
	ctx = WithTraceID(ctx, "test-trace")

	req := httptest.NewRequest("GET", "/downstream", nil)
	PropagateHeaders(ctx, req)

	if req.Header.Get(CorrelationIDHeader) != "test-correlation" {
		t.Errorf("correlation ID not propagated")
	}
	if req.Header.Get(TraceIDHeader) != "test-trace" {
		t.Errorf("trace ID not propagated")
	}
	if req.Header.Get(SpanIDHeader) == "" {
		t.Error("new span ID not generated for outgoing request")
	}
}

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	wrapped.WriteHeader(http.StatusCreated)

	if wrapped.statusCode != http.StatusCreated {
		t.Errorf("status code not captured: got %d, want %d", wrapped.statusCode, http.StatusCreated)
	}
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	// Write without calling WriteHeader
	wrapped.Write([]byte("test"))

	if wrapped.statusCode != http.StatusOK {
		t.Errorf("default status code should be 200, got %d", wrapped.statusCode)
	}
}

func TestGetRequestStartTime(t *testing.T) {
	// Without start time
	ctx := context.Background()
	if !GetRequestStartTime(ctx).IsZero() {
		t.Error("expected zero time for context without start time")
	}

	// With start time
	now := time.Now()
	ctx = context.WithValue(ctx, requestStartTimeKey{}, now)
	if got := GetRequestStartTime(ctx); !got.Equal(now) {
		t.Errorf("GetRequestStartTime: got %v, want %v", got, now)
	}
}
