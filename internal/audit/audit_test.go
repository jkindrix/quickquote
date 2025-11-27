package audit

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// getFieldMap extracts field values from a log entry into a map.
// Handles different zap field types (String, Int64, etc.)
func getFieldMap(fields []zapcore.Field) map[string]interface{} {
	result := make(map[string]interface{})
	for _, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			result[f.Key] = f.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			result[f.Key] = f.Integer
		case zapcore.TimeType:
			result[f.Key] = time.Unix(0, f.Integer).In(f.Interface.(*time.Location))
		case zapcore.ByteStringType:
			result[f.Key] = string(f.Interface.([]byte))
		default:
			result[f.Key] = f.Interface
		}
	}
	return result
}

func TestNewLogger(t *testing.T) {
	baseLogger := zap.NewNop()
	auditLogger := NewLogger(baseLogger)

	if auditLogger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if auditLogger.logger == nil {
		t.Fatal("audit logger has nil internal logger")
	}
}

func TestLogger_Log(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	ctx := context.Background()
	event := &Event{
		Type:       EventLoginSuccess,
		Severity:   SeverityInfo,
		ActorID:    "user-123",
		ActorType:  "user",
		ActorName:  "test@example.com",
		SourceIP:   "192.168.1.1",
		UserAgent:  "TestBrowser/1.0",
		RequestID:  "req-456",
		Action:     "user login",
		Outcome:    "success",
	}

	auditLogger.Log(ctx, event)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]

	// Verify message
	if entry.Message != "security audit event" {
		t.Errorf("unexpected message: %s", entry.Message)
	}

	// Verify fields
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "auth.login.success" {
		t.Errorf("event_type = %v, expected auth.login.success", fieldMap["event_type"])
	}
	if fieldMap["actor_id"] != "user-123" {
		t.Errorf("actor_id = %v, expected user-123", fieldMap["actor_id"])
	}
	if fieldMap["outcome"] != "success" {
		t.Errorf("outcome = %v, expected success", fieldMap["outcome"])
	}
}

func TestLogger_Log_SetsDefaults(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	ctx := context.Background()
	event := &Event{
		Type:     EventLoginSuccess,
		Severity: SeverityInfo,
		Action:   "test",
		Outcome:  "success",
	}

	auditLogger.Log(ctx, event)

	// Event should have ID and timestamp set
	if event.ID == "" {
		t.Error("event ID should be set automatically")
	}
	if event.Timestamp.IsZero() {
		t.Error("event timestamp should be set automatically")
	}

	if logs.Len() != 1 {
		t.Fatal("expected 1 log entry")
	}
}

func TestLogger_Log_SeverityLevels(t *testing.T) {
	tests := []struct {
		severity      Severity
		expectedLevel string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warn"},
		{SeverityError, "error"},
		{SeverityCritical, "error"}, // Critical maps to error level
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			baseLogger := zap.New(core)
			auditLogger := NewLogger(baseLogger)

			auditLogger.Log(context.Background(), &Event{
				Type:     EventLoginSuccess,
				Severity: tt.severity,
				Action:   "test",
				Outcome:  "success",
			})

			if logs.Len() != 1 {
				t.Fatalf("expected 1 log entry, got %d", logs.Len())
			}

			entry := logs.All()[0]
			if entry.Level.String() != tt.expectedLevel {
				t.Errorf("level = %s, expected %s", entry.Level.String(), tt.expectedLevel)
			}
		})
	}
}

func TestLogger_LoginSuccess(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.LoginSuccess(
		context.Background(),
		"user-123",
		"John Doe",
		"john@example.com",
		"192.168.1.1",
		"TestBrowser/1.0",
		"req-456",
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "auth.login.success" {
		t.Errorf("event_type = %v, expected auth.login.success", fieldMap["event_type"])
	}
	if fieldMap["actor_id"] != "user-123" {
		t.Errorf("actor_id = %v, expected user-123", fieldMap["actor_id"])
	}
	if fieldMap["outcome"] != "success" {
		t.Errorf("outcome = %v, expected success", fieldMap["outcome"])
	}
}

func TestLogger_LoginFailure(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.LoginFailure(
		context.Background(),
		"john@example.com",
		"192.168.1.1",
		"TestBrowser/1.0",
		"req-456",
		"invalid password",
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Level != zap.WarnLevel {
		t.Errorf("level = %s, expected warn", entry.Level.String())
	}

	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "auth.login.failure" {
		t.Errorf("event_type = %v, expected auth.login.failure", fieldMap["event_type"])
	}
	if fieldMap["outcome"] != "failure" {
		t.Errorf("outcome = %v, expected failure", fieldMap["outcome"])
	}
	if fieldMap["reason"] != "invalid password" {
		t.Errorf("reason = %v, expected 'invalid password'", fieldMap["reason"])
	}
}

func TestLogger_CSRFViolation(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.CSRFViolation(
		context.Background(),
		"192.168.1.1",
		"TestBrowser/1.0",
		"req-456",
		"/api/sensitive",
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "authz.csrf.violation" {
		t.Errorf("event_type = %v, expected authz.csrf.violation", fieldMap["event_type"])
	}
}

func TestLogger_RateLimitExceeded(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.RateLimitExceeded(
		context.Background(),
		"192.168.1.1",
		"192.168.1.1",
		"req-456",
		"login",
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "authz.ratelimit.exceeded" {
		t.Errorf("event_type = %v, expected authz.ratelimit.exceeded", fieldMap["event_type"])
	}
}

func TestLogger_WebhookReceived(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.WebhookReceived(
		context.Background(),
		"bland",
		"call-123",
		"10.0.0.1",
		"req-456",
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "webhook.received" {
		t.Errorf("event_type = %v, expected webhook.received", fieldMap["event_type"])
	}
	if fieldMap["resource_type"] != "call" {
		t.Errorf("resource_type = %v, expected call", fieldMap["resource_type"])
	}
}

func TestLogger_QuoteGenerated(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	auditLogger.QuoteGenerated(
		context.Background(),
		"call-123",
		"req-456",
		150,
	)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	fieldMap := getFieldMap(entry.Context)

	if fieldMap["event_type"] != "quote.generated" {
		t.Errorf("event_type = %v, expected quote.generated", fieldMap["event_type"])
	}
}

func TestLogger_ServiceLifecycle(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	ctx := context.Background()
	auditLogger.ServiceStarted(ctx, "1.0.0", "development")
	auditLogger.ServiceStopping(ctx, "SIGTERM received")

	if logs.Len() != 2 {
		t.Fatalf("expected 2 log entries, got %d", logs.Len())
	}

	// Check first entry (started)
	startEntry := logs.All()[0]
	startFields := getFieldMap(startEntry.Context)
	if startFields["event_type"] != "system.started" {
		t.Errorf("event_type = %v, expected system.started", startFields["event_type"])
	}

	// Check second entry (stopping)
	stopEntry := logs.All()[1]
	stopFields := getFieldMap(stopEntry.Context)
	if stopFields["event_type"] != "system.stopping" {
		t.Errorf("event_type = %v, expected system.stopping", stopFields["event_type"])
	}
}

func TestEvent_Timestamp(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	auditLogger := NewLogger(baseLogger)

	// Test that timestamp is set automatically
	before := time.Now().UTC()
	event := &Event{
		Type:     EventLoginSuccess,
		Severity: SeverityInfo,
		Action:   "test",
		Outcome:  "success",
	}
	auditLogger.Log(context.Background(), event)
	after := time.Now().UTC()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("timestamp %v should be between %v and %v", event.Timestamp, before, after)
	}

	// Test that pre-set timestamp is preserved
	customTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	event2 := &Event{
		Type:      EventLoginSuccess,
		Severity:  SeverityInfo,
		Timestamp: customTime,
		Action:    "test",
		Outcome:   "success",
	}
	auditLogger.Log(context.Background(), event2)

	if !event2.Timestamp.Equal(customTime) {
		t.Errorf("custom timestamp should be preserved: got %v, expected %v", event2.Timestamp, customTime)
	}
}
