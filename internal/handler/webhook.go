package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

// HandleVoiceWebhook processes incoming webhooks from any voice provider.
// It uses the provider registry to route webhooks to the appropriate adapter.
func (h *Handler) HandleVoiceWebhook(w http.ResponseWriter, r *http.Request) {
	if h.providerRegistry == nil {
		h.logger.Error("voice provider registry not configured")
		http.Error(w, "Voice provider not configured", http.StatusInternalServerError)
		return
	}

	// Determine provider from URL path
	path := r.URL.Path
	provider, err := h.providerRegistry.GetByWebhookPath(path)
	if err != nil {
		h.logger.Warn("unknown webhook path",
			zap.String("path", path),
			zap.Error(err),
		)
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
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	h.logger.Info("webhook processed successfully",
		zap.String("provider", string(event.Provider)),
		zap.String("provider_call_id", event.ProviderCallID),
		zap.String("internal_id", call.ID.String()),
		zap.String("status", string(call.Status)),
	)

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"call_id":  call.ID.String(),
		"provider": string(event.Provider),
	}); err != nil {
		h.logger.Debug("failed to write webhook response", zap.Error(err))
	}
}

// HandleBlandWebhook is a convenience endpoint that routes directly to Bland.
// This maintains backward compatibility with existing Bland webhook configurations.
func (h *Handler) HandleBlandWebhook(w http.ResponseWriter, r *http.Request) {
	// Rewrite path to use the unified handler
	r.URL.Path = "/webhook/bland"
	h.HandleVoiceWebhook(w, r)
}

// RegisterWebhookRoutes registers all webhook routes for voice providers.
func (h *Handler) RegisterWebhookRoutes(mux interface {
	Post(pattern string, handlerFn http.HandlerFunc)
}) {
	if h.providerRegistry == nil {
		h.logger.Warn("no voice provider registry configured, skipping webhook routes")
		return
	}

	// Register a route for each provider's webhook path
	for _, path := range h.providerRegistry.GetWebhookPaths() {
		h.logger.Info("registering webhook route", zap.String("path", path))
		mux.Post(path, h.HandleVoiceWebhook)
	}
}

// SetProviderRegistry sets the voice provider registry.
func (h *Handler) SetProviderRegistry(registry *voiceprovider.Registry) {
	h.providerRegistry = registry
}

