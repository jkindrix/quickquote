package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevelHandler handles runtime log level adjustment.
type LogLevelHandler struct {
	level  zap.AtomicLevel
	logger *zap.Logger
}

// NewLogLevelHandler creates a handler for log level management.
func NewLogLevelHandler(level zap.AtomicLevel, logger *zap.Logger) *LogLevelHandler {
	return &LogLevelHandler{
		level:  level,
		logger: logger,
	}
}

// LogLevelResponse is the response for log level queries.
type LogLevelResponse struct {
	Level           string   `json:"level"`
	AvailableLevels []string `json:"available_levels,omitempty"`
	Message         string   `json:"message,omitempty"`
}

// LogLevelRequest is the request body for changing log level.
type LogLevelRequest struct {
	Level string `json:"level"`
}

// GetLevel handles GET requests to return current log level.
func (h *LogLevelHandler) GetLevel(w http.ResponseWriter, r *http.Request) {
	resp := LogLevelResponse{
		Level: h.level.Level().String(),
		AvailableLevels: []string{
			"debug", "info", "warn", "error", "dpanic", "panic", "fatal",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SetLevel handles PUT/POST requests to change log level.
func (h *LogLevelHandler) SetLevel(w http.ResponseWriter, r *http.Request) {
	// Try query parameter first
	levelStr := r.URL.Query().Get("level")

	// Then try form value
	if levelStr == "" {
		if err := r.ParseForm(); err == nil {
			levelStr = r.FormValue("level")
		}
	}

	// Then try JSON body
	if levelStr == "" {
		var req LogLevelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			levelStr = req.Level
		}
	}

	if levelStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "level parameter is required",
		})
		return
	}

	newLevel, err := parseLogLevel(levelStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	previousLevel := h.level.Level().String()
	h.level.SetLevel(newLevel)

	h.logger.Info("log level changed",
		zap.String("previous_level", previousLevel),
		zap.String("new_level", newLevel.String()),
	)

	resp := LogLevelResponse{
		Level:   newLevel.String(),
		Message: fmt.Sprintf("log level changed from %s to %s", previousLevel, newLevel.String()),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ServeHTTP implements http.Handler for the log level endpoint.
func (h *LogLevelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetLevel(w, r)
	case http.MethodPut, http.MethodPost:
		h.SetLevel(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method not allowed",
		})
	}
}

// parseLogLevel parses a string into a zapcore.Level.
func parseLogLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "dpanic":
		return zapcore.DPanicLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %s (valid: debug, info, warn, error, dpanic, panic, fatal)", level)
	}
}
