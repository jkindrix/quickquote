package bland

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
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

func newTestProvider() *Provider {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "",
		APIURL:        "https://api.bland.ai/v1",
	}
	return New(cfg, logger)
}

func TestProvider_GetName(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetName(); got != voiceprovider.ProviderBland {
		t.Errorf("GetName() = %q, expected %q", got, voiceprovider.ProviderBland)
	}
}

func TestProvider_GetWebhookPath(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetWebhookPath(); got != "/webhook/bland" {
		t.Errorf("GetWebhookPath() = %q, expected %q", got, "/webhook/bland")
	}
}

func TestProvider_ParseWebhook_Success(t *testing.T) {
	provider := newTestProvider()

	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	payload := BlandWebhookPayload{
		CallID:                 "call-123",
		PhoneNumber:            "+1234567890",
		FromNumber:             "+19876543210",
		Status:                 "completed",
		Duration:               120,
		StartTime:              &startTime,
		EndTime:                &endTime,
		ConcatenatedTranscript: "Hello, I need a quote for a web project.",
		RecordingURL:           "https://recording.url/call.mp3",
		Variables: map[string]interface{}{
			"project_type": "web development",
			"budget_range": "$10,000-$20,000",
			"caller_name":  "John Doe",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Provider != voiceprovider.ProviderBland {
		t.Errorf("Provider = %q, expected %q", event.Provider, voiceprovider.ProviderBland)
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
	if event.Status != voiceprovider.CallStatusCompleted {
		t.Errorf("Status = %q, expected %q", event.Status, voiceprovider.CallStatusCompleted)
	}
	if event.DurationSecs != 120 {
		t.Errorf("DurationSecs = %d, expected 120", event.DurationSecs)
	}
	if event.Transcript != "Hello, I need a quote for a web project." {
		t.Errorf("Transcript = %q, expected specific value", event.Transcript)
	}
	if event.RecordingURL != "https://recording.url/call.mp3" {
		t.Errorf("RecordingURL = %q, expected specific value", event.RecordingURL)
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
	if event.CallerName != "John Doe" {
		t.Errorf("CallerName = %q, expected %q", event.CallerName, "John Doe")
	}
}

func TestProvider_ParseWebhook_MissingCallID(t *testing.T) {
	provider := newTestProvider()

	payload := BlandWebhookPayload{
		PhoneNumber: "+1234567890",
		Status:      "completed",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for missing call_id, got nil")
	}
}

func TestProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestProvider_ParseWebhook_StatusMapping(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		blandStatus    string
		answeredBy     string
		expectedStatus voiceprovider.CallStatus
	}{
		{"completed", "", voiceprovider.CallStatusCompleted},
		{"success", "", voiceprovider.CallStatusCompleted},
		{"failed", "", voiceprovider.CallStatusFailed},
		{"error", "", voiceprovider.CallStatusFailed},
		{"no_answer", "", voiceprovider.CallStatusNoAnswer},
		{"no-answer", "", voiceprovider.CallStatusNoAnswer},
		{"in_progress", "", voiceprovider.CallStatusInProgress},
		{"in-progress", "", voiceprovider.CallStatusInProgress},
		{"active", "", voiceprovider.CallStatusInProgress},
		{"voicemail", "", voiceprovider.CallStatusVoicemail},
		{"transferred", "", voiceprovider.CallStatusTransferred},
		{"", "voicemail", voiceprovider.CallStatusVoicemail},
		{"", "", voiceprovider.CallStatusPending},
		{"unknown", "", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.blandStatus+"_"+tt.answeredBy, func(t *testing.T) {
			payload := BlandWebhookPayload{
				CallID:      "call-123",
				PhoneNumber: "+1234567890",
				Status:      tt.blandStatus,
				AnsweredBy:  tt.answeredBy,
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader(body))
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

func TestProvider_ParseWebhook_WithTranscripts(t *testing.T) {
	provider := newTestProvider()

	payload := BlandWebhookPayload{
		CallID:      "call-123",
		PhoneNumber: "+1234567890",
		Status:      "completed",
		Transcripts: []TranscriptMessage{
			{Role: "assistant", Content: "Hello, how can I help you?", Timestamp: 0.5},
			{Role: "user", Content: "I need a quote for a website.", Timestamp: 2.3},
			{Role: "assistant", Content: "I'd be happy to help.", Timestamp: 4.1},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if len(event.TranscriptEntries) != 3 {
		t.Fatalf("TranscriptEntries length = %d, expected 3", len(event.TranscriptEntries))
	}

	if event.TranscriptEntries[0].Role != "assistant" {
		t.Errorf("TranscriptEntries[0].Role = %q, expected %q", event.TranscriptEntries[0].Role, "assistant")
	}
	if event.TranscriptEntries[0].Content != "Hello, how can I help you?" {
		t.Errorf("TranscriptEntries[0].Content = %q, expected specific value", event.TranscriptEntries[0].Content)
	}
	if event.TranscriptEntries[1].Role != "user" {
		t.Errorf("TranscriptEntries[1].Role = %q, expected %q", event.TranscriptEntries[1].Role, "user")
	}
}

func TestProvider_ValidateWebhook_NoSecret(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", nil)

	// Should return true when no secret is configured
	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return true when no secret is configured")
	}
}

func TestProvider_ValidateWebhook_MissingSignature(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: "test-secret",
	}
	provider := New(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader([]byte("{}")))
	// No signature header

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false when signature header is missing")
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
	payload := `{"call_id":"test-123","status":"completed"}`

	// Compute the expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Webhook-Secret", signature)

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

	payload := `{"call_id":"test-123","status":"completed"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Webhook-Secret", "invalid-signature")

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false for invalid signature")
	}
}

func TestProvider_ValidateWebhook_AlternativeHeader(t *testing.T) {
	logger := zap.NewNop()
	secret := "test-webhook-secret"
	cfg := &Config{
		APIKey:        "test-api-key",
		WebhookSecret: secret,
	}
	provider := New(cfg, logger)

	payload := `{"call_id":"test-123","status":"completed"}`

	// Compute the expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Bland-Signature", signature) // Using alternative header

	if !provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should accept X-Bland-Signature header")
	}
}

func TestProvider_New_DefaultAPIURL(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey: "test-api-key",
		APIURL: "", // Empty, should use default
	}
	provider := New(cfg, logger)

	if provider.config.APIURL != "https://api.bland.ai/v1" {
		t.Errorf("APIURL = %q, expected default value", provider.config.APIURL)
	}
}

func TestBlandWebhookPayload_GetToNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneNumber string
		to          string
		expected    string
	}{
		{"phone_number takes precedence", "+1111111111", "+2222222222", "+1111111111"},
		{"falls back to to", "", "+2222222222", "+2222222222"},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &BlandWebhookPayload{
				PhoneNumber: tt.phoneNumber,
				To:          tt.to,
			}
			if got := payload.getToNumber(); got != tt.expected {
				t.Errorf("getToNumber() = %q, expected %q", got, tt.expected)
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
		{"from_number takes precedence", "+1111111111", "+2222222222", "+1111111111"},
		{"falls back to from", "", "+2222222222", "+2222222222"},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &BlandWebhookPayload{
				FromNumber: tt.fromNumber,
				From:       tt.from,
			}
			if got := payload.getFromNumber(); got != tt.expected {
				t.Errorf("getFromNumber() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestProvider_ParseWebhook_EmptyBody(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/bland", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte{}))

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for empty body, got nil")
	}
}

func TestGetStringFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{"string value", map[string]interface{}{"key": "value"}, "key", "value"},
		{"int value", map[string]interface{}{"key": 123}, "key", "123"},
		{"missing key", map[string]interface{}{"other": "value"}, "key", ""},
		{"nil map", nil, "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringFromMap(tt.m, tt.key)
			if got != tt.expected {
				t.Errorf("getStringFromMap() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
