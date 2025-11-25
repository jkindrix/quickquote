// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/service"
)

// HealthChecker defines the interface for checking database health.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// Handler holds all HTTP handlers and their dependencies.
type Handler struct {
	callService   *service.CallService
	authService   *service.AuthService
	healthChecker HealthChecker
	logger        *zap.Logger
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

// SetHealthChecker sets the health checker for database connectivity.
func (h *Handler) SetHealthChecker(hc HealthChecker) {
	h.healthChecker = hc
}

// RegisterRoutes registers all routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public routes
	r.Get("/", h.HandleIndex)
	r.Get("/login", h.HandleLoginPage)
	r.Post("/login", h.HandleLogin)
	r.Get("/logout", h.HandleLogout)

	// Webhook routes (authenticated via signature)
	r.Post("/webhook/bland", h.HandleBlandWebhook)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Get("/dashboard", h.HandleDashboard)
		r.Get("/calls", h.HandleCallsList)
		r.Get("/calls/{id}", h.HandleCallDetail)
		r.Post("/calls/{id}/regenerate-quote", h.HandleRegenerateQuote)
	})

	// Health check
	r.Get("/health", h.HandleHealth)
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version,omitempty"`
	Checks   map[string]string `json:"checks,omitempty"`
}

// HandleHealth returns a health check response including database connectivity.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Checks:  make(map[string]string),
	}

	// Check database connectivity
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(ctx); err != nil {
			response.Status = "degraded"
			response.Checks["database"] = "unhealthy: " + err.Error()
			h.logger.Error("database health check failed", zap.Error(err))
		} else {
			response.Checks["database"] = "healthy"
		}
	}

	w.Header().Set("Content-Type", "application/json")

	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(response)
}
