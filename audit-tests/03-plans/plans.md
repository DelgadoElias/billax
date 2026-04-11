# 03-plans — Plan CRUD Operations and Idempotency

**Objective:** Verify complete CRUD operations on billing plans, including idempotency by slug.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed ([00-setup/setup.md](../00-setup/setup.md))
- Test tenant created ([02-tenants/tenants.md](../02-tenants/tenants.md))
- Valid API key obtained
- Application running on `http://localhost:8080`

---

## Setup

Store test credentials as environment variables:

```bash
export API_KEY="payd_test_..."       # From 02-tenants tests
export TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export BASE_URL="http://localhost:8080"
```

---

## Test Cases

### 3.1 Create Plan (201 Created)

**Objective:** Create a new plan via POST with plan name, amount, currency, and interval.

**Step:**
```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Basic Plan",
    "slug": "basic-plan",
    "description": "Basic monthly subscription",
    "amount": 99900,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1
  }' \
  $BASE_URL/v1/plans | jq .
```

**Expected Result:**
- HTTP Status: `201`
- Response includes: `id`, `slug`, `name`, `created_at`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

**Store Response:**
```
plan_id = (from response.id)
```

---

### 3.2 Idempotency: Create Same Plan Again (200 OK)

**Objective:** Re-submit the same plan by slug. Should return 200 OK, not 201.

**Step:**
```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Basic Plan",
    "slug": "basic-plan",
    "description": "Basic monthly subscription",
    "amount": 99900,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1
  }' \
  $BASE_URL/v1/plans | jq .
```

**Expected Result:**
- HTTP Status: `200` (not 201)
- Same plan ID returned
- `created_at` unchanged

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.3 List Plans (Pagination)

**Objective:** List all plans for the tenant with pagination.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans?limit=10" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response includes `data` array with plans
- Includes pagination info (`next_cursor` if more results)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.4 Filter Plans by is_active

**Objective:** List only active plans.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans?is_active=true" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- All returned plans have `is_active: true`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.5 Get Plan by ID

**Objective:** Retrieve a specific plan by its UUID.

**Step:**
```bash
export PLAN_ID="..."  # From 3.1 response

curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans/$PLAN_ID" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Plan details returned

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.6 Get Plan by Slug

**Objective:** Retrieve a plan by its slug (useful for payment buttons).

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans/slug/basic-plan" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Plan details returned

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.7 Update Plan (PATCH)

**Objective:** Update plan name and description (amount/currency/interval are immutable).

**Step:**
```bash
curl -s -X PATCH \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Basic Plan - Updated",
    "description": "Updated description"
  }' \
  "$BASE_URL/v1/plans/$PLAN_ID" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- `name` and `description` updated
- `amount`, `currency`, `interval` unchanged

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.8 Attempt to Update Immutable Fields (Should Fail 422)

**Objective:** Verify that amount, currency, and interval cannot be updated.

**Step:**
```bash
curl -s -X PATCH \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 199900
  }' \
  "$BASE_URL/v1/plans/$PLAN_ID" | jq .
```

**Expected Result:**
- HTTP Status: `422`
- Error message about immutable fields

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.9 Soft Delete Plan

**Objective:** Delete a plan by setting `is_active = false`.

**Step:**
```bash
curl -s -X DELETE \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans/$PLAN_ID" | jq .
```

**Expected Result:**
- HTTP Status: `204` (No Content) or `200` (if returns the plan)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 3.10 Verify Plan is Soft Deleted

**Objective:** Confirm the plan's `is_active` is false.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/plans/$PLAN_ID" | jq '.is_active'
```

**Expected Result:** `false`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Plan CRUD Status

| Test | Status | Notes |
|------|--------|-------|
| Create plan (201) | 🔴 | |
| Idempotency (200) | 🔴 | |
| List plans | 🔴 | |
| Filter by is_active | 🔴 | |
| Get by ID | 🔴 | |
| Get by slug | 🔴 | |
| Update (PATCH) | 🔴 | |
| Reject immutable update | 🔴 | |
| Soft delete | 🔴 | |
| Verify deletion | 🔴 | |

### Test Plans Created

```
1. basic-plan
   - Amount: 99900 (ARS)
   - Currency: ARS
   - Interval: 1 month
   - Status: (created/deleted)
```

### Issues Found

(to be filled during testing)

### Notes

- Plans use slug-based idempotency: if you POST the same slug twice, the second request returns 200 (not 201) with the existing plan
- Amount, currency, and interval are immutable by design
- Plans are soft-deleted (is_active flag), not hard-deleted

---

## Ready for Next Tests

Once plan CRUD is verified, proceed to [04-subscriptions/subscriptions.md](../04-subscriptions/subscriptions.md) to test subscription lifecycle.
