# Mercado Pago Payment Provider

This directory contains the Mercado Pago payment connector for payd, implementing the `PaymentProvider` interface.

## Overview

The Mercado Pago connector handles payment creation, refund processing, and webhook validation. It is stateless and production-ready for plan-based billing workflows.

**Status:** Week 3 complete. Pay-per-use and refunds planned for future enhancements.

## Architecture

The connector is split across four files:

- **`provider.go`** — Implements the `PaymentProvider` interface, delegates to sub-components
- **`client.go`** — HTTP transport layer with retry logic and MP wire types
- **`mapper.go`** — Converts between Mercado Pago types and payd domain types
- **`webhook.go`** — Webhook signature validation and event parsing
- **`provider_test.go`** — Comprehensive unit tests with HTTP mocking
- **`integration_test.go`** — Integration test with provider adapter

## Configuration

Required config keys (must be non-empty):

| Key | Purpose | Where to find |
|-----|---------|---|
| `access_token` | Bearer token for API calls | Mercado Pago dashboard → Settings → Integrations |
| `webhook_secret` | Signature validation secret | Mercado Pago dashboard → Webhooks configuration |

Example config in tenant settings:

```json
{
  "access_token": "APP_USR_1234567890123456789...",
  "webhook_secret": "whsec_1234567890..."
}
```

## Usage

The connector is registered automatically at startup in `cmd/payd/main.go`:

```go
registry.Register(mercadopago.New())
adapter := provider.NewAdapter(registry, yamlCaps)
```

### Creating a payment

```go
result, err := adapter.CreateCharge(ctx, "mercadopago", provider.ChargeRequest{
    Amount:             150,      // 150 centavos = 1.50 ARS
    Currency:           "ARS",
    Description:        "Monthly subscription",
    ExternalCustomerID: "user@example.com",  // Used as payer email
    IdempotencyKey:     "sub_1234_2024-04",   // Ensures idempotency
    Config: map[string]string{
        "access_token":   "APP_USR_...",
        "webhook_secret": "whsec_...",
    },
})
if err != nil {
    // Handle error
}
// result.ProviderChargeID contains the MP payment ID
// result.Status will be ChargeStatusSucceeded, ChargeStatusPending, or ChargeStatusFailed
```

### Handling webhooks

Mercado Pago sends webhook notifications to your endpoint. The HTTP handler must:

1. Extract the `x-signature` header from the request
2. Extract the `x-request-id` header
3. Read the raw request body
4. Assemble the signature parameter: `"<webhook_secret>|<x-request-id>|<x-signature_header_value>"`
5. Call the adapter

```go
// In your HTTP webhook handler:
xSignature := r.Header.Get("X-Signature")
xRequestID := r.Header.Get("X-Request-ID")
body, _ := io.ReadAll(r.Body)

signature := fmt.Sprintf("%s|%s|%s", config["webhook_secret"], xRequestID, xSignature)
event, err := adapter.HandleWebhook(ctx, "mercadopago", body, signature)
if err != nil {
    // Signature validation failed
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}

// event.ProviderChargeID contains the payment ID
// Fetch the payment to get current status: GET /v1/payments/{ProviderChargeID}
```

## Known Limitations

### 1. RefundCharge is not implemented

**Status:** ❌ Not supported in Week 3

**Reason:** The `ProviderAdapter` in `adapter.go` accepts a `config map[string]string` parameter but does not propagate it to the `PaymentProvider.RefundCharge()` method. The connector has no way to receive the `access_token` at refund time, making it impossible to call the Mercado Pago refund API.

**Workaround:** For now, tenants must process refunds manually through the Mercado Pago dashboard or via a separate admin API.

**Fix (for Week 4):**
1. Update the `PaymentProvider` interface to include config: `RefundCharge(ctx, chargeID, amount, config)`
2. Update `ProviderAdapter.RefundCharge` to pass config through
3. Implement refunds in the connector

### 2. Webhook signature convention requires pre-processing

**Status:** ⚠️ Requires HTTP handler coordination

**Issue:** The `PaymentProvider.HandleWebhook` interface receives only `payload []byte` and `signature string`. Mercado Pago's validation requires three pieces of context:
- Webhook secret (from tenant config)
- Request ID (`x-request-id` header)
- Signature (`x-signature` header)

**Solution:** The HTTP webhook handler must assemble these into a convention: `"<secret>|<requestID>|<rawSignature>"` before passing to the adapter.

This is documented but requires developer awareness. A future improvement is to update the interface to pass context data separately.

### 3. Webhook status is always pending

**Status:** ⚠️ Ping notifications only

**Issue:** Mercado Pago webhooks are ping-only notifications. The notification includes the payment ID but not the current status. The `WebhookEvent.Status` is set to `ChargeStatusPending` as a signal.

**Implication:** After receiving a webhook, the caller must perform a follow-up `GET /v1/payments/{ProviderChargeID}` to fetch the actual payment status.

This is intentional and documented in `webhook.go`. A future `FetchCharge` method on the interface would eliminate this round-trip.

### 4. ExternalCustomerID is used as payer email

**Status:** ⚠️ Design assumption

**Issue:** Mercado Pago requires a `payer.email` for most payment methods. The `ChargeRequest.ExternalCustomerID` is assumed to contain the customer email.

**Implication:** Tenants that store non-email identifiers (e.g., user UUIDs) in `ExternalCustomerID` will receive validation errors from Mercado Pago.

**Recommendation:** Tenants should ensure `ExternalCustomerID` contains the payer's email address.

## Amount Conversion

Payd stores monetary amounts as **centavos** (int64):
- 100 centavos = 1.00 primary currency unit
- 150 centavos = 1.50 ARS
- 1 centavo = 0.01 ARS

Mercado Pago APIs expect amounts in **primary currency units** (float64):
- 1.00
- 1.50
- 0.01

The connector automatically converts:
- **Outbound:** `centavosToUnits(100) → 1.0`
- **Inbound:** `unitsToCentavos(1.50) → 150`

Conversions are accurate to the cent (no floating-point surprises).

## Testing

### Unit tests

All methods have comprehensive unit test coverage with HTTP mocking:

```bash
go test ./internal/provider/mercadopago/... -v
```

Tests include:
- Happy path (approved, pending, rejected payments)
- Amount conversion (centavos ↔ primary units)
- Retry logic (429 rate limit with backoff)
- Config validation (missing/empty keys)
- Webhook signature validation (valid, invalid, malformed)
- Conversion helpers (centavos, units, round-trip)

### Integration tests

The connector integrates with the provider registry and adapter:

```bash
go test ./internal/provider/mercadopago/... -v -run TestAdapterIntegration
```

This verifies:
- Provider registration and lookup
- Capability resolution
- Config validation through adapter
- Error handling (not supported features)
- Default capability fallback for unknown providers

## Retry Logic

The HTTP client retries transient failures with exponential backoff:

- **Max attempts:** 4 (3 retries)
- **Base delay:** 1 second
- **Delays:** 1s, 2s, 4s (each doubled from base)
- **Jitter:** up to 50% of each delay, randomized
- **Conditions:** Only retry on 429 (rate limit) and network errors, not 4xx/5xx
- **Context:** Respects `ctx.Done()` and aborts immediately

Example: If MP returns 429, the client backs off and retries. If successful on retry 2, returns that result. If all 4 attempts are 429, returns error.

## Security

- **No raw card data** — only brand, type, last 4 digits stored in `PaymentMethodInfo`
- **Provider response never exposed** — stored in internal `payment_response` JSONB column, not in API responses
- **Webhook signatures validated** — HMAC-SHA256 with constant-time comparison
- **Access token never logged** — filtered from logs via slog filtering (tenant setup)
- **IdempotencyKey used** — all charge creations are idempotent (duplicate detection via DB unique constraint)

## Performance

- **Timeouts:** 30 seconds per HTTP request to Mercado Pago
- **No connection pooling configured** — uses Go's default HTTP transport (auto-pooled)
- **No caching** — each call is fresh (idempotency via API, not cache)
- **Webhook validation:** ~1ms (HMAC-SHA256 is fast)

For high-volume deployments, consider:
- Connection pooling tuning in `Client.httpClient`
- Request batching (not yet supported by MP API in basic mode)

## Future Enhancements

- [ ] **Week 4:** Implement `RefundCharge` (requires interface update)
- [ ] **Week 4:** Add `FetchCharge` method to fetch current payment status (eliminate webhook ping-pong)
- [ ] **Week 5:** Support subscription-based billing (MP has a subscriptions API)
- [ ] **Week 5:** Implement 3DS / advanced payment method handling
- [ ] **Beyond:** Support Mercado Pago's installments feature

## References

- [Mercado Pago Payments API](https://www.mercadopago.com.ar/developers/en/reference/payments/_payments/post)
- [Webhook Signature Validation](https://www.mercadopago.com.ar/developers/en/guides/webhooks/security)
- [PaymentProvider Interface](../provider.go)
- [Provider Adapter](../adapter.go)

## Support

For issues or questions:
1. Check the test cases in `provider_test.go` — they cover most scenarios
2. Review the error messages — they include context about what failed
3. Check Mercado Pago's sandbox API response in the `RawResponse` field
4. Open an issue with logs (redact `access_token` and `webhook_secret`)
