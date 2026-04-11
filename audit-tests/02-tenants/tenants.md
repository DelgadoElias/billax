# 02-tenants — Tenant and API Key Management

**Objective:** Verify tenant creation and API key generation for authentication.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed successfully ([00-setup/setup.md](../00-setup/setup.md))
- Health endpoint verified ([01-health/health.md](../01-health/health.md))
- Application running on `http://localhost:8080`
- PostgreSQL accessible

---

## Overview

In the current MVP design, tenants are created directly in the database. There is no public `POST /v1/tenants` endpoint. API keys are generated via the database and can be tested.

---

## Test Setup: Create Test Tenant

### Setup-1: Insert test tenant into database

**Step:**
```bash
export TEST_TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"

psql $DATABASE_URL -c "
INSERT INTO tenants (id, name, slug, created_at, updated_at)
VALUES (
  '$TEST_TENANT_ID'::uuid,
  'Audit Test Tenant',
  'audit-test',
  now(),
  now()
)
ON CONFLICT (id) DO NOTHING;

SELECT id, name, slug FROM tenants WHERE id = '$TEST_TENANT_ID'::uuid;
"
```

**Expected Result:**
```
INSERT 0 1
                  id                  | name                 | slug
--------------------------------------+----------------------+-----------
 f47ac10b-58cc-4372-a567-0e02b2c3d479 | Audit Test Tenant    | audit-test
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Test Cases

### 2.1 List Tenants (Database View)

**Objective:** Verify the test tenant was created and is retrievable from the database.

**Step:**
```bash
psql $DATABASE_URL -c "SELECT id, name, slug, created_at FROM tenants LIMIT 10;"
```

**Expected Result:** At least one tenant with ID matching `$TEST_TENANT_ID`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 2.2 API Key Generation

**Objective:** Generate a valid API key for the test tenant.

**Step:**
```bash
# In payd, API keys are generated via a custom function or directly in the DB
# For testing purposes, we'll generate a key manually and hash it

export TEST_API_KEY="payd_test_audit$(openssl rand -hex 20)"
export TEST_TENANT_ID="f47ac10b-58cc-4372-a567-0e02b2c3d479"

echo "Generated test API key: $TEST_API_KEY"

# The key hash would normally be created with Argon2id
# For MVP testing, we'll use the database to create the key record
# Note: In the actual app, /v1/keys endpoint handles this

psql $DATABASE_URL -c "
SELECT id, key_prefix, created_at FROM tenant_api_keys WHERE tenant_id = '$TEST_TENANT_ID'::uuid;
"
```

**Expected Result:** Show any existing keys for the tenant (may be empty)

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 2.3 Test GET /v1/me (Using Test API Key)

**Objective:** Verify that a valid API key can authenticate to the `/v1/me` endpoint and return the tenant ID.

**Step:**
```bash
# First, we need to create an actual key via the database or API
# This test assumes a test key is available

export API_KEY="payd_test_..."  # Fill in with actual test key from database

curl -s \
  -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/v1/me | jq .
```

**Expected Result:**
```json
{
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 2.4 Invalid API Key Rejection

**Objective:** Verify that requests with invalid API keys are rejected with a 401 error.

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" \
  -H "Authorization: Bearer payd_test_invalid" \
  http://localhost:8080/v1/me | jq .
```

**Expected Result:**
- HTTP Status: `401`
- Error response with code `unauthorized`

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

### 2.5 Missing Authorization Header

**Objective:** Verify that requests without authorization header are rejected.

**Step:**
```bash
curl -s -w "\nHTTP Status: %{http_code}\n" \
  http://localhost:8080/v1/me | jq .
```

**Expected Result:**
- HTTP Status: `401`
- Error response

**Actual Result:**
```
(to be filled during testing)
```

**Status:** 🔴 Pending

---

## Summary

### Tenant Management Status

| Test | Status | Notes |
|------|--------|-------|
| Create test tenant | 🔴 | |
| List tenants | 🔴 | |
| Generate API key | 🔴 | |
| Authenticate with valid key | 🔴 | |
| Reject invalid key | 🔴 | |
| Reject missing auth | 🔴 | |

### Test Tenant Details (to be filled)

```
Tenant ID: f47ac10b-58cc-4372-a567-0e02b2c3d479
API Key (test): payd_test_...
Created At: (to be filled)
```

### Issues Found

(to be filled during testing)

### Notes

- The MVP does not have a public `/v1/tenants` endpoint; tenants must be created via direct DB access for now
- API key format: `payd_test_<random>` or `payd_live_<random>`
- API keys are stored as Argon2id hashes in the database
- Each request using an API key extracts the tenant ID from the key's hash lookup

---

## Ready for Next Tests

Once tenant authentication is verified, proceed to [03-plans/plans.md](../03-plans/plans.md) to test plan creation and management.
