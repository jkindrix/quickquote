package bland

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Enterprise features for advanced telephony configurations

// ========== BYOT (Bring Your Own Twilio) ==========

// TwilioAccount represents a custom Twilio account configuration.
type TwilioAccount struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	AccountSID      string    `json:"account_sid"`
	AuthToken       string    `json:"auth_token,omitempty"` // Only on create/update
	TrunkSID        string    `json:"trunk_sid,omitempty"`
	IsActive        bool      `json:"is_active"`
	IsVerified      bool      `json:"is_verified"`
	PhoneNumbers    []string  `json:"phone_numbers,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
	LastUsed        time.Time `json:"last_used,omitempty"`
}

// CreateTwilioAccountRequest contains parameters for connecting a Twilio account.
type CreateTwilioAccountRequest struct {
	Name        string `json:"name"`
	AccountSID  string `json:"account_sid"`
	AuthToken   string `json:"auth_token"`
	TrunkSID    string `json:"trunk_sid,omitempty"` // For SIP trunking
}

// UpdateTwilioAccountRequest contains parameters for updating a Twilio account.
type UpdateTwilioAccountRequest struct {
	Name      *string `json:"name,omitempty"`
	AuthToken *string `json:"auth_token,omitempty"`
	TrunkSID  *string `json:"trunk_sid,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// ListTwilioAccountsResponse contains the response from listing Twilio accounts.
type ListTwilioAccountsResponse struct {
	Accounts []TwilioAccount `json:"accounts"`
	Total    int             `json:"total,omitempty"`
}

// CreateTwilioAccount connects a custom Twilio account.
func (c *Client) CreateTwilioAccount(ctx context.Context, req *CreateTwilioAccountRequest) (*TwilioAccount, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.AccountSID == "" {
		return nil, fmt.Errorf("account_sid is required")
	}
	if req.AuthToken == "" {
		return nil, fmt.Errorf("auth_token is required")
	}

	var account TwilioAccount
	if err := c.request(ctx, "POST", "/enterprise/twilio", req, &account); err != nil {
		return nil, err
	}

	c.logger.Info("twilio account connected",
		zap.String("id", account.ID),
		zap.String("name", account.Name),
	)

	return &account, nil
}

// GetTwilioAccount retrieves a specific Twilio account.
func (c *Client) GetTwilioAccount(ctx context.Context, accountID string) (*TwilioAccount, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	var account TwilioAccount
	if err := c.request(ctx, "GET", "/enterprise/twilio/"+accountID, nil, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// ListTwilioAccounts retrieves all connected Twilio accounts.
func (c *Client) ListTwilioAccounts(ctx context.Context) ([]TwilioAccount, error) {
	var resp ListTwilioAccountsResponse
	if err := c.request(ctx, "GET", "/enterprise/twilio", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Accounts, nil
}

// UpdateTwilioAccount updates a Twilio account configuration.
func (c *Client) UpdateTwilioAccount(ctx context.Context, accountID string, req *UpdateTwilioAccountRequest) (*TwilioAccount, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	var account TwilioAccount
	if err := c.request(ctx, "PATCH", "/enterprise/twilio/"+accountID, req, &account); err != nil {
		return nil, err
	}

	c.logger.Info("twilio account updated", zap.String("id", accountID))
	return &account, nil
}

// DeleteTwilioAccount removes a Twilio account connection.
func (c *Client) DeleteTwilioAccount(ctx context.Context, accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account_id is required")
	}

	if err := c.request(ctx, "DELETE", "/enterprise/twilio/"+accountID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("twilio account deleted", zap.String("id", accountID))
	return nil
}

// VerifyTwilioAccount tests the Twilio account credentials.
func (c *Client) VerifyTwilioAccount(ctx context.Context, accountID string) (bool, error) {
	if accountID == "" {
		return false, fmt.Errorf("account_id is required")
	}

	var resp struct {
		Verified bool   `json:"verified"`
		Error    string `json:"error,omitempty"`
	}
	if err := c.request(ctx, "POST", "/enterprise/twilio/"+accountID+"/verify", nil, &resp); err != nil {
		return false, err
	}

	return resp.Verified, nil
}

// ========== SIP Integration ==========

// SIPTrunk represents a SIP trunk configuration.
type SIPTrunk struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Domain           string            `json:"domain"`
	Host             string            `json:"host"`
	Port             int               `json:"port,omitempty"`
	Transport        string            `json:"transport,omitempty"` // UDP, TCP, TLS
	Username         string            `json:"username,omitempty"`
	Password         string            `json:"password,omitempty"` // Only on create
	Codecs           []string          `json:"codecs,omitempty"`
	MaxChannels      int               `json:"max_channels,omitempty"`
	OutboundEnabled  bool              `json:"outbound_enabled"`
	InboundEnabled   bool              `json:"inbound_enabled"`
	Authenticator    string            `json:"authenticator,omitempty"` // IP, digest, none
	AllowedIPs       []string          `json:"allowed_ips,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	IsActive         bool              `json:"is_active"`
	Status           string            `json:"status,omitempty"` // connected, disconnected, error
	CreatedAt        time.Time         `json:"created_at,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at,omitempty"`
}

// CreateSIPTrunkRequest contains parameters for creating a SIP trunk.
type CreateSIPTrunkRequest struct {
	Name            string            `json:"name"`
	Domain          string            `json:"domain"`
	Host            string            `json:"host"`
	Port            int               `json:"port,omitempty"`
	Transport       string            `json:"transport,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`
	Codecs          []string          `json:"codecs,omitempty"`
	MaxChannels     int               `json:"max_channels,omitempty"`
	OutboundEnabled bool              `json:"outbound_enabled,omitempty"`
	InboundEnabled  bool              `json:"inbound_enabled,omitempty"`
	Authenticator   string            `json:"authenticator,omitempty"`
	AllowedIPs      []string          `json:"allowed_ips,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// UpdateSIPTrunkRequest contains parameters for updating a SIP trunk.
type UpdateSIPTrunkRequest struct {
	Name            *string           `json:"name,omitempty"`
	Host            *string           `json:"host,omitempty"`
	Port            *int              `json:"port,omitempty"`
	Transport       *string           `json:"transport,omitempty"`
	Username        *string           `json:"username,omitempty"`
	Password        *string           `json:"password,omitempty"`
	Codecs          []string          `json:"codecs,omitempty"`
	MaxChannels     *int              `json:"max_channels,omitempty"`
	OutboundEnabled *bool             `json:"outbound_enabled,omitempty"`
	InboundEnabled  *bool             `json:"inbound_enabled,omitempty"`
	AllowedIPs      []string          `json:"allowed_ips,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	IsActive        *bool             `json:"is_active,omitempty"`
}

// ListSIPTrunksResponse contains the response from listing SIP trunks.
type ListSIPTrunksResponse struct {
	Trunks []SIPTrunk `json:"trunks"`
	Total  int        `json:"total,omitempty"`
}

// SIPTrunkStats contains usage statistics for a SIP trunk.
type SIPTrunkStats struct {
	TrunkID          string    `json:"trunk_id"`
	ActiveChannels   int       `json:"active_channels"`
	TotalCalls       int       `json:"total_calls"`
	SuccessfulCalls  int       `json:"successful_calls"`
	FailedCalls      int       `json:"failed_calls"`
	TotalMinutes     float64   `json:"total_minutes"`
	AvgCallDuration  float64   `json:"avg_call_duration"`
	Period           string    `json:"period"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
}

// CreateSIPTrunk creates a new SIP trunk.
func (c *Client) CreateSIPTrunk(ctx context.Context, req *CreateSIPTrunkRequest) (*SIPTrunk, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	if req.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	var trunk SIPTrunk
	if err := c.request(ctx, "POST", "/enterprise/sip", req, &trunk); err != nil {
		return nil, err
	}

	c.logger.Info("sip trunk created",
		zap.String("id", trunk.ID),
		zap.String("name", trunk.Name),
	)

	return &trunk, nil
}

// GetSIPTrunk retrieves a specific SIP trunk.
func (c *Client) GetSIPTrunk(ctx context.Context, trunkID string) (*SIPTrunk, error) {
	if trunkID == "" {
		return nil, fmt.Errorf("trunk_id is required")
	}

	var trunk SIPTrunk
	if err := c.request(ctx, "GET", "/enterprise/sip/"+trunkID, nil, &trunk); err != nil {
		return nil, err
	}

	return &trunk, nil
}

// ListSIPTrunks retrieves all SIP trunks.
func (c *Client) ListSIPTrunks(ctx context.Context) ([]SIPTrunk, error) {
	var resp ListSIPTrunksResponse
	if err := c.request(ctx, "GET", "/enterprise/sip", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Trunks, nil
}

// UpdateSIPTrunk updates a SIP trunk configuration.
func (c *Client) UpdateSIPTrunk(ctx context.Context, trunkID string, req *UpdateSIPTrunkRequest) (*SIPTrunk, error) {
	if trunkID == "" {
		return nil, fmt.Errorf("trunk_id is required")
	}

	var trunk SIPTrunk
	if err := c.request(ctx, "PATCH", "/enterprise/sip/"+trunkID, req, &trunk); err != nil {
		return nil, err
	}

	c.logger.Info("sip trunk updated", zap.String("id", trunkID))
	return &trunk, nil
}

// DeleteSIPTrunk deletes a SIP trunk.
func (c *Client) DeleteSIPTrunk(ctx context.Context, trunkID string) error {
	if trunkID == "" {
		return fmt.Errorf("trunk_id is required")
	}

	if err := c.request(ctx, "DELETE", "/enterprise/sip/"+trunkID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("sip trunk deleted", zap.String("id", trunkID))
	return nil
}

// TestSIPTrunk tests connectivity to a SIP trunk.
func (c *Client) TestSIPTrunk(ctx context.Context, trunkID string) (bool, error) {
	if trunkID == "" {
		return false, fmt.Errorf("trunk_id is required")
	}

	var resp struct {
		Connected bool   `json:"connected"`
		Latency   int    `json:"latency_ms,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	if err := c.request(ctx, "POST", "/enterprise/sip/"+trunkID+"/test", nil, &resp); err != nil {
		return false, err
	}

	return resp.Connected, nil
}

// GetSIPTrunkStats retrieves statistics for a SIP trunk.
func (c *Client) GetSIPTrunkStats(ctx context.Context, trunkID, period string) (*SIPTrunkStats, error) {
	if trunkID == "" {
		return nil, fmt.Errorf("trunk_id is required")
	}

	path := "/enterprise/sip/" + trunkID + "/stats"
	if period != "" {
		path += "?period=" + period
	}

	var stats SIPTrunkStats
	if err := c.request(ctx, "GET", path, nil, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ========== Custom Dialing Pools ==========

// DialingPool represents a pool of phone numbers for outbound calls.
type DialingPool struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	PhoneNumbers   []PoolNumber   `json:"phone_numbers,omitempty"`
	Strategy       string         `json:"strategy"` // round_robin, random, weighted, least_used
	MaxConcurrent  int            `json:"max_concurrent,omitempty"`
	CooldownPeriod int            `json:"cooldown_period,omitempty"` // seconds
	LocalPresence  bool           `json:"local_presence"`           // Match caller ID to recipient area code
	IsActive       bool           `json:"is_active"`
	Stats          *DialingPoolStats `json:"stats,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
}

// PoolNumber represents a phone number in a dialing pool.
type PoolNumber struct {
	PhoneNumber   string    `json:"phone_number"`
	Weight        int       `json:"weight,omitempty"`
	AreaCode      string    `json:"area_code,omitempty"`
	Region        string    `json:"region,omitempty"`
	MaxConcurrent int       `json:"max_concurrent,omitempty"`
	IsActive      bool      `json:"is_active"`
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
	TotalCalls    int       `json:"total_calls,omitempty"`
	AnswerRate    float64   `json:"answer_rate,omitempty"`
}

// DialingPoolStats contains statistics for a dialing pool.
type DialingPoolStats struct {
	TotalCalls      int     `json:"total_calls"`
	AnsweredCalls   int     `json:"answered_calls"`
	AnswerRate      float64 `json:"answer_rate"`
	AvgRingDuration float64 `json:"avg_ring_duration"`
	ActiveCalls     int     `json:"active_calls"`
}

// CreateDialingPoolRequest contains parameters for creating a dialing pool.
type CreateDialingPoolRequest struct {
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	PhoneNumbers   []PoolNumber `json:"phone_numbers,omitempty"`
	Strategy       string       `json:"strategy,omitempty"`
	MaxConcurrent  int          `json:"max_concurrent,omitempty"`
	CooldownPeriod int          `json:"cooldown_period,omitempty"`
	LocalPresence  bool         `json:"local_presence,omitempty"`
}

// UpdateDialingPoolRequest contains parameters for updating a dialing pool.
type UpdateDialingPoolRequest struct {
	Name           *string      `json:"name,omitempty"`
	Description    *string      `json:"description,omitempty"`
	Strategy       *string      `json:"strategy,omitempty"`
	MaxConcurrent  *int         `json:"max_concurrent,omitempty"`
	CooldownPeriod *int         `json:"cooldown_period,omitempty"`
	LocalPresence  *bool        `json:"local_presence,omitempty"`
	IsActive       *bool        `json:"is_active,omitempty"`
}

// ListDialingPoolsResponse contains the response from listing dialing pools.
type ListDialingPoolsResponse struct {
	Pools []DialingPool `json:"pools"`
	Total int           `json:"total,omitempty"`
}

// CreateDialingPool creates a new dialing pool.
func (c *Client) CreateDialingPool(ctx context.Context, req *CreateDialingPoolRequest) (*DialingPool, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if req.Strategy == "" {
		req.Strategy = "round_robin"
	}

	var pool DialingPool
	if err := c.request(ctx, "POST", "/enterprise/dialing-pools", req, &pool); err != nil {
		return nil, err
	}

	c.logger.Info("dialing pool created",
		zap.String("id", pool.ID),
		zap.String("name", pool.Name),
	)

	return &pool, nil
}

// GetDialingPool retrieves a specific dialing pool.
func (c *Client) GetDialingPool(ctx context.Context, poolID string) (*DialingPool, error) {
	if poolID == "" {
		return nil, fmt.Errorf("pool_id is required")
	}

	var pool DialingPool
	if err := c.request(ctx, "GET", "/enterprise/dialing-pools/"+poolID, nil, &pool); err != nil {
		return nil, err
	}

	return &pool, nil
}

// ListDialingPools retrieves all dialing pools.
func (c *Client) ListDialingPools(ctx context.Context) ([]DialingPool, error) {
	var resp ListDialingPoolsResponse
	if err := c.request(ctx, "GET", "/enterprise/dialing-pools", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Pools, nil
}

// UpdateDialingPool updates a dialing pool configuration.
func (c *Client) UpdateDialingPool(ctx context.Context, poolID string, req *UpdateDialingPoolRequest) (*DialingPool, error) {
	if poolID == "" {
		return nil, fmt.Errorf("pool_id is required")
	}

	var pool DialingPool
	if err := c.request(ctx, "PATCH", "/enterprise/dialing-pools/"+poolID, req, &pool); err != nil {
		return nil, err
	}

	c.logger.Info("dialing pool updated", zap.String("id", poolID))
	return &pool, nil
}

// DeleteDialingPool deletes a dialing pool.
func (c *Client) DeleteDialingPool(ctx context.Context, poolID string) error {
	if poolID == "" {
		return fmt.Errorf("pool_id is required")
	}

	if err := c.request(ctx, "DELETE", "/enterprise/dialing-pools/"+poolID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("dialing pool deleted", zap.String("id", poolID))
	return nil
}

// AddNumberToPool adds a phone number to a dialing pool.
func (c *Client) AddNumberToPool(ctx context.Context, poolID string, number *PoolNumber) error {
	if poolID == "" {
		return fmt.Errorf("pool_id is required")
	}
	if number == nil || number.PhoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}

	if err := c.request(ctx, "POST", "/enterprise/dialing-pools/"+poolID+"/numbers", number, nil); err != nil {
		return err
	}

	c.logger.Info("number added to pool",
		zap.String("pool_id", poolID),
		zap.String("phone_number", number.PhoneNumber),
	)
	return nil
}

// RemoveNumberFromPool removes a phone number from a dialing pool.
func (c *Client) RemoveNumberFromPool(ctx context.Context, poolID, phoneNumber string) error {
	if poolID == "" {
		return fmt.Errorf("pool_id is required")
	}
	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}

	if err := c.request(ctx, "DELETE", "/enterprise/dialing-pools/"+poolID+"/numbers/"+phoneNumber, nil, nil); err != nil {
		return err
	}

	c.logger.Info("number removed from pool",
		zap.String("pool_id", poolID),
		zap.String("phone_number", phoneNumber),
	)
	return nil
}

// SetNumberWeight updates the weight of a number in a pool.
func (c *Client) SetNumberWeight(ctx context.Context, poolID, phoneNumber string, weight int) error {
	if poolID == "" {
		return fmt.Errorf("pool_id is required")
	}
	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}

	req := map[string]int{"weight": weight}
	return c.request(ctx, "PATCH", "/enterprise/dialing-pools/"+poolID+"/numbers/"+phoneNumber, req, nil)
}

// GetDialingPoolStats retrieves statistics for a dialing pool.
func (c *Client) GetDialingPoolStats(ctx context.Context, poolID string) (*DialingPoolStats, error) {
	if poolID == "" {
		return nil, fmt.Errorf("pool_id is required")
	}

	var stats DialingPoolStats
	if err := c.request(ctx, "GET", "/enterprise/dialing-pools/"+poolID+"/stats", nil, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ========== Organization Management ==========

// Organization represents a Bland organization/workspace.
type Organization struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan,omitempty"`
	Seats        int       `json:"seats,omitempty"`
	UsedSeats    int       `json:"used_seats,omitempty"`
	Features     []string  `json:"features,omitempty"`
	BillingEmail string    `json:"billing_email,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// OrganizationMember represents a member of an organization.
type OrganizationMember struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Role      string    `json:"role"` // owner, admin, member, viewer
	Status    string    `json:"status,omitempty"`
	JoinedAt  time.Time `json:"joined_at,omitempty"`
	LastLogin time.Time `json:"last_login,omitempty"`
}

// GetOrganization retrieves the current organization details.
func (c *Client) GetOrganization(ctx context.Context) (*Organization, error) {
	var org Organization
	if err := c.request(ctx, "GET", "/organization", nil, &org); err != nil {
		return nil, err
	}

	return &org, nil
}

// ListOrganizationMembers retrieves all members of the organization.
func (c *Client) ListOrganizationMembers(ctx context.Context) ([]OrganizationMember, error) {
	var resp struct {
		Members []OrganizationMember `json:"members"`
	}
	if err := c.request(ctx, "GET", "/organization/members", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Members, nil
}

// InviteOrganizationMember invites a new member to the organization.
func (c *Client) InviteOrganizationMember(ctx context.Context, email, role string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if role == "" {
		role = "member"
	}

	req := map[string]string{
		"email": email,
		"role":  role,
	}

	if err := c.request(ctx, "POST", "/organization/members/invite", req, nil); err != nil {
		return err
	}

	c.logger.Info("organization member invited",
		zap.String("email", email),
		zap.String("role", role),
	)
	return nil
}

// RemoveOrganizationMember removes a member from the organization.
func (c *Client) RemoveOrganizationMember(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("member_id is required")
	}

	if err := c.request(ctx, "DELETE", "/organization/members/"+memberID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("organization member removed", zap.String("id", memberID))
	return nil
}

// UpdateMemberRole updates a member's role in the organization.
func (c *Client) UpdateMemberRole(ctx context.Context, memberID, role string) error {
	if memberID == "" {
		return fmt.Errorf("member_id is required")
	}
	if role == "" {
		return fmt.Errorf("role is required")
	}

	req := map[string]string{"role": role}
	return c.request(ctx, "PATCH", "/organization/members/"+memberID, req, nil)
}
