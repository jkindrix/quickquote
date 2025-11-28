package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/service"
)

// SettingsData holds data for the settings page.
type SettingsData struct {
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
	ProjectTypes          string
}

// UsageData holds data for the usage page.
type UsageData struct {
	TotalCalls          int
	TotalMinutes        float64
	TotalCost           float64
	CostLimit           float64
	MinuteLimit         float64
	DailyCallLimit      int
	AvgDuration         float64
	InboundCalls        int
	InboundCost         float64
	OutboundCalls       int
	OutboundCost        float64
	TranscriptionMinutes float64
	TranscriptionCost   float64
	AnalysisCount       int
	AnalysisCost        float64
	PhoneNumberCount    int
	PhoneNumberCost     float64
}

// DailyUsageData holds daily usage stats.
type DailyUsageData struct {
	Date    string
	Calls   int
	Minutes float64
	Cost    float64
}

// PricingData holds pricing information.
type PricingData struct {
	InboundPerMinute      float64
	OutboundPerMinute     float64
	TranscriptionPerMinute float64
	AnalysisPerCall       float64
	PhoneNumberPerMonth   float64
	EnhancedModelPremium  float64
}

// VoiceSettingsData holds voice settings.
type VoiceSettingsData struct {
	Stability       float64
	SimilarityBoost float64
	Style           float64
	SpeakerBoost    bool
}

// HandleSettingsPage serves the settings page.
func (h *Handler) HandleSettingsPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get current settings from database
	var settings *SettingsData
	if h.settingsService != nil {
		callSettings, err := h.settingsService.GetCallSettings(ctx)
		if err != nil {
			h.logger.Error("failed to load settings", zap.Error(err))
		} else {
			settings = callSettingsToSettingsData(callSettings)
		}
	}
	if settings == nil {
		settings = defaultSettingsData()
	}

	h.renderTemplate(w, r, "settings", map[string]interface{}{
		"Title":     "Settings",
		"ActiveNav": "settings",
		"User":      user,
		"Settings":  settings,
	})
}

// HandleSettingsUpdate handles POST to update settings.
func (h *Handler) HandleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := r.ParseForm(); err != nil {
		h.renderTemplate(w, r, "settings", map[string]interface{}{
			"Title":     "Settings",
			"ActiveNav": "settings",
			"User":      user,
			"Error":     "Failed to parse form",
			"Settings":  defaultSettingsData(),
		})
		return
	}

	// Parse settings from form
	settings := &SettingsData{
		BusinessName:          r.FormValue("business_name"),
		Voice:                 r.FormValue("voice"),
		Model:                 r.FormValue("model"),
		Language:              r.FormValue("language"),
		QualityPreset:         r.FormValue("quality_preset"),
		BackgroundTrack:       r.FormValue("background_track"),
		CustomGreeting:        r.FormValue("custom_greeting"),
		ProjectTypes:          r.FormValue("project_types"),
		WaitForGreeting:       r.FormValue("wait_for_greeting") == "on",
		NoiseCancellation:     r.FormValue("noise_cancellation") == "on",
		RecordCalls:           r.FormValue("record_calls") == "on",
		VoiceSpeakerBoost:     r.FormValue("voice_speaker_boost") == "on",
	}

	// Parse numeric values
	if v, err := strconv.ParseFloat(r.FormValue("voice_stability"), 64); err == nil {
		settings.VoiceStability = v / 100 // Convert from percentage
	}
	if v, err := strconv.ParseFloat(r.FormValue("voice_similarity_boost"), 64); err == nil {
		settings.VoiceSimilarityBoost = v / 100
	}
	if v, err := strconv.ParseFloat(r.FormValue("temperature"), 64); err == nil {
		settings.Temperature = v / 100
	}
	if v, err := strconv.Atoi(r.FormValue("interruption_threshold")); err == nil {
		settings.InterruptionThreshold = v
	}
	if v, err := strconv.Atoi(r.FormValue("max_duration")); err == nil {
		settings.MaxDurationMinutes = v
	}

	// Persist settings to database
	if h.settingsService != nil {
		callSettings := settingsDataToCallSettings(settings)
		if err := h.settingsService.SaveCallSettings(ctx, callSettings); err != nil {
			h.logger.Error("failed to save settings", zap.Error(err))
			h.renderTemplate(w, r, "settings", map[string]interface{}{
				"Title":     "Settings",
				"ActiveNav": "settings",
				"User":      user,
				"Error":     "Failed to save settings",
				"Settings":  settings,
			})
			return
		}
	}

	h.logger.Info("settings updated",
		zap.String("business_name", settings.BusinessName),
		zap.String("voice", settings.Voice),
		zap.String("model", settings.Model),
	)

	h.renderTemplate(w, r, "settings", map[string]interface{}{
		"Title":     "Settings",
		"ActiveNav": "settings",
		"User":      user,
		"Settings":  settings,
		"Success":   true,
	})
}

// HandlePhoneNumbersPage serves the phone numbers management page.
func (h *Handler) HandlePhoneNumbersPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var phoneNumbers []bland.PhoneNumber
	var blockedNumbers []bland.BlockedNumber
	var errMsg string

	// Fetch phone numbers from Bland API
	if h.blandService != nil {
		var err error
		phoneNumbers, err = h.blandService.ListPhoneNumbers(r.Context(), &bland.ListPhoneNumbersRequest{})
		if err != nil {
			h.logger.Error("failed to list phone numbers", zap.Error(err))
			errMsg = "Failed to load phone numbers"
		}

		blockedNumbers, err = h.blandService.ListBlockedNumbers(r.Context())
		if err != nil {
			h.logger.Error("failed to list blocked numbers", zap.Error(err))
			if errMsg == "" {
				errMsg = "Failed to load blocked numbers"
			}
		}
	}

	h.renderTemplate(w, r, "phone_numbers", map[string]interface{}{
		"Title":          "Phone Numbers",
		"ActiveNav":      "phone-numbers",
		"User":           user,
		"PhoneNumbers":   phoneNumbers,
		"BlockedNumbers": blockedNumbers,
		"Error":          errMsg,
	})
}

// HandleVoicesPage serves the voices management page.
func (h *Handler) HandleVoicesPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var voices []bland.Voice
	var errMsg string
	currentVoice := "maya"
	voiceSettings := VoiceSettingsData{
		Stability:       0.75,
		SimilarityBoost: 0.80,
		Style:           0.3,
		SpeakerBoost:    true,
	}

	// Load voice settings from database
	if h.settingsService != nil {
		callSettings, err := h.settingsService.GetCallSettings(ctx)
		if err != nil {
			h.logger.Error("failed to load voice settings", zap.Error(err))
		} else {
			currentVoice = callSettings.Voice
			voiceSettings.Stability = callSettings.VoiceStability
			voiceSettings.SimilarityBoost = callSettings.VoiceSimilarityBoost
			voiceSettings.Style = callSettings.VoiceStyle
			voiceSettings.SpeakerBoost = callSettings.VoiceSpeakerBoost
		}
	}

	// Fetch voices from Bland API
	if h.blandService != nil {
		var err error
		voices, err = h.blandService.ListVoices(ctx)
		if err != nil {
			h.logger.Error("failed to list voices", zap.Error(err))
			errMsg = "Failed to load voices"
		}
	}

	// Check for success query param
	success := r.URL.Query().Get("success") == "1"

	h.renderTemplate(w, r, "voices", map[string]interface{}{
		"Title":         "Voices",
		"ActiveNav":     "voices",
		"User":          user,
		"Voices":        voices,
		"CurrentVoice":  currentVoice,
		"VoiceSettings": voiceSettings,
		"Error":         errMsg,
		"Success":       success,
	})
}

// HandleVoiceSelect handles POST to select a voice.
func (h *Handler) HandleVoiceSelect(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	voiceID := r.FormValue("voice_id")
	h.logger.Info("voice selected", zap.String("voice_id", voiceID))

	// Persist to database
	if h.settingsService != nil && voiceID != "" {
		if err := h.settingsService.Set(ctx, domain.SettingKeyVoice, voiceID); err != nil {
			h.logger.Error("failed to save voice selection", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/voices?success=1", http.StatusSeeOther)
}

// HandleVoiceSettingsUpdate handles POST to update voice settings.
func (h *Handler) HandleVoiceSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Parse voice settings
	stability, _ := strconv.ParseFloat(r.FormValue("stability"), 64)
	similarityBoost, _ := strconv.ParseFloat(r.FormValue("similarity_boost"), 64)
	style, _ := strconv.ParseFloat(r.FormValue("style"), 64)
	speakerBoost := r.FormValue("speaker_boost") == "on"

	// Convert from percentage
	stability = stability / 100
	similarityBoost = similarityBoost / 100
	style = style / 100

	h.logger.Info("voice settings updated",
		zap.Float64("stability", stability),
		zap.Float64("similarity_boost", similarityBoost),
		zap.Float64("style", style),
		zap.Bool("speaker_boost", speakerBoost),
	)

	// Persist to database
	if h.settingsService != nil {
		settings := map[string]string{
			domain.SettingKeyVoiceStability:    strconv.FormatFloat(stability, 'f', 2, 64),
			domain.SettingKeyVoiceSimilarity:   strconv.FormatFloat(similarityBoost, 'f', 2, 64),
			domain.SettingKeyVoiceStyle:        strconv.FormatFloat(style, 'f', 2, 64),
			domain.SettingKeyVoiceSpeakerBoost: strconv.FormatBool(speakerBoost),
		}
		// Save each setting
		for key, value := range settings {
			if err := h.settingsService.Set(ctx, key, value); err != nil {
				h.logger.Error("failed to save voice setting", zap.String("key", key), zap.Error(err))
			}
		}
	}

	http.Redirect(w, r, "/voices?success=1", http.StatusSeeOther)
}

// HandleUsagePage serves the usage dashboard page.
func (h *Handler) HandleUsagePage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var errMsg string

	// Default usage data
	usage := UsageData{
		TotalCalls:       0,
		TotalMinutes:     0,
		TotalCost:        0,
		CostLimit:        100.00,
		MinuteLimit:      1000,
		DailyCallLimit:   100,
		AvgDuration:      0,
		InboundCalls:     0,
		InboundCost:      0,
		OutboundCalls:    0,
		OutboundCost:     0,
		PhoneNumberCount: 0,
		PhoneNumberCost:  0,
	}

	// Load pricing fallbacks from settings
	pricing := PricingData{
		InboundPerMinute:       0.09,
		OutboundPerMinute:      0.12,
		TranscriptionPerMinute: 0.02,
		AnalysisPerCall:        0.05,
		PhoneNumberPerMonth:    2.00,
		EnhancedModelPremium:   0.02,
	}
	if h.settingsService != nil {
		pricingSettings, err := h.settingsService.GetPricingSettings(ctx)
		if err != nil {
			h.logger.Warn("failed to load pricing settings", zap.Error(err))
		} else {
			pricing.InboundPerMinute = pricingSettings.InboundPerMinute
			pricing.OutboundPerMinute = pricingSettings.OutboundPerMinute
			pricing.TranscriptionPerMinute = pricingSettings.TranscriptionPerMinute
			pricing.AnalysisPerCall = pricingSettings.AnalysisPerCall
			pricing.PhoneNumberPerMonth = pricingSettings.PhoneNumberPerMonth
			pricing.EnhancedModelPremium = pricingSettings.EnhancedModelPremium
		}
	}

	var dailyUsage []DailyUsageData
	var alerts []map[string]interface{}

	// Fetch data from Bland API
	if h.blandService != nil {
		// Get usage summary
		summary, err := h.blandService.GetUsageSummary(ctx, &bland.GetUsageSummaryRequest{
			Period: "monthly",
		})
		if err != nil {
			h.logger.Error("failed to get usage summary", zap.Error(err))
			errMsg = "Failed to load usage data"
		} else if summary != nil {
			usage.TotalCalls = summary.TotalCalls
			usage.TotalMinutes = summary.TotalMinutes
			usage.TotalCost = summary.TotalCost
			usage.AvgDuration = summary.AvgCallDuration
			usage.TranscriptionCost = summary.TranscriptionCost
			usage.AnalysisCost = summary.AnalysisCost
			usage.PhoneNumberCost = summary.PhoneNumberCost
			usage.AnalysisCount = summary.AnalysesPerformed
		}

		// Get usage limits
		limits, err := h.blandService.GetUsageLimits(ctx)
		if err != nil {
			h.logger.Warn("failed to get usage limits", zap.Error(err))
		} else if limits != nil {
			usage.CostLimit = limits.MonthlyCostLimit
			usage.MinuteLimit = float64(limits.MonthlyMinutesLimit)
		}

		// Get pricing info
		pricingInfo, err := h.blandService.GetPricing(ctx)
		if err != nil {
			h.logger.Warn("failed to get pricing info", zap.Error(err))
		} else if pricingInfo != nil {
			pricing.InboundPerMinute = pricingInfo.InboundLocal
			pricing.OutboundPerMinute = pricingInfo.OutboundLocal
			pricing.TranscriptionPerMinute = pricingInfo.TranscriptionPerMin
			pricing.AnalysisPerCall = pricingInfo.AnalysisPerCall
			pricing.PhoneNumberPerMonth = pricingInfo.LocalNumberMonthly
		}

		// Get daily usage for last 30 days
		daily, err := h.blandService.GetDailyUsage(ctx, 30)
		if err != nil {
			h.logger.Warn("failed to get daily usage", zap.Error(err))
		} else {
			for _, d := range daily {
				dailyUsage = append(dailyUsage, DailyUsageData{
					Date:    d.Date.Format("Jan 2"),
					Calls:   d.Calls,
					Minutes: d.Minutes,
					Cost:    d.Cost,
				})
			}
		}

		// Get usage alerts
		usageAlerts, err := h.blandService.GetUsageAlerts(ctx)
		if err != nil {
			h.logger.Warn("failed to get usage alerts", zap.Error(err))
		} else {
			for _, a := range usageAlerts {
				if !a.Acknowledged {
					alerts = append(alerts, map[string]interface{}{
						"ID":      a.ID,
						"Type":    a.Type,
						"Message": a.Message,
						"Time":    a.TriggeredAt.Format("Jan 2, 3:04 PM"),
					})
				}
			}
		}

		// Get phone number count
		numbers, err := h.blandService.ListPhoneNumbers(ctx, &bland.ListPhoneNumbersRequest{})
		if err != nil {
			h.logger.Warn("failed to get phone numbers for count", zap.Error(err))
		} else {
			usage.PhoneNumberCount = len(numbers)
		}
	}

	h.renderTemplate(w, r, "usage", map[string]interface{}{
		"Title":      "Usage",
		"ActiveNav":  "usage",
		"User":       user,
		"Usage":      usage,
		"Pricing":    pricing,
		"DailyUsage": dailyUsage,
		"Alerts":     alerts,
		"Error":      errMsg,
	})
}

// HandleUsageLimitsUpdate handles POST to update usage limits.
func (h *Handler) HandleUsageLimitsUpdate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	costLimit, _ := strconv.ParseFloat(r.FormValue("cost_limit"), 64)
	minuteLimit, _ := strconv.ParseFloat(r.FormValue("minute_limit"), 64)

	h.logger.Info("usage limits update requested",
		zap.Float64("cost_limit", costLimit),
		zap.Float64("minute_limit", minuteLimit),
	)

	// Update limits via Bland API
	if h.blandService != nil {
		if costLimit > 0 {
			if err := h.blandService.SetUsageLimit(ctx, "monthly_cost", costLimit); err != nil {
				h.logger.Error("failed to set cost limit", zap.Error(err))
			}
		}
		if minuteLimit > 0 {
			if err := h.blandService.SetUsageLimit(ctx, "monthly_minutes", minuteLimit); err != nil {
				h.logger.Error("failed to set minute limit", zap.Error(err))
			}
		}
	}

	http.Redirect(w, r, "/usage", http.StatusSeeOther)
}

// HandleBlockNumber handles POST to block a number.
func (h *Handler) HandleBlockNumber(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	phoneNumber := r.FormValue("phone_number")
	reason := r.FormValue("reason")

	h.logger.Info("blocking phone number",
		zap.String("phone_number", phoneNumber),
		zap.String("reason", reason),
	)

	// Block via Bland API
	if h.blandService != nil && phoneNumber != "" {
		_, err := h.blandService.BlockNumber(ctx, &bland.BlockNumberRequest{
			PhoneNumber: phoneNumber,
			Reason:      reason,
		})
		if err != nil {
			h.logger.Error("failed to block number", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/phone-numbers", http.StatusSeeOther)
}

// HandleUnblockNumber handles POST to unblock a number.
func (h *Handler) HandleUnblockNumber(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	blockedID := chi.URLParam(r, "id")

	h.logger.Info("unblocking phone number", zap.String("blocked_id", blockedID))

	// Unblock via Bland API
	if h.blandService != nil && blockedID != "" {
		if err := h.blandService.UnblockNumber(ctx, blockedID); err != nil {
			h.logger.Error("failed to unblock number", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/phone-numbers", http.StatusSeeOther)
}

// defaultSettingsData returns default settings data for fallback.
func defaultSettingsData() *SettingsData {
	return &SettingsData{
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
		ProjectTypes:          "web_app,mobile_app,api,ecommerce,custom_software,integration",
	}
}

// callSettingsToSettingsData converts domain.CallSettings to SettingsData for templates.
func callSettingsToSettingsData(cs *domain.CallSettings) *SettingsData {
	return &SettingsData{
		BusinessName:          cs.BusinessName,
		Voice:                 cs.Voice,
		VoiceStability:        cs.VoiceStability,
		VoiceSimilarityBoost:  cs.VoiceSimilarityBoost,
		VoiceStyle:            cs.VoiceStyle,
		VoiceSpeakerBoost:     cs.VoiceSpeakerBoost,
		Model:                 cs.Model,
		Language:              cs.Language,
		Temperature:           cs.Temperature,
		InterruptionThreshold: cs.InterruptionThreshold,
		WaitForGreeting:       cs.WaitForGreeting,
		NoiseCancellation:     cs.NoiseCancellation,
		BackgroundTrack:       cs.BackgroundTrack,
		MaxDurationMinutes:    cs.MaxDurationMinutes,
		RecordCalls:           cs.RecordCalls,
		QualityPreset:         cs.QualityPreset,
		CustomGreeting:        cs.CustomGreeting,
		ProjectTypes:          strings.Join(cs.ProjectTypes, ","),
	}
}

// settingsDataToCallSettings converts SettingsData from form to domain.CallSettings.
func settingsDataToCallSettings(sd *SettingsData) *domain.CallSettings {
	var projectTypes []string
	if sd.ProjectTypes != "" {
		for _, pt := range strings.Split(sd.ProjectTypes, ",") {
			pt = strings.TrimSpace(pt)
			if pt != "" {
				projectTypes = append(projectTypes, pt)
			}
		}
	}

	return &domain.CallSettings{
		BusinessName:          sd.BusinessName,
		ProjectTypes:          projectTypes,
		Voice:                 sd.Voice,
		VoiceStability:        sd.VoiceStability,
		VoiceSimilarityBoost:  sd.VoiceSimilarityBoost,
		VoiceStyle:            sd.VoiceStyle,
		VoiceSpeakerBoost:     sd.VoiceSpeakerBoost,
		Model:                 sd.Model,
		Language:              sd.Language,
		Temperature:           sd.Temperature,
		InterruptionThreshold: sd.InterruptionThreshold,
		WaitForGreeting:       sd.WaitForGreeting,
		NoiseCancellation:     sd.NoiseCancellation,
		BackgroundTrack:       sd.BackgroundTrack,
		MaxDurationMinutes:    sd.MaxDurationMinutes,
		RecordCalls:           sd.RecordCalls,
		QualityPreset:         sd.QualityPreset,
		CustomGreeting:        sd.CustomGreeting,
	}
}

// updateDashboardNavbar updates the existing handlers to include ActiveNav.
// This is done by updating the renderTemplate calls in pages.go

// Helper to get active nav from URL path
func getActiveNav(path string) string {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		switch parts[0] {
		case "dashboard":
			return "dashboard"
		case "calls":
			return "calls"
		case "phone-numbers":
			return "phone-numbers"
		case "voices":
			return "voices"
		case "usage":
			return "usage"
		case "settings":
			return "settings"
		case "knowledge-bases":
			return "knowledge-bases"
		}
	}
	return "dashboard"
}

// ===============================================
// Knowledge Base Management
// ===============================================

// HandleKnowledgeBasesPage serves the knowledge bases management page.
func (h *Handler) HandleKnowledgeBasesPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var knowledgeBases []bland.KnowledgeBase
	var errMsg string

	// Fetch knowledge bases from Bland API
	if h.blandService != nil {
		var err error
		knowledgeBases, err = h.blandService.ListKnowledgeBases(ctx)
		if err != nil {
			h.logger.Error("failed to list knowledge bases", zap.Error(err))
			errMsg = "Failed to load knowledge bases"
		}
	}

	// Check for success query param
	success := r.URL.Query().Get("success") == "1"

	h.renderTemplate(w, r, "knowledge_bases", map[string]interface{}{
		"Title":          "Knowledge Bases",
		"ActiveNav":      "knowledge-bases",
		"User":           user,
		"KnowledgeBases": knowledgeBases,
		"Error":          errMsg,
		"Success":        success,
	})
}

// HandleKnowledgeBaseCreate handles POST to create a knowledge base.
func (h *Handler) HandleKnowledgeBaseCreate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	name := r.FormValue("name")
	description := r.FormValue("description")
	text := r.FormValue("text")

	if name == "" || text == "" {
		h.renderTemplate(w, r, "knowledge_bases", map[string]interface{}{
			"Title":     "Knowledge Bases",
			"ActiveNav": "knowledge-bases",
			"User":      user,
			"Error":     "Name and content are required",
		})
		return
	}

	h.logger.Info("creating knowledge base",
		zap.String("name", name),
		zap.Int("text_length", len(text)),
	)

	// Create knowledge base via Bland API
	if h.blandService != nil {
		_, err := h.blandService.CreateKnowledgeBase(ctx, &bland.CreateKnowledgeBaseRequest{
			Name:        name,
			Description: description,
			Text:        text,
		})
		if err != nil {
			h.logger.Error("failed to create knowledge base", zap.Error(err))
			h.renderTemplate(w, r, "knowledge_bases", map[string]interface{}{
				"Title":     "Knowledge Bases",
				"ActiveNav": "knowledge-bases",
				"User":      user,
				"Error":     "Failed to create knowledge base: " + err.Error(),
			})
			return
		}
	}

	http.Redirect(w, r, "/knowledge-bases?success=1", http.StatusSeeOther)
}

// HandleKnowledgeBaseUpdate handles POST to update a knowledge base.
func (h *Handler) HandleKnowledgeBaseUpdate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	vectorID := r.FormValue("vector_id")
	name := r.FormValue("name")
	description := r.FormValue("description")
	text := r.FormValue("text")

	if vectorID == "" {
		http.Redirect(w, r, "/knowledge-bases", http.StatusSeeOther)
		return
	}

	h.logger.Info("updating knowledge base",
		zap.String("vector_id", vectorID),
		zap.String("name", name),
	)

	// Update knowledge base via Bland API
	if h.blandService != nil {
		req := &bland.UpdateKnowledgeBaseRequest{}

		if name != "" {
			req.Name = &name
		}
		if description != "" {
			req.Description = &description
		}
		if text != "" {
			req.Text = &text
		}

		if err := h.blandService.UpdateKnowledgeBase(ctx, vectorID, req); err != nil {
			h.logger.Error("failed to update knowledge base", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/knowledge-bases?success=1", http.StatusSeeOther)
}

// HandleKnowledgeBaseDelete handles POST to delete a knowledge base.
func (h *Handler) HandleKnowledgeBaseDelete(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vectorID := chi.URLParam(r, "id")

	h.logger.Info("deleting knowledge base", zap.String("vector_id", vectorID))

	// Delete knowledge base via Bland API
	if h.blandService != nil && vectorID != "" {
		if err := h.blandService.DeleteKnowledgeBase(ctx, vectorID); err != nil {
			h.logger.Error("failed to delete knowledge base", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/knowledge-bases", http.StatusSeeOther)
}

// HandleKnowledgeBaseContent handles GET to retrieve knowledge base content.
func (h *Handler) HandleKnowledgeBaseContent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vectorID := chi.URLParam(r, "id")

	if h.blandService != nil && vectorID != "" {
		kb, err := h.blandService.GetKnowledgeBase(ctx, vectorID)
		if err != nil {
			h.logger.Error("failed to get knowledge base", zap.Error(err))
			http.Error(w, "Failed to load content", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		if kb.Text != "" {
			w.Write([]byte(kb.Text))
		} else {
			w.Write([]byte("(Content not available - text may need to be fetched with include-text parameter)"))
		}
		return
	}

	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// ===============================================
// Presets (Prompts) Management
// ===============================================

// HandlePresetsPage serves the presets management page.
func (h *Handler) HandlePresetsPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var errMsg string
	var successMsg string
	var presets []*PresetData
	var totalPresets int
	var phoneNumbers []bland.PhoneNumber

	// Check for success query param
	if r.URL.Query().Get("success") == "1" {
		successMsg = "Preset saved successfully!"
	}
	if r.URL.Query().Get("applied") == "1" {
		successMsg = "Preset applied to phone number!"
	}
	if r.URL.Query().Get("deleted") == "1" {
		successMsg = "Preset deleted."
	}

	// Fetch presets from database
	if h.promptService != nil {
		prompts, total, err := h.promptService.ListPrompts(ctx, 1, 100, false)
		if err != nil {
			h.logger.Error("failed to list presets", zap.Error(err))
			errMsg = "Failed to load presets"
		} else {
			totalPresets = total
			for _, p := range prompts {
				presets = append(presets, promptToPresetData(p))
			}
		}
	}

	// Fetch phone numbers for the apply modal
	if h.blandService != nil {
		var err error
		phoneNumbers, err = h.blandService.ListPhoneNumbers(ctx, &bland.ListPhoneNumbersRequest{})
		if err != nil {
			h.logger.Warn("failed to list phone numbers", zap.Error(err))
		}
	}

	h.renderTemplate(w, r, "presets", map[string]interface{}{
		"Title":          "Presets",
		"ActiveNav":      "presets",
		"User":           user,
		"Presets":        presets,
		"TotalPresets":   totalPresets,
		"PhoneNumbers":   phoneNumbers,
		"Error":          errMsg,
		"Success":        successMsg != "",
		"SuccessMessage": successMsg,
	})
}

// PresetData holds preset data for template rendering.
type PresetData struct {
	ID                    string
	Name                  string
	Description           string
	Task                  string
	Voice                 string
	Language              string
	Model                 string
	Temperature           float64
	InterruptionThreshold int
	MaxDuration           int
	FirstSentence         string
	WaitForGreeting       bool
	NoiseCancellation     bool
	Record                bool
	IsDefault             bool
	IsActive              bool
}

// promptToPresetData converts a *domain.Prompt to PresetData.
func promptToPresetData(p *domain.Prompt) *PresetData {
	if p == nil {
		return nil
	}

	pd := &PresetData{
		ID:                p.ID.String(),
		Name:              p.Name,
		Description:       p.Description,
		Task:              p.Task,
		Voice:             p.Voice,
		Language:          p.Language,
		Model:             p.Model,
		FirstSentence:     p.FirstSentence,
		WaitForGreeting:   p.WaitForGreeting,
		NoiseCancellation: p.NoiseCancellation,
		Record:            p.Record,
		IsDefault:         p.IsDefault,
		IsActive:          p.IsActive,
	}

	if p.Temperature != nil {
		pd.Temperature = *p.Temperature
	}
	if p.InterruptionThreshold != nil {
		pd.InterruptionThreshold = *p.InterruptionThreshold
	}
	if p.MaxDuration != nil {
		pd.MaxDuration = *p.MaxDuration
	}

	return pd
}

// HandlePresetCreate handles POST to create a new preset.
func (h *Handler) HandlePresetCreate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", zap.Error(err))
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	// Parse form values
	req := &service.CreatePromptRequest{
		Name:              r.FormValue("name"),
		Description:       r.FormValue("description"),
		Task:              r.FormValue("task"),
		Voice:             r.FormValue("voice"),
		Language:          r.FormValue("language"),
		Model:             r.FormValue("model"),
		FirstSentence:     r.FormValue("first_sentence"),
		WaitForGreeting:   r.FormValue("wait_for_greeting") == "on",
		NoiseCancellation: r.FormValue("noise_cancellation") == "on",
		Record:            r.FormValue("record") == "on",
	}

	// Parse numeric values
	if temp, err := strconv.ParseFloat(r.FormValue("temperature"), 64); err == nil {
		req.Temperature = &temp
	}
	if thresh, err := strconv.Atoi(r.FormValue("interruption_threshold")); err == nil {
		req.InterruptionThreshold = &thresh
	}
	if dur, err := strconv.Atoi(r.FormValue("max_duration")); err == nil {
		req.MaxDuration = &dur
	}

	// Create the preset
	if h.promptService != nil {
		_, err := h.promptService.CreatePrompt(ctx, req)
		if err != nil {
			h.logger.Error("failed to create preset", zap.Error(err))
			h.renderTemplate(w, r, "presets", map[string]interface{}{
				"Title":     "Presets",
				"ActiveNav": "presets",
				"User":      user,
				"Error":     "Failed to create preset: " + err.Error(),
			})
			return
		}
	}

	http.Redirect(w, r, "/presets?success=1", http.StatusSeeOther)
}

// HandlePresetEditPage serves the preset edit page.
func (h *Handler) HandlePresetEditPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	presetID := chi.URLParam(r, "id")
	id, err := uuid.Parse(presetID)
	if err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	if h.promptService == nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	prompt, err := h.promptService.GetPrompt(ctx, id)
	if err != nil {
		h.logger.Error("failed to get preset", zap.Error(err))
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	// Convert to preset data for template
	preset := &PresetData{
		ID:              prompt.ID.String(),
		Name:            prompt.Name,
		Description:     prompt.Description,
		Task:            prompt.Task,
		Voice:           prompt.Voice,
		Language:        prompt.Language,
		Model:           prompt.Model,
		FirstSentence:   prompt.FirstSentence,
		WaitForGreeting: prompt.WaitForGreeting,
		NoiseCancellation: prompt.NoiseCancellation,
		Record:          prompt.Record,
		IsDefault:       prompt.IsDefault,
		IsActive:        prompt.IsActive,
	}
	if prompt.Temperature != nil {
		preset.Temperature = *prompt.Temperature
	}
	if prompt.InterruptionThreshold != nil {
		preset.InterruptionThreshold = *prompt.InterruptionThreshold
	}
	if prompt.MaxDuration != nil {
		preset.MaxDuration = *prompt.MaxDuration
	}

	h.renderTemplate(w, r, "preset_edit", map[string]interface{}{
		"Title":     "Edit Preset",
		"ActiveNav": "presets",
		"User":      user,
		"Preset":    preset,
	})
}

// HandlePresetUpdate handles POST to update a preset.
func (h *Handler) HandlePresetUpdate(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	presetID := chi.URLParam(r, "id")
	id, err := uuid.Parse(presetID)
	if err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	// Parse form values
	name := r.FormValue("name")
	description := r.FormValue("description")
	task := r.FormValue("task")
	voice := r.FormValue("voice")
	language := r.FormValue("language")
	model := r.FormValue("model")
	firstSentence := r.FormValue("first_sentence")
	waitForGreeting := r.FormValue("wait_for_greeting") == "on"
	noiseCancellation := r.FormValue("noise_cancellation") == "on"
	record := r.FormValue("record") == "on"

	req := &service.UpdatePromptRequest{
		Name:              &name,
		Description:       &description,
		Task:              &task,
		Voice:             &voice,
		Language:          &language,
		Model:             &model,
		FirstSentence:     &firstSentence,
		WaitForGreeting:   &waitForGreeting,
		NoiseCancellation: &noiseCancellation,
		Record:            &record,
	}

	// Parse numeric values
	if temp, err := strconv.ParseFloat(r.FormValue("temperature"), 64); err == nil {
		req.Temperature = &temp
	}
	if thresh, err := strconv.Atoi(r.FormValue("interruption_threshold")); err == nil {
		req.InterruptionThreshold = &thresh
	}
	if dur, err := strconv.Atoi(r.FormValue("max_duration")); err == nil {
		req.MaxDuration = &dur
	}

	if h.promptService != nil {
		_, err := h.promptService.UpdatePrompt(ctx, id, req)
		if err != nil {
			h.logger.Error("failed to update preset", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/presets?success=1", http.StatusSeeOther)
}

// HandlePresetDelete handles POST to delete a preset.
func (h *Handler) HandlePresetDelete(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	presetID := chi.URLParam(r, "id")
	id, err := uuid.Parse(presetID)
	if err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	if h.promptService != nil {
		if err := h.promptService.DeletePrompt(ctx, id); err != nil {
			h.logger.Error("failed to delete preset", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/presets?deleted=1", http.StatusSeeOther)
}

// HandlePresetSetDefault handles POST to set a preset as default.
func (h *Handler) HandlePresetSetDefault(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	presetID := chi.URLParam(r, "id")
	id, err := uuid.Parse(presetID)
	if err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	if h.promptService != nil {
		if err := h.promptService.SetDefaultPrompt(ctx, id); err != nil {
			h.logger.Error("failed to set default preset", zap.Error(err))
		}
	}

	http.Redirect(w, r, "/presets?success=1", http.StatusSeeOther)
}

// HandlePresetApply handles POST to apply a preset to a phone number.
func (h *Handler) HandlePresetApply(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	presetID := r.FormValue("preset_id")
	phoneNumber := r.FormValue("phone_number")

	id, err := uuid.Parse(presetID)
	if err != nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	if h.promptService == nil || h.blandService == nil {
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	// Get the preset
	prompt, err := h.promptService.GetPrompt(ctx, id)
	if err != nil {
		h.logger.Error("failed to get preset for apply", zap.Error(err))
		http.Redirect(w, r, "/presets", http.StatusSeeOther)
		return
	}

	// Build the inbound config from the preset
	config := &bland.InboundConfig{
		Task:              prompt.Task,
		Voice:             prompt.Voice,
		Language:          prompt.Language,
		Model:             prompt.Model,
		FirstSentence:     prompt.FirstSentence,
		WaitForGreeting:   prompt.WaitForGreeting,
		NoiseCancellation: prompt.NoiseCancellation,
		Record:            prompt.Record,
		SummaryPrompt:     prompt.SummaryPrompt,
		Keywords:          prompt.Keywords,
	}
	if prompt.Temperature != nil {
		config.Temperature = *prompt.Temperature
	}
	if prompt.InterruptionThreshold != nil {
		config.InterruptionThreshold = *prompt.InterruptionThreshold
	}
	if prompt.MaxDuration != nil {
		config.MaxDuration = *prompt.MaxDuration
	}

	// Apply to the phone number
	_, err = h.blandService.ConfigureInboundAgent(ctx, phoneNumber, config)
	if err != nil {
		h.logger.Error("failed to apply preset to phone number",
			zap.Error(err),
			zap.String("preset_id", presetID),
			zap.String("phone_number", phoneNumber),
		)
		h.renderTemplate(w, r, "presets", map[string]interface{}{
			"Title":     "Presets",
			"ActiveNav": "presets",
			"User":      user,
			"Error":     "Failed to apply preset: " + err.Error(),
		})
		return
	}

	h.logger.Info("preset applied to phone number",
		zap.String("preset_name", prompt.Name),
		zap.String("phone_number", phoneNumber),
	)

	http.Redirect(w, r, "/presets?applied=1", http.StatusSeeOther)
}
