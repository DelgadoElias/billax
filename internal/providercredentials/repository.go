package providercredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
)

// CredentialsRepo defines operations on provider credentials
type CredentialsRepo interface {
	// Set creates or updates credentials for a provider
	Set(ctx context.Context, tenantID uuid.UUID, providerName string, config map[string]string) error

	// Get retrieves credentials for a specific provider
	Get(ctx context.Context, tenantID uuid.UUID, providerName string) (map[string]string, error)

	// List returns all configured providers for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]ProviderConfig, error)

	// Delete removes credentials for a provider
	Delete(ctx context.Context, tenantID uuid.UUID, providerName string) error
}

// ProviderConfig represents stored provider configuration
type ProviderConfig struct {
	ProviderName string
	Config       map[string]string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// postgresRepository implements CredentialsRepo using PostgreSQL
type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new credentials repository
func NewRepository(pool *pgxpool.Pool) CredentialsRepo {
	return &postgresRepository{pool: pool}
}

// Set creates or updates credentials via INSERT ... ON CONFLICT
func (r *postgresRepository) Set(ctx context.Context, tenantID uuid.UUID, providerName string, config map[string]string) error {
	// Validate inputs
	if tenantID == uuid.Nil {
		return fmt.Errorf("set credentials: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return fmt.Errorf("set credentials: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}
	if len(config) == 0 {
		return fmt.Errorf("set credentials: config cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Marshal config to JSONB
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("set credentials: marshaling config: %w", err)
	}

	// Set RLS context
	ctx = middleware.WithTenantID(ctx, tenantID)

	// INSERT ... ON CONFLICT ... DO UPDATE (upsert)
	query := `
		INSERT INTO provider_credentials (tenant_id, provider_name, config, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (tenant_id, provider_name)
		DO UPDATE SET config = EXCLUDED.config, updated_at = NOW()
	`

	_, execErr := r.pool.Exec(ctx, query, tenantID, providerName, configJSON)
	if execErr != nil {
		// Check if constraint violation is due to invalid provider
		if execErr.Error() == "new row for relation \"provider_credentials\" violates check constraint \"valid_provider_name\"" {
			return fmt.Errorf("set credentials: invalid provider %q: %w", providerName, errors.ErrInvalidInput)
		}
		return fmt.Errorf("set credentials: %w", execErr)
	}

	return nil
}

// Get retrieves credentials for a specific provider
func (r *postgresRepository) Get(ctx context.Context, tenantID uuid.UUID, providerName string) (map[string]string, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("get credentials: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return nil, fmt.Errorf("get credentials: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Set RLS context
	ctx = middleware.WithTenantID(ctx, tenantID)

	var configJSON json.RawMessage
	query := `
		SELECT config FROM provider_credentials
		WHERE tenant_id = $1 AND provider_name = $2
	`

	err := r.pool.QueryRow(ctx, query, tenantID, providerName).Scan(&configJSON)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("get credentials: %w", errors.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	var config map[string]string
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("get credentials: unmarsaling config: %w", err)
	}

	return config, nil
}

// List returns all configured providers for a tenant
func (r *postgresRepository) List(ctx context.Context, tenantID uuid.UUID) ([]ProviderConfig, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("list credentials: %w", errors.ErrMissingTenantID)
	}

	// Set RLS context
	ctx = middleware.WithTenantID(ctx, tenantID)

	query := `
		SELECT provider_name, config, created_at, updated_at
		FROM provider_credentials
		WHERE tenant_id = $1
		ORDER BY provider_name
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	var configs []ProviderConfig
	for rows.Next() {
		var pc ProviderConfig
		var configJSON json.RawMessage

		if err := rows.Scan(&pc.ProviderName, &configJSON, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list credentials: scanning row: %w", err)
		}

		if err := json.Unmarshal(configJSON, &pc.Config); err != nil {
			return nil, fmt.Errorf("list credentials: unmarshaling config: %w", err)
		}

		configs = append(configs, pc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}

	return configs, nil
}

// Delete removes credentials for a provider
func (r *postgresRepository) Delete(ctx context.Context, tenantID uuid.UUID, providerName string) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("delete credentials: %w", errors.ErrMissingTenantID)
	}
	if providerName == "" {
		return fmt.Errorf("delete credentials: provider_name cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Set RLS context
	ctx = middleware.WithTenantID(ctx, tenantID)

	query := `
		DELETE FROM provider_credentials
		WHERE tenant_id = $1 AND provider_name = $2
	`

	result, err := r.pool.Exec(ctx, query, tenantID, providerName)
	if err != nil {
		return fmt.Errorf("delete credentials: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete credentials: %w", errors.ErrNotFound)
	}

	return nil
}
