// Package audit provides security event logging for compliance and forensics.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventType represents the type of audit event.
type EventType string

// Security audit event types.
const (
	// Authentication events
	EventLoginSuccess    EventType = "auth.login.success"
	EventLoginFailure    EventType = "auth.login.failure"
	EventLogout          EventType = "auth.logout"
	EventSessionExpired  EventType = "auth.session.expired"
	EventSessionRotated  EventType = "auth.session.rotated"
	EventPasswordChanged EventType = "auth.password.changed"

	// Authorization events
	EventAccessDenied     EventType = "authz.access.denied"
	EventCSRFViolation    EventType = "authz.csrf.violation"
	EventRateLimitExceeded EventType = "authz.ratelimit.exceeded"

	// Data access events
	EventDataAccess   EventType = "data.access"
	EventDataModified EventType = "data.modified"
	EventDataDeleted  EventType = "data.deleted"

	// Webhook events
	EventWebhookReceived       EventType = "webhook.received"
	EventWebhookValidationFail EventType = "webhook.validation.failed"

	// API events
	EventAPICallMade    EventType = "api.call.made"
	EventAPICallFailed  EventType = "api.call.failed"
	EventQuoteGenerated EventType = "quote.generated"

	// System events
	EventServiceStarted  EventType = "system.started"
	EventServiceStopping EventType = "system.stopping"
	EventConfigChanged   EventType = "system.config.changed"

	// Admin operations
	EventAdminPromptCreated  EventType = "admin.prompt.created"
	EventAdminPromptUpdated  EventType = "admin.prompt.updated"
	EventAdminPromptDeleted  EventType = "admin.prompt.deleted"
	EventAdminSettingChanged EventType = "admin.setting.changed"
	EventAdminCallInitiated  EventType = "admin.call.initiated"
	EventAdminCallEnded      EventType = "admin.call.ended"
	EventAdminCallAnalyzed   EventType = "admin.call.analyzed"
)

// Severity represents the severity level of an audit event.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents an audit log entry.
type Event struct {
	// ID is a unique identifier for this event.
	ID string `json:"id"`

	// Timestamp when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Type of event (e.g., "auth.login.success").
	Type EventType `json:"type"`

	// Severity level.
	Severity Severity `json:"severity"`

	// Actor identification (who performed the action).
	ActorID   string `json:"actor_id,omitempty"`   // User ID if authenticated
	ActorType string `json:"actor_type,omitempty"` // "user", "system", "webhook"
	ActorName string `json:"actor_name,omitempty"` // Human-readable name

	// Source of the event.
	SourceIP   string `json:"source_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	RequestID  string `json:"request_id,omitempty"`  // Correlation ID
	SessionID  string `json:"session_id,omitempty"`

	// Resource being accessed/modified.
	ResourceType string `json:"resource_type,omitempty"` // "call", "user", "session"
	ResourceID   string `json:"resource_id,omitempty"`

	// Action details.
	Action  string `json:"action"`          // Brief action description
	Outcome string `json:"outcome"`         // "success", "failure", "denied"
	Reason  string `json:"reason,omitempty"` // Failure/denial reason

	// Additional context.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Logger provides audit logging capabilities.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a new audit logger.
func NewLogger(baseLogger *zap.Logger) *Logger {
	return &Logger{
		logger: baseLogger.Named("audit"),
	}
}

// Log records an audit event.
func (l *Logger) Log(ctx context.Context, event *Event) {
	// Ensure ID and timestamp are set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Get severity-appropriate log level
	level := zap.InfoLevel
	switch event.Severity {
	case SeverityWarning:
		level = zap.WarnLevel
	case SeverityError:
		level = zap.ErrorLevel
	case SeverityCritical:
		level = zap.ErrorLevel // Critical also uses error level
	}

	// Convert metadata to JSON for logging
	var metadataJSON []byte
	if len(event.Metadata) > 0 {
		var err error
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			// If metadata can't be marshaled, log the error but continue
			metadataJSON = []byte(`{"error":"failed to marshal metadata"}`)
		}
	}

	// Log the event with structured fields
	fields := []zap.Field{
		zap.String("audit_id", event.ID),
		zap.Time("audit_timestamp", event.Timestamp),
		zap.String("event_type", string(event.Type)),
		zap.String("severity", string(event.Severity)),
		zap.String("action", event.Action),
		zap.String("outcome", event.Outcome),
	}

	// Add optional fields
	if event.ActorID != "" {
		fields = append(fields, zap.String("actor_id", event.ActorID))
	}
	if event.ActorType != "" {
		fields = append(fields, zap.String("actor_type", event.ActorType))
	}
	if event.ActorName != "" {
		fields = append(fields, zap.String("actor_name", event.ActorName))
	}
	if event.SourceIP != "" {
		fields = append(fields, zap.String("source_ip", event.SourceIP))
	}
	if event.UserAgent != "" {
		fields = append(fields, zap.String("user_agent", event.UserAgent))
	}
	if event.RequestID != "" {
		fields = append(fields, zap.String("request_id", event.RequestID))
	}
	if event.SessionID != "" {
		fields = append(fields, zap.String("session_id", event.SessionID))
	}
	if event.ResourceType != "" {
		fields = append(fields, zap.String("resource_type", event.ResourceType))
	}
	if event.ResourceID != "" {
		fields = append(fields, zap.String("resource_id", event.ResourceID))
	}
	if event.Reason != "" {
		fields = append(fields, zap.String("reason", event.Reason))
	}
	if len(metadataJSON) > 0 {
		fields = append(fields, zap.ByteString("metadata", metadataJSON))
	}

	// Log at appropriate level
	if ce := l.logger.Check(level, "security audit event"); ce != nil {
		ce.Write(fields...)
	}
}

// Helper methods for common audit scenarios

// LoginSuccess logs a successful login.
func (l *Logger) LoginSuccess(ctx context.Context, userID, userName, email, ip, userAgent, requestID string) {
	l.Log(ctx, &Event{
		Type:       EventLoginSuccess,
		Severity:   SeverityInfo,
		ActorID:    userID,
		ActorType:  "user",
		ActorName:  userName,
		SourceIP:   ip,
		UserAgent:  userAgent,
		RequestID:  requestID,
		Action:     "user login",
		Outcome:    "success",
		Metadata: map[string]interface{}{
			"email": email,
		},
	})
}

// LoginFailure logs a failed login attempt.
func (l *Logger) LoginFailure(ctx context.Context, email, ip, userAgent, requestID, reason string) {
	l.Log(ctx, &Event{
		Type:      EventLoginFailure,
		Severity:  SeverityWarning,
		ActorType: "user",
		ActorName: email,
		SourceIP:  ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Action:    "user login",
		Outcome:   "failure",
		Reason:    reason,
		Metadata: map[string]interface{}{
			"email": email,
		},
	})
}

// Logout logs a user logout.
func (l *Logger) Logout(ctx context.Context, userID, userName, sessionID, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:      EventLogout,
		Severity:  SeverityInfo,
		ActorID:   userID,
		ActorType: "user",
		ActorName: userName,
		SessionID: sessionID,
		SourceIP:  ip,
		RequestID: requestID,
		Action:    "user logout",
		Outcome:   "success",
	})
}

// SessionExpired logs a session expiration.
func (l *Logger) SessionExpired(ctx context.Context, userID, sessionID string) {
	l.Log(ctx, &Event{
		Type:      EventSessionExpired,
		Severity:  SeverityInfo,
		ActorID:   userID,
		ActorType: "system",
		SessionID: sessionID,
		Action:    "session expired",
		Outcome:   "success",
	})
}

// AccessDenied logs an access denial.
func (l *Logger) AccessDenied(ctx context.Context, userID, resource, action, ip, requestID, reason string) {
	l.Log(ctx, &Event{
		Type:         EventAccessDenied,
		Severity:     SeverityWarning,
		ActorID:      userID,
		ActorType:    "user",
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: resource,
		Action:       action,
		Outcome:      "denied",
		Reason:       reason,
	})
}

// CSRFViolation logs a CSRF validation failure.
func (l *Logger) CSRFViolation(ctx context.Context, ip, userAgent, requestID, path string) {
	l.Log(ctx, &Event{
		Type:      EventCSRFViolation,
		Severity:  SeverityWarning,
		ActorType: "unknown",
		SourceIP:  ip,
		UserAgent: userAgent,
		RequestID: requestID,
		Action:    "CSRF validation",
		Outcome:   "failure",
		Reason:    "invalid or missing CSRF token",
		Metadata: map[string]interface{}{
			"path": path,
		},
	})
}

// RateLimitExceeded logs a rate limit violation.
func (l *Logger) RateLimitExceeded(ctx context.Context, identifier, ip, requestID, limiterType string) {
	l.Log(ctx, &Event{
		Type:      EventRateLimitExceeded,
		Severity:  SeverityWarning,
		ActorID:   identifier,
		ActorType: "client",
		SourceIP:  ip,
		RequestID: requestID,
		Action:    "request rate limited",
		Outcome:   "denied",
		Reason:    "rate limit exceeded",
		Metadata: map[string]interface{}{
			"limiter_type": limiterType,
		},
	})
}

// WebhookReceived logs an incoming webhook.
func (l *Logger) WebhookReceived(ctx context.Context, provider, callID, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventWebhookReceived,
		Severity:     SeverityInfo,
		ActorType:    "webhook",
		ActorName:    provider,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "call",
		ResourceID:   callID,
		Action:       "webhook received",
		Outcome:      "success",
	})
}

// WebhookValidationFailed logs a webhook validation failure.
func (l *Logger) WebhookValidationFailed(ctx context.Context, provider, ip, requestID, reason string) {
	l.Log(ctx, &Event{
		Type:      EventWebhookValidationFail,
		Severity:  SeverityWarning,
		ActorType: "webhook",
		ActorName: provider,
		SourceIP:  ip,
		RequestID: requestID,
		Action:    "webhook validation",
		Outcome:   "failure",
		Reason:    reason,
	})
}

// QuoteGenerated logs successful quote generation.
func (l *Logger) QuoteGenerated(ctx context.Context, callID, requestID string, durationMs int64) {
	l.Log(ctx, &Event{
		Type:         EventQuoteGenerated,
		Severity:     SeverityInfo,
		ActorType:    "system",
		RequestID:    requestID,
		ResourceType: "call",
		ResourceID:   callID,
		Action:       "quote generated",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"duration_ms": durationMs,
		},
	})
}

// APICallFailed logs a failed external API call.
func (l *Logger) APICallFailed(ctx context.Context, service, operation, requestID, reason string) {
	l.Log(ctx, &Event{
		Type:      EventAPICallFailed,
		Severity:  SeverityError,
		ActorType: "system",
		RequestID: requestID,
		Action:    "external API call",
		Outcome:   "failure",
		Reason:    reason,
		Metadata: map[string]interface{}{
			"service":   service,
			"operation": operation,
		},
	})
}

// ServiceStarted logs service startup.
func (l *Logger) ServiceStarted(ctx context.Context, version, environment string) {
	l.Log(ctx, &Event{
		Type:      EventServiceStarted,
		Severity:  SeverityInfo,
		ActorType: "system",
		Action:    "service started",
		Outcome:   "success",
		Metadata: map[string]interface{}{
			"version":     version,
			"environment": environment,
		},
	})
}

// ServiceStopping logs service shutdown initiation.
func (l *Logger) ServiceStopping(ctx context.Context, reason string) {
	l.Log(ctx, &Event{
		Type:      EventServiceStopping,
		Severity:  SeverityInfo,
		ActorType: "system",
		Action:    "service stopping",
		Outcome:   "success",
		Reason:    reason,
	})
}

// Admin operation helpers

// PromptCreated logs a prompt creation by an admin.
func (l *Logger) PromptCreated(ctx context.Context, userID, userName, promptID, promptName, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventAdminPromptCreated,
		Severity:     SeverityInfo,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "prompt",
		ResourceID:   promptID,
		Action:       "prompt created",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"prompt_name": promptName,
		},
	})
}

// PromptUpdated logs a prompt update by an admin.
func (l *Logger) PromptUpdated(ctx context.Context, userID, userName, promptID, promptName, ip, requestID string, changes map[string]interface{}) {
	l.Log(ctx, &Event{
		Type:         EventAdminPromptUpdated,
		Severity:     SeverityInfo,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "prompt",
		ResourceID:   promptID,
		Action:       "prompt updated",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"prompt_name": promptName,
			"changes":     changes,
		},
	})
}

// PromptDeleted logs a prompt deletion by an admin.
func (l *Logger) PromptDeleted(ctx context.Context, userID, userName, promptID, promptName, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventAdminPromptDeleted,
		Severity:     SeverityWarning,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "prompt",
		ResourceID:   promptID,
		Action:       "prompt deleted",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"prompt_name": promptName,
		},
	})
}

// SettingChanged logs a setting change by an admin.
func (l *Logger) SettingChanged(ctx context.Context, userID, userName, settingKey, ip, requestID string, oldValue, newValue interface{}) {
	l.Log(ctx, &Event{
		Type:         EventAdminSettingChanged,
		Severity:     SeverityWarning,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "setting",
		ResourceID:   settingKey,
		Action:       "setting changed",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"key":       settingKey,
			"old_value": oldValue,
			"new_value": newValue,
		},
	})
}

// CallInitiated logs an outbound call initiation by an admin.
func (l *Logger) CallInitiated(ctx context.Context, userID, userName, callID, phoneNumber, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventAdminCallInitiated,
		Severity:     SeverityInfo,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "call",
		ResourceID:   callID,
		Action:       "call initiated",
		Outcome:      "success",
		Metadata: map[string]interface{}{
			"phone_number": phoneNumber,
		},
	})
}

// CallEnded logs a call termination by an admin.
func (l *Logger) CallEnded(ctx context.Context, userID, userName, callID, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventAdminCallEnded,
		Severity:     SeverityInfo,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "call",
		ResourceID:   callID,
		Action:       "call ended",
		Outcome:      "success",
	})
}

// CallAnalyzed logs a call analysis request by an admin.
func (l *Logger) CallAnalyzed(ctx context.Context, userID, userName, callID, ip, requestID string) {
	l.Log(ctx, &Event{
		Type:         EventAdminCallAnalyzed,
		Severity:     SeverityInfo,
		ActorID:      userID,
		ActorType:    "admin",
		ActorName:    userName,
		SourceIP:     ip,
		RequestID:    requestID,
		ResourceType: "call",
		ResourceID:   callID,
		Action:       "call analyzed",
		Outcome:      "success",
	})
}
