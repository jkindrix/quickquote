package validation

import (
	"strings"
	"testing"
)

func TestValidator_Required(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"non-empty", "hello", true},
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"tabs only", "\t\t", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.Required("field", tt.value)
			if result != tt.isValid {
				t.Errorf("Required() = %v, want %v", result, tt.isValid)
			}
			if tt.isValid && len(v.Errors()) > 0 {
				t.Errorf("expected no errors, got %v", v.Errors())
			}
			if !tt.isValid && len(v.Errors()) == 0 {
				t.Error("expected errors, got none")
			}
		})
	}
}

func TestValidator_MaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		max     int
		isValid bool
	}{
		{"under limit", "hello", 10, true},
		{"at limit", "hello", 5, true},
		{"over limit", "hello world", 5, false},
		{"empty string", "", 5, true},
		{"unicode characters", "héllo", 5, true},
		{"unicode over limit", "héllo wörld", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.MaxLength("field", tt.value, tt.max)
			if result != tt.isValid {
				t.Errorf("MaxLength() = %v, want %v", result, tt.isValid)
			}
		})
	}
}

func TestValidator_MinLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min     int
		isValid bool
	}{
		{"over minimum", "hello world", 5, true},
		{"at minimum", "hello", 5, true},
		{"under minimum", "hi", 5, false},
		{"empty string", "", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.MinLength("field", tt.value, tt.min)
			if result != tt.isValid {
				t.Errorf("MinLength() = %v, want %v", result, tt.isValid)
			}
		})
	}
}

func TestValidator_PhoneNumber(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"valid E.164", "+14155551234", true},
		{"valid without plus", "14155551234", true},
		{"valid with spaces", "+1 415 555 1234", true},
		{"valid with dashes", "+1-415-555-1234", true},
		{"valid with parens", "+1 (415) 555-1234", true},
		{"valid international", "+442071234567", true},
		{"empty allowed", "", true},
		{"too short", "+1", false},
		{"letters invalid", "+1abc5551234", false},
		{"too long", "+123456789012345678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.PhoneNumber("phone", tt.value)
			if result != tt.isValid {
				t.Errorf("PhoneNumber(%q) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_UUID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"valid lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid mixed case", "550E8400-e29b-41D4-A716-446655440000", true},
		{"empty allowed", "", true},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false},
		{"wrong length", "550e8400-e29b-41d4-a716-44665544000", false},
		{"invalid chars", "550e8400-e29b-41d4-a716-44665544000g", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.UUID("id", tt.value)
			if result != tt.isValid {
				t.Errorf("UUID(%q) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_URL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"valid https", "https://example.com/path", true},
		{"valid http", "http://example.com", true},
		{"with query", "https://example.com/path?q=1", true},
		{"with fragment", "https://example.com/path#section", true},
		{"empty allowed", "", true},
		{"no scheme", "example.com", false},
		{"ftp scheme", "ftp://example.com", false},
		{"javascript", "javascript:alert(1)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.URL("url", tt.value)
			if result != tt.isValid {
				t.Errorf("URL(%q) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_OneOf(t *testing.T) {
	allowed := []string{"apple", "banana", "cherry"}

	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"first option", "apple", true},
		{"last option", "cherry", true},
		{"not allowed", "orange", false},
		{"empty allowed", "", true},
		{"case sensitive", "Apple", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.OneOf("fruit", tt.value, allowed)
			if result != tt.isValid {
				t.Errorf("OneOf(%q) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_NoScriptTags(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"clean text", "Hello world", true},
		{"html safe", "<b>bold</b>", true},
		{"script tag", "<script>alert(1)</script>", false},
		{"uppercase script", "<SCRIPT>alert(1)</SCRIPT>", false},
		{"mixed case script", "<ScRiPt>alert(1)</script>", false},
		{"javascript protocol", "javascript:alert(1)", false},
		{"clean url", "https://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.NoScriptTags("content", tt.value)
			if result != tt.isValid {
				t.Errorf("NoScriptTags(%q) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_SafeString(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"normal text", "Hello world", true},
		{"with newline", "Hello\nworld", true},
		{"with tab", "Hello\tworld", true},
		{"with carriage return", "Hello\rworld", true},
		{"with null byte", "Hello\x00world", false},
		{"with control char", "Hello\x01world", false},
		{"with bell", "Hello\x07world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.SafeString("text", tt.value)
			if result != tt.isValid {
				t.Errorf("SafeString() = %v, want %v", result, tt.isValid)
			}
		})
	}
}

func TestValidator_NonNegativeInt(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		isValid bool
	}{
		{"positive", 5, true},
		{"zero", 0, true},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.NonNegativeInt("count", tt.value)
			if result != tt.isValid {
				t.Errorf("NonNegativeInt(%d) = %v, want %v", tt.value, result, tt.isValid)
			}
		})
	}
}

func TestValidator_Range(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		min     int
		max     int
		isValid bool
	}{
		{"in range", 5, 1, 10, true},
		{"at min", 1, 1, 10, true},
		{"at max", 10, 1, 10, true},
		{"below min", 0, 1, 10, false},
		{"above max", 11, 1, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := v.Range("value", tt.value, tt.min, tt.max)
			if result != tt.isValid {
				t.Errorf("Range(%d, %d, %d) = %v, want %v", tt.value, tt.min, tt.max, result, tt.isValid)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "name", Message: "is required", Code: CodeRequired},
		{Field: "email", Message: "is invalid", Code: CodeInvalidFormat},
	}

	result := errs.Error()
	if !strings.Contains(result, "name") || !strings.Contains(result, "email") {
		t.Errorf("Error() should contain field names, got: %s", result)
	}
}

func TestValidationErrors_FieldErrors(t *testing.T) {
	errs := ValidationErrors{
		{Field: "name", Message: "is required"},
		{Field: "email", Message: "is invalid"},
		{Field: "name", Message: "is too short"},
	}

	nameErrors := errs.FieldErrors("name")
	if len(nameErrors) != 2 {
		t.Errorf("FieldErrors(name) = %d errors, want 2", len(nameErrors))
	}
}

func TestCallEventValidator_ValidateAll(t *testing.T) {
	// Valid case
	v := NewCallEventValidator()
	errs := v.ValidateAll(
		"call-123",
		"+14155551234",
		"+14155550000",
		"Hello, this is a test transcript.",
		"John Doe",
		"https://example.com/recording.mp3",
		"completed",
		120,
	)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid input, got: %v", errs)
	}

	// Invalid case - empty call ID
	v2 := NewCallEventValidator()
	errs2 := v2.ValidateAll(
		"",
		"+14155551234",
		"+14155550000",
		"Test",
		"John",
		"https://example.com",
		"completed",
		120,
	)
	if len(errs2) == 0 {
		t.Error("expected errors for empty call_id")
	}
}

func TestCallEventValidator_ValidateTranscript_XSS(t *testing.T) {
	v := NewCallEventValidator()
	v.ValidateTranscript("<script>alert('xss')</script>")
	if v.IsValid() {
		t.Error("expected validation to fail for script tag in transcript")
	}
}

func TestQuickValidateCallID(t *testing.T) {
	tests := []struct {
		name    string
		callID  string
		wantErr bool
	}{
		{"valid", "call-123", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"too long", strings.Repeat("a", 300), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuickValidateCallID(tt.callID)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuickValidateCallID(%q) error = %v, wantErr %v", tt.callID, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean string", "hello world", "hello world"},
		{"with null byte", "hello\x00world", "helloworld"},
		{"with control char", "hello\x01world", "hello world"},
		{"preserves newline", "hello\nworld", "hello\nworld"},
		{"preserves tab", "hello\tworld", "hello\tworld"},
		{"trims whitespace", "  hello  ", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"E.164 format", "+14155551234", "+14155551234"},
		{"with spaces", "+1 415 555 1234", "+14155551234"},
		{"with dashes", "+1-415-555-1234", "+14155551234"},
		{"with parens", "+1 (415) 555-1234", "+14155551234"},
		{"no plus", "14155551234", "14155551234"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePhoneNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
