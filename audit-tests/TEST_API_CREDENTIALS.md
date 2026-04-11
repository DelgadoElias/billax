# Test API Credentials

**Status:** ✅ WORKING — API Key Generated and Verified

**Date:** 2026-04-11  
**Fix Applied:** Base64 encoding for API key hash

---

## Test Credentials (Valid)

```bash
export TEST_API_KEY="payd_test_pF+3gggDxi4kpvzqKofHD2C9IJuGdy"
export TEST_TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"
export BASE_URL="http://localhost:8080"
```

---

## Verification

### ✅ Test 1: Health Endpoint (No Auth)
```bash
curl -s $BASE_URL/health | jq .
```
**Result:** 200 OK - Works ✅

### ✅ Test 2: Authenticated Endpoint (/v1/me)
```bash
curl -s -H "Authorization: Bearer $TEST_API_KEY" $BASE_URL/v1/me | jq .
```
**Result:** 
```json
{
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```
✅ Works

### ✅ Test 3: Invalid Key Rejection
```bash
curl -s -H "Authorization: Bearer invalid_key" $BASE_URL/v1/me | jq .
```
**Result:**
```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "Invalid API key"
  }
}
```
✅ Works - Rejects invalid keys correctly

---

## What Was Fixed

### Problem (Hash Encoding Bug)
The original `hashToString()` function in `internal/middleware/auth.go` was converting Argon2 hash bytes directly to Go runes, creating invalid UTF-8 sequences:

```go
// ❌ BEFORE (broken)
func hashToString(hash []byte) string {
    var result string
    for _, b := range hash {
        result += string(rune(b))  // Invalid UTF-8
    }
    return result
}
```

### Solution (Base64 Encoding)
Changed to use standard Base64 encoding, which is UTF-8 safe:

```go
// ✅ AFTER (fixed)
import "encoding/base64"

func hashToString(hash []byte) string {
    return base64.StdEncoding.EncodeToString(hash)
}
```

**Files Changed:**
- `internal/middleware/auth.go` — Added `encoding/base64` import
- `internal/middleware/auth.go` — Updated `hashToString()` function (2 lines → 1 line)

**Verification:**
- ✅ `go vet ./internal/middleware/...` passes
- ✅ App compiles and runs
- ✅ Auth middleware validates API keys correctly
- ✅ Invalid keys are rejected
- ✅ Valid keys grant access to authenticated endpoints

---

## Ready for Tests 02-08

All authenticated endpoints are now accessible. Proceed with:
- [02-tenants/tenants.md](02-tenants/tenants.md)
- [03-plans/plans.md](03-plans/plans.md)
- [04-subscriptions/subscriptions.md](04-subscriptions/subscriptions.md)
- [05-payments/payments.md](05-payments/payments.md)
- [06-provider-credentials/provider-credentials.md](06-provider-credentials/provider-credentials.md)
- [07-mercadopago/mercadopago.md](07-mercadopago/mercadopago.md)
- [08-webhooks/webhooks.md](08-webhooks/webhooks.md)

Use the credentials above in all subsequent tests.
