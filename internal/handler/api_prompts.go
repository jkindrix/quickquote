package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/audit"
	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/service"
)

// PromptAPIHandler handles prompt-related API endpoints.
type PromptAPIHandler struct {
	promptService *service.PromptService
	blandService  *service.BlandService
	auditLogger   *audit.Logger
	logger        *zap.Logger
}

// NewPromptAPIHandler creates a new PromptAPIHandler.
func NewPromptAPIHandler(promptService *service.PromptService, auditLogger *audit.Logger, logger *zap.Logger) *PromptAPIHandler {
	return &PromptAPIHandler{
		promptService: promptService,
		auditLogger:   auditLogger,
		logger:        logger,
	}
}

// SetBlandService sets the Bland service for applying presets to inbound numbers.
func (h *PromptAPIHandler) SetBlandService(bs *service.BlandService) {
	h.blandService = bs
}

// RegisterRoutes registers prompt API routes.
func (h *PromptAPIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/prompts", func(r chi.Router) {
		r.Get("/", h.ListPrompts)
		r.Post("/", h.CreatePrompt)
		r.Get("/default", h.GetDefaultPrompt)
		r.Get("/{promptID}", h.GetPrompt)
		r.Put("/{promptID}", h.UpdatePrompt)
		r.Delete("/{promptID}", h.DeletePrompt)
		r.Post("/{promptID}/default", h.SetDefaultPrompt)
		r.Post("/{promptID}/duplicate", h.DuplicatePrompt)
		r.Post("/{promptID}/apply-inbound", h.ApplyToInbound)
	})
}

// ListPrompts handles GET /api/v1/prompts
// @Summary List prompts
// @Description Retrieves a paginated list of prompts
// @Tags prompts
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param active_only query bool false "Only return active prompts" default(true)
// @Success 200 {object} ListPromptsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prompts [get]
func (h *PromptAPIHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	activeOnly := true
	if r.URL.Query().Get("active_only") == "false" {
		activeOnly = false
	}

	prompts, total, err := h.promptService.ListPrompts(r.Context(), page, pageSize, activeOnly)
	if err != nil {
		h.logger.Error("failed to list prompts", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list prompts")
		return
	}

	h.respondJSON(w, http.StatusOK, ListPromptsResponse{
		Prompts:  prompts,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListPromptsResponse is the response for listing prompts.
type ListPromptsResponse struct {
	Prompts  interface{} `json:"prompts"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// CreatePrompt handles POST /api/v1/prompts
// @Summary Create a prompt
// @Description Creates a new AI agent prompt configuration
// @Tags prompts
// @Accept json
// @Produce json
// @Param request body service.CreatePromptRequest true "Prompt configuration"
// @Success 201 {object} domain.Prompt
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prompts [post]
func (h *PromptAPIHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	var req service.CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Task == "" {
		h.respondError(w, http.StatusBadRequest, "task is required")
		return
	}

	prompt, err := h.promptService.CreatePrompt(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create prompt", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create prompt: "+err.Error())
		return
	}

	// Audit log the prompt creation
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		h.auditLogger.PromptCreated(r.Context(), userID, userName, prompt.ID.String(), prompt.Name, getClientIP(r), GetRequestIDFromContext(r.Context()))
	}

	h.respondJSON(w, http.StatusCreated, prompt)
}

// GetPrompt handles GET /api/v1/prompts/{promptID}
// @Summary Get a prompt
// @Description Retrieves a prompt by ID
// @Tags prompts
// @Produce json
// @Param promptID path string true "Prompt ID"
// @Success 200 {object} domain.Prompt
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID} [get]
func (h *PromptAPIHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	prompt, err := h.promptService.GetPrompt(r.Context(), promptID)
	if err != nil {
		h.logger.Error("failed to get prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "prompt not found")
		return
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

// GetDefaultPrompt handles GET /api/v1/prompts/default
// @Summary Get default prompt
// @Description Retrieves the default prompt
// @Tags prompts
// @Produce json
// @Success 200 {object} domain.Prompt
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/prompts/default [get]
func (h *PromptAPIHandler) GetDefaultPrompt(w http.ResponseWriter, r *http.Request) {
	prompt, err := h.promptService.GetDefaultPrompt(r.Context())
	if err != nil {
		h.logger.Error("failed to get default prompt", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "no default prompt configured")
		return
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

// UpdatePrompt handles PUT /api/v1/prompts/{promptID}
// @Summary Update a prompt
// @Description Updates an existing prompt configuration
// @Tags prompts
// @Accept json
// @Produce json
// @Param promptID path string true "Prompt ID"
// @Param request body service.UpdatePromptRequest true "Update fields"
// @Success 200 {object} domain.Prompt
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID} [put]
func (h *PromptAPIHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	var req service.UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	prompt, err := h.promptService.UpdatePrompt(r.Context(), promptID, &req)
	if err != nil {
		h.logger.Error("failed to update prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update prompt: "+err.Error())
		return
	}

	// Audit log the prompt update
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		// Convert request to changes map for audit
		changes := make(map[string]interface{})
		if req.Name != nil {
			changes["name"] = *req.Name
		}
		if req.Task != nil {
			changes["task"] = "(updated)"
		}
		if req.Voice != nil {
			changes["voice"] = *req.Voice
		}
		if req.IsActive != nil {
			changes["is_active"] = *req.IsActive
		}
		h.auditLogger.PromptUpdated(r.Context(), userID, userName, prompt.ID.String(), prompt.Name, getClientIP(r), GetRequestIDFromContext(r.Context()), changes)
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

// DeletePrompt handles DELETE /api/v1/prompts/{promptID}
// @Summary Delete a prompt
// @Description Soft-deletes a prompt
// @Tags prompts
// @Produce json
// @Param promptID path string true "Prompt ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID} [delete]
func (h *PromptAPIHandler) DeletePrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	if err := h.promptService.DeletePrompt(r.Context(), promptID); err != nil {
		h.logger.Error("failed to delete prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete prompt")
		return
	}

	// Audit log the prompt deletion
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		h.auditLogger.PromptDeleted(r.Context(), userID, userName, promptIDStr, "", getClientIP(r), GetRequestIDFromContext(r.Context()))
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "prompt deleted",
	})
}

// SetDefaultPrompt handles POST /api/v1/prompts/{promptID}/default
// @Summary Set default prompt
// @Description Sets a prompt as the default for new calls
// @Tags prompts
// @Produce json
// @Param promptID path string true "Prompt ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID}/default [post]
func (h *PromptAPIHandler) SetDefaultPrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	if err := h.promptService.SetDefaultPrompt(r.Context(), promptID); err != nil {
		h.logger.Error("failed to set default prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to set default prompt")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "default prompt set",
	})
}

// DuplicatePromptRequest is the request body for duplicating a prompt.
type DuplicatePromptRequest struct {
	Name string `json:"name"`
}

// DuplicatePrompt handles POST /api/v1/prompts/{promptID}/duplicate
// @Summary Duplicate a prompt
// @Description Creates a copy of an existing prompt
// @Tags prompts
// @Accept json
// @Produce json
// @Param promptID path string true "Prompt ID to duplicate"
// @Param request body DuplicatePromptRequest true "New prompt name"
// @Success 201 {object} domain.Prompt
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID}/duplicate [post]
func (h *PromptAPIHandler) DuplicatePrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	var req DuplicatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	prompt, err := h.promptService.DuplicatePrompt(r.Context(), promptID, req.Name)
	if err != nil {
		h.logger.Error("failed to duplicate prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to duplicate prompt: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, prompt)
}

func (h *PromptAPIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, data)
}

func (h *PromptAPIHandler) respondError(w http.ResponseWriter, status int, message string) {
	APIError(w, status, message)
}

// ApplyToInboundRequest contains optional phone number override.
type ApplyToInboundRequest struct {
	PhoneNumber string `json:"phone_number,omitempty"` // Optional - defaults to BLAND_INBOUND_NUMBER env var
}

// ApplyToInbound handles POST /api/v1/prompts/{promptID}/apply-inbound
// @Summary Apply prompt to inbound number
// @Description Applies a prompt preset to the configured Bland inbound phone number
// @Tags prompts
// @Accept json
// @Produce json
// @Param promptID path string true "Prompt ID"
// @Param request body ApplyToInboundRequest false "Optional phone number override"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prompts/{promptID}/apply-inbound [post]
func (h *PromptAPIHandler) ApplyToInbound(w http.ResponseWriter, r *http.Request) {
	if h.blandService == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Bland service not configured")
		return
	}

	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	// Parse optional request body
	var req ApplyToInboundRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	// Get phone number from request or environment
	phoneNumber := req.PhoneNumber
	if phoneNumber == "" {
		phoneNumber = os.Getenv("BLAND_INBOUND_NUMBER")
	}
	if phoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "no phone number specified and BLAND_INBOUND_NUMBER not set")
		return
	}

	// Load the prompt
	prompt, err := h.promptService.GetPrompt(r.Context(), promptID)
	if err != nil {
		h.logger.Error("failed to get prompt", zap.String("id", promptIDStr), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "prompt not found")
		return
	}

	// Build inbound config from prompt
	config := &bland.InboundConfig{
		Task:          prompt.Task,
		Voice:         prompt.Voice,
		Language:      prompt.Language,
		Model:         prompt.Model,
		FirstSentence: prompt.FirstSentence,
		WaitForGreeting: prompt.WaitForGreeting,
		Record:        prompt.Record,
		SummaryPrompt: prompt.SummaryPrompt,
		AnalysisSchema: prompt.AnalysisSchema,
		Keywords:      prompt.Keywords,
		KnowledgeBases: prompt.KnowledgeBaseIDs,
		Tools:         prompt.CustomToolIDs,
	}

	// Set optional numeric fields
	if prompt.Temperature != nil {
		config.Temperature = *prompt.Temperature
	}
	if prompt.InterruptionThreshold != nil {
		config.InterruptionThreshold = *prompt.InterruptionThreshold
	}
	if prompt.MaxDuration != nil {
		config.MaxDuration = *prompt.MaxDuration
	}
	if prompt.BackgroundTrack != nil {
		config.BackgroundTrack = *prompt.BackgroundTrack
	}
	config.NoiseCancellation = prompt.NoiseCancellation

	// Apply to Bland inbound number
	result, err := h.blandService.ConfigureInboundAgent(r.Context(), phoneNumber, config)
	if err != nil {
		h.logger.Error("failed to apply prompt to inbound",
			zap.String("prompt_id", promptIDStr),
			zap.String("phone_number", phoneNumber),
			zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to apply prompt: "+err.Error())
		return
	}

	h.logger.Info("applied prompt to inbound number",
		zap.String("prompt_id", promptIDStr),
		zap.String("prompt_name", prompt.Name),
		zap.String("phone_number", phoneNumber))

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "success",
		"message":      "prompt applied to inbound number",
		"prompt_id":    promptID,
		"prompt_name":  prompt.Name,
		"phone_number": phoneNumber,
		"result":       result,
	})
}
