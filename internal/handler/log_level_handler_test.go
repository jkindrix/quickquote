package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogLevelHandler_GetLevel(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	req := httptest.NewRequest(http.MethodGet, "/admin/log-level", nil)
	rec := httptest.NewRecorder()

	handler.GetLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp LogLevelResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Level != "info" {
		t.Errorf("expected level = info, got %s", resp.Level)
	}

	if len(resp.AvailableLevels) != 7 {
		t.Errorf("expected 7 available levels, got %d", len(resp.AvailableLevels))
	}
}

func TestLogLevelHandler_SetLevel_QueryParam(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	req := httptest.NewRequest(http.MethodPut, "/admin/log-level?level=debug", nil)
	rec := httptest.NewRecorder()

	handler.SetLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if level.Level() != zapcore.DebugLevel {
		t.Errorf("expected level to be debug, got %s", level.Level().String())
	}
}

func TestLogLevelHandler_SetLevel_JSONBody(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	body := bytes.NewBufferString(`{"level": "warn"}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/log-level", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SetLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if level.Level() != zapcore.WarnLevel {
		t.Errorf("expected level to be warn, got %s", level.Level().String())
	}
}

func TestLogLevelHandler_SetLevel_FormValue(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	body := bytes.NewBufferString("level=error")
	req := httptest.NewRequest(http.MethodPost, "/admin/log-level", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.SetLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if level.Level() != zapcore.ErrorLevel {
		t.Errorf("expected level to be error, got %s", level.Level().String())
	}
}

func TestLogLevelHandler_SetLevel_MissingLevel(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	req := httptest.NewRequest(http.MethodPut, "/admin/log-level", nil)
	rec := httptest.NewRecorder()

	handler.SetLevel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestLogLevelHandler_SetLevel_InvalidLevel(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	req := httptest.NewRequest(http.MethodPut, "/admin/log-level?level=invalid", nil)
	rec := httptest.NewRecorder()

	handler.SetLevel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	// Level should not have changed
	if level.Level() != zapcore.InfoLevel {
		t.Errorf("expected level to remain info, got %s", level.Level().String())
	}
}

func TestLogLevelHandler_ServeHTTP(t *testing.T) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.NewNop()
	handler := NewLogLevelHandler(level, logger)

	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{"GET", http.MethodGet, "/admin/log-level", http.StatusOK},
		{"PUT", http.MethodPut, "/admin/log-level?level=debug", http.StatusOK},
		{"POST", http.MethodPost, "/admin/log-level?level=warn", http.StatusOK},
		{"DELETE", http.MethodDelete, "/admin/log-level", http.StatusMethodNotAllowed},
		{"PATCH", http.MethodPatch, "/admin/log-level", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
		wantErr  bool
	}{
		{"debug", zapcore.DebugLevel, false},
		{"DEBUG", zapcore.DebugLevel, false},
		{"info", zapcore.InfoLevel, false},
		{"warn", zapcore.WarnLevel, false},
		{"warning", zapcore.WarnLevel, false},
		{"error", zapcore.ErrorLevel, false},
		{"dpanic", zapcore.DPanicLevel, false},
		{"panic", zapcore.PanicLevel, false},
		{"fatal", zapcore.FatalLevel, false},
		{"  info  ", zapcore.InfoLevel, false},
		{"invalid", zapcore.InfoLevel, true},
		{"", zapcore.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLogLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLogLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && level != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}
