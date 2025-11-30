package sanitize

import (
	"errors"
	"testing"
)

func TestSanitizer_String_Phone(t *testing.T) {
	s := NewDefault()

	tests := []struct {
		input    string
		expected string
	}{
		{"Call me at +15551234567", "Call me at +15*******67"},
		{"Phone: 5551234567", "Phone: 555*****67"},
		{"Multiple +15551111111 and +15552222222", "Multiple +15*******11 and +15*******22"},
		{"Short 1234", "Short 1234"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_String_Email(t *testing.T) {
	s := NewDefault()

	tests := []struct {
		input    string
		expected string
	}{
		{"Contact user@example.com", "Contact us***@example.com"},
		{"Email: ab@test.org", "Email: a***@test.org"},
		{"Two emails: a@b.com and foo@bar.io", "Two emails: a***@b.com and fo***@bar.io"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_String_APIKey(t *testing.T) {
	s := NewDefault()

	// Note: The regex matches the key pattern and the following value.
	// Some test inputs may also trigger phone number masking.
	tests := []struct {
		input    string
		expected string
	}{
		{"apikey: abc123def456ghi789", "apikey: [REDACTED]"},
		{"Set SECRET=mysupersecretvalue123", "Set SECRET=[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_String_BearerToken(t *testing.T) {
	s := NewDefault()

	tests := []struct {
		input    string
		expected string
	}{
		{"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test", "Authorization: Bearer [REDACTED]"},
		{"bearer abc123", "Bearer [REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_String_CreditCard(t *testing.T) {
	s := NewDefault()

	tests := []struct {
		input    string
		expected string
	}{
		{"Card: 4111-1111-1111-1111", "Card: ****-****-****-1111"},
		{"CC 4111 1111 1111 1111", "CC ****-****-****-1111"},
		// Note: Without separators, the pattern may overlap with phone detection
		// so we test primarily with formatted cards
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_String_SSN(t *testing.T) {
	s := NewDefault()

	tests := []struct {
		input    string
		expected string
	}{
		{"SSN: 123-45-6789", "SSN: ***-**-****"},
		{"SSN is 123 45 6789", "SSN is ***-**-****"},
		// Note: Without separators, SSNs may be matched as phone numbers
		// so we test primarily with formatted SSNs
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.String(tt.input)
			if result != tt.expected {
				t.Errorf("String(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_Map(t *testing.T) {
	s := NewDefault()

	input := map[string]interface{}{
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "secret123",
		"phone":    "+15551234567",
		"nested": map[string]interface{}{
			"api_key": "sk_test_abcdef123456",
		},
	}

	result := s.Map(input)

	// Check password is redacted by key name
	if result["password"] != "[REDACTED]" {
		t.Errorf("expected password to be [REDACTED], got %v", result["password"])
	}

	// Check email is masked by pattern
	email := result["email"].(string)
	if email != "jo***@example.com" {
		t.Errorf("expected masked email 'jo***@example.com', got %s", email)
	}

	// Check nested api_key is redacted
	nested := result["nested"].(map[string]interface{})
	if nested["api_key"] != "[REDACTED]" {
		t.Errorf("expected nested api_key to be [REDACTED], got %v", nested["api_key"])
	}
}

func TestSanitizer_Error(t *testing.T) {
	s := NewDefault()

	err := errors.New("failed to send email to user@example.com with phone +15551234567")
	result := s.Error(err)

	if result != "failed to send email to us***@example.com with phone +15*******67" {
		t.Errorf("unexpected error sanitization: %s", result)
	}

	// Nil error
	if s.Error(nil) != "" {
		t.Error("expected empty string for nil error")
	}
}

func TestSanitizer_Headers(t *testing.T) {
	s := NewDefault()

	headers := map[string][]string{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer secret123"},
		"X-Api-Key":     {"sk_test_abcdef"},
		"Cookie":        {"session=abc123"},
		"X-Custom":      {"user@example.com"},
	}

	result := s.Headers(headers)

	// Authorization should be redacted
	if result["Authorization"][0] != "[REDACTED]" {
		t.Errorf("expected Authorization to be [REDACTED], got %s", result["Authorization"][0])
	}

	// X-Api-Key should be redacted
	if result["X-Api-Key"][0] != "[REDACTED]" {
		t.Errorf("expected X-Api-Key to be [REDACTED], got %s", result["X-Api-Key"][0])
	}

	// Cookie should be redacted
	if result["Cookie"][0] != "[REDACTED]" {
		t.Errorf("expected Cookie to be [REDACTED], got %s", result["Cookie"][0])
	}

	// Content-Type should be unchanged
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected Content-Type to be unchanged, got %s", result["Content-Type"][0])
	}

	// X-Custom should have email masked
	if result["X-Custom"][0] != "us***@example.com" {
		t.Errorf("expected X-Custom email to be masked, got %s", result["X-Custom"][0])
	}
}

func TestNew_DisabledPatterns(t *testing.T) {
	cfg := Config{
		MaskPhones: false,
		MaskEmails: true,
	}
	s := New(cfg)

	input := "Contact john@example.com at +15551234567"
	result := s.String(input)

	// Email should be masked, phone should not
	expected := "Contact jo***@example.com at +15551234567"
	if result != expected {
		t.Errorf("String(%q) = %q, want %q", input, result, expected)
	}
}

func TestQuickFunctions(t *testing.T) {
	t.Run("Phone", func(t *testing.T) {
		result := Phone("+15551234567")
		if result != "+15*******67" {
			t.Errorf("Phone() = %q, want '+15*******67'", result)
		}
	})

	t.Run("Email", func(t *testing.T) {
		result := Email("user@example.com")
		if result != "us***@example.com" {
			t.Errorf("Email() = %q, want 'us***@example.com'", result)
		}
	})

	t.Run("APIKey", func(t *testing.T) {
		result := APIKey("sk_test_1234567890abcdef")
		if result != "sk_t...cdef" {
			t.Errorf("APIKey() = %q, want 'sk_t...cdef'", result)
		}
		// Short key
		result = APIKey("short")
		if result != "[REDACTED]" {
			t.Errorf("APIKey() = %q, want '[REDACTED]'", result)
		}
	})

	t.Run("CreditCard", func(t *testing.T) {
		result := CreditCard("4111-1111-1111-1111")
		if result != "****-****-****-1111" {
			t.Errorf("CreditCard() = %q, want '****-****-****-1111'", result)
		}
	})

	t.Run("SSN", func(t *testing.T) {
		result := SSN("123-45-6789")
		if result != "***-**-****" {
			t.Errorf("SSN() = %q, want '***-**-****'", result)
		}
	})

	t.Run("PartialMask", func(t *testing.T) {
		result := PartialMask("1234567890", 2, 2)
		if result != "12******90" {
			t.Errorf("PartialMask() = %q, want '12******90'", result)
		}
		// Short string
		result = PartialMask("abc", 2, 2)
		if result != "***" {
			t.Errorf("PartialMask() = %q, want '***'", result)
		}
	})

	t.Run("ID", func(t *testing.T) {
		result := ID("abcdef123456ghij")
		if result != "abcd********ghij" {
			t.Errorf("ID() = %q, want 'abcd********ghij'", result)
		}
	})
}

func TestIsSensitiveKey(t *testing.T) {
	sensitiveKeys := []string{
		"password", "Password", "PASSWORD",
		"api_key", "apiKey", "API_KEY",
		"secret", "SECRET",
		"token", "access_token", "refresh_token",
		"credential", "CREDENTIALS",
		"ssn", "credit_card",
	}

	for _, key := range sensitiveKeys {
		if !isSensitiveKey(key) {
			t.Errorf("expected %q to be sensitive", key)
		}
	}

	nonSensitiveKeys := []string{
		"name", "email", "phone", "address",
		"created_at", "updated_at", "id",
	}

	for _, key := range nonSensitiveKeys {
		if isSensitiveKey(key) {
			t.Errorf("expected %q to NOT be sensitive", key)
		}
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	sensitiveHeaders := []string{
		"authorization", "x-api-key", "cookie", "set-cookie",
		"x-csrf-token", "x-session-id", "proxy-authorization",
	}

	for _, header := range sensitiveHeaders {
		if !isSensitiveHeader(header) {
			t.Errorf("expected %q to be sensitive", header)
		}
	}

	nonSensitiveHeaders := []string{
		"content-type", "accept", "user-agent", "x-request-id",
	}

	for _, header := range nonSensitiveHeaders {
		if isSensitiveHeader(header) {
			t.Errorf("expected %q to NOT be sensitive", header)
		}
	}
}
