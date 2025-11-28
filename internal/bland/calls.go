package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// SendCallRequest contains parameters for initiating an outbound call.
type SendCallRequest struct {
	// Required: Target phone number in E.164 format
	PhoneNumber string `json:"phone_number"`

	// Task/Prompt: Instructions for the AI agent (required if no pathway_id)
	Task string `json:"task,omitempty"`

	// PathwayID: Pre-built conversation pathway ID (alternative to task)
	PathwayID string `json:"pathway_id,omitempty"`

	// PathwayVersion: Specific pathway version (defaults to production)
	PathwayVersion *int `json:"pathway_version,omitempty"`

	// PersonaID: Pre-configured persona template
	PersonaID string `json:"persona_id,omitempty"`

	// Voice: Agent voice ID or preset name (maya, josh, etc.)
	Voice string `json:"voice,omitempty"`

	// FirstSentence: Specific opening phrase for the agent
	FirstSentence string `json:"first_sentence,omitempty"`

	// Model: "base" or "turbo"
	Model string `json:"model,omitempty"`

	// Language: Language code (en-US, es, etc.) - supports 40+ languages
	Language string `json:"language,omitempty"`

	// WaitForGreeting: If true, agent waits for recipient to speak first
	WaitForGreeting bool `json:"wait_for_greeting,omitempty"`

	// Temperature: Controls randomness (0-1), default 0.7
	Temperature *float64 `json:"temperature,omitempty"`

	// InterruptionThreshold: Milliseconds before responding (50-500 recommended)
	InterruptionThreshold *int `json:"interruption_threshold,omitempty"`

	// BlockInterruptions: Prevents AI from processing user interruptions
	BlockInterruptions bool `json:"block_interruptions,omitempty"`

	// From: Caller ID number (must be owned by you)
	From string `json:"from,omitempty"`

	// MaxDuration: Maximum call length in minutes (default 30)
	MaxDuration *int `json:"max_duration,omitempty"`

	// TransferPhoneNumber: Number to transfer to under specified conditions
	TransferPhoneNumber string `json:"transfer_phone_number,omitempty"`

	// TransferList: Multiple transfer targets by department
	TransferList map[string]string `json:"transfer_list,omitempty"`

	// Voicemail: Voicemail handling configuration
	Voicemail *VoicemailConfig `json:"voicemail,omitempty"`

	// Record: Enable call recording
	Record bool `json:"record,omitempty"`

	// BackgroundTrack: Ambient audio (null, "office", "cafe", "restaurant", "none")
	BackgroundTrack *string `json:"background_track,omitempty"`

	// NoiseCancellation: Filter background noise
	NoiseCancellation bool `json:"noise_cancellation,omitempty"`

	// Tools: Custom tool and knowledge base IDs
	Tools []string `json:"tools,omitempty"`

	// DynamicData: External API endpoints for live data
	DynamicData []DynamicDataConfig `json:"dynamic_data,omitempty"`

	// Webhook: HTTPS URL for call completion data
	Webhook string `json:"webhook,omitempty"`

	// WebhookEvents: Event types to stream
	WebhookEvents []string `json:"webhook_events,omitempty"`

	// RequestData: Custom variables accessible in prompts as {{variable}}
	RequestData map[string]interface{} `json:"request_data,omitempty"`

	// Metadata: Custom tracking data returned in webhooks
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// PronunciationGuide: Words with custom pronunciations
	PronunciationGuide []PronunciationEntry `json:"pronunciation_guide,omitempty"`

	// Keywords: Words boosted in transcription (supports boost factors)
	Keywords []string `json:"keywords,omitempty"`

	// SummaryPrompt: Custom instructions for post-call summary (max 2000 chars)
	SummaryPrompt string `json:"summary_prompt,omitempty"`

	// Dispositions: Custom outcome tags for call classification
	Dispositions []string `json:"dispositions,omitempty"`

	// CitationSchemaIDs: Schema UUIDs for variable extraction
	CitationSchemaIDs []string `json:"citation_schema_ids,omitempty"`

	// StartTime: Schedule call for future time (YYYY-MM-DD HH:MM:SS -HH:MM)
	StartTime string `json:"start_time,omitempty"`

	// Timezone: TZ identifier (default "America/Los_Angeles")
	Timezone string `json:"timezone,omitempty"`

	// MemoryID: Memory store ID for context persistence
	MemoryID string `json:"memory_id,omitempty"`

	// Retry: Retry configuration for failed calls
	Retry *RetryConfig `json:"retry,omitempty"`

	// IgnoreButtonPress: Disable DTMF keypad input
	IgnoreButtonPress bool `json:"ignore_button_press,omitempty"`

	// PrecallDTMFSequence: DTMF digits sent before call starts
	PrecallDTMFSequence string `json:"precall_dtmf_sequence,omitempty"`
}

// VoicemailConfig configures voicemail handling.
type VoicemailConfig struct {
	// Action: "hangup", "leave_message", or "ignore"
	Action string `json:"action,omitempty"`

	// Message: Message to leave if voicemail detected
	Message string `json:"message,omitempty"`

	// SMS: Optional SMS notification config
	SMS *SMSNotificationConfig `json:"sms,omitempty"`

	// Sensitive: AI-based frequent voicemail detection
	Sensitive bool `json:"sensitive,omitempty"`
}

// SMSNotificationConfig configures SMS notifications.
type SMSNotificationConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
}

// DynamicDataConfig configures external API integration during calls.
type DynamicDataConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"` // GET or POST
	Headers        map[string]string `json:"headers,omitempty"`
	Body           interface{}       `json:"body,omitempty"`
	Cache          bool              `json:"cache,omitempty"`
	ResponsePath   string            `json:"response_path,omitempty"`
	ResponseFormat string            `json:"response_format,omitempty"`
}

// PronunciationEntry defines custom pronunciation for a word.
type PronunciationEntry struct {
	Word          string `json:"word"`
	Pronunciation string `json:"pronunciation"`
}

// RetryConfig configures call retry behavior.
type RetryConfig struct {
	Wait            int    `json:"wait,omitempty"`             // Seconds to wait before retry
	VoicemailAction string `json:"voicemail_action,omitempty"` // Action on voicemail
	VoicemailMessage string `json:"voicemail_message,omitempty"`
}

// SendCallResponse is returned when a call is successfully initiated.
type SendCallResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	CallID  string `json:"call_id,omitempty"`
	BatchID string `json:"batch_id,omitempty"`
}

// CallDetails contains detailed information about a call.
type CallDetails struct {
	CallID               string                 `json:"call_id"`
	Status               string                 `json:"status"`
	CreatedAt            time.Time              `json:"created_at,omitempty"`
	StartedAt            *time.Time             `json:"started_at,omitempty"`
	EndedAt              *time.Time             `json:"ended_at,omitempty"`
	Duration             float64                `json:"call_length,omitempty"`
	ToNumber             string                 `json:"to,omitempty"`
	FromNumber           string                 `json:"from,omitempty"`
	Completed            bool                   `json:"completed"`
	QueueStatus          string                 `json:"queue_status,omitempty"`
	Endpoint             string                 `json:"endpoint_url,omitempty"`
	MaxDuration          int                    `json:"max_duration,omitempty"`
	ErrorMessage         string                 `json:"error_message,omitempty"`
	Variables            map[string]interface{} `json:"variables,omitempty"`
	AnsweredBy           string                 `json:"answered_by,omitempty"`
	RecordingURL         string                 `json:"recording_url,omitempty"`
	ConcatenatedTranscript string               `json:"concatenated_transcript,omitempty"`
	Transcripts          []TranscriptMessage    `json:"transcripts,omitempty"`
	Summary              string                 `json:"summary,omitempty"`
	Price                float64                `json:"price,omitempty"`
	LocalDialingEnabled  bool                   `json:"local_dialing,omitempty"`
	BatchID              string                 `json:"batch_id,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	PathwayLogs          []PathwayLog           `json:"pathway_logs,omitempty"`
	Analysis             *CallAnalysis          `json:"analysis,omitempty"`
}

// TranscriptMessage represents a single message in the conversation.
type TranscriptMessage struct {
	ID        int       `json:"id,omitempty"`
	Role      string    `json:"role"`      // "assistant", "user"
	Content   string    `json:"text"`      // The spoken text
	Timestamp float64   `json:"created_at,omitempty"`
}

// PathwayLog represents a pathway node transition during a call.
type PathwayLog struct {
	NodeID    string    `json:"node_id"`
	NodeName  string    `json:"node_name,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// CallAnalysis contains post-call analysis data.
type CallAnalysis struct {
	Summary      string                 `json:"summary,omitempty"`
	Sentiment    string                 `json:"sentiment,omitempty"`
	Disposition  string                 `json:"disposition,omitempty"`
	ExtractedData map[string]interface{} `json:"extracted_data,omitempty"`
}

// TranscriptResponse contains the transcript for a call.
type TranscriptResponse struct {
	Transcript           string              `json:"concatenated_transcript,omitempty"`
	Transcripts          []TranscriptMessage `json:"transcripts,omitempty"`
}

// AnalyzeCallRequest contains parameters for analyzing a completed call.
type AnalyzeCallRequest struct {
	Goal      string   `json:"goal,omitempty"`
	Questions []string `json:"questions,omitempty"`
}

// AnalyzeCallResponse contains the analysis results.
type AnalyzeCallResponse struct {
	Status         string                 `json:"status"`
	CorrectedTranscript string            `json:"corrected_transcript,omitempty"`
	Answers        []AnalysisAnswer       `json:"answers,omitempty"`
	ExtractedData  map[string]interface{} `json:"extracted_data,omitempty"`
}

// AnalysisAnswer contains an answer to an analysis question.
type AnalysisAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// ActiveCallsResponse contains a list of active calls.
type ActiveCallsResponse struct {
	Calls []CallDetails `json:"calls"`
	Count int           `json:"count"`
}

// SendCall initiates an outbound call.
func (c *Client) SendCall(ctx context.Context, req *SendCallRequest) (*SendCallResponse, error) {
	if req.PhoneNumber == "" {
		return nil, fmt.Errorf("phone_number is required")
	}
	if req.Task == "" && req.PathwayID == "" && req.PersonaID == "" {
		return nil, fmt.Errorf("one of task, pathway_id, or persona_id is required")
	}

	var resp SendCallResponse
	if err := c.request(ctx, "POST", "/calls", req, &resp); err != nil {
		return nil, err
	}

	c.logger.Info("call initiated",
		zap.String("call_id", resp.CallID),
		zap.String("phone_number", req.PhoneNumber),
	)

	return &resp, nil
}

// GetCall retrieves details for a specific call.
func (c *Client) GetCall(ctx context.Context, callID string) (*CallDetails, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var resp CallDetails
	if err := c.request(ctx, "GET", "/calls/"+callID, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetCallTranscript retrieves the transcript for a completed call.
func (c *Client) GetCallTranscript(ctx context.Context, callID string) (*TranscriptResponse, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var resp TranscriptResponse
	if err := c.request(ctx, "GET", "/calls/"+callID+"/transcript", nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetCallRecording retrieves the recording URL for a call.
func (c *Client) GetCallRecording(ctx context.Context, callID string) (string, error) {
	call, err := c.GetCall(ctx, callID)
	if err != nil {
		return "", err
	}
	return call.RecordingURL, nil
}

// EndCall terminates an active call.
func (c *Client) EndCall(ctx context.Context, callID string) error {
	if callID == "" {
		return fmt.Errorf("call_id is required")
	}

	if err := c.request(ctx, "POST", "/calls/"+callID+"/stop", nil, nil); err != nil {
		return err
	}

	c.logger.Info("call ended", zap.String("call_id", callID))
	return nil
}

// AnalyzeCall performs post-call analysis to extract structured data.
func (c *Client) AnalyzeCall(ctx context.Context, callID string, req *AnalyzeCallRequest) (*AnalyzeCallResponse, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var resp AnalyzeCallResponse
	if err := c.request(ctx, "POST", "/calls/"+callID+"/analyze", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetActiveCalls retrieves all currently active calls.
func (c *Client) GetActiveCalls(ctx context.Context) (*ActiveCallsResponse, error) {
	var resp ActiveCallsResponse
	if err := c.request(ctx, "GET", "/calls/active", nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListCalls retrieves a paginated list of calls with optional filters.
func (c *Client) ListCalls(ctx context.Context, limit, offset int) ([]CallDetails, error) {
	path := fmt.Sprintf("/calls?limit=%d&offset=%d", limit, offset)

	var resp struct {
		Calls []CallDetails `json:"calls"`
	}
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Calls, nil
}
