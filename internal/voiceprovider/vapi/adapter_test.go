package vapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

func newTestProvider() *Provider {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "",
		APIURL:        "https://api.vapi.ai",
	}
	return New(cfg, logger)
}

func TestProvider_GetName(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetName(); got != voiceprovider.ProviderVapi {
		t.Errorf("GetName() = %q, expected %q", got, voiceprovider.ProviderVapi)
	}
}

func TestProvider_GetWebhookPath(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetWebhookPath(); got != "/webhook/vapi" {
		t.Errorf("GetWebhookPath() = %q, expected %q", got, "/webhook/vapi")
	}
}

func TestProvider_New_DefaultAPIURL(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey: "test-api-key",
		APIURL: "", // Empty, should use default
	}
	provider := New(cfg, logger)

	if provider.config.APIURL != "https://api.vapi.ai" {
		t.Errorf("APIURL = %q, expected default value", provider.config.APIURL)
	}
}

func TestProvider_ParseWebhook_EndOfCallReport_Success(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "end-of-call-report",
			Call: VapiCall{
				ID:        "call-123",
				Type:      "inboundPhoneCall",
				Status:    "ended",
				StartedAt: "2024-01-15T10:00:00Z",
				EndedAt:   "2024-01-15T10:02:00Z",
				Customer: VapiCustomer{
					Number: "+19876543210",
					Name:   "John Doe",
				},
				PhoneNumber: VapiPhoneNumber{
					ID:     "phone-456",
					Number: "+1234567890",
				},
			},
			Transcript:   "Hello, I need a quote for a web project.",
			Summary:      "Customer requested quote for web project",
			RecordingURL: "https://recording.url/call.mp3",
			Messages: []VapiTranscriptMessage{
				{Role: "assistant", Content: "Hello, how can I help you?", StartTime: 0.5, EndTime: 2.0},
				{Role: "user", Content: "I need a quote for a website.", StartTime: 2.5, EndTime: 5.0},
			},
			Analysis: &VapiAnalysis{
				Summary: "Customer needs website quote",
				StructuredData: map[string]interface{}{
					"project_type": "web development",
					"budget_range": "$10,000-$20,000",
					"caller_name":  "John Doe",
				},
				SuccessScore: 0.95,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Provider != voiceprovider.ProviderVapi {
		t.Errorf("Provider = %q, expected %q", event.Provider, voiceprovider.ProviderVapi)
	}
	if event.ProviderCallID != "call-123" {
		t.Errorf("ProviderCallID = %q, expected %q", event.ProviderCallID, "call-123")
	}
	if event.ToNumber != "+19876543210" {
		t.Errorf("ToNumber = %q, expected %q", event.ToNumber, "+19876543210")
	}
	if event.FromNumber != "+1234567890" {
		t.Errorf("FromNumber = %q, expected %q", event.FromNumber, "+1234567890")
	}
	if event.Status != voiceprovider.CallStatusCompleted {
		t.Errorf("Status = %q, expected %q", event.Status, voiceprovider.CallStatusCompleted)
	}
	if event.DurationSecs != 120 {
		t.Errorf("DurationSecs = %d, expected 120", event.DurationSecs)
	}
	if event.Transcript != "Hello, I need a quote for a web project." {
		t.Errorf("Transcript = %q, expected specific value", event.Transcript)
	}
	if event.Summary != "Customer requested quote for web project" {
		t.Errorf("Summary = %q, expected specific value", event.Summary)
	}
	if event.RecordingURL != "https://recording.url/call.mp3" {
		t.Errorf("RecordingURL = %q, expected specific value", event.RecordingURL)
	}
	if len(event.TranscriptEntries) != 2 {
		t.Fatalf("TranscriptEntries length = %d, expected 2", len(event.TranscriptEntries))
	}
	if event.TranscriptEntries[0].Role != "assistant" {
		t.Errorf("TranscriptEntries[0].Role = %q, expected %q", event.TranscriptEntries[0].Role, "assistant")
	}
	if event.ExtractedData == nil {
		t.Fatal("ExtractedData is nil")
	}
	if event.ExtractedData.ProjectType != "web development" {
		t.Errorf("ProjectType = %q, expected %q", event.ExtractedData.ProjectType, "web development")
	}
	if event.ExtractedData.BudgetRange != "$10,000-$20,000" {
		t.Errorf("BudgetRange = %q, expected %q", event.ExtractedData.BudgetRange, "$10,000-$20,000")
	}
}

func TestProvider_ParseWebhook_StatusUpdate(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type:   "status-update",
			Status: "in-progress",
			Call: VapiCall{
				ID: "call-123",
				Customer: VapiCustomer{
					Number: "+19876543210",
				},
				PhoneNumber: VapiPhoneNumber{
					Number: "+1234567890",
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.ProviderCallID != "call-123" {
		t.Errorf("ProviderCallID = %q, expected %q", event.ProviderCallID, "call-123")
	}
	if event.Status != voiceprovider.CallStatusInProgress {
		t.Errorf("Status = %q, expected %q", event.Status, voiceprovider.CallStatusInProgress)
	}
}

func TestProvider_ParseWebhook_Transcript(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type:       "transcript",
			Transcript: "Current transcript text",
			Call: VapiCall{
				ID: "call-123",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Status != voiceprovider.CallStatusInProgress {
		t.Errorf("Status = %q, expected %q", event.Status, voiceprovider.CallStatusInProgress)
	}
	if event.Transcript != "Current transcript text" {
		t.Errorf("Transcript = %q, expected specific value", event.Transcript)
	}
}

func TestProvider_ParseWebhook_UnknownType(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "unknown-type",
			Call: VapiCall{
				ID: "call-123",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	// Unknown types should return in-progress status
	if event.Status != voiceprovider.CallStatusInProgress {
		t.Errorf("Status = %q, expected %q", event.Status, voiceprovider.CallStatusInProgress)
	}
}

func TestProvider_ParseWebhook_MissingCallID(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "end-of-call-report",
			Call: VapiCall{
				Status: "ended",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for missing call_id, got nil")
	}
}

func TestProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestProvider_ParseWebhook_EmptyBody(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte{}))

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for empty body, got nil")
	}
}

func TestProvider_ParseWebhook_StatusMapping(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		vapiStatus     string
		expectedStatus voiceprovider.CallStatus
	}{
		{"ended", voiceprovider.CallStatusCompleted},
		{"completed", voiceprovider.CallStatusCompleted},
		{"failed", voiceprovider.CallStatusFailed},
		{"error", voiceprovider.CallStatusFailed},
		{"no-answer", voiceprovider.CallStatusNoAnswer},
		{"in-progress", voiceprovider.CallStatusInProgress},
		{"ringing", voiceprovider.CallStatusInProgress},
		{"queued", voiceprovider.CallStatusInProgress},
		{"forwarding", voiceprovider.CallStatusTransferred},
		{"", voiceprovider.CallStatusPending},
		{"unknown", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.vapiStatus, func(t *testing.T) {
			payload := VapiWebhookPayload{
				Message: VapiMessage{
					Type:   "status-update",
					Status: tt.vapiStatus,
					Call: VapiCall{
						ID: "call-123",
					},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			event, err := provider.ParseWebhook(req)
			if err != nil {
				t.Fatalf("ParseWebhook() error = %v", err)
			}

			if event.Status != tt.expectedStatus {
				t.Errorf("Status = %q, expected %q", event.Status, tt.expectedStatus)
			}
		})
	}
}

func TestProvider_ValidateWebhook_NoSecret(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", nil)

	// Should return true when no secret is configured
	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return true when no secret is configured")
	}
}

func TestProvider_ValidateWebhook_MissingAllAuth(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "test-secret",
	}
	provider := New(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte("{}")))
	// No auth headers

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false when no auth is provided")
	}
}

func TestProvider_ValidateWebhook_ValidSignature(t *testing.T) {
	logger := zap.NewNop()
	secret := "test-webhook-secret"
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: secret,
	}
	provider := New(cfg, logger)

	// Create a payload
	payload := `{"message":{"type":"end-of-call-report","call":{"id":"test-123"}}}`

	// Compute the expected signature using HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Vapi-Signature", signature)

	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return true for valid signature")
	}

	// Verify body can still be read after validation
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read body after validation: %v", err)
	}
	if string(body) != payload {
		t.Errorf("body was not restored correctly after validation")
	}
}

func TestProvider_ValidateWebhook_InvalidSignature(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "test-secret",
	}
	provider := New(cfg, logger)

	payload := `{"message":{"type":"end-of-call-report","call":{"id":"test-123"}}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Vapi-Signature", "invalid-signature")

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false for invalid signature")
	}
}

func TestProvider_ValidateWebhook_BearerAuth(t *testing.T) {
	logger := zap.NewNop()
	secret := "test-webhook-secret"
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: secret,
	}
	provider := New(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+secret)

	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return true for valid Bearer auth")
	}
}

func TestProvider_ValidateWebhook_SecretHeader(t *testing.T) {
	logger := zap.NewNop()
	secret := "test-webhook-secret"
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: secret,
	}
	provider := New(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-Vapi-Secret", secret)

	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return true for valid X-Vapi-Secret header")
	}
}

func TestProvider_ValidateWebhook_InvalidBearerAuth(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "test-secret",
	}
	provider := New(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer wrong-secret")

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false for invalid Bearer auth")
	}
}

func TestProvider_ExtractData_WithAllFields(t *testing.T) {
	provider := newTestProvider()

	analysis := &VapiAnalysis{
		Summary: "Test summary",
		StructuredData: map[string]interface{}{
			"project_type":       "mobile app",
			"requirements":       "iOS and Android app",
			"timeline":           "3 months",
			"budget_range":       "$50,000-$100,000",
			"contact_preference": "phone",
			"caller_name":        "Jane Smith",
			"extra_field":        "extra value",
		},
		SuccessScore: 0.85,
	}

	data := provider.extractData(analysis)

	if data == nil {
		t.Fatal("extractData() returned nil")
	}
	if data.ProjectType != "mobile app" {
		t.Errorf("ProjectType = %q, expected %q", data.ProjectType, "mobile app")
	}
	if data.Requirements != "iOS and Android app" {
		t.Errorf("Requirements = %q, expected %q", data.Requirements, "iOS and Android app")
	}
	if data.Timeline != "3 months" {
		t.Errorf("Timeline = %q, expected %q", data.Timeline, "3 months")
	}
	if data.BudgetRange != "$50,000-$100,000" {
		t.Errorf("BudgetRange = %q, expected %q", data.BudgetRange, "$50,000-$100,000")
	}
	if data.ContactPreference != "phone" {
		t.Errorf("ContactPreference = %q, expected %q", data.ContactPreference, "phone")
	}
	if data.CallerName != "Jane Smith" {
		t.Errorf("CallerName = %q, expected %q", data.CallerName, "Jane Smith")
	}
	if data.Custom["extra_field"] != "extra value" {
		t.Errorf("Custom[extra_field] = %v, expected %q", data.Custom["extra_field"], "extra value")
	}
}

func TestProvider_ExtractData_NilAnalysis(t *testing.T) {
	provider := newTestProvider()

	data := provider.extractData(nil)

	if data != nil {
		t.Errorf("extractData(nil) = %v, expected nil", data)
	}
}

func TestProvider_ExtractData_EmptyAnalysis(t *testing.T) {
	provider := newTestProvider()

	analysis := &VapiAnalysis{}

	data := provider.extractData(analysis)

	if data == nil {
		t.Fatal("extractData() returned nil for empty analysis")
	}
	if data.ProjectType != "" {
		t.Errorf("ProjectType = %q, expected empty", data.ProjectType)
	}
	if data.Custom == nil {
		t.Error("Custom map should be initialized")
	}
}

func TestProvider_ParseWebhook_TimestampParsing(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "end-of-call-report",
			Call: VapiCall{
				ID:        "call-123",
				Status:    "ended",
				StartedAt: "2024-01-15T10:00:00Z",
				EndedAt:   "2024-01-15T10:01:30Z", // 90 seconds later
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.StartedAt == nil {
		t.Fatal("StartedAt is nil")
	}
	if event.EndedAt == nil {
		t.Fatal("EndedAt is nil")
	}
	if event.DurationSecs != 90 {
		t.Errorf("DurationSecs = %d, expected 90", event.DurationSecs)
	}
}

func TestProvider_ParseWebhook_InvalidTimestamps(t *testing.T) {
	provider := newTestProvider()

	payload := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "end-of-call-report",
			Call: VapiCall{
				ID:        "call-123",
				Status:    "ended",
				StartedAt: "not-a-timestamp",
				EndedAt:   "also-not-a-timestamp",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/vapi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	// Invalid timestamps should result in nil StartedAt/EndedAt
	if event.StartedAt != nil {
		t.Error("StartedAt should be nil for invalid timestamp")
	}
	if event.EndedAt != nil {
		t.Error("EndedAt should be nil for invalid timestamp")
	}
	if event.DurationSecs != 0 {
		t.Errorf("DurationSecs = %d, expected 0 for invalid timestamps", event.DurationSecs)
	}
}

func TestVapiWebhookPayload_JSONSerialization(t *testing.T) {
	original := VapiWebhookPayload{
		Message: VapiMessage{
			Type: "end-of-call-report",
			Call: VapiCall{
				ID:        "call-123",
				OrgID:     "org-456",
				Type:      "inboundPhoneCall",
				Status:    "ended",
				StartedAt: "2024-01-15T10:00:00Z",
				EndedAt:   "2024-01-15T10:02:00Z",
			},
			Transcript: "Test transcript",
			Summary:    "Test summary",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VapiWebhookPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Message.Type != original.Message.Type {
		t.Errorf("Type = %q, expected %q", decoded.Message.Type, original.Message.Type)
	}
	if decoded.Message.Call.ID != original.Message.Call.ID {
		t.Errorf("CallID = %q, expected %q", decoded.Message.Call.ID, original.Message.Call.ID)
	}
	if decoded.Message.Transcript != original.Message.Transcript {
		t.Errorf("Transcript = %q, expected %q", decoded.Message.Transcript, original.Message.Transcript)
	}
}

func TestVapiAnalysis_JSONSerialization(t *testing.T) {
	original := VapiAnalysis{
		Summary: "Test summary",
		StructuredData: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		SuccessScore: 0.92,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VapiAnalysis
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, expected %q", decoded.Summary, original.Summary)
	}
	if decoded.SuccessScore != original.SuccessScore {
		t.Errorf("SuccessScore = %v, expected %v", decoded.SuccessScore, original.SuccessScore)
	}
}

func TestVapiTranscriptMessage_JSONSerialization(t *testing.T) {
	original := VapiTranscriptMessage{
		Role:      "assistant",
		Content:   "Hello, how can I help?",
		StartTime: 0.5,
		EndTime:   2.5,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VapiTranscriptMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, expected %q", decoded.Role, original.Role)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content = %q, expected %q", decoded.Content, original.Content)
	}
	if decoded.StartTime != original.StartTime {
		t.Errorf("StartTime = %v, expected %v", decoded.StartTime, original.StartTime)
	}
}

func TestProvider_NormalizeStatus(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		status   string
		expected voiceprovider.CallStatus
	}{
		{"ended", voiceprovider.CallStatusCompleted},
		{"completed", voiceprovider.CallStatusCompleted},
		{"failed", voiceprovider.CallStatusFailed},
		{"error", voiceprovider.CallStatusFailed},
		{"no-answer", voiceprovider.CallStatusNoAnswer},
		{"in-progress", voiceprovider.CallStatusInProgress},
		{"ringing", voiceprovider.CallStatusInProgress},
		{"queued", voiceprovider.CallStatusInProgress},
		{"forwarding", voiceprovider.CallStatusTransferred},
		{"unknown", voiceprovider.CallStatusPending},
		{"", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := provider.normalizeStatus(tt.status)
			if got != tt.expected {
				t.Errorf("normalizeStatus(%q) = %q, expected %q",
					tt.status, got, tt.expected)
			}
		})
	}
}

func TestVapiCustomer_JSONSerialization(t *testing.T) {
	original := VapiCustomer{
		Number: "+1234567890",
		Name:   "John Doe",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VapiCustomer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Number != original.Number {
		t.Errorf("Number = %q, expected %q", decoded.Number, original.Number)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name = %q, expected %q", decoded.Name, original.Name)
	}
}

func TestVapiPhoneNumber_JSONSerialization(t *testing.T) {
	original := VapiPhoneNumber{
		ID:     "phone-123",
		Number: "+1234567890",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded VapiPhoneNumber
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, expected %q", decoded.ID, original.ID)
	}
	if decoded.Number != original.Number {
		t.Errorf("Number = %q, expected %q", decoded.Number, original.Number)
	}
}
