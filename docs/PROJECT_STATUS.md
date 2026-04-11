# Project Status — billax (as of 2026-04-11)

This document summarizes the current state of the billax billing engine and remaining work.

---

## ✅ Completed (Production-Ready)

### Core Billing Engine
- [x] Multi-tenant foundation with Row Level Security
- [x] Plan management with idempotent slug-based creation
- [x] Subscription lifecycle (create, list, update, cancel)
- [x] Payment recording with idempotent Idempotency-Key header
- [x] Support for plan-based, planless, and pay-per-use billing
- [x] Tag-based filtering with PostgreSQL GIN indexing
- [x] Metadata support (JSONB) for custom fields
- [x] Migration system with sequential numbered SQL files (001-005)

### Mercado Pago Connector (Week 3 + April 2026 Fixes)
- [x] Full PaymentProvider interface implementation
- [x] Payment creation with rate-limit retry logic (exponential backoff)
- [x] Payment refund support
- [x] Webhook signature validation (HMAC-SHA256)
- [x] Complete type mapping (MP → domain models)
- [x] Unit tests with httptest mocking (46 tests, all passing)
- [x] **Critical Fix (April 11):** Fixed payer object structure to match MP API spec
  - Before: flat `payer_email` field (rejected by MP API)
  - After: nested `payer: {email}` object (matches MP API)
- [x] **Critical Fix (April 11):** Fixed subscription idempotency implementation
  - Migration 005 added idempotency_key column with UNIQUE constraint
  - Service layer returns (subscription, created bool, error)
  - Repository enforces deduplication on (tenant_id, idempotency_key)
  - All 15 API endpoints verified working (100% pass rate)

### Authentication & Authorization
- [x] API key management with Argon2id hashing
- [x] Per-tenant rate limiting (100 req/min default, configurable)
- [x] Request ID tracking and structured logging
- [x] Multi-tenant isolation with RLS enforced at DB level

### Observability
- [x] Prometheus metrics for HTTP requests, latency, in-flight requests
- [x] Payment charge attempt instrumentation (success/failure by provider)
- [x] Active subscriptions gauge with 30-second polling
- [x] Grafana dashboard with 9 panels (pre-provisioned in docker-compose)
- [x] Structured logging with slog (stdlib)

### Infrastructure
- [x] Docker Compose development stack (app, PostgreSQL, Prometheus, Grafana)
- [x] Dockerfile for production multi-stage builds
- [x] Graceful shutdown with timeout
- [x] Health check endpoint
- [x] Railway one-click deployment support

### Documentation
- [x] Updated README with feature overview
- [x] Mercado Pago setup instructions
- [x] Idempotency examples
- [x] Provider connector pattern documentation
- [x] docs/OPERATIONS.md — production deployment and troubleshooting
- [x] docs/RAILWAY_DEPLOYMENT.md — Railway one-click setup
- [x] docs/GETTING_STARTED.md — complete end-to-end walkthrough (NEW, April 11)
- [x] docs/TESTING.md — comprehensive API testing guide with curl examples (NEW, April 11)
- [x] OpenAPI 3.1 specification with error schemas

### Testing
- [x] Unit tests for all domains (plan, subscription, payment, provider)
- [x] Provider connector tests with mocked HTTP (46 MP tests)
- [x] Manual API testing (documented in docs/TESTING.md)
- [x] End-to-end workflow verification (all 15 endpoints tested and working)

---

## 📊 Test Results — Week 4.5 (April 11, 2026)

**Final Status:** 🟢 **15/15 endpoints working (100% pass rate)**

### Critical Issues Fixed
1. ✅ **List endpoints returning empty** — RLS/connection state resolved
2. ✅ **Subscription GET 404 error** — Route conflict between handlers fixed
3. ✅ **Payment credentials not fetched** — CredentialsService added to payment handler
4. ✅ **Subscription idempotency 500 errors** — Migration 005 applied, metadata NULL handling fixed

### Endpoint Summary
| Category | Status | Notes |
|----------|--------|-------|
| Health | ✅ | /health responds 200 |
| Authentication | ✅ | /v1/me works, auth enforced |
| Plans | ✅ | Create (idempotent), List, Get by ID/slug working |
| Subscriptions | ✅ | Create (idempotent), List (with filters), Get, Update, Cancel working |
| Payments | ✅ | Create (idempotent), List by subscription/tenant, Mercado Pago routing working |
| Credentials | ✅ | Store/retrieve provider credentials working |

---

## 🚀 In Progress / Next Priority

### Week 5 — Documentation (Primary Focus)
- [x] Getting Started guide (NEW — April 11)
- [x] Testing guide with API examples (NEW — April 11)
- [ ] Architecture deep-dive documentation
- [ ] API integration examples (Python, Node.js, Go SDKs)
- [ ] Webhook integration guide (how to receive settlement notifications)

### Security & Compliance
- [ ] Security audit (OWASP Top 10, SQL injection prevention, etc.)
- [ ] Penetration testing (rate limit bypass, auth bypass, data isolation)
- [ ] PCI compliance verification (no card data storage, provider integration only)

### Performance & Load Testing
- [ ] Load testing (target: >100 RPS capacity)
- [ ] Database query optimization
- [ ] Connection pooling tuning
- [ ] Stress testing under sustained load

### Provider Expansion
- [ ] Helipagos connector (Latin America, high adoption)
- [ ] Stripe connector (global fallback)
- [ ] Custom connector template/example

### Production Readiness
- [ ] Staging environment setup
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Production checklist
- [ ] Incident response runbooks
- [ ] Monitoring/alerting rules

---

## 📋 Known Limitations

### Current
- Mercado Pago only provider (Helipagos and Stripe planned)
- No automatic subscription renewal job (manual payment creation required)
- No built-in invoice generation
- Webhook delivery is stateless (no retry queue for failed deliveries)

### Intentional Design Choices
- No subscription-level customer name/email storage (use external_customer_id)
- No full audit trail API (git commit history is source of truth)
- No dashboard UI (API-first; clients build their own)
- Monetary amounts always in minor currency units (no float precision issues)

---

## 🎯 Launch Checklist (Days 36-40)

Before production launch:

- [ ] Security audit completed and issues resolved
- [ ] Load testing passed (>100 RPS verified)
- [ ] Staging environment fully functional
- [ ] Production database backup/recovery tested
- [ ] Monitoring and alerting configured
- [ ] Incident response runbooks written
- [ ] Legal/compliance review (PCI, data residency, etc.)
- [ ] Customer support playbooks
- [ ] Release notes and changelog prepared
- [ ] Version 1.0.0 tagged on main branch

---

## 📈 Metrics & Observability

### Current Metrics
- Request rate per endpoint
- Request latency (p50, p99 percentiles)
- Error rate by endpoint
- Payment charge attempts by provider
- Active subscriptions by status
- HTTP in-flight requests

### Recommended Additional Instrumentation
- Database query latency histogram
- RLS policy evaluation time
- Provider API response times
- Webhook delivery success rate
- Subscription renewal SLO tracking

---

## 🔗 References

- Main documentation: [README.md](../README.md)
- Getting started: [docs/GETTING_STARTED.md](GETTING_STARTED.md)
- Testing guide: [docs/TESTING.md](TESTING.md)
- Operations: [docs/OPERATIONS.md](OPERATIONS.md)
- Architecture: See CLAUDE.md in project root (developer guide)
- OpenAPI spec: [api/openapi.yaml](../api/openapi.yaml)

---

## 💬 Quick Links

- **Source code:** `/internal/` organized by domain (plan, subscription, payment, provider)
- **Database schema:** `/migrations/` numbered SQL files
- **Configuration:** Environment variables documented in README
- **Provider pattern:** `internal/provider/provider.go` (interface definition)

---

## Latest Changes (April 11, 2026)

### Documentation (this session)
- Created `docs/GETTING_STARTED.md` with complete end-to-end example
- Created `docs/TESTING.md` with unit/integration/manual testing examples
- Updated README with Getting Started reference and Mercado Pago setup section
- Documented idempotency with concrete curl examples

### Code (previous session, April 11)
- Fixed Mercado Pago payer object structure (nested payer: {email})
- Implemented subscription idempotency with UNIQUE constraint
- Fixed route conflict preventing subscription GET endpoint
- Added credentials service to payment handler
- All 15 API endpoints verified working

---

## What Users Can Do Now

1. **Local Development:** Clone, `docker-compose up`, and start building with billax API
2. **Integration Testing:** Follow docs/TESTING.md examples against local or staging instance
3. **Mercado Pago:** Connect real Mercado Pago account and process payments end-to-end
4. **Monitoring:** Use Grafana dashboards to observe system health and metrics
5. **Deployment:** Deploy to Railway with one click via provided button

---

## Questions or Issues?

- Check [docs/TESTING.md](TESTING.md) for API troubleshooting
- Check [docs/OPERATIONS.md](OPERATIONS.md) for operational questions
- Refer to CLAUDE.md in project root for architecture and design decisions
- Review [api/openapi.yaml](../api/openapi.yaml) for spec-level details
