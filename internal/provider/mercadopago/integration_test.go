package mercadopago

import (
	"context"
	"testing"

	"github.com/DelgadoElias/billax/internal/provider"
)

// TestAdapterIntegration verifies the connector works through the provider adapter
func TestAdapterIntegration(t *testing.T) {
	// Initialize provider registry with Mercado Pago
	registry := provider.NewRegistry()
	registry.Register(New())

	// Initialize adapter with empty YAML capabilities (uses defaults)
	adapter := provider.NewAdapter(registry, provider.CapabilitiesConfig{})

	// Test 1: Verify provider is registered
	p, err := registry.Lookup("mercadopago")
	if err != nil {
		t.Fatalf("Provider not registered: %v", err)
	}
	if p.GetProviderName() != "mercadopago" {
		t.Errorf("Provider name = %s, want mercadopago", p.GetProviderName())
	}
	t.Log("✓ Provider registered: mercadopago")

	// Test 2: Verify capabilities via adapter
	caps := adapter.GetCapabilities("mercadopago")
	if !caps.Plans {
		t.Error("Capabilities.Plans = false, want true")
	}
	if caps.PayPerUse {
		t.Error("Capabilities.PayPerUse = true, want false")
	}
	t.Logf("✓ Capabilities: Plans=%v, PayPerUse=%v", caps.Plans, caps.PayPerUse)

	// Test 3: Verify config validation through adapter
	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}
	err = adapter.ValidateConfig(context.Background(), "mercadopago", config)
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}
	t.Log("✓ Config validation passed")

	// Test 4: Verify RefundCharge returns not supported
	result, err := adapter.RefundCharge(context.Background(), "mercadopago", "123", 100, config)
	if result != nil {
		t.Errorf("RefundCharge() returned non-nil result: %v", result)
	}
	if err == nil {
		t.Error("RefundCharge() should return error (not supported)")
	}
	t.Logf("✓ RefundCharge correctly returns not supported")

	// Test 5: Verify adapter returns correct error for unknown provider
	caps = adapter.GetCapabilities("unknown_provider")
	// Should return default capabilities (Plans=true, PayPerUse=false)
	if !caps.Plans {
		t.Error("Default capabilities.Plans = false, want true")
	}
	t.Log("✓ Adapter returns safe defaults for unknown provider")

	t.Log("\nAll adapter integration tests passed! ✓")
}
