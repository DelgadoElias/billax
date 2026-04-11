# audit-tests — Manual Testing Documentation for payd

**Project:** payd — Multi-tenant billing and subscription management engine  
**Date Started:** 2026-04-11  
**Status:** 🟡 In Progress

---

## Testing Overview

This folder contains complete manual testing documentation for the payd application, covering:
- Installation and setup (prerequisites, environment, migrations)
- Core API endpoints (plans, subscriptions, payments)
- Provider credentials configuration
- Mercado Pago connector integration
- Webhook handling

Each section includes **Objective → Preconditions → Steps → Expected Result → Actual Result → Notes**.

---

## Test Status Summary

| Section | Test | Status | Notes |
|---------|------|--------|-------|
| 00 | Setup & Installation | ✅ PASS (7/7) | Go, Docker, .env, PostgreSQL, migrations, app startup all working |
| 01 | Health Endpoint | ✅ PASS (4/4) | `/health` endpoint verification complete, no auth required |
| 02 | Tenants & API Keys | ✅ READY | Created test tenant, valid API key generated and verified |
| 03 | Plans CRUD | 🟡 READY | Auth now working, ready to test |
| 04 | Subscriptions | 🟡 READY | Auth now working, ready to test |
| 05 | Payments | 🟡 READY | Auth now working, ready to test |
| 06 | Provider Credentials | 🟡 READY | Auth now working, ready to test |
| 07 | Mercado Pago Integration | 🟡 READY | Auth now working, ready to test |
| 08 | Webhooks | 🟡 READY | Auth now working, ready to test |
| — | API Credentials | ✅ READY | Valid test API key: `payd_test_pF+3gggDxi4kpvzqKofHD2C9IJuGdy` |

**Blocker Status:** ✅ **RESOLVED** — API Key Hash Encoding Bug Fixed
- **Fix Applied:** Changed `hashToString()` to use Base64 encoding
- **Verification:** Authentication now works ✅
- **Test API Key Ready:** See `TEST_API_CREDENTIALS.md`
- All authenticated endpoints now accessible

---

## Quick Links to Test Sections

1. [**00-setup/**](00-setup/) — Installation and environment setup
2. [**01-health/**](01-health/) — Health check endpoint
3. [**02-tenants/**](02-tenants/) — Tenant and API key management
4. [**03-plans/**](03-plans/) — Plan CRUD operations
5. [**04-subscriptions/**](04-subscriptions/) — Subscription lifecycle
6. [**05-payments/**](05-payments/) — Payment recording and idempotency
7. [**06-provider-credentials/**](06-provider-credentials/) — Provider credential configuration
8. [**07-mercadopago/**](07-mercadopago/) — Mercado Pago sandbox integration
9. [**08-webhooks/**](08-webhooks/) — Webhook processing
10. [**findings.md**](findings.md) — Issues, limitations, and recommendations

---

## Mercado Pago Test Credentials

These are sandbox (TEST) credentials used for this audit:

```
access_token: TEST-8194488031946085-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-286672332
webhook_secret: ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49
public_key: TEST-b776ac89-be8d-4eba-aabe-1782d56b00d8
```

**Note:** ngrok is running on port 8080 for webhook tunneling.

---

## Test Execution Order

Follow this order to properly test the system:

1. **Setup** — Ensure the app is running with a clean database
2. **Health** — Verify basic connectivity
3. **Tenants** — Create a test tenant and API key
4. **Plans** — Create and manage billing plans
5. **Subscriptions** — Create subscriptions linked to plans
6. **Payments** — Record and verify payment idempotency
7. **Provider Credentials** — Configure Mercado Pago test credentials
8. **Mercado Pago** — Create a real charge in MP sandbox
9. **Webhooks** — Simulate and verify webhook handling

---

## Environment Info

```
Go version: 1.22+
Database: PostgreSQL 15
Docker Compose: Available
ngrok: Running on :4040 (tunneling :8080)
```

---

## Next Steps

Start with [00-setup/setup.md](00-setup/setup.md) to verify the installation and bring up the app.
