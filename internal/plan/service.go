package plan

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
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
	// Validate slug: non-empty, lowercase, alphanumeric + hyphens
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug == "" {
		return errors.ErrInvalidInput
	}

	slugPattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$|^[a-z0-9]$`)
	if !slugPattern.MatchString(slug) {
		return errors.ErrInvalidInput
	}

	// Validate name
	if strings.TrimSpace(input.Name) == "" {
		return errors.ErrInvalidInput
	}

	// Validate amount > 0
	if input.Amount <= 0 {
		return errors.ErrInvalidInput
	}

	// Validate interval
	switch input.Interval {
	case IntervalDay, IntervalWeek, IntervalMonth, IntervalYear:
	default:
		return errors.ErrInvalidInput
	}

	return nil
}
