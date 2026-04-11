# 04-subscriptions — Subscription Lifecycle and Management

**Objective:** Verify subscription creation, listing, filtering, tagging, and cancellation.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed ([00-setup/setup.md](../00-setup/setup.md))
- Tenant created ([02-tenants/tenants.md](../02-tenants/tenants.md))
- Plans created ([03-plans/plans.md](../03-plans/plans.md))
- Valid API key obtained
- Application running on `http://localhost:8080`

---

## Setup

```bash
export API_KEY="payd_test_..."
export TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export BASE_URL="http://localhost:8080"
export PLAN_ID="..."  # From 03-plans test
export IDEMPOTENCY_KEY=$(uuidgen)  # Generate unique key for each test
```

---

## Test Cases

### 4.1 Create Subscription with Plan (Plan-based Billing)

**Objective:** Create a subscription linked to an existing plan.

**Step:**
```bash
export IDEMPOTENCY_KEY=$(uuidgen)

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "plan_slug": "basic-plan",
    "external_customer_id": "customer-001@example.com",
    "provider_name": "mercadopago"
  }' \
  $BASE_URL/v1/subscriptions | jq .
```

**Expected Result:**
- HTTP Status: `201`
- Response includes: `subscription_key` (UUIDv7), `status: "active"`, `plan_id`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

**Store Response:**
```
subscription_key_1 = (from response.subscription_key)
```

---

### 4.2 Idempotency: Create Same Subscription Again (200 OK)

**Objective:** Re-submit with same Idempotency-Key. Should return 200 with existing subscription.

**Step:**
```bash
# Use the SAME Idempotency-Key as 4.1

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "plan_slug": "basic-plan",
    "external_customer_id": "customer-001@example.com",
    "provider_name": "mercadopago"
  }' \
  $BASE_URL/v1/subscriptions | jq .
```

**Expected Result:**
- HTTP Status: `200` (not 201)
- Same `subscription_key` as first request
- `created_at` unchanged

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.3 Create Planless Subscription (Pay-Per-Use)

**Objective:** Create a subscription without a plan (custom amount/currency/interval).

**Step:**
```bash
export IDEMPOTENCY_KEY_2=$(uuidgen)

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY_2" \
  -d '{
    "external_customer_id": "customer-002@example.com",
    "amount": 50000,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1,
    "provider_name": "mercadopago"
  }' \
  $BASE_URL/v1/subscriptions | jq .
```

**Expected Result:**
- HTTP Status: `201`
- `plan_id: null` (planless)
- `amount: 50000`, `currency: "ARS"`, `interval: "month"`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

**Store Response:**
```
subscription_key_2 = (from response.subscription_key)
```

---

### 4.4 List Subscriptions

**Objective:** List all subscriptions for the tenant.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?limit=10" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response includes `data` array with subscriptions (should include 4.1 and 4.3)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.5 Filter Subscriptions by Status

**Objective:** Filter subscriptions by status (e.g., only active).

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?status=active" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- All returned subscriptions have `status: "active"`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.6 Add Tags to Subscription

**Objective:** Update subscription with tags (via PATCH).

**Step:**
```bash
curl -s -X PATCH \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "tags": ["vip", "early-adopter"]
  }' \
  "$BASE_URL/v1/subscriptions/$subscription_key_1" | jq '.tags'
```

**Expected Result:**
```json
["vip", "early-adopter"]
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.7 Filter Subscriptions by Tags

**Objective:** List subscriptions filtered by tags (AND semantics).

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?tag=vip&tag=early-adopter" | jq '.data | length'
```

**Expected Result:** At least 1 (the subscription from 4.6)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.8 Get Subscription with Embedded Payments

**Objective:** Retrieve a subscription and verify the last 10 payments are embedded.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions/$subscription_key_1" | jq '{subscription_key: .subscription_key, status: .status, payments: .payments}'
```

**Expected Result:**
```json
{
  "subscription_key": "...",
  "status": "active",
  "payments": []  // (or array of recent payments if any exist)
}
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.9 Update Subscription Metadata

**Objective:** Update subscription metadata via PATCH.

**Step:**
```bash
curl -s -X PATCH \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "custom_field": "custom_value",
      "order_id": "ORD-12345"
    }
  }' \
  "$BASE_URL/v1/subscriptions/$subscription_key_1" | jq '.metadata'
```

**Expected Result:**
```json
{
  "custom_field": "custom_value",
  "order_id": "ORD-12345"
}
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.10 Cancel Subscription Immediately

**Objective:** Cancel a subscription right away.

**Step:**
```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "at_period_end": false
  }' \
  "$BASE_URL/v1/subscriptions/$subscription_key_1/cancel" | jq '{status: .status, canceled_at: .canceled_at}'
```

**Expected Result:**
- HTTP Status: `200`
- `status: "canceled"`
- `canceled_at` is set to current time

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 4.11 Cancel Subscription at Period End

**Objective:** Schedule a subscription to cancel at the end of the current billing period.

**Step:**
```bash
export IDEMPOTENCY_KEY_3=$(uuidgen)

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY_3" \
  -d '{
    "plan_slug": "basic-plan",
    "external_customer_id": "customer-003@example.com",
    "provider_name": "mercadopago"
  }' \
  $BASE_URL/v1/subscriptions | jq .subscription_key > /tmp/sub_key_3.txt

export SUB_KEY_3=$(cat /tmp/sub_key_3.txt | tr -d '"')

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "at_period_end": true
  }' \
  "$BASE_URL/v1/subscriptions/$SUB_KEY_3/cancel" | jq '{status: .status, cancel_at_period_end: .cancel_at_period_end}'
```

**Expected Result:**
```json
{
  "status": "active",
  "cancel_at_period_end": true
}
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Subscription Lifecycle Status

| Test | Status | Notes |
|------|--------|-------|
| Create with plan | 🔴 | |
| Idempotency | 🔴 | |
| Create planless | 🔴 | |
| List subscriptions | 🔴 | |
| Filter by status | 🔴 | |
| Add tags | 🔴 | |
| Filter by tags | 🔴 | |
| Get with payments | 🔴 | |
| Update metadata | 🔴 | |
| Cancel immediate | 🔴 | |
| Cancel at period end | 🔴 | |

### Test Subscriptions Created

```
1. subscription_key_1 (plan-based)
   - Plan: basic-plan
   - Customer: customer-001@example.com
   - Tags: vip, early-adopter
   - Status: canceled
   
2. subscription_key_2 (planless)
   - Amount: 50000 ARS
   - Customer: customer-002@example.com
   - Status: active
   
3. subscription_key_3 (plan-based)
   - Plan: basic-plan
   - Customer: customer-003@example.com
   - Status: active (cancel_at_period_end: true)
```

### Issues Found

(to be filled during testing)

### Notes

- Subscription keys are UUIDv7 (time-ordered, can be used for sorting)
- Tags are stored as native PostgreSQL TEXT[] (not JSONB) for efficiency
- Filtering by multiple tags uses AND semantics (subscription must have all requested tags)
- Subscription enrichment includes the last 10 payments automatically

---

## Ready for Next Tests

Once subscriptions are verified, proceed to [05-payments/payments.md](../05-payments/payments.md) to test payment recording.
