package mercadopago

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/provider"
)

// Provider implements the PaymentProvider interface for Mercado Pago
type Provider struct {
	client *Client
}

// New creates a new Mercado Pago provider
func New() *Provider {
	return &Provider{
		client: newClient(),
	}
}

// newWithClient creates a provider with a custom client (for testing)
func newWithClient(c *Client) *Provider {
	return &Provider{
		client: c,
	}
}

// GetProviderName returns the provider's unique identifier
func (p *Provider) GetProviderName() string {
	return "mercadopago"
}

// Capabilities returns the provider's self-declared capabilities
// (YAML config can override these at deployment time)
func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Plans:     true,
		PayPerUse: false,
	}
}

// RequiredConfig returns the list of configuration keys required for this provider
// Note: This is a helper method not in the interface; it documents the contract for callers
func (p *Provider) RequiredConfig() []string {
	return []string{"access_token", "webhook_secret"}
}

// ValidateConfig validates that the provider configuration is complete
func (p *Provider) ValidateConfig(config map[string]string) error {
	required := p.RequiredConfig()
	for _, key := range required {
		val := config[key]
		if val == "" {
			return fmt.Errorf("mercadopago: missing required config key %q: %w", key, errors.ErrInvalidInput)
		}
	}
	return nil
}

// CreateCharge creates a payment with Mercado Pago
func (p *Provider) CreateCharge(ctx context.Context, req provider.ChargeRequest) (*provider.ChargeResult, error) {
	// Extract access token from config
	accessToken := req.Config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("mercadopago.CreateCharge: access_token is required: %w", errors.ErrInvalidInput)
	}

	// Build the Mercado Pago request
	mpReq := buildCreatePaymentRequest(req)

	// Call Mercado Pago API
	mpPayment, err := p.client.CreatePayment(ctx, mpReq, accessToken, req.IdempotencyKey)
	if err != nil {
		slog.Error("mercadopago: CreatePayment failed", "error", err, "tenant_id", "context")
		return nil, fmt.Errorf("mercadopago.CreateCharge: %w", errors.ErrProviderError)
	}

	// Map the response to payd's domain type
	result, err := mapToChargeResult(mpPayment)
	if err != nil {
		return nil, fmt.Errorf("mercadopago.CreateCharge: mapping result: %w", err)
	}

	return result, nil
}

// RefundCharge is not implemented for Mercado Pago due to an interface limitation
//
// KNOWN LIMITATION:
// The ProviderAdapter in adapter.go:38-50 accepts a config map[string]string
// but does not propagate it to this method. The PaymentProvider interface's
// RefundCharge signature is (ctx context.Context, chargeID string, amount int64),
// with no config parameter. Therefore, the connector has no access to access_token
// at refund time, making it impossible to call the Mercado Pago refund API.
//
// To fix:
// 1. Update the PaymentProvider interface to include config: (ctx, chargeID, amount, config)
// 2. Update ProviderAdapter.RefundCharge to pass config through
// 3. Then implement this method as: POST /v1/payments/{chargeID}/refunds with body {"amount": centavosToUnits(amount)}
//
// See: https://github.com/DelgadoElias/billax/issues/TODO
func (p *Provider) RefundCharge(ctx context.Context, chargeID string, amount int64) (*provider.RefundResult, error) {
	return nil, fmt.Errorf(
		"mercadopago.RefundCharge: not implemented — "+
			"the PaymentProvider interface does not propagate per-call credentials to this method: %w",
		errors.ErrNotSupported,
	)
}

// HandleWebhook processes incoming webhook notifications from Mercado Pago
//
// CONVENTION:
// The signature parameter must be encoded as: "<webhook_secret>|<x_request_id>|<raw_x_signature_header>"
// where:
//   - webhook_secret: the tenant's webhook secret (from config)
//   - x_request_id: the value of the x-request-id header from the HTTP request
//   - raw_x_signature_header: the value of the x-signature header from Mercado Pago
//
// The HTTP webhook handler must extract these three values from the request
// and assemble them before calling the adapter.
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*provider.WebhookEvent, error) {
	// Parse the signature parameter — expect exactly 3 pipe-separated parts
	parts := strings.Split(signature, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf(
			"mercadopago.HandleWebhook: signature parameter must be 'secret|requestID|rawSignature': %w",
			errors.ErrInvalidInput,
		)
	}

	secret := parts[0]
	requestID := parts[1]
	rawSignature := parts[2]

	// Validate signature and parse the webhook event
	event, err := validateAndParseWebhook(payload, rawSignature, secret, requestID)
	if err != nil {
		return nil, fmt.Errorf("mercadopago.HandleWebhook: %w", errors.ErrInvalidInput)
	}

	return event, nil
}
