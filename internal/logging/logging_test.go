package logging

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
		wantErr  bool
	}{
		{"debug", zapcore.DebugLevel, false},
		{"DEBUG", zapcore.DebugLevel, false},
		{"info", zapcore.InfoLevel, false},
		{"INFO", zapcore.InfoLevel, false},
		{"warn", zapcore.WarnLevel, false},
		{"warning", zapcore.WarnLevel, false},
		{"error", zapcore.ErrorLevel, false},
		{"dpanic", zapcore.DPanicLevel, false},
		{"panic", zapcore.PanicLevel, false},
		{"fatal", zapcore.FatalLevel, false},
		{"invalid", zapcore.InfoLevel, true},
		{"", zapcore.InfoLevel, true},
		{"  info  ", zapcore.InfoLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && level != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Level != "info" {
		t.Errorf("expected Level = info, got %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected Format = json, got %s", cfg.Format)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected Environment = development, got %s", cfg.Environment)
	}
}

func TestNewLogger(t *testing.T) {
	t.Run("nil config uses defaults", func(t *testing.T) {
		logger, err := New(nil)
		if err != nil {
			t.Fatalf("New(nil) error = %v", err)
		}
		if logger == nil {
			t.Fatal("expected logger to be non-nil")
		}
		if logger.GetLevel() != "info" {
			t.Errorf("expected level = info, got %s", logger.GetLevel())
		}
	})

	t.Run("custom config", func(t *testing.T) {
		cfg := &Config{
			Level:       "debug",
			Format:      "console",
			Environment: "production",
		}
		logger, err := New(cfg)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if logger.GetLevel() != "debug" {
			t.Errorf("expected level = debug, got %s", logger.GetLevel())
		}
	})

	t.Run("invalid level returns error", func(t *testing.T) {
		cfg := &Config{
			Level: "invalid",
		}
		_, err := New(cfg)
		if err == nil {
			t.Error("expected error for invalid level")
		}
	})
}

func TestLogger_SetLevel(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})

	t.Run("set valid level", func(t *testing.T) {
		if err := logger.SetLevel("debug"); err != nil {
			t.Errorf("SetLevel(debug) error = %v", err)
		}
		if logger.GetLevel() != "debug" {
			t.Errorf("expected level = debug, got %s", logger.GetLevel())
		}
	})

	t.Run("set invalid level returns error", func(t *testing.T) {
		if err := logger.SetLevel("invalid"); err == nil {
			t.Error("expected error for invalid level")
		}
	})
}

func TestLogger_GetLevel(t *testing.T) {
	logger, _ := New(&Config{Level: "warn"})
	if logger.GetLevel() != "warn" {
		t.Errorf("expected level = warn, got %s", logger.GetLevel())
	}
}

func TestLogger_GetLevelInfo(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})
	info := logger.GetLevelInfo()

	if info.CurrentLevel != "info" {
		t.Errorf("expected CurrentLevel = info, got %s", info.CurrentLevel)
	}
	if len(info.AvailableLevels) != 7 {
		t.Errorf("expected 7 available levels, got %d", len(info.AvailableLevels))
	}
}

func TestLogger_ServeHTTP_Get(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})

	req := httptest.NewRequest(http.MethodGet, "/log/level", nil)
	rec := httptest.NewRecorder()

	logger.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"level":"info"`) {
		t.Errorf("expected response to contain level, got %s", rec.Body.String())
	}
}

func TestLogger_ServeHTTP_Put(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})

	t.Run("set level via query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/log/level?level=debug", nil)
		rec := httptest.NewRecorder()

		logger.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if logger.GetLevel() != "debug" {
			t.Errorf("expected level = debug, got %s", logger.GetLevel())
		}
	})

	t.Run("missing level returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/log/level", nil)
		rec := httptest.NewRecorder()

		logger.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid level returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/log/level?level=invalid", nil)
		rec := httptest.NewRecorder()

		logger.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestLogger_ServeHTTP_Post(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})

	req := httptest.NewRequest(http.MethodPost, "/log/level?level=warn", nil)
	rec := httptest.NewRecorder()

	logger.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if logger.GetLevel() != "warn" {
		t.Errorf("expected level = warn, got %s", logger.GetLevel())
	}
}

func TestLogger_ServeHTTP_MethodNotAllowed(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})

	req := httptest.NewRequest(http.MethodDelete, "/log/level", nil)
	rec := httptest.NewRecorder()

	logger.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestLogger_Named(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})
	named := logger.Named("test")

	if named == nil {
		t.Fatal("expected named logger to be non-nil")
	}
	// Named should share the same level
	if named.GetLevel() != logger.GetLevel() {
		t.Errorf("expected same level, got %s vs %s", named.GetLevel(), logger.GetLevel())
	}

	// Changing level on one should affect the other
	logger.SetLevel("debug")
	if named.GetLevel() != "debug" {
		t.Errorf("expected named logger level to change, got %s", named.GetLevel())
	}
}

func TestLogger_With(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})
	// Just verify it doesn't panic and returns a logger
	withLogger := logger.With()
	if withLogger == nil {
		t.Fatal("expected with logger to be non-nil")
	}
}

func TestLogger_Zap(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})
	if logger.Zap() == nil {
		t.Error("expected Zap() to return non-nil")
	}
}

func TestLogger_AtomicLevel(t *testing.T) {
	logger, _ := New(&Config{Level: "info"})
	level := logger.AtomicLevel()

	// Verify it returns a working atomic level
	level.SetLevel(zapcore.DebugLevel)
	if logger.GetLevel() != "debug" {
		t.Errorf("expected atomic level change to affect logger, got %s", logger.GetLevel())
	}
}
