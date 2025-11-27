package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	// Use a fresh registry to avoid conflicts
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	if m == nil {
		t.Fatal("NewMetricsWithRegistry returned nil")
	}

	// Verify some metrics are initialized
	if m.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal not initialized")
	}
	if m.QuoteGenerationsTotal == nil {
		t.Error("QuoteGenerationsTotal not initialized")
	}
	if m.WebhooksReceivedTotal == nil {
		t.Error("WebhooksReceivedTotal not initialized")
	}
}

func TestMetrics_RecordAuthAttempt(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	// Record success
	m.RecordAuthAttempt(true)
	m.RecordAuthAttempt(true)

	// Record failure
	m.RecordAuthAttempt(false)

	// Verify counts
	successCount := testutil.ToFloat64(m.AuthAttemptsTotal.WithLabelValues("success"))
	failureCount := testutil.ToFloat64(m.AuthAttemptsTotal.WithLabelValues("failure"))

	if successCount != 2 {
		t.Errorf("success count = %f, expected 2", successCount)
	}
	if failureCount != 1 {
		t.Errorf("failure count = %f, expected 1", failureCount)
	}
}

func TestMetrics_RecordQuoteGeneration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	// Record success
	m.RecordQuoteGeneration(true, 5*time.Second)
	m.RecordQuoteGeneration(true, 3*time.Second)

	// Record failure
	m.RecordQuoteGeneration(false, 1*time.Second)

	// Record timeout
	m.RecordQuoteTimeout()

	// Verify counts
	successCount := testutil.ToFloat64(m.QuoteGenerationsTotal.WithLabelValues("success"))
	failureCount := testutil.ToFloat64(m.QuoteGenerationsTotal.WithLabelValues("failure"))
	timeoutCount := testutil.ToFloat64(m.QuoteGenerationsTotal.WithLabelValues("timeout"))

	if successCount != 2 {
		t.Errorf("success count = %f, expected 2", successCount)
	}
	if failureCount != 1 {
		t.Errorf("failure count = %f, expected 1", failureCount)
	}
	if timeoutCount != 1 {
		t.Errorf("timeout count = %f, expected 1", timeoutCount)
	}
}

func TestMetrics_RecordWebhook(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.RecordWebhook("bland", "valid", 100*time.Millisecond)
	m.RecordWebhook("bland", "valid", 50*time.Millisecond)
	m.RecordWebhook("vapi", "invalid_signature", 10*time.Millisecond)

	// Verify counts
	blandValid := testutil.ToFloat64(m.WebhooksReceivedTotal.WithLabelValues("bland", "valid"))
	vapiInvalid := testutil.ToFloat64(m.WebhooksReceivedTotal.WithLabelValues("vapi", "invalid_signature"))

	if blandValid != 2 {
		t.Errorf("bland valid count = %f, expected 2", blandValid)
	}
	if vapiInvalid != 1 {
		t.Errorf("vapi invalid count = %f, expected 1", vapiInvalid)
	}
}

func TestMetrics_RecordClaudeAPICall(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.RecordClaudeAPICall(true, 2*time.Second)
	m.RecordClaudeAPICall(false, 500*time.Millisecond)
	m.RecordCircuitOpen()

	successCount := testutil.ToFloat64(m.ClaudeAPICallsTotal.WithLabelValues("success"))
	failureCount := testutil.ToFloat64(m.ClaudeAPICallsTotal.WithLabelValues("failure"))
	circuitOpenCount := testutil.ToFloat64(m.ClaudeAPICallsTotal.WithLabelValues("circuit_open"))
	tripCount := testutil.ToFloat64(m.CircuitBreakerTrips)

	if successCount != 1 {
		t.Errorf("success count = %f, expected 1", successCount)
	}
	if failureCount != 1 {
		t.Errorf("failure count = %f, expected 1", failureCount)
	}
	if circuitOpenCount != 1 {
		t.Errorf("circuit_open count = %f, expected 1", circuitOpenCount)
	}
	if tripCount != 1 {
		t.Errorf("trip count = %f, expected 1", tripCount)
	}
}

func TestMetrics_SetCircuitBreakerState(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.SetCircuitBreakerState("claude", 0) // closed
	state := testutil.ToFloat64(m.CircuitBreakerState.WithLabelValues("claude"))
	if state != 0 {
		t.Errorf("state = %f, expected 0 (closed)", state)
	}

	m.SetCircuitBreakerState("claude", 2) // open
	state = testutil.ToFloat64(m.CircuitBreakerState.WithLabelValues("claude"))
	if state != 2 {
		t.Errorf("state = %f, expected 2 (open)", state)
	}
}

func TestMetrics_UpdateDBConnections(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.UpdateDBConnections(10, 5)

	open := testutil.ToFloat64(m.DBConnectionsOpen)
	inUse := testutil.ToFloat64(m.DBConnectionsInUse)

	if open != 10 {
		t.Errorf("open = %f, expected 10", open)
	}
	if inUse != 5 {
		t.Errorf("inUse = %f, expected 5", inUse)
	}
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	// Success
	m.RecordDBQuery("select", 50*time.Millisecond, nil)

	// Error
	m.RecordDBQuery("insert", 100*time.Millisecond, http.ErrAbortHandler)

	selectErrors := testutil.ToFloat64(m.DBQueryErrors.WithLabelValues("select"))
	insertErrors := testutil.ToFloat64(m.DBQueryErrors.WithLabelValues("insert"))

	if selectErrors != 0 {
		t.Errorf("select errors = %f, expected 0", selectErrors)
	}
	if insertErrors != 1 {
		t.Errorf("insert errors = %f, expected 1", insertErrors)
	}
}

func TestMetrics_RateLimiting(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.RecordRateLimitHit("login")
	m.RecordRateLimitHit("login")
	m.RecordRateLimitHit("general")

	m.SetRateLimitUsage("login", "minute", 5)
	m.SetRateLimitUsage("quote", "day", 100)

	loginHits := testutil.ToFloat64(m.RateLimitHitsTotal.WithLabelValues("login"))
	generalHits := testutil.ToFloat64(m.RateLimitHitsTotal.WithLabelValues("general"))

	if loginHits != 2 {
		t.Errorf("login hits = %f, expected 2", loginHits)
	}
	if generalHits != 1 {
		t.Errorf("general hits = %f, expected 1", generalHits)
	}

	loginMinute := testutil.ToFloat64(m.RateLimitCurrent.WithLabelValues("login", "minute"))
	quoteDay := testutil.ToFloat64(m.RateLimitCurrent.WithLabelValues("quote", "day"))

	if loginMinute != 5 {
		t.Errorf("login minute = %f, expected 5", loginMinute)
	}
	if quoteDay != 100 {
		t.Errorf("quote day = %f, expected 100", quoteDay)
	}
}

func TestMetrics_SessionMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.RecordSessionCreated()
	m.RecordSessionCreated()
	m.RecordSessionExpired()
	m.SetActiveSessions(10)

	created := testutil.ToFloat64(m.SessionsCreated)
	expired := testutil.ToFloat64(m.SessionsExpired)
	active := testutil.ToFloat64(m.SessionsActive)

	if created != 2 {
		t.Errorf("created = %f, expected 2", created)
	}
	if expired != 1 {
		t.Errorf("expired = %f, expected 1", expired)
	}
	if active != 10 {
		t.Errorf("active = %f, expected 10", active)
	}
}

func TestMetrics_QuoteJobMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	m.SetQuoteJobsInQueue(5)
	m.RecordQuoteJobProcessed("completed")
	m.RecordQuoteJobProcessed("failed")

	inQueue := testutil.ToFloat64(m.QuoteJobsInQueue)
	completed := testutil.ToFloat64(m.QuoteJobsProcessed.WithLabelValues("completed"))
	failed := testutil.ToFloat64(m.QuoteJobsProcessed.WithLabelValues("failed"))

	if inQueue != 5 {
		t.Errorf("inQueue = %f, expected 5", inQueue)
	}
	if completed != 1 {
		t.Errorf("completed = %f, expected 1", completed)
	}
	if failed != 1 {
		t.Errorf("failed = %f, expected 1", failed)
	}
}

func TestMetrics_Middleware(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// Make test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, expected %d", rr.Code, http.StatusOK)
	}

	// Verify metrics were recorded
	count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "200"))
	if count != 1 {
		t.Errorf("request count = %f, expected 1", count)
	}
}

func TestMetrics_Middleware_InFlight(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	// Check initial value
	initial := testutil.ToFloat64(m.HTTPRequestsInFlight)
	if initial != 0 {
		t.Errorf("initial in-flight = %f, expected 0", initial)
	}

	inFlightDuringHandler := float64(-1)
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlightDuringHandler = testutil.ToFloat64(m.HTTPRequestsInFlight)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// During handler, should have been 1
	if inFlightDuringHandler != 1 {
		t.Errorf("in-flight during handler = %f, expected 1", inFlightDuringHandler)
	}

	// After handler, should be back to 0
	after := testutil.ToFloat64(m.HTTPRequestsInFlight)
	if after != 0 {
		t.Errorf("in-flight after = %f, expected 0", after)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/", "/"},
		{"/login", "/login"},
		{"/logout", "/logout"},
		{"/dashboard", "/dashboard"},
		{"/calls", "/calls"},
		{"/health", "/health"},
		{"/ready", "/ready"},
		{"/live", "/live"},
		{"/calls/123", "/calls/:id"},
		{"/calls/abc-def-123", "/calls/:id"},
		{"/webhook/bland", "/webhook/:provider"},
		{"/webhook/vapi", "/webhook/:provider"},
		{"/static/css/style.css", "/static/*"},
		{"/static/js/app.js", "/static/*"},
		{"/unknown/path", "/unknown/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePath(tt.input)
			if got != tt.expected {
				t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	// Test WriteHeader
	t.Run("WriteHeader", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		if rw.statusCode != http.StatusNotFound {
			t.Errorf("statusCode = %d, expected %d", rw.statusCode, http.StatusNotFound)
		}

		// Second call should be ignored
		rw.WriteHeader(http.StatusOK)
		if rw.statusCode != http.StatusNotFound {
			t.Errorf("statusCode after second call = %d, expected %d", rw.statusCode, http.StatusNotFound)
		}
	})

	// Test Write (implicit 200)
	t.Run("Write", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.Write([]byte("test"))
		if rw.statusCode != http.StatusOK {
			t.Errorf("statusCode = %d, expected %d", rw.statusCode, http.StatusOK)
		}
		if !rw.written {
			t.Error("written should be true after Write")
		}
	})
}

func TestMetrics_Handler(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry(reg)

	handler := m.Handler()
	if handler == nil {
		t.Fatal("Handler returned nil")
	}

	// Make request to metrics handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, expected %d", rr.Code, http.StatusOK)
	}
}
