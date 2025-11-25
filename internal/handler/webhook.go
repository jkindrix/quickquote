package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/webhook"
)

// HandleBlandWebhook processes incoming webhooks from Bland AI.
func (h *Handler) HandleBlandWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	h.logger.Debug("received bland webhook",
		zap.String("content_type", r.Header.Get("Content-Type")),
		zap.Int("body_length", len(body)),
	)

	// Parse payload
	var payload webhook.BlandWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload",
			zap.Error(err),
			zap.String("body", string(body)),
		)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.CallID == "" {
		h.logger.Warn("webhook missing call_id")
		http.Error(w, "Missing call_id", http.StatusBadRequest)
		return
	}

	h.logger.Info("processing bland webhook",
		zap.String("call_id", payload.CallID),
		zap.String("status", payload.Status),
		zap.String("phone_number", payload.GetPhoneNumber()),
	)

	// Process the webhook
	call, err := h.callService.ProcessWebhook(r.Context(), &payload)
	if err != nil {
		h.logger.Error("failed to process webhook",
			zap.Error(err),
			zap.String("call_id", payload.CallID),
		)
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	h.logger.Info("webhook processed successfully",
		zap.String("call_id", payload.CallID),
		zap.String("internal_id", call.ID.String()),
		zap.String("status", string(call.Status)),
	)

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"call_id": call.ID.String(),
	})
}
