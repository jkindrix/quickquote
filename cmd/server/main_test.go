package main

import (
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/config"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
)

func TestInitLogger_Development(t *testing.T) {
	// Save original env
	original := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", original)

	// Set to non-production
	os.Setenv("APP_ENV", "development")

	logger, err := initLogger()
	if err != nil {
		t.Fatalf("initLogger() error = %v", err)
	}
	defer func() { _ = logger.Sync() }()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Development logger should be able to log debug messages
	// Just verify it doesn't panic
	logger.Debug("test debug message")
	logger.Info("test info message")
}

func TestInitLogger_Production(t *testing.T) {
	// Save original env
	original := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", original)

	// Set to production
	os.Setenv("APP_ENV", "production")

	logger, err := initLogger()
	if err != nil {
		t.Fatalf("initLogger() error = %v", err)
	}
	defer func() { _ = logger.Sync() }()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Production logger should work
	logger.Info("test info message")
}

func TestInitLogger_EmptyEnv(t *testing.T) {
	// Save original env
	original := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", original)

	// Clear env
	os.Unsetenv("APP_ENV")

	logger, err := initLogger()
	if err != nil {
		t.Fatalf("initLogger() error = %v", err)
	}
	defer func() { _ = logger.Sync() }()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestInitVoiceProviders_NoProviders(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland:   config.BlandProviderConfig{Enabled: false},
			Vapi:    config.VapiProviderConfig{Enabled: false},
			Retell:  config.RetellProviderConfig{Enabled: false},
		},
		Bland: config.BlandConfig{APIKey: ""},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// No providers should be registered
	providers := registry.GetAll()
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

func TestInitVoiceProviders_BlandEnabled(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland: config.BlandProviderConfig{
				Enabled:       true,
				APIKey:        "test-bland-key",
				WebhookSecret: "test-secret",
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Bland should be registered
	providers := registry.GetAll()
	found := false
	for _, p := range providers {
		if p.GetName() == voiceprovider.ProviderBland {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Bland provider to be registered")
	}
}

func TestInitVoiceProviders_LegacyBlandConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland:   config.BlandProviderConfig{Enabled: false},
		},
		Bland: config.BlandConfig{
			APIKey:        "legacy-bland-key",
			WebhookSecret: "legacy-secret",
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Bland should be registered via legacy config
	providers := registry.GetAll()
	found := false
	for _, p := range providers {
		if p.GetName() == voiceprovider.ProviderBland {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Bland provider to be registered via legacy config")
	}
}

func TestInitVoiceProviders_VapiEnabled(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "vapi",
			Vapi: config.VapiProviderConfig{
				Enabled:       true,
				APIKey:        "test-vapi-key",
				WebhookSecret: "test-secret",
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Vapi should be registered
	providers := registry.GetAll()
	found := false
	for _, p := range providers {
		if p.GetName() == voiceprovider.ProviderVapi {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Vapi provider to be registered")
	}
}

func TestInitVoiceProviders_RetellEnabled(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "retell",
			Retell: config.RetellProviderConfig{
				Enabled:       true,
				APIKey:        "test-retell-key",
				WebhookSecret: "test-secret",
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Retell should be registered
	providers := registry.GetAll()
	found := false
	for _, p := range providers {
		if p.GetName() == voiceprovider.ProviderRetell {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Retell provider to be registered")
	}
}

func TestInitVoiceProviders_AllEnabled(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland: config.BlandProviderConfig{
				Enabled: true,
				APIKey:  "test-bland-key",
			},
			Vapi: config.VapiProviderConfig{
				Enabled: true,
				APIKey:  "test-vapi-key",
			},
			Retell: config.RetellProviderConfig{
				Enabled: true,
				APIKey:  "test-retell-key",
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// All three providers should be registered
	providers := registry.GetAll()
	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}

	// Verify each provider type
	providerTypes := make(map[voiceprovider.ProviderType]bool)
	for _, p := range providers {
		providerTypes[p.GetName()] = true
	}

	if !providerTypes[voiceprovider.ProviderBland] {
		t.Error("expected Bland provider to be registered")
	}
	if !providerTypes[voiceprovider.ProviderVapi] {
		t.Error("expected Vapi provider to be registered")
	}
	if !providerTypes[voiceprovider.ProviderRetell] {
		t.Error("expected Retell provider to be registered")
	}
}

func TestInitVoiceProviders_DefaultPrimary(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "", // Empty, should default to "bland"
			Bland: config.BlandProviderConfig{
				Enabled: true,
				APIKey:  "test-bland-key",
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Primary should be bland by default
	primary, err := registry.GetPrimary()
	if err != nil {
		t.Fatalf("GetPrimary() error = %v", err)
	}
	if primary.GetName() != voiceprovider.ProviderBland {
		t.Errorf("expected primary to be bland, got %s", primary.GetName())
	}
}

func TestInitVoiceProviders_InvalidPrimary(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "nonexistent",
			Bland: config.BlandProviderConfig{
				Enabled: true,
				APIKey:  "test-bland-key",
			},
		},
	}

	// Should not panic even with invalid primary
	registry := initVoiceProviders(cfg, logger)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestInitVoiceProviders_VapiDisabledNoKey(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland: config.BlandProviderConfig{
				Enabled: true,
				APIKey:  "test-bland-key",
			},
			Vapi: config.VapiProviderConfig{
				Enabled: true,
				APIKey:  "", // Empty key should not register
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	// Only Bland should be registered
	providers := registry.GetAll()
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].GetName() != voiceprovider.ProviderBland {
		t.Error("expected only Bland provider to be registered")
	}
}

func TestInitVoiceProviders_RetellDisabledNoKey(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		VoiceProvider: config.VoiceProviderConfig{
			Primary: "bland",
			Bland: config.BlandProviderConfig{
				Enabled: true,
				APIKey:  "test-bland-key",
			},
			Retell: config.RetellProviderConfig{
				Enabled: true,
				APIKey:  "", // Empty key should not register
			},
		},
	}

	registry := initVoiceProviders(cfg, logger)

	// Only Bland should be registered
	providers := registry.GetAll()
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].GetName() != voiceprovider.ProviderBland {
		t.Error("expected only Bland provider to be registered")
	}
}
