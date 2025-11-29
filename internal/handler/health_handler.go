package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

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

// HealthHandler handles health check HTTP requests.
type HealthHandler struct {
	healthChecker    HealthChecker
	aiHealthChecker  AIHealthChecker
	providerRegistry *voiceprovider.Registry
	logger           *zap.Logger
}

// HealthHandlerConfig holds configuration for HealthHandler.
type HealthHandlerConfig struct {
	HealthChecker    HealthChecker
	AIHealthChecker  AIHealthChecker
	ProviderRegistry *voiceprovider.Registry
	Logger           *zap.Logger
}

// NewHealthHandler creates a new HealthHandler with all required dependencies.
func NewHealthHandler(cfg HealthHandlerConfig) *HealthHandler {
	if cfg.Logger == nil {
		panic("logger is required")
	}
	return &HealthHandler{
		healthChecker:    cfg.HealthChecker,
		aiHealthChecker:  cfg.AIHealthChecker,
		providerRegistry: cfg.ProviderRegistry,
		logger:           cfg.Logger,
	}
}

// RegisterRoutes registers health routes on the router.
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.HandleHealth)
	r.Get("/ready", h.HandleReadiness)
	r.Get("/live", h.HandleLiveness)
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status         string                     `json:"status"`
	Version        string                     `json:"version,omitempty"`
	Checks         map[string]ComponentHealth `json:"checks,omitempty"`
	VoiceProviders []VoiceProviderHealth      `json:"voice_providers,omitempty"`
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
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Checks:  make(map[string]ComponentHealth),
	}

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

	// Check AI service health via circuit breaker
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

	// Add request ID header
	if reqID := GetRequestIDFromContext(r.Context()); reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}

	w.Header().Set("Content-Type", "application/json")

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)

	if err := encodeJSON(w, response); err != nil {
		h.logger.Debug("failed to write health response", zap.Error(err))
	}
}

// HandleReadiness returns a simple readiness probe response.
func (h *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Only check database - the critical dependency
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(ctx); err != nil {
			h.logger.Error("readiness check failed", zap.Error(err))
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

// HandleLiveness returns a simple liveness probe response.
func (h *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("alive"))
}
