package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/provider"
)

type PaymentService struct {
	repo    PaymentRepo
	adapter *provider.ProviderAdapter
}

func NewService(repo PaymentRepo, adapter *provider.ProviderAdapter) *PaymentService {
	return &PaymentService{
		repo:    repo,
		adapter: adapter,
	}
}

// CreatePayment creates a new payment via a payment provider
// Uses idempotency key for duplicate prevention
// Normalizes provider response into domain Payment model via the adapter
func (s *PaymentService) CreatePayment(ctx context.Context, input CreatePaymentInput) (Payment, bool, error) {
	// Extract tenant ID from context
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return Payment{}, false, errors.ErrMissingTenantID
	}

	// Validate input
	if input.IdempotencyKey == "" {
		return Payment{}, false, errors.ErrMissingIdempotencyKey
	}

	if input.Amount <= 0 {
		return Payment{}, false, errors.ErrInvalidInput
	}

	// Call the provider adapter to create the charge
	chargeReq := provider.ChargeRequest{
		Amount:             input.Amount,
		Currency:           input.Currency,
		Description:        input.Description,
		ExternalCustomerID: input.ExternalCustomerID,
		IdempotencyKey:     input.IdempotencyKey,
		Config:             input.ProviderConfig,
		Metadata:           input.Metadata,
	}

	chargeResult, err := s.adapter.CreateCharge(ctx, input.ProviderName, chargeReq)
	if err != nil {
		return Payment{}, false, fmt.Errorf("provider charge: %w", err)
	}

	// Build Payment domain object from provider result
	payment := Payment{
		ID:               uuid.New(),
		TenantID:         tenantID,
		SubscriptionID:   input.SubscriptionID,
		IdempotencyKey:   input.IdempotencyKey,
		ProviderName:     input.ProviderName,
		ProviderChargeID: chargeResult.ProviderChargeID,
		Amount:           input.Amount,
		Currency:         input.Currency,
		Status:           convertChargeStatus(chargeResult.Status),
		FailureReason:    chargeResult.FailureReason,
		PaymentMethod:    chargeResult.PaymentMethod,
		ProviderResponse: chargeResult.RawResponse,
	}

	// Create in repository (idempotent via ON CONFLICT)
	result, err := s.repo.Create(ctx, payment)
	if err != nil {
		return Payment{}, false, fmt.Errorf("creating payment: %w", err)
	}

	return result.Payment, result.Created, nil
}

// GetByID retrieves a payment by ID
func (s *PaymentService) GetByID(ctx context.Context, id uuid.UUID) (Payment, error) {
	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	return payment, nil
}

// List returns a paginated list of payments for the authenticated tenant
func (s *PaymentService) List(ctx context.Context, input ListPaymentsInput) (ListPaymentsResult, error) {
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		return ListPaymentsResult{}, errors.ErrMissingTenantID
	}

	result, err := s.repo.List(ctx, tenantID, input)
	if err != nil {
		return ListPaymentsResult{}, err
	}

	return result, nil
}

// ListBySubscription returns recent payments for a subscription
func (s *PaymentService) ListBySubscription(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]Payment, error) {
	payments, err := s.repo.ListBySubscription(ctx, subscriptionID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing payments by subscription: %w", err)
	}

	return payments, nil
}

// convertChargeStatus converts provider ChargeStatus to domain Status
func convertChargeStatus(chargeStatus provider.ChargeStatus) Status {
	switch chargeStatus {
	case provider.ChargeStatusPending:
		return StatusPending
	case provider.ChargeStatusSucceeded:
		return StatusSucceeded
	case provider.ChargeStatusFailed:
		return StatusFailed
	default:
		return StatusPending
	}
}
