package subscription

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/plan"
	"github.com/DelgadoElias/billax/internal/provider"
	"github.com/DelgadoElias/billax/internal/validation"
)

const (
	MaxTagCount  = 20
	MaxTagLength = 50
)

type SubscriptionService struct {
	repo        SubscriptionRepo
	planRepo    plan.PlanRepo
	paymentRepo payment.PaymentRepo
	adapter     *provider.ProviderAdapter
}

func NewService(repo SubscriptionRepo, planRepo plan.PlanRepo, paymentRepo payment.PaymentRepo, adapter *provider.ProviderAdapter) *SubscriptionService {
	return &SubscriptionService{
		repo:        repo,
		planRepo:    planRepo,
		paymentRepo: paymentRepo,
		adapter:     adapter,
	}
}

// Create creates a new subscription (plan-based or planless)
func (s *SubscriptionService) Create(ctx context.Context, input CreateSubscriptionInput) (Subscription, error) {
	// Extract tenant ID from context
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return Subscription{}, errors.ErrMissingTenantID
	}

	periodStart := time.Now()
	subscriptionKey := uuid.New()
	tags := s.normalizeTags(input.Tags)
	status := StatusActive
	var trialEndsAt *time.Time
	var amount int64
	var currency string
	var interval plan.Interval
	var intervalCount int
	var planID *uuid.UUID

	// Plan-based path
	if input.PlanSlug != "" {
		planObj, err := s.planRepo.GetBySlug(ctx, tenantID, input.PlanSlug)
		if err != nil {
			return Subscription{}, fmt.Errorf("resolving plan: %w", err)
		}

		if !planObj.IsActive {
			return Subscription{}, errors.ErrPlanNotActive
		}

		// Check provider capability for plan-based billing if provider is specified
		if input.ProviderName != "" {
			caps := s.adapter.GetCapabilities(input.ProviderName)
			if !caps.Plans {
				return Subscription{}, errors.ErrPlansNotSupported
			}
		}

		// Copy from plan
		planID = &planObj.ID
		amount = planObj.Amount
		currency = planObj.Currency
		interval = planObj.Interval
		intervalCount = planObj.IntervalCount

		// Determine trial
		if planObj.TrialDays > 0 {
			status = StatusTrialing
			trialEnd := periodStart.AddDate(0, 0, planObj.TrialDays)
			trialEndsAt = &trialEnd
		}
	} else {
		// Planless path — validate inputs with strict rules
		v := &validation.ValidationError{}
		v.Add(validation.PositiveInt("amount", input.Amount))
		v.Add(validation.NonEmpty("currency", input.Currency))
		if input.Currency != "" {
			v.Add(validation.ISOCurrency("currency", input.Currency))
		}
		// interval validation: "day", "week", "month", "year"
		allowedIntervals := []string{string(plan.IntervalDay), string(plan.IntervalWeek), string(plan.IntervalMonth), string(plan.IntervalYear)}
		v.Add(validation.ValidEnum("interval", string(input.Interval), allowedIntervals))
		v.Add(validation.MinInt("interval_count", int64(input.IntervalCount), 1))

		if err := v.Err(); err != nil {
			return Subscription{}, err
		}

		amount = input.Amount
		currency = input.Currency
		interval = input.Interval
		intervalCount = input.IntervalCount
		// planless subs have no trial
	}

	// Validate external_customer_id length
	if input.ExternalCustomerID != "" {
		if fe := validation.MaxLength("external_customer_id", input.ExternalCustomerID, 255); fe != nil {
			v := &validation.ValidationError{}
			v.Add(fe)
			return Subscription{}, v.Err()
		}
	}

	// Validate tags before creating
	if err := s.validateTags(input.Tags); err != nil {
		return Subscription{}, err
	}

	// Calculate period end
	periodEnd := s.calculatePeriodEnd(periodStart, interval, intervalCount)

	// Create subscription
	sub := Subscription{
		TenantID:               tenantID,
		PlanID:                 planID,
		Amount:                 amount,
		Currency:               currency,
		Interval:               interval,
		IntervalCount:          intervalCount,
		SubscriptionKey:        subscriptionKey,
		ExternalCustomerID:     input.ExternalCustomerID,
		Status:                 status,
		ProviderName:           input.ProviderName,
		ProviderSubscriptionID: input.ProviderSubscriptionID,
		CurrentPeriodStart:     periodStart,
		CurrentPeriodEnd:       periodEnd,
		TrialEndsAt:            trialEndsAt,
		CancelAtPeriodEnd:      false,
		Tags:                   tags,
		Metadata:               input.Metadata,
	}

	createdSub, err := s.repo.Create(ctx, sub)
	if err != nil {
		return Subscription{}, fmt.Errorf("creating subscription: %w", err)
	}

	return createdSub, nil
}

// GetByKey retrieves a subscription by its external key with enriched payment history
func (s *SubscriptionService) GetByKey(ctx context.Context, key uuid.UUID) (Subscription, error) {
	sub, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return Subscription{}, err
	}

	// Enrich with payment history (last 10)
	payments, err := s.paymentRepo.ListBySubscription(ctx, sub.ID, 10)
	if err != nil {
		// Log but don't fail the request if payments can't be fetched
		payments = []payment.Payment{}
	}

	sub.Payments = payments
	return sub, nil
}

// GetByID retrieves a subscription by its UUID
func (s *SubscriptionService) GetByID(ctx context.Context, id uuid.UUID) (Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

// Update applies partial updates to a subscription
func (s *SubscriptionService) Update(ctx context.Context, id uuid.UUID, input UpdateSubscriptionInput) (Subscription, error) {
	// Fetch current subscription to check capabilities and transitions
	currentSub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Subscription{}, err
	}

	// Pay-per-use gate: check if amount is being updated
	if input.Amount != nil {
		if currentSub.ProviderName == "" {
			return Subscription{}, errors.ErrProviderRequired
		}
		caps := s.adapter.GetCapabilities(currentSub.ProviderName)
		if !caps.PayPerUse {
			return Subscription{}, errors.ErrPayPerUseNotSupported
		}
	}

	// Validate status transition if provided
	if input.Status != nil {
		if err := s.validateStatusTransition(currentSub.Status, *input.Status); err != nil {
			return Subscription{}, err
		}
	}

	// Validate and normalize tags if provided
	if input.TagsProvided {
		if err := s.validateTags(input.Tags); err != nil {
			return Subscription{}, err
		}
		input.Tags = s.normalizeTags(input.Tags)
	}

	sub, err := s.repo.Update(ctx, id, input)
	if err != nil {
		return Subscription{}, err
	}

	return sub, nil
}

// List returns a paginated list of subscriptions for the authenticated tenant
func (s *SubscriptionService) List(ctx context.Context, input ListSubscriptionsInput) (ListSubscriptionsResult, error) {
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return ListSubscriptionsResult{}, errors.ErrMissingTenantID
	}

	// Validate status filter values
	allowedStatuses := []string{string(StatusTrialing), string(StatusActive), string(StatusPastDue), string(StatusCanceled), string(StatusExpired)}
	v := &validation.ValidationError{}
	for i, status := range input.Status {
		if status != "" {
			v.Add(validation.ValidEnum(fmt.Sprintf("status[%d]", i), string(status), allowedStatuses))
		}
	}
	if err := v.Err(); err != nil {
		return ListSubscriptionsResult{}, err
	}

	// Cap limit
	const MaxLimitParam = 100
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > MaxLimitParam {
		input.Limit = MaxLimitParam
	}

	result, err := s.repo.List(ctx, tenantID, input)
	if err != nil {
		return ListSubscriptionsResult{}, err
	}

	return result, nil
}

// Cancel cancels a subscription
func (s *SubscriptionService) Cancel(ctx context.Context, key uuid.UUID, atPeriodEnd bool) (Subscription, error) {
	// Get the subscription first to find its ID
	sub, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return Subscription{}, err
	}

	// Cancel via repository
	canceledSub, err := s.repo.Cancel(ctx, sub.ID, atPeriodEnd)
	if err != nil {
		return Subscription{}, fmt.Errorf("canceling subscription: %w", err)
	}

	return canceledSub, nil
}

// calculatePeriodEnd calculates the end of a billing period
func (s *SubscriptionService) calculatePeriodEnd(start time.Time, interval plan.Interval, count int) time.Time {
	switch interval {
	case plan.IntervalDay:
		return start.AddDate(0, 0, count)
	case plan.IntervalWeek:
		return start.AddDate(0, 0, count*7)
	case plan.IntervalMonth:
		return start.AddDate(0, count, 0)
	case plan.IntervalYear:
		return start.AddDate(count, 0, 0)
	default:
		return start.AddDate(0, 1, 0) // default to 1 month
	}
}

// validateTags validates tag count and individual tag lengths
func (s *SubscriptionService) validateTags(tags []string) error {
	v := &validation.ValidationError{}

	if len(tags) > MaxTagCount {
		v.Add(&validation.FieldError{
			Field:   "tags",
			Message: fmt.Sprintf("must not exceed %d tags", MaxTagCount),
		})
	}

	for i, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if len(trimmed) > MaxTagLength {
			v.Add(&validation.FieldError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: fmt.Sprintf("must be at most %d characters", MaxTagLength),
			})
		}
	}

	return v.Err()
}

// validateStatusTransition checks if a status transition is allowed
func (s *SubscriptionService) validateStatusTransition(current, next Status) error {
	// Define forbidden transitions
	forbiddenTransitions := map[Status]map[Status]bool{
		StatusCanceled: {
			StatusActive:   true,
			StatusTrialing: true,
			StatusPastDue:  true,
		},
		StatusExpired: {
			StatusActive:   true,
			StatusTrialing: true,
		},
	}

	if forbidden, ok := forbiddenTransitions[current]; ok {
		if forbidden[next] {
			return &errors.DomainError{
				Code:       "invalid_status_transition",
				Message:    fmt.Sprintf("cannot transition from %s to %s", current, next),
				HTTPStatus: 422,
				Cause:      errors.ErrInvalidInput,
			}
		}
	}

	return nil
}

// normalizeTags lowercases and deduplicates tags
func (s *SubscriptionService) normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	var normalized []string

	for _, tag := range tags {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if lower != "" && !seen[lower] {
			seen[lower] = true
			normalized = append(normalized, lower)
		}
	}

	return normalized
}
