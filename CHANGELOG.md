# Changelog

All notable changes to billax are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- CI/CD pipeline with GitHub Actions (ci.yml, release.yml)
- Semantic release automation with conventional commits
- Comprehensive OpenAPI 3.1 specification with all endpoints
- Tenant onboarding guide with API key best practices
- Version injection via ldflags in Docker build

### Changed
- Docker binary renamed from `billax` to `payd` for consistency

---

## [0.1.0] — 2026-04-11

### Added

**Core Infrastructure**
- Go module with chi router framework
- PostgreSQL 15 database schema with RLS (Row Level Security)
- Multi-tenant architecture with shared schema
- Graceful shutdown and signal handling
- Structured logging with log/slog
- Health check endpoint

**Authentication & Security**
- API key generation with Argon2id hashing
- Bearer token authentication middleware
- Per-tenant rate limiting (token bucket, 100 req/min default)
- Request ID tracking for debugging
- Recovery middleware for panic handling

**API Layer**
- Error envelope with machine-readable codes
- Validation error responses with field-level details
- Cursor-based pagination
- Idempotency header support (Idempotency-Key)

**Billing Domain**
- Plans with idempotent upsert by slug
- Subscriptions with plan-based and planless support
- Subscription tags (native TEXT[] with GIN index)
- Payment recording with idempotency
- Subscription enrichment with payment history

**Payment Providers**
- Provider adapter pattern (generic abstraction)
- Provider registry and plugin system
- Mercado Pago connector with full integration
- Webhook signature validation
- Payment method metadata extraction

**Observability**
- Prometheus metrics (request rate, latency, in-flight requests)
- Grafana dashboard with 9 panels
- Subscription metrics poller (background job)
- Payment charge attempts tracking by provider

**Tenant Management**
- Self-service tenant signup (POST /v1/signup)
- API key management (create, list, revoke)
- Tenant info retrieval

**Pay-Per-Use & Features**
- Planless subscriptions with custom billing
- Subscription amount updates (pay-per-use)
- Provider capability system with YAML gates
- Flags for plans and pay-per-use support

**Database**
- 6 migrations (schema, plan slug, planless subscriptions, provider credentials, idempotency, tenant email)
- Automated migration runner on startup
- UUIDv7 for subscription keys (ordered, time-based)

**Documentation**
- README with quick start and architecture overview
- GETTING_STARTED.md with end-to-end billing flow examples
- OPERATIONS.md with deployment, monitoring, and scaling
- OPERATIONS.md troubleshooting section
- Railway one-click deployment support

**Docker**
- Multi-stage production build (Alpine Linux)
- Development stack with Docker Compose
- Health check integration
- Prometheus and Grafana profiling

**Testing**
- Comprehensive test suite for all domains
- Audit test scripts with curl examples
- Integration test support with testcontainers

### Fixed

- Subscription idempotency (migration 005)
- Mercado Pago payer object structure (nested payer field)
- Payment handler route registration (flat path registration)
- Auth middleware Base64 encoding
- Credentials retrieval in payment handler

### Known Issues

- Provider credential encryption not yet implemented (stored plaintext in DB)
- Webhook delivery retries not yet implemented
- Email notifications not yet implemented

---

## [0.0.1] — 2026-02-28

### Added

- Initial project setup
- Go module scaffolding
- PostgreSQL Docker image

---

[Unreleased]: https://github.com/DelgadoElias/billax/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/DelgadoElias/billax/releases/tag/v0.1.0
[0.0.1]: https://github.com/DelgadoElias/billax/releases/tag/v0.0.1
