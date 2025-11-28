// Package config provides application configuration management using Viper.
// It supports loading from environment variables, config files, and defaults.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	VoiceProvider VoiceProviderConfig
	Anthropic     AnthropicConfig
	Auth          AuthConfig
	App           AppConfig
	Log           LogConfig
	RateLimit     RateLimitConfig
	CallSettings  CallSettingsConfig

	// Backward compatibility - deprecated, use VoiceProvider.Bland instead
	Bland BlandConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host        string
	Port        int
	Environment string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	Name                  string
	SSLMode               string
	MaxConnections        int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

// ConnectionString returns a PostgreSQL connection string.
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// VoiceProviderConfig holds configuration for voice AI providers.
type VoiceProviderConfig struct {
	// Primary provider to use (bland, vapi, retell)
	Primary string

	// Bland AI configuration
	Bland BlandProviderConfig

	// Vapi configuration
	Vapi VapiProviderConfig

	// Retell configuration
	Retell RetellProviderConfig
}

// BlandProviderConfig holds Bland AI API settings.
type BlandProviderConfig struct {
	Enabled       bool
	APIKey        string
	InboundNumber string
	WebhookSecret string
	APIURL        string
}

// VapiProviderConfig holds Vapi API settings.
type VapiProviderConfig struct {
	Enabled       bool
	APIKey        string
	WebhookSecret string
	APIURL        string
}

// RetellProviderConfig holds Retell AI API settings.
type RetellProviderConfig struct {
	Enabled       bool
	APIKey        string
	WebhookSecret string
	APIURL        string
}

// BlandConfig holds Bland AI API settings (deprecated - for backward compatibility).
type BlandConfig struct {
	APIKey        string
	InboundNumber string
	WebhookSecret string
	APIURL        string
}

// AnthropicConfig holds Claude AI settings for quote generation.
type AnthropicConfig struct {
	APIKey string
	Model  string
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	SessionSecret   string
	SessionDuration time.Duration
}

// AppConfig holds general application settings.
type AppConfig struct {
	PublicURL string
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string
	Format string
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

// CallSettingsConfig holds inbound call configuration.
type CallSettingsConfig struct {
	// Business identity
	BusinessName string

	// Voice configuration
	Voice                 string
	VoiceStability        float64
	VoiceSimilarityBoost  float64
	VoiceStyle            float64
	VoiceSpeakerBoost     bool

	// Model configuration
	Model       string // "base" or "enhanced"
	Language    string
	Temperature float64

	// Conversation settings
	InterruptionThreshold int  // milliseconds (50-500)
	WaitForGreeting       bool
	NoiseCancellation     bool
	BackgroundTrack       string // "none", "office", "cafe", "restaurant"

	// Call limits
	MaxDurationMinutes int
	RecordCalls        bool

	// Quality preset (overrides individual settings if set)
	// Options: "default", "high_quality", "fast_response", "accessibility"
	QualityPreset string

	// Custom greeting (optional)
	CustomGreeting string

	// Project types offered (comma-separated)
	ProjectTypes string
}

// Load reads configuration from environment variables and config files.
// Environment variables take precedence over config file values.
func Load() (*Config, error) {
	v := viper.New()

	// Set config file options
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/quickquote")

	// Enable environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	// Try to read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		var configNotFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFoundErr) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Build config struct
	cfg := &Config{
		Server: ServerConfig{
			Host:        v.GetString("server.host"),
			Port:        v.GetInt("server.port"),
			Environment: v.GetString("server.env"),
		},
		Database: DatabaseConfig{
			Host:                  v.GetString("database.host"),
			Port:                  v.GetInt("database.port"),
			User:                  v.GetString("database.user"),
			Password:              v.GetString("database.password"),
			Name:                  v.GetString("database.name"),
			SSLMode:               v.GetString("database.sslmode"),
			MaxConnections:        v.GetInt("database.max_connections"),
			MaxIdleConnections:    v.GetInt("database.max_idle_connections"),
			ConnectionMaxLifetime: v.GetDuration("database.connection_max_lifetime"),
		},
		VoiceProvider: VoiceProviderConfig{
			Primary: v.GetString("voice_provider.primary"),
			Bland: BlandProviderConfig{
				Enabled:       v.GetBool("voice_provider.bland.enabled"),
				APIKey:        v.GetString("voice_provider.bland.api_key"),
				InboundNumber: v.GetString("voice_provider.bland.inbound_number"),
				WebhookSecret: v.GetString("voice_provider.bland.webhook_secret"),
				APIURL:        v.GetString("voice_provider.bland.api_url"),
			},
			Vapi: VapiProviderConfig{
				Enabled:       v.GetBool("voice_provider.vapi.enabled"),
				APIKey:        v.GetString("voice_provider.vapi.api_key"),
				WebhookSecret: v.GetString("voice_provider.vapi.webhook_secret"),
				APIURL:        v.GetString("voice_provider.vapi.api_url"),
			},
			Retell: RetellProviderConfig{
				Enabled:       v.GetBool("voice_provider.retell.enabled"),
				APIKey:        v.GetString("voice_provider.retell.api_key"),
				WebhookSecret: v.GetString("voice_provider.retell.webhook_secret"),
				APIURL:        v.GetString("voice_provider.retell.api_url"),
			},
		},
		// Backward compatibility - copy from legacy or new config
		Bland: BlandConfig{
			APIKey:        v.GetString("bland.api_key"),
			InboundNumber: v.GetString("bland.inbound_number"),
			WebhookSecret: v.GetString("bland.webhook_secret"),
			APIURL:        v.GetString("bland.api_url"),
		},
		Anthropic: AnthropicConfig{
			APIKey: v.GetString("anthropic.api_key"),
			Model:  v.GetString("anthropic.model"),
		},
		Auth: AuthConfig{
			SessionSecret:   v.GetString("session.secret"),
			SessionDuration: v.GetDuration("session.duration"),
		},
		App: AppConfig{
			PublicURL: v.GetString("app.public_url"),
		},
		Log: LogConfig{
			Level:  v.GetString("log.level"),
			Format: v.GetString("log.format"),
		},
		RateLimit: RateLimitConfig{
			Requests: v.GetInt("rate_limit.requests"),
			Window:   v.GetDuration("rate_limit.window"),
		},
		CallSettings: CallSettingsConfig{
			BusinessName:          v.GetString("call.business_name"),
			Voice:                 v.GetString("call.voice"),
			VoiceStability:        v.GetFloat64("call.voice_stability"),
			VoiceSimilarityBoost:  v.GetFloat64("call.voice_similarity_boost"),
			VoiceStyle:            v.GetFloat64("call.voice_style"),
			VoiceSpeakerBoost:     v.GetBool("call.voice_speaker_boost"),
			Model:                 v.GetString("call.model"),
			Language:              v.GetString("call.language"),
			Temperature:           v.GetFloat64("call.temperature"),
			InterruptionThreshold: v.GetInt("call.interruption_threshold"),
			WaitForGreeting:       v.GetBool("call.wait_for_greeting"),
			NoiseCancellation:     v.GetBool("call.noise_cancellation"),
			BackgroundTrack:       v.GetString("call.background_track"),
			MaxDurationMinutes:    v.GetInt("call.max_duration_minutes"),
			RecordCalls:           v.GetBool("call.record"),
			QualityPreset:         v.GetString("call.quality_preset"),
			CustomGreeting:        v.GetString("call.custom_greeting"),
			ProjectTypes:          v.GetString("call.project_types"),
		},
	}

	// Backward compatibility: if legacy Bland config is set but new config is not,
	// copy values to new structure
	if cfg.Bland.APIKey != "" && cfg.VoiceProvider.Bland.APIKey == "" {
		cfg.VoiceProvider.Primary = "bland"
		cfg.VoiceProvider.Bland.Enabled = true
		cfg.VoiceProvider.Bland.APIKey = cfg.Bland.APIKey
		cfg.VoiceProvider.Bland.InboundNumber = cfg.Bland.InboundNumber
		cfg.VoiceProvider.Bland.WebhookSecret = cfg.Bland.WebhookSecret
		cfg.VoiceProvider.Bland.APIURL = cfg.Bland.APIURL
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setDefaults configures default values for all settings.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.env", "development")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "quickquote")
	v.SetDefault("database.name", "quickquote")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_connections", 25)
	v.SetDefault("database.max_idle_connections", 5)
	v.SetDefault("database.connection_max_lifetime", "5m")

	// Voice provider defaults
	v.SetDefault("voice_provider.primary", "bland")
	v.SetDefault("voice_provider.bland.enabled", true)
	v.SetDefault("voice_provider.bland.api_url", "https://api.bland.ai/v1")
	v.SetDefault("voice_provider.vapi.enabled", false)
	v.SetDefault("voice_provider.vapi.api_url", "https://api.vapi.ai")
	v.SetDefault("voice_provider.retell.enabled", false)
	v.SetDefault("voice_provider.retell.api_url", "https://api.retellai.com")

	// Legacy Bland AI defaults (for backward compatibility)
	v.SetDefault("bland.api_url", "https://api.bland.ai/v1")

	// Anthropic defaults
	v.SetDefault("anthropic.model", "claude-sonnet-4-20250514")

	// Auth defaults
	v.SetDefault("session.duration", "24h")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// Rate limit defaults
	v.SetDefault("rate_limit.requests", 100)
	v.SetDefault("rate_limit.window", "1m")

	// Call settings defaults (optimized for quote collection)
	v.SetDefault("call.business_name", "QuickQuote")
	v.SetDefault("call.voice", "maya")
	v.SetDefault("call.voice_stability", 0.75)
	v.SetDefault("call.voice_similarity_boost", 0.80)
	v.SetDefault("call.voice_style", 0.3)
	v.SetDefault("call.voice_speaker_boost", true)
	v.SetDefault("call.model", "enhanced")
	v.SetDefault("call.language", "en-US")
	v.SetDefault("call.temperature", 0.6)
	v.SetDefault("call.interruption_threshold", 100)
	v.SetDefault("call.wait_for_greeting", true)
	v.SetDefault("call.noise_cancellation", true)
	v.SetDefault("call.background_track", "office")
	v.SetDefault("call.max_duration_minutes", 15)
	v.SetDefault("call.record", true)
	v.SetDefault("call.quality_preset", "default")
	v.SetDefault("call.project_types", "web_app,mobile_app,api,ecommerce,custom_software,integration")
}

// Validate checks that all required configuration values are present.
func (c *Config) Validate() error {
	var missing []string

	if c.Database.Password == "" {
		missing = append(missing, "DATABASE_PASSWORD")
	}

	// Validate at least one voice provider is configured
	hasVoiceProvider := false
	if c.VoiceProvider.Bland.Enabled && c.VoiceProvider.Bland.APIKey != "" {
		hasVoiceProvider = true
	}
	if c.VoiceProvider.Vapi.Enabled && c.VoiceProvider.Vapi.APIKey != "" {
		hasVoiceProvider = true
	}
	if c.VoiceProvider.Retell.Enabled && c.VoiceProvider.Retell.APIKey != "" {
		hasVoiceProvider = true
	}
	// Backward compatibility: check legacy Bland config
	if c.Bland.APIKey != "" {
		hasVoiceProvider = true
	}
	if !hasVoiceProvider {
		missing = append(missing, "VOICE_PROVIDER (at least one of BLAND_API_KEY, VAPI_API_KEY, or RETELL_API_KEY)")
	}

	if c.Anthropic.APIKey == "" {
		missing = append(missing, "ANTHROPIC_API_KEY")
	}
	if c.Auth.SessionSecret == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	if c.App.PublicURL == "" {
		missing = append(missing, "APP_PUBLIC_URL")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// GetProjectTypes returns the project types as a slice.
func (c *CallSettingsConfig) GetProjectTypes() []string {
	if c.ProjectTypes == "" {
		return []string{"web_app", "mobile_app", "api", "ecommerce", "custom_software", "integration"}
	}
	types := strings.Split(c.ProjectTypes, ",")
	for i := range types {
		types[i] = strings.TrimSpace(types[i])
	}
	return types
}

