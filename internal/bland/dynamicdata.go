package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// DynamicDataSource represents a source of dynamic data that can be used in calls.
// Dynamic data allows real-time information to be injected into conversations.
type DynamicDataSource struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	Type          string                   `json:"type"` // webhook, static, database
	Config        *DynamicDataSourceConfig `json:"config,omitempty"`
	Variables     []DynamicVariable        `json:"variables,omitempty"`
	DefaultValues map[string]interface{}   `json:"default_values,omitempty"`
	IsActive      bool                     `json:"is_active"`
	CreatedAt     time.Time                `json:"created_at,omitempty"`
	UpdatedAt     time.Time                `json:"updated_at,omitempty"`
}

// DynamicDataSourceConfig contains configuration for a dynamic data source.
// This is different from DynamicDataConfig in calls.go which is for inline dynamic data during calls.
type DynamicDataSourceConfig struct {
	// For webhook type
	URL           string            `json:"url,omitempty"`
	Method        string            `json:"method,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RefreshRate   int               `json:"refresh_rate,omitempty"` // seconds
	CacheDuration int               `json:"cache_duration,omitempty"`

	// For database type
	ConnectionString string `json:"connection_string,omitempty"`
	Query            string `json:"query,omitempty"`

	// Authentication
	AuthType   string                 `json:"auth_type,omitempty"` // none, api_key, oauth, basic
	AuthConfig map[string]interface{} `json:"auth_config,omitempty"`

	// Error handling
	FallbackValues map[string]interface{} `json:"fallback_values,omitempty"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
}

// DynamicVariable represents a variable that can be populated dynamically.
type DynamicVariable struct {
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	Type         string      `json:"type"` // string, number, boolean, object, array
	Required     bool        `json:"required,omitempty"`
	Default      interface{} `json:"default,omitempty"`
	Source       string      `json:"source,omitempty"` // JSONPath or field mapping
	Validation   string      `json:"validation,omitempty"`
	Transform    string      `json:"transform,omitempty"` // Transform expression
}

// CreateDynamicDataSourceRequest contains parameters for creating a data source.
type CreateDynamicDataSourceRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	Type          string                   `json:"type"`
	Config        *DynamicDataSourceConfig `json:"config,omitempty"`
	Variables     []DynamicVariable        `json:"variables,omitempty"`
	DefaultValues map[string]interface{}   `json:"default_values,omitempty"`
}

// UpdateDynamicDataSourceRequest contains parameters for updating a data source.
type UpdateDynamicDataSourceRequest struct {
	Name          *string                  `json:"name,omitempty"`
	Description   *string                  `json:"description,omitempty"`
	Config        *DynamicDataSourceConfig `json:"config,omitempty"`
	Variables     []DynamicVariable        `json:"variables,omitempty"`
	DefaultValues map[string]interface{}   `json:"default_values,omitempty"`
	IsActive      *bool                    `json:"is_active,omitempty"`
}

// ListDynamicDataSourcesResponse contains the response from listing data sources.
type ListDynamicDataSourcesResponse struct {
	Sources []DynamicDataSource `json:"sources"`
	Total   int                 `json:"total,omitempty"`
}

// DynamicDataTestResult contains the result of testing a data source.
type DynamicDataTestResult struct {
	Success    bool                   `json:"success"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	LatencyMs  int                    `json:"latency_ms"`
	TestedAt   time.Time              `json:"tested_at"`
}

// CreateDynamicDataSource creates a new dynamic data source.
func (c *Client) CreateDynamicDataSource(ctx context.Context, req *CreateDynamicDataSourceRequest) (*DynamicDataSource, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	var source DynamicDataSource
	if err := c.request(ctx, "POST", "/dynamic-data", req, &source); err != nil {
		return nil, err
	}

	c.logger.Info("dynamic data source created",
		zap.String("id", source.ID),
		zap.String("name", source.Name),
		zap.String("type", source.Type),
	)

	return &source, nil
}

// GetDynamicDataSource retrieves a specific data source.
func (c *Client) GetDynamicDataSource(ctx context.Context, sourceID string) (*DynamicDataSource, error) {
	if sourceID == "" {
		return nil, fmt.Errorf("source_id is required")
	}

	var source DynamicDataSource
	if err := c.request(ctx, "GET", "/dynamic-data/"+sourceID, nil, &source); err != nil {
		return nil, err
	}

	return &source, nil
}

// ListDynamicDataSources retrieves all dynamic data sources.
func (c *Client) ListDynamicDataSources(ctx context.Context) ([]DynamicDataSource, error) {
	var resp ListDynamicDataSourcesResponse
	if err := c.request(ctx, "GET", "/dynamic-data", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Sources, nil
}

// UpdateDynamicDataSource updates an existing data source.
func (c *Client) UpdateDynamicDataSource(ctx context.Context, sourceID string, req *UpdateDynamicDataSourceRequest) (*DynamicDataSource, error) {
	if sourceID == "" {
		return nil, fmt.Errorf("source_id is required")
	}

	var source DynamicDataSource
	if err := c.request(ctx, "PATCH", "/dynamic-data/"+sourceID, req, &source); err != nil {
		return nil, err
	}

	c.logger.Info("dynamic data source updated", zap.String("id", sourceID))
	return &source, nil
}

// DeleteDynamicDataSource deletes a data source.
func (c *Client) DeleteDynamicDataSource(ctx context.Context, sourceID string) error {
	if sourceID == "" {
		return fmt.Errorf("source_id is required")
	}

	if err := c.request(ctx, "DELETE", "/dynamic-data/"+sourceID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("dynamic data source deleted", zap.String("id", sourceID))
	return nil
}

// TestDynamicDataSource tests a data source connection and retrieval.
func (c *Client) TestDynamicDataSource(ctx context.Context, sourceID string, testParams map[string]interface{}) (*DynamicDataTestResult, error) {
	if sourceID == "" {
		return nil, fmt.Errorf("source_id is required")
	}

	req := map[string]interface{}{
		"params": testParams,
	}

	var result DynamicDataTestResult
	if err := c.request(ctx, "POST", "/dynamic-data/"+sourceID+"/test", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RefreshDynamicDataSource manually triggers a refresh of cached data.
func (c *Client) RefreshDynamicDataSource(ctx context.Context, sourceID string) error {
	if sourceID == "" {
		return fmt.Errorf("source_id is required")
	}

	if err := c.request(ctx, "POST", "/dynamic-data/"+sourceID+"/refresh", nil, nil); err != nil {
		return err
	}

	c.logger.Info("dynamic data source refreshed", zap.String("id", sourceID))
	return nil
}

// EnableDynamicDataSource activates a data source.
func (c *Client) EnableDynamicDataSource(ctx context.Context, sourceID string) error {
	active := true
	_, err := c.UpdateDynamicDataSource(ctx, sourceID, &UpdateDynamicDataSourceRequest{IsActive: &active})
	return err
}

// DisableDynamicDataSource deactivates a data source.
func (c *Client) DisableDynamicDataSource(ctx context.Context, sourceID string) error {
	active := false
	_, err := c.UpdateDynamicDataSource(ctx, sourceID, &UpdateDynamicDataSourceRequest{IsActive: &active})
	return err
}

// Helper functions for common dynamic data patterns

// NewWebhookDataSource creates a webhook-based dynamic data source.
func NewWebhookDataSource(name, url string, variables []DynamicVariable) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:        name,
		Description: "Webhook-based dynamic data source",
		Type:        "webhook",
		Config: &DynamicDataSourceConfig{
			URL:           url,
			Method:        "POST",
			Headers:       map[string]string{"Content-Type": "application/json"},
			CacheDuration: 300, // 5 minutes
			TimeoutSeconds: 10,
		},
		Variables: variables,
	}
}

// NewStaticDataSource creates a static data source with preset values.
func NewStaticDataSource(name string, values map[string]interface{}) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:          name,
		Description:   "Static data source with preset values",
		Type:          "static",
		DefaultValues: values,
	}
}

// NewCustomerDataSource creates a data source for customer information.
func NewCustomerDataSource(webhookURL string) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:        "customer_data",
		Description: "Retrieves customer information based on phone number",
		Type:        "webhook",
		Config: &DynamicDataSourceConfig{
			URL:           webhookURL,
			Method:        "POST",
			CacheDuration: 60,
			TimeoutSeconds: 5,
			FallbackValues: map[string]interface{}{
				"customer_name":    "Valued Customer",
				"account_status":   "unknown",
				"previous_calls":   0,
			},
		},
		Variables: []DynamicVariable{
			{
				Name:        "phone_number",
				Description: "Customer's phone number",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "customer_name",
				Description: "Customer's name from CRM",
				Type:        "string",
				Source:      "$.data.name",
			},
			{
				Name:        "account_status",
				Description: "Customer's account status",
				Type:        "string",
				Source:      "$.data.status",
			},
			{
				Name:        "account_balance",
				Description: "Current account balance",
				Type:        "number",
				Source:      "$.data.balance",
			},
			{
				Name:        "previous_calls",
				Description: "Number of previous calls",
				Type:        "number",
				Source:      "$.data.call_count",
			},
			{
				Name:        "last_interaction",
				Description: "Date of last interaction",
				Type:        "string",
				Source:      "$.data.last_contact",
			},
			{
				Name:        "notes",
				Description: "Account notes",
				Type:        "string",
				Source:      "$.data.notes",
			},
		},
	}
}

// NewPricingDataSource creates a data source for dynamic pricing.
func NewPricingDataSource(webhookURL string) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:        "pricing_data",
		Description: "Retrieves current pricing information",
		Type:        "webhook",
		Config: &DynamicDataSourceConfig{
			URL:           webhookURL,
			Method:        "GET",
			RefreshRate:   300, // Refresh every 5 minutes
			CacheDuration: 300,
			TimeoutSeconds: 10,
		},
		Variables: []DynamicVariable{
			{
				Name:        "base_price",
				Description: "Base product price",
				Type:        "number",
				Source:      "$.pricing.base",
			},
			{
				Name:        "discount_percent",
				Description: "Current discount percentage",
				Type:        "number",
				Source:      "$.pricing.discount",
				Default:     0,
			},
			{
				Name:        "promo_code",
				Description: "Active promo code",
				Type:        "string",
				Source:      "$.pricing.promo_code",
			},
			{
				Name:        "promo_valid_until",
				Description: "Promo expiration date",
				Type:        "string",
				Source:      "$.pricing.promo_expires",
			},
		},
	}
}

// NewInventoryDataSource creates a data source for inventory/availability.
func NewInventoryDataSource(webhookURL string) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:        "inventory_data",
		Description: "Retrieves real-time inventory and availability",
		Type:        "webhook",
		Config: &DynamicDataSourceConfig{
			URL:           webhookURL,
			Method:        "POST",
			RefreshRate:   60,  // Refresh every minute
			CacheDuration: 60,
			TimeoutSeconds: 5,
		},
		Variables: []DynamicVariable{
			{
				Name:        "product_id",
				Description: "Product ID to check",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "in_stock",
				Description: "Whether item is in stock",
				Type:        "boolean",
				Source:      "$.inventory.available",
			},
			{
				Name:        "quantity_available",
				Description: "Number of units available",
				Type:        "number",
				Source:      "$.inventory.quantity",
			},
			{
				Name:        "estimated_delivery",
				Description: "Estimated delivery time",
				Type:        "string",
				Source:      "$.inventory.delivery_estimate",
			},
			{
				Name:        "backorder_date",
				Description: "Expected backorder date if out of stock",
				Type:        "string",
				Source:      "$.inventory.backorder_date",
			},
		},
	}
}

// NewAppointmentSlotsDataSource creates a data source for appointment availability.
func NewAppointmentSlotsDataSource(webhookURL string) *CreateDynamicDataSourceRequest {
	return &CreateDynamicDataSourceRequest{
		Name:        "appointment_slots",
		Description: "Retrieves available appointment slots",
		Type:        "webhook",
		Config: &DynamicDataSourceConfig{
			URL:           webhookURL,
			Method:        "POST",
			RefreshRate:   60,
			CacheDuration: 30, // Short cache for real-time availability
			TimeoutSeconds: 10,
		},
		Variables: []DynamicVariable{
			{
				Name:        "date",
				Description: "Date to check availability",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "service_type",
				Description: "Type of appointment",
				Type:        "string",
				Required:    false,
			},
			{
				Name:        "available_slots",
				Description: "List of available time slots",
				Type:        "array",
				Source:      "$.slots",
			},
			{
				Name:        "next_available",
				Description: "Next available slot if none on requested date",
				Type:        "string",
				Source:      "$.next_available",
			},
		},
	}
}
