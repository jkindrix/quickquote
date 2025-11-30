package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return zap.New(core), logs
}

func TestBusinessEventLogger_CallReceived(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	callID := uuid.New()
	bel.CallReceived(context.Background(), callID, "bland", "+15551234567")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "call_received" {
		t.Errorf("expected message 'call_received', got '%s'", entry.Message)
	}

	// Check fields
	fields := entry.ContextMap()
	if fields["event_type"] != "call.received" {
		t.Errorf("expected event_type 'call.received', got '%v'", fields["event_type"])
	}
	if fields["call_id"] != callID.String() {
		t.Errorf("expected call_id '%s', got '%v'", callID.String(), fields["call_id"])
	}
	if fields["provider"] != "bland" {
		t.Errorf("expected provider 'bland', got '%v'", fields["provider"])
	}
	// Phone should be masked
	if fields["from_number"] != "+15****67" {
		t.Errorf("expected masked phone '+15****67', got '%v'", fields["from_number"])
	}
}

func TestBusinessEventLogger_CallCompleted(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	callID := uuid.New()
	bel.CallCompleted(context.Background(), callID, "bland", 5*time.Minute, "completed")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "call_completed" {
		t.Errorf("expected message 'call_completed', got '%s'", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["event_type"] != "call.completed" {
		t.Errorf("expected event_type 'call.completed', got '%v'", fields["event_type"])
	}
	if fields["status"] != "completed" {
		t.Errorf("expected status 'completed', got '%v'", fields["status"])
	}
}

func TestBusinessEventLogger_QuoteGenerated_Success(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	callID := uuid.New()
	estimatedValue := 5000.00
	bel.QuoteGenerated(context.Background(), callID, 10*time.Second, true, &estimatedValue)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "quote_generated" {
		t.Errorf("expected message 'quote_generated', got '%s'", entry.Message)
	}
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("expected INFO level, got %v", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["success"] != true {
		t.Errorf("expected success=true, got '%v'", fields["success"])
	}
	if fields["estimated_value"] != 5000.00 {
		t.Errorf("expected estimated_value=5000.00, got '%v'", fields["estimated_value"])
	}
}

func TestBusinessEventLogger_QuoteGenerated_Failure(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	callID := uuid.New()
	bel.QuoteGenerated(context.Background(), callID, 10*time.Second, false, nil)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "quote_generation_failed" {
		t.Errorf("expected message 'quote_generation_failed', got '%s'", entry.Message)
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("expected WARN level, got %v", entry.Level)
	}
}

func TestBusinessEventLogger_UserLogin(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		bel.UserLogin(context.Background(), userID, "user@example.com", "192.168.1.1", true)

		entries := logs.TakeAll()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		entry := entries[0]
		if entry.Message != "user_login" {
			t.Errorf("expected message 'user_login', got '%s'", entry.Message)
		}
		if entry.Level != zapcore.InfoLevel {
			t.Errorf("expected INFO level, got %v", entry.Level)
		}

		fields := entry.ContextMap()
		// Email should be masked
		if fields["email"] != "us***@example.com" {
			t.Errorf("expected masked email 'us***@example.com', got '%v'", fields["email"])
		}
	})

	t.Run("failure", func(t *testing.T) {
		bel.UserLogin(context.Background(), uuid.Nil, "baduser@example.com", "10.0.0.1", false)

		entries := logs.TakeAll()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		entry := entries[0]
		if entry.Message != "user_login_failed" {
			t.Errorf("expected message 'user_login_failed', got '%s'", entry.Message)
		}
		if entry.Level != zapcore.WarnLevel {
			t.Errorf("expected WARN level, got %v", entry.Level)
		}
	})
}

func TestBusinessEventLogger_WebhookReceived(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	t.Run("valid webhook", func(t *testing.T) {
		bel.WebhookReceived(context.Background(), "bland", "call.completed", "abc123", true)

		entries := logs.TakeAll()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		entry := entries[0]
		if entry.Message != "webhook_received" {
			t.Errorf("expected message 'webhook_received', got '%s'", entry.Message)
		}
		if entry.Level != zapcore.InfoLevel {
			t.Errorf("expected INFO level, got %v", entry.Level)
		}
	})

	t.Run("invalid webhook", func(t *testing.T) {
		bel.WebhookReceived(context.Background(), "bland", "call.completed", "abc123", false)

		entries := logs.TakeAll()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		entry := entries[0]
		if entry.Message != "webhook_invalid" {
			t.Errorf("expected message 'webhook_invalid', got '%s'", entry.Message)
		}
		if entry.Level != zapcore.WarnLevel {
			t.Errorf("expected WARN level, got %v", entry.Level)
		}
	})
}

func TestBusinessEventLogger_DailyStats(t *testing.T) {
	logger, logs := newTestLogger()
	bel := NewBusinessEventLogger(logger)

	stats := DailyStatsData{
		TotalCalls:      100,
		CompletedCalls:  85,
		QuotesGenerated: 75,
		AvgCallDuration: 5 * time.Minute,
		TotalQuoteValue: 250000.00,
	}

	bel.DailyStats(context.Background(), time.Now(), stats)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "daily_stats" {
		t.Errorf("expected message 'daily_stats', got '%s'", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["total_calls"] != int64(100) {
		t.Errorf("expected total_calls=100, got '%v'", fields["total_calls"])
	}
	if fields["completed_calls"] != int64(85) {
		t.Errorf("expected completed_calls=85, got '%v'", fields["completed_calls"])
	}
}

func TestMaskPhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+15551234567", "+15****67"},
		{"1234", "****"},
		{"123", "****"},
		{"12345678", "123****78"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskPhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("maskPhoneNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "us***@example.com"},
		{"ab@example.com", "a***@example.com"}, // 2 chars before @ shows first char only
		{"a@example.com", "a***@example.com"},
		{"noemail", "****"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskEmail(tt.input)
			if result != tt.expected {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.100", "19****00"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcdef", "ab****ef"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("maskIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
