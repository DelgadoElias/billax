package payment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/DelgadoElias/billax/internal/provider"
)

// Status represents payment status
type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
)

// Payment is the domain model
type Payment struct {
	ID               uuid.UUID           `json:"id"`
	TenantID         uuid.UUID           `json:"tenant_id"`
	SubscriptionID   uuid.UUID           `json:"subscription_id"`
	IdempotencyKey   string              `json:"idempotency_key"`
	ProviderName     string              `json:"provider_name"`
	ProviderChargeID string              `json:"provider_charge_id,omitempty"`
	Amount           int64               `json:"amount"` // centavos, never float
	Currency         string              `json:"currency"`
	Status           Status              `json:"status"`
	FailureReason    string              `json:"failure_reason,omitempty"`
	PaymentMethod    *provider.PaymentMethodInfo `json:"payment_method,omitempty"` // non-sensitive metadata
	ProviderResponse json.RawMessage     `json:"-"` // raw payload, never exposed to client
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// CreatePaymentInput is what the service accepts from the handler
type CreatePaymentInput struct {
	SubscriptionID   uuid.UUID           `json:"subscription_id"`
	IdempotencyKey   string              `json:"idempotency_key"`
	ProviderName     string              `json:"provider_name"`
	Amount           int64               `json:"amount"`
	Currency         string              `json:"currency"`
	Description      string              `json:"description"`
	ExternalCustomerID string             `json:"external_customer_id"`
	ProviderConfig   map[string]string   `json:"provider_config,omitempty"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
}

// ListPaymentsInput for cursor-based pagination + filtering
type ListPaymentsInput struct {
	ProviderName *string
	Status       []Status
	Limit        int    // default 20, max 100
	Cursor       string // opaque base64(created_at:id)
}

// ListPaymentsResult wraps paginated results
type ListPaymentsResult struct {
	Payments   []Payment `json:"payments"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

// CreatePaymentResult includes the payment and whether it was newly created
type CreatePaymentResult struct {
	Payment Payment
	Created bool
}
