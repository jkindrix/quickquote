// Package validation provides input validation for webhook payloads and API requests.
package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a validation failure with field context.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// FieldErrors returns errors for a specific field.
func (e ValidationErrors) FieldErrors(field string) ValidationErrors {
	var result ValidationErrors
	for _, err := range e {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}

// Error codes for validation failures.
const (
	CodeRequired      = "required"
	CodeInvalidFormat = "invalid_format"
	CodeTooLong       = "too_long"
	CodeTooShort      = "too_short"
	CodeInvalidValue  = "invalid_value"
	CodeMalicious     = "malicious_content"
)

// Validator provides validation methods for webhook payloads.
type Validator struct {
	errors ValidationErrors
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{}
}

// Errors returns all accumulated validation errors.
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// IsValid returns true if no validation errors occurred.
func (v *Validator) IsValid() bool {
	return len(v.errors) == 0
}

// AddError adds a validation error.
func (v *Validator) AddError(field, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// Required validates that a string field is not empty.
func (v *Validator) Required(field, value string) bool {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "is required", CodeRequired)
		return false
	}
	return true
}

// MaxLength validates string length doesn't exceed maximum.
func (v *Validator) MaxLength(field, value string, maxLen int) bool {
	if utf8.RuneCountInString(value) > maxLen {
		v.AddError(field, fmt.Sprintf("must be at most %d characters", maxLen), CodeTooLong)
		return false
	}
	return true
}

// MinLength validates string length meets minimum.
func (v *Validator) MinLength(field, value string, minLen int) bool {
	if utf8.RuneCountInString(value) < minLen {
		v.AddError(field, fmt.Sprintf("must be at least %d characters", minLen), CodeTooShort)
		return false
	}
	return true
}

// phoneRegex matches international phone numbers.
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

// PhoneNumber validates a phone number format.
func (v *Validator) PhoneNumber(field, value string) bool {
	if value == "" {
		return true // Use Required() separately if needed
	}
	// Remove common formatting characters
	cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(value)
	if !phoneRegex.MatchString(cleaned) {
		v.AddError(field, "must be a valid phone number in E.164 format", CodeInvalidFormat)
		return false
	}
	return true
}

// uuidRegex matches UUID format.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID validates a UUID format.
func (v *Validator) UUID(field, value string) bool {
	if value == "" {
		return true // Use Required() separately if needed
	}
	if !uuidRegex.MatchString(value) {
		v.AddError(field, "must be a valid UUID", CodeInvalidFormat)
		return false
	}
	return true
}

// urlRegex matches http/https URLs.
var urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].\S*$`)

// URL validates a URL format.
func (v *Validator) URL(field, value string) bool {
	if value == "" {
		return true
	}
	if !urlRegex.MatchString(value) {
		v.AddError(field, "must be a valid URL", CodeInvalidFormat)
		return false
	}
	return true
}

// OneOf validates that value is one of the allowed values.
func (v *Validator) OneOf(field, value string, allowed []string) bool {
	if value == "" {
		return true
	}
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	v.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")), CodeInvalidValue)
	return false
}

// NoScriptTags validates that the value doesn't contain script tags (XSS prevention).
func (v *Validator) NoScriptTags(field, value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "javascript:") {
		v.AddError(field, "contains potentially malicious content", CodeMalicious)
		return false
	}
	return true
}

// SafeString validates a string is safe for display (no control characters except newlines).
func (v *Validator) SafeString(field, value string) bool {
	for _, r := range value {
		// Allow printable characters, newlines, tabs
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			v.AddError(field, "contains invalid control characters", CodeMalicious)
			return false
		}
	}
	return true
}

// NonNegativeInt validates that an integer is not negative (zero or positive).
func (v *Validator) NonNegativeInt(field string, value int) bool {
	if value < 0 {
		v.AddError(field, "must not be negative", CodeInvalidValue)
		return false
	}
	return true
}

// Range validates an integer is within range.
func (v *Validator) Range(field string, value, minVal, maxVal int) bool {
	if value < minVal || value > maxVal {
		v.AddError(field, fmt.Sprintf("must be between %d and %d", minVal, maxVal), CodeInvalidValue)
		return false
	}
	return true
}

// CallEventValidator provides validation for CallEvent structures.
type CallEventValidator struct {
	*Validator
}

// NewCallEventValidator creates a CallEvent validator.
func NewCallEventValidator() *CallEventValidator {
	return &CallEventValidator{
		Validator: New(),
	}
}

// ValidateCallID validates the provider call ID.
func (v *CallEventValidator) ValidateCallID(callID string) {
	v.Required("provider_call_id", callID)
	v.MaxLength("provider_call_id", callID, 256)
	v.SafeString("provider_call_id", callID)
}

// ValidatePhoneNumbers validates phone number fields.
func (v *CallEventValidator) ValidatePhoneNumbers(toNumber, fromNumber string) {
	if toNumber != "" {
		v.PhoneNumber("to_number", toNumber)
	}
	if fromNumber != "" {
		v.PhoneNumber("from_number", fromNumber)
	}
}

// ValidateTranscript validates transcript content.
func (v *CallEventValidator) ValidateTranscript(transcript string) {
	v.MaxLength("transcript", transcript, 1000000) // 1MB limit
	v.SafeString("transcript", transcript)
	v.NoScriptTags("transcript", transcript)
}

// ValidateCallerName validates the caller name.
func (v *CallEventValidator) ValidateCallerName(name string) {
	if name == "" {
		return
	}
	v.MaxLength("caller_name", name, 256)
	v.SafeString("caller_name", name)
	v.NoScriptTags("caller_name", name)
}

// ValidateRecordingURL validates the recording URL.
func (v *CallEventValidator) ValidateRecordingURL(url string) {
	if url == "" {
		return
	}
	v.URL("recording_url", url)
	v.MaxLength("recording_url", url, 2048)
}

// ValidateStatus validates call status.
func (v *CallEventValidator) ValidateStatus(status string) {
	allowedStatuses := []string{
		"pending", "in_progress", "in-progress", "active",
		"completed", "success", "done",
		"failed", "error",
		"no_answer", "no-answer",
		"voicemail",
		"transferred",
		"",
	}
	v.OneOf("status", strings.ToLower(status), allowedStatuses)
}

// ValidateDuration validates call duration.
func (v *CallEventValidator) ValidateDuration(duration int) {
	v.NonNegativeInt("duration", duration)
	v.Range("duration", duration, 0, 86400) // Max 24 hours
}

// ValidateAll performs all validations and returns errors.
func (v *CallEventValidator) ValidateAll(
	callID, toNumber, fromNumber, transcript, callerName, recordingURL, status string,
	duration int,
) ValidationErrors {
	v.ValidateCallID(callID)
	v.ValidatePhoneNumbers(toNumber, fromNumber)
	v.ValidateTranscript(transcript)
	v.ValidateCallerName(callerName)
	v.ValidateRecordingURL(recordingURL)
	v.ValidateStatus(status)
	v.ValidateDuration(duration)
	return v.Errors()
}

// QuickValidateCallID performs a quick call ID validation.
func QuickValidateCallID(callID string) error {
	if strings.TrimSpace(callID) == "" {
		return errors.New("call_id is required")
	}
	if len(callID) > 256 {
		return errors.New("call_id exceeds maximum length")
	}
	return nil
}

// SanitizeString removes potentially dangerous characters from a string.
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Replace control characters (except newlines/tabs) with spaces
	var builder strings.Builder
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			builder.WriteRune(' ')
		} else {
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}

// SanitizePhoneNumber normalizes a phone number to E.164-ish format.
func SanitizePhoneNumber(phone string) string {
	// Remove all non-digit characters except leading +
	hasPlus := strings.HasPrefix(phone, "+")
	digits := strings.Builder{}
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	result := digits.String()
	if hasPlus && result != "" {
		return "+" + result
	}
	return result
}

// PaginationConfig contains constraints for pagination parameters.
type PaginationConfig struct {
	MaxLimit     int // Maximum allowed limit (default: 1000)
	DefaultLimit int // Default limit when not specified (default: 20)
	MaxOffset    int // Maximum allowed offset (default: 100000)
}

// DefaultPaginationConfig returns sensible defaults for pagination.
func DefaultPaginationConfig() *PaginationConfig {
	return &PaginationConfig{
		MaxLimit:     1000,
		DefaultLimit: 20,
		MaxOffset:    100000, // Prevent resource exhaustion from huge offsets
	}
}

// PaginationParams represents validated pagination parameters.
type PaginationParams struct {
	Limit  int
	Offset int
}

// ValidatePagination validates and normalizes pagination parameters.
// Returns validated params or an error if validation fails.
func ValidatePagination(limit, offset int, cfg *PaginationConfig) (*PaginationParams, error) {
	if cfg == nil {
		cfg = DefaultPaginationConfig()
	}

	// Normalize limit
	if limit <= 0 {
		limit = cfg.DefaultLimit
	}
	if limit > cfg.MaxLimit {
		return nil, fmt.Errorf("limit must not exceed %d (got %d)", cfg.MaxLimit, limit)
	}

	// Validate offset
	if offset < 0 {
		return nil, fmt.Errorf("offset must not be negative (got %d)", offset)
	}
	if offset > cfg.MaxOffset {
		return nil, fmt.Errorf("offset must not exceed %d (got %d)", cfg.MaxOffset, offset)
	}

	return &PaginationParams{
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ValidatePaginationWithDefaults validates pagination with default configuration.
func ValidatePaginationWithDefaults(limit, offset int) (*PaginationParams, error) {
	return ValidatePagination(limit, offset, nil)
}

// NormalizePaginationParams normalizes limit and offset without strict validation.
// Use this when you want to silently clamp values instead of returning errors.
func NormalizePaginationParams(limit, offset int, cfg *PaginationConfig) PaginationParams {
	if cfg == nil {
		cfg = DefaultPaginationConfig()
	}

	// Clamp limit
	if limit <= 0 {
		limit = cfg.DefaultLimit
	} else if limit > cfg.MaxLimit {
		limit = cfg.MaxLimit
	}

	// Clamp offset
	if offset < 0 {
		offset = 0
	} else if offset > cfg.MaxOffset {
		offset = cfg.MaxOffset
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// PaginationValidator extends Validator with pagination validation.
type PaginationValidator struct {
	*Validator
	config *PaginationConfig
}

// NewPaginationValidator creates a pagination validator with optional config.
func NewPaginationValidator(cfg *PaginationConfig) *PaginationValidator {
	if cfg == nil {
		cfg = DefaultPaginationConfig()
	}
	return &PaginationValidator{
		Validator: New(),
		config:    cfg,
	}
}

// ValidateLimit validates the limit parameter.
func (v *PaginationValidator) ValidateLimit(limit int) bool {
	if limit <= 0 {
		return true // Will be normalized to default
	}
	if limit > v.config.MaxLimit {
		v.AddError("limit", fmt.Sprintf("must not exceed %d", v.config.MaxLimit), CodeInvalidValue)
		return false
	}
	return true
}

// ValidateOffset validates the offset parameter.
func (v *PaginationValidator) ValidateOffset(offset int) bool {
	if offset < 0 {
		v.AddError("offset", "must not be negative", CodeInvalidValue)
		return false
	}
	if offset > v.config.MaxOffset {
		v.AddError("offset", fmt.Sprintf("must not exceed %d", v.config.MaxOffset), CodeInvalidValue)
		return false
	}
	return true
}

// Validate validates both limit and offset.
func (v *PaginationValidator) Validate(limit, offset int) bool {
	limitOk := v.ValidateLimit(limit)
	offsetOk := v.ValidateOffset(offset)
	return limitOk && offsetOk
}
