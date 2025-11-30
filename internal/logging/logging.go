// Package logging provides structured logging with runtime level adjustment.
package logging

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with runtime level adjustment capabilities.
type Logger struct {
	*zap.Logger
	level       zap.AtomicLevel
	mu          sync.RWMutex
	environment string
}

// Config holds configuration for logger initialization.
type Config struct {
	// Level is the initial log level (debug, info, warn, error)
	Level string
	// Format is the output format (json, console)
	Format string
	// Environment is the deployment environment (development, production)
	Environment string
}

// DefaultConfig returns sensible defaults for development.
func DefaultConfig() *Config {
	return &Config{
		Level:       "info",
		Format:      "json",
		Environment: "development",
	}
}

// New creates a new Logger with runtime level adjustment support.
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse initial level
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	atomicLevel := zap.NewAtomicLevelAt(level)

	// Build encoder config
	var encoderConfig zapcore.EncoderConfig
	if cfg.Environment == "production" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// Build encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Build core - write to stderr (standard for logs)
	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stderr),
		atomicLevel,
	)

	// Build logger with appropriate options
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	if cfg.Environment == "development" {
		opts = append(opts, zap.Development())
	}

	zapLogger := zap.New(core, opts...)

	return &Logger{
		Logger:      zapLogger,
		level:       atomicLevel,
		environment: cfg.Environment,
	}, nil
}

// ParseLevel parses a level string into a zapcore.Level.
func ParseLevel(level string) (zapcore.Level, error) {
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
		return zapcore.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

// SetLevel changes the log level at runtime.
func (l *Logger) SetLevel(level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	previousLevel := l.level.String()

	parsed, err := ParseLevel(level)
	if err != nil {
		return err
	}

	l.level.SetLevel(parsed)
	l.Logger.Info("log level changed",
		zap.String("new_level", level),
		zap.String("previous_level", previousLevel),
	)
	return nil
}

// GetLevel returns the current log level.
func (l *Logger) GetLevel() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level.String()
}

// ServeHTTP provides an HTTP handler for level management.
// GET returns current level, PUT/POST sets new level.
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"level":"%s"}`, l.GetLevel())

	case http.MethodPut, http.MethodPost:
		newLevel := r.URL.Query().Get("level")
		if newLevel == "" {
			// Try to read from body
			if err := r.ParseForm(); err == nil {
				newLevel = r.FormValue("level")
			}
		}
		if newLevel == "" {
			http.Error(w, `{"error":"level parameter required"}`, http.StatusBadRequest)
			return
		}

		if err := l.SetLevel(newLevel); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"level":"%s","message":"level updated"}`, l.GetLevel())

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// Named returns a named child logger.
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		Logger:      l.Logger.Named(name),
		level:       l.level,
		environment: l.environment,
	}
}

// With creates a child logger with additional fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		Logger:      l.Logger.With(fields...),
		level:       l.level,
		environment: l.environment,
	}
}

// Zap returns the underlying zap.Logger for compatibility.
func (l *Logger) Zap() *zap.Logger {
	return l.Logger
}

// AtomicLevel returns the atomic level for external integration.
func (l *Logger) AtomicLevel() zap.AtomicLevel {
	return l.level
}

// LevelInfo represents log level information.
type LevelInfo struct {
	CurrentLevel    string   `json:"current_level"`
	AvailableLevels []string `json:"available_levels"`
}

// GetLevelInfo returns information about current and available log levels.
func (l *Logger) GetLevelInfo() LevelInfo {
	return LevelInfo{
		CurrentLevel: l.GetLevel(),
		AvailableLevels: []string{
			"debug",
			"info",
			"warn",
			"error",
			"dpanic",
			"panic",
			"fatal",
		},
	}
}
