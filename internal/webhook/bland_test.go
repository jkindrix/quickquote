package webhook

import (
	"testing"
)

func TestBlandWebhookPayload_GetPhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneNumber string
		to          string
		expected    string
	}{
		{"uses PhoneNumber when set", "+1234567890", "+19876543210", "+1234567890"},
		{"falls back to To when PhoneNumber empty", "", "+19876543210", "+19876543210"},
		{"returns empty when both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{
				PhoneNumber: tt.phoneNumber,
				To:          tt.to,
			}
			if got := p.GetPhoneNumber(); got != tt.expected {
				t.Errorf("GetPhoneNumber() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_GetFromNumber(t *testing.T) {
	tests := []struct {
		name       string
		fromNumber string
		from       string
		expected   string
	}{
		{"uses FromNumber when set", "+1234567890", "+19876543210", "+1234567890"},
		{"falls back to From when FromNumber empty", "", "+19876543210", "+19876543210"},
		{"returns empty when both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{
				FromNumber: tt.fromNumber,
				From:       tt.from,
			}
			if got := p.GetFromNumber(); got != tt.expected {
				t.Errorf("GetFromNumber() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_GetDurationSeconds(t *testing.T) {
	tests := []struct {
		name     string
		duration float64
		expected int
	}{
		{"zero duration", 0, 0},
		{"whole number", 60.0, 60},
		{"fractional duration rounds down", 60.9, 60},
		{"small fraction", 0.5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{Duration: tt.duration}
			if got := p.GetDurationSeconds(); got != tt.expected {
				t.Errorf("GetDurationSeconds() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_GetTranscript(t *testing.T) {
	transcript := "Hello, this is a test transcript."
	p := &BlandWebhookPayload{ConcatenatedTranscript: transcript}

	if got := p.GetTranscript(); got != transcript {
		t.Errorf("GetTranscript() = %q, expected %q", got, transcript)
	}
}

func TestBlandWebhookPayload_IsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"completed status", "completed", true},
		{"success status", "success", true},
		{"in_progress status", "in_progress", false},
		{"failed status", "failed", false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{Status: tt.status}
			if got := p.IsCompleted(); got != tt.expected {
				t.Errorf("IsCompleted() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"failed status", "failed", true},
		{"error status", "error", true},
		{"completed status", "completed", false},
		{"in_progress status", "in_progress", false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{Status: tt.status}
			if got := p.IsFailed(); got != tt.expected {
				t.Errorf("IsFailed() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_IsNoAnswer(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		answeredBy string
		expected   bool
	}{
		{"no_answer status", "no_answer", "", true},
		{"no-answer status", "no-answer", "", true},
		{"voicemail answered", "completed", "voicemail", true},
		{"completed status", "completed", "", false},
		{"human answered", "completed", "human", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{Status: tt.status, AnsweredBy: tt.answeredBy}
			if got := p.IsNoAnswer(); got != tt.expected {
				t.Errorf("IsNoAnswer() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_ExtractVariable(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]interface{}
		key       string
		expected  string
	}{
		{
			name:      "string value",
			variables: map[string]interface{}{"name": "John"},
			key:       "name",
			expected:  "John",
		},
		{
			name:      "missing key",
			variables: map[string]interface{}{"name": "John"},
			key:       "email",
			expected:  "",
		},
		{
			name:      "nil variables",
			variables: nil,
			key:       "name",
			expected:  "",
		},
		{
			name:      "empty variables",
			variables: map[string]interface{}{},
			key:       "name",
			expected:  "",
		},
		{
			name:      "number value converts to JSON",
			variables: map[string]interface{}{"count": 42},
			key:       "count",
			expected:  "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BlandWebhookPayload{Variables: tt.variables}
			if got := p.ExtractVariable(tt.key); got != tt.expected {
				t.Errorf("ExtractVariable(%q) = %q, expected %q", tt.key, got, tt.expected)
			}
		})
	}
}

func TestBlandWebhookPayload_ExtractedVariables(t *testing.T) {
	p := &BlandWebhookPayload{
		Variables: map[string]interface{}{
			"project_type":       "Web Development",
			"requirements":       "Need a landing page",
			"timeline":           "2 weeks",
			"budget_range":       "$5000-$10000",
			"contact_preference": "email",
			"caller_name":        "John Doe",
		},
	}

	extracted := p.ExtractedVariables()

	if extracted.ProjectType != "Web Development" {
		t.Errorf("ProjectType = %q, expected 'Web Development'", extracted.ProjectType)
	}
	if extracted.Requirements != "Need a landing page" {
		t.Errorf("Requirements = %q, expected 'Need a landing page'", extracted.Requirements)
	}
	if extracted.Timeline != "2 weeks" {
		t.Errorf("Timeline = %q, expected '2 weeks'", extracted.Timeline)
	}
	if extracted.BudgetRange != "$5000-$10000" {
		t.Errorf("BudgetRange = %q, expected '$5000-$10000'", extracted.BudgetRange)
	}
	if extracted.ContactPreference != "email" {
		t.Errorf("ContactPreference = %q, expected 'email'", extracted.ContactPreference)
	}
	if extracted.CallerName != "John Doe" {
		t.Errorf("CallerName = %q, expected 'John Doe'", extracted.CallerName)
	}
}

func TestBlandWebhookPayload_ExtractedVariables_Empty(t *testing.T) {
	p := &BlandWebhookPayload{}

	extracted := p.ExtractedVariables()

	if extracted.ProjectType != "" {
		t.Errorf("ProjectType = %q, expected empty", extracted.ProjectType)
	}
	if extracted.CallerName != "" {
		t.Errorf("CallerName = %q, expected empty", extracted.CallerName)
	}
}
