# 01-health — Health Endpoint Verification

**Objective:** Verify that the `/health` endpoint responds correctly and the application is operational.

**Date:** 2026-04-11

**Preconditions:**
- Setup completed successfully ([00-setup/setup.md](../00-setup/setup.md))
- Application running on `http://localhost:8080`
- curl available

---

## Test Cases

### 1.1 Basic Health Check (No Authentication)

**Objective:** Verify the `/health` endpoint does not require authentication and returns expected response.

**Step:**
```bash
curl -s http://localhost:8080/health | jq .
```

**Expected Result:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

**Actual Result:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

**Status:** ✅ PASS

**Notes:**
- The `/health` endpoint is intentionally unauthenticated for load balancer/monitoring integration
- It should return HTTP 200 OK

---

### 1.2 Health Check with Headers

**Objective:** Verify that `/health` ignores authentication headers and still responds.

**Step:**
```bash
curl -s -H "Authorization: Bearer invalid_key" http://localhost:8080/health | jq .
```

**Expected Result:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

**Actual Result:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

**Status:** ✅ PASS

**Notes:**
- Even with an invalid API key, `/health` should respond normally

---

### 1.3 HTTP Status Code

**Objective:** Verify the correct HTTP status code is returned.

**Step:**
```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
```

**Expected Result:** `200`

**Actual Result:**
```
200
```

**Status:** ✅ PASS

---

### 1.4 Response Headers

**Objective:** Verify response headers (Content-Type, etc.)

**Step:**
```bash
curl -s -i http://localhost:8080/health | head -15
```

**Expected Result:**
- `Content-Type: application/json`
- `HTTP/1.1 200 OK`

**Actual Result:**
```
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: req_c4692a46-3a7b-40ed-a4df-68977d18c60f
Date: Sat, 11 Apr 2026 14:21:58 GMT
Content-Length: 33

{"status":"ok","version":"0.1.0"}
```

**Status:** ✅ PASS

**Notes:**
- Response includes X-Request-Id header for tracing
- Content-Type is correctly set to application/json

---

## Summary

### Health Endpoint Status

| Test | Status | Notes |
|------|--------|-------|
| Basic health check | ✅ | Returns correct JSON structure |
| Health with headers | ✅ | Ignores invalid auth headers correctly |
| HTTP status code | ✅ | Returns 200 OK |
| Response headers | ✅ | Includes X-Request-Id, Content-Type is application/json |

### Issues Found

✅ **No issues found** — The `/health` endpoint works correctly and doesn't require authentication.

---

## Ready for Next Tests

Once health checks pass, proceed to [02-tenants/tenants.md](../02-tenants/tenants.md) to test tenant and API key management.
