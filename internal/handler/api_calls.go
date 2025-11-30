package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/audit"
	"github.com/jkindrix/quickquote/internal/service"
)

// CallAPIHandler handles call-related API endpoints.
type CallAPIHandler struct {
	blandService *service.BlandService
	auditLogger  *audit.Logger
	logger       *zap.Logger
}

// NewCallAPIHandler creates a new CallAPIHandler.
func NewCallAPIHandler(blandService *service.BlandService, auditLogger *audit.Logger, logger *zap.Logger) *CallAPIHandler {
	return &CallAPIHandler{
		blandService: blandService,
		auditLogger:  auditLogger,
		logger:       logger,
	}
}

// RegisterRoutes registers call API routes.
func (h *CallAPIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/calls", func(r chi.Router) {
		r.Post("/", h.InitiateCall)
		r.Get("/active", h.GetActiveCalls)
		r.Get("/{callID}", h.GetCallStatus)
		r.Post("/{callID}/end", h.EndCall)
		r.Get("/{callID}/transcript", h.GetCallTranscript)
		r.Post("/{callID}/analyze", h.AnalyzeCall)
	})
}

// InitiateCallRequest is the API request body for initiating a call.
type InitiateCallRequest struct {
	PhoneNumber   string                 `json:"phone_number"`
	PromptID      string                 `json:"prompt_id,omitempty"`
	Task          string                 `json:"task,omitempty"`
	Voice         string                 `json:"voice,omitempty"`
	FirstSentence string                 `json:"first_sentence,omitempty"`
	RequestData   map[string]interface{} `json:"request_data,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	PathwayID     string                 `json:"pathway_id,omitempty"`
	PersonaID     string                 `json:"persona_id,omitempty"`
	MaxDuration   *int                   `json:"max_duration,omitempty"`
	Record        *bool                  `json:"record,omitempty"`
	ScheduledTime string                 `json:"scheduled_time,omitempty"`
}

// InitiateCall handles POST /api/v1/calls
// @Summary Initiate a new outbound call
// @Description Starts a new AI-powered voice call via Bland AI
// @Tags calls
// @Accept json
// @Produce json
// @Param request body InitiateCallRequest true "Call initiation request"
// @Success 201 {object} service.InitiateCallResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls [post]
func (h *CallAPIHandler) InitiateCall(w http.ResponseWriter, r *http.Request) {
	var req InitiateCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.PhoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	// Build service request
	svcReq := &service.InitiateCallRequest{
		PhoneNumber:   req.PhoneNumber,
		Task:          req.Task,
		Voice:         req.Voice,
		FirstSentence: req.FirstSentence,
		RequestData:   req.RequestData,
		Metadata:      req.Metadata,
		PathwayID:     req.PathwayID,
		PersonaID:     req.PersonaID,
		MaxDuration:   req.MaxDuration,
		Record:        req.Record,
		ScheduledTime: req.ScheduledTime,
	}

	// Parse prompt ID if provided
	if req.PromptID != "" {
		promptID, err := uuid.Parse(req.PromptID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
			return
		}
		svcReq.PromptID = &promptID
	}

	// Initiate the call
	resp, err := h.blandService.InitiateCall(r.Context(), svcReq)
	if err != nil {
		h.logger.Error("failed to initiate call", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to initiate call: "+err.Error())
		return
	}

	// Audit log the call initiation
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		h.auditLogger.CallInitiated(r.Context(), userID, userName, resp.CallID.String(), req.PhoneNumber, getClientIP(r), GetRequestIDFromContext(r.Context()))
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// GetCallStatus handles GET /api/v1/calls/{callID}
// @Summary Get call status
// @Description Retrieves the current status of a call
// @Tags calls
// @Produce json
// @Param callID path string true "Bland Call ID"
// @Success 200 {object} bland.CallDetails
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/calls/{callID} [get]
func (h *CallAPIHandler) GetCallStatus(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	details, err := h.blandService.GetCallStatus(r.Context(), callID)
	if err != nil {
		h.logger.Error("failed to get call status", zap.String("call_id", callID), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get call status")
		return
	}

	h.respondJSON(w, http.StatusOK, details)
}

// EndCall handles POST /api/v1/calls/{callID}/end
// @Summary End an active call
// @Description Terminates an ongoing call
// @Tags calls
// @Produce json
// @Param callID path string true "Bland Call ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls/{callID}/end [post]
func (h *CallAPIHandler) EndCall(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	if err := h.blandService.EndCall(r.Context(), callID); err != nil {
		h.logger.Error("failed to end call", zap.String("call_id", callID), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to end call")
		return
	}

	// Audit log the call termination
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		h.auditLogger.CallEnded(r.Context(), userID, userName, callID, getClientIP(r), GetRequestIDFromContext(r.Context()))
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "call ended",
	})
}

// GetCallTranscript handles GET /api/v1/calls/{callID}/transcript
// @Summary Get call transcript
// @Description Retrieves the transcript for a completed call
// @Tags calls
// @Produce json
// @Param callID path string true "Bland Call ID"
// @Success 200 {object} bland.TranscriptResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls/{callID}/transcript [get]
func (h *CallAPIHandler) GetCallTranscript(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	transcript, err := h.blandService.GetCallTranscript(r.Context(), callID)
	if err != nil {
		h.logger.Error("failed to get transcript", zap.String("call_id", callID), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get transcript")
		return
	}

	h.respondJSON(w, http.StatusOK, transcript)
}

// AnalyzeCallRequest is the request body for analyzing a call.
type AnalyzeCallRequest struct {
	Goal      string   `json:"goal,omitempty"`
	Questions []string `json:"questions,omitempty"`
}

// AnalyzeCall handles POST /api/v1/calls/{callID}/analyze
// @Summary Analyze a completed call
// @Description Performs post-call analysis to extract insights
// @Tags calls
// @Accept json
// @Produce json
// @Param callID path string true "Bland Call ID"
// @Param request body AnalyzeCallRequest true "Analysis parameters"
// @Success 200 {object} bland.AnalyzeCallResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls/{callID}/analyze [post]
func (h *CallAPIHandler) AnalyzeCall(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	var req AnalyzeCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	analysis, err := h.blandService.AnalyzeCall(r.Context(), callID, req.Goal, req.Questions)
	if err != nil {
		h.logger.Error("failed to analyze call", zap.String("call_id", callID), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to analyze call")
		return
	}

	// Audit log the call analysis
	if h.auditLogger != nil {
		user := GetUserFromContext(r.Context())
		userID, userName := "", ""
		if user != nil {
			userID = user.ID.String()
			userName = user.Email
		}
		h.auditLogger.CallAnalyzed(r.Context(), userID, userName, callID, getClientIP(r), GetRequestIDFromContext(r.Context()))
	}

	h.respondJSON(w, http.StatusOK, analysis)
}

// GetActiveCalls handles GET /api/v1/calls/active
// @Summary Get active calls
// @Description Retrieves all currently active calls
// @Tags calls
// @Produce json
// @Success 200 {object} bland.ActiveCallsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls/active [get]
func (h *CallAPIHandler) GetActiveCalls(w http.ResponseWriter, r *http.Request) {
	active, err := h.blandService.GetActiveCalls(r.Context())
	if err != nil {
		h.logger.Error("failed to get active calls", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get active calls")
		return
	}

	h.respondJSON(w, http.StatusOK, active)
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func (h *CallAPIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, data)
}

func (h *CallAPIHandler) respondError(w http.ResponseWriter, status int, message string) {
	APIError(w, status, message)
}
