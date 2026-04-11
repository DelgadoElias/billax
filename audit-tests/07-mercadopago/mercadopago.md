# 07-mercadopago — Mercado Pago Sandbox Integration

**Objective:** Verify complete payment flow with Mercado Pago sandbox: credential configuration, charge creation, and status tracking.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed ([00-setup/setup.md](../00-setup/setup.md))
- Provider credentials stored ([06-provider-credentials/provider-credentials.md](../06-provider-credentials/provider-credentials.md))
- Valid API key obtained
- Application running on `http://localhost:8080`
- ngrok tunnel active (for webhook testing in next section)

**Sandbox Credentials:**
```
access_token: TEST-8194488031946085-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-286672332
webhook_secret: ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49
```

**Test Cards (from MP sandbox docs):**
```
Visa:       4111 1111 1111 1111  CVV: 123  Exp: 11/25
Mastercard: 5425 2334 3010 9903  CVV: 123  Exp: 11/25
```

---

## Setup

```bash
export API_KEY="payd_test_..."
export BASE_URL="http://localhost:8080"
export SUBSCRIPTION_KEY="..."  # From 04-subscriptions test
export MP_ACCESS_TOKEN="TEST-8194488031946085-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-286672332"

# Get ngrok public URL
export NGROK_PUBLIC_URL=$(curl -s http://localhost:4040/api/tunnels | jq -r '.tunnels[0].public_url')
echo "ngrok URL: $NGROK_PUBLIC_URL"
```

---

## Test Cases

### 7.1 Create Payment via Mercado Pago

**Objective:** Create a real payment in MP sandbox and receive a charge ID.

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
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments" | jq .
```

**Expected Result:**
- HTTP Status: `201`
- Response includes:
  - `provider_charge_id`: MP payment ID (numeric string from MP API)
  - `status`: `pending` or `succeeded`
  - `payment_method`: Card info (brand, last_four, etc.)
  - `provider_name`: "mercadopago"

**Actual Result:**
```
(to be filled during testing)
```

**Store Response:**
```
provider_charge_id = (from response.provider_charge_id)
payment_id = (from response.id)
```

**Status:** 🔴 Pending

---

### 7.2 Verify Payment in Mercado Pago Dashboard

**Objective:** Confirm the payment appears in the MP sandbox dashboard.

**Steps:**
1. Log in to Mercado Pago sandbox dashboard at `https://sandbox.mercadopago.com/`
2. Navigate to **Movements** or **Payments** section
3. Look for the transaction with amount 999.00 ARS (99900 centavos)

**Expected Result:**
- Payment appears in the dashboard
- Status matches the response from payd

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 7.3 Payment Status Query (GET Details)

**Objective:** Retrieve payment details and verify status from payd.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments/$payment_id" | jq '{id, status, provider_charge_id, amount, currency}'
```

**Expected Result:**
- HTTP Status: `200`
- Payment details match the created payment
- `status` is either `pending` or `succeeded`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 7.4 Retry Idempotency with Same Key

**Objective:** Verify that retrying the same payment creation returns the existing payment.

**Step:**
```bash
# Use the SAME Idempotency-Key as 7.1

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 99900,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments" | jq '{id, provider_charge_id, status}'
```

**Expected Result:**
- HTTP Status: `200` (not 201)
- Same `id` and `provider_charge_id` as the original

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 7.5 Verify Payment Method Metadata

**Objective:** Check that payment method info is captured (card brand, last 4, etc.) without exposing full card number.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments/$payment_id" | jq '.payment_method'
```

**Expected Result:**
```json
{
  "brand": "visa",
  "last_four": "1111"
}
```

Or similar non-sensitive card metadata. NO full PAN, NO CVV.

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 7.6 Create Payment with Planless Subscription

**Objective:** Create a payment for a planless (pay-per-use) subscription via Mercado Pago.

**Step:**
```bash
# First, create a planless subscription (from 04-subscriptions test, if not already done)
export IDEMPOTENCY_KEY_SUB=$(uuidgen)
export PLANLESS_SUB_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY_SUB" \
  -d '{
    "external_customer_id": "audit-customer@example.com",
    "amount": 50000,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1,
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions")

export PLANLESS_SUB_KEY=$(echo $PLANLESS_SUB_RESPONSE | jq -r '.subscription_key')

# Now create a payment for it
export IDEMPOTENCY_KEY_PAY=$(uuidgen)

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY_PAY" \
  -d '{
    "amount": 50000,
    "currency": "ARS",
    "provider_name": "mercadopago"
  }' \
  "$BASE_URL/v1/subscriptions/$PLANLESS_SUB_KEY/payments" | jq .
```

**Expected Result:**
- HTTP Status: `201`
- Payment created successfully for planless subscription

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 7.7 Verify External Reference in Mercado Pago

**Objective:** Confirm that the payment reference in MP matches the subscription key.

**Steps:**
1. In MP sandbox dashboard, find the payment created in 7.1
2. Check the "External Reference" or "Order ID" field
3. It should match the `subscription_key` from payd

**Expected Result:**
- External reference matches the subscription key

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Mercado Pago Integration Status

| Test | Status | Notes |
|------|--------|-------|
| Create payment | 🔴 | |
| Verify in MP dashboard | 🔴 | |
| Get payment details | 🔴 | |
| Retry idempotency | 🔴 | |
| Verify payment method | 🔴 | |
| Planless payment | 🔴 | |
| External reference | 🔴 | |

### Payments Created in Sandbox

```
1. MP Payment (subscription-based)
   - Provider Charge ID: (to be filled)
   - Amount: 999.00 ARS
   - Subscription Key: (from test)
   - Status: (pending/succeeded)
   - Idempotency Key: (to be filled)

2. MP Payment (planless)
   - Provider Charge ID: (to be filled)
   - Amount: 500.00 ARS
   - Subscription Key: (to be filled)
   - Status: (pending/succeeded)
```

### Issues Found

(to be filled during testing)

### Security & PCI Compliance Notes

- ✅ No raw card data stored in payd
- ✅ Only sanitized payment method metadata (brand, last_four) exposed via API
- ✅ Card information stays within MP sandbox
- ✅ Idempotency prevents duplicate charges
- ✅ Provider response stored internally but not exposed to clients

### Known Limitations

- Refund endpoint is not implemented (returns `ErrNotSupported`)
- Refunds must be processed via MP dashboard manually
- Webhook status is "ping-only" — MP sends event notification; payd must query MP to get actual payment status

---

## Ready for Next Tests

Once Mercado Pago payments are verified, proceed to [08-webhooks/webhooks.md](../08-webhooks/webhooks.md) to test webhook handling.
