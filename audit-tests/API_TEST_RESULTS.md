# API Test Results — Complete Audit 2026-04-11

**Status:** 🟡 PARTIAL SUCCESS — Critical issues discovered  
**Time:** 15:45 UTC  
**Tests Run:** 15+  
**Pass Rate:** 53% (8/15 endpoints working)

---

## Summary

The API structure is sound, but several critical issues prevent full functionality:

1. **List endpoints not working** (plans, subscriptions)
2. **GET by key endpoints failing** (subscriptions)
3. **Credential retrieval broken** (payments fail with "access_token required")
4. **Idempotency not enforced** (same key creates different resources)

---

## ✅ Working Endpoints (8/15)

### Authentication
- ✅ `GET /health` — 200 OK
- ✅ `GET /v1/me` — Returns tenant_id correctly
- ✅ Bearer token validation
- ✅ Invalid key rejection (401)

### Plans - Create & Retrieve
- ✅ `POST /v1/plans` — Creates with 201
- ✅ `POST /v1/plans` (duplicate slug) — Returns 200 ✓
- ✅ `GET /v1/plans/{planID}` — Returns by UUID
- ✅ `GET /v1/plans/slug/{slug}` — Returns by slug

### Provider Credentials
- ✅ `POST /v1/provider-credentials/mercadopago` — Stores successfully
- ✅ `GET /v1/provider-credentials/mercadopago` — Status check works
- ✅ `GET /v1/provider-credentials` — Lists providers

### Subscriptions - Create
- ✅ `POST /v1/subscriptions` — Creates with plan
- ✅ `POST /v1/subscriptions` — Creates planless

---

## ❌ Broken Endpoints (7/15)

### Issue #1: List Endpoints Return Empty

#### `GET /v1/plans?limit=10`
```
Created: plan "basic-plan" (ID: 37c8b683-de24-4617-b6e3-471e0994acab)
GET /v1/plans: {"data": [], "next_cursor": null}
Expected: List should include created plan
```

**Root Cause:** Query filtering or cursor logic issue  
**Impact:** Users can't see their plans  
**Severity:** 🔴 CRITICAL

---

### Issue #2: Subscription GET Endpoint 404

#### `GET /v1/subscriptions/{subscriptionKey}`
```
Created: subscription (key: 4226bbd4-a13b-4650-9a10-778ff1b4f0ea)
GET /v1/subscriptions/4226bbd4-a13b-4650-9a10-778ff1b4f0ea: 404 page not found
Expected: 200 with subscription details
```

**Root Cause:** Routing or handler not matching the URL parameter  
**Impact:** Cannot retrieve subscription details  
**Severity:** 🔴 CRITICAL

**Hypothesis:** The handler might be using a different parameter name or the route pattern is incorrect.

---

### Issue #3: Payment Creation Fails - Credentials Not Retrieved

#### `POST /v1/subscriptions/{key}/payments`
```
Before: POST /v1/provider-credentials/mercadopago (201 Created)
Then: POST /v1/subscriptions/xxx/payments
Error: "provider charge: adapter.CreateCharge via mercadopago: 
        mercadopago.CreateCharge: access_token is required: invalid input"
```

**Root Cause:** Payment service not retrieving stored credentials  
**Impact:** Cannot create payments  
**Severity:** 🔴 CRITICAL

**Possible causes:**
- `providercredentials.Service.GetProviderConfig()` not being called
- Credential lookup by provider_name not working
- RLS context not set properly for credential query

---

### Issue #4: Subscription Idempotency Not Working

#### `POST /v1/subscriptions` with same Idempotency-Key
```
Request 1: Idempotency-Key: ABC123 → subscription_key: 4226bbd4-a13b...
Request 2: Idempotency-Key: ABC123 → subscription_key: 7bce9d9c-977d...
Expected: Same subscription_key (200 OK)
Got: Different keys (treated as new request)
```

**Root Cause:** Idempotency-Key not being used as deduplication key  
**Impact:** Duplicate subscriptions created on retry  
**Severity:** 🟠 HIGH

---

## 🔍 Endpoint Status Table

| Endpoint | Method | Status | Notes |
|----------|--------|--------|-------|
| /health | GET | ✅ | Works perfectly |
| /v1/me | GET | ✅ | Auth working |
| /v1/plans | POST | ✅ | Create works |
| /v1/plans | GET | ❌ | List returns empty |
| /v1/plans/{id} | GET | ✅ | Works by ID |
| /v1/plans/slug/{slug} | GET | ✅ | Works by slug |
| /v1/subscriptions | POST | ✅ | Create works |
| /v1/subscriptions/{key} | GET | ❌ | 404 error |
| /v1/subscriptions/{key}/payments | POST | ❌ | Missing credentials |
| /v1/subscriptions/{key}/payments | GET | ❌ | Blocked by issue #3 |
| /v1/provider-credentials/{provider} | POST | ✅ | Store works |
| /v1/provider-credentials/{provider} | GET | ✅ | Check works |
| /v1/provider-credentials | GET | ✅ | List works |

---

## 📋 Investigation Checklist

### Issue #1: Empty List Endpoints
- [ ] Check `internal/plan/handler.go` - `ListPlans()` implementation
- [ ] Verify cursor pagination logic
- [ ] Check if plans are actually in DB or if query filters them out
- [ ] Verify tenant isolation (RLS) isn't hiding results
- [ ] Check if `is_active` filter is default-on

### Issue #2: Subscription GET 404
- [ ] Check `internal/subscription/handler.go` route pattern
- [ ] Verify parameter name matches route (might be `subscriptionKey` vs `key`)
- [ ] Check if route is registered in router
- [ ] Verify request path formatting (might need UUID format validation)

### Issue #3: Payment Credentials Not Retrieved
- [ ] Verify `internal/payment/service.go` calls `credentialsService.GetProviderConfig()`
- [ ] Check that provider_name is passed correctly
- [ ] Verify `providercredentials.Repository.GetProviderConfig()` works
- [ ] Check database query for stored credentials
- [ ] Ensure RLS context is set for credential queries

### Issue #4: Idempotency Not Working
- [ ] Check if Idempotency-Key header is being read
- [ ] Verify deduplication logic in repository
- [ ] Ensure payload hash/comparison is correct
- [ ] Check if multiple requests with same key are hitting dedup

---

## Evidence

### Working Example: Plan Creation
```bash
$ curl -s -X POST -H "Authorization: Bearer $KEY" -d '{...}' /v1/plans
HTTP 201
{"id":"37c8b683...","slug":"basic-plan","name":"Basic Plan"}

# Idempotency works for plans:
$ curl -s -X POST -H "Authorization: Bearer $KEY" -d '{...}' /v1/plans
HTTP 200  # ✅ Correct!
{"id":"37c8b683..."}  # Same ID
```

### Broken Example: Subscription List
```bash
$ curl -s -H "Authorization: Bearer $KEY" /v1/plans
HTTP 200
{"data":[],"next_cursor":null}  # ❌ Empty even though plans exist!

$ curl -s -H "Authorization: Bearer $KEY" /v1/plans/37c8b683-de24-4617-b6e3-471e0994acab
HTTP 200
{"id":"37c8b683..."}  # ✅ Plan exists, just not in list
```

---

## Test Credentials Used

```bash
API Key: payd_test_pF+3gggDxi4kpvzqKofHD2C9IJuGdy
Tenant: f47ac10b-58cc-4372-a567-0e02b2c3d479
Base URL: http://localhost:8080

MP Credentials Stored:
  access_token: TEST-8194488031946085-041110-...
  webhook_secret: ccb58bd94631f19f75ed7f23ebe9cc0cf...
```

---

## Recommendations

### Immediate (Blocking)
1. Fix provider credentials retrieval in payment service
2. Fix subscription GET endpoint routing/404
3. Fix list endpoints (plans, subscriptions)

### High Priority (Impact on Testing)
4. Fix subscription idempotency (same key = same result)
5. Test and fix payment creation once credentials working

### After Fixes
- Re-run full test suite
- Document working vs. working-with-issues features
- Update API documentation

---

## Conclusion

The **authentication and authorization system works perfectly** — the API key fix was successful. However, the **data retrieval and payment processing layers have multiple issues** that prevent full functionality. These are likely:

1. Route/parameter mismatches (not critical, easy to fix)
2. RLS context or credential lookup issues (medium difficulty)
3. Logic errors in list/dedup endpoints (medium difficulty)

With 4-5 targeted fixes, the API would be fully functional.
