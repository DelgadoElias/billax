package tenant

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a customer account (tenant) in billax
type Tenant struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Slug                  string    `json:"slug"`
	Email                 string    `json:"email"`
	IsActive              bool      `json:"is_active"`
	DefaultProviderName   string    `json:"default_provider_name"` // default payment provider (mercadopago, stripe, helipagos)
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// APIKey represents an API key for a tenant
// The plaintext key is never stored — only the hash is persisted
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	KeyPrefix   string     `json:"key_prefix"`   // First 12 chars of the plaintext key
	Scopes      []string   `json:"scopes"`       // Permissions (read, write, etc.)
	Description string     `json:"description"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PlaintextKey wraps an APIKey with its plaintext representation
// The plaintext key is shown ONCE to the user and never persisted
type PlaintextKey struct {
	Key    string `json:"key"`    // Full plaintext key: "payd_test_xxxxx" or "payd_live_xxxxx"
	APIKey APIKey `json:"api_key"`
}

// SignupInput is the request body for POST /v1/signup
type SignupInput struct {
	Name  string `json:"name"`
	Slug  string `json:"slug,omitempty"` // Auto-generated from name if empty
	Email string `json:"email"`
}

// CreateKeyInput is the request body for POST /v1/keys
type CreateKeyInput struct {
	Description string     `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// SignupResponse is the response for POST /v1/signup
type SignupResponse struct {
	Tenant   Tenant       `json:"tenant"`
	APIKey   PlaintextKey `json:"api_key"`
	Warning  string       `json:"warning"`
}

// ListKeysResponse is the response for GET /v1/keys
type ListKeysResponse struct {
	Keys []APIKey `json:"keys"`
}
