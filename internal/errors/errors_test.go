package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "simple message",
			err:      New(CodeNotFound, "user not found"),
			expected: "user not found",
		},
		{
			name: "with operation",
			err: &Error{
				Code:    CodeNotFound,
				Message: "user not found",
				Op:      "users.GetByID",
			},
			expected: "users.GetByID: user not found",
		},
		{
			name: "with underlying error",
			err: &Error{
				Code:    CodeDatabase,
				Message: "query failed",
				Err:     errors.New("connection refused"),
			},
			expected: "query failed: connection refused",
		},
		{
			name: "with operation and underlying error",
			err: &Error{
				Code:    CodeDatabase,
				Message: "query failed",
				Op:      "users.Create",
				Err:     errors.New("connection refused"),
			},
			expected: "users.Create: query failed: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("root cause")
	err := Wrap(underlying, "op", CodeInternal, "wrapped")

	if !errors.Is(err, underlying) {
		t.Error("Unwrap should allow errors.Is to find underlying error")
	}
}

func TestError_Is(t *testing.T) {
	err1 := New(CodeNotFound, "resource not found")
	err2 := New(CodeNotFound, "different message")
	err3 := New(CodeUnauthorized, "not authorized")

	if !errors.Is(err1, err2) {
		t.Error("errors with same code should match")
	}
	if errors.Is(err1, err3) {
		t.Error("errors with different codes should not match")
	}
}

func TestError_HTTPStatus(t *testing.T) {
	tests := []struct {
		code     Code
		expected int
	}{
		{CodeUnauthorized, http.StatusUnauthorized},
		{CodeInvalidCredentials, http.StatusUnauthorized},
		{CodeSessionExpired, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeCSRFInvalid, http.StatusForbidden},
		{CodeValidation, http.StatusBadRequest},
		{CodeInvalidInput, http.StatusBadRequest},
		{CodeMissingField, http.StatusBadRequest},
		{CodeNotFound, http.StatusNotFound},
		{CodeConflict, http.StatusConflict},
		{CodeAlreadyExists, http.StatusConflict},
		{CodeRateLimited, http.StatusTooManyRequests},
		{CodeTimeout, http.StatusGatewayTimeout},
		{CodeExternalService, http.StatusBadGateway},
		{CodeCircuitOpen, http.StatusBadGateway},
		{CodeInternal, http.StatusInternalServerError},
		{CodeDatabase, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test")
			if got := err.HTTPStatus(); got != tt.expected {
				t.Errorf("HTTPStatus() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestError_IsRetriable(t *testing.T) {
	tests := []struct {
		code      Code
		retriable bool
	}{
		{CodeRateLimited, true},
		{CodeTimeout, true},
		{CodeCircuitOpen, true},
		{CodeExternalService, true},
		{CodeProviderError, true},
		{CodeNotFound, false},
		{CodeValidation, false},
		{CodeUnauthorized, false},
		{CodeInternal, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test")
			if got := err.IsRetriable(); got != tt.retriable {
				t.Errorf("IsRetriable() = %v, expected %v", got, tt.retriable)
			}
		})
	}
}

func TestError_IsUserError(t *testing.T) {
	tests := []struct {
		code   Code
		isUser bool
	}{
		{CodeValidation, true},
		{CodeInvalidInput, true},
		{CodeUnauthorized, true},
		{CodeForbidden, true},
		{CodeNotFound, true},
		{CodeInternal, false},
		{CodeDatabase, false},
		{CodeRateLimited, false}, // Transient, not user
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test")
			if got := err.IsUserError(); got != tt.isUser {
				t.Errorf("IsUserError() = %v, expected %v", got, tt.isUser)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("root cause")
	err := Wrap(underlying, "auth.Login", CodeUnauthorized, "login failed")

	if err.Code != CodeUnauthorized {
		t.Errorf("Code = %q, expected %q", err.Code, CodeUnauthorized)
	}
	if err.Op != "auth.Login" {
		t.Errorf("Op = %q, expected %q", err.Op, "auth.Login")
	}
	if err.Message != "login failed" {
		t.Errorf("Message = %q, expected %q", err.Message, "login failed")
	}
	if !errors.Is(err, underlying) {
		t.Error("wrapped error should contain underlying error")
	}
}

func TestWrapWithOp(t *testing.T) {
	// Wrap an existing Error
	original := New(CodeNotFound, "user not found")
	wrapped := WrapWithOp(original, "handler.GetUser")

	if wrapped.Code != CodeNotFound {
		t.Errorf("Code = %q, expected %q", wrapped.Code, CodeNotFound)
	}
	if wrapped.Op != "handler.GetUser" {
		t.Errorf("Op = %q, expected %q", wrapped.Op, "handler.GetUser")
	}

	// Wrap a standard error
	stdErr := errors.New("some error")
	wrapped2 := WrapWithOp(stdErr, "handler.DoSomething")

	if wrapped2.Code != CodeInternal {
		t.Errorf("Code = %q, expected %q for non-Error", wrapped2.Code, CodeInternal)
	}
}

func TestSentinelErrors(t *testing.T) {
	// Test that sentinel errors have correct codes
	if ErrNotFound.Code != CodeNotFound {
		t.Errorf("ErrNotFound.Code = %q, expected %q", ErrNotFound.Code, CodeNotFound)
	}
	if ErrUnauthorized.Code != CodeUnauthorized {
		t.Errorf("ErrUnauthorized.Code = %q, expected %q", ErrUnauthorized.Code, CodeUnauthorized)
	}
	if ErrRateLimited.Code != CodeRateLimited {
		t.Errorf("ErrRateLimited.Code = %q, expected %q", ErrRateLimited.Code, CodeRateLimited)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("user")
	if err.Code != CodeNotFound {
		t.Errorf("Code = %q, expected %q", err.Code, CodeNotFound)
	}
	if err.Message != "user not found" {
		t.Errorf("Message = %q, expected %q", err.Message, "user not found")
	}
}

func TestMissingField(t *testing.T) {
	err := MissingField("email")
	if err.Code != CodeMissingField {
		t.Errorf("Code = %q, expected %q", err.Code, CodeMissingField)
	}
	if err.Message != "missing required field: email" {
		t.Errorf("Message = %q", err.Message)
	}
}

func TestInvalidFormat(t *testing.T) {
	err := InvalidFormat("phone", "E.164 format")
	if err.Code != CodeInvalidFormat {
		t.Errorf("Code = %q, expected %q", err.Code, CodeInvalidFormat)
	}
	if err.Message != "invalid format for phone: expected E.164 format" {
		t.Errorf("Message = %q", err.Message)
	}
}

func TestDatabaseError(t *testing.T) {
	underlying := errors.New("connection refused")
	err := DatabaseError("users.Create", underlying)

	if err.Code != CodeDatabase {
		t.Errorf("Code = %q, expected %q", err.Code, CodeDatabase)
	}
	if err.Op != "users.Create" {
		t.Errorf("Op = %q, expected %q", err.Op, "users.Create")
	}
	if !errors.Is(err, underlying) {
		t.Error("should wrap underlying error")
	}
}

func TestExternalServiceError(t *testing.T) {
	underlying := errors.New("503 service unavailable")
	err := ExternalServiceError("Claude", underlying)

	if err.Code != CodeExternalService {
		t.Errorf("Code = %q, expected %q", err.Code, CodeExternalService)
	}
	if err.Message != "Claude service error" {
		t.Errorf("Message = %q", err.Message)
	}
	if err.Kind != KindTransient {
		t.Errorf("Kind = %v, expected KindTransient", err.Kind)
	}
}

func TestQuoteGenerationError(t *testing.T) {
	underlying := errors.New("API timeout")
	err := QuoteGenerationError(underlying)

	if err.Code != CodeQuoteGenerationFailed {
		t.Errorf("Code = %q, expected %q", err.Code, CodeQuoteGenerationFailed)
	}
	if err.Kind != KindTransient {
		t.Errorf("Kind = %v, expected KindTransient", err.Kind)
	}
}

func TestGetCode(t *testing.T) {
	// App error
	appErr := New(CodeNotFound, "not found")
	if got := GetCode(appErr); got != CodeNotFound {
		t.Errorf("GetCode(appErr) = %q, expected %q", got, CodeNotFound)
	}

	// Standard error
	stdErr := errors.New("some error")
	if got := GetCode(stdErr); got != CodeInternal {
		t.Errorf("GetCode(stdErr) = %q, expected %q", got, CodeInternal)
	}
}

func TestGetHTTPStatus(t *testing.T) {
	// App error
	appErr := New(CodeNotFound, "not found")
	if got := GetHTTPStatus(appErr); got != http.StatusNotFound {
		t.Errorf("GetHTTPStatus(appErr) = %d, expected %d", got, http.StatusNotFound)
	}

	// Standard error
	stdErr := errors.New("some error")
	if got := GetHTTPStatus(stdErr); got != http.StatusInternalServerError {
		t.Errorf("GetHTTPStatus(stdErr) = %d, expected %d", got, http.StatusInternalServerError)
	}
}

func TestIsRetriableHelper(t *testing.T) {
	if !IsRetriable(New(CodeRateLimited, "test")) {
		t.Error("CodeRateLimited should be retriable")
	}
	if IsRetriable(New(CodeNotFound, "test")) {
		t.Error("CodeNotFound should not be retriable")
	}
	if IsRetriable(errors.New("standard error")) {
		t.Error("standard errors should not be retriable")
	}
}

func TestIsNotFoundHelper(t *testing.T) {
	if !IsNotFound(New(CodeNotFound, "test")) {
		t.Error("CodeNotFound should be recognized")
	}
	if IsNotFound(New(CodeInternal, "test")) {
		t.Error("CodeInternal should not be recognized as not found")
	}
}

func TestIsUserErrorHelper(t *testing.T) {
	if !IsUserError(New(CodeValidation, "test")) {
		t.Error("CodeValidation should be user error")
	}
	if IsUserError(New(CodeInternal, "test")) {
		t.Error("CodeInternal should not be user error")
	}
}

func TestError_ToResponse(t *testing.T) {
	err := New(CodeNotFound, "user not found")
	resp := err.ToResponse()

	if resp.Error.Code != CodeNotFound {
		t.Errorf("Response.Error.Code = %q, expected %q", resp.Error.Code, CodeNotFound)
	}
	if resp.Error.Message != "user not found" {
		t.Errorf("Response.Error.Message = %q, expected %q", resp.Error.Message, "user not found")
	}
}

func TestErrorChaining(t *testing.T) {
	// Simulate error chain: database -> repository -> service -> handler
	dbErr := errors.New("connection refused")
	repoErr := DatabaseError("repo.GetUser", dbErr)
	serviceErr := WrapWithOp(repoErr, "service.GetUser")
	handlerErr := WrapWithOp(serviceErr, "handler.GetUser")

	// Should be able to find original error
	if !errors.Is(handlerErr, dbErr) {
		t.Error("should be able to find original database error in chain")
	}

	// Check error message includes all context (operation + message + underlying error)
	errMsg := handlerErr.Error()
	expected := "handler.GetUser: database operation failed: connection refused"
	if errMsg != expected {
		t.Errorf("Error() = %q, expected %q", errMsg, expected)
	}
}

func TestErrorWithFmtErrorf(t *testing.T) {
	// Test that errors work with fmt.Errorf wrapping
	original := New(CodeNotFound, "user not found")
	wrapped := fmt.Errorf("handler failed: %w", original)

	var appErr *Error
	if !errors.As(wrapped, &appErr) {
		t.Error("errors.As should find Error in fmt.Errorf wrapped error")
	}
	if appErr.Code != CodeNotFound {
		t.Errorf("Code = %q, expected %q", appErr.Code, CodeNotFound)
	}
}
