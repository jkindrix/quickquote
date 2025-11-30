// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
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
		if err := encodeJSON(w, data); err != nil {
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

// JSON writes a JSON response with the appropriate headers.
// This is a package-level helper for handlers that don't embed BaseHandler.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// JSONWithRequest writes a JSON response, including request ID in headers.
// This is the preferred method when the request is available.
func JSONWithRequest(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if reqID := GetRequestIDFromContext(r.Context()); reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// APIError writes an API error response in a consistent format.
// This is a package-level helper for handlers that don't embed BaseHandler.
func APIError(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// APIErrorWithRequest writes an API error response, including request context.
// This is the preferred method when the request is available.
func APIErrorWithRequest(w http.ResponseWriter, r *http.Request, status int, message string) {
	JSONWithRequest(w, r, status, map[string]interface{}{
		"error":      http.StatusText(status),
		"message":    message,
		"status":     status,
		"request_id": GetRequestIDFromContext(r.Context()),
	})
}

// ValidationFieldError represents a single field validation error.
type ValidationFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationErrorResponse represents a structured validation error response.
type ValidationErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Status  int                    `json:"status"`
	Errors  []ValidationFieldError `json:"errors"`
}

// APIValidationError writes a validation error response with field-level details.
func APIValidationError(w http.ResponseWriter, errors []ValidationFieldError) {
	resp := ValidationErrorResponse{
		Error:   "Bad Request",
		Message: "Validation failed",
		Status:  http.StatusBadRequest,
		Errors:  errors,
	}
	JSON(w, http.StatusBadRequest, resp)
}

// APIValidationErrorWithRequest writes a validation error response with request context.
func APIValidationErrorWithRequest(w http.ResponseWriter, r *http.Request, errors []ValidationFieldError) {
	resp := map[string]interface{}{
		"error":      "Bad Request",
		"message":    "Validation failed",
		"status":     http.StatusBadRequest,
		"errors":     errors,
		"request_id": GetRequestIDFromContext(r.Context()),
	}
	JSONWithRequest(w, r, http.StatusBadRequest, resp)
}

// NewValidationError creates a single field validation error.
func NewValidationError(field, message, code string) ValidationFieldError {
	return ValidationFieldError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

// RequiredFieldError creates a validation error for a required field.
func RequiredFieldError(field string) ValidationFieldError {
	return ValidationFieldError{
		Field:   field,
		Message: "is required",
		Code:    "required",
	}
}

// InvalidFormatError creates a validation error for invalid format.
func InvalidFormatError(field, expectedFormat string) ValidationFieldError {
	return ValidationFieldError{
		Field:   field,
		Message: "must be " + expectedFormat,
		Code:    "invalid_format",
	}
}

// TooLongError creates a validation error for exceeding maximum length.
func TooLongError(field string, maxLen int) ValidationFieldError {
	return ValidationFieldError{
		Field:   field,
		Message: fmt.Sprintf("must be at most %d characters", maxLen),
		Code:    "too_long",
	}
}

// InvalidValueError creates a validation error for an invalid value.
func InvalidValueError(field, reason string) ValidationFieldError {
	return ValidationFieldError{
		Field:   field,
		Message: reason,
		Code:    "invalid_value",
	}
}
