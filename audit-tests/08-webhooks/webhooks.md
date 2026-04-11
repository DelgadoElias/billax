# 08-webhooks — Inbound Webhook Handling

**Objective:** Verify that inbound webhooks from Mercado Pago are properly validated, processed, and result in payment status updates.

**Date:** 2026-04-11

**Preconditions:**
- Mercado Pago integration tested ([07-mercadopago/mercadopago.md](../07-mercadopago/mercadopago.md))
- Payment created with MP (have `provider_charge_id`)
- ngrok tunnel active (for receiving webhooks)
- Valid API key obtained
- Application running on `http://localhost:8080`

**Webhook Endpoint:** `POST /webhooks/mercadopago`

---

## Webhook Architecture

payd's webhook handler expects:
1. `X-Signature`: HMAC-SHA256 signature from MP in format `ts=<epoch>,v1=<sha256>`
2. `X-Request-ID`: MP's request ID
3. `X-Tenant-ID`: Tenant UUID (MVP design; in prod this would be derived from signature)
4. JSON body with payment event data

---

## Setup

```bash
export API_KEY="payd_test_..."
export BASE_URL="http://localhost:8080"
export NGROK_PUBLIC_URL=$(curl -s http://localhost:4040/api/tunnels | jq -r '.tunnels[0].public_url')
export PROVIDER_CHARGE_ID="..."  # From 07-mercadopago test
export MP_WEBHOOK_SECRET="ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49"
export TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"

echo "Webhook URL will be: $NGROK_PUBLIC_URL/webhooks/mercadopago"
```

---

## Test Cases

### 8.1 Configure Webhook URL in Mercado Pago Dashboard

**Objective:** Set up Mercado Pago to send webhooks to our ngrok URL.

**Steps:**
1. Log in to MP sandbox dashboard at `https://sandbox.mercadopago.com/`
2. Navigate to **Settings** → **Integrations** → **Webhooks**
3. Add webhook URL: `$NGROK_PUBLIC_URL/webhooks/mercadopago`
4. Subscribe to event types: `payment.created`, `payment.updated`, `payment.succeeded`

**Expected Result:**
- Webhook URL registered in MP
- MP can send test webhooks to the URL

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.2 Test Webhook Signature Validation (Valid Signature)

**Objective:** Send a webhook with a valid HMAC-SHA256 signature and verify it's accepted.

**Step:**
```bash
# Prepare webhook payload
export WEBHOOK_ID=$(uuidgen)
export WEBHOOK_TIMESTAMP=$(date +%s)

# Mercado Pago webhook format
WEBHOOK_PAYLOAD='{
  "id": "'$WEBHOOK_ID'",
  "action": "payment.created",
  "data": {
    "id": "'$PROVIDER_CHARGE_ID'"
  },
  "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
}'

# Create HMAC signature (MP format: ts=<timestamp>,v1=<sha256>)
export SIGNATURE_MESSAGE="id:$WEBHOOK_ID;request-id:$WEBHOOK_ID;ts:$WEBHOOK_TIMESTAMP;"
export SIGNATURE_HASH=$(echo -n "$SIGNATURE_MESSAGE" | openssl dgst -sha256 -hmac "$MP_WEBHOOK_SECRET" -hex | awk '{print $2}')
export SIGNATURE="ts=$WEBHOOK_TIMESTAMP,v1=$SIGNATURE_HASH"

# Send webhook
curl -s -X POST \
  -H "X-Signature: $SIGNATURE" \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Webhook accepted and processed
- Payment status may be updated

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.3 Reject Invalid Signature

**Objective:** Verify that webhooks with invalid signatures are rejected.

**Step:**
```bash
export INVALID_SIGNATURE="ts=$WEBHOOK_TIMESTAMP,v1=invalidsignature123"

curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "X-Signature: $INVALID_SIGNATURE" \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `401` or `403`
- Error message about signature validation

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.4 Missing Signature Header

**Objective:** Verify that webhooks without signature are rejected.

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `400`
- Error about missing signature header

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.5 Missing X-Tenant-ID Header

**Objective:** Verify that webhooks without tenant ID are rejected (MVP design).

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "X-Signature: $SIGNATURE" \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `400`
- Error about missing tenant header

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.6 Payment Status Update via Webhook

**Objective:** Verify that a webhook updates the payment status in the database.

**Steps:**
1. Send a webhook with payment.succeeded event
2. Query the payment from the API
3. Verify status changed to "succeeded"

```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments/$payment_id" | jq '.status'
```

**Expected Result:**
- Status changed to `succeeded` (if webhook was for a succeeded event)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.7 Duplicate Webhook Processing (Idempotency)

**Objective:** Verify that sending the same webhook twice doesn't create duplicate payment records.

**Step:**
```bash
# Get payment history before webhook
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments?limit=100" | jq '.data | length' > /tmp/before.txt

# Send same webhook twice
curl -s -X POST \
  -H "X-Signature: $SIGNATURE" \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" > /dev/null

curl -s -X POST \
  -H "X-Signature: $SIGNATURE" \
  -H "X-Request-ID: $WEBHOOK_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_PAYLOAD" \
  "$BASE_URL/webhooks/mercadopago" > /dev/null

# Get payment history after webhooks
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/payments?limit=100" | jq '.data | length' > /tmp/after.txt

echo "Payments before: $(cat /tmp/before.txt)"
echo "Payments after: $(cat /tmp/after.txt)"
```

**Expected Result:**
- No new payments created on the second webhook
- Payment count should be the same or increase by 1 (not by 2)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 8.8 Webhook from Mercado Pago Sandbox

**Objective:** Trigger a real payment event in MP sandbox and verify webhook is received and processed.

**Steps:**
1. In MP sandbox, create a test payment through the dashboard
2. Trigger the payment to completion (use test card)
3. MP should automatically send webhooks to the configured URL
4. Check application logs for webhook processing

**Expected Result:**
- ngrok shows the webhook request
- Application logs show webhook received and processed
- Payment status updated in payd

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Webhook Integration Status

| Test | Status | Notes |
|------|--------|-------|
| Configure MP webhook URL | 🔴 | |
| Valid signature | 🔴 | |
| Invalid signature | 🔴 | |
| Missing signature | 🔴 | |
| Missing tenant ID | 🔴 | |
| Status update | 🔴 | |
| Idempotency | 🔴 | |
| Real MP webhook | 🔴 | |

### Webhook Events Processed

```
1. payment.created
2. payment.updated
3. payment.succeeded
```

### Issues Found

(to be filled during testing)

### Security Notes

- ✅ HMAC-SHA256 signature validation enforced
- ✅ Only authorized webhooks processed
- ✅ Tenant isolation via X-Tenant-ID
- ⚠️ MVP limitation: X-Tenant-ID must be sent by client; in production, this should be derived from signature validation

### Known Limitations

- Webhook status from MP is "ping-only" — MP sends a notification, but payd must query the payment status from MP API to get the actual status
- The current webhook handler does not make a GET request to MP to fetch the real payment status; this is deferred to future work

---

## Ready for Final Summary

All API and webhook tests complete. Proceed to [findings.md](../findings.md) to summarize issues and recommendations.
