package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	pingErr error
}

func (m *mockHealthChecker) Ping(ctx context.Context) error {
	return m.pingErr
}

// mockAIHealthChecker implements AIHealthChecker for testing
type mockAIHealthChecker struct {
	circuitOpen bool
}

func (m *mockAIHealthChecker) IsCircuitOpen() bool {
	return m.circuitOpen
}

func TestNewHealthHandler(t *testing.T) {
	logger := zap.NewNop()

	h := NewHealthHandler(HealthHandlerConfig{
		Logger: logger,
	})

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHealthHandler_HandleLiveness(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/live", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleLiveness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "alive" {
		t.Errorf("expected body 'alive', got %q", rr.Body.String())
	}
}

func TestHealthHandler_HandleReadiness_NoHealthChecker(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleReadiness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "ready" {
		t.Errorf("expected body 'ready', got %q", rr.Body.String())
	}
}

func TestHealthHandler_HandleReadiness_HealthyDatabase(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		HealthChecker: &mockHealthChecker{pingErr: nil},
		Logger:        logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleReadiness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHealthHandler_HandleReadiness_UnhealthyDatabase(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		HealthChecker: &mockHealthChecker{pingErr: errors.New("database error")},
		Logger:        logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleReadiness(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestHealthHandler_HandleHealth_AllHealthy(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		HealthChecker:   &mockHealthChecker{pingErr: nil},
		AIHealthChecker: &mockAIHealthChecker{circuitOpen: false},
		Logger:          logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Checks["database"].Status != "healthy" {
		t.Errorf("expected database healthy, got %q", resp.Checks["database"].Status)
	}
	if resp.Checks["ai_service"].Status != "healthy" {
		t.Errorf("expected ai_service healthy, got %q", resp.Checks["ai_service"].Status)
	}
}

func TestHealthHandler_HandleHealth_DatabaseUnhealthy(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		HealthChecker: &mockHealthChecker{pingErr: errors.New("connection refused")},
		Logger:        logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleHealth(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", resp.Status)
	}
	if resp.Checks["database"].Status != "unhealthy" {
		t.Errorf("expected database unhealthy")
	}
}

func TestHealthHandler_HandleHealth_AICircuitOpen(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		HealthChecker:   &mockHealthChecker{pingErr: nil},
		AIHealthChecker: &mockAIHealthChecker{circuitOpen: true},
		Logger:          logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleHealth(rr, req)

	// Should still return OK for degraded state
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", resp.Status)
	}
	if resp.Checks["ai_service"].Status != "degraded" {
		t.Errorf("expected ai_service degraded")
	}
}

func TestHealthHandler_HandleHealth_NoCheckers(t *testing.T) {
	logger := zap.NewNop()
	h := NewHealthHandler(HealthHandlerConfig{
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestHealthResponse_JSONSerialization(t *testing.T) {
	resp := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Checks: map[string]ComponentHealth{
			"database": {Status: "healthy"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("status mismatch")
	}
	if decoded.Version != resp.Version {
		t.Errorf("version mismatch")
	}
}

func TestComponentHealth_JSONSerialization(t *testing.T) {
	ch := ComponentHealth{
		Status:  "unhealthy",
		Message: "connection refused",
	}

	data, err := json.Marshal(ch)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ComponentHealth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != ch.Status {
		t.Errorf("status mismatch")
	}
	if decoded.Message != ch.Message {
		t.Errorf("message mismatch")
	}
}

func TestVoiceProviderHealth_JSONSerialization(t *testing.T) {
	vph := VoiceProviderHealth{
		Name:      "bland",
		Status:    "available",
		IsPrimary: true,
		Message:   "API key configured",
	}

	data, err := json.Marshal(vph)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VoiceProviderHealth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != vph.Name {
		t.Errorf("name mismatch")
	}
	if decoded.IsPrimary != vph.IsPrimary {
		t.Errorf("is_primary mismatch")
	}
}

func TestBaseHandler_WriteJSON(t *testing.T) {
	logger := zap.NewNop()
	h := NewBaseHandler(BaseHandlerConfig{
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	h.WriteJSON(rr, req, http.StatusOK, map[string]string{"foo": "bar"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %q", resp["foo"])
	}
}

func TestBaseHandler_WriteError(t *testing.T) {
	logger := zap.NewNop()
	h := NewBaseHandler(BaseHandlerConfig{
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	h.WriteError(rr, req, http.StatusBadRequest, "test error")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "test error" {
		t.Errorf("expected error='test error', got %q", resp["error"])
	}
	if resp["status"] != float64(http.StatusBadRequest) {
		t.Errorf("expected status=%d, got %v", http.StatusBadRequest, resp["status"])
	}
}
