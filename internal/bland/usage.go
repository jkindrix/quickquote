package bland

import (
	"context"
	"fmt"
	"time"
)

// UsageSummary contains overall API usage statistics.
type UsageSummary struct {
	Period        string    `json:"period"` // daily, weekly, monthly
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`

	// Call metrics
	TotalCalls        int     `json:"total_calls"`
	SuccessfulCalls   int     `json:"successful_calls"`
	FailedCalls       int     `json:"failed_calls"`
	TotalMinutes      float64 `json:"total_minutes"`
	AvgCallDuration   float64 `json:"avg_call_duration_seconds"`

	// Cost breakdown
	TotalCost         float64 `json:"total_cost"`
	CallCost          float64 `json:"call_cost"`
	SMSCost           float64 `json:"sms_cost"`
	TranscriptionCost float64 `json:"transcription_cost"`
	AnalysisCost      float64 `json:"analysis_cost"`
	PhoneNumberCost   float64 `json:"phone_number_cost"`

	// SMS metrics
	TotalSMS          int     `json:"total_sms"`
	SMSSent           int     `json:"sms_sent"`
	SMSReceived       int     `json:"sms_received"`

	// API usage
	APIRequests       int     `json:"api_requests"`
	APIErrors         int     `json:"api_errors"`

	// Analysis metrics
	TranscriptionsGenerated int `json:"transcriptions_generated"`
	AnalysesPerformed       int `json:"analyses_performed"`
}

// DailyUsage contains usage for a specific day.
type DailyUsage struct {
	Date              time.Time `json:"date"`
	Calls             int       `json:"calls"`
	Minutes           float64   `json:"minutes"`
	Cost              float64   `json:"cost"`
	SMS               int       `json:"sms"`
	APIRequests       int       `json:"api_requests"`
}

// CallCostBreakdown shows cost breakdown for a specific call.
type CallCostBreakdown struct {
	CallID            string    `json:"call_id"`
	Duration          float64   `json:"duration_seconds"`
	BaseCost          float64   `json:"base_cost"`
	TranscriptionCost float64   `json:"transcription_cost"`
	AnalysisCost      float64   `json:"analysis_cost"`
	TotalCost         float64   `json:"total_cost"`
	PhoneNumberType   string    `json:"phone_number_type"` // local, toll-free
	Direction         string    `json:"direction"`         // inbound, outbound
	CreatedAt         time.Time `json:"created_at"`
}

// UsageLimits contains account limits and current usage.
type UsageLimits struct {
	// Monthly limits
	MonthlyMinutesLimit     int     `json:"monthly_minutes_limit"`
	MonthlyMinutesUsed      float64 `json:"monthly_minutes_used"`
	MonthlyCostLimit        float64 `json:"monthly_cost_limit"`
	MonthlyCostUsed         float64 `json:"monthly_cost_used"`

	// Concurrent limits
	MaxConcurrentCalls      int     `json:"max_concurrent_calls"`
	CurrentConcurrentCalls  int     `json:"current_concurrent_calls"`

	// Rate limits
	CallsPerMinute          int     `json:"calls_per_minute"`
	CallsPerHour            int     `json:"calls_per_hour"`
	APIRequestsPerSecond    int     `json:"api_requests_per_second"`

	// Notifications
	NotificationThresholds  []int   `json:"notification_thresholds"` // Percentages (e.g., 50, 75, 90)
}

// PricingInfo contains current pricing information.
type PricingInfo struct {
	Currency         string  `json:"currency"`

	// Per-minute rates
	OutboundLocal    float64 `json:"outbound_local_per_min"`
	OutboundTollFree float64 `json:"outbound_toll_free_per_min"`
	InboundLocal     float64 `json:"inbound_local_per_min"`
	InboundTollFree  float64 `json:"inbound_toll_free_per_min"`

	// Feature costs
	TranscriptionPerMin  float64 `json:"transcription_per_min"`
	AnalysisPerCall      float64 `json:"analysis_per_call"`
	SMSOutbound          float64 `json:"sms_outbound"`
	SMSInbound           float64 `json:"sms_inbound"`

	// Phone number costs
	LocalNumberMonthly     float64 `json:"local_number_monthly"`
	TollFreeNumberMonthly  float64 `json:"toll_free_number_monthly"`
}

// UsageAlert represents an alert for usage thresholds.
type UsageAlert struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // minutes, cost, calls
	Threshold     float64   `json:"threshold"`
	ThresholdType string    `json:"threshold_type"` // percentage, absolute
	CurrentValue  float64   `json:"current_value"`
	Message       string    `json:"message"`
	TriggeredAt   time.Time `json:"triggered_at"`
	Acknowledged  bool      `json:"acknowledged"`
}

// GetUsageSummaryRequest contains parameters for fetching usage.
type GetUsageSummaryRequest struct {
	Period    string     `json:"period,omitempty"` // daily, weekly, monthly, custom
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// GetUsageSummary retrieves usage statistics for a period.
func (c *Client) GetUsageSummary(ctx context.Context, req *GetUsageSummaryRequest) (*UsageSummary, error) {
	path := "/usage/summary"

	if req != nil {
		params := ""
		if req.Period != "" {
			params += fmt.Sprintf("period=%s&", req.Period)
		}
		if req.StartDate != nil {
			params += fmt.Sprintf("start_date=%s&", req.StartDate.Format("2006-01-02"))
		}
		if req.EndDate != nil {
			params += fmt.Sprintf("end_date=%s&", req.EndDate.Format("2006-01-02"))
		}
		if params != "" {
			path += "?" + params[:len(params)-1]
		}
	}

	var summary UsageSummary
	if err := c.request(ctx, "GET", path, nil, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetDailyUsage retrieves day-by-day usage for a date range.
func (c *Client) GetDailyUsage(ctx context.Context, startDate, endDate time.Time) ([]DailyUsage, error) {
	path := fmt.Sprintf("/usage/daily?start_date=%s&end_date=%s",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)

	var resp struct {
		Usage []DailyUsage `json:"usage"`
	}
	if err := c.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Usage, nil
}

// GetCallCost retrieves the cost breakdown for a specific call.
func (c *Client) GetCallCost(ctx context.Context, callID string) (*CallCostBreakdown, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var cost CallCostBreakdown
	if err := c.request(ctx, "GET", "/usage/calls/"+callID+"/cost", nil, &cost); err != nil {
		return nil, err
	}

	return &cost, nil
}

// GetUsageLimits retrieves current account limits and usage.
func (c *Client) GetUsageLimits(ctx context.Context) (*UsageLimits, error) {
	var limits UsageLimits
	if err := c.request(ctx, "GET", "/usage/limits", nil, &limits); err != nil {
		return nil, err
	}

	return &limits, nil
}

// SetUsageLimit sets a usage limit.
func (c *Client) SetUsageLimit(ctx context.Context, limitType string, value float64) error {
	req := map[string]interface{}{
		"type":  limitType,
		"value": value,
	}

	return c.request(ctx, "POST", "/usage/limits", req, nil)
}

// GetPricing retrieves current pricing information.
func (c *Client) GetPricing(ctx context.Context) (*PricingInfo, error) {
	var pricing PricingInfo
	if err := c.request(ctx, "GET", "/usage/pricing", nil, &pricing); err != nil {
		return nil, err
	}

	return &pricing, nil
}

// GetUsageAlerts retrieves active usage alerts.
func (c *Client) GetUsageAlerts(ctx context.Context) ([]UsageAlert, error) {
	var resp struct {
		Alerts []UsageAlert `json:"alerts"`
	}
	if err := c.request(ctx, "GET", "/usage/alerts", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Alerts, nil
}

// SetAlertThreshold creates or updates an alert threshold.
func (c *Client) SetAlertThreshold(ctx context.Context, alertType string, threshold float64, thresholdType string) error {
	req := map[string]interface{}{
		"type":           alertType,
		"threshold":      threshold,
		"threshold_type": thresholdType,
	}

	return c.request(ctx, "POST", "/usage/alerts", req, nil)
}

// AcknowledgeAlert marks an alert as acknowledged.
func (c *Client) AcknowledgeAlert(ctx context.Context, alertID string) error {
	if alertID == "" {
		return fmt.Errorf("alert_id is required")
	}

	return c.request(ctx, "POST", "/usage/alerts/"+alertID+"/acknowledge", nil, nil)
}

// EstimateCallCost estimates the cost for a potential call.
func (c *Client) EstimateCallCost(ctx context.Context, durationMinutes float64, direction, numberType string, includeTranscription, includeAnalysis bool) (float64, error) {
	req := map[string]interface{}{
		"duration_minutes":      durationMinutes,
		"direction":             direction,
		"number_type":           numberType,
		"include_transcription": includeTranscription,
		"include_analysis":      includeAnalysis,
	}

	var resp struct {
		EstimatedCost float64 `json:"estimated_cost"`
	}
	if err := c.request(ctx, "POST", "/usage/estimate", req, &resp); err != nil {
		return 0, err
	}

	return resp.EstimatedCost, nil
}

// GetCurrentMonthUsage is a convenience method for current month's usage.
func (c *Client) GetCurrentMonthUsage(ctx context.Context) (*UsageSummary, error) {
	return c.GetUsageSummary(ctx, &GetUsageSummaryRequest{Period: "monthly"})
}

// GetTodayUsage is a convenience method for today's usage.
func (c *Client) GetTodayUsage(ctx context.Context) (*UsageSummary, error) {
	return c.GetUsageSummary(ctx, &GetUsageSummaryRequest{Period: "daily"})
}

// IsNearLimit checks if usage is approaching a limit.
func (c *Client) IsNearLimit(ctx context.Context, threshold float64) (bool, string, error) {
	limits, err := c.GetUsageLimits(ctx)
	if err != nil {
		return false, "", err
	}

	// Check minutes limit
	if limits.MonthlyMinutesLimit > 0 {
		percentUsed := (limits.MonthlyMinutesUsed / float64(limits.MonthlyMinutesLimit)) * 100
		if percentUsed >= threshold {
			return true, fmt.Sprintf("Minutes usage at %.1f%%", percentUsed), nil
		}
	}

	// Check cost limit
	if limits.MonthlyCostLimit > 0 {
		percentUsed := (limits.MonthlyCostUsed / limits.MonthlyCostLimit) * 100
		if percentUsed >= threshold {
			return true, fmt.Sprintf("Cost usage at %.1f%%", percentUsed), nil
		}
	}

	// Check concurrent calls
	if limits.MaxConcurrentCalls > 0 {
		percentUsed := (float64(limits.CurrentConcurrentCalls) / float64(limits.MaxConcurrentCalls)) * 100
		if percentUsed >= threshold {
			return true, fmt.Sprintf("Concurrent calls at %.1f%%", percentUsed), nil
		}
	}

	return false, "", nil
}
