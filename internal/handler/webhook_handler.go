package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/metrics"
	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/service"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

// WebhookHandler handles incoming webhooks from voice providers.
type WebhookHandler struct {
	callService      *service.CallService
	providerRegistry *voiceprovider.Registry
	logger           *zap.Logger
	metrics          *metrics.Metrics
}

// WebhookHandlerConfig holds configuration for WebhookHandler.
type WebhookHandlerConfig struct {
	CallService      *service.CallService
	ProviderRegistry *voiceprovider.Registry
	Logger           *zap.Logger
	Metrics          *metrics.Metrics
}

// NewWebhookHandler creates a new WebhookHandler with all required dependencies.
func NewWebhookHandler(cfg WebhookHandlerConfig) *WebhookHandler {
	if cfg.Logger == nil {
		panic("logger is required")
	}
	return &WebhookHandler{
		callService:      cfg.CallService,
		providerRegistry: cfg.ProviderRegistry,
		logger:           cfg.Logger,
		metrics:          cfg.Metrics,
	}
}

// RegisterRoutes registers webhook routes on the router.
func (h *WebhookHandler) RegisterRoutes(r chi.Router) {
	if h.providerRegistry != nil {
		for _, path := range h.providerRegistry.GetWebhookPaths() {
			h.logger.Info("registering webhook route", zap.String("path", path))
			r.With(middleware.BodySizeLimiterWebhook()).Post(path, h.HandleVoiceWebhook)
		}
	} else {
		// Fallback to legacy Bland-only route
		r.With(middleware.BodySizeLimiterWebhook()).Post("/webhook/bland", h.HandleBlandWebhook)
	}
}

// HandleVoiceWebhook processes incoming webhooks from any voice provider.
func (h *WebhookHandler) HandleVoiceWebhook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if h.providerRegistry == nil {
		h.logger.Error("voice provider registry not configured")
		h.recordWebhookMetrics("unknown", "registry_missing", start)
		http.Error(w, "Voice provider not configured", http.StatusInternalServerError)
		return
	}

	path := r.URL.Path
	provider, err := h.providerRegistry.GetByWebhookPath(path)
	if err != nil {
		h.logger.Warn("unknown webhook path",
			zap.String("path", path),
			zap.Error(err),
		)
		h.recordWebhookMetrics("unknown", "unknown_path", start)
		http.Error(w, "Unknown webhook provider", http.StatusNotFound)
		return
	}

	h.logger.Debug("received voice webhook",
		zap.String("provider", string(provider.GetName())),
		zap.String("content_type", r.Header.Get("Content-Type")),
	)

	// Validate webhook authenticity
	if !provider.ValidateWebhook(r) {
		h.logger.Warn("webhook validation failed",
			zap.String("provider", string(provider.GetName())),
		)
		h.recordWebhookMetrics(string(provider.GetName()), "invalid_signature", start)
		http.Error(w, "Invalid webhook signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook into normalized CallEvent
	event, err := provider.ParseWebhook(r)
	if err != nil {
		h.logger.Error("failed to parse webhook",
			zap.String("provider", string(provider.GetName())),
			zap.Error(err),
		)
		h.recordWebhookMetrics(string(provider.GetName()), "parse_error", start)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	h.logger.Info("processing voice webhook",
		zap.String("provider", string(event.Provider)),
		zap.String("provider_call_id", event.ProviderCallID),
		zap.String("status", string(event.Status)),
	)

	// Process the normalized event
	call, err := h.callService.ProcessCallEvent(r.Context(), event)
	if err != nil {
		h.logger.Error("failed to process webhook",
			zap.Error(err),
			zap.String("provider_call_id", event.ProviderCallID),
		)
		h.recordWebhookMetrics(string(event.Provider), "processing_error", start)
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	if h.metrics != nil {
		h.metrics.RecordProviderCall(string(event.Provider), string(event.Status))
	}

	h.logger.Info("webhook processed successfully",
		zap.String("provider", string(event.Provider)),
		zap.String("provider_call_id", event.ProviderCallID),
		zap.String("internal_id", call.ID.String()),
		zap.String("status", string(call.Status)),
	)

	// Add request ID header if available
	if reqID := GetRequestIDFromContext(r.Context()); reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"call_id":  call.ID.String(),
		"provider": string(event.Provider),
	}); err != nil {
		h.logger.Debug("failed to write webhook response", zap.Error(err))
	}

	h.recordWebhookMetrics(string(event.Provider), "success", start)
}

// HandleBlandWebhook is a convenience endpoint for backward compatibility.
func (h *WebhookHandler) HandleBlandWebhook(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "/webhook/bland"
	h.HandleVoiceWebhook(w, r)
}

func (h *WebhookHandler) recordWebhookMetrics(provider, status string, started time.Time) {
	if h.metrics == nil {
		return
	}
	if provider == "" {
		provider = "unknown"
	}
	h.metrics.RecordWebhook(provider, status, time.Since(started))
}
