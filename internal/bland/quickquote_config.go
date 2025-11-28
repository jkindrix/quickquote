package bland

import (
	"strings"
)

// QuickQuoteConfig contains all configuration for the QuickQuote inbound experience.
// This configuration is designed to maximize call quality and user experience.
type QuickQuoteConfig struct {
	// Voice settings
	Voice         string         `json:"voice"`
	VoiceSettings *VoiceSettings `json:"voice_settings,omitempty"`

	// Conversation settings
	Language              string  `json:"language"`
	Model                 string  `json:"model"`
	Temperature           float64 `json:"temperature"`
	InterruptionThreshold int     `json:"interruption_threshold"`
	WaitForGreeting       bool    `json:"wait_for_greeting"`

	// Audio quality
	NoiseCancellation bool    `json:"noise_cancellation"`
	BackgroundTrack   *string `json:"background_track,omitempty"`

	// Call limits
	MaxDuration int  `json:"max_duration"` // minutes
	Record      bool `json:"record"`

	// Integration
	WebhookURL    string   `json:"webhook_url"`
	WebhookEvents []string `json:"webhook_events,omitempty"`

	// Knowledge and tools
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
	ToolIDs          []string `json:"tool_ids,omitempty"`

	// Customization
	BusinessName string   `json:"business_name"`
	Greeting     string   `json:"greeting,omitempty"`
	ProjectTypes []string `json:"project_types,omitempty"`
}

// DefaultQuickQuoteConfig returns the default optimized configuration.
func DefaultQuickQuoteConfig(webhookURL, businessName string) *QuickQuoteConfig {
	officeTrack := "office"
	return &QuickQuoteConfig{
		// Voice: "maya" is professional, warm, and clear
		Voice: "maya",
		VoiceSettings: &VoiceSettings{
			Stability:       0.75, // Higher stability for consistent professional tone
			SimilarityBoost: 0.80, // Natural voice quality
			Style:           0.3,  // Slight expressiveness without being over-the-top
			SpeakerBoost:    true, // Enhanced clarity
		},

		// Conversation: Balanced for quote collection
		Language:              "en-US",
		Model:                 "enhanced", // Best quality for customer interactions
		Temperature:           0.6,        // Lower for more consistent responses
		InterruptionThreshold: 100,        // 100ms - responsive but not cutting off
		WaitForGreeting:       true,       // Let caller speak first on inbound

		// Audio: Professional environment
		NoiseCancellation: true,
		BackgroundTrack:   &officeTrack, // Subtle office ambiance

		// Limits: Generous but bounded
		MaxDuration: 15,   // 15 minutes max
		Record:      true, // Always record for quality assurance

		// Integration
		WebhookURL:    webhookURL,
		WebhookEvents: []string{"call.started", "call.completed", "call.analyzed"},

		// Business
		BusinessName: businessName,
		ProjectTypes: []string{
			"web_app", "mobile_app", "api", "ecommerce", "custom_software", "integration",
		},
	}
}

// BuildInboundConfig converts QuickQuoteConfig to InboundConfig for Bland API.
func (c *QuickQuoteConfig) BuildInboundConfig() *InboundConfig {
	return &InboundConfig{
		Task:                  c.buildPrompt(),
		Voice:                 c.Voice,
		VoiceSettings:         c.VoiceSettings,
		Language:              c.Language,
		Model:                 c.Model,
		Temperature:           c.Temperature,
		FirstSentence:         c.buildGreeting(),
		WaitForGreeting:       c.WaitForGreeting,
		InterruptionThreshold: c.InterruptionThreshold,
		KnowledgeBases:        c.KnowledgeBaseIDs,
		Tools:                 c.ToolIDs,
		Record:                c.Record,
		WebhookURL:            c.WebhookURL,
		WebhookEvents:         c.WebhookEvents,
		MaxDuration:           c.MaxDuration * 60, // Convert to seconds
		AnalysisSchema:        c.buildAnalysisSchema(),
	}
}

// buildPrompt constructs the AI agent prompt optimized for project quote collection.
func (c *QuickQuoteConfig) buildPrompt() string {
	projectList := "web applications, mobile apps, APIs, e-commerce, custom software, or integrations"
	if len(c.ProjectTypes) > 0 {
		projectList = ""
		for i, t := range c.ProjectTypes {
			if i > 0 {
				if i == len(c.ProjectTypes)-1 {
					projectList += ", or "
				} else {
					projectList += ", "
				}
			}
			projectList += formatProjectType(t)
		}
	}

	return `You are a friendly and professional software project consultant at ` + c.BusinessName + `.

## Your Personality
- Warm and approachable, but efficient
- Patient when explaining technical options
- Knowledgeable about software development
- Enthusiastic about helping bring ideas to life

## Your Goals
1. Understand what type of project the caller needs (` + projectList + `)
2. Collect the essential information needed to generate an accurate quote
3. Make the caller feel confident their project will be understood
4. Schedule a follow-up if needed

## Conversation Guidelines
- Ask ONE question at a time - never overwhelm with multiple questions
- Listen actively and acknowledge what the caller says
- If they seem unsure, offer helpful suggestions
- Use natural transitions between topics
- Confirm important details by restating them
- Don't use too much technical jargon - match their level

## Information to Collect (by project type)

### Web Application
- Main purpose and functionality
- Target users (internal, customers, public)
- Key features needed (auth, payments, dashboards, etc.)
- Existing systems to integrate with
- Estimated size/complexity

### Mobile App
- What the app does and the problem it solves
- Target platforms (iOS, Android, or both)
- Key features (push, camera, location, etc.)
- Backend requirements
- Offline functionality needs

### API / Backend
- What data or functionality it provides
- What systems will consume it
- Performance requirements
- Data migration needs

### E-commerce
- What products/services will be sold
- Estimated catalog size
- Inventory management needs
- Payment and shipping requirements

### Custom Software
- Business problem to solve
- Key processes to handle
- Number of users
- Compliance requirements

### Integration / Automation
- Systems to connect
- Data that needs to flow
- Trigger mechanism
- Sync frequency

## Always Collect
- Timeline / deadlines
- Budget range (if they're comfortable sharing)
- Contact information (name, email, phone)
- Company name if applicable

## Closing the Call
1. Summarize what you've collected
2. Let them know they'll receive their personalized quote within 24-48 hours
3. Ask for their preferred contact method
4. Thank them warmly for choosing ` + c.BusinessName + `
5. Ask if they have any other questions before ending

## Important Rules
- NEVER make up pricing or timeline estimates
- If you don't know something, say a project consultant will follow up
- Always be honest about what the quote process involves
- If caller seems frustrated, offer to have a human call them back`
}

// formatProjectType converts internal project type codes to human-readable names.
func formatProjectType(pt string) string {
	switch pt {
	case "web_app":
		return "web applications"
	case "mobile_app":
		return "mobile apps"
	case "api":
		return "APIs"
	case "ecommerce":
		return "e-commerce"
	case "custom_software":
		return "custom software"
	case "integration":
		return "integrations"
	default:
		return pt
	}
}

// buildGreeting constructs the opening greeting.
func (c *QuickQuoteConfig) buildGreeting() string {
	if c.Greeting != "" {
		return c.Greeting
	}
	return "Hello! Thank you for calling " + c.BusinessName + ". I'm here to help you get a quote for your software project. What kind of project are you looking to build?"
}

// buildAnalysisSchema returns the schema for post-call data extraction.
func (c *QuickQuoteConfig) buildAnalysisSchema() map[string]interface{} {
	return map[string]interface{}{
		"project_type": map[string]interface{}{
			"type":        "string",
			"description": "The type of project requested (web_app, mobile_app, api, ecommerce, custom_software, integration)",
		},
		"customer_info": map[string]interface{}{
			"type":        "object",
			"description": "Customer contact and identification",
			"properties": map[string]interface{}{
				"name":              "Full name of the caller",
				"phone":             "Phone number if provided",
				"email":             "Email address if provided",
				"company":           "Company name if provided",
				"preferred_contact": "How they prefer to be contacted",
			},
		},
		"project_details": map[string]interface{}{
			"type":        "object",
			"description": "Details about the project",
			"properties": map[string]interface{}{
				"description":   "Main purpose or description of the project",
				"target_users":  "Who will use the software (internal, customers, public)",
				"key_features":  "Key features mentioned",
				"integrations":  "Systems to integrate with",
				"platforms":     "Target platforms for mobile (iOS, Android, both)",
				"estimated_size": "Estimated size/complexity (small, medium, large)",
			},
		},
		"timeline_budget": map[string]interface{}{
			"type":        "object",
			"description": "Timeline and budget information",
			"properties": map[string]interface{}{
				"deadline":        "When they need it completed",
				"budget_range":    "Budget range if provided",
				"ongoing_support": "Whether they need ongoing support",
			},
		},
		"call_outcome": map[string]interface{}{
			"type":        "string",
			"description": "Overall outcome: quote_requested, callback_scheduled, transferred, declined, incomplete",
		},
		"follow_up_required": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether follow-up action is needed",
		},
		"follow_up_reason": map[string]interface{}{
			"type":        "string",
			"description": "Reason for follow-up if needed",
		},
		"sentiment": map[string]interface{}{
			"type":        "string",
			"description": "Caller sentiment: positive, neutral, frustrated, undecided",
		},
		"notes": map[string]interface{}{
			"type":        "string",
			"description": "Any additional notes or special requests",
		},
	}
}

// BuildPersonaRequest creates a Bland Persona from this configuration.
func (c *QuickQuoteConfig) BuildPersonaRequest() *CreatePersonaRequest {
	officeTrack := "office"
	return &CreatePersonaRequest{
		Name:               c.BusinessName + " Project Consultant",
		Description:        "Optimized agent for software project quote collection",
		Prompt:             c.buildPrompt(),
		Voice:              c.Voice,
		Language:           c.Language,
		Model:              c.Model,
		Temperature:        c.Temperature,
		FirstSentence:      c.buildGreeting(),
		WaitForGreeting:    c.WaitForGreeting,
		InterruptThreshold: c.InterruptionThreshold,
		MaxDuration:        c.MaxDuration,
		Record:             c.Record,
		BackgroundTrack:    officeTrack,
		NoiseCancellation:  true,
		Tools:              c.ToolIDs,
		KnowledgeBaseIDs:   c.KnowledgeBaseIDs,
	}
}

// CallSettings represents the subset of config.CallSettingsConfig needed here.
// This avoids a circular import with the config package.
type CallSettings struct {
	BusinessName          string
	Voice                 string
	VoiceStability        float64
	VoiceSimilarityBoost  float64
	VoiceStyle            float64
	VoiceSpeakerBoost     bool
	Model                 string
	Language              string
	Temperature           float64
	InterruptionThreshold int
	WaitForGreeting       bool
	NoiseCancellation     bool
	BackgroundTrack       string
	MaxDurationMinutes    int
	RecordCalls           bool
	QualityPreset         string
	CustomGreeting        string
	ProjectTypes          []string
}

// NewQuickQuoteConfigFromSettings creates a QuickQuoteConfig from application settings.
func NewQuickQuoteConfigFromSettings(settings *CallSettings, webhookURL string) *QuickQuoteConfig {
	// Start with defaults
	cfg := DefaultQuickQuoteConfig(webhookURL, settings.BusinessName)

	// Apply quality preset if specified
	switch settings.QualityPreset {
	case "high_quality":
		cfg = HighQualityConfig(webhookURL, settings.BusinessName)
	case "fast_response":
		cfg = FastResponseConfig(webhookURL, settings.BusinessName)
	case "accessibility":
		cfg = AccessibilityConfig(webhookURL, settings.BusinessName)
	}

	// Override with specific settings if provided
	if settings.Voice != "" {
		cfg.Voice = settings.Voice
	}
	if settings.VoiceStability > 0 {
		if cfg.VoiceSettings == nil {
			cfg.VoiceSettings = &VoiceSettings{}
		}
		cfg.VoiceSettings.Stability = settings.VoiceStability
	}
	if settings.VoiceSimilarityBoost > 0 {
		if cfg.VoiceSettings == nil {
			cfg.VoiceSettings = &VoiceSettings{}
		}
		cfg.VoiceSettings.SimilarityBoost = settings.VoiceSimilarityBoost
	}
	if settings.VoiceStyle > 0 {
		if cfg.VoiceSettings == nil {
			cfg.VoiceSettings = &VoiceSettings{}
		}
		cfg.VoiceSettings.Style = settings.VoiceStyle
	}
	if cfg.VoiceSettings != nil {
		cfg.VoiceSettings.SpeakerBoost = settings.VoiceSpeakerBoost
	}
	if settings.Model != "" {
		cfg.Model = settings.Model
	}
	if settings.Language != "" {
		cfg.Language = settings.Language
	}
	if settings.Temperature > 0 {
		cfg.Temperature = settings.Temperature
	}
	if settings.InterruptionThreshold > 0 {
		cfg.InterruptionThreshold = settings.InterruptionThreshold
	}
	cfg.WaitForGreeting = settings.WaitForGreeting
	cfg.NoiseCancellation = settings.NoiseCancellation
	if settings.BackgroundTrack != "" && settings.BackgroundTrack != "none" {
		cfg.BackgroundTrack = &settings.BackgroundTrack
	} else if settings.BackgroundTrack == "none" {
		cfg.BackgroundTrack = nil
	}
	if settings.MaxDurationMinutes > 0 {
		cfg.MaxDuration = settings.MaxDurationMinutes
	}
	cfg.Record = settings.RecordCalls
	if settings.CustomGreeting != "" {
		cfg.Greeting = settings.CustomGreeting
	}
	if len(settings.ProjectTypes) > 0 {
		cfg.ProjectTypes = settings.ProjectTypes
	}

	return cfg
}

// ParseProjectTypes parses a comma-separated string of project types.
func ParseProjectTypes(types string) []string {
	if types == "" {
		return nil
	}
	parts := strings.Split(types, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// Quality presets for different scenarios

// HighQualityConfig returns configuration optimized for premium experience.
func HighQualityConfig(webhookURL, businessName string) *QuickQuoteConfig {
	cfg := DefaultQuickQuoteConfig(webhookURL, businessName)
	cfg.Model = "enhanced"
	cfg.Temperature = 0.5 // More consistent
	cfg.InterruptionThreshold = 80 // Slightly more responsive
	cfg.VoiceSettings = &VoiceSettings{
		Stability:       0.8,
		SimilarityBoost: 0.85,
		Style:           0.25,
		SpeakerBoost:    true,
	}
	return cfg
}

// FastResponseConfig returns configuration optimized for quick interactions.
func FastResponseConfig(webhookURL, businessName string) *QuickQuoteConfig {
	cfg := DefaultQuickQuoteConfig(webhookURL, businessName)
	cfg.Model = "base" // Faster response
	cfg.Temperature = 0.7
	cfg.InterruptionThreshold = 50 // Very responsive
	return cfg
}

// AccessibilityConfig returns configuration optimized for accessibility.
func AccessibilityConfig(webhookURL, businessName string) *QuickQuoteConfig {
	cfg := DefaultQuickQuoteConfig(webhookURL, businessName)
	cfg.Temperature = 0.4 // Very consistent
	cfg.InterruptionThreshold = 200 // More time to respond
	cfg.VoiceSettings = &VoiceSettings{
		Stability:       0.9, // Very stable
		SimilarityBoost: 0.7,
		Style:           0.1, // Minimal variation
		SpeakerBoost:    true,
	}
	return cfg
}
