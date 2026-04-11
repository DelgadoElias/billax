package mercadopago

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/provider"
)

// mpWebhookPayload is the structure of Mercado Pago's webhook notification body
type mpWebhookPayload struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`   // "payment", "plan", "subscription"
	Action string `json:"action"` // "payment.created", "payment.updated"
	Data   struct {
		ID string `json:"id"` // Payment ID as string in the data field
	} `json:"data"`
}

// validateAndParseWebhook validates the webhook signature and parses the event
// The signature parameter must be encoded as: "<webhook_secret>|<x_request_id>|<raw_x_signature_header>"
// where raw_x_signature_header is the value of the x-signature header from Mercado Pago
func validateAndParseWebhook(payload []byte, rawSignature, secret, requestID string) (*provider.WebhookEvent, error) {
	// Validate the signature
	if err := validateSignature(payload, secret, requestID, rawSignature); err != nil {
		// Wrap signature validation errors with the sentinel for webhook signature invalid
		return nil, fmt.Errorf("mercadopago.validateAndParseWebhook: %w", errors.ErrWebhookSignatureInvalid)
	}

	// Parse the webhook event
	if event, err := parseWebhookEvent(payload); err != nil {
		// Payload parsing errors are treated as invalid input
		return nil, fmt.Errorf("mercadopago.validateAndParseWebhook: %w", errors.ErrInvalidInput)
	} else {
		return event, nil
	}
}

// validateSignature validates the HMAC-SHA256 signature from Mercado Pago
// MP format: x-signature: ts=<epoch>,v1=<sha256hmac>
// HMAC message format: id:<payment_id>;request-id:<request_id>;ts:<ts>;
func validateSignature(payload []byte, secret, requestID, rawSignature string) error {
	// Parse the signature header: "ts=1234567890,v1=abcd1234"
	var ts, v1 string
	parts := strings.Split(rawSignature, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			if kv[0] == "ts" {
				ts = kv[1]
			} else if kv[0] == "v1" {
				v1 = kv[1]
			}
		}
	}

	if ts == "" || v1 == "" {
		return fmt.Errorf("mercadopago webhook: malformed signature (missing ts or v1)")
	}

	// Extract payment ID from payload
	var wh mpWebhookPayload
	if err := json.Unmarshal(payload, &wh); err != nil {
		return fmt.Errorf("mercadopago webhook: parsing payload for ID: %w", err)
	}

	paymentID := wh.Data.ID
	if paymentID == "" {
		paymentID = fmt.Sprintf("%d", wh.ID)
	}

	// Build the HMAC message per MP's format
	message := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", paymentID, requestID, ts)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	computedV1 := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	if !hmac.Equal([]byte(computedV1), []byte(v1)) {
		return fmt.Errorf("mercadopago webhook: signature validation failed")
	}

	return nil
}

// parseWebhookEvent parses the webhook payload into a WebhookEvent
// Note: MP webhooks are ping notifications only — the actual payment status must be fetched separately
func parseWebhookEvent(payload []byte) (*provider.WebhookEvent, error) {
	var wh mpWebhookPayload
	if err := json.Unmarshal(payload, &wh); err != nil {
		return nil, fmt.Errorf("mercadopago webhook: parsing payload: %w", err)
	}

	event := &provider.WebhookEvent{
		ProviderName: "mercadopago",
		EventType:    wh.Type,
		RawPayload:   payload,
	}

	if wh.Type == "payment" {
		// Extract the payment ID from the data field
		event.ProviderChargeID = wh.Data.ID
		if event.ProviderChargeID == "" {
			event.ProviderChargeID = fmt.Sprintf("%d", wh.ID)
		}

		// Important: MP webhooks are ping notifications only.
		// The status field in this event is not the actual payment status.
		// The webhook receiver must fetch the payment via GET /v1/payments/{id} to get the current status.
		// We return ChargeStatusPending as a signal that the receiver must fetch the actual status.
		event.Status = provider.ChargeStatusPending
	}

	return event, nil
}
