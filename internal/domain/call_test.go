package domain

import (
	"testing"
	"time"
)

func TestNewCall(t *testing.T) {
	providerCallID := "provider-123"
	provider := "bland"
	phoneNumber := "+12345678901"
	fromNumber := "+19876543210"

	call := NewCall(providerCallID, provider, phoneNumber, fromNumber)

	if call.ID.String() == "" {
		t.Error("expected call ID to be generated")
	}
	if call.ProviderCallID != providerCallID {
		t.Errorf("expected ProviderCallID %s, got %s", providerCallID, call.ProviderCallID)
	}
	if call.Provider != provider {
		t.Errorf("expected Provider %s, got %s", provider, call.Provider)
	}
	if call.PhoneNumber != phoneNumber {
		t.Errorf("expected PhoneNumber %s, got %s", phoneNumber, call.PhoneNumber)
	}
	if call.FromNumber != fromNumber {
		t.Errorf("expected FromNumber %s, got %s", fromNumber, call.FromNumber)
	}
	if call.Status != CallStatusPending {
		t.Errorf("expected status %s, got %s", CallStatusPending, call.Status)
	}
	if call.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if call.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestCall_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   CallStatus
		expected bool
	}{
		{"pending is not complete", CallStatusPending, false},
		{"in_progress is not complete", CallStatusInProgress, false},
		{"completed is complete", CallStatusCompleted, true},
		{"failed is complete", CallStatusFailed, true},
		{"no_answer is complete", CallStatusNoAnswer, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &Call{Status: tt.status}
			if got := call.IsComplete(); got != tt.expected {
				t.Errorf("IsComplete() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCall_HasQuote(t *testing.T) {
	tests := []struct {
		name     string
		quote    *string
		expected bool
	}{
		{"nil quote", nil, false},
		{"empty quote", strPtr(""), false},
		{"valid quote", strPtr("This is a quote"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &Call{QuoteSummary: tt.quote}
			if got := call.HasQuote(); got != tt.expected {
				t.Errorf("HasQuote() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCall_Duration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  *int
		expected time.Duration
	}{
		{"nil duration", nil, 0},
		{"zero duration", intPtr(0), 0},
		{"60 seconds", intPtr(60), 60 * time.Second},
		{"90 seconds", intPtr(90), 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &Call{DurationSeconds: tt.seconds}
			if got := call.Duration(); got != tt.expected {
				t.Errorf("Duration() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCall_FormattedDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  *int
		expected string
	}{
		{"nil duration", nil, "-"},
		{"zero duration", intPtr(0), "-"},
		{"30 seconds", intPtr(30), "30s"},
		{"60 seconds", intPtr(60), "1m 0s"},
		{"90 seconds", intPtr(90), "1m 30s"},
		{"125 seconds", intPtr(125), "2m 5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &Call{DurationSeconds: tt.seconds}
			if got := call.FormattedDuration(); got != tt.expected {
				t.Errorf("FormattedDuration() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

// Helper functions for creating pointers
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
