package tenant

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/validation"
)

// TenantService handles business logic for tenants and API keys
type TenantService struct {
	repo TenantRepo
	env  string // "production" or "development"
}

// NewService creates a new tenant service
func NewService(repo TenantRepo, env string) *TenantService {
	return &TenantService{
		repo: repo,
		env:  env,
	}
}

// Signup creates a new tenant and its first API key in a single operation
// Returns the tenant and plaintext key (shown once only)
func (s *TenantService) Signup(ctx context.Context, input SignupInput) (Tenant, PlaintextKey, error) {
	// Validate inputs
	v := &validation.ValidationError{}
	v.Add(validation.NonEmpty("name", input.Name))
	v.Add(validation.MaxLength("name", input.Name, 255))
	v.Add(validation.NonEmpty("email", input.Email))

	if input.Slug != "" {
		if err := validateSlug(input.Slug); err != nil {
			v.Add(err)
		}
	}

	if err := v.Err(); err != nil {
		return Tenant{}, PlaintextKey{}, err
	}

	// Generate slug if not provided
	slug := input.Slug
	if slug == "" {
		slug = generateSlugFromName(input.Name)
	}

	// Create tenant
	tenant := Tenant{
		ID:       uuid.New(),
		Name:     input.Name,
		Slug:     slug,
		Email:    input.Email,
		IsActive: true,
	}

	created, err := s.repo.Create(ctx, tenant)
	if err != nil {
		return Tenant{}, PlaintextKey{}, err
	}

	// Generate first API key
	plaintextKey, err := s.CreateKey(ctx, created.ID, CreateKeyInput{
		Description: "Initial key",
	})
	if err != nil {
		return Tenant{}, PlaintextKey{}, fmt.Errorf("creating initial key: %w", err)
	}

	return created, plaintextKey, nil
}

// CreateKey generates a new API key for an authenticated tenant
func (s *TenantService) CreateKey(ctx context.Context, tenantID uuid.UUID, input CreateKeyInput) (PlaintextKey, error) {
	// Validate expiry if provided
	if input.ExpiresAt != nil {
		if input.ExpiresAt.Before(time.Now()) {
			v := &validation.ValidationError{}
			v.Add(&validation.FieldError{
				Field:   "expires_at",
				Message: "must be in the future",
			})
			return PlaintextKey{}, v.Err()
		}
	}

	// Generate cryptographically secure key
	plaintext, prefix, hash, err := GenerateAPIKey(s.env)
	if err != nil {
		return PlaintextKey{}, fmt.Errorf("generating api key: %w", err)
	}

	// Store in database (hash only, never plaintext)
	apiKey, err := s.repo.CreateAPIKey(ctx, tenantID, prefix, hash, input)
	if err != nil {
		return PlaintextKey{}, err
	}

	return PlaintextKey{
		Key:    plaintext,
		APIKey: apiKey,
	}, nil
}

// ListKeys returns all active API keys for a tenant (metadata only, no hash)
func (s *TenantService) ListKeys(ctx context.Context, tenantID uuid.UUID) ([]APIKey, error) {
	return s.repo.ListAPIKeys(ctx, tenantID)
}

// RevokeKey deactivates an API key
func (s *TenantService) RevokeKey(ctx context.Context, tenantID, keyID uuid.UUID) error {
	return s.repo.RevokeAPIKey(ctx, tenantID, keyID)
}

// SetDefaultProvider updates the tenant's default payment provider
func (s *TenantService) SetDefaultProvider(ctx context.Context, tenantID uuid.UUID, providerName string) (Tenant, error) {
	// Validate provider name
	if err := validateProviderName(providerName); err != nil {
		return Tenant{}, err
	}

	return s.repo.UpdateDefaultProvider(ctx, tenantID, providerName)
}

// --- Helper functions ---

// validateProviderName checks if the provider name is valid
func validateProviderName(providerName string) error {
	validProviders := map[string]bool{
		"mercadopago": true,
		"stripe":      true,
		"helipagos":   true,
	}

	if !validProviders[providerName] {
		v := &validation.ValidationError{}
		v.Add(&validation.FieldError{
			Field:   "provider_name",
			Message: "must be one of: mercadopago, stripe, helipagos",
		})
		return v.Err()
	}

	return nil
}

// generateSlugFromName auto-generates a slug from a tenant name
// Example: "ACME Corp" → "acme-corp"
func generateSlugFromName(name string) string {
	// Lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit to 100 characters
	if len(slug) > 100 {
		slug = slug[:100]
	}

	return slug
}

// validateSlug checks if a slug is valid
// Returns *validation.FieldError which can be nil
func validateSlug(slug string) *validation.FieldError {
	if len(slug) < 2 {
		return &validation.FieldError{
			Field:   "slug",
			Message: "must be at least 2 characters",
		}
	}
	if len(slug) > 100 {
		return &validation.FieldError{
			Field:   "slug",
			Message: "must be at most 100 characters",
		}
	}

	// Allow only lowercase letters, numbers, and hyphens
	reg := regexp.MustCompile(`^[a-z0-9\-]+$`)
	if !reg.MatchString(slug) {
		return &validation.FieldError{
			Field:   "slug",
			Message: "must contain only lowercase letters, numbers, and hyphens",
		}
	}

	return nil
}
