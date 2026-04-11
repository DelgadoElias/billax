package subscription

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/payment"
)

// PaymentServicer defines the interface for creating payments
// Used by lifecycle runner to charge for renewals
type PaymentServicer interface {
	CreatePayment(ctx context.Context, input payment.CreatePaymentInput) (payment.Payment, bool, error)
}

// CredentialsFetcher defines the interface for fetching provider credentials
// Used by lifecycle runner to get access tokens for provider calls
type CredentialsFetcher interface {
	ValidateAndFetch(ctx context.Context, tenantID uuid.UUID, providerName string) (map[string]string, error)
}

// LifecycleRunner manages subscription state transitions and renewals
// It runs three periodic jobs:
// 1. Renew active subscriptions (charge for the next period)
// 2. Transition expired trials to active (charge first period)
// 3. Expire past_due subscriptions (after grace period)
type LifecycleRunner struct {
	subRepo   SubscriptionRepo
	paySvc    PaymentServicer
	credSvc   CredentialsFetcher
	logger    *slog.Logger
	graceDays int
}

// NewLifecycleRunner creates a new subscription lifecycle manager
func NewLifecycleRunner(
	subRepo SubscriptionRepo,
	paySvc PaymentServicer,
	credSvc CredentialsFetcher,
	logger *slog.Logger,
	graceDays int,
) *LifecycleRunner {
	return &LifecycleRunner{
		subRepo:   subRepo,
		paySvc:    paySvc,
		credSvc:   credSvc,
		logger:    logger,
		graceDays: graceDays,
	}
}

// Run starts the lifecycle job loop at the specified interval
// Runs all three jobs sequentially in each iteration
// Context cancellation stops the loop
func (r *LifecycleRunner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.logger.Info("lifecycle runner started", "interval", interval.String())

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("lifecycle runner stopped")
			return
		case <-ticker.C:
			r.runAllJobs(ctx)
		}
	}
}

// runAllJobs executes all three lifecycle jobs in sequence
func (r *LifecycleRunner) runAllJobs(ctx context.Context) {
	now := time.Now().UTC()

	// Job 1: Renew subscriptions due for renewal
	if err := r.RunRenewals(ctx); err != nil {
		r.logger.Error("renewal job failed", "error", err)
	}

	// Job 2: Transition expired trials to active
	if err := r.RunTrialExpiry(ctx); err != nil {
		r.logger.Error("trial expiry job failed", "error", err)
	}

	// Job 3: Expire past_due subscriptions
	gracePeriodEnd := now.AddDate(0, 0, -r.graceDays)
	if err := r.RunPastDueExpiry(ctx, gracePeriodEnd); err != nil {
		r.logger.Error("past due expiry job failed", "error", err)
	}
}

// RunRenewals processes subscriptions due for renewal
// For each subscription:
// 1. Fetch provider credentials
// 2. Create a charge for the next period
// 3. If successful, advance the billing period
// 4. If failed, transition to past_due
func (r *LifecycleRunner) RunRenewals(ctx context.Context) error {
	now := time.Now().UTC()

	subs, err := r.subRepo.ListDueForRenewal(ctx, now)
	if err != nil {
		return fmt.Errorf("running renewals: listing due subscriptions: %w", err)
	}

	if len(subs) == 0 {
		return nil
	}

	r.logger.Info("processing subscription renewals", "count", len(subs))

	for _, sub := range subs {
		if err := r.renewSubscription(ctx, sub); err != nil {
			r.logger.Error("renewal failed", "subscription_id", sub.ID, "subscription_key", sub.SubscriptionKey, "error", err)
		}
	}

	return nil
}

// renewSubscription attempts to renew a single subscription
// Idempotency: uses renewal:{key}:{date} format for idempotency key
// Prevents duplicate charges if job retries
func (r *LifecycleRunner) renewSubscription(ctx context.Context, sub Subscription) error {
	// Fetch provider credentials
	config, err := r.credSvc.ValidateAndFetch(ctx, sub.TenantID, sub.ProviderName)
	if err != nil {
		// Credentials missing → move to past_due
		if err = r.subRepo.UpdateStatus(ctx, sub.ID, StatusPastDue); err != nil {
			return fmt.Errorf("updating status to past_due: %w", err)
		}
		return fmt.Errorf("credentials not configured")
	}

	// Generate idempotency key: renewal:{subscription_key}:{period_end_date}
	// This prevents double-charging if the job retries
	idempotencyKey := fmt.Sprintf("renewal:%s:%s", sub.SubscriptionKey, sub.CurrentPeriodEnd.Format("2006-01-02"))

	// Set tenant context for RLS and audit logging
	jobCtx := middleware.WithTenantID(ctx, sub.TenantID)

	// Attempt to create charge
	_, _, err = r.paySvc.CreatePayment(jobCtx, payment.CreatePaymentInput{
		SubscriptionID:     sub.ID,
		Amount:             sub.Amount,
		Currency:           sub.Currency,
		ProviderName:       sub.ProviderName,
		ProviderConfig:     config,
		IdempotencyKey:     idempotencyKey,
		ExternalCustomerID: sub.ExternalCustomerID,
	})

	if err != nil {
		// Provider error → move to past_due
		if err = r.subRepo.UpdateStatus(ctx, sub.ID, StatusPastDue); err != nil {
			return fmt.Errorf("updating status to past_due after charge failure: %w", err)
		}
		return fmt.Errorf("charge failed: %w", err)
	}

	// Charge succeeded → advance the period
	newStart := sub.CurrentPeriodEnd
	newEnd := nextPeriodEnd(sub.CurrentPeriodEnd, string(sub.Interval), sub.IntervalCount)

	if err := r.subRepo.AdvancePeriod(ctx, sub.ID, newStart, newEnd); err != nil {
		return fmt.Errorf("advancing period: %w", err)
	}

	r.logger.Info("subscription renewed",
		"tenant_id", sub.TenantID.String(),
		"subscription_key", sub.SubscriptionKey,
		"provider", sub.ProviderName,
		"new_period_end", newEnd.Format("2006-01-02"),
	)

	return nil
}

// RunTrialExpiry transitions subscriptions with expired trials from trialing → active
// For each subscription:
// 1. Update status to active
// 2. Set trial_ends_at to now (already expired)
func (r *LifecycleRunner) RunTrialExpiry(ctx context.Context) error {
	now := time.Now().UTC()

	subs, err := r.subRepo.ListExpiredTrials(ctx, now)
	if err != nil {
		return fmt.Errorf("running trial expiry: listing expired trials: %w", err)
	}

	if len(subs) == 0 {
		return nil
	}

	r.logger.Info("transitioning expired trials to active", "count", len(subs))

	for _, sub := range subs {
		if err := r.subRepo.UpdateStatus(ctx, sub.ID, StatusActive); err != nil {
			r.logger.Error("trial transition failed",
				"subscription_id", sub.ID,
				"subscription_key", sub.SubscriptionKey,
				"error", err,
			)
			continue
		}

		r.logger.Info("trial expired → active",
			"tenant_id", sub.TenantID.String(),
			"subscription_key", sub.SubscriptionKey,
		)
	}

	return nil
}

// RunPastDueExpiry expires subscriptions in past_due status
// For each subscription:
// 1. Update status to expired
// 2. Set canceled_at to now
func (r *LifecycleRunner) RunPastDueExpiry(ctx context.Context, gracePeriodEnd time.Time) error {
	subs, err := r.subRepo.ListPastDuePendingExpiry(ctx, gracePeriodEnd)
	if err != nil {
		return fmt.Errorf("running past due expiry: listing pending expiry: %w", err)
	}

	if len(subs) == 0 {
		return nil
	}

	r.logger.Info("expiring past_due subscriptions", "count", len(subs))

	for _, sub := range subs {
		if err := r.subRepo.UpdateStatus(ctx, sub.ID, StatusExpired); err != nil {
			r.logger.Error("expiry failed",
				"subscription_id", sub.ID,
				"subscription_key", sub.SubscriptionKey,
				"error", err,
			)
			continue
		}

		r.logger.Info("subscription expired",
			"tenant_id", sub.TenantID.String(),
			"subscription_key", sub.SubscriptionKey,
		)
	}

	return nil
}

// nextPeriodEnd calculates the end date of the next billing period
// Supports all standard intervals: day, week, month, year
func nextPeriodEnd(from time.Time, interval string, count int) time.Time {
	switch interval {
	case "day":
		return from.AddDate(0, 0, count)
	case "week":
		return from.AddDate(0, 0, count*7)
	case "month":
		return from.AddDate(0, count, 0)
	case "year":
		return from.AddDate(count, 0, 0)
	default:
		// Fallback to month if interval is unknown
		return from.AddDate(0, count, 0)
	}
}
