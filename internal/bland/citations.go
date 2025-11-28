package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// CitationSchema defines a schema for extracting structured data from calls.
// Citation schemas tell Bland AI what information to extract and in what format.
type CitationSchema struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]SchemaField `json:"schema"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
}

// SchemaField defines a single field in a citation schema.
type SchemaField struct {
	// Type: string, number, boolean, array, object, enum
	Type        string `json:"type"`
	Description string `json:"description"`

	// For enum type
	Enum []string `json:"enum,omitempty"`

	// For array type
	Items *SchemaField `json:"items,omitempty"`

	// For object type
	Properties map[string]SchemaField `json:"properties,omitempty"`

	// Validation
	Required bool        `json:"required,omitempty"`
	Default  interface{} `json:"default,omitempty"`

	// Extraction hints for the AI
	Examples    []string `json:"examples,omitempty"`
	ExtractFrom string   `json:"extract_from,omitempty"` // transcript, metadata, inferred
}

// CreateCitationSchemaRequest contains parameters for creating a citation schema.
type CreateCitationSchemaRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]SchemaField `json:"schema"`
}

// UpdateCitationSchemaRequest contains parameters for updating a citation schema.
type UpdateCitationSchemaRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Schema      map[string]SchemaField `json:"schema,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

// ListCitationSchemasResponse contains the response from listing schemas.
type ListCitationSchemasResponse struct {
	Schemas []CitationSchema `json:"schemas"`
	Total   int              `json:"total,omitempty"`
}

// CitationResult represents extracted data from a call using a schema.
type CitationResult struct {
	SchemaID    string                 `json:"schema_id"`
	SchemaName  string                 `json:"schema_name"`
	CallID      string                 `json:"call_id"`
	ExtractedAt time.Time              `json:"extracted_at"`
	Data        map[string]interface{} `json:"data"`
	Confidence  map[string]float64     `json:"confidence,omitempty"`
	Sources     map[string][]string    `json:"sources,omitempty"` // Which transcript segments each field came from
}

// CreateCitationSchema creates a new citation schema.
func (c *Client) CreateCitationSchema(ctx context.Context, req *CreateCitationSchemaRequest) (*CitationSchema, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Schema) == 0 {
		return nil, fmt.Errorf("schema is required")
	}

	var schema CitationSchema
	if err := c.request(ctx, "POST", "/citations/schemas", req, &schema); err != nil {
		return nil, err
	}

	c.logger.Info("citation schema created",
		zap.String("id", schema.ID),
		zap.String("name", schema.Name),
	)

	return &schema, nil
}

// GetCitationSchema retrieves a specific citation schema.
func (c *Client) GetCitationSchema(ctx context.Context, schemaID string) (*CitationSchema, error) {
	if schemaID == "" {
		return nil, fmt.Errorf("schema_id is required")
	}

	var schema CitationSchema
	if err := c.request(ctx, "GET", "/citations/schemas/"+schemaID, nil, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// ListCitationSchemas retrieves all citation schemas.
func (c *Client) ListCitationSchemas(ctx context.Context) ([]CitationSchema, error) {
	var resp ListCitationSchemasResponse
	if err := c.request(ctx, "GET", "/citations/schemas", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Schemas, nil
}

// UpdateCitationSchema updates an existing citation schema.
func (c *Client) UpdateCitationSchema(ctx context.Context, schemaID string, req *UpdateCitationSchemaRequest) (*CitationSchema, error) {
	if schemaID == "" {
		return nil, fmt.Errorf("schema_id is required")
	}

	var schema CitationSchema
	if err := c.request(ctx, "PATCH", "/citations/schemas/"+schemaID, req, &schema); err != nil {
		return nil, err
	}

	c.logger.Info("citation schema updated", zap.String("id", schemaID))
	return &schema, nil
}

// DeleteCitationSchema deletes a citation schema.
func (c *Client) DeleteCitationSchema(ctx context.Context, schemaID string) error {
	if schemaID == "" {
		return fmt.Errorf("schema_id is required")
	}

	if err := c.request(ctx, "DELETE", "/citations/schemas/"+schemaID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("citation schema deleted", zap.String("id", schemaID))
	return nil
}

// EnableCitationSchema activates a citation schema.
func (c *Client) EnableCitationSchema(ctx context.Context, schemaID string) error {
	active := true
	_, err := c.UpdateCitationSchema(ctx, schemaID, &UpdateCitationSchemaRequest{IsActive: &active})
	return err
}

// DisableCitationSchema deactivates a citation schema.
func (c *Client) DisableCitationSchema(ctx context.Context, schemaID string) error {
	active := false
	_, err := c.UpdateCitationSchema(ctx, schemaID, &UpdateCitationSchemaRequest{IsActive: &active})
	return err
}

// GetCallCitations retrieves citation results for a specific call.
func (c *Client) GetCallCitations(ctx context.Context, callID string) ([]CitationResult, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	var resp struct {
		Citations []CitationResult `json:"citations"`
	}
	if err := c.request(ctx, "GET", "/calls/"+callID+"/citations", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Citations, nil
}

// ExtractCitations manually triggers citation extraction for a call.
func (c *Client) ExtractCitations(ctx context.Context, callID string, schemaIDs []string) ([]CitationResult, error) {
	if callID == "" {
		return nil, fmt.Errorf("call_id is required")
	}

	req := map[string]interface{}{
		"schema_ids": schemaIDs,
	}

	var resp struct {
		Citations []CitationResult `json:"citations"`
	}
	if err := c.request(ctx, "POST", "/calls/"+callID+"/citations/extract", req, &resp); err != nil {
		return nil, err
	}

	return resp.Citations, nil
}

// Helper functions for common citation schema patterns

// NewProjectQuoteCitationSchema creates a schema for software project quote extraction.
func NewProjectQuoteCitationSchema() *CreateCitationSchemaRequest {
	return &CreateCitationSchemaRequest{
		Name:        "project_quote_extraction",
		Description: "Extract key information from software project quote calls",
		Schema: map[string]SchemaField{
			"project_type": {
				Type:        "enum",
				Description: "The type of software project the customer needs",
				Enum:        []string{"web_app", "mobile_app", "api", "ecommerce", "custom_software", "integration", "other"},
				Required:    true,
				Examples:    []string{"web application", "mobile app", "API backend"},
			},
			"customer_name": {
				Type:        "string",
				Description: "The full name of the customer",
				Required:    false,
				ExtractFrom: "transcript",
			},
			"company_name": {
				Type:        "string",
				Description: "Customer's company or organization name",
				Required:    false,
			},
			"phone_number": {
				Type:        "string",
				Description: "Customer's contact phone number if provided",
				Required:    false,
			},
			"email": {
				Type:        "string",
				Description: "Customer's email address if provided",
				Required:    false,
			},
			"project_details": {
				Type:        "object",
				Description: "Details about the software project",
				Properties: map[string]SchemaField{
					"description": {
						Type:        "string",
						Description: "Main purpose or description of the project",
					},
					"target_users": {
						Type:        "string",
						Description: "Who will use the software (internal, customers, public)",
					},
					"key_features": {
						Type:        "array",
						Description: "Key features requested",
						Items: &SchemaField{
							Type: "string",
						},
					},
					"platforms": {
						Type:        "string",
						Description: "Target platforms (web, iOS, Android, both)",
					},
					"integrations": {
						Type:        "array",
						Description: "Systems to integrate with",
						Items: &SchemaField{
							Type: "string",
						},
					},
				},
			},
			"technical_requirements": {
				Type:        "object",
				Description: "Technical requirements mentioned",
				Properties: map[string]SchemaField{
					"estimated_complexity": {
						Type:        "string",
						Description: "Estimated complexity: small, medium, large, enterprise",
					},
					"performance_needs": {
						Type:        "string",
						Description: "Performance or scalability requirements",
					},
					"compliance_needs": {
						Type:        "string",
						Description: "Compliance requirements (HIPAA, SOC2, etc.)",
					},
				},
			},
			"budget_range": {
				Type:        "string",
				Description: "Customer's budget range if provided",
				Required:    false,
			},
			"timeline": {
				Type:        "string",
				Description: "When the customer needs the project completed",
				Required:    false,
				Examples:    []string{"ASAP", "3 months", "end of Q2", "flexible"},
			},
			"ongoing_support": {
				Type:        "boolean",
				Description: "Whether they need ongoing support after launch",
				Required:    false,
			},
			"call_outcome": {
				Type:        "enum",
				Description: "The outcome of the call",
				Enum:        []string{"quote_requested", "callback_scheduled", "transferred", "declined", "incomplete"},
				Required:    true,
			},
			"follow_up_required": {
				Type:        "boolean",
				Description: "Whether follow-up action is needed",
				Required:    true,
				Default:     false,
			},
			"follow_up_notes": {
				Type:        "string",
				Description: "Notes about what follow-up is needed",
				Required:    false,
			},
			"sentiment": {
				Type:        "enum",
				Description: "Overall customer sentiment during the call",
				Enum:        []string{"positive", "neutral", "negative", "frustrated"},
				Required:    false,
				ExtractFrom: "inferred",
			},
		},
	}
}

// NewSupportTicketCitationSchema creates a schema for support call extraction.
func NewSupportTicketCitationSchema() *CreateCitationSchemaRequest {
	return &CreateCitationSchemaRequest{
		Name:        "support_ticket_extraction",
		Description: "Extract support ticket information from customer service calls",
		Schema: map[string]SchemaField{
			"issue_category": {
				Type:        "enum",
				Description: "The category of the customer's issue",
				Enum:        []string{"billing", "technical", "account", "product", "shipping", "returns", "general", "complaint"},
				Required:    true,
			},
			"issue_summary": {
				Type:        "string",
				Description: "A brief summary of the customer's issue",
				Required:    true,
			},
			"issue_details": {
				Type:        "string",
				Description: "Detailed description of the issue",
				Required:    false,
			},
			"customer_id": {
				Type:        "string",
				Description: "Customer's account or ID number if provided",
				Required:    false,
			},
			"order_number": {
				Type:        "string",
				Description: "Order or reference number if applicable",
				Required:    false,
			},
			"resolution_status": {
				Type:        "enum",
				Description: "Whether the issue was resolved",
				Enum:        []string{"resolved", "escalated", "pending", "unresolved"},
				Required:    true,
			},
			"resolution_details": {
				Type:        "string",
				Description: "How the issue was resolved or what next steps are needed",
				Required:    false,
			},
			"priority": {
				Type:        "enum",
				Description: "Priority level based on issue severity",
				Enum:        []string{"low", "medium", "high", "urgent"},
				Required:    false,
				ExtractFrom: "inferred",
			},
			"sentiment": {
				Type:        "enum",
				Description: "Customer sentiment at end of call",
				Enum:        []string{"satisfied", "neutral", "dissatisfied", "angry"},
				Required:    false,
			},
		},
	}
}

// NewAppointmentCitationSchema creates a schema for appointment scheduling extraction.
func NewAppointmentCitationSchema() *CreateCitationSchemaRequest {
	return &CreateCitationSchemaRequest{
		Name:        "appointment_extraction",
		Description: "Extract appointment scheduling information from calls",
		Schema: map[string]SchemaField{
			"appointment_type": {
				Type:        "string",
				Description: "The type of appointment being scheduled",
				Required:    true,
			},
			"preferred_date": {
				Type:        "string",
				Description: "Customer's preferred date",
				Required:    false,
			},
			"preferred_time": {
				Type:        "string",
				Description: "Customer's preferred time",
				Required:    false,
			},
			"scheduled_datetime": {
				Type:        "string",
				Description: "The actual scheduled date and time",
				Required:    false,
			},
			"location": {
				Type:        "string",
				Description: "Location for the appointment if applicable",
				Required:    false,
			},
			"booking_status": {
				Type:        "enum",
				Description: "Status of the booking",
				Enum:        []string{"booked", "waitlisted", "cancelled", "rescheduled", "failed"},
				Required:    true,
			},
			"confirmation_sent": {
				Type:        "boolean",
				Description: "Whether a confirmation was sent",
				Required:    false,
			},
			"special_requests": {
				Type:        "string",
				Description: "Any special requests or notes",
				Required:    false,
			},
		},
	}
}

// NewLeadQualificationCitationSchema creates a schema for sales lead qualification.
func NewLeadQualificationCitationSchema() *CreateCitationSchemaRequest {
	return &CreateCitationSchemaRequest{
		Name:        "lead_qualification",
		Description: "Extract lead qualification data from sales calls",
		Schema: map[string]SchemaField{
			"company_name": {
				Type:        "string",
				Description: "Name of the prospect's company",
				Required:    false,
			},
			"contact_name": {
				Type:        "string",
				Description: "Name of the contact person",
				Required:    false,
			},
			"contact_role": {
				Type:        "string",
				Description: "The contact's role or title",
				Required:    false,
			},
			"budget": {
				Type:        "string",
				Description: "Budget range or constraints mentioned",
				Required:    false,
			},
			"authority": {
				Type:        "enum",
				Description: "Decision-making authority level",
				Enum:        []string{"decision_maker", "influencer", "user", "unknown"},
				Required:    false,
			},
			"need": {
				Type:        "string",
				Description: "Primary need or pain point expressed",
				Required:    false,
			},
			"timeline": {
				Type:        "string",
				Description: "Purchase timeline or urgency",
				Required:    false,
			},
			"lead_score": {
				Type:        "enum",
				Description: "Qualification score based on BANT criteria",
				Enum:        []string{"hot", "warm", "cold", "disqualified"},
				Required:    true,
				ExtractFrom: "inferred",
			},
			"next_steps": {
				Type:        "array",
				Description: "Agreed upon next steps",
				Items: &SchemaField{
					Type: "string",
				},
			},
			"competitors_mentioned": {
				Type:        "array",
				Description: "Competitors mentioned during the call",
				Items: &SchemaField{
					Type: "string",
				},
			},
			"objections": {
				Type:        "array",
				Description: "Objections or concerns raised",
				Items: &SchemaField{
					Type: "string",
				},
			},
		},
	}
}
