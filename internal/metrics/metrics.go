// Package metrics provides Prometheus metrics collection for the application.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Outcome/status label values for metrics.
const (
	outcomeSuccess = "success"
	outcomeFailure = "failure"
)

// Metrics holds all Prometheus metrics for the application.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Authentication metrics
	AuthAttemptsTotal  *prometheus.CounterVec
	SessionsActive     prometheus.Gauge
	SessionsCreated    prometheus.Counter
	SessionsExpired    prometheus.Counter

	// Quote generation metrics
	QuoteGenerationsTotal    *prometheus.CounterVec
	QuoteGenerationDuration  prometheus.Histogram
	QuoteJobsInQueue         prometheus.Gauge
	QuoteJobsProcessed       *prometheus.CounterVec

	// Voice provider metrics
	WebhooksReceivedTotal   *prometheus.CounterVec
	WebhookProcessDuration  *prometheus.HistogramVec
	ProviderCallsTotal      *prometheus.CounterVec

	// External service metrics
	ClaudeAPICallsTotal     *prometheus.CounterVec
	ClaudeAPICallDuration   prometheus.Histogram
	CircuitBreakerState     *prometheus.GaugeVec
	CircuitBreakerTrips     prometheus.Counter

	// Database metrics
	DBConnectionsOpen   prometheus.Gauge
	DBConnectionsInUse  prometheus.Gauge
	DBQueryDuration     *prometheus.HistogramVec
	DBQueryErrors       *prometheus.CounterVec

	// Rate limiting metrics
	RateLimitHitsTotal  *prometheus.CounterVec
	RateLimitCurrent    *prometheus.GaugeVec

	// Registry used for this metrics instance (nil means default registry)
	registry prometheus.Gatherer
}

// NewMetrics creates a new Metrics instance with all collectors registered.
func NewMetrics() *Metrics {
	m := newMetricsWithRegistry(prometheus.DefaultRegisterer)
	m.registry = prometheus.DefaultGatherer
	return m
}

// NewMetricsWithRegistry creates metrics using a custom registry (for testing).
func NewMetricsWithRegistry(reg *prometheus.Registry) *Metrics {
	m := newMetricsWithRegistry(reg)
	m.registry = reg
	return m
}

func newMetricsWithRegistry(registerer prometheus.Registerer) *Metrics {
	factory := promauto.With(registerer)

	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_http_requests_total",
				Help: "Total number of HTTP requests by method, path, and status code",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "quickquote_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "quickquote_http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		// Authentication metrics
		AuthAttemptsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_auth_attempts_total",
				Help: "Total number of authentication attempts by outcome",
			},
			[]string{"outcome"}, // "success", "failure", "rate_limited"
		),
		SessionsActive: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "quickquote_sessions_active",
				Help: "Number of active sessions",
			},
		),
		SessionsCreated: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "quickquote_sessions_created_total",
				Help: "Total number of sessions created",
			},
		),
		SessionsExpired: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "quickquote_sessions_expired_total",
				Help: "Total number of sessions expired",
			},
		),

		// Quote generation metrics
		QuoteGenerationsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_quote_generations_total",
				Help: "Total number of quote generation attempts by status",
			},
			[]string{"status"}, // "success", "failure", "timeout"
		),
		QuoteGenerationDuration: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quickquote_quote_generation_duration_seconds",
				Help:    "Time taken to generate a quote",
				Buckets: []float64{1, 2, 5, 10, 15, 30, 60},
			},
		),
		QuoteJobsInQueue: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "quickquote_quote_jobs_in_queue",
				Help: "Number of quote generation jobs currently in queue",
			},
		),
		QuoteJobsProcessed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_quote_jobs_processed_total",
				Help: "Total number of quote jobs processed by status",
			},
			[]string{"status"}, // "completed", "failed", "retried"
		),

		// Voice provider metrics
		WebhooksReceivedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_webhooks_received_total",
				Help: "Total number of webhooks received by provider and status",
			},
			[]string{"provider", "status"}, // status: "valid", "invalid_signature", "parse_error"
		),
		WebhookProcessDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "quickquote_webhook_process_duration_seconds",
				Help:    "Time taken to process webhooks",
				Buckets: []float64{.01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"provider"},
		),
		ProviderCallsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_provider_calls_total",
				Help: "Total number of calls received by provider and status",
			},
			[]string{"provider", "call_status"},
		),

		// External service metrics
		ClaudeAPICallsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_claude_api_calls_total",
				Help: "Total number of Claude API calls by status",
			},
			[]string{"status"}, // "success", "failure", "circuit_open"
		),
		ClaudeAPICallDuration: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quickquote_claude_api_call_duration_seconds",
				Help:    "Duration of Claude API calls",
				Buckets: []float64{.5, 1, 2, 5, 10, 15, 30},
			},
		),
		CircuitBreakerState: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quickquote_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"service"},
		),
		CircuitBreakerTrips: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "quickquote_circuit_breaker_trips_total",
				Help: "Total number of times the circuit breaker has tripped",
			},
		),

		// Database metrics
		DBConnectionsOpen: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "quickquote_db_connections_open",
				Help: "Number of open database connections",
			},
		),
		DBConnectionsInUse: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "quickquote_db_connections_in_use",
				Help: "Number of database connections currently in use",
			},
		),
		DBQueryDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "quickquote_db_query_duration_seconds",
				Help:    "Duration of database queries",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"}, // "select", "insert", "update", "delete"
		),
		DBQueryErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"operation"},
		),

		// Rate limiting metrics
		RateLimitHitsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quickquote_rate_limit_hits_total",
				Help: "Total number of rate limit hits by type",
			},
			[]string{"limiter"}, // "general", "login", "quote"
		),
		RateLimitCurrent: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quickquote_rate_limit_current",
				Help: "Current rate limit usage",
			},
			[]string{"limiter", "window"}, // window: "minute", "hour", "day"
		),
	}
}

// Handler returns the Prometheus HTTP handler for scraping metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware returns an HTTP middleware that records request metrics.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()

		// Normalize path for metrics (avoid high cardinality)
		path := normalizePath(r.URL.Path)

		m.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			path,
			strconv.Itoa(wrapped.statusCode),
		).Inc()

		m.HTTPRequestDuration.WithLabelValues(
			r.Method,
			path,
		).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
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
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// normalizePath normalizes URL paths to prevent high cardinality labels.
func normalizePath(path string) string {
	// Map specific paths
	switch path {
	case "/", "/login", "/logout", "/dashboard", "/calls", "/health", "/ready", "/live":
		return path
	}

	// Normalize dynamic paths
	if len(path) > 7 && path[:7] == "/calls/" {
		return "/calls/:id"
	}
	if len(path) > 9 && path[:9] == "/webhook/" {
		return "/webhook/:provider"
	}
	if len(path) > 8 && path[:8] == "/static/" {
		return "/static/*"
	}

	return path
}

// Helper methods for recording specific events

// RecordAuthAttempt records an authentication attempt.
func (m *Metrics) RecordAuthAttempt(success bool) {
	outcome := outcomeFailure
	if success {
		outcome = outcomeSuccess
	}
	m.AuthAttemptsTotal.WithLabelValues(outcome).Inc()
}

// RecordAuthRateLimited records a rate-limited auth attempt.
func (m *Metrics) RecordAuthRateLimited() {
	m.AuthAttemptsTotal.WithLabelValues("rate_limited").Inc()
}

// RecordSessionCreated records a new session creation.
func (m *Metrics) RecordSessionCreated() {
	m.SessionsCreated.Inc()
}

// RecordSessionExpired records a session expiration.
func (m *Metrics) RecordSessionExpired() {
	m.SessionsExpired.Inc()
}

// RecordQuoteGeneration records a quote generation attempt.
func (m *Metrics) RecordQuoteGeneration(success bool, duration time.Duration) {
	status := outcomeFailure
	if success {
		status = outcomeSuccess
	}
	m.QuoteGenerationsTotal.WithLabelValues(status).Inc()
	m.QuoteGenerationDuration.Observe(duration.Seconds())
}

// RecordQuoteTimeout records a quote generation timeout.
func (m *Metrics) RecordQuoteTimeout() {
	m.QuoteGenerationsTotal.WithLabelValues("timeout").Inc()
}

// RecordWebhook records a webhook receipt.
func (m *Metrics) RecordWebhook(provider, status string, duration time.Duration) {
	m.WebhooksReceivedTotal.WithLabelValues(provider, status).Inc()
	m.WebhookProcessDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordProviderCall records a call from a voice provider.
func (m *Metrics) RecordProviderCall(provider, callStatus string) {
	m.ProviderCallsTotal.WithLabelValues(provider, callStatus).Inc()
}

// RecordClaudeAPICall records a Claude API call.
func (m *Metrics) RecordClaudeAPICall(success bool, duration time.Duration) {
	status := outcomeFailure
	if success {
		status = outcomeSuccess
	}
	m.ClaudeAPICallsTotal.WithLabelValues(status).Inc()
	m.ClaudeAPICallDuration.Observe(duration.Seconds())
}

// RecordCircuitOpen records a circuit breaker opening.
func (m *Metrics) RecordCircuitOpen() {
	m.ClaudeAPICallsTotal.WithLabelValues("circuit_open").Inc()
	m.CircuitBreakerTrips.Inc()
}

// SetCircuitBreakerState sets the circuit breaker state for a service.
// State: 0=closed, 1=half-open, 2=open
func (m *Metrics) SetCircuitBreakerState(service string, state int) {
	m.CircuitBreakerState.WithLabelValues(service).Set(float64(state))
}

// UpdateDBConnections updates database connection metrics.
func (m *Metrics) UpdateDBConnections(open, inUse int) {
	m.DBConnectionsOpen.Set(float64(open))
	m.DBConnectionsInUse.Set(float64(inUse))
}

// RecordDBQuery records a database query.
func (m *Metrics) RecordDBQuery(operation string, duration time.Duration, err error) {
	m.DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
	if err != nil {
		m.DBQueryErrors.WithLabelValues(operation).Inc()
	}
}

// RecordRateLimitHit records a rate limit hit.
func (m *Metrics) RecordRateLimitHit(limiter string) {
	m.RateLimitHitsTotal.WithLabelValues(limiter).Inc()
}

// SetRateLimitUsage sets current rate limit usage.
func (m *Metrics) SetRateLimitUsage(limiter, window string, current float64) {
	m.RateLimitCurrent.WithLabelValues(limiter, window).Set(current)
}

// SetQuoteJobsInQueue sets the number of jobs in the quote queue.
func (m *Metrics) SetQuoteJobsInQueue(count int) {
	m.QuoteJobsInQueue.Set(float64(count))
}

// RecordQuoteJobProcessed records a processed quote job.
func (m *Metrics) RecordQuoteJobProcessed(status string) {
	m.QuoteJobsProcessed.WithLabelValues(status).Inc()
}

// SetActiveSessions sets the number of active sessions.
func (m *Metrics) SetActiveSessions(count int) {
	m.SessionsActive.Set(float64(count))
}
