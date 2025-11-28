package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/service"
)

// BlandAPIHandler handles Bland AI management API endpoints.
type BlandAPIHandler struct {
	blandService *service.BlandService
	logger       *zap.Logger
}

// NewBlandAPIHandler creates a new BlandAPIHandler.
func NewBlandAPIHandler(blandService *service.BlandService, logger *zap.Logger) *BlandAPIHandler {
	return &BlandAPIHandler{
		blandService: blandService,
		logger:       logger,
	}
}

// RegisterRoutes registers all Bland API routes.
func (h *BlandAPIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bland", func(r chi.Router) {
		// Voices
		r.Route("/voices", func(r chi.Router) {
			r.Get("/", h.ListVoices)
			r.Get("/{voiceID}", h.GetVoice)
			r.Post("/clone", h.CloneVoice)
			r.Post("/{voiceID}/sample", h.GenerateVoiceSample)
			r.Delete("/{voiceID}", h.DeleteVoice)
		})

		// Personas
		r.Route("/personas", func(r chi.Router) {
			r.Get("/", h.ListPersonas)
			r.Post("/", h.CreatePersona)
			r.Get("/{personaID}", h.GetPersona)
			r.Put("/{personaID}", h.UpdatePersona)
			r.Delete("/{personaID}", h.DeletePersona)
		})

		// Knowledge Bases
		r.Route("/knowledge-bases", func(r chi.Router) {
			r.Get("/", h.ListKnowledgeBases)
			r.Post("/", h.CreateKnowledgeBase)
			r.Get("/{vectorID}", h.GetKnowledgeBase)
			r.Patch("/{vectorID}", h.UpdateKnowledgeBase)
			r.Delete("/{vectorID}", h.DeleteKnowledgeBase)
		})

		// Pathways
		r.Route("/pathways", func(r chi.Router) {
			r.Get("/", h.ListPathways)
			r.Post("/", h.CreatePathway)
			r.Get("/{pathwayID}", h.GetPathway)
			r.Patch("/{pathwayID}", h.UpdatePathway)
			r.Delete("/{pathwayID}", h.DeletePathway)
			r.Post("/{pathwayID}/publish", h.PublishPathway)
		})

		// Memory
		r.Route("/memory", func(r chi.Router) {
			r.Get("/", h.GetCustomerMemory)
			r.Post("/", h.StoreCustomerMemory)
			r.Delete("/", h.ClearCustomerMemory)
		})

		// Batches
		r.Route("/batches", func(r chi.Router) {
			r.Get("/", h.ListBatches)
			r.Post("/", h.CreateBatch)
			r.Get("/{batchID}", h.GetBatch)
			r.Post("/{batchID}/pause", h.PauseBatch)
			r.Post("/{batchID}/resume", h.ResumeBatch)
			r.Post("/{batchID}/cancel", h.CancelBatch)
			r.Get("/{batchID}/analytics", h.GetBatchAnalytics)
		})

		// SMS
		r.Route("/sms", func(r chi.Router) {
			r.Post("/", h.SendSMS)
			r.Post("/conversation", h.StartSMSConversation)
			r.Get("/conversation/{conversationID}", h.GetSMSConversation)
			r.Post("/conversation/{conversationID}/end", h.EndSMSConversation)
		})

		// Tools
		r.Route("/tools", func(r chi.Router) {
			r.Get("/", h.ListTools)
			r.Post("/", h.CreateTool)
			r.Get("/{toolID}", h.GetTool)
			r.Patch("/{toolID}", h.UpdateTool)
			r.Delete("/{toolID}", h.DeleteTool)
			r.Post("/{toolID}/test", h.TestTool)
		})

		// Phone Numbers
		r.Route("/numbers", func(r chi.Router) {
			r.Get("/", h.ListPhoneNumbers)
			r.Get("/available", h.SearchAvailableNumbers)
			r.Post("/purchase", h.PurchaseNumber)
			r.Get("/{numberID}", h.GetPhoneNumber)
			r.Patch("/{numberID}", h.UpdatePhoneNumber)
			r.Delete("/{numberID}", h.ReleasePhoneNumber)
			r.Post("/{numberID}/configure-inbound", h.ConfigureInboundAgent)
			// Blocked numbers
			r.Get("/blocked", h.ListBlockedNumbers)
			r.Post("/blocked", h.BlockNumber)
			r.Delete("/blocked/{blockedID}", h.UnblockNumber)
		})

		// Citations
		r.Route("/citations", func(r chi.Router) {
			r.Get("/schemas", h.ListCitationSchemas)
			r.Post("/schemas", h.CreateCitationSchema)
			r.Get("/schemas/{schemaID}", h.GetCitationSchema)
			r.Patch("/schemas/{schemaID}", h.UpdateCitationSchema)
			r.Delete("/schemas/{schemaID}", h.DeleteCitationSchema)
			r.Get("/calls/{callID}", h.GetCallCitations)
			r.Post("/calls/{callID}/extract", h.ExtractCitations)
		})

		// Dynamic Data
		r.Route("/dynamic-data", func(r chi.Router) {
			r.Get("/", h.ListDynamicDataSources)
			r.Post("/", h.CreateDynamicDataSource)
			r.Get("/{sourceID}", h.GetDynamicDataSource)
			r.Patch("/{sourceID}", h.UpdateDynamicDataSource)
			r.Delete("/{sourceID}", h.DeleteDynamicDataSource)
			r.Post("/{sourceID}/test", h.TestDynamicDataSource)
			r.Post("/{sourceID}/refresh", h.RefreshDynamicDataSource)
		})

		// Enterprise - Twilio BYOT
		r.Route("/enterprise/twilio", func(r chi.Router) {
			r.Get("/", h.ListTwilioAccounts)
			r.Post("/", h.CreateTwilioAccount)
			r.Get("/{accountID}", h.GetTwilioAccount)
			r.Patch("/{accountID}", h.UpdateTwilioAccount)
			r.Delete("/{accountID}", h.DeleteTwilioAccount)
			r.Post("/{accountID}/verify", h.VerifyTwilioAccount)
		})

		// Enterprise - SIP
		r.Route("/enterprise/sip", func(r chi.Router) {
			r.Get("/", h.ListSIPTrunks)
			r.Post("/", h.CreateSIPTrunk)
			r.Get("/{trunkID}", h.GetSIPTrunk)
			r.Patch("/{trunkID}", h.UpdateSIPTrunk)
			r.Delete("/{trunkID}", h.DeleteSIPTrunk)
			r.Post("/{trunkID}/test", h.TestSIPTrunk)
			r.Get("/{trunkID}/stats", h.GetSIPTrunkStats)
		})

		// Enterprise - Dialing Pools
		r.Route("/enterprise/dialing-pools", func(r chi.Router) {
			r.Get("/", h.ListDialingPools)
			r.Post("/", h.CreateDialingPool)
			r.Get("/{poolID}", h.GetDialingPool)
			r.Patch("/{poolID}", h.UpdateDialingPool)
			r.Delete("/{poolID}", h.DeleteDialingPool)
			r.Post("/{poolID}/numbers", h.AddNumberToPool)
			r.Delete("/{poolID}/numbers/{phoneNumber}", h.RemoveNumberFromPool)
			r.Get("/{poolID}/stats", h.GetDialingPoolStats)
		})

		// Usage & Billing
		r.Route("/usage", func(r chi.Router) {
			r.Get("/summary", h.GetUsageSummary)
			r.Get("/daily", h.GetDailyUsage)
			r.Get("/limits", h.GetUsageLimits)
			r.Post("/limits", h.SetUsageLimit)
			r.Get("/pricing", h.GetPricing)
			r.Get("/alerts", h.GetUsageAlerts)
			r.Post("/alerts", h.SetAlertThreshold)
			r.Post("/alerts/{alertID}/acknowledge", h.AcknowledgeAlert)
			r.Post("/estimate", h.EstimateCallCost)
		})

		// Organization
		r.Route("/organization", func(r chi.Router) {
			r.Get("/", h.GetOrganization)
			r.Get("/members", h.ListOrganizationMembers)
			r.Post("/members/invite", h.InviteOrganizationMember)
			r.Delete("/members/{memberID}", h.RemoveOrganizationMember)
			r.Patch("/members/{memberID}", h.UpdateMemberRole)
		})

		// Circuit breaker stats
		r.Get("/health", h.GetCircuitBreakerStats)
	})
}

// ===============================================
// Voice Handlers
// ===============================================

// ListVoices handles GET /api/v1/bland/voices
func (h *BlandAPIHandler) ListVoices(w http.ResponseWriter, r *http.Request) {
	voices, err := h.blandService.ListVoices(r.Context())
	if err != nil {
		h.logger.Error("failed to list voices", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list voices")
		return
	}
	h.respondJSON(w, http.StatusOK, voices)
}

// GetVoice handles GET /api/v1/bland/voices/{voiceID}
func (h *BlandAPIHandler) GetVoice(w http.ResponseWriter, r *http.Request) {
	voiceID := chi.URLParam(r, "voiceID")
	voice, err := h.blandService.GetVoice(r.Context(), voiceID)
	if err != nil {
		h.logger.Error("failed to get voice", zap.String("voice_id", voiceID), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "voice not found")
		return
	}
	h.respondJSON(w, http.StatusOK, voice)
}

// CloneVoice handles POST /api/v1/bland/voices/clone
func (h *BlandAPIHandler) CloneVoice(w http.ResponseWriter, r *http.Request) {
	var req bland.CloneVoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.CloneVoice(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to clone voice", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to clone voice: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, result)
}

// GenerateVoiceSample handles POST /api/v1/bland/voices/{voiceID}/sample
func (h *BlandAPIHandler) GenerateVoiceSample(w http.ResponseWriter, r *http.Request) {
	voiceID := chi.URLParam(r, "voiceID")
	var req bland.GenerateSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.GenerateVoiceSample(r.Context(), voiceID, &req)
	if err != nil {
		h.logger.Error("failed to generate sample", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to generate sample: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// DeleteVoice handles DELETE /api/v1/bland/voices/{voiceID}
func (h *BlandAPIHandler) DeleteVoice(w http.ResponseWriter, r *http.Request) {
	voiceID := chi.URLParam(r, "voiceID")
	if err := h.blandService.DeleteVoice(r.Context(), voiceID); err != nil {
		h.logger.Error("failed to delete voice", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete voice")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Persona Handlers
// ===============================================

// ListPersonas handles GET /api/v1/bland/personas
func (h *BlandAPIHandler) ListPersonas(w http.ResponseWriter, r *http.Request) {
	personas, err := h.blandService.ListPersonas(r.Context())
	if err != nil {
		h.logger.Error("failed to list personas", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list personas")
		return
	}
	h.respondJSON(w, http.StatusOK, personas)
}

// GetPersona handles GET /api/v1/bland/personas/{personaID}
func (h *BlandAPIHandler) GetPersona(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "personaID")
	persona, err := h.blandService.GetPersona(r.Context(), personaID)
	if err != nil {
		h.logger.Error("failed to get persona", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "persona not found")
		return
	}
	h.respondJSON(w, http.StatusOK, persona)
}

// CreatePersona handles POST /api/v1/bland/personas
func (h *BlandAPIHandler) CreatePersona(w http.ResponseWriter, r *http.Request) {
	var req bland.CreatePersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	persona, err := h.blandService.CreatePersona(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create persona", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create persona: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, persona)
}

// UpdatePersona handles PUT /api/v1/bland/personas/{personaID}
func (h *BlandAPIHandler) UpdatePersona(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "personaID")
	var req bland.UpdatePersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	persona, err := h.blandService.UpdatePersona(r.Context(), personaID, &req)
	if err != nil {
		h.logger.Error("failed to update persona", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update persona: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, persona)
}

// DeletePersona handles DELETE /api/v1/bland/personas/{personaID}
func (h *BlandAPIHandler) DeletePersona(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "personaID")
	if err := h.blandService.DeletePersona(r.Context(), personaID); err != nil {
		h.logger.Error("failed to delete persona", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete persona")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Knowledge Base Handlers
// ===============================================

// ListKnowledgeBases handles GET /api/v1/bland/knowledge-bases
func (h *BlandAPIHandler) ListKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	kbs, err := h.blandService.ListKnowledgeBases(r.Context())
	if err != nil {
		h.logger.Error("failed to list knowledge bases", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list knowledge bases")
		return
	}
	h.respondJSON(w, http.StatusOK, kbs)
}

// GetKnowledgeBase handles GET /api/v1/bland/knowledge-bases/{vectorID}
func (h *BlandAPIHandler) GetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	vectorID := chi.URLParam(r, "vectorID")
	kb, err := h.blandService.GetKnowledgeBase(r.Context(), vectorID)
	if err != nil {
		h.logger.Error("failed to get knowledge base", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "knowledge base not found")
		return
	}
	h.respondJSON(w, http.StatusOK, kb)
}

// CreateKnowledgeBase handles POST /api/v1/bland/knowledge-bases
func (h *BlandAPIHandler) CreateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateKnowledgeBaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.CreateKnowledgeBase(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create knowledge base", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create knowledge base: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, result)
}

// UpdateKnowledgeBase handles PATCH /api/v1/bland/knowledge-bases/{vectorID}
func (h *BlandAPIHandler) UpdateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	vectorID := chi.URLParam(r, "vectorID")
	var req bland.UpdateKnowledgeBaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.UpdateKnowledgeBase(r.Context(), vectorID, &req); err != nil {
		h.logger.Error("failed to update knowledge base", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update knowledge base: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DeleteKnowledgeBase handles DELETE /api/v1/bland/knowledge-bases/{vectorID}
func (h *BlandAPIHandler) DeleteKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	vectorID := chi.URLParam(r, "vectorID")
	if err := h.blandService.DeleteKnowledgeBase(r.Context(), vectorID); err != nil {
		h.logger.Error("failed to delete knowledge base", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete knowledge base")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Pathway Handlers
// ===============================================

// ListPathways handles GET /api/v1/bland/pathways
func (h *BlandAPIHandler) ListPathways(w http.ResponseWriter, r *http.Request) {
	pathways, err := h.blandService.ListPathways(r.Context())
	if err != nil {
		h.logger.Error("failed to list pathways", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list pathways")
		return
	}
	h.respondJSON(w, http.StatusOK, pathways)
}

// GetPathway handles GET /api/v1/bland/pathways/{pathwayID}
func (h *BlandAPIHandler) GetPathway(w http.ResponseWriter, r *http.Request) {
	pathwayID := chi.URLParam(r, "pathwayID")
	pathway, err := h.blandService.GetPathway(r.Context(), pathwayID)
	if err != nil {
		h.logger.Error("failed to get pathway", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "pathway not found")
		return
	}
	h.respondJSON(w, http.StatusOK, pathway)
}

// CreatePathway handles POST /api/v1/bland/pathways
func (h *BlandAPIHandler) CreatePathway(w http.ResponseWriter, r *http.Request) {
	var req bland.CreatePathwayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pathway, err := h.blandService.CreatePathway(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create pathway", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create pathway: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, pathway)
}

// UpdatePathway handles PATCH /api/v1/bland/pathways/{pathwayID}
func (h *BlandAPIHandler) UpdatePathway(w http.ResponseWriter, r *http.Request) {
	pathwayID := chi.URLParam(r, "pathwayID")
	var req bland.UpdatePathwayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pathway, err := h.blandService.UpdatePathway(r.Context(), pathwayID, &req)
	if err != nil {
		h.logger.Error("failed to update pathway", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update pathway: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, pathway)
}

// DeletePathway handles DELETE /api/v1/bland/pathways/{pathwayID}
func (h *BlandAPIHandler) DeletePathway(w http.ResponseWriter, r *http.Request) {
	pathwayID := chi.URLParam(r, "pathwayID")
	if err := h.blandService.DeletePathway(r.Context(), pathwayID); err != nil {
		h.logger.Error("failed to delete pathway", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete pathway")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PublishPathway handles POST /api/v1/bland/pathways/{pathwayID}/publish
func (h *BlandAPIHandler) PublishPathway(w http.ResponseWriter, r *http.Request) {
	pathwayID := chi.URLParam(r, "pathwayID")
	if err := h.blandService.PublishPathway(r.Context(), pathwayID); err != nil {
		h.logger.Error("failed to publish pathway", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to publish pathway")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Memory Handlers
// ===============================================

// GetCustomerMemory handles GET /api/v1/bland/memory?phone_number=...
func (h *BlandAPIHandler) GetCustomerMemory(w http.ResponseWriter, r *http.Request) {
	phoneNumber := r.URL.Query().Get("phone_number")
	if phoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	memory, err := h.blandService.GetCustomerMemory(r.Context(), phoneNumber)
	if err != nil {
		h.logger.Error("failed to get customer memory", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get customer memory")
		return
	}
	h.respondJSON(w, http.StatusOK, memory)
}

// StoreCustomerMemoryRequest is the request body for storing memory.
type StoreCustomerMemoryRequest struct {
	PhoneNumber string                 `json:"phone_number"`
	Data        map[string]interface{} `json:"data"`
}

// StoreCustomerMemory handles POST /api/v1/bland/memory
func (h *BlandAPIHandler) StoreCustomerMemory(w http.ResponseWriter, r *http.Request) {
	var req StoreCustomerMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PhoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	if err := h.blandService.StoreCustomerMemory(r.Context(), req.PhoneNumber, req.Data); err != nil {
		h.logger.Error("failed to store customer memory", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to store customer memory")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ClearCustomerMemory handles DELETE /api/v1/bland/memory?phone_number=...
func (h *BlandAPIHandler) ClearCustomerMemory(w http.ResponseWriter, r *http.Request) {
	phoneNumber := r.URL.Query().Get("phone_number")
	if phoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	if err := h.blandService.ClearCustomerMemory(r.Context(), phoneNumber); err != nil {
		h.logger.Error("failed to clear customer memory", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to clear customer memory")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Batch Handlers
// ===============================================

// ListBatches handles GET /api/v1/bland/batches
func (h *BlandAPIHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	batches, err := h.blandService.ListBatches(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list batches", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list batches")
		return
	}
	h.respondJSON(w, http.StatusOK, batches)
}

// CreateBatch handles POST /api/v1/bland/batches
func (h *BlandAPIHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.CreateBatch(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create batch", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create batch: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, result)
}

// GetBatch handles GET /api/v1/bland/batches/{batchID}
func (h *BlandAPIHandler) GetBatch(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "batchID")
	batch, err := h.blandService.GetBatch(r.Context(), batchID)
	if err != nil {
		h.logger.Error("failed to get batch", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "batch not found")
		return
	}
	h.respondJSON(w, http.StatusOK, batch)
}

// PauseBatch handles POST /api/v1/bland/batches/{batchID}/pause
func (h *BlandAPIHandler) PauseBatch(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "batchID")
	if err := h.blandService.PauseBatch(r.Context(), batchID); err != nil {
		h.logger.Error("failed to pause batch", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to pause batch")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ResumeBatch handles POST /api/v1/bland/batches/{batchID}/resume
func (h *BlandAPIHandler) ResumeBatch(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "batchID")
	if err := h.blandService.ResumeBatch(r.Context(), batchID); err != nil {
		h.logger.Error("failed to resume batch", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to resume batch")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// CancelBatch handles POST /api/v1/bland/batches/{batchID}/cancel
func (h *BlandAPIHandler) CancelBatch(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "batchID")
	if err := h.blandService.CancelBatch(r.Context(), batchID); err != nil {
		h.logger.Error("failed to cancel batch", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to cancel batch")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GetBatchAnalytics handles GET /api/v1/bland/batches/{batchID}/analytics
func (h *BlandAPIHandler) GetBatchAnalytics(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "batchID")
	analytics, err := h.blandService.GetBatchAnalytics(r.Context(), batchID)
	if err != nil {
		h.logger.Error("failed to get batch analytics", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get batch analytics")
		return
	}
	h.respondJSON(w, http.StatusOK, analytics)
}

// ===============================================
// SMS Handlers
// ===============================================

// SendSMS handles POST /api/v1/bland/sms
func (h *BlandAPIHandler) SendSMS(w http.ResponseWriter, r *http.Request) {
	var req bland.SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.SendSMS(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to send SMS", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to send SMS: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// StartSMSConversation handles POST /api/v1/bland/sms/conversation
func (h *BlandAPIHandler) StartSMSConversation(w http.ResponseWriter, r *http.Request) {
	var req bland.StartSMSConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.StartSMSConversation(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to start SMS conversation", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to start SMS conversation: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, result)
}

// GetSMSConversation handles GET /api/v1/bland/sms/conversation/{conversationID}
func (h *BlandAPIHandler) GetSMSConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationID")
	conv, err := h.blandService.GetSMSConversation(r.Context(), conversationID)
	if err != nil {
		h.logger.Error("failed to get SMS conversation", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "conversation not found")
		return
	}
	h.respondJSON(w, http.StatusOK, conv)
}

// EndSMSConversation handles POST /api/v1/bland/sms/conversation/{conversationID}/end
func (h *BlandAPIHandler) EndSMSConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationID")
	if err := h.blandService.EndSMSConversation(r.Context(), conversationID); err != nil {
		h.logger.Error("failed to end SMS conversation", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to end conversation")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Tool Handlers
// ===============================================

// ListTools handles GET /api/v1/bland/tools
func (h *BlandAPIHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.blandService.ListTools(r.Context())
	if err != nil {
		h.logger.Error("failed to list tools", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list tools")
		return
	}
	h.respondJSON(w, http.StatusOK, tools)
}

// GetTool handles GET /api/v1/bland/tools/{toolID}
func (h *BlandAPIHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	toolID := chi.URLParam(r, "toolID")
	tool, err := h.blandService.GetTool(r.Context(), toolID)
	if err != nil {
		h.logger.Error("failed to get tool", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "tool not found")
		return
	}
	h.respondJSON(w, http.StatusOK, tool)
}

// CreateTool handles POST /api/v1/bland/tools
func (h *BlandAPIHandler) CreateTool(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tool, err := h.blandService.CreateTool(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create tool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create tool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, tool)
}

// UpdateTool handles PATCH /api/v1/bland/tools/{toolID}
func (h *BlandAPIHandler) UpdateTool(w http.ResponseWriter, r *http.Request) {
	toolID := chi.URLParam(r, "toolID")
	var req bland.UpdateToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tool, err := h.blandService.UpdateTool(r.Context(), toolID, &req)
	if err != nil {
		h.logger.Error("failed to update tool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update tool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, tool)
}

// DeleteTool handles DELETE /api/v1/bland/tools/{toolID}
func (h *BlandAPIHandler) DeleteTool(w http.ResponseWriter, r *http.Request) {
	toolID := chi.URLParam(r, "toolID")
	if err := h.blandService.DeleteTool(r.Context(), toolID); err != nil {
		h.logger.Error("failed to delete tool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete tool")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// TestToolRequest is the request body for testing a tool.
type TestToolRequest struct {
	Input map[string]interface{} `json:"input"`
}

// TestTool handles POST /api/v1/bland/tools/{toolID}/test
func (h *BlandAPIHandler) TestTool(w http.ResponseWriter, r *http.Request) {
	toolID := chi.URLParam(r, "toolID")
	var req TestToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.TestTool(r.Context(), toolID, req.Input)
	if err != nil {
		h.logger.Error("failed to test tool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to test tool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// ===============================================
// Health Handler
// ===============================================

// GetCircuitBreakerStats handles GET /api/v1/bland/health
func (h *BlandAPIHandler) GetCircuitBreakerStats(w http.ResponseWriter, r *http.Request) {
	stats := h.blandService.CircuitBreakerStats()
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"circuit_breaker": stats,
	})
}

// ===============================================
// Helper Methods
// ===============================================

func (h *BlandAPIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *BlandAPIHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// ===============================================
// Phone Number Handlers
// ===============================================

// ListPhoneNumbers handles GET /api/v1/bland/numbers
func (h *BlandAPIHandler) ListPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	numbers, err := h.blandService.ListPhoneNumbers(r.Context(), nil)
	if err != nil {
		h.logger.Error("failed to list phone numbers", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list phone numbers")
		return
	}
	h.respondJSON(w, http.StatusOK, numbers)
}

// GetPhoneNumber handles GET /api/v1/bland/numbers/{numberID}
func (h *BlandAPIHandler) GetPhoneNumber(w http.ResponseWriter, r *http.Request) {
	numberID := chi.URLParam(r, "numberID")
	number, err := h.blandService.GetPhoneNumber(r.Context(), numberID)
	if err != nil {
		h.logger.Error("failed to get phone number", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "phone number not found")
		return
	}
	h.respondJSON(w, http.StatusOK, number)
}

// SearchAvailableNumbers handles GET /api/v1/bland/numbers/available
func (h *BlandAPIHandler) SearchAvailableNumbers(w http.ResponseWriter, r *http.Request) {
	countryCode := r.URL.Query().Get("country_code")
	if countryCode == "" {
		countryCode = "US"
	}

	req := &bland.SearchAvailableNumbersRequest{
		CountryCode: countryCode,
		AreaCode:    r.URL.Query().Get("area_code"),
		Type:        r.URL.Query().Get("type"),
		Contains:    r.URL.Query().Get("contains"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	numbers, err := h.blandService.SearchAvailableNumbers(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to search available numbers", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to search available numbers")
		return
	}
	h.respondJSON(w, http.StatusOK, numbers)
}

// PurchaseNumber handles POST /api/v1/bland/numbers/purchase
func (h *BlandAPIHandler) PurchaseNumber(w http.ResponseWriter, r *http.Request) {
	var req bland.PurchaseNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	number, err := h.blandService.PurchaseNumber(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to purchase number", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to purchase number: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, number)
}

// UpdatePhoneNumber handles PATCH /api/v1/bland/numbers/{numberID}
func (h *BlandAPIHandler) UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {
	numberID := chi.URLParam(r, "numberID")
	var req bland.UpdatePhoneNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	number, err := h.blandService.UpdatePhoneNumber(r.Context(), numberID, &req)
	if err != nil {
		h.logger.Error("failed to update phone number", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update phone number: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, number)
}

// ReleasePhoneNumber handles DELETE /api/v1/bland/numbers/{numberID}
func (h *BlandAPIHandler) ReleasePhoneNumber(w http.ResponseWriter, r *http.Request) {
	numberID := chi.URLParam(r, "numberID")
	if err := h.blandService.ReleasePhoneNumber(r.Context(), numberID); err != nil {
		h.logger.Error("failed to release phone number", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to release phone number")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ConfigureInboundAgent handles POST /api/v1/bland/numbers/{numberID}/configure-inbound
func (h *BlandAPIHandler) ConfigureInboundAgent(w http.ResponseWriter, r *http.Request) {
	numberID := chi.URLParam(r, "numberID")
	var config bland.InboundConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	number, err := h.blandService.ConfigureInboundAgent(r.Context(), numberID, &config)
	if err != nil {
		h.logger.Error("failed to configure inbound agent", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to configure inbound agent: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, number)
}

// ListBlockedNumbers handles GET /api/v1/bland/numbers/blocked
func (h *BlandAPIHandler) ListBlockedNumbers(w http.ResponseWriter, r *http.Request) {
	numbers, err := h.blandService.ListBlockedNumbers(r.Context())
	if err != nil {
		h.logger.Error("failed to list blocked numbers", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list blocked numbers")
		return
	}
	h.respondJSON(w, http.StatusOK, numbers)
}

// BlockNumber handles POST /api/v1/bland/numbers/blocked
func (h *BlandAPIHandler) BlockNumber(w http.ResponseWriter, r *http.Request) {
	var req bland.BlockNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	blocked, err := h.blandService.BlockNumber(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to block number", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to block number: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, blocked)
}

// UnblockNumber handles DELETE /api/v1/bland/numbers/blocked/{blockedID}
func (h *BlandAPIHandler) UnblockNumber(w http.ResponseWriter, r *http.Request) {
	blockedID := chi.URLParam(r, "blockedID")
	if err := h.blandService.UnblockNumber(r.Context(), blockedID); err != nil {
		h.logger.Error("failed to unblock number", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to unblock number")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Citation Handlers
// ===============================================

// ListCitationSchemas handles GET /api/v1/bland/citations/schemas
func (h *BlandAPIHandler) ListCitationSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := h.blandService.ListCitationSchemas(r.Context())
	if err != nil {
		h.logger.Error("failed to list citation schemas", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list citation schemas")
		return
	}
	h.respondJSON(w, http.StatusOK, schemas)
}

// GetCitationSchema handles GET /api/v1/bland/citations/schemas/{schemaID}
func (h *BlandAPIHandler) GetCitationSchema(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "schemaID")
	schema, err := h.blandService.GetCitationSchema(r.Context(), schemaID)
	if err != nil {
		h.logger.Error("failed to get citation schema", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "schema not found")
		return
	}
	h.respondJSON(w, http.StatusOK, schema)
}

// CreateCitationSchema handles POST /api/v1/bland/citations/schemas
func (h *BlandAPIHandler) CreateCitationSchema(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateCitationSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	schema, err := h.blandService.CreateCitationSchema(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create citation schema", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create citation schema: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, schema)
}

// UpdateCitationSchema handles PATCH /api/v1/bland/citations/schemas/{schemaID}
func (h *BlandAPIHandler) UpdateCitationSchema(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "schemaID")
	var req bland.UpdateCitationSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	schema, err := h.blandService.UpdateCitationSchema(r.Context(), schemaID, &req)
	if err != nil {
		h.logger.Error("failed to update citation schema", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update citation schema: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, schema)
}

// DeleteCitationSchema handles DELETE /api/v1/bland/citations/schemas/{schemaID}
func (h *BlandAPIHandler) DeleteCitationSchema(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "schemaID")
	if err := h.blandService.DeleteCitationSchema(r.Context(), schemaID); err != nil {
		h.logger.Error("failed to delete citation schema", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete citation schema")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GetCallCitations handles GET /api/v1/bland/citations/calls/{callID}
func (h *BlandAPIHandler) GetCallCitations(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	citations, err := h.blandService.GetCallCitations(r.Context(), callID)
	if err != nil {
		h.logger.Error("failed to get call citations", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get call citations")
		return
	}
	h.respondJSON(w, http.StatusOK, citations)
}

// ExtractCitationsRequest is the request body for extracting citations.
type ExtractCitationsRequest struct {
	SchemaIDs []string `json:"schema_ids"`
}

// ExtractCitations handles POST /api/v1/bland/citations/calls/{callID}/extract
func (h *BlandAPIHandler) ExtractCitations(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	var req ExtractCitationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	citations, err := h.blandService.ExtractCitations(r.Context(), callID, req.SchemaIDs)
	if err != nil {
		h.logger.Error("failed to extract citations", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to extract citations: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, citations)
}

// ===============================================
// Dynamic Data Handlers
// ===============================================

// ListDynamicDataSources handles GET /api/v1/bland/dynamic-data
func (h *BlandAPIHandler) ListDynamicDataSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.blandService.ListDynamicDataSources(r.Context())
	if err != nil {
		h.logger.Error("failed to list dynamic data sources", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list dynamic data sources")
		return
	}
	h.respondJSON(w, http.StatusOK, sources)
}

// GetDynamicDataSource handles GET /api/v1/bland/dynamic-data/{sourceID}
func (h *BlandAPIHandler) GetDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	source, err := h.blandService.GetDynamicDataSource(r.Context(), sourceID)
	if err != nil {
		h.logger.Error("failed to get dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "dynamic data source not found")
		return
	}
	h.respondJSON(w, http.StatusOK, source)
}

// CreateDynamicDataSource handles POST /api/v1/bland/dynamic-data
func (h *BlandAPIHandler) CreateDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateDynamicDataSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source, err := h.blandService.CreateDynamicDataSource(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create dynamic data source: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, source)
}

// UpdateDynamicDataSource handles PATCH /api/v1/bland/dynamic-data/{sourceID}
func (h *BlandAPIHandler) UpdateDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	var req bland.UpdateDynamicDataSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source, err := h.blandService.UpdateDynamicDataSource(r.Context(), sourceID, &req)
	if err != nil {
		h.logger.Error("failed to update dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update dynamic data source: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, source)
}

// DeleteDynamicDataSource handles DELETE /api/v1/bland/dynamic-data/{sourceID}
func (h *BlandAPIHandler) DeleteDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	if err := h.blandService.DeleteDynamicDataSource(r.Context(), sourceID); err != nil {
		h.logger.Error("failed to delete dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete dynamic data source")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// TestDynamicDataRequest is the request body for testing dynamic data.
type TestDynamicDataRequest struct {
	Params map[string]interface{} `json:"params"`
}

// TestDynamicDataSource handles POST /api/v1/bland/dynamic-data/{sourceID}/test
func (h *BlandAPIHandler) TestDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	var req TestDynamicDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.blandService.TestDynamicDataSource(r.Context(), sourceID, req.Params)
	if err != nil {
		h.logger.Error("failed to test dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to test dynamic data source: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// RefreshDynamicDataSource handles POST /api/v1/bland/dynamic-data/{sourceID}/refresh
func (h *BlandAPIHandler) RefreshDynamicDataSource(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	if err := h.blandService.RefreshDynamicDataSource(r.Context(), sourceID); err != nil {
		h.logger.Error("failed to refresh dynamic data source", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to refresh dynamic data source")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ===============================================
// Enterprise - Twilio BYOT Handlers
// ===============================================

// ListTwilioAccounts handles GET /api/v1/bland/enterprise/twilio
func (h *BlandAPIHandler) ListTwilioAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.blandService.ListTwilioAccounts(r.Context())
	if err != nil {
		h.logger.Error("failed to list Twilio accounts", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list Twilio accounts")
		return
	}
	h.respondJSON(w, http.StatusOK, accounts)
}

// GetTwilioAccount handles GET /api/v1/bland/enterprise/twilio/{accountID}
func (h *BlandAPIHandler) GetTwilioAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountID")
	account, err := h.blandService.GetTwilioAccount(r.Context(), accountID)
	if err != nil {
		h.logger.Error("failed to get Twilio account", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "Twilio account not found")
		return
	}
	h.respondJSON(w, http.StatusOK, account)
}

// CreateTwilioAccount handles POST /api/v1/bland/enterprise/twilio
func (h *BlandAPIHandler) CreateTwilioAccount(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateTwilioAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.blandService.CreateTwilioAccount(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create Twilio account", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create Twilio account: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, account)
}

// UpdateTwilioAccount handles PATCH /api/v1/bland/enterprise/twilio/{accountID}
func (h *BlandAPIHandler) UpdateTwilioAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountID")
	var req bland.UpdateTwilioAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.blandService.UpdateTwilioAccount(r.Context(), accountID, &req)
	if err != nil {
		h.logger.Error("failed to update Twilio account", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update Twilio account: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, account)
}

// DeleteTwilioAccount handles DELETE /api/v1/bland/enterprise/twilio/{accountID}
func (h *BlandAPIHandler) DeleteTwilioAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountID")
	if err := h.blandService.DeleteTwilioAccount(r.Context(), accountID); err != nil {
		h.logger.Error("failed to delete Twilio account", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete Twilio account")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// VerifyTwilioAccount handles POST /api/v1/bland/enterprise/twilio/{accountID}/verify
func (h *BlandAPIHandler) VerifyTwilioAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountID")
	verified, err := h.blandService.VerifyTwilioAccount(r.Context(), accountID)
	if err != nil {
		h.logger.Error("failed to verify Twilio account", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to verify Twilio account")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]bool{"verified": verified})
}

// ===============================================
// Enterprise - SIP Handlers
// ===============================================

// ListSIPTrunks handles GET /api/v1/bland/enterprise/sip
func (h *BlandAPIHandler) ListSIPTrunks(w http.ResponseWriter, r *http.Request) {
	trunks, err := h.blandService.ListSIPTrunks(r.Context())
	if err != nil {
		h.logger.Error("failed to list SIP trunks", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list SIP trunks")
		return
	}
	h.respondJSON(w, http.StatusOK, trunks)
}

// GetSIPTrunk handles GET /api/v1/bland/enterprise/sip/{trunkID}
func (h *BlandAPIHandler) GetSIPTrunk(w http.ResponseWriter, r *http.Request) {
	trunkID := chi.URLParam(r, "trunkID")
	trunk, err := h.blandService.GetSIPTrunk(r.Context(), trunkID)
	if err != nil {
		h.logger.Error("failed to get SIP trunk", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "SIP trunk not found")
		return
	}
	h.respondJSON(w, http.StatusOK, trunk)
}

// CreateSIPTrunk handles POST /api/v1/bland/enterprise/sip
func (h *BlandAPIHandler) CreateSIPTrunk(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateSIPTrunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trunk, err := h.blandService.CreateSIPTrunk(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create SIP trunk", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create SIP trunk: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, trunk)
}

// UpdateSIPTrunk handles PATCH /api/v1/bland/enterprise/sip/{trunkID}
func (h *BlandAPIHandler) UpdateSIPTrunk(w http.ResponseWriter, r *http.Request) {
	trunkID := chi.URLParam(r, "trunkID")
	var req bland.UpdateSIPTrunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trunk, err := h.blandService.UpdateSIPTrunk(r.Context(), trunkID, &req)
	if err != nil {
		h.logger.Error("failed to update SIP trunk", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update SIP trunk: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, trunk)
}

// DeleteSIPTrunk handles DELETE /api/v1/bland/enterprise/sip/{trunkID}
func (h *BlandAPIHandler) DeleteSIPTrunk(w http.ResponseWriter, r *http.Request) {
	trunkID := chi.URLParam(r, "trunkID")
	if err := h.blandService.DeleteSIPTrunk(r.Context(), trunkID); err != nil {
		h.logger.Error("failed to delete SIP trunk", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete SIP trunk")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// TestSIPTrunk handles POST /api/v1/bland/enterprise/sip/{trunkID}/test
func (h *BlandAPIHandler) TestSIPTrunk(w http.ResponseWriter, r *http.Request) {
	trunkID := chi.URLParam(r, "trunkID")
	connected, err := h.blandService.TestSIPTrunk(r.Context(), trunkID)
	if err != nil {
		h.logger.Error("failed to test SIP trunk", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to test SIP trunk")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]bool{"connected": connected})
}

// GetSIPTrunkStats handles GET /api/v1/bland/enterprise/sip/{trunkID}/stats
func (h *BlandAPIHandler) GetSIPTrunkStats(w http.ResponseWriter, r *http.Request) {
	trunkID := chi.URLParam(r, "trunkID")
	period := r.URL.Query().Get("period")
	stats, err := h.blandService.GetSIPTrunkStats(r.Context(), trunkID, period)
	if err != nil {
		h.logger.Error("failed to get SIP trunk stats", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get SIP trunk stats")
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}

// ===============================================
// Enterprise - Dialing Pool Handlers
// ===============================================

// ListDialingPools handles GET /api/v1/bland/enterprise/dialing-pools
func (h *BlandAPIHandler) ListDialingPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.blandService.ListDialingPools(r.Context())
	if err != nil {
		h.logger.Error("failed to list dialing pools", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list dialing pools")
		return
	}
	h.respondJSON(w, http.StatusOK, pools)
}

// GetDialingPool handles GET /api/v1/bland/enterprise/dialing-pools/{poolID}
func (h *BlandAPIHandler) GetDialingPool(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	pool, err := h.blandService.GetDialingPool(r.Context(), poolID)
	if err != nil {
		h.logger.Error("failed to get dialing pool", zap.Error(err))
		h.respondError(w, http.StatusNotFound, "dialing pool not found")
		return
	}
	h.respondJSON(w, http.StatusOK, pool)
}

// CreateDialingPool handles POST /api/v1/bland/enterprise/dialing-pools
func (h *BlandAPIHandler) CreateDialingPool(w http.ResponseWriter, r *http.Request) {
	var req bland.CreateDialingPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pool, err := h.blandService.CreateDialingPool(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create dialing pool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to create dialing pool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, pool)
}

// UpdateDialingPool handles PATCH /api/v1/bland/enterprise/dialing-pools/{poolID}
func (h *BlandAPIHandler) UpdateDialingPool(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	var req bland.UpdateDialingPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pool, err := h.blandService.UpdateDialingPool(r.Context(), poolID, &req)
	if err != nil {
		h.logger.Error("failed to update dialing pool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update dialing pool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, pool)
}

// DeleteDialingPool handles DELETE /api/v1/bland/enterprise/dialing-pools/{poolID}
func (h *BlandAPIHandler) DeleteDialingPool(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	if err := h.blandService.DeleteDialingPool(r.Context(), poolID); err != nil {
		h.logger.Error("failed to delete dialing pool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete dialing pool")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// AddNumberToPool handles POST /api/v1/bland/enterprise/dialing-pools/{poolID}/numbers
func (h *BlandAPIHandler) AddNumberToPool(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	var number bland.PoolNumber
	if err := json.NewDecoder(r.Body).Decode(&number); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.AddNumberToPool(r.Context(), poolID, &number); err != nil {
		h.logger.Error("failed to add number to pool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to add number to pool: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusCreated, map[string]string{"status": "success"})
}

// RemoveNumberFromPool handles DELETE /api/v1/bland/enterprise/dialing-pools/{poolID}/numbers/{phoneNumber}
func (h *BlandAPIHandler) RemoveNumberFromPool(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	phoneNumber := chi.URLParam(r, "phoneNumber")
	if err := h.blandService.RemoveNumberFromPool(r.Context(), poolID, phoneNumber); err != nil {
		h.logger.Error("failed to remove number from pool", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to remove number from pool")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GetDialingPoolStats handles GET /api/v1/bland/enterprise/dialing-pools/{poolID}/stats
func (h *BlandAPIHandler) GetDialingPoolStats(w http.ResponseWriter, r *http.Request) {
	poolID := chi.URLParam(r, "poolID")
	stats, err := h.blandService.GetDialingPoolStats(r.Context(), poolID)
	if err != nil {
		h.logger.Error("failed to get dialing pool stats", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get dialing pool stats")
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}

// ===============================================
// Usage & Billing Handlers
// ===============================================

// GetUsageSummary handles GET /api/v1/bland/usage/summary
func (h *BlandAPIHandler) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	var req *bland.GetUsageSummaryRequest
	if period := r.URL.Query().Get("period"); period != "" {
		req = &bland.GetUsageSummaryRequest{Period: period}
	}

	summary, err := h.blandService.GetUsageSummary(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to get usage summary", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get usage summary")
		return
	}
	h.respondJSON(w, http.StatusOK, summary)
}

// GetDailyUsage handles GET /api/v1/bland/usage/daily
func (h *BlandAPIHandler) GetDailyUsage(w http.ResponseWriter, r *http.Request) {
	// For simplicity, default to last 30 days
	usage, err := h.blandService.GetDailyUsage(r.Context(), 30)
	if err != nil {
		h.logger.Error("failed to get daily usage", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get daily usage")
		return
	}
	h.respondJSON(w, http.StatusOK, usage)
}

// GetUsageLimits handles GET /api/v1/bland/usage/limits
func (h *BlandAPIHandler) GetUsageLimits(w http.ResponseWriter, r *http.Request) {
	limits, err := h.blandService.GetUsageLimits(r.Context())
	if err != nil {
		h.logger.Error("failed to get usage limits", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get usage limits")
		return
	}
	h.respondJSON(w, http.StatusOK, limits)
}

// SetUsageLimitRequest is the request body for setting usage limits.
type SetUsageLimitRequest struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// SetUsageLimit handles POST /api/v1/bland/usage/limits
func (h *BlandAPIHandler) SetUsageLimit(w http.ResponseWriter, r *http.Request) {
	var req SetUsageLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.SetUsageLimit(r.Context(), req.Type, req.Value); err != nil {
		h.logger.Error("failed to set usage limit", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to set usage limit")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GetPricing handles GET /api/v1/bland/usage/pricing
func (h *BlandAPIHandler) GetPricing(w http.ResponseWriter, r *http.Request) {
	pricing, err := h.blandService.GetPricing(r.Context())
	if err != nil {
		h.logger.Error("failed to get pricing", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get pricing")
		return
	}
	h.respondJSON(w, http.StatusOK, pricing)
}

// GetUsageAlerts handles GET /api/v1/bland/usage/alerts
func (h *BlandAPIHandler) GetUsageAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.blandService.GetUsageAlerts(r.Context())
	if err != nil {
		h.logger.Error("failed to get usage alerts", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get usage alerts")
		return
	}
	h.respondJSON(w, http.StatusOK, alerts)
}

// SetAlertThresholdRequest is the request body for setting alert thresholds.
type SetAlertThresholdRequest struct {
	Type          string  `json:"type"`
	Threshold     float64 `json:"threshold"`
	ThresholdType string  `json:"threshold_type"`
}

// SetAlertThreshold handles POST /api/v1/bland/usage/alerts
func (h *BlandAPIHandler) SetAlertThreshold(w http.ResponseWriter, r *http.Request) {
	var req SetAlertThresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.SetAlertThreshold(r.Context(), req.Type, req.Threshold, req.ThresholdType); err != nil {
		h.logger.Error("failed to set alert threshold", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to set alert threshold")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// AcknowledgeAlert handles POST /api/v1/bland/usage/alerts/{alertID}/acknowledge
func (h *BlandAPIHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")
	if err := h.blandService.AcknowledgeAlert(r.Context(), alertID); err != nil {
		h.logger.Error("failed to acknowledge alert", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// EstimateCallCostRequest is the request body for estimating call cost.
type EstimateCallCostRequest struct {
	DurationMinutes      float64 `json:"duration_minutes"`
	Direction            string  `json:"direction"`
	NumberType           string  `json:"number_type"`
	IncludeTranscription bool    `json:"include_transcription"`
	IncludeAnalysis      bool    `json:"include_analysis"`
}

// EstimateCallCost handles POST /api/v1/bland/usage/estimate
func (h *BlandAPIHandler) EstimateCallCost(w http.ResponseWriter, r *http.Request) {
	var req EstimateCallCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cost, err := h.blandService.EstimateCallCost(r.Context(), req.DurationMinutes, req.Direction,
		req.NumberType, req.IncludeTranscription, req.IncludeAnalysis)
	if err != nil {
		h.logger.Error("failed to estimate call cost", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to estimate call cost")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]float64{"estimated_cost": cost})
}

// ===============================================
// Organization Handlers
// ===============================================

// GetOrganization handles GET /api/v1/bland/organization
func (h *BlandAPIHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	org, err := h.blandService.GetOrganization(r.Context())
	if err != nil {
		h.logger.Error("failed to get organization", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}
	h.respondJSON(w, http.StatusOK, org)
}

// ListOrganizationMembers handles GET /api/v1/bland/organization/members
func (h *BlandAPIHandler) ListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	members, err := h.blandService.ListOrganizationMembers(r.Context())
	if err != nil {
		h.logger.Error("failed to list organization members", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to list organization members")
		return
	}
	h.respondJSON(w, http.StatusOK, members)
}

// InviteMemberRequest is the request body for inviting members.
type InviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InviteOrganizationMember handles POST /api/v1/bland/organization/members/invite
func (h *BlandAPIHandler) InviteOrganizationMember(w http.ResponseWriter, r *http.Request) {
	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.InviteOrganizationMember(r.Context(), req.Email, req.Role); err != nil {
		h.logger.Error("failed to invite organization member", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to invite organization member")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// RemoveOrganizationMember handles DELETE /api/v1/bland/organization/members/{memberID}
func (h *BlandAPIHandler) RemoveOrganizationMember(w http.ResponseWriter, r *http.Request) {
	memberID := chi.URLParam(r, "memberID")
	if err := h.blandService.RemoveOrganizationMember(r.Context(), memberID); err != nil {
		h.logger.Error("failed to remove organization member", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to remove organization member")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// UpdateMemberRoleRequest is the request body for updating member role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

// UpdateMemberRole handles PATCH /api/v1/bland/organization/members/{memberID}
func (h *BlandAPIHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	memberID := chi.URLParam(r, "memberID")
	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.blandService.UpdateMemberRole(r.Context(), memberID, req.Role); err != nil {
		h.logger.Error("failed to update member role", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to update member role")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
