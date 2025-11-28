package domain

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Setting represents a single configuration setting.
type Setting struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	ValueType   string    `json:"value_type" db:"value_type"`
	Category    string    `json:"category" db:"category"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Setting categories
const (
	SettingCategoryBusiness = "business"
	SettingCategoryVoice    = "voice"
	SettingCategoryAI       = "ai"
	SettingCategoryCall     = "call"
)

// Setting value types
const (
	SettingTypeString = "string"
	SettingTypeInt    = "int"
	SettingTypeFloat  = "float"
	SettingTypeBool   = "bool"
	SettingTypeJSON   = "json"
)

// Setting keys (defined as constants to avoid typos)
const (
	SettingKeyBusinessName        = "business_name"
	SettingKeyProjectTypes        = "project_types"
	SettingKeyVoice               = "voice"
	SettingKeyVoiceStability      = "voice_stability"
	SettingKeyVoiceSimilarity     = "voice_similarity_boost"
	SettingKeyVoiceStyle          = "voice_style"
	SettingKeyVoiceSpeakerBoost   = "voice_speaker_boost"
	SettingKeyModel               = "model"
	SettingKeyLanguage            = "language"
	SettingKeyTemperature         = "temperature"
	SettingKeyInterruptThreshold  = "interruption_threshold"
	SettingKeyWaitForGreeting     = "wait_for_greeting"
	SettingKeyNoiseCancellation   = "noise_cancellation"
	SettingKeyBackgroundTrack     = "background_track"
	SettingKeyMaxDuration         = "max_duration_minutes"
	SettingKeyRecordCalls         = "record_calls"
	SettingKeyQualityPreset       = "quality_preset"
	SettingKeyCustomGreeting      = "custom_greeting"

	// Pricing keys (fallback values when API unavailable)
	SettingKeyPricingInboundPerMin      = "pricing_inbound_per_minute"
	SettingKeyPricingOutboundPerMin     = "pricing_outbound_per_minute"
	SettingKeyPricingTranscriptionPerMin = "pricing_transcription_per_minute"
	SettingKeyPricingAnalysisPerCall    = "pricing_analysis_per_call"
	SettingKeyPricingPhoneNumberPerMonth = "pricing_phone_number_per_month"
	SettingKeyPricingEnhancedModelPremium = "pricing_enhanced_model_premium"
)

// SettingsRepository defines the interface for settings persistence.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetByCategory(ctx context.Context, category string) ([]*Setting, error)
	GetAll(ctx context.Context) ([]*Setting, error)
	Set(ctx context.Context, key, value string) error
	SetMany(ctx context.Context, settings map[string]string) error
	Delete(ctx context.Context, key string) error
}

// CallSettings holds all call-related settings as typed values.
// This is populated from the database settings.
type CallSettings struct {
	BusinessName          string
	ProjectTypes          []string
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
}

// NewCallSettingsFromMap creates CallSettings from a map of setting key -> value.
func NewCallSettingsFromMap(settings map[string]string) *CallSettings {
	cs := &CallSettings{
		// Defaults in case settings are missing
		BusinessName:          "QuickQuote",
		Voice:                 "maya",
		VoiceStability:        0.75,
		VoiceSimilarityBoost:  0.80,
		VoiceStyle:            0.3,
		VoiceSpeakerBoost:     true,
		Model:                 "enhanced",
		Language:              "en-US",
		Temperature:           0.6,
		InterruptionThreshold: 100,
		WaitForGreeting:       true,
		NoiseCancellation:     true,
		BackgroundTrack:       "office",
		MaxDurationMinutes:    15,
		RecordCalls:           true,
		QualityPreset:         "default",
	}

	// Override with actual values from map
	if v, ok := settings[SettingKeyBusinessName]; ok && v != "" {
		cs.BusinessName = v
	}
	if v, ok := settings[SettingKeyProjectTypes]; ok && v != "" {
		cs.ProjectTypes = parseStringList(v)
	}
	if v, ok := settings[SettingKeyVoice]; ok && v != "" {
		cs.Voice = v
	}
	if v, ok := settings[SettingKeyVoiceStability]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cs.VoiceStability = f
		}
	}
	if v, ok := settings[SettingKeyVoiceSimilarity]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cs.VoiceSimilarityBoost = f
		}
	}
	if v, ok := settings[SettingKeyVoiceStyle]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cs.VoiceStyle = f
		}
	}
	if v, ok := settings[SettingKeyVoiceSpeakerBoost]; ok {
		cs.VoiceSpeakerBoost = parseBool(v)
	}
	if v, ok := settings[SettingKeyModel]; ok && v != "" {
		cs.Model = v
	}
	if v, ok := settings[SettingKeyLanguage]; ok && v != "" {
		cs.Language = v
	}
	if v, ok := settings[SettingKeyTemperature]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cs.Temperature = f
		}
	}
	if v, ok := settings[SettingKeyInterruptThreshold]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			cs.InterruptionThreshold = i
		}
	}
	if v, ok := settings[SettingKeyWaitForGreeting]; ok {
		cs.WaitForGreeting = parseBool(v)
	}
	if v, ok := settings[SettingKeyNoiseCancellation]; ok {
		cs.NoiseCancellation = parseBool(v)
	}
	if v, ok := settings[SettingKeyBackgroundTrack]; ok && v != "" {
		cs.BackgroundTrack = v
	}
	if v, ok := settings[SettingKeyMaxDuration]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			cs.MaxDurationMinutes = i
		}
	}
	if v, ok := settings[SettingKeyRecordCalls]; ok {
		cs.RecordCalls = parseBool(v)
	}
	if v, ok := settings[SettingKeyQualityPreset]; ok && v != "" {
		cs.QualityPreset = v
	}
	if v, ok := settings[SettingKeyCustomGreeting]; ok {
		cs.CustomGreeting = v
	}

	return cs
}

// ToMap converts CallSettings back to a map for saving.
func (cs *CallSettings) ToMap() map[string]string {
	return map[string]string{
		SettingKeyBusinessName:       cs.BusinessName,
		SettingKeyProjectTypes:       strings.Join(cs.ProjectTypes, ","),
		SettingKeyVoice:              cs.Voice,
		SettingKeyVoiceStability:     strconv.FormatFloat(cs.VoiceStability, 'f', 2, 64),
		SettingKeyVoiceSimilarity:    strconv.FormatFloat(cs.VoiceSimilarityBoost, 'f', 2, 64),
		SettingKeyVoiceStyle:         strconv.FormatFloat(cs.VoiceStyle, 'f', 2, 64),
		SettingKeyVoiceSpeakerBoost:  strconv.FormatBool(cs.VoiceSpeakerBoost),
		SettingKeyModel:              cs.Model,
		SettingKeyLanguage:           cs.Language,
		SettingKeyTemperature:        strconv.FormatFloat(cs.Temperature, 'f', 2, 64),
		SettingKeyInterruptThreshold: strconv.Itoa(cs.InterruptionThreshold),
		SettingKeyWaitForGreeting:    strconv.FormatBool(cs.WaitForGreeting),
		SettingKeyNoiseCancellation:  strconv.FormatBool(cs.NoiseCancellation),
		SettingKeyBackgroundTrack:    cs.BackgroundTrack,
		SettingKeyMaxDuration:        strconv.Itoa(cs.MaxDurationMinutes),
		SettingKeyRecordCalls:        strconv.FormatBool(cs.RecordCalls),
		SettingKeyQualityPreset:      cs.QualityPreset,
		SettingKeyCustomGreeting:     cs.CustomGreeting,
	}
}

func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// PricingSettings holds pricing fallback values.
type PricingSettings struct {
	InboundPerMinute       float64
	OutboundPerMinute      float64
	TranscriptionPerMinute float64
	AnalysisPerCall        float64
	PhoneNumberPerMonth    float64
	EnhancedModelPremium   float64
}

// NewPricingSettingsFromMap creates PricingSettings from a settings map.
func NewPricingSettingsFromMap(settings map[string]string) *PricingSettings {
	ps := &PricingSettings{
		// Defaults matching Bland's typical pricing
		InboundPerMinute:       0.09,
		OutboundPerMinute:      0.12,
		TranscriptionPerMinute: 0.02,
		AnalysisPerCall:        0.05,
		PhoneNumberPerMonth:    2.00,
		EnhancedModelPremium:   0.02,
	}

	if v, ok := settings[SettingKeyPricingInboundPerMin]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.InboundPerMinute = f
		}
	}
	if v, ok := settings[SettingKeyPricingOutboundPerMin]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.OutboundPerMinute = f
		}
	}
	if v, ok := settings[SettingKeyPricingTranscriptionPerMin]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.TranscriptionPerMinute = f
		}
	}
	if v, ok := settings[SettingKeyPricingAnalysisPerCall]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.AnalysisPerCall = f
		}
	}
	if v, ok := settings[SettingKeyPricingPhoneNumberPerMonth]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.PhoneNumberPerMonth = f
		}
	}
	if v, ok := settings[SettingKeyPricingEnhancedModelPremium]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ps.EnhancedModelPremium = f
		}
	}

	return ps
}
