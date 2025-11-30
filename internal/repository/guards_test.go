package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGuard_RequireUUID(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{"valid uuid", uuid.New(), false},
		{"nil uuid", uuid.Nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireUUID(tt.id, "id")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireString(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		s       string
		wantErr bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"string with spaces", " hello ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireString(tt.s, "name")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireNonNegative(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"positive", 5, false},
		{"zero", 0, false},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireNonNegative(tt.n, "count")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireNonNegative() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequirePositive(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"positive", 5, false},
		{"zero", 0, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequirePositive(tt.n, "limit")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequirePositive() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireInRange(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		n       int
		min     int
		max     int
		wantErr bool
	}{
		{"in range", 5, 1, 10, false},
		{"at min", 1, 1, 10, false},
		{"at max", 10, 1, 10, false},
		{"below min", 0, 1, 10, true},
		{"above max", 11, 1, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireInRange(tt.n, tt.min, tt.max, "value")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireInRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireMaxLength(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		s       string
		maxLen  int
		wantErr bool
	}{
		{"under max", "hello", 10, false},
		{"at max", "hello", 5, false},
		{"over max", "hello world", 5, true},
		{"empty string", "", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireMaxLength(tt.s, tt.maxLen, "name")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireMaxLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireValidEmail(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "test@mail.example.com", false},
		{"empty", "", true},
		{"no at sign", "testexample.com", true},
		{"no domain", "test@", true},
		{"no local part", "@example.com", true},
		{"no tld", "test@example", true},
		{"multiple at signs", "test@@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireValidEmail(tt.email, "email")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireValidEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireEnum(t *testing.T) {
	g := NewGuard()
	allowed := []string{"active", "inactive", "pending"}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid value", "active", false},
		{"another valid", "pending", false},
		{"invalid value", "unknown", true},
		{"empty value", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireEnum(tt.value, allowed, "status")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireEnum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireNotInFuture(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		t       time.Time
		wantErr bool
	}{
		{"past", time.Now().Add(-time.Hour), false},
		{"now", time.Now(), false},
		{"near future (within clock skew)", time.Now().Add(30 * time.Second), false},
		{"far future", time.Now().Add(time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireNotInFuture(tt.t, "timestamp")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireNotInFuture() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_RequireNotInPast(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name    string
		t       time.Time
		wantErr bool
	}{
		{"future", time.Now().Add(time.Hour), false},
		{"now", time.Now(), false},
		{"near past (within clock skew)", time.Now().Add(-30 * time.Second), false},
		{"far past", time.Now().Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.RequireNotInPast(tt.t, "expires_at")
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireNotInPast() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_ValidatePagination(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name     string
		limit    int
		offset   int
		maxLimit int
		wantErr  bool
	}{
		{"valid", 10, 0, 100, false},
		{"max limit", 100, 0, 100, false},
		{"negative limit", -1, 0, 100, true},
		{"over max limit", 101, 0, 100, true},
		{"negative offset", 10, -1, 100, true},
		{"zero limit", 0, 0, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.ValidatePagination(tt.limit, tt.offset, tt.maxLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePagination() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_NormalizePagination(t *testing.T) {
	g := NewGuard()

	tests := []struct {
		name        string
		limit       int
		offset      int
		defLimit    int
		maxLimit    int
		wantLimit   int
		wantOffset  int
	}{
		{"normal", 10, 5, 20, 100, 10, 5},
		{"zero limit uses default", 0, 0, 20, 100, 20, 0},
		{"negative limit uses default", -1, 0, 20, 100, 20, 0},
		{"over max uses max", 150, 0, 20, 100, 100, 0},
		{"negative offset becomes zero", 10, -5, 20, 100, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset := g.NormalizePagination(tt.limit, tt.offset, tt.defLimit, tt.maxLimit)
			if gotLimit != tt.wantLimit || gotOffset != tt.wantOffset {
				t.Errorf("NormalizePagination() = (%d, %d), want (%d, %d)",
					gotLimit, gotOffset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestValidationResult(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		v := NewValidationResult()
		if v.HasErrors() {
			t.Error("should not have errors")
		}
		if v.Error() != nil {
			t.Error("Error() should return nil")
		}
		if v.Count() != 0 {
			t.Errorf("Count() = %d, want 0", v.Count())
		}
	})

	t.Run("single error", func(t *testing.T) {
		v := NewValidationResult().
			RequireUUID(uuid.Nil, "id")

		if !v.HasErrors() {
			t.Error("should have errors")
		}
		if v.Count() != 1 {
			t.Errorf("Count() = %d, want 1", v.Count())
		}
		if v.Error() == nil {
			t.Error("Error() should not return nil")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		v := NewValidationResult().
			RequireUUID(uuid.Nil, "id").
			RequireString("", "name").
			RequirePositive(0, "count")

		if !v.HasErrors() {
			t.Error("should have errors")
		}
		if v.Count() != 3 {
			t.Errorf("Count() = %d, want 3", v.Count())
		}
		if len(v.Errors()) != 3 {
			t.Errorf("Errors() length = %d, want 3", len(v.Errors()))
		}
	})

	t.Run("Check method", func(t *testing.T) {
		v := NewValidationResult().
			Check(false, "field", "must be true").
			Check(true, "other", "will not error")

		if v.Count() != 1 {
			t.Errorf("Count() = %d, want 1", v.Count())
		}
	})

	t.Run("Add nil error", func(t *testing.T) {
		v := NewValidationResult().
			Add(nil).
			Add(nil)

		if v.HasErrors() {
			t.Error("should not have errors from nil adds")
		}
	})

	t.Run("fluent chain", func(t *testing.T) {
		v := Validate().
			RequireUUID(uuid.New(), "id").
			RequireString("hello", "name").
			RequireNonNegative(5, "count").
			RequireMaxLength("short", 100, "description")

		if v.HasErrors() {
			t.Errorf("should not have errors: %v", v.Error())
		}
	})
}

func TestGuard_ValidateCallStatus(t *testing.T) {
	g := NewGuard()

	validStatuses := []string{"pending", "in_progress", "completed", "failed", "quote_generated"}
	for _, status := range validStatuses {
		if err := g.ValidateCallStatus(status); err != nil {
			t.Errorf("ValidateCallStatus(%q) should be valid", status)
		}
	}

	if err := g.ValidateCallStatus("invalid"); err == nil {
		t.Error("ValidateCallStatus('invalid') should error")
	}
}

func TestGuard_ValidateProviderType(t *testing.T) {
	g := NewGuard()

	validProviders := []string{"bland", "vapi", "retell"}
	for _, provider := range validProviders {
		if err := g.ValidateProviderType(provider); err != nil {
			t.Errorf("ValidateProviderType(%q) should be valid", provider)
		}
	}

	if err := g.ValidateProviderType("unknown"); err == nil {
		t.Error("ValidateProviderType('unknown') should error")
	}
}

func TestGuard_ValidateSyncStatus(t *testing.T) {
	g := NewGuard()

	validStatuses := []string{"draft", "syncing", "active", "error"}
	for _, status := range validStatuses {
		if err := g.ValidateSyncStatus(status); err != nil {
			t.Errorf("ValidateSyncStatus(%q) should be valid", status)
		}
	}

	if err := g.ValidateSyncStatus("unknown"); err == nil {
		t.Error("ValidateSyncStatus('unknown') should error")
	}
}

func TestGuard_ValidateDocumentStatus(t *testing.T) {
	g := NewGuard()

	validStatuses := []string{"pending", "processing", "active", "error"}
	for _, status := range validStatuses {
		if err := g.ValidateDocumentStatus(status); err != nil {
			t.Errorf("ValidateDocumentStatus(%q) should be valid", status)
		}
	}

	if err := g.ValidateDocumentStatus("unknown"); err == nil {
		t.Error("ValidateDocumentStatus('unknown') should error")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("GuardUUID", func(t *testing.T) {
		if err := GuardUUID(uuid.New(), "id"); err != nil {
			t.Error("GuardUUID with valid uuid should not error")
		}
		if err := GuardUUID(uuid.Nil, "id"); err == nil {
			t.Error("GuardUUID with nil uuid should error")
		}
	})

	t.Run("GuardString", func(t *testing.T) {
		if err := GuardString("hello", "name"); err != nil {
			t.Error("GuardString with valid string should not error")
		}
		if err := GuardString("", "name"); err == nil {
			t.Error("GuardString with empty string should error")
		}
	})

	t.Run("GuardPagination", func(t *testing.T) {
		if err := GuardPagination(10, 0, 100); err != nil {
			t.Error("GuardPagination with valid params should not error")
		}
		if err := GuardPagination(-1, 0, 100); err == nil {
			t.Error("GuardPagination with negative limit should error")
		}
	})

	t.Run("GuardEmail", func(t *testing.T) {
		if err := GuardEmail("test@example.com", "email"); err != nil {
			t.Error("GuardEmail with valid email should not error")
		}
		if err := GuardEmail("invalid", "email"); err == nil {
			t.Error("GuardEmail with invalid email should error")
		}
	})
}

func TestValidationResult_RequireValidEmail(t *testing.T) {
	t.Run("valid email", func(t *testing.T) {
		v := Validate().RequireValidEmail("test@example.com", "email")
		if v.HasErrors() {
			t.Errorf("should not error for valid email: %v", v.Error())
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		v := Validate().RequireValidEmail("invalid", "email")
		if !v.HasErrors() {
			t.Error("should error for invalid email")
		}
	})
}

func TestValidationResult_RequireEnum(t *testing.T) {
	allowed := []string{"a", "b", "c"}

	t.Run("valid value", func(t *testing.T) {
		v := Validate().RequireEnum("a", allowed, "choice")
		if v.HasErrors() {
			t.Errorf("should not error for valid value: %v", v.Error())
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		v := Validate().RequireEnum("x", allowed, "choice")
		if !v.HasErrors() {
			t.Error("should error for invalid value")
		}
	})
}
