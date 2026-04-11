# 06-provider-credentials — Provider Credential Configuration

**Objective:** Verify that provider credentials (access tokens, webhook secrets) can be stored and retrieved securely.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed ([00-setup/setup.md](../00-setup/setup.md))
- Tenant created ([02-tenants/tenants.md](../02-tenants/tenants.md))
- Valid API key obtained
- Application running on `http://localhost:8080`

**Credentials to Use (Sandbox):**
```
provider_name: mercadopago
access_token: TEST-8194488031946085-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-286672332
webhook_secret: ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49
```

---

## Setup

```bash
export API_KEY="payd_test_..."
export TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export BASE_URL="http://localhost:8080"
export MP_ACCESS_TOKEN="TEST-8194488031946085-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-286672332"
export MP_WEBHOOK_SECRET="ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49"
```

---

## Test Cases

### 6.1 Store Mercado Pago Credentials

**Objective:** Store access token and webhook secret for Mercado Pago.

**Step:**
```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"access_token\": \"$MP_ACCESS_TOKEN\",
    \"webhook_secret\": \"$MP_WEBHOOK_SECRET\"
  }" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200` or `201`
- Response indicates successful storage
- Credentials are NOT returned in the response (security)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.2 Check Credentials Status (Without Exposing Secrets)

**Objective:** Verify credentials are configured without exposing the actual secrets.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response shows `provider: "mercadopago"`, `configured: true`
- Does NOT include `access_token` or `webhook_secret` in response

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.3 List Configured Providers

**Objective:** List all providers with credentials configured for this tenant.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/provider-credentials" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response includes list of configured providers
- Shows at least `mercadopago` with `configured: true`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.4 Update Credentials

**Objective:** Update Mercado Pago credentials (e.g., rotate token).

**Step:**
```bash
export NEW_MP_ACCESS_TOKEN="TEST-newtoken123"

curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"access_token\": \"$NEW_MP_ACCESS_TOKEN\",
    \"webhook_secret\": \"$MP_WEBHOOK_SECRET\"
  }" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Credentials updated successfully

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.5 Validate Credentials Format

**Objective:** Attempt to store invalid credentials and verify rejection.

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"access_token\": \"\",
    \"webhook_secret\": \"\"
  }" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `422` (Unprocessable Entity)
- Error message about empty/invalid credentials

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.6 Delete Credentials

**Objective:** Remove stored credentials.

**Step:**
```bash
curl -s -X DELETE \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200` or `204`
- Credentials are deleted

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.7 Verify Deletion

**Objective:** Confirm credentials are removed.

**Step:**
```bash
curl -s \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200`
- Response shows `configured: false`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 6.8 Re-store Credentials (For Mercado Pago Tests)

**Objective:** Store credentials again for use in real payment tests.

**Step:**
```bash
curl -s -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"access_token\": \"$MP_ACCESS_TOKEN\",
    \"webhook_secret\": \"$MP_WEBHOOK_SECRET\"
  }" \
  "$BASE_URL/v1/provider-credentials/mercadopago" | jq .
```

**Expected Result:**
- HTTP Status: `200` or `201`
- Credentials re-stored successfully

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Provider Credentials Status

| Test | Status | Notes |
|------|--------|-------|
| Store credentials | 🔴 | |
| Check status | 🔴 | |
| List providers | 🔴 | |
| Update credentials | 🔴 | |
| Validate format | 🔴 | |
| Delete credentials | 🔴 | |
| Verify deletion | 🔴 | |
| Re-store | 🔴 | |

### Security Observations

- [ ] Credentials are never returned in API responses
- [ ] Credentials are stored encrypted in the database (JSONB config column)
- [ ] Only token prefix is visible in logs (not full token)
- [ ] Secrets not exposed in error messages

### Issues Found

(to be filled during testing)

### Notes

- Credentials are stored per-tenant and per-provider (unique constraint: `tenant_id, provider_name`)
- The API does not expose the actual credentials in any response (only status)
- Validation is performed when credentials are stored

---

## Ready for Next Tests

Once credentials are stored, proceed to [07-mercadopago/mercadopago.md](../07-mercadopago/mercadopago.md) to test real payment creation with Mercado Pago.
