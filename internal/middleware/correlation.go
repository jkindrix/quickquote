// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Correlation ID constants.
const (
	// CorrelationIDHeader is the HTTP header name for correlation IDs.
	CorrelationIDHeader = "X-Correlation-ID"
	// RequestIDHeader is the HTTP header name for request IDs.
	RequestIDHeader = "X-Request-ID"
	// TraceIDHeader is the header for distributed tracing.
	TraceIDHeader = "X-Trace-ID"
	// SpanIDHeader is the header for span identification.
	SpanIDHeader = "X-Span-ID"
)

// correlationIDKey is the context key for correlation ID.
type correlationIDKey struct{}

// requestIDKey is the context key for request ID.
type requestIDKey struct{}

// traceIDKey is the context key for trace ID.
type traceIDKey struct{}

// spanIDKey is the context key for span ID.
type spanIDKey struct{}

// requestStartTimeKey is the context key for request start time.
type requestStartTimeKey struct{}

// RequestCorrelation provides request correlation and tracing middleware.
type RequestCorrelation struct {
	logger *zap.Logger
}

// NewRequestCorrelation creates a new correlation middleware.
func NewRequestCorrelation(logger *zap.Logger) *RequestCorrelation {
	return &RequestCorrelation{
		logger: logger,
	}
}

// Middleware returns the HTTP middleware handler.
func (rc *RequestCorrelation) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		startTime := time.Now()

		// Extract or generate correlation ID
		correlationID := r.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = generateID()
		}

		// Extract or generate request ID (always unique per request)
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = generateID()
		}

		// Extract or generate trace ID
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = generateID()
		}

		// Always generate a new span ID
		spanID := generateID()[:16] // Shorter span ID

		// Add to context
		ctx = context.WithValue(ctx, correlationIDKey{}, correlationID)
		ctx = context.WithValue(ctx, requestIDKey{}, requestID)
		ctx = context.WithValue(ctx, traceIDKey{}, traceID)
		ctx = context.WithValue(ctx, spanIDKey{}, spanID)
		ctx = context.WithValue(ctx, requestStartTimeKey{}, startTime)

		// Set response headers
		w.Header().Set(CorrelationIDHeader, correlationID)
		w.Header().Set(RequestIDHeader, requestID)
		w.Header().Set(TraceIDHeader, traceID)
		w.Header().Set(SpanIDHeader, spanID)

		// Create wrapped response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Log request completion with correlation info
		duration := time.Since(startTime)
		rc.logger.Debug("request completed",
			zap.String("correlation_id", correlationID),
			zap.String("request_id", requestID),
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// GetCorrelationID retrieves the correlation ID from context.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GetSpanID retrieves the span ID from context.
func GetSpanID(ctx context.Context) string {
	if id, ok := ctx.Value(spanIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GetRequestStartTime retrieves the request start time from context.
func GetRequestStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(requestStartTimeKey{}).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// WithCorrelationID creates a new context with the given correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// WithTraceID creates a new context with the given trace ID.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

// generateID generates a random ID suitable for correlation/tracing.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto rand fails
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}

// LoggerWithCorrelation returns a logger with correlation fields added.
func LoggerWithCorrelation(ctx context.Context, logger *zap.Logger) *zap.Logger {
	fields := make([]zap.Field, 0, 4)

	if id := GetCorrelationID(ctx); id != "" {
		fields = append(fields, zap.String("correlation_id", id))
	}
	if id := GetRequestID(ctx); id != "" {
		fields = append(fields, zap.String("request_id", id))
	}
	if id := GetTraceID(ctx); id != "" {
		fields = append(fields, zap.String("trace_id", id))
	}
	if id := GetSpanID(ctx); id != "" {
		fields = append(fields, zap.String("span_id", id))
	}

	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields...)
}

// PropagateHeaders adds correlation headers to an outgoing HTTP request.
func PropagateHeaders(ctx context.Context, req *http.Request) {
	if id := GetCorrelationID(ctx); id != "" {
		req.Header.Set(CorrelationIDHeader, id)
	}
	if id := GetTraceID(ctx); id != "" {
		req.Header.Set(TraceIDHeader, id)
	}
	// Create new span ID for outgoing call
	req.Header.Set(SpanIDHeader, generateID()[:16])
}
