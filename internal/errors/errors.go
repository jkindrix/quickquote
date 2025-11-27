// Package errors provides comprehensive error types for the QuickQuote application.
// It includes domain-specific errors, error classification, and HTTP status mapping.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Code represents an application error code.
type Code string

// Error codes for different error categories.
const (
	// Authentication/Authorization errors
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeSessionExpired     Code = "SESSION_EXPIRED"
	CodeCSRFInvalid        Code = "CSRF_INVALID"

	// Validation errors
	CodeValidation       Code = "VALIDATION_ERROR"
	CodeInvalidInput     Code = "INVALID_INPUT"
	CodeMissingField     Code = "MISSING_FIELD"
	CodeInvalidFormat    Code = "INVALID_FORMAT"
	CodeConstraintFailed Code = "CONSTRAINT_FAILED"

	// Resource errors
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeAlreadyExists Code = "ALREADY_EXISTS"

	// External service errors
	CodeExternalService   Code = "EXTERNAL_SERVICE_ERROR"
	CodeCircuitOpen       Code = "CIRCUIT_OPEN"
	CodeRateLimited       Code = "RATE_LIMITED"
	CodeTimeout           Code = "TIMEOUT"
	CodeWebhookInvalid    Code = "WEBHOOK_INVALID"
	CodeProviderError     Code = "PROVIDER_ERROR"

	// Internal errors
	CodeInternal   Code = "INTERNAL_ERROR"
	CodeDatabase   Code = "DATABASE_ERROR"
	CodeConfig     Code = "CONFIG_ERROR"

	// Quote/Call errors
	CodeQuoteGenerationFailed Code = "QUOTE_GENERATION_FAILED"
	CodeCallNotReady          Code = "CALL_NOT_READY"
	CodeTranscriptMissing     Code = "TRANSCRIPT_MISSING"
)

// Kind represents the kind of error for classification.
type Kind int

const (
	// KindUnknown is an unknown error kind.
	KindUnknown Kind = iota
	// KindUser indicates a user-caused error (bad input, unauthorized, etc.).
	KindUser
	// KindSystem indicates a system error (database down, external service failure).
	KindSystem
	// KindTransient indicates a temporary error that may succeed on retry.
	KindTransient
)

// Error is the base application error type.
type Error struct {
	// Code is the machine-readable error code.
	Code Code `json:"code"`
	// Message is the human-readable error message.
	Message string `json:"message"`
	// Kind classifies the error for handling decisions.
	Kind Kind `json:"-"`
	// Op is the operation being performed (e.g., "auth.Login").
	Op string `json:"-"`
	// Err is the underlying error, if any.
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Op != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
		}
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether target matches this error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// HTTPStatus returns the appropriate HTTP status code for this error.
func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeUnauthorized, CodeInvalidCredentials, CodeSessionExpired:
		return http.StatusUnauthorized
	case CodeForbidden, CodeCSRFInvalid:
		return http.StatusForbidden
	case CodeValidation, CodeInvalidInput, CodeMissingField, CodeInvalidFormat, CodeConstraintFailed:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeAlreadyExists:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeExternalService, CodeCircuitOpen, CodeProviderError, CodeWebhookInvalid:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// IsRetriable returns true if the error may succeed on retry.
func (e *Error) IsRetriable() bool {
	return e.Kind == KindTransient
}

// IsUserError returns true if the error was caused by user action.
func (e *Error) IsUserError() bool {
	return e.Kind == KindUser
}

// ErrorResponse represents the JSON response for API errors.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error details in API responses.
type ErrorDetail struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

// ToResponse converts an Error to an API response.
func (e *Error) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    e.Code,
			Message: e.Message,
		},
	}
}

// Constructor functions for common errors

// New creates a new Error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Kind:    kindForCode(code),
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, op string, code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Kind:    kindForCode(code),
		Op:      op,
		Err:     err,
	}
}

// WrapWithOp wraps an existing error preserving its code but adding operation context.
func WrapWithOp(err error, op string) *Error {
	var e *Error
	if errors.As(err, &e) {
		return &Error{
			Code:    e.Code,
			Message: e.Message,
			Kind:    e.Kind,
			Op:      op,
			Err:     e.Err,
		}
	}
	return &Error{
		Code:    CodeInternal,
		Message: err.Error(),
		Kind:    KindSystem,
		Op:      op,
		Err:     err,
	}
}

// kindForCode returns the default Kind for a given Code.
func kindForCode(code Code) Kind {
	switch code {
	case CodeUnauthorized, CodeForbidden, CodeInvalidCredentials, CodeSessionExpired, CodeCSRFInvalid:
		return KindUser
	case CodeValidation, CodeInvalidInput, CodeMissingField, CodeInvalidFormat, CodeConstraintFailed:
		return KindUser
	case CodeNotFound, CodeConflict, CodeAlreadyExists:
		return KindUser
	case CodeRateLimited, CodeTimeout, CodeCircuitOpen:
		return KindTransient
	case CodeExternalService, CodeProviderError:
		return KindTransient
	default:
		return KindSystem
	}
}

// Sentinel errors for common cases

var (
	// ErrNotFound indicates a requested resource was not found.
	ErrNotFound = New(CodeNotFound, "resource not found")

	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized = New(CodeUnauthorized, "authentication required")

	// ErrForbidden indicates the user lacks permission.
	ErrForbidden = New(CodeForbidden, "access denied")

	// ErrInvalidCredentials indicates wrong username/password.
	ErrInvalidCredentials = New(CodeInvalidCredentials, "invalid email or password")

	// ErrSessionExpired indicates the session has expired.
	ErrSessionExpired = New(CodeSessionExpired, "session has expired")

	// ErrCSRFInvalid indicates CSRF token validation failed.
	ErrCSRFInvalid = New(CodeCSRFInvalid, "invalid CSRF token")

	// ErrRateLimited indicates too many requests.
	ErrRateLimited = New(CodeRateLimited, "rate limit exceeded")

	// ErrCircuitOpen indicates the circuit breaker is open.
	ErrCircuitOpen = New(CodeCircuitOpen, "service temporarily unavailable")

	// ErrTimeout indicates an operation timed out.
	ErrTimeout = New(CodeTimeout, "operation timed out")
)

// Specialized error constructors

// NotFound creates a not found error for a specific resource.
func NotFound(resource string) *Error {
	return &Error{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Kind:    KindUser,
	}
}

// ValidationFailed creates a validation error with details.
func ValidationFailed(message string) *Error {
	return &Error{
		Code:    CodeValidation,
		Message: message,
		Kind:    KindUser,
	}
}

// MissingField creates a missing field validation error.
func MissingField(field string) *Error {
	return &Error{
		Code:    CodeMissingField,
		Message: fmt.Sprintf("missing required field: %s", field),
		Kind:    KindUser,
	}
}

// InvalidFormat creates an invalid format validation error.
func InvalidFormat(field, expected string) *Error {
	return &Error{
		Code:    CodeInvalidFormat,
		Message: fmt.Sprintf("invalid format for %s: expected %s", field, expected),
		Kind:    KindUser,
	}
}

// DatabaseError creates a database error with the underlying cause.
func DatabaseError(op string, err error) *Error {
	return &Error{
		Code:    CodeDatabase,
		Message: "database operation failed",
		Kind:    KindSystem,
		Op:      op,
		Err:     err,
	}
}

// ExternalServiceError creates an external service error.
func ExternalServiceError(service string, err error) *Error {
	return &Error{
		Code:    CodeExternalService,
		Message: fmt.Sprintf("%s service error", service),
		Kind:    KindTransient,
		Err:     err,
	}
}

// ProviderError creates a voice provider error.
func ProviderError(provider string, err error) *Error {
	return &Error{
		Code:    CodeProviderError,
		Message: fmt.Sprintf("voice provider %s error", provider),
		Kind:    KindTransient,
		Err:     err,
	}
}

// WebhookError creates a webhook validation error.
func WebhookError(message string) *Error {
	return &Error{
		Code:    CodeWebhookInvalid,
		Message: message,
		Kind:    KindUser,
	}
}

// QuoteGenerationError creates a quote generation error.
func QuoteGenerationError(err error) *Error {
	return &Error{
		Code:    CodeQuoteGenerationFailed,
		Message: "failed to generate quote",
		Kind:    KindTransient,
		Err:     err,
	}
}

// InternalError creates a generic internal error.
func InternalError(message string, err error) *Error {
	return &Error{
		Code:    CodeInternal,
		Message: message,
		Kind:    KindSystem,
		Err:     err,
	}
}

// Helper functions

// GetCode extracts the error code from an error, returning CodeInternal for non-app errors.
func GetCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}

// GetHTTPStatus extracts the HTTP status from an error, returning 500 for non-app errors.
func GetHTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.HTTPStatus()
	}
	return http.StatusInternalServerError
}

// IsRetriable checks if an error is retriable.
func IsRetriable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.IsRetriable()
	}
	return false
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == CodeNotFound
	}
	return false
}

// IsUserError checks if an error was caused by user action.
func IsUserError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.IsUserError()
	}
	return false
}
