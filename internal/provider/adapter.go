package provider

import (
	"context"
	"fmt"
)

// ProviderAdapter is the single point of contact between domain logic and external payment providers
// It resolves the correct connector from the registry and normalizes responses into domain types
type ProviderAdapter struct {
	registry *Registry
}

// NewAdapter creates a new provider adapter with the given registry
func NewAdapter(registry *Registry) *ProviderAdapter {
	return &ProviderAdapter{registry: registry}
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
