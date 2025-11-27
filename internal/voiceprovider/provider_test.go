package voiceprovider

import (
	"testing"
)

func TestCallStatus_String(t *testing.T) {
	tests := []struct {
		status   CallStatus
		expected string
	}{
		{CallStatusPending, "pending"},
		{CallStatusInProgress, "in_progress"},
		{CallStatusCompleted, "completed"},
		{CallStatusFailed, "failed"},
		{CallStatusNoAnswer, "no_answer"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := string(tt.status); got != tt.expected {
				t.Errorf("CallStatus string = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected string
	}{
		{ProviderBland, "bland"},
		{ProviderVapi, "vapi"},
		{ProviderRetell, "retell"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := string(tt.provider); got != tt.expected {
				t.Errorf("ProviderType string = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestCallEvent_HasTranscript(t *testing.T) {
	tests := []struct {
		name       string
		transcript string
		expected   bool
	}{
		{"empty transcript", "", false},
		{"whitespace only", "   ", false},
		{"valid transcript", "Hello, I need a quote", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &CallEvent{Transcript: tt.transcript}
			if got := event.HasTranscript(); got != tt.expected {
				t.Errorf("HasTranscript() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCallEvent_IsComplete(t *testing.T) {
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
			event := &CallEvent{Status: tt.status}
			if got := event.IsComplete(); got != tt.expected {
				t.Errorf("IsComplete() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestExtractedData_Fields(t *testing.T) {
	data := &ExtractedData{
		Name:           "John Doe",
		Email:          "john@example.com",
		Phone:          "+1234567890",
		Company:        "Acme Inc",
		ProjectType:    "Web Development",
		Budget:         "$10,000",
		Timeline:       "3 months",
		Requirements:   "Build a modern web application",
		AdditionalInfo: "Need mobile support",
	}

	if data.Name != "John Doe" {
		t.Errorf("Name = %q, expected %q", data.Name, "John Doe")
	}
	if data.Email != "john@example.com" {
		t.Errorf("Email = %q, expected %q", data.Email, "john@example.com")
	}
	if data.Phone != "+1234567890" {
		t.Errorf("Phone = %q, expected %q", data.Phone, "+1234567890")
	}
	if data.Company != "Acme Inc" {
		t.Errorf("Company = %q, expected %q", data.Company, "Acme Inc")
	}
	if data.ProjectType != "Web Development" {
		t.Errorf("ProjectType = %q, expected %q", data.ProjectType, "Web Development")
	}
	if data.Budget != "$10,000" {
		t.Errorf("Budget = %q, expected %q", data.Budget, "$10,000")
	}
	if data.Timeline != "3 months" {
		t.Errorf("Timeline = %q, expected %q", data.Timeline, "3 months")
	}
	if data.Requirements != "Build a modern web application" {
		t.Errorf("Requirements = %q, expected %q", data.Requirements, "Build a modern web application")
	}
	if data.AdditionalInfo != "Need mobile support" {
		t.Errorf("AdditionalInfo = %q, expected %q", data.AdditionalInfo, "Need mobile support")
	}
}

func TestTranscriptEntry_Fields(t *testing.T) {
	entry := TranscriptEntry{
		Role:    "assistant",
		Content: "Hello, how can I help you today?",
	}

	if entry.Role != "assistant" {
		t.Errorf("Role = %q, expected %q", entry.Role, "assistant")
	}
	if entry.Content != "Hello, how can I help you today?" {
		t.Errorf("Content = %q, expected %q", entry.Content, "Hello, how can I help you today?")
	}
}

func TestCallEvent_Complete(t *testing.T) {
	event := &CallEvent{
		Provider:       ProviderBland,
		ProviderCallID: "call-123",
		ToNumber:       "+1234567890",
		FromNumber:     "+19876543210",
		Status:         CallStatusCompleted,
		Transcript:     "Test transcript",
		DurationSecs:   120,
		ExtractedData: &ExtractedData{
			Name:  "Test User",
			Email: "test@example.com",
		},
		TranscriptEntries: []TranscriptEntry{
			{Role: "assistant", Content: "Hello"},
			{Role: "user", Content: "Hi there"},
		},
	}

	if event.Provider != ProviderBland {
		t.Errorf("Provider = %q, expected %q", event.Provider, ProviderBland)
	}
	if event.ProviderCallID != "call-123" {
		t.Errorf("ProviderCallID = %q, expected %q", event.ProviderCallID, "call-123")
	}
	if event.ToNumber != "+1234567890" {
		t.Errorf("ToNumber = %q, expected %q", event.ToNumber, "+1234567890")
	}
	if event.FromNumber != "+19876543210" {
		t.Errorf("FromNumber = %q, expected %q", event.FromNumber, "+19876543210")
	}
	if !event.IsComplete() {
		t.Error("expected event to be complete")
	}
	if !event.HasTranscript() {
		t.Error("expected event to have transcript")
	}
	if event.DurationSecs != 120 {
		t.Errorf("DurationSecs = %d, expected 120", event.DurationSecs)
	}
	if event.ExtractedData == nil {
		t.Error("expected ExtractedData to be set")
	}
	if len(event.TranscriptEntries) != 2 {
		t.Errorf("TranscriptEntries len = %d, expected 2", len(event.TranscriptEntries))
	}
}
