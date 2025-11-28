package service

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/repository"
)

// SettingsService manages application settings.
type SettingsService struct {
	repo   *repository.SettingsRepository
	logger *zap.Logger

	// Cache for settings to avoid repeated DB queries
	cache    map[string]string
	cacheMu  sync.RWMutex
	cacheSet bool
}

// NewSettingsService creates a new settings service.
func NewSettingsService(repo *repository.SettingsRepository, logger *zap.Logger) *SettingsService {
	return &SettingsService{
		repo:   repo,
		logger: logger,
		cache:  make(map[string]string),
	}
}

// GetCallSettings retrieves all call-related settings as a typed struct.
func (s *SettingsService) GetCallSettings(ctx context.Context) (*domain.CallSettings, error) {
	settingsMap, err := s.getAllAsMap(ctx)
	if err != nil {
		return nil, err
	}

	return domain.NewCallSettingsFromMap(settingsMap), nil
}

// SaveCallSettings saves all call-related settings from a typed struct.
func (s *SettingsService) SaveCallSettings(ctx context.Context, settings *domain.CallSettings) error {
	settingsMap := settings.ToMap()

	if err := s.repo.SetMany(ctx, settingsMap); err != nil {
		return err
	}

	// Invalidate cache
	s.invalidateCache()

	s.logger.Info("call settings saved",
		zap.String("business_name", settings.BusinessName),
		zap.String("voice", settings.Voice),
		zap.String("model", settings.Model),
	)

	return nil
}

// Get retrieves a single setting value.
func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	// Check cache first
	s.cacheMu.RLock()
	if s.cacheSet {
		if v, ok := s.cache[key]; ok {
			s.cacheMu.RUnlock()
			return v, nil
		}
	}
	s.cacheMu.RUnlock()

	// Fetch from DB
	setting, err := s.repo.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", nil
	}

	return setting.Value, nil
}

// Set updates a single setting value.
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	if err := s.repo.Set(ctx, key, value); err != nil {
		return err
	}

	// Update cache
	s.cacheMu.Lock()
	if s.cacheSet {
		s.cache[key] = value
	}
	s.cacheMu.Unlock()

	s.logger.Info("setting updated", zap.String("key", key))

	return nil
}

// GetAllSettings retrieves all settings.
func (s *SettingsService) GetAllSettings(ctx context.Context) ([]*domain.Setting, error) {
	return s.repo.GetAll(ctx)
}

// getAllAsMap retrieves all settings as a map, using cache if available.
func (s *SettingsService) getAllAsMap(ctx context.Context) (map[string]string, error) {
	s.cacheMu.RLock()
	if s.cacheSet {
		// Return copy of cache
		result := make(map[string]string, len(s.cache))
		for k, v := range s.cache {
			result[k] = v
		}
		s.cacheMu.RUnlock()
		return result, nil
	}
	s.cacheMu.RUnlock()

	// Fetch from DB and populate cache
	settingsMap, err := s.repo.GetAsMap(ctx)
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.cache = settingsMap
	s.cacheSet = true
	s.cacheMu.Unlock()

	return settingsMap, nil
}

// invalidateCache clears the settings cache.
func (s *SettingsService) invalidateCache() {
	s.cacheMu.Lock()
	s.cache = make(map[string]string)
	s.cacheSet = false
	s.cacheMu.Unlock()
}

// RefreshCache forces a reload of the settings cache from the database.
func (s *SettingsService) RefreshCache(ctx context.Context) error {
	s.invalidateCache()
	_, err := s.getAllAsMap(ctx)
	return err
}

// GetPricingSettings retrieves pricing fallback settings as a typed struct.
func (s *SettingsService) GetPricingSettings(ctx context.Context) (*domain.PricingSettings, error) {
	settingsMap, err := s.getAllAsMap(ctx)
	if err != nil {
		return nil, err
	}

	return domain.NewPricingSettingsFromMap(settingsMap), nil
}
