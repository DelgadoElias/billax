package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domerrors "github.com/DelgadoElias/billax/internal/errors"
)

// TenantRepo defines database operations for tenants and API keys
type TenantRepo interface {
	Create(ctx context.Context, t Tenant) (Tenant, error)
	GetBySlug(ctx context.Context, slug string) (Tenant, error)
	GetByID(ctx context.Context, id uuid.UUID) (Tenant, error)
	UpdateDefaultProvider(ctx context.Context, tenantID uuid.UUID, providerName string) (Tenant, error)
	CreateAPIKey(ctx context.Context, tenantID uuid.UUID, keyPrefix, keyHash string, input CreateKeyInput) (APIKey, error)
	ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]APIKey, error)
	GetAPIKeyByID(ctx context.Context, tenantID uuid.UUID, keyID uuid.UUID) (APIKey, error)
	RevokeAPIKey(ctx context.Context, tenantID uuid.UUID, keyID uuid.UUID) error
}

// Repository implements TenantRepo
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new tenant repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new tenant into the database
func (r *Repository) Create(ctx context.Context, t Tenant) (Tenant, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var created Tenant
	err = conn.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, email, is_active)
		 VALUES ($1, $2, $3, true)
		 RETURNING id, name, slug, email, is_active, default_provider_name, created_at, updated_at`,
		t.Name, t.Slug, t.Email,
	).Scan(&created.ID, &created.Name, &created.Slug, &created.Email, &created.IsActive, &created.DefaultProviderName, &created.CreatedAt, &created.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tenant{}, domerrors.ErrNotFound
		}
		// Check for duplicate slug or email
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Tenant{}, err
		}
		errMsg := err.Error()
		if containsStr(errMsg, "unique constraint", "slug") || containsStr(errMsg, "duplicate key", "slug") {
			return Tenant{}, &domerrors.DomainError{
				Code:       "duplicate_slug",
				Message:    "a tenant with this slug already exists",
				HTTPStatus: 409,
				Cause:      domerrors.ErrConflict,
			}
		}
		if containsStr(errMsg, "unique constraint", "email") || containsStr(errMsg, "duplicate key", "email") {
			return Tenant{}, &domerrors.DomainError{
				Code:       "duplicate_email",
				Message:    "this email is already registered",
				HTTPStatus: 409,
				Cause:      domerrors.ErrConflict,
			}
		}
		return Tenant{}, fmt.Errorf("creating tenant: %w", err)
	}

	return created, nil
}

// GetBySlug retrieves a tenant by slug
func (r *Repository) GetBySlug(ctx context.Context, slug string) (Tenant, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var t Tenant
	err = conn.QueryRow(ctx,
		`SELECT id, name, slug, email, is_active, default_provider_name, created_at, updated_at
		 FROM tenants WHERE slug = $1`,
		slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Email, &t.IsActive, &t.DefaultProviderName, &t.CreatedAt, &t.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, domerrors.ErrNotFound
	}
	if err != nil {
		return Tenant{}, fmt.Errorf("getting tenant by slug: %w", err)
	}

	return t, nil
}

// GetByID retrieves a tenant by UUID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Tenant, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var t Tenant
	err = conn.QueryRow(ctx,
		`SELECT id, name, slug, email, is_active, default_provider_name, created_at, updated_at
		 FROM tenants WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Email, &t.IsActive, &t.DefaultProviderName, &t.CreatedAt, &t.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, domerrors.ErrNotFound
	}
	if err != nil {
		return Tenant{}, fmt.Errorf("getting tenant by id: %w", err)
	}

	return t, nil
}

// UpdateDefaultProvider updates the default provider for a tenant and returns the updated tenant
func (r *Repository) UpdateDefaultProvider(ctx context.Context, tenantID uuid.UUID, providerName string) (Tenant, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var t Tenant
	err = conn.QueryRow(ctx,
		`UPDATE tenants
		 SET default_provider_name = $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING id, name, slug, email, is_active, default_provider_name, created_at, updated_at`,
		providerName, tenantID,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Email, &t.IsActive, &t.DefaultProviderName, &t.CreatedAt, &t.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, domerrors.ErrNotFound
	}
	if err != nil {
		return Tenant{}, fmt.Errorf("updating default provider: %w", err)
	}

	return t, nil
}

// CreateAPIKey inserts a new API key for a tenant
func (r *Repository) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, keyPrefix, keyHash string, input CreateKeyInput) (APIKey, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return APIKey{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var created APIKey
	err = conn.QueryRow(ctx,
		`INSERT INTO tenant_api_keys (tenant_id, key_prefix, key_hash, scopes, description, expires_at, is_active)
		 VALUES ($1, $2, $3, '{read,write}', $4, $5, true)
		 RETURNING id, tenant_id, key_prefix, scopes, description, last_used_at, expires_at, is_active, created_at`,
		tenantID, keyPrefix, keyHash, input.Description, input.ExpiresAt,
	).Scan(&created.ID, &created.TenantID, &created.KeyPrefix, &created.Scopes, &created.Description, &created.LastUsedAt, &created.ExpiresAt, &created.IsActive, &created.CreatedAt)

	if err != nil {
		return APIKey{}, fmt.Errorf("creating api key: %w", err)
	}

	return created, nil
}

// ListAPIKeys returns all active API keys for a tenant
func (r *Repository) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]APIKey, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	rows, err := conn.Query(ctx,
		`SELECT id, tenant_id, key_prefix, scopes, description, last_used_at, expires_at, is_active, created_at
		 FROM tenant_api_keys WHERE tenant_id = $1 AND is_active = true
		 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		err := rows.Scan(&k.ID, &k.TenantID, &k.KeyPrefix, &k.Scopes, &k.Description, &k.LastUsedAt, &k.ExpiresAt, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning api key: %w", err)
		}
		keys = append(keys, k)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return keys, nil
}

// GetAPIKeyByID retrieves a specific API key
func (r *Repository) GetAPIKeyByID(ctx context.Context, tenantID uuid.UUID, keyID uuid.UUID) (APIKey, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return APIKey{}, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	var k APIKey
	err = conn.QueryRow(ctx,
		`SELECT id, tenant_id, key_prefix, scopes, description, last_used_at, expires_at, is_active, created_at
		 FROM tenant_api_keys WHERE id = $1 AND tenant_id = $2`,
		keyID, tenantID,
	).Scan(&k.ID, &k.TenantID, &k.KeyPrefix, &k.Scopes, &k.Description, &k.LastUsedAt, &k.ExpiresAt, &k.IsActive, &k.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, &domerrors.DomainError{
			Code:       "key_not_found",
			Message:    "API key not found",
			HTTPStatus: 404,
			Cause:      domerrors.ErrNotFound,
		}
	}
	if err != nil {
		return APIKey{}, fmt.Errorf("getting api key: %w", err)
	}

	return k, nil
}

// RevokeAPIKey deactivates an API key
func (r *Repository) RevokeAPIKey(ctx context.Context, tenantID uuid.UUID, keyID uuid.UUID) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	result, err := conn.Exec(ctx,
		`UPDATE tenant_api_keys SET is_active = false
		 WHERE id = $1 AND tenant_id = $2`,
		keyID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("revoking api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return &domerrors.DomainError{
			Code:       "key_not_found",
			Message:    "API key not found",
			HTTPStatus: 404,
			Cause:      domerrors.ErrNotFound,
		}
	}

	return nil
}

// Helper function to check if all substrings are in a string
func containsStr(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !contains(s, substr) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
