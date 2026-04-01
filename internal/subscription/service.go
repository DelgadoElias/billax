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
		// Planless path
		if input.Amount <= 0 {
			return Subscription{}, errors.ErrInvalidInput
		}
		if input.Currency == "" {
			return Subscription{}, errors.ErrInvalidInput
		}
		// interval validation: "day", "week", "month", "year"
		if input.Interval != plan.IntervalDay && input.Interval != plan.IntervalWeek &&
			input.Interval != plan.IntervalMonth && input.Interval != plan.IntervalYear {
			return Subscription{}, errors.ErrInvalidInput
		}
		if input.IntervalCount <= 0 {
			return Subscription{}, errors.ErrInvalidInput
		}

		amount = input.Amount
		currency = input.Currency
		interval = input.Interval
		intervalCount = input.IntervalCount
		// planless subs have no trial
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
	// Fetch current subscription to check capabilities for amount update
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

	// Normalize tags if provided
	if input.TagsProvided && input.Tags != nil {
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
