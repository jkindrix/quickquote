// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/service"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

// HealthChecker defines the interface for checking database health.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// AIHealthChecker defines the interface for checking AI service health.
type AIHealthChecker interface {
	IsCircuitOpen() bool
}

// Handler holds all HTTP handlers and their dependencies.
type Handler struct {
	callService      *service.CallService
	authService      *service.AuthService
	healthChecker    HealthChecker
	aiHealthChecker  AIHealthChecker
	loginRateLimiter *middleware.LoginRateLimiter
	csrfProtection   *middleware.CSRFProtection
	templateEngine   *TemplateEngine
	providerRegistry *voiceprovider.Registry
	logger           *zap.Logger
}

// New creates a new Handler with all dependencies.
func New(
	callService *service.CallService,
	authService *service.AuthService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		callService: callService,
		authService: authService,
		logger:      logger,
	}
}

// SetTemplateEngine sets the template engine for rendering HTML.
func (h *Handler) SetTemplateEngine(te *TemplateEngine) {
	h.templateEngine = te
}

// SetHealthChecker sets the health checker for database connectivity.
func (h *Handler) SetHealthChecker(hc HealthChecker) {
	h.healthChecker = hc
}

// SetAIHealthChecker sets the AI service health checker.
func (h *Handler) SetAIHealthChecker(ahc AIHealthChecker) {
	h.aiHealthChecker = ahc
}

// SetLoginRateLimiter sets the login rate limiter.
func (h *Handler) SetLoginRateLimiter(lrl *middleware.LoginRateLimiter) {
	h.loginRateLimiter = lrl
}

// SetCSRFProtection sets the CSRF protection middleware.
func (h *Handler) SetCSRFProtection(csrf *middleware.CSRFProtection) {
	h.csrfProtection = csrf
}

// RegisterRoutes registers all routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public routes
	r.Get("/", h.HandleIndex)
	r.Get("/login", h.HandleLoginPage)
	r.Post("/login", h.HandleLogin)
	r.Get("/logout", h.HandleLogout)

	// Webhook routes - register dynamically based on configured providers
	if h.providerRegistry != nil {
		for _, path := range h.providerRegistry.GetWebhookPaths() {
			h.logger.Info("registering webhook route", zap.String("path", path))
			r.Post(path, h.HandleVoiceWebhook)
		}
	} else {
		// Fallback to legacy Bland-only route
		r.Post("/webhook/bland", h.HandleBlandWebhook)
	}

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Get("/dashboard", h.HandleDashboard)
		r.Get("/calls", h.HandleCallsList)
		r.Get("/calls/{id}", h.HandleCallDetail)
		r.Post("/calls/{id}/regenerate-quote", h.HandleRegenerateQuote)
	})

	// Health and readiness endpoints
	r.Get("/health", h.HandleHealth)
	r.Get("/ready", h.HandleReadiness)
	r.Get("/live", h.HandleLiveness)
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status         string                      `json:"status"`
	Version        string                      `json:"version,omitempty"`
	Checks         map[string]ComponentHealth  `json:"checks,omitempty"`
	VoiceProviders []VoiceProviderHealth       `json:"voice_providers,omitempty"`
}

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// VoiceProviderHealth represents the health of a voice provider.
type VoiceProviderHealth struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IsPrimary bool   `json:"is_primary"`
	Message   string `json:"message,omitempty"`
}

// HandleHealth returns a health check response including all service dependencies.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Checks:  make(map[string]ComponentHealth),
	}

	// Track if any critical component is unhealthy
	hasCriticalFailure := false
	hasDegradation := false

	// Check database connectivity (critical)
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(ctx); err != nil {
			hasCriticalFailure = true
			response.Checks["database"] = ComponentHealth{
				Status:  "unhealthy",
				Message: err.Error(),
			}
			h.logger.Error("database health check failed", zap.Error(err))
		} else {
			response.Checks["database"] = ComponentHealth{
				Status: "healthy",
			}
		}
	}

	// Check AI service (Claude) health via circuit breaker
	if h.aiHealthChecker != nil {
		if h.aiHealthChecker.IsCircuitOpen() {
			hasDegradation = true
			response.Checks["ai_service"] = ComponentHealth{
				Status:  "degraded",
				Message: "circuit breaker open - service temporarily unavailable",
			}
			h.logger.Warn("AI service circuit breaker is open")
		} else {
			response.Checks["ai_service"] = ComponentHealth{
				Status: "healthy",
			}
		}
	}

	// Check voice providers
	if h.providerRegistry != nil {
		statuses := h.providerRegistry.HealthStatus()
		response.VoiceProviders = make([]VoiceProviderHealth, len(statuses))
		for i, status := range statuses {
			response.VoiceProviders[i] = VoiceProviderHealth{
				Name:      string(status.Name),
				Status:    "available",
				IsPrimary: status.IsPrimary,
				Message:   status.Message,
			}
			if !status.Available {
				response.VoiceProviders[i].Status = "unavailable"
				hasDegradation = true
			}
		}

		// Check if we have any providers
		if h.providerRegistry.IsEmpty() {
			hasDegradation = true
			response.Checks["voice_providers"] = ComponentHealth{
				Status:  "degraded",
				Message: "no voice providers registered",
			}
		} else {
			response.Checks["voice_providers"] = ComponentHealth{
				Status:  "healthy",
				Message: fmt.Sprintf("%d provider(s) registered", h.providerRegistry.Count()),
			}
		}
	}

	// Determine overall status
	if hasCriticalFailure {
		response.Status = "unhealthy"
	} else if hasDegradation {
		response.Status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")

	statusCode := http.StatusOK
	switch response.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusOK // Still respond OK for degraded (service is running)
	}
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Debug("failed to write health response", zap.Error(err))
	}
}

// HandleReadiness returns a simple readiness probe response.
// This endpoint should be used by Kubernetes/container orchestrators for readiness checks.
func (h *Handler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Only check database - the critical dependency for handling requests
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(ctx); err != nil {
			h.logger.Error("readiness check failed", zap.Error(err))
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ready")); err != nil {
		h.logger.Debug("failed to write readiness response", zap.Error(err))
	}
}

// HandleLiveness returns a simple liveness probe response.
// This endpoint should be used by Kubernetes/container orchestrators for liveness checks.
func (h *Handler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness just confirms the process is running and responsive
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("alive")); err != nil {
		h.logger.Debug("failed to write liveness response", zap.Error(err))
	}
}
