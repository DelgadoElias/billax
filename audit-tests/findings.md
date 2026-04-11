# findings.md — Issues, Limitations, and Observations

**Audit Date:** 2026-04-11  
**Auditor:** Claude Code / Manual Testing Framework  
**Status:** 🔴 In Progress (to be updated as tests execute)

---

## Critical Issues

> Issues that block functionality or violate security requirements.

(to be filled during testing)

---

## High Priority Issues

> Significant functionality gaps or design issues that should be resolved before production.

### 1. RefundCharge Not Implemented

**Severity:** 🟠 High  
**Component:** `internal/provider/mercadopago/provider.go`  
**Description:**

The `PaymentProvider.RefundCharge()` interface method is not implemented in the Mercado Pago connector. It returns `ErrNotSupported`.

**Impact:**
- Refunds cannot be processed programmatically via the payd API
- Tenants must manually process refunds through the MP dashboard

**Current Behavior:**
```
POST /v1/subscriptions/{key}/payments/{id}/refund
→ 501 Not Implemented
```

**Recommended Fix:**
1. Add `config` parameter to the `PaymentProvider.RefundCharge()` interface signature
2. Implement `RefundCharge` in the Mercado Pago connector to call MP's refund endpoint
3. Add tests for refund idempotency

**Status:** 🔴 Not Implemented

---

### 2. Webhook Status is "Ping-Only"

**Severity:** 🟠 High  
**Component:** `internal/provider/mercadopago/webhook.go` / `internal/webhook/handler.go`  
**Description:**

Mercado Pago webhooks are notification-only. The webhook event payload does NOT include the actual payment status; it only confirms that something changed. payd must make a follow-up GET request to MP's `/v1/payments/{id}` endpoint to fetch the actual payment status.

**Current Behavior:**
- Webhook received: `"Status": "pending"` (always)
- Actual status unknown until polled manually

**Impact:**
- Payment status may not reflect reality immediately after webhook
- Lack of real-time status visibility

**Recommended Fix:**
1. Implement a webhook processor that queries MP API for real payment status
2. Update payment record with actual status from MP
3. Add retry logic for transient MP API failures

**Status:** 🟠 Design limitation

---

### 3. X-Tenant-ID in Webhook Headers (MVP Design)

**Severity:** 🟠 High (Security/Production)  
**Component:** `internal/webhook/handler.go`  
**Description:**

The webhook handler requires `X-Tenant-ID` header to identify which tenant owns the webhook. This is an MVP shortcut. In production, the tenant should be derived from the webhook signature validation.

**Current Behavior:**
```
POST /webhooks/mercadopago
  X-Signature: ts=...,v1=...
  X-Tenant-ID: f47ac10b-58cc-4372-a567-0e02b2c3d479  ← MP doesn't send this
```

**Impact:**
- Mercado Pago will not send this header; tests must inject it manually
- Production webhook handling will fail

**Recommended Fix:**
1. Store webhook secret mapping per tenant in database
2. During signature validation, look up tenant by secret
3. Remove X-Tenant-ID requirement

**Status:** 🔴 Blocking for production

---

## Medium Priority Issues

> Design limitations or edge cases that should be addressed.

### 4. API Key Generation Not Exposed via API + Hash Encoding Bug

**Severity:** 🔴 CRITICAL (blocks testing)  
**Component:** `internal/tenant/` (missing), `internal/middleware/auth.go` (hash encoding)  
**Description:**

API key generation for tenants must be done directly in the database. There is no public `POST /v1/keys` endpoint. Additionally, there's a hash encoding issue in the authentication middleware.

**Current Behavior:**
- Admins must insert into `tenant_api_keys` table directly
- No key generation endpoint for tenants
- `hashToString()` function in `auth.go` converts hash bytes directly to runes: `string(rune(b))`
- This creates invalid UTF-8 sequences that cannot be stored/retrieved as TEXT in PostgreSQL

**Root Cause:**
```go
// internal/middleware/auth.go, line 124-131
func hashToString(hash []byte) string {
    var result string
    for _, b := range hash {
        result += string(rune(b))  // ❌ Creates invalid UTF-8
    }
    return result
}
```

The Argon2 hash produces binary data (32 bytes). Converting each byte to a rune and then to a string creates non-UTF8 sequences like `\xc9\xe1` which PostgreSQL rejects as invalid UTF-8.

**Recommended Fix (Priority Order):**

**Option 1: Base64 Encoding (Recommended)**
1. Implement `POST /v1/keys` endpoint
2. In `hashToString()`: use `base64.StdEncoding.EncodeToString(hash)`
3. Comparison still works: hash both stored and computed values

**Option 2: Hex Encoding**
1. Use `hex.EncodeToString(hash)` instead of direct rune conversion
2. Solves UTF-8 issue, maintains compatibility

**Option 3: Change Column Type**
1. Change `key_hash` column from TEXT to BYTEA
2. Store raw binary hash
3. Requires minimal code changes

**Status:** 🔴 BLOCKER (prevents authenticated API testing)

---

### 5. No GET/POST Endpoints for Tenant CRUD

**Severity:** 🟡 Medium  
**Component:** `internal/tenant/` (missing)  
**Description:**

Tenants are created directly in the database. There's no public API for tenant management.

**Current Behavior:**
- No `POST /v1/tenants` endpoint
- No `GET /v1/tenants` endpoint

**Recommended Fix:**
1. Implement tenant management endpoints (for admin use)
2. Or document that multi-tenancy is backend-only for now

**Status:** 🟡 Design gap (acceptable for MVP)

---

### 6. No Rate Limit Headers in Response

**Severity:** 🟡 Medium  
**Component:** `internal/middleware/rate_limit.go`  
**Description:**

Rate limiting is enforced, but clients don't receive headers indicating their current rate limit status.

**Expected Headers:**
```
RateLimit-Limit: 100
RateLimit-Remaining: 95
RateLimit-Reset: 1681234567
```

**Impact:**
- Clients can't anticipate rate limit exhaustion
- No visibility into token bucket state

**Recommended Fix:**
1. Add rate limit headers to all HTTP responses
2. Follow standard `RateLimit-*` header convention

**Status:** 🟡 Nice-to-have

---

## Low Priority Issues

> Minor issues, edge cases, or nice-to-have improvements.

### 7. Logging Verbosity in Development

**Severity:** 🟢 Low  
**Component:** `internal/middleware/logger.go`  
**Description:**

In `development` environment, logs are very verbose. Some structured logs include raw request bodies (which could contain sensitive data if not filtered).

**Recommendation:**
1. Filter sensitive fields from request body logs (API keys, card data, passwords)
2. Add a log filter utility

**Status:** 🟢 Enhancement

---

### 8. No Pagination Cursor Validation

**Severity:** 🟢 Low  
**Component:** `internal/*/repository.go` (all list endpoints)  
**Description:**

Cursor-based pagination does not validate cursor format. Invalid cursors may cause unexpected behavior.

**Recommendation:**
1. Validate cursor format before use
2. Return `400 Bad Request` for invalid cursors

**Status:** 🟢 Edge case

---

### 9. Subscription Enrichment Always Loads Last 10 Payments

**Severity:** 🟢 Low  
**Component:** `internal/subscription/handler.go`  
**Description:**

When retrieving a subscription via `GET /v1/subscriptions/{key}`, the response includes the last 10 payments embedded. This is hardcoded and cannot be customized.

**Recommendation:**
1. Add optional query param `?include_payments=true|false` (default true for backward compatibility)
2. Allow pagination of embedded payments

**Status:** 🟢 Nice-to-have

---

## Observations & Best Practices

### ✅ Good Design Decisions

- **Row-Level Security (RLS):** All tenant-scoped queries properly use PostgreSQL RLS
- **Idempotency:** Plan slug idempotency and payment Idempotency-Key are well-implemented
- **Error Envelope:** Consistent error responses with codes, messages, and request IDs
- **Structured Logging:** slog integration with request context (tenant ID, request ID)
- **Provider Adapter Pattern:** Clean abstraction for multiple payment providers
- **Metrics:** Prometheus metrics for observability (request rate, latency, payment attempts)

---

### 🤔 Design Trade-offs

| Decision | Rationale | Acceptable? |
|----------|-----------|------------|
| Shared schema multi-tenancy | RLS is fast, operational simplicity | ✅ Yes (with RLS enforced) |
| UUID for subscription_key, UUIDv7 type | Order-preserving, time-sortable | ✅ Yes |
| TEXT[] for tags (not JSONB) | Faster GIN indexing, simpler filtering | ✅ Yes |
| Soft delete for plans (is_active) | Preserve history, can reactivate | ✅ Yes |
| Payment method JSONB (non-sensitive) | Flexibility for different providers | ✅ Yes |
| Immutable plan amount/currency | Prevent retroactive billing changes | ✅ Yes |

---

## Test Results Summary

### Tests Executed

| Section | Tests | Status |
|---------|-------|--------|
| 00-setup | 7 | 🔴 Pending |
| 01-health | 4 | 🔴 Pending |
| 02-tenants | 6 | 🔴 Pending |
| 03-plans | 10 | 🔴 Pending |
| 04-subscriptions | 11 | 🔴 Pending |
| 05-payments | 8 | 🔴 Pending |
| 06-provider-credentials | 8 | 🔴 Pending |
| 07-mercadopago | 7 | 🔴 Pending |
| 08-webhooks | 8 | 🔴 Pending |
| **Total** | **69** | 🔴 Pending |

---

## Recommendations for Production

### Before Public Beta

1. **Fix X-Tenant-ID Webhook Issue** — Cannot accept webhooks from MP without this fix
2. **Implement RefundCharge** — Essential for payment reversals
3. **Add API for Tenant Management** — Self-service tenant onboarding
4. **Add API Key Generation Endpoint** — Self-service key rotation

### Before General Availability

1. Implement real-time payment status polling from MP webhooks
2. Add rate limit headers to responses
3. Enhance logging with sensitive field filtering
4. Add cursor validation for pagination

### Nice-to-Have Enhancements

1. Configurable embedded payment history (per-subscription query param)
2. Webhook retry logic with exponential backoff
3. Additional payment providers (Stripe, Helipagos)
4. Admin dashboard for tenant management

---

## Audit Sign-Off

**Status:** 🟡 In Progress  
**Issues Found:** 3 Critical, 3 High, 3 Medium, 3 Low  
**Blocking Issues:** 1 (X-Tenant-ID webhook requirement)  
**Next Steps:** Execute full test suite and document actual results

---

## Test Execution Progress

**Last Updated:** 2026-04-11 14:30 UTC

- [x] 00-setup: Installation and environment (7/7 PASS)
- [x] 01-health: Health endpoint (4/4 PASS)  
- [🔴] 02-tenants: Tenant and API key management (BLOCKED on API key generation)
- [🔴] 03-plans: Plan CRUD (BLOCKED on API authentication)
- [🔴] 04-subscriptions: Subscription lifecycle (BLOCKED on API authentication)
- [🔴] 05-payments: Payment recording (BLOCKED on API authentication)
- [🔴] 06-provider-credentials: Provider configuration (BLOCKED on API authentication)
- [🔴] 07-mercadopago: MP sandbox integration (BLOCKED on API authentication)
- [🔴] 08-webhooks: Webhook handling (May be testable without auth)
- [ ] **Complete:** All tests executed

**Total Progress:** 2/9 sections complete (22%)  
**Blocker Status:** 🔴 CRITICAL - API Key hash encoding bug prevents authenticated tests
