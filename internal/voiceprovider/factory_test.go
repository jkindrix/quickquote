package voiceprovider

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
)

// mockProvider is a test implementation of the Provider interface.
type mockProvider struct {
	name        ProviderType
	webhookPath string
}

func (m *mockProvider) GetName() ProviderType {
	return m.name
}

func (m *mockProvider) ParseWebhook(r *http.Request) (*CallEvent, error) {
	return &CallEvent{
		Provider:       m.name,
		ProviderCallID: "mock-call-123",
		Status:         CallStatusCompleted,
	}, nil
}

func (m *mockProvider) ValidateWebhook(r *http.Request) bool {
	return true
}

func (m *mockProvider) GetWebhookPath() string {
	return m.webhookPath
}

func newMockProvider(name ProviderType, webhookPath string) *mockProvider {
	return &mockProvider{
		name:        name,
		webhookPath: webhookPath,
	}
}

func TestNewRegistry(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.providers == nil {
		t.Error("providers map is nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	registry.Register(bland)

	if len(registry.providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(registry.providers))
	}

	if _, ok := registry.providers[ProviderBland]; !ok {
		t.Error("bland provider not found in registry")
	}
}

func TestRegistry_Register_Multiple(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	registry.Register(newMockProvider(ProviderBland, "/webhook/bland"))
	registry.Register(newMockProvider(ProviderVapi, "/webhook/vapi"))
	registry.Register(newMockProvider(ProviderRetell, "/webhook/retell"))

	if len(registry.providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(registry.providers))
	}
}

func TestRegistry_Get(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	registry.Register(bland)

	provider, err := registry.Get(ProviderBland)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if provider.GetName() != ProviderBland {
		t.Errorf("provider name = %q, expected %q", provider.GetName(), ProviderBland)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	_, err := registry.Get(ProviderBland)
	if err == nil {
		t.Error("expected error for non-existent provider, got nil")
	}
}

func TestRegistry_SetPrimary(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	registry.Register(bland)

	err := registry.SetPrimary(ProviderBland)
	if err != nil {
		t.Fatalf("SetPrimary() error = %v", err)
	}

	if registry.primary != ProviderBland {
		t.Errorf("primary = %q, expected %q", registry.primary, ProviderBland)
	}
}

func TestRegistry_SetPrimary_NotRegistered(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	err := registry.SetPrimary(ProviderBland)
	if err == nil {
		t.Error("expected error for non-registered provider, got nil")
	}
}

func TestRegistry_GetPrimary(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	registry.Register(bland)
	registry.SetPrimary(ProviderBland)

	provider, err := registry.GetPrimary()
	if err != nil {
		t.Fatalf("GetPrimary() error = %v", err)
	}

	if provider.GetName() != ProviderBland {
		t.Errorf("primary provider name = %q, expected %q", provider.GetName(), ProviderBland)
	}
}

func TestRegistry_GetPrimary_NotSet(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	_, err := registry.GetPrimary()
	if err == nil {
		t.Error("expected error when no primary set, got nil")
	}
}

func TestRegistry_GetByWebhookPath(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	vapi := newMockProvider(ProviderVapi, "/webhook/vapi")
	registry.Register(bland)
	registry.Register(vapi)

	tests := []struct {
		path     string
		expected ProviderType
		wantErr  bool
	}{
		{"/webhook/bland", ProviderBland, false},
		{"/webhook/vapi", ProviderVapi, false},
		{"/webhook/unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			provider, err := registry.GetByWebhookPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByWebhookPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider.GetName() != tt.expected {
				t.Errorf("provider name = %q, expected %q", provider.GetName(), tt.expected)
			}
		})
	}
}

func TestRegistry_GetAll(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	registry.Register(newMockProvider(ProviderBland, "/webhook/bland"))
	registry.Register(newMockProvider(ProviderVapi, "/webhook/vapi"))

	providers := registry.GetAll()

	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestRegistry_GetAll_Empty(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	providers := registry.GetAll()

	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

func TestRegistry_HasProvider(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	bland := newMockProvider(ProviderBland, "/webhook/bland")
	registry.Register(bland)

	if !registry.HasProvider(ProviderBland) {
		t.Error("expected HasProvider(bland) to return true")
	}

	if registry.HasProvider(ProviderVapi) {
		t.Error("expected HasProvider(vapi) to return false")
	}
}

func TestRegistry_IsEmpty(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	if !registry.IsEmpty() {
		t.Error("expected IsEmpty() to return true for new registry")
	}

	registry.Register(newMockProvider(ProviderBland, "/webhook/bland"))

	if registry.IsEmpty() {
		t.Error("expected IsEmpty() to return false after registering provider")
	}
}
