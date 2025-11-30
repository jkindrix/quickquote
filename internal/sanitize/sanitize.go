// Package sanitize provides utilities for sanitizing sensitive data in logs, errors, and responses.
package sanitize

import (
	"regexp"
	"strings"
)

// Common patterns for sensitive data
var (
	// Phone patterns
	phonePattern = regexp.MustCompile(`\+?[1-9]\d{6,14}`)

	// Email pattern
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// API key patterns (various formats)
	apiKeyPattern = regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password|auth)[=:\s"']*([\w-]{16,})`)

	// Bearer token pattern
	bearerPattern = regexp.MustCompile(`(?i)bearer\s+[\w.-]+`)

	// Credit card pattern (basic)
	creditCardPattern = regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`)

	// SSN pattern
	ssnPattern = regexp.MustCompile(`\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b`)

	// UUID pattern for redacting specific IDs if needed
	uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// Sanitizer provides methods for sanitizing sensitive data.
type Sanitizer struct {
	// Patterns to apply
	patterns []patternConfig
}

type patternConfig struct {
	pattern     *regexp.Regexp
	replacement func(string) string
	enabled     bool
}

// Config holds configuration for the sanitizer.
type Config struct {
	// MaskPhones masks phone numbers
	MaskPhones bool
	// MaskEmails masks email addresses
	MaskEmails bool
	// MaskAPIKeys masks API keys and tokens
	MaskAPIKeys bool
	// MaskCreditCards masks credit card numbers
	MaskCreditCards bool
	// MaskSSN masks social security numbers
	MaskSSN bool
	// MaskBearerTokens masks bearer tokens
	MaskBearerTokens bool
}

// DefaultConfig returns a configuration with all masking enabled.
func DefaultConfig() Config {
	return Config{
		MaskPhones:       true,
		MaskEmails:       true,
		MaskAPIKeys:      true,
		MaskCreditCards:  true,
		MaskSSN:          true,
		MaskBearerTokens: true,
	}
}

// New creates a new Sanitizer with the given configuration.
func New(cfg Config) *Sanitizer {
	s := &Sanitizer{
		patterns: []patternConfig{
			{
				pattern:     phonePattern,
				replacement: maskPhone,
				enabled:     cfg.MaskPhones,
			},
			{
				pattern:     emailPattern,
				replacement: maskEmail,
				enabled:     cfg.MaskEmails,
			},
			{
				pattern:     apiKeyPattern,
				replacement: maskAPIKey,
				enabled:     cfg.MaskAPIKeys,
			},
			{
				pattern:     bearerPattern,
				replacement: maskBearer,
				enabled:     cfg.MaskBearerTokens,
			},
			{
				pattern:     creditCardPattern,
				replacement: maskCreditCard,
				enabled:     cfg.MaskCreditCards,
			},
			{
				pattern:     ssnPattern,
				replacement: maskSSN,
				enabled:     cfg.MaskSSN,
			},
		},
	}
	return s
}

// NewDefault creates a sanitizer with default configuration.
func NewDefault() *Sanitizer {
	return New(DefaultConfig())
}

// String sanitizes a string by masking all sensitive data.
func (s *Sanitizer) String(input string) string {
	result := input
	for _, p := range s.patterns {
		if p.enabled {
			result = p.pattern.ReplaceAllStringFunc(result, p.replacement)
		}
	}
	return result
}

// Map sanitizes string values in a map.
func (s *Sanitizer) Map(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(input))
	for k, v := range input {
		switch val := v.(type) {
		case string:
			// Check if key suggests sensitive data
			if isSensitiveKey(k) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = s.String(val)
			}
		case map[string]interface{}:
			result[k] = s.Map(val)
		default:
			result[k] = v
		}
	}
	return result
}

// Error sanitizes an error message.
func (s *Sanitizer) Error(err error) string {
	if err == nil {
		return ""
	}
	return s.String(err.Error())
}

// Headers sanitizes HTTP headers.
func (s *Sanitizer) Headers(headers map[string][]string) map[string][]string {
	result := make(map[string][]string, len(headers))
	for k, vals := range headers {
		lowerKey := strings.ToLower(k)
		if isSensitiveHeader(lowerKey) {
			result[k] = []string{"[REDACTED]"}
		} else {
			sanitized := make([]string, len(vals))
			for i, v := range vals {
				sanitized[i] = s.String(v)
			}
			result[k] = sanitized
		}
	}
	return result
}

// Masking functions

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	// Keep first 3 and last 2 characters
	return phone[:3] + strings.Repeat("*", len(phone)-5) + phone[len(phone)-2:]
}

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return "[email]"
	}
	if at <= 2 {
		return email[:1] + "***" + email[at:]
	}
	return email[:2] + "***" + email[at:]
}

func maskAPIKey(match string) string {
	// Find the key/value separator
	parts := apiKeyPattern.FindStringSubmatch(match)
	if len(parts) >= 2 {
		// Preserve the key name but mask the value
		prefix := strings.TrimSuffix(match, parts[len(parts)-1])
		return prefix + "[REDACTED]"
	}
	return "[REDACTED-KEY]"
}

func maskBearer(match string) string {
	return "Bearer [REDACTED]"
}

func maskCreditCard(cc string) string {
	// Keep last 4 digits
	clean := strings.ReplaceAll(strings.ReplaceAll(cc, "-", ""), " ", "")
	if len(clean) < 4 {
		return "****"
	}
	return "****-****-****-" + clean[len(clean)-4:]
}

func maskSSN(ssn string) string {
	return "***-**-****"
}

// Helper functions

// isSensitiveKey checks if a map key suggests sensitive data.
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"secret", "token", "auth",
		"api_key", "apikey", "api-key",
		"private", "credential",
		"ssn", "social",
		"credit_card", "creditcard", "card_number",
		"cvv", "cvc", "pin",
		"session_id", "sessionid",
		"refresh_token", "access_token",
	}
	for _, sk := range sensitiveKeys {
		if strings.Contains(lower, sk) {
			return true
		}
	}
	return false
}

// isSensitiveHeader checks if an HTTP header is sensitive.
func isSensitiveHeader(header string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
		"x-auth-token",
		"cookie",
		"set-cookie",
		"x-csrf-token",
		"x-session-id",
		"proxy-authorization",
		"www-authenticate",
	}
	for _, h := range sensitiveHeaders {
		if header == h {
			return true
		}
	}
	return false
}

// Quick sanitization functions for common use cases

// Phone masks a phone number.
func Phone(phone string) string {
	return maskPhone(phone)
}

// Email masks an email address.
func Email(email string) string {
	return maskEmail(email)
}

// APIKey masks an API key.
func APIKey(key string) string {
	if len(key) <= 8 {
		return "[REDACTED]"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// CreditCard masks a credit card number.
func CreditCard(cc string) string {
	return maskCreditCard(cc)
}

// SSN masks a social security number.
func SSN(ssn string) string {
	return maskSSN(ssn)
}

// PartialMask masks the middle portion of a string, keeping first and last N chars.
func PartialMask(s string, keepStart, keepEnd int) string {
	if len(s) <= keepStart+keepEnd {
		return strings.Repeat("*", len(s))
	}
	return s[:keepStart] + strings.Repeat("*", len(s)-keepStart-keepEnd) + s[len(s)-keepEnd:]
}

// ID partially masks an identifier, showing first 4 and last 4 characters.
func ID(id string) string {
	return PartialMask(id, 4, 4)
}
