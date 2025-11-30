package database

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestQueryLoggerConfig_Defaults(t *testing.T) {
	cfg := DefaultQueryLoggerConfig()

	if cfg.SlowQueryThreshold != 100*time.Millisecond {
		t.Errorf("expected SlowQueryThreshold = 100ms, got %v", cfg.SlowQueryThreshold)
	}
	if cfg.VerySlowQueryThreshold != 500*time.Millisecond {
		t.Errorf("expected VerySlowQueryThreshold = 500ms, got %v", cfg.VerySlowQueryThreshold)
	}
	if cfg.LogAllQueries {
		t.Error("expected LogAllQueries = false")
	}
	if cfg.SampleRate != 0.1 {
		t.Errorf("expected SampleRate = 0.1, got %v", cfg.SampleRate)
	}
}

func TestQueryStats_GetStats(t *testing.T) {
	stats := &QueryStats{}

	stats.TotalQueries = 100
	stats.SlowQueries = 5
	stats.VerySlowQueries = 1
	stats.FailedQueries = 2
	stats.TotalDuration = 10 * time.Second

	total, slow, verySlow, failed, avgDuration := stats.GetStats()

	if total != 100 {
		t.Errorf("expected total = 100, got %d", total)
	}
	if slow != 5 {
		t.Errorf("expected slow = 5, got %d", slow)
	}
	if verySlow != 1 {
		t.Errorf("expected verySlow = 1, got %d", verySlow)
	}
	if failed != 2 {
		t.Errorf("expected failed = 2, got %d", failed)
	}
	if avgDuration != 100*time.Millisecond {
		t.Errorf("expected avgDuration = 100ms, got %v", avgDuration)
	}
}

func TestQueryStats_GetSlowestQuery(t *testing.T) {
	stats := &QueryStats{
		slowestQuery:    "SELECT * FROM users",
		slowestDuration: 2 * time.Second,
	}

	query, duration := stats.GetSlowestQuery()

	if query != "SELECT * FROM users" {
		t.Errorf("expected query = 'SELECT * FROM users', got '%s'", query)
	}
	if duration != 2*time.Second {
		t.Errorf("expected duration = 2s, got %v", duration)
	}
}

func TestNewQueryLogger(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Test with nil config
	ql := NewQueryLogger(nil, logger)
	if ql.config == nil {
		t.Error("expected config to be set to defaults")
	}
	if ql.stats == nil {
		t.Error("expected stats to be initialized")
	}

	// Test with custom config
	cfg := &QueryLoggerConfig{
		SlowQueryThreshold: 200 * time.Millisecond,
	}
	ql = NewQueryLogger(cfg, logger)
	if ql.config.SlowQueryThreshold != 200*time.Millisecond {
		t.Errorf("expected SlowQueryThreshold = 200ms, got %v", ql.config.SlowQueryThreshold)
	}
}

func TestQueryLogger_ResetStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ql := NewQueryLogger(nil, logger)

	// Set some stats
	ql.stats.TotalQueries = 100
	ql.stats.SlowQueries = 10
	ql.stats.VerySlowQueries = 2
	ql.stats.FailedQueries = 5
	ql.stats.TotalDuration = 10 * time.Second
	ql.stats.slowestQuery = "SELECT * FROM calls"
	ql.stats.slowestDuration = 5 * time.Second

	// Reset
	ql.ResetStats()

	total, slow, verySlow, failed, avgDuration := ql.stats.GetStats()
	if total != 0 || slow != 0 || verySlow != 0 || failed != 0 || avgDuration != 0 {
		t.Error("expected all stats to be reset to 0")
	}

	query, duration := ql.stats.GetSlowestQuery()
	if query != "" || duration != 0 {
		t.Error("expected slowest query to be reset")
	}
}

func TestTruncateSQL(t *testing.T) {
	tests := []struct {
		sql    string
		maxLen int
		want   string
	}{
		{"SELECT * FROM users", 100, "SELECT * FROM users"},
		{"SELECT * FROM users WHERE id = 1", 20, "SELECT * FROM use..."},
		{"", 10, ""},
		{"short", 5, "short"},
		{"short", 4, "s..."},
	}

	for _, tt := range tests {
		got := truncateSQL(tt.sql, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateSQL(%q, %d) = %q, want %q", tt.sql, tt.maxLen, got, tt.want)
		}
	}
}

func TestQueryLogger_ShouldSample(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Test with sample rate 1.0 (always sample)
	cfg := &QueryLoggerConfig{SampleRate: 1.0}
	ql := NewQueryLogger(cfg, logger)
	for i := 0; i < 10; i++ {
		if !ql.shouldSample() {
			t.Error("expected all queries to be sampled with rate 1.0")
		}
	}

	// Test with sample rate 0 (never sample)
	cfg = &QueryLoggerConfig{SampleRate: 0}
	ql = NewQueryLogger(cfg, logger)
	for i := 0; i < 10; i++ {
		if ql.shouldSample() {
			t.Error("expected no queries to be sampled with rate 0")
		}
	}

	// Test with sample rate 0.5 (should sample some)
	cfg = &QueryLoggerConfig{SampleRate: 0.5}
	ql = NewQueryLogger(cfg, logger)
	sampled := 0
	for i := 0; i < 100; i++ {
		if ql.shouldSample() {
			sampled++
		}
	}
	// With 50% sample rate, we expect roughly half to be sampled
	// Allow for some variance (25-75)
	if sampled < 25 || sampled > 75 {
		t.Errorf("expected roughly 50 samples with 0.5 rate, got %d", sampled)
	}
}
