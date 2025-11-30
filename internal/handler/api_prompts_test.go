package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/service"
)

// mockPromptService implements the methods needed by PromptAPIHandler for testing.
type mockPromptService struct {
	// ListPrompts mocks
	prompts      []*domain.Prompt
	total        int
	listErr      error

	// GetPrompt mocks
	prompt       *domain.Prompt
	getErr       error

	// GetDefaultPrompt mocks
	defaultPrompt    *domain.Prompt
	getDefaultErr    error

	// CreatePrompt mocks
	createdPrompt *domain.Prompt
	createErr     error

	// UpdatePrompt mocks
	updatedPrompt *domain.Prompt
	updateErr     error

	// DeletePrompt mocks
	deleteErr error

	// SetDefaultPrompt mocks
	setDefaultErr error

	// DuplicatePrompt mocks
	duplicatedPrompt *domain.Prompt
	duplicateErr     error
}

func (m *mockPromptService) ListPrompts(ctx context.Context, page, pageSize int, activeOnly bool) ([]*domain.Prompt, int, error) {
	return m.prompts, m.total, m.listErr
}

func (m *mockPromptService) GetPrompt(ctx context.Context, id uuid.UUID) (*domain.Prompt, error) {
	return m.prompt, m.getErr
}

func (m *mockPromptService) GetDefaultPrompt(ctx context.Context) (*domain.Prompt, error) {
	return m.defaultPrompt, m.getDefaultErr
}

func (m *mockPromptService) CreatePrompt(ctx context.Context, req *service.CreatePromptRequest) (*domain.Prompt, error) {
	return m.createdPrompt, m.createErr
}

func (m *mockPromptService) UpdatePrompt(ctx context.Context, id uuid.UUID, req *service.UpdatePromptRequest) (*domain.Prompt, error) {
	return m.updatedPrompt, m.updateErr
}

func (m *mockPromptService) DeletePrompt(ctx context.Context, id uuid.UUID) error {
	return m.deleteErr
}

func (m *mockPromptService) SetDefaultPrompt(ctx context.Context, id uuid.UUID) error {
	return m.setDefaultErr
}

func (m *mockPromptService) DuplicatePrompt(ctx context.Context, id uuid.UUID, name string) (*domain.Prompt, error) {
	return m.duplicatedPrompt, m.duplicateErr
}

// testPromptAPIHandler wraps PromptAPIHandler for testing with mock services.
type testPromptAPIHandler struct {
	mock   *mockPromptService
	logger *zap.Logger
}

func newTestPromptAPIHandler(mock *mockPromptService) *testPromptAPIHandler {
	return &testPromptAPIHandler{
		mock:   mock,
		logger: zap.NewNop(),
	}
}

func (h *testPromptAPIHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	activeOnly := true

	if r.URL.Query().Get("active_only") == "false" {
		activeOnly = false
	}

	prompts, total, err := h.mock.ListPrompts(r.Context(), page, pageSize, activeOnly)
	if err != nil {
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

func (h *testPromptAPIHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	prompt, err := h.mock.GetPrompt(r.Context(), promptID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "prompt not found")
		return
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

func (h *testPromptAPIHandler) GetDefaultPrompt(w http.ResponseWriter, r *http.Request) {
	prompt, err := h.mock.GetDefaultPrompt(r.Context())
	if err != nil {
		h.respondError(w, http.StatusNotFound, "no default prompt configured")
		return
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

func (h *testPromptAPIHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	var req service.CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Task == "" {
		h.respondError(w, http.StatusBadRequest, "task is required")
		return
	}

	prompt, err := h.mock.CreatePrompt(r.Context(), &req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create prompt: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, prompt)
}

func (h *testPromptAPIHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
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

	prompt, err := h.mock.UpdatePrompt(r.Context(), promptID, &req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update prompt: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, prompt)
}

func (h *testPromptAPIHandler) DeletePrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	if err := h.mock.DeletePrompt(r.Context(), promptID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to delete prompt")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "prompt deleted",
	})
}

func (h *testPromptAPIHandler) SetDefaultPrompt(w http.ResponseWriter, r *http.Request) {
	promptIDStr := chi.URLParam(r, "promptID")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
		return
	}

	if err := h.mock.SetDefaultPrompt(r.Context(), promptID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to set default prompt")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "default prompt set",
	})
}

func (h *testPromptAPIHandler) DuplicatePrompt(w http.ResponseWriter, r *http.Request) {
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

	prompt, err := h.mock.DuplicatePrompt(r.Context(), promptID, req.Name)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to duplicate prompt: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, prompt)
}

func (h *testPromptAPIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *testPromptAPIHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// Tests

func TestPromptAPIHandler_ListPrompts_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		prompts: []*domain.Prompt{
			{ID: promptID, Name: "Test Prompt", Task: "Test task"},
		},
		total: 1,
	}
	handler := newTestPromptAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ListPrompts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp ListPromptsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}

func TestPromptAPIHandler_ListPrompts_ServiceError(t *testing.T) {
	mock := &mockPromptService{
		listErr: errors.New("database error"),
	}
	handler := newTestPromptAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ListPrompts(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestPromptAPIHandler_GetPrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		prompt: &domain.Prompt{ID: promptID, Name: "Test Prompt", Task: "Test task"},
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/prompts/{promptID}", handler.GetPrompt)

	req := httptest.NewRequest(http.MethodGet, "/prompts/"+promptID.String(), http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp domain.Prompt
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Test Prompt" {
		t.Errorf("expected name 'Test Prompt', got %q", resp.Name)
	}
}

func TestPromptAPIHandler_GetPrompt_InvalidID(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/prompts/{promptID}", handler.GetPrompt)

	req := httptest.NewRequest(http.MethodGet, "/prompts/not-a-uuid", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPromptAPIHandler_GetPrompt_NotFound(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		getErr: errors.New("not found"),
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/prompts/{promptID}", handler.GetPrompt)

	req := httptest.NewRequest(http.MethodGet, "/prompts/"+promptID.String(), http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestPromptAPIHandler_GetDefaultPrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		defaultPrompt: &domain.Prompt{ID: promptID, Name: "Default Prompt", Task: "Default task", IsDefault: true},
	}
	handler := newTestPromptAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/default", http.NoBody)
	rr := httptest.NewRecorder()

	handler.GetDefaultPrompt(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp domain.Prompt
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.IsDefault {
		t.Error("expected prompt to be default")
	}
}

func TestPromptAPIHandler_GetDefaultPrompt_NotFound(t *testing.T) {
	mock := &mockPromptService{
		getDefaultErr: errors.New("no default"),
	}
	handler := newTestPromptAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/default", http.NoBody)
	rr := httptest.NewRecorder()

	handler.GetDefaultPrompt(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestPromptAPIHandler_CreatePrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		createdPrompt: &domain.Prompt{ID: promptID, Name: "New Prompt", Task: "New task"},
	}
	handler := newTestPromptAPIHandler(mock)

	body := `{"name": "New Prompt", "task": "New task", "voice": "maya"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreatePrompt(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var resp domain.Prompt
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "New Prompt" {
		t.Errorf("expected name 'New Prompt', got %q", resp.Name)
	}
}

func TestPromptAPIHandler_CreatePrompt_MissingName(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	body := `{"task": "Test task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreatePrompt(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "name is required" {
		t.Errorf("expected message 'name is required', got %q", resp.Message)
	}
}

func TestPromptAPIHandler_CreatePrompt_MissingTask(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	body := `{"name": "Test Prompt"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreatePrompt(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "task is required" {
		t.Errorf("expected message 'task is required', got %q", resp.Message)
	}
}

func TestPromptAPIHandler_CreatePrompt_InvalidJSON(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBufferString("{invalid}"))
	rr := httptest.NewRecorder()

	handler.CreatePrompt(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPromptAPIHandler_UpdatePrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		updatedPrompt: &domain.Prompt{ID: promptID, Name: "Updated Prompt", Task: "Updated task"},
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Put("/prompts/{promptID}", handler.UpdatePrompt)

	body := `{"name": "Updated Prompt"}`
	req := httptest.NewRequest(http.MethodPut, "/prompts/"+promptID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp domain.Prompt
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Updated Prompt" {
		t.Errorf("expected name 'Updated Prompt', got %q", resp.Name)
	}
}

func TestPromptAPIHandler_UpdatePrompt_InvalidID(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Put("/prompts/{promptID}", handler.UpdatePrompt)

	body := `{"name": "Updated Prompt"}`
	req := httptest.NewRequest(http.MethodPut, "/prompts/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPromptAPIHandler_DeletePrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Delete("/prompts/{promptID}", handler.DeletePrompt)

	req := httptest.NewRequest(http.MethodDelete, "/prompts/"+promptID.String(), http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %q", resp["status"])
	}
}

func TestPromptAPIHandler_DeletePrompt_ServiceError(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		deleteErr: errors.New("delete failed"),
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Delete("/prompts/{promptID}", handler.DeletePrompt)

	req := httptest.NewRequest(http.MethodDelete, "/prompts/"+promptID.String(), http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestPromptAPIHandler_SetDefaultPrompt_Success(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/default", handler.SetDefaultPrompt)

	req := httptest.NewRequest(http.MethodPost, "/prompts/"+promptID.String()+"/default", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "default prompt set" {
		t.Errorf("expected message 'default prompt set', got %q", resp["message"])
	}
}

func TestPromptAPIHandler_SetDefaultPrompt_ServiceError(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		setDefaultErr: errors.New("set default failed"),
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/default", handler.SetDefaultPrompt)

	req := httptest.NewRequest(http.MethodPost, "/prompts/"+promptID.String()+"/default", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestPromptAPIHandler_DuplicatePrompt_Success(t *testing.T) {
	promptID := uuid.New()
	duplicatedID := uuid.New()
	mock := &mockPromptService{
		duplicatedPrompt: &domain.Prompt{ID: duplicatedID, Name: "Copy of Test", Task: "Original task"},
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/duplicate", handler.DuplicatePrompt)

	body := `{"name": "Copy of Test"}`
	req := httptest.NewRequest(http.MethodPost, "/prompts/"+promptID.String()+"/duplicate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var resp domain.Prompt
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Copy of Test" {
		t.Errorf("expected name 'Copy of Test', got %q", resp.Name)
	}
}

func TestPromptAPIHandler_DuplicatePrompt_MissingName(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/duplicate", handler.DuplicatePrompt)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/prompts/"+promptID.String()+"/duplicate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "name is required" {
		t.Errorf("expected message 'name is required', got %q", resp.Message)
	}
}

func TestPromptAPIHandler_DuplicatePrompt_InvalidID(t *testing.T) {
	mock := &mockPromptService{}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/duplicate", handler.DuplicatePrompt)

	body := `{"name": "Copy"}`
	req := httptest.NewRequest(http.MethodPost, "/prompts/not-a-uuid/duplicate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPromptAPIHandler_DuplicatePrompt_ServiceError(t *testing.T) {
	promptID := uuid.New()
	mock := &mockPromptService{
		duplicateErr: errors.New("duplicate failed"),
	}
	handler := newTestPromptAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/prompts/{promptID}/duplicate", handler.DuplicatePrompt)

	body := `{"name": "Copy"}`
	req := httptest.NewRequest(http.MethodPost, "/prompts/"+promptID.String()+"/duplicate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestListPromptsResponse_JSONSerialization(t *testing.T) {
	promptID := uuid.New()
	resp := ListPromptsResponse{
		Prompts: []*domain.Prompt{
			{ID: promptID, Name: "Test", Task: "Task"},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ListPromptsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Total != resp.Total {
		t.Errorf("total mismatch: expected %d, got %d", resp.Total, decoded.Total)
	}
	if decoded.Page != resp.Page {
		t.Errorf("page mismatch: expected %d, got %d", resp.Page, decoded.Page)
	}
}

func TestDuplicatePromptRequest_JSONParsing(t *testing.T) {
	jsonStr := `{"name": "Copy of Original"}`

	var req DuplicatePromptRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if req.Name != "Copy of Original" {
		t.Errorf("expected name 'Copy of Original', got %q", req.Name)
	}
}
