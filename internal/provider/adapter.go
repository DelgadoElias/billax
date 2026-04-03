package provider

import (
	"context"
	"fmt"
)

// ProviderAdapter is the single point of contact between domain logic and external payment providers
// It resolves the correct connector from the registry and normalizes responses into domain types
type ProviderAdapter struct {
	registry *Registry
	yamlCaps CapabilitiesConfig
}

// NewAdapter creates a new provider adapter with the given registry and capability config
func NewAdapter(registry *Registry, yamlCaps CapabilitiesConfig) *ProviderAdapter {
	return &ProviderAdapter{registry: registry, yamlCaps: yamlCaps}
}

// CreateCharge looks up the named provider, calls CreateCharge, returns normalized ChargeResult
// If the provider is not registered, returns ErrProviderNotFound
func (a *ProviderAdapter) CreateCharge(ctx context.Context, providerName string, req ChargeRequest) (*ChargeResult, error) {
	provider, err := a.registry.Lookup(providerName)
	if err != nil {
		return nil, err
	}

	result, err := provider.CreateCharge(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("adapter.CreateCharge via %s: %w", providerName, err)
	}

	return result, nil
}

// RefundCharge routes the refund to the correct connector
// If the provider is not registered, returns ErrProviderNotFound
func (a *ProviderAdapter) RefundCharge(ctx context.Context, providerName string, chargeID string, amount int64, config map[string]string) (*RefundResult, error) {
	provider, err := a.registry.Lookup(providerName)
	if err != nil {
		return nil, err
	}

	result, err := provider.RefundCharge(ctx, chargeID, amount)
	if err != nil {
		return nil, fmt.Errorf("adapter.RefundCharge via %s: %w", providerName, err)
	}

	return result, nil
}

// HandleWebhook routes an inbound webhook to the correct connector for signature validation
// and event parsing, then returns a normalized WebhookEvent
// If the provider is not registered, returns ErrProviderNotFound
func (a *ProviderAdapter) HandleWebhook(ctx context.Context, providerName string, payload []byte, signature string) (*WebhookEvent, error) {
	provider, err := a.registry.Lookup(providerName)
	if err != nil {
		return nil, err
	}

	event, err := provider.HandleWebhook(ctx, payload, signature)
	if err != nil {
		return nil, fmt.Errorf("adapter.HandleWebhook via %s: %w", providerName, err)
	}

	return event, nil
}

// ValidateConfig validates provider configuration for a given provider
// If the provider is not registered, returns ErrProviderNotFound
func (a *ProviderAdapter) ValidateConfig(ctx context.Context, providerName string, config map[string]string) error {
	provider, err := a.registry.Lookup(providerName)
	if err != nil {
		return err
	}

	if err := provider.ValidateConfig(config); err != nil {
		return fmt.Errorf("adapter.ValidateConfig via %s: %w", providerName, err)
	}

	return nil
}

// GetCapabilities returns the effective capabilities for a provider name.
// Resolution order:
// 1. Look up the connector in the registry.
// 2. Start with the connector's self-declared Capabilities().
// 3. If a YAML entry exists for this provider name, its booleans override the connector's.
// If the provider is not registered, returns defaultCapabilities (safe fallback).
func (a *ProviderAdapter) GetCapabilities(providerName string) ProviderCapabilities {
	p, err := a.registry.Lookup(providerName)
	if err != nil {
		return defaultCapabilities
	}
	caps := p.Capabilities() // connector self-declares

	// YAML gates override
	if entry, ok := a.yamlCaps[providerName]; ok {
		caps = entry
	}

	return caps
}

// LookupProvider returns a registered payment provider by name
// Used by the credentials service to validate provider existence
// If the provider is not registered, returns ErrProviderNotFound
func (a *ProviderAdapter) LookupProvider(providerName string) (PaymentProvider, error) {
	return a.registry.Lookup(providerName)
}
