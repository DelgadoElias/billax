package provider

import (
	"context"
)

// PaymentProvider is the interface every payment connector must implement
type PaymentProvider interface {
	GetProviderName() string
	CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
	RefundCharge(ctx context.Context, chargeID string, amount int64) (*RefundResult, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
	ValidateConfig(config map[string]string) error
}

// ChargeRequest is the generic, provider-agnostic input for creating a charge
type ChargeRequest struct {
	Amount             int64             // centavos, never float
	Currency           string            // ISO 4217
	Description        string
	ExternalCustomerID string
	IdempotencyKey     string
	Config             map[string]string // tenant-stored provider credentials
	Metadata           map[string]string
}

// ChargeStatus normalized across all providers
type ChargeStatus string

const (
	ChargeStatusPending   ChargeStatus = "pending"
	ChargeStatusSucceeded ChargeStatus = "succeeded"
	ChargeStatusFailed    ChargeStatus = "failed"
)

// PaymentMethodInfo is what gets stored in payments.payment_method JSONB
// Connectors extract this from their raw response — never store raw card data
type PaymentMethodInfo struct {
	Type     string `json:"type"`               // credit_card | debit_card | wallet | bank_transfer
	Brand    string `json:"brand,omitempty"`    // visa | mastercard | mercadopago | etc.
	LastFour string `json:"last_four,omitempty"` // last 4 digits only, if applicable
	Bank     string `json:"bank,omitempty"`
}

// ChargeResult is the normalized output after a charge — no provider-specific fields
type ChargeResult struct {
	ProviderChargeID string
	Status           ChargeStatus
	FailureReason    string
	PaymentMethod    *PaymentMethodInfo // non-sensitive, extracted by the connector
	RawResponse      []byte             // stored in provider_response JSONB, never exposed to clients
}

// RefundResult normalized across all providers
type RefundResult struct {
	ProviderRefundID string
	Status           ChargeStatus
	RawResponse      []byte
}

// WebhookEvent is the normalized inbound event from any provider
type WebhookEvent struct {
	ProviderName     string
	EventType        string // subscription.renewed | payment.succeeded | payment.failed | etc.
	ProviderChargeID string
	Status           ChargeStatus
	RawPayload       []byte
}
