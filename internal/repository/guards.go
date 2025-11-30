// Package repository provides validation guards for repository operations.
// Guards validate inputs before database operations to fail fast and provide clear errors.
package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/jkindrix/quickquote/internal/errors"
)

// Guard provides validation methods for common repository inputs.
type Guard struct{}

// NewGuard creates a new validation guard.
func NewGuard() *Guard {
	return &Guard{}
}

// RequireUUID validates that a UUID is not nil/zero.
func (g *Guard) RequireUUID(id uuid.UUID, field string) error {
	if id == uuid.Nil {
		return apperrors.MissingField(field)
	}
	return nil
}

// RequireString validates that a string is not empty.
func (g *Guard) RequireString(s string, field string) error {
	if strings.TrimSpace(s) == "" {
		return apperrors.MissingField(field)
	}
	return nil
}

// RequireNonNegative validates that an integer is not negative.
func (g *Guard) RequireNonNegative(n int, field string) error {
	if n < 0 {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must not be negative", field))
	}
	return nil
}

// RequireNonNegativeInt64 validates that an int64 is not negative.
func (g *Guard) RequireNonNegativeInt64(n int64, field string) error {
	if n < 0 {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must not be negative", field))
	}
	return nil
}

// RequirePositive validates that an integer is positive.
func (g *Guard) RequirePositive(n int, field string) error {
	if n <= 0 {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must be positive", field))
	}
	return nil
}

// RequirePositiveInt64 validates that an int64 is positive.
func (g *Guard) RequirePositiveInt64(n int64, field string) error {
	if n <= 0 {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must be positive", field))
	}
	return nil
}

// RequireInRange validates that an integer is within a range.
func (g *Guard) RequireInRange(n, min, max int, field string) error {
	if n < min || n > max {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
	return nil
}

// RequireMaxLength validates that a string doesn't exceed a length.
func (g *Guard) RequireMaxLength(s string, maxLen int, field string) error {
	if len(s) > maxLen {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must not exceed %d characters", field, maxLen))
	}
	return nil
}

// RequireMinLength validates that a string has at least a minimum length.
func (g *Guard) RequireMinLength(s string, minLen int, field string) error {
	if len(s) < minLen {
		return apperrors.ValidationFailed(fmt.Sprintf("%s must be at least %d characters", field, minLen))
	}
	return nil
}

// RequireNotInFuture validates that a time is not in the future.
func (g *Guard) RequireNotInFuture(t time.Time, field string) error {
	if t.After(time.Now().Add(time.Minute)) { // Allow 1 minute clock skew
		return apperrors.ValidationFailed(fmt.Sprintf("%s must not be in the future", field))
	}
	return nil
}

// RequireNotInPast validates that a time is not in the past.
func (g *Guard) RequireNotInPast(t time.Time, field string) error {
	if t.Before(time.Now().Add(-time.Minute)) { // Allow 1 minute clock skew
		return apperrors.ValidationFailed(fmt.Sprintf("%s must not be in the past", field))
	}
	return nil
}

// RequireValidEmail performs basic email validation.
func (g *Guard) RequireValidEmail(email, field string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return apperrors.MissingField(field)
	}
	if !strings.Contains(email, "@") {
		return apperrors.InvalidFormat(field, "valid email address")
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return apperrors.InvalidFormat(field, "valid email address")
	}
	if !strings.Contains(parts[1], ".") {
		return apperrors.InvalidFormat(field, "valid email address")
	}
	return nil
}

// RequireEnum validates that a value is one of the allowed values.
func (g *Guard) RequireEnum(value string, allowed []string, field string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return apperrors.ValidationFailed(fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
}

// ValidationResult collects multiple validation errors.
type ValidationResult struct {
	errors []error
}

// NewValidationResult creates a new validation result collector.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		errors: make([]error, 0),
	}
}

// Add adds an error to the result if it's not nil.
func (v *ValidationResult) Add(err error) *ValidationResult {
	if err != nil {
		v.errors = append(v.errors, err)
	}
	return v
}

// Check validates a condition and adds an error if it fails.
func (v *ValidationResult) Check(condition bool, field, message string) *ValidationResult {
	if !condition {
		v.errors = append(v.errors, apperrors.ValidationFailed(fmt.Sprintf("%s %s", field, message)))
	}
	return v
}

// RequireUUID adds a UUID validation.
func (v *ValidationResult) RequireUUID(id uuid.UUID, field string) *ValidationResult {
	if id == uuid.Nil {
		v.errors = append(v.errors, apperrors.MissingField(field))
	}
	return v
}

// RequireString adds a string validation.
func (v *ValidationResult) RequireString(s string, field string) *ValidationResult {
	if strings.TrimSpace(s) == "" {
		v.errors = append(v.errors, apperrors.MissingField(field))
	}
	return v
}

// RequireNonNegative adds a non-negative integer validation.
func (v *ValidationResult) RequireNonNegative(n int, field string) *ValidationResult {
	if n < 0 {
		v.errors = append(v.errors, apperrors.ValidationFailed(fmt.Sprintf("%s must not be negative", field)))
	}
	return v
}

// RequirePositive adds a positive integer validation.
func (v *ValidationResult) RequirePositive(n int, field string) *ValidationResult {
	if n <= 0 {
		v.errors = append(v.errors, apperrors.ValidationFailed(fmt.Sprintf("%s must be positive", field)))
	}
	return v
}

// RequireMaxLength adds a max length validation.
func (v *ValidationResult) RequireMaxLength(s string, maxLen int, field string) *ValidationResult {
	if len(s) > maxLen {
		v.errors = append(v.errors, apperrors.ValidationFailed(fmt.Sprintf("%s must not exceed %d characters", field, maxLen)))
	}
	return v
}

// RequireValidEmail adds an email validation.
func (v *ValidationResult) RequireValidEmail(email, field string) *ValidationResult {
	email = strings.TrimSpace(email)
	if email == "" {
		v.errors = append(v.errors, apperrors.MissingField(field))
	} else if !strings.Contains(email, "@") || len(strings.Split(email, "@")) != 2 {
		v.errors = append(v.errors, apperrors.InvalidFormat(field, "valid email address"))
	}
	return v
}

// RequireEnum adds an enum validation.
func (v *ValidationResult) RequireEnum(value string, allowed []string, field string) *ValidationResult {
	found := false
	for _, a := range allowed {
		if value == a {
			found = true
			break
		}
	}
	if !found {
		v.errors = append(v.errors, apperrors.ValidationFailed(fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", "))))
	}
	return v
}

// HasErrors returns true if there are any validation errors.
func (v *ValidationResult) HasErrors() bool {
	return len(v.errors) > 0
}

// Error returns a combined error message, or nil if no errors.
func (v *ValidationResult) Error() error {
	if len(v.errors) == 0 {
		return nil
	}
	if len(v.errors) == 1 {
		return v.errors[0]
	}

	messages := make([]string, len(v.errors))
	for i, err := range v.errors {
		messages[i] = err.Error()
	}
	return fmt.Errorf("validation errors: %s", strings.Join(messages, "; "))
}

// Errors returns all validation errors.
func (v *ValidationResult) Errors() []error {
	return v.errors
}

// Count returns the number of validation errors.
func (v *ValidationResult) Count() int {
	return len(v.errors)
}

// Pagination validation helpers

// ValidatePagination validates limit and offset parameters.
func (g *Guard) ValidatePagination(limit, offset, maxLimit int) error {
	if limit < 0 {
		return apperrors.ValidationFailed("limit must not be negative")
	}
	if limit > maxLimit {
		return apperrors.ValidationFailed(fmt.Sprintf("limit must not exceed %d", maxLimit))
	}
	if offset < 0 {
		return apperrors.ValidationFailed("offset must not be negative")
	}
	return nil
}

// NormalizePagination normalizes limit and offset to safe values.
// Returns normalized limit and offset.
func (g *Guard) NormalizePagination(limit, offset, defaultLimit, maxLimit int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// Entity-specific validation

// ValidateCallStatus validates a call status string.
func (g *Guard) ValidateCallStatus(status string) error {
	validStatuses := []string{"pending", "in_progress", "completed", "failed", "quote_generated"}
	return g.RequireEnum(status, validStatuses, "status")
}

// ValidateProviderType validates a voice provider type.
func (g *Guard) ValidateProviderType(provider string) error {
	validProviders := []string{"bland", "vapi", "retell"}
	return g.RequireEnum(provider, validProviders, "provider")
}

// ValidateSyncStatus validates a sync status string.
func (g *Guard) ValidateSyncStatus(status string) error {
	validStatuses := []string{"draft", "syncing", "active", "error"}
	return g.RequireEnum(status, validStatuses, "status")
}

// ValidateDocumentStatus validates a document status string.
func (g *Guard) ValidateDocumentStatus(status string) error {
	validStatuses := []string{"pending", "processing", "active", "error"}
	return g.RequireEnum(status, validStatuses, "status")
}

// Default guard instance for convenience
var defaultGuard = NewGuard()

// Validate returns a new ValidationResult for fluent validation.
func Validate() *ValidationResult {
	return NewValidationResult()
}

// GuardUUID is a convenience function for UUID validation.
func GuardUUID(id uuid.UUID, field string) error {
	return defaultGuard.RequireUUID(id, field)
}

// GuardString is a convenience function for string validation.
func GuardString(s string, field string) error {
	return defaultGuard.RequireString(s, field)
}

// GuardPagination is a convenience function for pagination validation.
func GuardPagination(limit, offset, maxLimit int) error {
	return defaultGuard.ValidatePagination(limit, offset, maxLimit)
}

// GuardEmail is a convenience function for email validation.
func GuardEmail(email, field string) error {
	return defaultGuard.RequireValidEmail(email, field)
}
