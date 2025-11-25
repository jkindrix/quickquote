// Package config provides application configuration management using Viper.
// It supports loading from environment variables, config files, and defaults.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Bland     BlandConfig
	Anthropic AnthropicConfig
	Auth      AuthConfig
	App       AppConfig
	Log       LogConfig
	RateLimit RateLimitConfig
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

// BlandConfig holds Bland AI API settings.
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
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
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

	// Bland AI defaults
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
}

// Validate checks that all required configuration values are present.
func (c *Config) Validate() error {
	var missing []string

	if c.Database.Password == "" {
		missing = append(missing, "DATABASE_PASSWORD")
	}
	if c.Bland.APIKey == "" {
		missing = append(missing, "BLAND_API_KEY")
	}
	if c.Bland.InboundNumber == "" {
		missing = append(missing, "BLAND_INBOUND_NUMBER")
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
