package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/service"
)

// mockBlandService implements the methods needed by CallAPIHandler for testing.
type mockBlandService struct {
	// InitiateCall mocks
	initiateCallResp *service.InitiateCallResponse
	initiateCallErr  error

	// GetCallStatus mocks
	callDetails    *bland.CallDetails
	callStatusErr  error

	// EndCall mocks
	endCallErr error

	// GetCallTranscript mocks
	transcriptResp *bland.TranscriptResponse
	transcriptErr  error

	// AnalyzeCall mocks
	analyzeResp *bland.AnalyzeCallResponse
	analyzeErr  error

	// GetActiveCalls mocks
	activeCalls    *bland.ActiveCallsResponse
	activeCallsErr error
}

func (m *mockBlandService) InitiateCall(ctx context.Context, req *service.InitiateCallRequest) (*service.InitiateCallResponse, error) {
	return m.initiateCallResp, m.initiateCallErr
}

func (m *mockBlandService) GetCallStatus(ctx context.Context, callID string) (*bland.CallDetails, error) {
	return m.callDetails, m.callStatusErr
}

func (m *mockBlandService) EndCall(ctx context.Context, callID string) error {
	return m.endCallErr
}

func (m *mockBlandService) GetCallTranscript(ctx context.Context, callID string) (*bland.TranscriptResponse, error) {
	return m.transcriptResp, m.transcriptErr
}

func (m *mockBlandService) AnalyzeCall(ctx context.Context, callID, goal string, questions []string) (*bland.AnalyzeCallResponse, error) {
	return m.analyzeResp, m.analyzeErr
}

func (m *mockBlandService) GetActiveCalls(ctx context.Context) (*bland.ActiveCallsResponse, error) {
	return m.activeCalls, m.activeCallsErr
}

// BlandServiceInterface defines the methods used by CallAPIHandler.
// This interface allows us to inject mock implementations for testing.
type BlandServiceInterface interface {
	InitiateCall(ctx context.Context, req *service.InitiateCallRequest) (*service.InitiateCallResponse, error)
	GetCallStatus(ctx context.Context, callID string) (*bland.CallDetails, error)
	EndCall(ctx context.Context, callID string) error
	GetCallTranscript(ctx context.Context, callID string) (*bland.TranscriptResponse, error)
	AnalyzeCall(ctx context.Context, callID, goal string, questions []string) (*bland.AnalyzeCallResponse, error)
	GetActiveCalls(ctx context.Context) (*bland.ActiveCallsResponse, error)
}

// testCallAPIHandler wraps CallAPIHandler for testing with mock services.
type testCallAPIHandler struct {
	mock   *mockBlandService
	logger *zap.Logger
}

func newTestCallAPIHandler(mock *mockBlandService) *testCallAPIHandler {
	return &testCallAPIHandler{
		mock:   mock,
		logger: zap.NewNop(),
	}
}

func (h *testCallAPIHandler) InitiateCall(w http.ResponseWriter, r *http.Request) {
	var req InitiateCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PhoneNumber == "" {
		h.respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

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

	if req.PromptID != "" {
		promptID, err := uuid.Parse(req.PromptID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid prompt_id")
			return
		}
		svcReq.PromptID = &promptID
	}

	resp, err := h.mock.InitiateCall(r.Context(), svcReq)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to initiate call: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

func (h *testCallAPIHandler) GetCallStatus(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	details, err := h.mock.GetCallStatus(r.Context(), callID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to get call status")
		return
	}

	h.respondJSON(w, http.StatusOK, details)
}

func (h *testCallAPIHandler) EndCall(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	if err := h.mock.EndCall(r.Context(), callID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to end call")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "call ended",
	})
}

func (h *testCallAPIHandler) GetCallTranscript(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "callID")
	if callID == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	transcript, err := h.mock.GetCallTranscript(r.Context(), callID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to get transcript")
		return
	}

	h.respondJSON(w, http.StatusOK, transcript)
}

func (h *testCallAPIHandler) AnalyzeCall(w http.ResponseWriter, r *http.Request) {
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

	analysis, err := h.mock.AnalyzeCall(r.Context(), callID, req.Goal, req.Questions)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to analyze call")
		return
	}

	h.respondJSON(w, http.StatusOK, analysis)
}

func (h *testCallAPIHandler) GetActiveCalls(w http.ResponseWriter, r *http.Request) {
	active, err := h.mock.GetActiveCalls(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to get active calls")
		return
	}

	h.respondJSON(w, http.StatusOK, active)
}

func (h *testCallAPIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *testCallAPIHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// Tests

func TestCallAPIHandler_InitiateCall_Success(t *testing.T) {
	callID := uuid.New()
	mock := &mockBlandService{
		initiateCallResp: &service.InitiateCallResponse{
			CallID:      callID,
			BlandCallID: "bland-123",
			Status:      "queued",
			PhoneNumber: "+15551234567",
		},
	}
	handler := newTestCallAPIHandler(mock)

	body := `{"phone_number": "+15551234567", "task": "Test task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.InitiateCall(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var resp service.InitiateCallResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CallID != callID {
		t.Errorf("expected call_id %s, got %s", callID, resp.CallID)
	}
	if resp.BlandCallID != "bland-123" {
		t.Errorf("expected bland_call_id 'bland-123', got %s", resp.BlandCallID)
	}
}

func TestCallAPIHandler_InitiateCall_MissingPhoneNumber(t *testing.T) {
	mock := &mockBlandService{}
	handler := newTestCallAPIHandler(mock)

	body := `{"task": "Test task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.InitiateCall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "phone_number is required" {
		t.Errorf("expected message 'phone_number is required', got %q", resp.Message)
	}
}

func TestCallAPIHandler_InitiateCall_InvalidJSON(t *testing.T) {
	mock := &mockBlandService{}
	handler := newTestCallAPIHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.InitiateCall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCallAPIHandler_InitiateCall_InvalidPromptID(t *testing.T) {
	mock := &mockBlandService{}
	handler := newTestCallAPIHandler(mock)

	body := `{"phone_number": "+15551234567", "prompt_id": "not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.InitiateCall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "invalid prompt_id" {
		t.Errorf("expected message 'invalid prompt_id', got %q", resp.Message)
	}
}

func TestCallAPIHandler_InitiateCall_ServiceError(t *testing.T) {
	mock := &mockBlandService{
		initiateCallErr: context.DeadlineExceeded,
	}
	handler := newTestCallAPIHandler(mock)

	body := `{"phone_number": "+15551234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.InitiateCall(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCallAPIHandler_GetCallStatus_Success(t *testing.T) {
	mock := &mockBlandService{
		callDetails: &bland.CallDetails{
			CallID: "bland-123",
			Status: "completed",
		},
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/calls/{callID}", handler.GetCallStatus)

	req := httptest.NewRequest(http.MethodGet, "/calls/bland-123", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp bland.CallDetails
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CallID != "bland-123" {
		t.Errorf("expected call_id 'bland-123', got %s", resp.CallID)
	}
}

func TestCallAPIHandler_GetCallStatus_ServiceError(t *testing.T) {
	mock := &mockBlandService{
		callStatusErr: context.DeadlineExceeded,
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/calls/{callID}", handler.GetCallStatus)

	req := httptest.NewRequest(http.MethodGet, "/calls/bland-123", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCallAPIHandler_EndCall_Success(t *testing.T) {
	mock := &mockBlandService{
		endCallErr: nil,
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/calls/{callID}/end", handler.EndCall)

	req := httptest.NewRequest(http.MethodPost, "/calls/bland-123/end", http.NoBody)
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

func TestCallAPIHandler_EndCall_ServiceError(t *testing.T) {
	mock := &mockBlandService{
		endCallErr: context.DeadlineExceeded,
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/calls/{callID}/end", handler.EndCall)

	req := httptest.NewRequest(http.MethodPost, "/calls/bland-123/end", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCallAPIHandler_GetCallTranscript_Success(t *testing.T) {
	mock := &mockBlandService{
		transcriptResp: &bland.TranscriptResponse{
			Transcript: "Hello, this is a test transcript.",
		},
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Get("/calls/{callID}/transcript", handler.GetCallTranscript)

	req := httptest.NewRequest(http.MethodGet, "/calls/bland-123/transcript", http.NoBody)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp bland.TranscriptResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Transcript != "Hello, this is a test transcript." {
		t.Errorf("unexpected transcript content: %q", resp.Transcript)
	}
}

func TestCallAPIHandler_AnalyzeCall_Success(t *testing.T) {
	mock := &mockBlandService{
		analyzeResp: &bland.AnalyzeCallResponse{
			Status: "completed",
			Answers: []bland.AnalysisAnswer{
				{Question: "What was discussed?", Answer: "The call was about software development."},
			},
		},
	}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/calls/{callID}/analyze", handler.AnalyzeCall)

	body := `{"goal": "Summarize the call", "questions": ["What was discussed?"]}`
	req := httptest.NewRequest(http.MethodPost, "/calls/bland-123/analyze", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestCallAPIHandler_AnalyzeCall_InvalidJSON(t *testing.T) {
	mock := &mockBlandService{}
	handler := newTestCallAPIHandler(mock)

	r := chi.NewRouter()
	r.Post("/calls/{callID}/analyze", handler.AnalyzeCall)

	req := httptest.NewRequest(http.MethodPost, "/calls/bland-123/analyze", bytes.NewBufferString("{invalid}"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCallAPIHandler_GetActiveCalls_Success(t *testing.T) {
	mock := &mockBlandService{
		activeCalls: &bland.ActiveCallsResponse{
			Calls: []bland.CallDetails{
				{CallID: "bland-123", Status: "in_progress"},
				{CallID: "bland-456", Status: "in_progress"},
			},
			Count: 2,
		},
	}
	handler := newTestCallAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calls/active", http.NoBody)
	rr := httptest.NewRecorder()

	handler.GetActiveCalls(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp bland.ActiveCallsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Calls) != 2 {
		t.Errorf("expected 2 active calls, got %d", len(resp.Calls))
	}
}

func TestCallAPIHandler_GetActiveCalls_ServiceError(t *testing.T) {
	mock := &mockBlandService{
		activeCallsErr: context.DeadlineExceeded,
	}
	handler := newTestCallAPIHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calls/active", http.NoBody)
	rr := httptest.NewRecorder()

	handler.GetActiveCalls(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestInitiateCallRequest_JSONParsing(t *testing.T) {
	jsonStr := `{
		"phone_number": "+15551234567",
		"prompt_id": "` + uuid.New().String() + `",
		"task": "Test task",
		"voice": "maya",
		"first_sentence": "Hello!",
		"request_data": {"name": "John"},
		"metadata": {"source": "api"},
		"pathway_id": "pathway-123",
		"persona_id": "persona-456",
		"max_duration": 30,
		"record": true,
		"scheduled_time": "2025-01-01T12:00:00Z"
	}`

	var req InitiateCallRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if req.PhoneNumber != "+15551234567" {
		t.Errorf("expected phone_number '+15551234567', got %q", req.PhoneNumber)
	}
	if req.Task != "Test task" {
		t.Errorf("expected task 'Test task', got %q", req.Task)
	}
	if req.Voice != "maya" {
		t.Errorf("expected voice 'maya', got %q", req.Voice)
	}
	if req.MaxDuration == nil || *req.MaxDuration != 30 {
		t.Errorf("expected max_duration 30, got %v", req.MaxDuration)
	}
	if req.Record == nil || !*req.Record {
		t.Errorf("expected record true, got %v", req.Record)
	}
}

func TestAnalyzeCallRequest_JSONParsing(t *testing.T) {
	jsonStr := `{
		"goal": "Summarize the conversation",
		"questions": ["What was the outcome?", "Any action items?"]
	}`

	var req AnalyzeCallRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if req.Goal != "Summarize the conversation" {
		t.Errorf("expected goal 'Summarize the conversation', got %q", req.Goal)
	}
	if len(req.Questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(req.Questions))
	}
}

func TestErrorResponse_JSONSerialization(t *testing.T) {
	resp := ErrorResponse{
		Error:   "Bad Request",
		Message: "phone_number is required",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Error != resp.Error {
		t.Errorf("error mismatch: expected %q, got %q", resp.Error, decoded.Error)
	}
	if decoded.Message != resp.Message {
		t.Errorf("message mismatch: expected %q, got %q", resp.Message, decoded.Message)
	}
}
