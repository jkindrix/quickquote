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

func TestNew(t *testing.T) {
	logger := zap.NewNop()

	h := New(nil, nil, logger)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestHandler_SetHealthChecker(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)

	hc := &mockHealthChecker{}
	h.SetHealthChecker(hc)

	if h.healthChecker != hc {
		t.Error("expected health checker to be set")
	}
}

func TestHandler_SetAIHealthChecker(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)

	ahc := &mockAIHealthChecker{}
	h.SetAIHealthChecker(ahc)

	if h.aiHealthChecker != ahc {
		t.Error("expected AI health checker to be set")
	}
}

func TestHandler_HandleLiveness(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)

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

func TestHandler_HandleReadiness_NoHealthChecker(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)

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

func TestHandler_HandleReadiness_HealthyDatabase(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)
	h.SetHealthChecker(&mockHealthChecker{pingErr: nil})

	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleReadiness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandler_HandleReadiness_UnhealthyDatabase(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)
	h.SetHealthChecker(&mockHealthChecker{pingErr: errors.New("database error")})

	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	rr := httptest.NewRecorder()

	h.HandleReadiness(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestHandler_HandleHealth_AllHealthy(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)
	h.SetHealthChecker(&mockHealthChecker{pingErr: nil})
	h.SetAIHealthChecker(&mockAIHealthChecker{circuitOpen: false})

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

func TestHandler_HandleHealth_DatabaseUnhealthy(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)
	h.SetHealthChecker(&mockHealthChecker{pingErr: errors.New("connection refused")})

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

func TestHandler_HandleHealth_AICircuitOpen(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)
	h.SetHealthChecker(&mockHealthChecker{pingErr: nil})
	h.SetAIHealthChecker(&mockAIHealthChecker{circuitOpen: true})

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

func TestHandler_HandleHealth_NoCheckers(t *testing.T) {
	logger := zap.NewNop()
	h := New(nil, nil, logger)

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
