# 05-payments — Payment Recording and Idempotency

**Objective:** Verify payment creation, idempotency via Idempotency-Key header, and payment history retrieval.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed ([00-setup/setup.md](../00-setup/setup.md))
- Tenant created ([02-tenants/tenants.md](../02-tenants/tenants.md))
- Subscriptions created ([04-subscriptions/subscriptions.md](../04-subscriptions/subscriptions.md))
- Valid API key obtained
- Application running on `http://localhost:8080`

**Note:** Payment creation without a provider (manual recording) or with a mocked provider are tested here. Real Mercado Pago payment integration is tested in [07-mercadopago/mercadopago.md](../07-mercadopago/mercadopago.md).

---

## Setup

```bash
export API_KEY="payd_test_..."
export TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export BASE_URL="http://localhost:8080"
export SUBSCRIPTION_KEY_1="..."  # From 04-subscriptions test
export IDEMPOTENCY_KEY=$(uuidgen)
```

---

## Test Cases

### 5.1 Create Payment (Idempotent via Idempotency-Key)

**Objective:** Create a payment record for a subscription.

**Step:**
```bash
export IDEMPOTENCY_KEY=$(uuidgen)

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 99900,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY_1/payments" | jq .
```

**Expected Result:**
- HTTP Status: `201`
- Response includes: `id`, `idempotency_key`, `status`, `provider_charge_id`, `provider_name`
- `status` may be `pending` or `succeeded` depending on provider

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

**Store Response:**
```
payment_id_1 = (from response.id)
idempotency_key_1 = $IDEMPOTENCY_KEY
```

---

### 5.2 Idempotency: Create Same Payment Again (200 OK)

**Objective:** Re-submit with the same Idempotency-Key. Should return 200 with the existing payment.

**Step:**
```bash
# Use the SAME Idempotency-Key as 5.1

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 99900,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY_1/payments" | jq .
```

**Expected Result:**
- HTTP Status: `200` (not 201)
- Same `id` and `idempotency_key` as first request
- Exact same response as the first call

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.3 Missing Idempotency-Key Header (Should Fail 400)

**Objective:** Verify that POST without Idempotency-Key is rejected.

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 99900,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY_1/payments" | jq .
```

**Expected Result:**
- HTTP Status: `400`
- Error code: `missing_idempotency_key`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.4 Duplicate Idempotency-Key Different Amount (Should Fail 409)

**Objective:** Verify conflict detection when same Idempotency-Key is used with different payload.

**Step:**
```bash
# Reuse the same IDEMPOTENCY_KEY but with different amount

curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 50000,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY_1/payments" | jq .
```

**Expected Result:**
- HTTP Status: `409` (Conflict)
- Error code: `duplicate_idempotency_key`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.5 Get Payment History for Subscription

**Objective:** List all payments for a specific subscription.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY_1/payments?limit=10" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response includes `data` array with payment records (should include payment from 5.1)
- Includes pagination info

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.6 Get All Payments (Tenant View)

**Objective:** List all payments for the entire tenant.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments?limit=10" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response includes all payments for this tenant

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.7 Filter Payments by Provider

**Objective:** List only payments from a specific provider (e.g., "mercadopago").

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments?provider=mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- All returned payments have `provider_name: "mercadopago"`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 5.8 Payment Method Metadata

**Objective:** Verify payment method metadata is captured (card brand, last 4, etc.).

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments/$payment_id_1" | jq '.payment_method'
```

**Expected Result:**
- HTTP Status: `200`
- `payment_method` includes: `brand` (visa/mastercard/etc), `last_four`, or similar non-sensitive info
- Raw provider response is NOT exposed to client

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Payment Recording Status

| Test | Status | Notes |
|------|--------|-------|
| Create payment (201) | 🔴 | |
| Idempotency (200) | 🔴 | |
| Missing Idempotency-Key | 🔴 | |
| Duplicate key conflict | 🔴 | |
| Get history for subscription | 🔴 | |
| Get all payments | 🔴 | |
| Filter by provider | 🔴 | |
| Payment method metadata | 🔴 | |

### Test Payments Created

```
1. payment_id_1
   - Subscription: $SUBSCRIPTION_KEY_1
   - Idempotency-Key: $IDEMPOTENCY_KEY
   - Amount: 99900 ARS
   - Provider: mercadopago
   - Status: (pending/succeeded/failed)
```

### Issues Found

(to be filled during testing)

### Notes

- Idempotency is enforced via `Idempotency-Key` header (not request body)
- Same key → same response (status code 200)
- Different key → new payment recorded
- Payment method metadata is sanitized (brand, last_four, etc. only, no full card numbers)
- Provider's internal response is stored in `provider_response` (JSONB) but NOT exposed to API clients

---

## Ready for Next Tests

Once payment idempotency is verified, proceed to [06-provider-credentials/provider-credentials.md](../06-provider-credentials/provider-credentials.md) to configure Mercado Pago credentials.
