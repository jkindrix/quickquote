// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/middleware"
)

// Context key for user
type contextKey string

const (
	userContextKey      contextKey = "user"
	requestIDContextKey contextKey = "request_id"
)

// GetUserFromContext retrieves the authenticated user from the context.
func GetUserFromContext(ctx context.Context) *domain.User {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

// GetRequestIDFromContext retrieves the request ID from the context.
func GetRequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(requestIDContextKey).(string)
	if !ok {
		return ""
	}
	return id
}

// BaseHandler provides shared functionality for all handlers.
type BaseHandler struct {
	templateEngine *TemplateEngine
	csrfProtection *middleware.CSRFProtection
	logger         *zap.Logger
}

// BaseHandlerConfig holds configuration for BaseHandler.
type BaseHandlerConfig struct {
	TemplateEngine *TemplateEngine
	CSRFProtection *middleware.CSRFProtection
	Logger         *zap.Logger
}

// NewBaseHandler creates a new BaseHandler with all required dependencies.
func NewBaseHandler(cfg BaseHandlerConfig) *BaseHandler {
	if cfg.Logger == nil {
		panic("logger is required")
	}
	return &BaseHandler{
		templateEngine: cfg.TemplateEngine,
		csrfProtection: cfg.CSRFProtection,
		logger:         cfg.Logger,
	}
}

// Logger returns the handler's logger.
func (b *BaseHandler) Logger() *zap.Logger {
	return b.logger
}

// RenderTemplate renders an HTML template with the given data.
func (b *BaseHandler) RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Add CSRF token to all templates if not already present
	if _, ok := data["CSRFToken"]; !ok {
		data["CSRFToken"] = b.GetCSRFToken(r)
	}

	// Add request ID for debugging
	if reqID := GetRequestIDFromContext(r.Context()); reqID != "" {
		data["RequestID"] = reqID
	}

	if b.templateEngine != nil && b.templateEngine.HasTemplate(name) {
		if err := b.templateEngine.Render(w, name, data); err != nil {
			b.logger.Error("failed to render template", zap.String("name", name), zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// No fallback templates in the new architecture
	b.logger.Error("template not found", zap.String("name", name))
	http.Error(w, "Template not found", http.StatusInternalServerError)
}

// GetCSRFToken returns the CSRF token for the current request.
func (b *BaseHandler) GetCSRFToken(r *http.Request) string {
	if b.csrfProtection != nil {
		return b.csrfProtection.GetToken(r)
	}
	return ""
}

// Render renders a template with typed DTO data.
// This is the preferred method over RenderTemplate for type safety.
func (b *BaseHandler) Render(w http.ResponseWriter, r *http.Request, name string, data TemplateData) {
	b.RenderTemplate(w, r, name, data.ToMap())
}

// WriteJSON writes a JSON response with the appropriate headers.
func (b *BaseHandler) WriteJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// Add X-Request-ID header if available
	if reqID := GetRequestIDFromContext(r.Context()); reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}

	w.WriteHeader(status)
	if data != nil {
		if err := writeJSON(w, data); err != nil {
			b.logger.Debug("failed to write JSON response", zap.Error(err))
		}
	}
}

// WriteError writes an error response in JSON format.
func (b *BaseHandler) WriteError(w http.ResponseWriter, r *http.Request, status int, message string) {
	b.WriteJSON(w, r, status, map[string]interface{}{
		"error":      message,
		"status":     status,
		"request_id": GetRequestIDFromContext(r.Context()),
	})
}

// helper to write JSON
func encodeJSON(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}
