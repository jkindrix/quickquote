package voiceprovider

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Registry holds all registered voice providers and allows dynamic lookup.
type Registry struct {
	providers map[ProviderType]Provider
	primary   ProviderType
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewRegistry creates a new provider registry.
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		providers: make(map[ProviderType]Provider),
		logger:    logger,
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.GetName()
	r.providers[name] = provider
	r.logger.Info("registered voice provider", zap.String("provider", string(name)))
}

// SetPrimary sets the primary provider for outbound operations.
func (r *Registry) SetPrimary(providerType ProviderType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[providerType]; !exists {
		return fmt.Errorf("provider %s not registered", providerType)
	}

	r.primary = providerType
	r.logger.Info("set primary voice provider", zap.String("provider", string(providerType)))
	return nil
}

// Get retrieves a provider by type.
func (r *Registry) Get(providerType ProviderType) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerType)
	}
	return provider, nil
}

// GetPrimary returns the primary provider.
func (r *Registry) GetPrimary() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.primary == "" {
		// If no primary set, return the first registered provider
		for _, provider := range r.providers {
			return provider, nil
		}
		return nil, fmt.Errorf("no providers registered")
	}

	provider, exists := r.providers[r.primary]
	if !exists {
		return nil, fmt.Errorf("primary provider %s not registered", r.primary)
	}
	return provider, nil
}

// GetByWebhookPath finds a provider by its webhook path.
func (r *Registry) GetByWebhookPath(path string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		if provider.GetWebhookPath() == path {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no provider registered for webhook path: %s", path)
}

// List returns all registered provider types.
func (r *Registry) List() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ProviderType, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}
	return types
}

// GetWebhookPaths returns all webhook paths from registered providers.
func (r *Registry) GetWebhookPaths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.providers))
	for _, provider := range r.providers {
		paths = append(paths, provider.GetWebhookPath())
	}
	return paths
}

// GetAll returns all registered providers.
func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// HasProvider checks if a provider is registered.
func (r *Registry) HasProvider(providerType ProviderType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[providerType]
	return exists
}

// IsEmpty returns true if no providers are registered.
func (r *Registry) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers) == 0
}

// ProviderStatus represents the health status of a voice provider.
type ProviderStatus struct {
	Name      ProviderType `json:"name"`
	Available bool         `json:"available"`
	IsPrimary bool         `json:"is_primary"`
	Message   string       `json:"message,omitempty"`
}

// HealthStatus returns the health status of all registered providers.
func (r *Registry) HealthStatus() []ProviderStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make([]ProviderStatus, 0, len(r.providers))
	for providerType, provider := range r.providers {
		status := ProviderStatus{
			Name:      providerType,
			Available: true, // Provider is available if registered
			IsPrimary: providerType == r.primary,
			Message:   fmt.Sprintf("webhook: %s", provider.GetWebhookPath()),
		}
		statuses = append(statuses, status)
	}
	return statuses
}

// PrimaryProviderName returns the name of the primary provider, or empty if none.
func (r *Registry) PrimaryProviderName() ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

// Count returns the number of registered providers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

// ProviderConfig is a generic configuration that can be used to create providers.
type ProviderConfig struct {
	Type          ProviderType
	APIKey        string
	WebhookSecret string
	APIURL        string
	// Provider-specific settings
	Extra map[string]interface{}
}
