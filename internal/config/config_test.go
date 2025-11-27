package config

import (
	"testing"
	"time"
)

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if got := cfg.ConnectionString(); got != expected {
		t.Errorf("ConnectionString() = %q, expected %q", got, expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with legacy bland",
			config: Config{
				Database:  DatabaseConfig{Password: "pass"},
				Bland:     BlandConfig{APIKey: "key", InboundNumber: "+1234567890"},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: false,
		},
		{
			name: "valid config with new voice provider config",
			config: Config{
				Database: DatabaseConfig{Password: "pass"},
				VoiceProvider: VoiceProviderConfig{
					Primary: "bland",
					Bland:   BlandProviderConfig{Enabled: true, APIKey: "key"},
				},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: false,
		},
		{
			name: "valid config with vapi provider",
			config: Config{
				Database: DatabaseConfig{Password: "pass"},
				VoiceProvider: VoiceProviderConfig{
					Primary: "vapi",
					Vapi:    VapiProviderConfig{Enabled: true, APIKey: "key"},
				},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: false,
		},
		{
			name: "valid config with retell provider",
			config: Config{
				Database: DatabaseConfig{Password: "pass"},
				VoiceProvider: VoiceProviderConfig{
					Primary: "retell",
					Retell:  RetellProviderConfig{Enabled: true, APIKey: "key"},
				},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: false,
		},
		{
			name: "missing database password",
			config: Config{
				Bland:     BlandConfig{APIKey: "key", InboundNumber: "+1234567890"},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: true,
		},
		{
			name: "missing voice provider",
			config: Config{
				Database:  DatabaseConfig{Password: "pass"},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: true,
		},
		{
			name: "voice provider enabled but no api key",
			config: Config{
				Database: DatabaseConfig{Password: "pass"},
				VoiceProvider: VoiceProviderConfig{
					Primary: "bland",
					Bland:   BlandProviderConfig{Enabled: true}, // No API key
				},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: true,
		},
		{
			name: "missing anthropic api key",
			config: Config{
				Database: DatabaseConfig{Password: "pass"},
				Bland:    BlandConfig{APIKey: "key", InboundNumber: "+1234567890"},
				Auth:     AuthConfig{SessionSecret: "secret"},
				App:      AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: true,
		},
		{
			name: "missing session secret",
			config: Config{
				Database:  DatabaseConfig{Password: "pass"},
				Bland:     BlandConfig{APIKey: "key", InboundNumber: "+1234567890"},
				Anthropic: AnthropicConfig{APIKey: "key"},
				App:       AppConfig{PublicURL: "http://localhost"},
			},
			wantErr: true,
		},
		{
			name: "missing public url",
			config: Config{
				Database:  DatabaseConfig{Password: "pass"},
				Bland:     BlandConfig{APIKey: "key", InboundNumber: "+1234567890"},
				Anthropic: AnthropicConfig{APIKey: "key"},
				Auth:      AuthConfig{SessionSecret: "secret"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Server: ServerConfig{Environment: tt.env}}
			if got := cfg.IsDevelopment(); got != tt.expected {
				t.Errorf("IsDevelopment() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Server: ServerConfig{Environment: tt.env}}
			if got := cfg.IsProduction(); got != tt.expected {
				t.Errorf("IsProduction() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestRateLimitConfig(t *testing.T) {
	cfg := RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
	}

	if cfg.Requests != 100 {
		t.Errorf("Requests = %d, expected 100", cfg.Requests)
	}
	if cfg.Window != time.Minute {
		t.Errorf("Window = %v, expected %v", cfg.Window, time.Minute)
	}
}
