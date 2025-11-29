// Package handler provides HTTP handlers for the application.
// This file contains typed DTOs for template data.
package handler

import (
	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/domain"
)

// BasePageData contains common fields for all page templates.
type BasePageData struct {
	Title     string
	ActiveNav string
	User      *domain.User
	CSRFToken string
}

// LoginPageData contains data for the login page template.
type LoginPageData struct {
	Title string
	Error string
	Email string
}

// DashboardPageData contains data for the dashboard template.
type DashboardPageData struct {
	BasePageData
	Calls         []*domain.Call
	TotalCalls    int
	PendingQuotes int
}

// CallsPageData contains data for the calls list template.
type CallsPageData struct {
	BasePageData
	Calls      []*domain.Call
	TotalCalls int
	Page       int
	PageSize   int
	TotalPages int
}

// CallDetailPageData contains data for the call detail template.
type CallDetailPageData struct {
	BasePageData
	Call *domain.Call
}

// SettingsPageData contains data for the settings template.
// Settings uses interface{} as the actual type varies by usage context.
type SettingsPageData struct {
	BasePageData
	Settings        interface{}
	Success         string
	Error           string
	Voices          []bland.Voice
	SelectedVoiceID string
}

// PhoneNumbersPageData contains data for the phone numbers template.
type PhoneNumbersPageData struct {
	BasePageData
	Numbers    []bland.PhoneNumber
	AreaCode   string
	Country    string
	NumberType string
	Success    string
	Error      string
}

// VoicesPageData contains data for the voices template.
// CustomVoices is interface{} as it may be a list of cloned voices.
type VoicesPageData struct {
	BasePageData
	Voices       []bland.Voice
	CustomVoices interface{}
	Error        string
	Success      string
}

// UsageAlert represents a usage alert for the template.
type UsageAlert struct {
	Type    string
	Title   string
	Message string
}

// UsagePageData contains data for the usage template.
// Some fields use interface{} as they come from various API responses.
type UsagePageData struct {
	BasePageData
	Summary    *bland.UsageSummary
	Limits     *bland.UsageLimits
	Pricing    *bland.PricingInfo
	CallCosts  interface{}
	Alerts     []UsageAlert
	HasPricing bool
	Error      string
}

// KnowledgeBasesPageData contains data for the knowledge bases template.
type KnowledgeBasesPageData struct {
	BasePageData
	KnowledgeBases []bland.KnowledgeBase
	Success        string
	Error          string
}

// PresetsPageData contains data for the presets/prompts list template.
type PresetsPageData struct {
	BasePageData
	Prompts []*domain.Prompt
	Success string
	Error   string
}

// PresetEditPageData contains data for the preset edit template.
// CustomVoices is interface{} as it may be a list of cloned voices.
type PresetEditPageData struct {
	BasePageData
	Prompt       *domain.Prompt
	IsNew        bool
	Error        string
	Success      string
	Voices       []bland.Voice
	CustomVoices interface{}
}

// ToMap converts BasePageData to a map for template rendering.
// This allows gradual migration from map[string]interface{} to typed DTOs.
func (d *BasePageData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Title":     d.Title,
		"ActiveNav": d.ActiveNav,
		"User":      d.User,
		"CSRFToken": d.CSRFToken,
	}
}

// ToMap converts LoginPageData to a map for template rendering.
func (d *LoginPageData) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"Title": d.Title,
	}
	if d.Error != "" {
		m["Error"] = d.Error
	}
	if d.Email != "" {
		m["Email"] = d.Email
	}
	return m
}

// ToMap converts DashboardPageData to a map for template rendering.
func (d *DashboardPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Calls"] = d.Calls
	m["TotalCalls"] = d.TotalCalls
	m["PendingQuotes"] = d.PendingQuotes
	return m
}

// ToMap converts CallsPageData to a map for template rendering.
func (d *CallsPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Calls"] = d.Calls
	m["TotalCalls"] = d.TotalCalls
	m["Page"] = d.Page
	m["PageSize"] = d.PageSize
	m["TotalPages"] = d.TotalPages
	return m
}

// ToMap converts CallDetailPageData to a map for template rendering.
func (d *CallDetailPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Call"] = d.Call
	return m
}

// ToMap converts SettingsPageData to a map for template rendering.
func (d *SettingsPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Settings"] = d.Settings
	if d.Success != "" {
		m["Success"] = d.Success
	}
	if d.Error != "" {
		m["Error"] = d.Error
	}
	if len(d.Voices) > 0 {
		m["Voices"] = d.Voices
	}
	if d.SelectedVoiceID != "" {
		m["SelectedVoiceID"] = d.SelectedVoiceID
	}
	return m
}

// ToMap converts PhoneNumbersPageData to a map for template rendering.
func (d *PhoneNumbersPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Numbers"] = d.Numbers
	if d.AreaCode != "" {
		m["AreaCode"] = d.AreaCode
	}
	if d.Country != "" {
		m["Country"] = d.Country
	}
	if d.NumberType != "" {
		m["NumberType"] = d.NumberType
	}
	if d.Success != "" {
		m["Success"] = d.Success
	}
	if d.Error != "" {
		m["Error"] = d.Error
	}
	return m
}

// ToMap converts VoicesPageData to a map for template rendering.
func (d *VoicesPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Voices"] = d.Voices
	m["CustomVoices"] = d.CustomVoices
	if d.Error != "" {
		m["Error"] = d.Error
	}
	if d.Success != "" {
		m["Success"] = d.Success
	}
	return m
}

// ToMap converts UsagePageData to a map for template rendering.
func (d *UsagePageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Summary"] = d.Summary
	m["Limits"] = d.Limits
	m["Pricing"] = d.Pricing
	m["CallCosts"] = d.CallCosts
	m["Alerts"] = d.Alerts
	m["HasPricing"] = d.HasPricing
	if d.Error != "" {
		m["Error"] = d.Error
	}
	return m
}

// ToMap converts KnowledgeBasesPageData to a map for template rendering.
func (d *KnowledgeBasesPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["KnowledgeBases"] = d.KnowledgeBases
	if d.Success != "" {
		m["Success"] = d.Success
	}
	if d.Error != "" {
		m["Error"] = d.Error
	}
	return m
}

// ToMap converts PresetsPageData to a map for template rendering.
func (d *PresetsPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Prompts"] = d.Prompts
	if d.Success != "" {
		m["Success"] = d.Success
	}
	if d.Error != "" {
		m["Error"] = d.Error
	}
	return m
}

// ToMap converts PresetEditPageData to a map for template rendering.
func (d *PresetEditPageData) ToMap() map[string]interface{} {
	m := d.BasePageData.ToMap()
	m["Prompt"] = d.Prompt
	m["IsNew"] = d.IsNew
	m["Voices"] = d.Voices
	m["CustomVoices"] = d.CustomVoices
	if d.Error != "" {
		m["Error"] = d.Error
	}
	if d.Success != "" {
		m["Success"] = d.Success
	}
	return m
}

// TemplateData is an interface for all template data types.
type TemplateData interface {
	ToMap() map[string]interface{}
}
