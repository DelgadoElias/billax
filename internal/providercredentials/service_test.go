package providercredentials

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/provider"
	"github.com/DelgadoElias/billax/internal/provider/mercadopago"
)

// mockRepository implements CredentialsRepo for testing
type mockRepository struct {
	data map[string]map[string]map[string]string // tenant -> provider -> config
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		data: make(map[string]map[string]map[string]string),
	}
}

func (m *mockRepository) Set(ctx context.Context, tenantID uuid.UUID, providerName string, config map[string]string) error {
	if len(config) == 0 {
		return errors.ErrInvalidInput
	}
	tenantKey := tenantID.String()
	if _, ok := m.data[tenantKey]; !ok {
		m.data[tenantKey] = make(map[string]map[string]string)
	}
	m.data[tenantKey][providerName] = config
	return nil
}

func (m *mockRepository) Get(ctx context.Context, tenantID uuid.UUID, providerName string) (map[string]string, error) {
	tenantKey := tenantID.String()
	if configs, ok := m.data[tenantKey]; ok {
		if config, ok := configs[providerName]; ok {
			// Return a copy
			result := make(map[string]string)
			for k, v := range config {
				result[k] = v
			}
			return result, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (m *mockRepository) List(ctx context.Context, tenantID uuid.UUID) ([]ProviderConfig, error) {
	tenantKey := tenantID.String()
	configs, ok := m.data[tenantKey]
	if !ok {
		return []ProviderConfig{}, nil
	}

	var result []ProviderConfig
	for providerName, config := range configs {
		result = append(result, ProviderConfig{
			ProviderName: providerName,
			Config:       config,
		})
	}
	return result, nil
}

func (m *mockRepository) Delete(ctx context.Context, tenantID uuid.UUID, providerName string) error {
	tenantKey := tenantID.String()
	if configs, ok := m.data[tenantKey]; ok {
		if _, ok := configs[providerName]; ok {
			delete(configs, providerName)
			return nil
		}
	}
	return errors.ErrNotFound
}

func (m *mockRepository) GetTenantByWebhookSecret(ctx context.Context, secret string) (uuid.UUID, string, map[string]string, error) {
	// Mock implementation: search for the secret across all tenants
	for tenantKey, providers := range m.data {
		for providerName, config := range providers {
			if config["webhook_secret"] == secret {
				tenantID, _ := uuid.Parse(tenantKey)
				// Return a copy of config
				result := make(map[string]string)
				for k, v := range config {
					result[k] = v
				}
				return tenantID, providerName, result, nil
			}
		}
	}
	return uuid.Nil, "", nil, errors.ErrNotFound
}

// Setup test service
func setupTestService(t *testing.T) (*CredentialsService, uuid.UUID) {
	t.Helper()

	// Create registry with mercadopago
	registry := provider.NewRegistry()
	registry.Register(mercadopago.New())

	// Create adapter
	adapter := provider.NewAdapter(registry, provider.CapabilitiesConfig{})

	// Create service with mock repo
	mockRepo := newMockRepository()
	svc := NewService(mockRepo, adapter)

	tenantID := uuid.New()
	return svc, tenantID
}

// Tests

func TestSetProviderConfigValid(t *testing.T) {
	svc, tenantID := setupTestService(t)

	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}

	err := svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)
	if err != nil {
		t.Fatalf("SetProviderConfig() error: %v", err)
	}
}

func TestSetProviderConfigMissingToken(t *testing.T) {
	svc, tenantID := setupTestService(t)

	config := map[string]string{
		"webhook_secret": "test_secret",
	}

	err := svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)
	if err == nil {
		t.Error("SetProviderConfig() should return error for missing access_token")
	}
}

func TestSetProviderConfigMissingSecret(t *testing.T) {
	svc, tenantID := setupTestService(t)

	config := map[string]string{
		"access_token": "test_token",
	}

	err := svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)
	if err == nil {
		t.Error("SetProviderConfig() should return error for missing webhook_secret")
	}
}

func TestSetProviderConfigUnknownProvider(t *testing.T) {
	svc, tenantID := setupTestService(t)

	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}

	err := svc.SetProviderConfig(context.Background(), tenantID, "unknown_provider", config)
	if err == nil {
		t.Error("SetProviderConfig() should return error for unknown provider")
	}
}

func TestGetProviderConfigExists(t *testing.T) {
	svc, tenantID := setupTestService(t)

	// Set first
	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}
	svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)

	// Get
	retrieved, err := svc.GetProviderConfig(context.Background(), tenantID, "mercadopago")
	if err != nil {
		t.Fatalf("GetProviderConfig() error: %v", err)
	}

	if retrieved["access_token"] != "test_token" {
		t.Errorf("access_token = %s, want test_token", retrieved["access_token"])
	}
}

func TestGetProviderConfigNotExists(t *testing.T) {
	svc, tenantID := setupTestService(t)

	_, err := svc.GetProviderConfig(context.Background(), tenantID, "mercadopago")
	if err == nil {
		t.Error("GetProviderConfig() should return error when config not found")
	}
}

func TestValidateAndFetch(t *testing.T) {
	svc, tenantID := setupTestService(t)

	// Set first
	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}
	svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)

	// Validate and fetch
	retrieved, err := svc.ValidateAndFetch(context.Background(), tenantID, "mercadopago")
	if err != nil {
		t.Fatalf("ValidateAndFetch() error: %v", err)
	}

	if retrieved["access_token"] != "test_token" {
		t.Errorf("access_token = %s, want test_token", retrieved["access_token"])
	}
}

func TestValidateAndFetchUnregisteredProvider(t *testing.T) {
	svc, tenantID := setupTestService(t)

	_, err := svc.ValidateAndFetch(context.Background(), tenantID, "unknown_provider")
	if err == nil {
		t.Error("ValidateAndFetch() should return error for unregistered provider")
	}
}

func TestListProviders(t *testing.T) {
	svc, tenantID := setupTestService(t)

	// Set multiple configs
	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}
	svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)

	// List
	providers, err := svc.ListProviders(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListProviders() error: %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("ListProviders() returned %d providers, want 1", len(providers))
	}
	if providers[0] != "mercadopago" {
		t.Errorf("First provider = %s, want mercadopago", providers[0])
	}
}

func TestDeleteProviderConfig(t *testing.T) {
	svc, tenantID := setupTestService(t)

	// Set first
	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}
	svc.SetProviderConfig(context.Background(), tenantID, "mercadopago", config)

	// Delete
	err := svc.DeleteProviderConfig(context.Background(), tenantID, "mercadopago")
	if err != nil {
		t.Fatalf("DeleteProviderConfig() error: %v", err)
	}

	// Verify deleted
	_, err = svc.GetProviderConfig(context.Background(), tenantID, "mercadopago")
	if err == nil {
		t.Error("Config should be deleted")
	}
}

func TestMissingTenantID(t *testing.T) {
	svc, _ := setupTestService(t)
	nilID := uuid.UUID{}

	config := map[string]string{
		"access_token":   "test_token",
		"webhook_secret": "test_secret",
	}

	err := svc.SetProviderConfig(context.Background(), nilID, "mercadopago", config)
	if err == nil {
		t.Error("SetProviderConfig() should return error for nil tenant_id")
	}
}
