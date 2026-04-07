package providercredentials

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/provider"
)

// CredentialsService provides operations for managing provider credentials
type CredentialsService struct {
	repo    CredentialsRepo
	adapter *provider.ProviderAdapter
}

// NewService creates a new credentials service
func NewService(repo CredentialsRepo, adapter *provider.ProviderAdapter) *CredentialsService {
	return &CredentialsService{
		repo:    repo,
		adapter: adapter,
	}
}

// SetProviderConfig validates and stores credentials for a provider
// The provider is looked up in the registry and its ValidateConfig is called
// Returns ErrInvalidInput if provider not found or config invalid
func (s *CredentialsService) SetProviderConfig(ctx context.Context, tenantID uuid.UUID,
	providerName string, config map[string]string) error {

	if tenantID == uuid.Nil {
		return fmt.Errorf("set provider config: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return fmt.Errorf("set provider config: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}
	if len(config) == 0 {
		return fmt.Errorf("set provider config: config cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Validate that no config keys or values are empty strings
	for k, v := range config {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			return fmt.Errorf("set provider config: config keys and values must be non-empty strings: %w", errors.ErrInvalidInput)
		}
	}

	// Step 1: Verify provider is registered in the adapter
	p, err := s.adapter.LookupProvider(providerName)
	if err != nil {
		return fmt.Errorf("set provider config: provider %q not registered: %w", providerName, errors.ErrInvalidInput)
	}

	// Step 2: Validate config with the provider
	if err := p.ValidateConfig(config); err != nil {
		return fmt.Errorf("set provider config: invalid config for %s: %w", providerName, errors.ErrInvalidInput)
	}

	// Step 3: Store in database
	if err := s.repo.Set(ctx, tenantID, providerName, config); err != nil {
		return fmt.Errorf("set provider config: storing credentials: %w", err)
	}

	// Step 4: Audit log (do NOT log config values)
	slog.Info("provider credentials configured",
		"tenant_id", tenantID.String(),
		"provider", providerName,
	)

	return nil
}

// GetProviderConfig retrieves credentials for a specific provider
// Returns ErrInvalidInput if credentials not found (privacy: doesn't leak existence)
func (s *CredentialsService) GetProviderConfig(ctx context.Context, tenantID uuid.UUID,
	providerName string) (map[string]string, error) {

	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("get provider config: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return nil, fmt.Errorf("get provider config: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}

	config, err := s.repo.Get(ctx, tenantID, providerName)
	if err != nil {
		// Return ErrInvalidInput instead of ErrNotFound
		// This doesn't leak whether credentials exist or not
		return nil, fmt.Errorf("get provider config: provider not configured for this tenant: %w", errors.ErrInvalidInput)
	}

	return config, nil
}

// ValidateAndFetch verifies provider is registered, then fetches config
// This is a convenience method for the payment service
// Returns ErrInvalidInput if provider not registered or config not found
func (s *CredentialsService) ValidateAndFetch(ctx context.Context, tenantID uuid.UUID,
	providerName string) (map[string]string, error) {

	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("validate and fetch: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return nil, fmt.Errorf("validate and fetch: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Step 1: Verify provider is registered
	_, err := s.adapter.LookupProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("validate and fetch: provider %q not registered: %w", providerName, errors.ErrInvalidInput)
	}

	// Step 2: Fetch config from database
	config, err := s.repo.Get(ctx, tenantID, providerName)
	if err != nil {
		// Return ErrInvalidInput (privacy)
		return nil, fmt.Errorf("validate and fetch: provider not configured: %w", errors.ErrInvalidInput)
	}

	return config, nil
}

// ListProviders returns all configured providers for a tenant
func (s *CredentialsService) ListProviders(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("list providers: %w", errors.ErrMissingTenantID)
	}

	configs, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	providers := make([]string, len(configs))
	for i, cfg := range configs {
		providers[i] = cfg.ProviderName
	}

	return providers, nil
}

// DeleteProviderConfig removes credentials for a provider
func (s *CredentialsService) DeleteProviderConfig(ctx context.Context, tenantID uuid.UUID,
	providerName string) error {

	if tenantID == uuid.Nil {
		return fmt.Errorf("delete provider config: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return fmt.Errorf("delete provider config: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}

	if err := s.repo.Delete(ctx, tenantID, providerName); err != nil {
		return fmt.Errorf("delete provider config: %w", err)
	}

	// Audit log
	slog.Info("provider credentials deleted",
		"tenant_id", tenantID.String(),
		"provider", providerName,
	)

	return nil
}
