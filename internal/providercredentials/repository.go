package providercredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DelgadoElias/billax/internal/crypto"
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

	// GetTenantByWebhookSecret returns tenant_id, provider_name, and config given a webhook secret
	// Used by webhook handlers to route incoming webhooks without auth middleware
	// Returns ErrNotFound if secret doesn't exist
	GetTenantByWebhookSecret(ctx context.Context, secret string) (uuid.UUID, string, map[string]string, error)
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
	pool   *pgxpool.Pool
	encKey []byte // 32 bytes for AES-256, or empty to disable encryption
}

// NewRepository creates a new credentials repository with optional encryption
func NewRepository(pool *pgxpool.Pool, encKey []byte) CredentialsRepo {
	return &postgresRepository{pool: pool, encKey: encKey}
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

	// Extract webhook_secret from config for separate column (before encryption)
	// The webhook_secret is part of the URL, not a password, so it's OK to store plaintext
	webhookSecret := config["webhook_secret"]

	// Marshal config to JSON
	plainJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("set credentials: marshaling config: %w", err)
	}

	// Encrypt if key is configured
	var configJSON []byte
	if len(r.encKey) == 32 {
		encrypted, err := crypto.Encrypt(plainJSON, r.encKey)
		if err != nil {
			return fmt.Errorf("set credentials: encrypting config: %w", err)
		}
		// Wrap encrypted data with marker
		wrapper := map[string]string{"_enc": encrypted}
		configJSON, _ = json.Marshal(wrapper)
	} else {
		configJSON = plainJSON
	}

	// Set RLS context
	ctx = middleware.WithTenantID(ctx, tenantID)

	// INSERT ... ON CONFLICT ... DO UPDATE (upsert)
	query := `
		INSERT INTO provider_credentials (tenant_id, provider_name, config, webhook_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (tenant_id, provider_name)
		DO UPDATE SET config = EXCLUDED.config, webhook_secret = EXCLUDED.webhook_secret, updated_at = NOW()
	`

	_, execErr := r.pool.Exec(ctx, query, tenantID, providerName, configJSON, webhookSecret)
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

	var rawJSON json.RawMessage
	query := `
		SELECT config FROM provider_credentials
		WHERE tenant_id = $1 AND provider_name = $2
	`

	err := r.pool.QueryRow(ctx, query, tenantID, providerName).Scan(&rawJSON)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("get credentials: %w", errors.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	// Detect if encrypted and decrypt if needed
	var wrapper map[string]string
	if json.Unmarshal(rawJSON, &wrapper) == nil && wrapper["_enc"] != "" {
		// Encrypted data detected
		if len(r.encKey) == 0 {
			return nil, fmt.Errorf("get credentials: credentials are encrypted but CREDENTIALS_ENCRYPTION_KEY is not configured")
		}
		decrypted, err := crypto.Decrypt(wrapper["_enc"], r.encKey)
		if err != nil {
			return nil, fmt.Errorf("get credentials: decrypting config: %w", err)
		}
		rawJSON = decrypted
	}

	var config map[string]string
	if err := json.Unmarshal(rawJSON, &config); err != nil {
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
		var rawJSON json.RawMessage

		if err := rows.Scan(&pc.ProviderName, &rawJSON, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list credentials: scanning row: %w", err)
		}

		// Detect if encrypted and decrypt if needed
		var wrapper map[string]string
		if json.Unmarshal(rawJSON, &wrapper) == nil && wrapper["_enc"] != "" {
			// Encrypted data detected
			if len(r.encKey) == 0 {
				return nil, fmt.Errorf("list credentials: credentials are encrypted but CREDENTIALS_ENCRYPTION_KEY is not configured")
			}
			decrypted, err := crypto.Decrypt(wrapper["_enc"], r.encKey)
			if err != nil {
				return nil, fmt.Errorf("list credentials: decrypting config: %w", err)
			}
			rawJSON = decrypted
		}

		if err := json.Unmarshal(rawJSON, &pc.Config); err != nil {
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

// GetTenantByWebhookSecret looks up credentials by webhook secret (reverse lookup)
// Uses SECURITY DEFINER function to bypass RLS — webhook handler doesn't have auth context
// Returns (tenantID, providerName, decrypted config, error)
func (r *postgresRepository) GetTenantByWebhookSecret(ctx context.Context, secret string) (uuid.UUID, string, map[string]string, error) {
	if secret == "" {
		return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: secret cannot be empty: %w", errors.ErrInvalidInput)
	}

	var tenantID uuid.UUID
	var providerName string
	var rawJSON json.RawMessage

	// Call SECURITY DEFINER function — bypasses RLS intentionally
	query := `SELECT tenant_id, provider_name, config FROM lookup_tenant_by_webhook_secret($1)`

	err := r.pool.QueryRow(ctx, query, secret).Scan(&tenantID, &providerName, &rawJSON)
	if err == pgx.ErrNoRows {
		return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: %w", errors.ErrNotFound)
	}
	if err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: %w", err)
	}

	// Detect if encrypted and decrypt if needed (same logic as Get())
	var wrapper map[string]string
	if json.Unmarshal(rawJSON, &wrapper) == nil && wrapper["_enc"] != "" {
		// Encrypted data detected
		if len(r.encKey) == 0 {
			return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: credentials are encrypted but CREDENTIALS_ENCRYPTION_KEY is not configured")
		}
		decrypted, err := crypto.Decrypt(wrapper["_enc"], r.encKey)
		if err != nil {
			return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: decrypting config: %w", err)
		}
		rawJSON = decrypted
	}

	var config map[string]string
	if err := json.Unmarshal(rawJSON, &config); err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("get tenant by webhook secret: unmarshaling config: %w", err)
	}

	return tenantID, providerName, config, nil
}
