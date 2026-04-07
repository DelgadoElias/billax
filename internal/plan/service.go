package plan

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/validation"
)

type PlanService struct {
	repo PlanRepo
}

func NewService(repo PlanRepo) *PlanService {
	return &PlanService{repo: repo}
}

// CreateOrGet implements idempotent plan creation by slug
// Returns (plan, created, nil) if success
func (s *PlanService) CreateOrGet(ctx context.Context, input CreatePlanInput) (Plan, bool, error) {
	// Extract tenant ID from context
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return Plan{}, false, errors.ErrMissingTenantID
	}

	// Validate input
	if err := s.validateCreateInput(input); err != nil {
		return Plan{}, false, err
	}

	// Default currency
	if input.Currency == "" {
		input.Currency = "ARS"
	}

	// Default interval_count
	if input.IntervalCount == 0 {
		input.IntervalCount = 1
	}

	// Normalize slug
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))

	// Upsert (idempotent by slug)
	result, err := s.repo.Upsert(ctx, tenantID, input)
	if err != nil {
		return Plan{}, false, fmt.Errorf("creating plan: %w", err)
	}

	return result.Plan, result.Created, nil
}

// GetByID retrieves a plan by its UUID
func (s *PlanService) GetByID(ctx context.Context, id uuid.UUID) (Plan, error) {
	plan, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// GetBySlug retrieves a plan by its slug, scoped to the authenticated tenant
func (s *PlanService) GetBySlug(ctx context.Context, slug string) (Plan, error) {
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return Plan{}, errors.ErrMissingTenantID
	}

	slug = strings.ToLower(strings.TrimSpace(slug))

	plan, err := s.repo.GetBySlug(ctx, tenantID, slug)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// Update applies partial updates to a plan
func (s *PlanService) Update(ctx context.Context, id uuid.UUID, input UpdatePlanInput) (Plan, error) {
	// Validate input before update
	if err := s.validateUpdateInput(input); err != nil {
		return Plan{}, err
	}

	plan, err := s.repo.Update(ctx, id, input)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// List returns a cursor-paginated list of plans for the authenticated tenant
func (s *PlanService) List(ctx context.Context, input ListPlansInput) (ListPlansResult, error) {
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return ListPlansResult{}, errors.ErrMissingTenantID
	}

	// Cap limit to prevent excessive results
	const MaxLimitParam = 100
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > MaxLimitParam {
		input.Limit = MaxLimitParam
	}

	result, err := s.repo.List(ctx, tenantID, input)
	if err != nil {
		return ListPlansResult{}, err
	}

	return result, nil
}

// Delete soft-deletes a plan by setting is_active = false
func (s *PlanService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting plan: %w", err)
	}
	return nil
}

// validateCreateInput validates the plan creation input
func (s *PlanService) validateCreateInput(input CreatePlanInput) error {
	v := &validation.ValidationError{}

	// Validate name
	v.Add(validation.NonEmpty("name", input.Name))
	v.Add(validation.MaxLength("name", input.Name, 255))

	// Validate slug format
	v.Add(validation.NonEmpty("slug", input.Slug))
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug != "" && !isValidSlugFormat(slug) {
		v.Add(&validation.FieldError{
			Field:   "slug",
			Message: "must be lowercase alphanumeric with hyphens, 1-64 chars",
		})
	}

	// Validate amount
	v.Add(validation.PositiveInt("amount", input.Amount))

	// Validate currency
	if input.Currency != "" {
		v.Add(validation.ISOCurrency("currency", input.Currency))
	}

	// Validate interval
	if input.Interval != "" {
		allowedIntervals := []string{string(IntervalDay), string(IntervalWeek), string(IntervalMonth), string(IntervalYear)}
		v.Add(validation.ValidEnum("interval", string(input.Interval), allowedIntervals))
	}

	// Validate trial_days is not negative
	if input.TrialDays < 0 {
		v.Add(&validation.FieldError{Field: "trial_days", Message: "must not be negative"})
	}

	return v.Err()
}

// validateUpdateInput validates the plan update input
func (s *PlanService) validateUpdateInput(input UpdatePlanInput) error {
	v := &validation.ValidationError{}

	if input.Name != nil {
		v.Add(validation.NonEmpty("name", *input.Name))
		v.Add(validation.MaxLength("name", *input.Name, 255))
	}

	return v.Err()
}

// isValidSlugFormat checks if a slug matches the allowed pattern
func isValidSlugFormat(slug string) bool {
	// Pattern: lowercase alphanumeric + hyphens, 1-64 chars
	// Must start and end with alphanumeric if more than 1 char
	if len(slug) == 0 || len(slug) > 64 {
		return false
	}
	if len(slug) == 1 {
		return isAlphanumericLower(rune(slug[0]))
	}
	// Multi-char: must start and end with alphanumeric
	if !isAlphanumericLower(rune(slug[0])) || !isAlphanumericLower(rune(slug[len(slug)-1])) {
		return false
	}
	// All chars must be alphanumeric or hyphen
	for _, ch := range slug {
		if !isAlphanumericLower(ch) && ch != '-' {
			return false
		}
	}
	return true
}

func isAlphanumericLower(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
