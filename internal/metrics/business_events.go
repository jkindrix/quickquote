// Package metrics provides metrics collection including business event logging.
package metrics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BusinessEventLogger provides structured logging for business events.
// This complements Prometheus metrics by providing detailed, searchable logs
// for business intelligence, debugging, and compliance.
type BusinessEventLogger struct {
	logger *zap.Logger
}

// NewBusinessEventLogger creates a new business event logger.
func NewBusinessEventLogger(logger *zap.Logger) *BusinessEventLogger {
	return &BusinessEventLogger{
		logger: logger.Named("business_events"),
	}
}

// CallReceived logs when an inbound call is received.
func (l *BusinessEventLogger) CallReceived(ctx context.Context, callID uuid.UUID, provider, fromNumber string) {
	l.logger.Info("call_received",
		zap.String("event_type", "call.received"),
		zap.String("call_id", callID.String()),
		zap.String("provider", provider),
		zap.String("from_number", maskPhoneNumber(fromNumber)),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// CallCompleted logs when a call is completed.
func (l *BusinessEventLogger) CallCompleted(ctx context.Context, callID uuid.UUID, provider string, duration time.Duration, status string) {
	l.logger.Info("call_completed",
		zap.String("event_type", "call.completed"),
		zap.String("call_id", callID.String()),
		zap.String("provider", provider),
		zap.Duration("duration", duration),
		zap.String("status", status),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// QuoteGenerated logs when a quote is generated.
func (l *BusinessEventLogger) QuoteGenerated(ctx context.Context, callID uuid.UUID, duration time.Duration, success bool, estimatedValue *float64) {
	fields := []zap.Field{
		zap.String("event_type", "quote.generated"),
		zap.String("call_id", callID.String()),
		zap.Duration("generation_duration", duration),
		zap.Bool("success", success),
		zap.Time("timestamp", time.Now().UTC()),
	}
	if estimatedValue != nil {
		fields = append(fields, zap.Float64("estimated_value", *estimatedValue))
	}

	if success {
		l.logger.Info("quote_generated", fields...)
	} else {
		l.logger.Warn("quote_generation_failed", fields...)
	}
}

// QuoteJobQueued logs when a quote job is added to the processing queue.
func (l *BusinessEventLogger) QuoteJobQueued(ctx context.Context, jobID, callID uuid.UUID, scheduledAt time.Time) {
	l.logger.Info("quote_job_queued",
		zap.String("event_type", "quote_job.queued"),
		zap.String("job_id", jobID.String()),
		zap.String("call_id", callID.String()),
		zap.Time("scheduled_at", scheduledAt),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// QuoteJobProcessed logs when a quote job is processed.
func (l *BusinessEventLogger) QuoteJobProcessed(ctx context.Context, jobID, callID uuid.UUID, status string, attempts int, duration time.Duration) {
	l.logger.Info("quote_job_processed",
		zap.String("event_type", "quote_job.processed"),
		zap.String("job_id", jobID.String()),
		zap.String("call_id", callID.String()),
		zap.String("status", status),
		zap.Int("attempts", attempts),
		zap.Duration("processing_duration", duration),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// UserLogin logs a user login event.
func (l *BusinessEventLogger) UserLogin(ctx context.Context, userID uuid.UUID, email, ip string, success bool) {
	if success {
		l.logger.Info("user_login",
			zap.String("event_type", "user.login"),
			zap.String("user_id", userID.String()),
			zap.String("email", maskEmail(email)),
			zap.String("ip", ip),
			zap.Bool("success", true),
			zap.Time("timestamp", time.Now().UTC()),
		)
	} else {
		l.logger.Warn("user_login_failed",
			zap.String("event_type", "user.login_failed"),
			zap.String("email", maskEmail(email)),
			zap.String("ip", ip),
			zap.Bool("success", false),
			zap.Time("timestamp", time.Now().UTC()),
		)
	}
}

// UserLogout logs a user logout event.
func (l *BusinessEventLogger) UserLogout(ctx context.Context, userID uuid.UUID, email string) {
	l.logger.Info("user_logout",
		zap.String("event_type", "user.logout"),
		zap.String("user_id", userID.String()),
		zap.String("email", maskEmail(email)),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// WebhookReceived logs when a webhook is received from a voice provider.
func (l *BusinessEventLogger) WebhookReceived(ctx context.Context, provider, eventType string, callID string, valid bool) {
	level := l.logger.Info
	eventName := "webhook_received"
	if !valid {
		level = l.logger.Warn
		eventName = "webhook_invalid"
	}
	level(eventName,
		zap.String("event_type", "webhook.received"),
		zap.String("provider", provider),
		zap.String("webhook_event_type", eventType),
		zap.String("call_id", callID),
		zap.Bool("valid", valid),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// SettingsUpdated logs when settings are updated.
func (l *BusinessEventLogger) SettingsUpdated(ctx context.Context, userID uuid.UUID, settingType string, changes map[string]interface{}) {
	l.logger.Info("settings_updated",
		zap.String("event_type", "settings.updated"),
		zap.String("user_id", userID.String()),
		zap.String("setting_type", settingType),
		zap.Int("changes_count", len(changes)),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// PromptCreated logs when a prompt preset is created.
func (l *BusinessEventLogger) PromptCreated(ctx context.Context, userID, promptID uuid.UUID, promptName string) {
	l.logger.Info("prompt_created",
		zap.String("event_type", "prompt.created"),
		zap.String("user_id", userID.String()),
		zap.String("prompt_id", promptID.String()),
		zap.String("prompt_name", promptName),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// PromptApplied logs when a prompt is applied to inbound calls.
func (l *BusinessEventLogger) PromptApplied(ctx context.Context, userID, promptID uuid.UUID, promptName string) {
	l.logger.Info("prompt_applied",
		zap.String("event_type", "prompt.applied"),
		zap.String("user_id", userID.String()),
		zap.String("prompt_id", promptID.String()),
		zap.String("prompt_name", promptName),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// APIError logs an API error for monitoring.
func (l *BusinessEventLogger) APIError(ctx context.Context, endpoint, method string, statusCode int, errorMsg string) {
	l.logger.Error("api_error",
		zap.String("event_type", "api.error"),
		zap.String("endpoint", endpoint),
		zap.String("method", method),
		zap.Int("status_code", statusCode),
		zap.String("error", errorMsg),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// ExternalAPICall logs calls to external APIs (Claude, Bland, etc.).
func (l *BusinessEventLogger) ExternalAPICall(ctx context.Context, service, endpoint string, duration time.Duration, success bool, statusCode int) {
	level := l.logger.Info
	eventName := "external_api_call"
	if !success {
		level = l.logger.Warn
		eventName = "external_api_call_failed"
	}
	level(eventName,
		zap.String("event_type", "external_api.call"),
		zap.String("service", service),
		zap.String("endpoint", endpoint),
		zap.Duration("duration", duration),
		zap.Bool("success", success),
		zap.Int("status_code", statusCode),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// RateLimitExceeded logs when a rate limit is exceeded.
func (l *BusinessEventLogger) RateLimitExceeded(ctx context.Context, limiterType string, identifier string) {
	l.logger.Warn("rate_limit_exceeded",
		zap.String("event_type", "rate_limit.exceeded"),
		zap.String("limiter_type", limiterType),
		zap.String("identifier", maskIdentifier(identifier)),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// DailyStats logs daily aggregate statistics.
func (l *BusinessEventLogger) DailyStats(ctx context.Context, date time.Time, stats DailyStatsData) {
	l.logger.Info("daily_stats",
		zap.String("event_type", "stats.daily"),
		zap.Time("date", date),
		zap.Int("total_calls", stats.TotalCalls),
		zap.Int("completed_calls", stats.CompletedCalls),
		zap.Int("quotes_generated", stats.QuotesGenerated),
		zap.Duration("avg_call_duration", stats.AvgCallDuration),
		zap.Float64("total_quote_value", stats.TotalQuoteValue),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

// DailyStatsData holds aggregate statistics for a day.
type DailyStatsData struct {
	TotalCalls      int
	CompletedCalls  int
	QuotesGenerated int
	AvgCallDuration time.Duration
	TotalQuoteValue float64
}

// Helper functions for data masking

// maskPhoneNumber masks a phone number for privacy.
func maskPhoneNumber(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-2:]
}

// maskEmail masks an email for privacy.
func maskEmail(email string) string {
	if len(email) == 0 {
		return ""
	}
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "****"
	}
	if at <= 2 {
		return email[0:1] + "***" + email[at:]
	}
	return email[0:2] + "***" + email[at:]
}

// maskIdentifier masks an identifier for privacy.
func maskIdentifier(id string) string {
	if len(id) <= 4 {
		return "****"
	}
	return id[:2] + "****" + id[len(id)-2:]
}
