package retell

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
		APIURL:        "https://api.retellai.com",
	}
	return New(cfg, logger)
}

func TestProvider_GetName(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetName(); got != voiceprovider.ProviderRetell {
		t.Errorf("GetName() = %q, expected %q", got, voiceprovider.ProviderRetell)
	}
}

func TestProvider_GetWebhookPath(t *testing.T) {
	provider := newTestProvider()

	if got := provider.GetWebhookPath(); got != "/webhook/retell" {
		t.Errorf("GetWebhookPath() = %q, expected %q", got, "/webhook/retell")
	}
}

func TestProvider_New_DefaultAPIURL(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		APIKey: "test-api-key",
		APIURL: "", // Empty, should use default
	}
	provider := New(cfg, logger)

	if provider.config.APIURL != "https://api.retellai.com" {
		t.Errorf("APIURL = %q, expected default value", provider.config.APIURL)
	}
}

func TestProvider_ParseWebhook_Success(t *testing.T) {
	provider := newTestProvider()

	startTimestamp := int64(1700000000000) // milliseconds
	endTimestamp := int64(1700000120000)   // 2 minutes later

	payload := RetellWebhookPayload{
		Event: "call_ended",
		Call: RetellCall{
			CallID:         "call-123",
			AgentID:        "agent-456",
			CallType:       "inbound",
			CallStatus:     "ended",
			FromNumber:     "+19876543210",
			ToNumber:       "+1234567890",
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
			Transcript:     "Hello, I need a quote for a web project.",
			RecordingURL:   "https://recording.url/call.mp3",
			TranscriptObject: []RetellTranscriptEntry{
				{Role: "agent", Content: "Hello, how can I help you?"},
				{Role: "user", Content: "I need a quote for a website."},
			},
			CallAnalysis: &RetellCallAnalysis{
				CallSummary:   "Customer requested quote for web project",
				CallSentiment: "positive",
				CustomAnalysisData: map[string]interface{}{
					"project_type": "web development",
					"budget_range": "$10,000-$20,000",
					"caller_name":  "John Doe",
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Provider != voiceprovider.ProviderRetell {
		t.Errorf("Provider = %q, expected %q", event.Provider, voiceprovider.ProviderRetell)
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
	if event.Summary != "Customer requested quote for web project" {
		t.Errorf("Summary = %q, expected specific value", event.Summary)
	}
	if len(event.TranscriptEntries) != 2 {
		t.Fatalf("TranscriptEntries length = %d, expected 2", len(event.TranscriptEntries))
	}
	if event.TranscriptEntries[0].Role != "agent" {
		t.Errorf("TranscriptEntries[0].Role = %q, expected %q", event.TranscriptEntries[0].Role, "agent")
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
	if event.ExtractedData.CallerName != "John Doe" {
		t.Errorf("CallerName = %q, expected %q", event.ExtractedData.CallerName, "John Doe")
	}
}

func TestProvider_ParseWebhook_MissingCallID(t *testing.T) {
	provider := newTestProvider()

	payload := RetellWebhookPayload{
		Event: "call_ended",
		Call: RetellCall{
			ToNumber:   "+1234567890",
			CallStatus: "ended",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for missing call_id, got nil")
	}
}

func TestProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestProvider_ParseWebhook_EmptyBody(t *testing.T) {
	provider := newTestProvider()

	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte{}))

	_, err := provider.ParseWebhook(req)
	if err == nil {
		t.Error("expected error for empty body, got nil")
	}
}

func TestProvider_ParseWebhook_StatusMapping(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		event          string
		callStatus     string
		expectedStatus voiceprovider.CallStatus
	}{
		{"call_started", "ongoing", voiceprovider.CallStatusInProgress},
		{"call_ended", "ended", voiceprovider.CallStatusCompleted},
		{"call_ended", "registered", voiceprovider.CallStatusCompleted},
		{"call_ended", "error", voiceprovider.CallStatusFailed},
		{"call_ended", "ongoing", voiceprovider.CallStatusInProgress},
		{"call_analyzed", "", voiceprovider.CallStatusCompleted},
		{"", "ended", voiceprovider.CallStatusCompleted},
		{"", "ongoing", voiceprovider.CallStatusInProgress},
		{"", "error", voiceprovider.CallStatusFailed},
		{"", "", voiceprovider.CallStatusPending},
		{"unknown_event", "unknown_status", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.event+"_"+tt.callStatus, func(t *testing.T) {
			payload := RetellWebhookPayload{
				Event: tt.event,
				Call: RetellCall{
					CallID:     "call-123",
					ToNumber:   "+1234567890",
					CallStatus: tt.callStatus,
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader(body))
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

	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", nil)

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

	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader([]byte("{}")))
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
	payload := `{"event":"call_ended","call":{"call_id":"test-123"}}`

	// Compute the expected signature using HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Retell-Signature", signature)

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

	payload := `{"event":"call_ended","call":{"call_id":"test-123"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-Retell-Signature", "invalid-signature")

	if provider.ValidateWebhook(req) {
		t.Error("ValidateWebhook() should return false for invalid signature")
	}
}

func TestProvider_ExtractData_WithAllFields(t *testing.T) {
	provider := newTestProvider()

	analysis := &RetellCallAnalysis{
		CallSummary:   "Test summary",
		CallSentiment: "positive",
		CustomAnalysisData: map[string]interface{}{
			"project_type":       "mobile app",
			"requirements":       "iOS and Android app",
			"timeline":           "3 months",
			"budget_range":       "$50,000-$100,000",
			"contact_preference": "phone",
			"caller_name":        "Jane Smith",
			"extra_field":        "extra value",
		},
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

	analysis := &RetellCallAnalysis{}

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

func TestProvider_ParseWebhook_WithDisconnectionReason(t *testing.T) {
	provider := newTestProvider()

	payload := RetellWebhookPayload{
		Event: "call_ended",
		Call: RetellCall{
			CallID:              "call-123",
			ToNumber:            "+1234567890",
			CallStatus:          "ended",
			DisconnectionReason: "user_hangup",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Disposition != "user_hangup" {
		t.Errorf("Disposition = %q, expected %q", event.Disposition, "user_hangup")
	}
}

func TestProvider_ParseWebhook_TimestampParsing(t *testing.T) {
	provider := newTestProvider()

	// Use specific timestamps (in milliseconds)
	startMs := int64(1700000000000) // Nov 14, 2023 22:13:20 UTC
	endMs := int64(1700000060000)   // 60 seconds later

	payload := RetellWebhookPayload{
		Event: "call_ended",
		Call: RetellCall{
			CallID:         "call-123",
			ToNumber:       "+1234567890",
			CallStatus:     "ended",
			StartTimestamp: startMs,
			EndTimestamp:   endMs,
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/retell", bytes.NewReader(body))
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
	if event.DurationSecs != 60 {
		t.Errorf("DurationSecs = %d, expected 60", event.DurationSecs)
	}
}

func TestRetellWebhookPayload_JSONSerialization(t *testing.T) {
	original := RetellWebhookPayload{
		Event: "call_ended",
		Call: RetellCall{
			CallID:     "call-123",
			AgentID:    "agent-456",
			CallType:   "inbound",
			CallStatus: "ended",
			FromNumber: "+19876543210",
			ToNumber:   "+1234567890",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RetellWebhookPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Event != original.Event {
		t.Errorf("Event = %q, expected %q", decoded.Event, original.Event)
	}
	if decoded.Call.CallID != original.Call.CallID {
		t.Errorf("CallID = %q, expected %q", decoded.Call.CallID, original.Call.CallID)
	}
	if decoded.Call.AgentID != original.Call.AgentID {
		t.Errorf("AgentID = %q, expected %q", decoded.Call.AgentID, original.Call.AgentID)
	}
}

func TestRetellCallAnalysis_JSONSerialization(t *testing.T) {
	original := RetellCallAnalysis{
		CallSummary:         "Test summary",
		CallSentiment:       "positive",
		InVoicemailDetected: true,
		UserSentiment:       "neutral",
		CustomAnalysisData: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RetellCallAnalysis
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.CallSummary != original.CallSummary {
		t.Errorf("CallSummary = %q, expected %q", decoded.CallSummary, original.CallSummary)
	}
	if decoded.InVoicemailDetected != original.InVoicemailDetected {
		t.Errorf("InVoicemailDetected = %v, expected %v", decoded.InVoicemailDetected, original.InVoicemailDetected)
	}
}

func TestRetellTranscriptEntry_JSONSerialization(t *testing.T) {
	original := RetellTranscriptEntry{
		Role:    "agent",
		Content: "Hello, how can I help?",
		Words: []RetellWord{
			{Word: "Hello", Start: 0.0, End: 0.5},
			{Word: "how", Start: 0.6, End: 0.8},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RetellTranscriptEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, expected %q", decoded.Role, original.Role)
	}
	if len(decoded.Words) != 2 {
		t.Fatalf("Words length = %d, expected 2", len(decoded.Words))
	}
	if decoded.Words[0].Word != "Hello" {
		t.Errorf("Words[0].Word = %q, expected %q", decoded.Words[0].Word, "Hello")
	}
}

func TestProvider_NormalizeStatus(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		event      string
		callStatus string
		expected   voiceprovider.CallStatus
	}{
		// Event-based status takes precedence
		{"call_started", "", voiceprovider.CallStatusInProgress},
		{"call_analyzed", "", voiceprovider.CallStatusCompleted},
		{"call_ended", "ended", voiceprovider.CallStatusCompleted},
		{"call_ended", "error", voiceprovider.CallStatusFailed},

		// Fall back to call status when no recognized event
		{"", "ended", voiceprovider.CallStatusCompleted},
		{"", "registered", voiceprovider.CallStatusCompleted},
		{"", "ongoing", voiceprovider.CallStatusInProgress},
		{"", "error", voiceprovider.CallStatusFailed},
		{"", "unknown", voiceprovider.CallStatusPending},
		{"", "", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.event+"_"+tt.callStatus, func(t *testing.T) {
			got := provider.normalizeStatus(tt.event, tt.callStatus)
			if got != tt.expected {
				t.Errorf("normalizeStatus(%q, %q) = %q, expected %q",
					tt.event, tt.callStatus, got, tt.expected)
			}
		})
	}
}

func TestProvider_NormalizeCallStatus(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		status   string
		expected voiceprovider.CallStatus
	}{
		{"ended", voiceprovider.CallStatusCompleted},
		{"registered", voiceprovider.CallStatusCompleted},
		{"error", voiceprovider.CallStatusFailed},
		{"ongoing", voiceprovider.CallStatusInProgress},
		{"unknown", voiceprovider.CallStatusPending},
		{"", voiceprovider.CallStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := provider.normalizeCallStatus(tt.status)
			if got != tt.expected {
				t.Errorf("normalizeCallStatus(%q) = %q, expected %q",
					tt.status, got, tt.expected)
			}
		})
	}
}
