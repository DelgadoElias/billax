# billax

> **Swap payment providers. Never migrate your subscriptions.**

billax is an open source, multi-tenant billing and subscription engine built for Latin America — and anywhere else payments are messy. It sits between your application and your payment providers, giving you a clean API to manage plans, subscriptions, and payments regardless of which provider processes the charge.

---

## Why billax?

You built your billing on Mercado Pago. Then your enterprise clients need Stripe. Then you expand to another country where a local processor has better rates. Traditional billing stacks force you to migrate data, rewrite integrations, and coordinate cutover windows.

billax decouples billing logic from payment providers entirely. When you need a new provider, you register the connector. When you want to change a provider for a specific tenant, region, or card type — you update config. Your subscription data stays exactly where it is.

```
Your app  →  billax API  →  [ Mercado Pago | Helipagos | Stripe | ... ]
                               ↓
                         one subscription model
                         one idempotency contract
                         one set of webhooks
```

---

## Features

### Multi-provider routing
Connect multiple payment providers simultaneously. Route by region, amount, card type, currency, or your own logic. No migrations required when switching providers.

### Subscription flexibility
- **Plan-based subscriptions** — create reusable plans with pricing and intervals; reference them from any subscription
- **Planless subscriptions** — define amount, currency, and interval per subscription for custom or bespoke clients
- **Pay-per-use billing** — update a subscription's amount before each billing cycle for metered or variable workloads

### Multi-tenant from day one
Every table, every query, every response is scoped to a tenant. Row Level Security enforced at the database level. API keys per tenant. Rate limits per tenant.

### Idempotent by design
Every mutating endpoint accepts an `Idempotency-Key` header. Duplicate requests return the same response. Plans upsert by slug. Payments deduplicate by key. Retry freely.

### Tags and metadata
First-class tag filtering on subscriptions (`tags @> $1::text[]`, GIN indexed). Arbitrary `metadata` JSONB blob for anything domain-specific. Filter subscriptions by any combination of tags.

### Self-hosted, no vendor lock-in
Docker Compose in one command. No proprietary cloud services. PostgreSQL is your data store. You own everything.

---

## Quick start

### Local development (with observability)

```bash
git clone https://github.com/DelgadoElias/billax.git
cd billax

# Start the full stack: app, database, Prometheus, Grafana
docker-compose --profile observability up -d

# Apply migrations
docker exec payd_db psql -U payd_app -d payd < migrations/001_init.sql
docker exec payd_db psql -U payd_app -d payd < migrations/002_plan_slug_subscription_tags.sql
docker exec payd_db psql -U payd_app -d payd < migrations/003_planless_subscriptions.sql

# Configure environment (optional in dev)
cp .env.example .env

# App is ready at http://localhost:8080
# Grafana dashboard at http://localhost:3000 (admin/admin)
# Prometheus metrics at http://localhost:9091
```

Health check:
```bash
curl http://localhost:8080/health
# {"status":"ok","version":"0.1.0"}
```

### Running without observability

```bash
docker-compose up -d  # omit --profile observability
go run ./cmd/payd
```

---

## Configuration

billax is configured via environment variables with sensible defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | **required** | PostgreSQL DSN |
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | HTTP port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `RATE_LIMIT_DEFAULT` | `100` | Requests per minute per tenant |
| `PROVIDERS_CONFIG_PATH` | `providers.yml` | Path to provider capability gates |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics endpoint |
| `METRICS_PORT` | `9090` | Port for `/metrics` endpoint |

### `providers.yml` — capability gates

Declare which features each provider supports for this deployment. The file is committed with safe defaults; edit it without redeploying.

```yaml
mercadopago:
  plans: true
  pay_per_use: false

helipagos:
  plans: true
  pay_per_use: true

stripe:
  plans: true
  pay_per_use: true
```

If a provider is missing from the file, defaults apply: `plans: true`, `pay_per_use: false`. If the file is absent entirely, all providers use these safe defaults.

---

## API overview

### Authentication

Every request to `/v1/*` requires an API key issued per tenant:

```
Authorization: Bearer payd_live_<key>
```

### Error responses

All errors are returned as a structured JSON envelope with an optional `fields` array for validation errors:

```json
{
  "error": {
    "code": "invalid_input",
    "message": "currency: must be a supported ISO 4217 currency code (got \"XYZ\")",
    "request_id": "req_01j0hk7gq9rkr1a2b3c4d5e6f7g8h9i0j",
    "fields": [
      {
        "field": "currency",
        "message": "must be a supported ISO 4217 currency code (got \"XYZ\")"
      },
      {
        "field": "trial_days",
        "message": "must not be negative"
      }
    ]
  }
}
```

The `fields` array is only present for validation errors (HTTP 400). Other error types omit it.

### Plans

Plans are idempotent by slug. POST the same slug twice — get back the same plan.

```bash
# Create or get a plan
POST /v1/plans
{
  "slug": "pro-monthly",
  "name": "Pro Monthly",
  "amount": 299900,
  "currency": "ARS",
  "interval": "month",
  "interval_count": 1
}
# 201 Created on first call, 200 OK on subsequent calls — same plan returned

GET /v1/plans/slug/pro-monthly  # for payment buttons
```

Amount is in minor currency units (centavos). Always integers. Never floats.

### Subscriptions

```bash
# Plan-based subscription
POST /v1/subscriptions
Idempotency-Key: <uuid>
{
  "plan_slug": "pro-monthly",
  "external_customer_id": "cust_abc123",
  "provider_name": "mercadopago",
  "tags": ["premium", "latam"]
}

# Planless subscription — custom billing for a specific client
POST /v1/subscriptions
Idempotency-Key: <uuid>
{
  "amount": 1500000,
  "currency": "ARS",
  "interval": "month",
  "interval_count": 1,
  "external_customer_id": "enterprise_client",
  "provider_name": "helipagos",
  "tags": ["enterprise", "custom"]
}

# Filter by tag
GET /v1/subscriptions?tag=premium&tag=latam

# Pay-per-use: update amount before next charge
PATCH /v1/subscriptions/{key}
{ "amount": 1750000 }
```

Subscription GET responses include the last 10 payments embedded. Full history via `GET /v1/subscriptions/{key}/payments`.

### Payments

```bash
# Record a payment (routes through ProviderAdapter)
POST /v1/subscriptions/{key}/payments
Idempotency-Key: <uuid>
{
  "provider_name": "mercadopago",
  "amount": 299900,
  "currency": "ARS",
  "description": "Pro Monthly — March 2026",
  "external_customer_id": "cust_abc123"
}
# 201 Created on first call, 200 OK on replay — same payment returned

# All payments for tenant (cross-subscription)
GET /v1/payments?provider=mercadopago
```

---

## Provider connectors

Each payment connector lives in `internal/provider/<name>/` and implements one interface:

```go
type PaymentProvider interface {
    GetProviderName() string
    Capabilities() ProviderCapabilities
    CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
    RefundCharge(ctx context.Context, chargeID string, amount int64) (*RefundResult, error)
    HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
    ValidateConfig(config map[string]string) error
}
```

Connectors are:
- **Stateless** — credentials are passed per-call as `map[string]string`; connectors hold no state
- **Isolated** — provider types never leak into services; only normalized domain types cross the boundary
- **Self-declaring** — each connector declares what it supports via `Capabilities()`

Register at startup:

```go
registry := provider.NewRegistry()
registry.Register(mercadopago.New())
registry.Register(helipagos.New())
// registry.Register(stripe.New())  // coming soon
```

billax ships with connectors for:
- [x] Mercado Pago (production-ready)
- [ ] Helipagos (planned)
- [ ] Stripe (planned)

Writing your own connector: implement the six-method interface, add an entry to `providers.yml`, register at startup. That's it.

---

## Architecture

```
cmd/payd/main.go           ← wiring: config, DB pool, registry, services, router
internal/
  config/                  ← env-based config (godotenv for .env in dev)
  db/                      ← pgx pool + connection helpers
  middleware/               ← auth (Argon2id), rate limiting, request ID, logging, recovery
  errors/                  ← sentinel errors + HTTP status + machine-readable codes
  httputil/                ← RespondJSON, RespondError, RespondCreated, RespondOK
  plan/                    ← plans domain (model, repo, service, handler)
  subscription/            ← subscription domain (model, repo, service, handler)
  payment/                 ← payment domain (model, repo, service, handler)
  provider/
    provider.go            ← PaymentProvider interface + domain types
    capabilities.go        ← ProviderCapabilities + YAML loader
    registry.go            ← runtime registry (thread-safe, panic on duplicate)
    adapter.go             ← ProviderAdapter: single point between services and providers
    mercadopago/           ← connector (Week 3)
migrations/                ← sequential numbered SQL files (001, 002, 003...)
providers.yml              ← deployment-level capability gates
```

**Layering rule**: handlers parse and respond. Services own business logic. Repositories own SQL. Adapters normalize provider responses. No layer reaches past the next one.

---

## Data model highlights

- **Monetary amounts**: always `int64` in minor currency units (centavos for ARS, cents for USD). No floats, ever.
- **Subscription key**: UUIDv7 — time-ordered, stable, safe to expose in URLs.
- **Tags**: `TEXT[]` with GIN index. Filter with AND semantics via `@>` containment operator.
- **Idempotency**: `UNIQUE(tenant_id, idempotency_key)` enforced at DB level. ON CONFLICT returns existing row.
- **Payment method metadata**: non-sensitive info (brand, last four, wallet type) stored as JSONB. Raw provider response stored separately and never exposed via API.
- **Row Level Security**: every tenant-scoped table has an RLS policy. The app sets `SET LOCAL app.current_tenant_id` on every request. Even a compromised query cannot cross tenant boundaries.

---

## Security

- **Tenant isolation**: RLS on every table, enforced at DB level — not application level
- **API keys**: Argon2id hashed, only prefix stored for lookup
- **Idempotency**: all mutating endpoints require `Idempotency-Key`
- **No raw card data**: billax never accepts, logs, or stores card numbers, CVVs, or full PANs
- **Webhook signatures**: outbound webhooks signed with HMAC-SHA256
- **Inbound webhooks**: provider signature validated before any processing
- **Rate limiting**: per-tenant token bucket, default 100 req/min, configurable

---

## Observability

billax exposes Prometheus metrics and includes Grafana dashboards for operational visibility.

### Starting the observability stack

```bash
docker-compose --profile observability up -d
```

This brings up:
- **Prometheus** (http://localhost:9091) — scrapes metrics from `/metrics` every 15s
- **Grafana** (http://localhost:3000) — dashboards auto-provisioned from `deploy/grafana/provisioning/`
  - Username: `admin` / Password: `admin`
  - Dashboard: "payd — Operational Overview" shows request rates, latency percentiles, error rates, and payment success rates by provider

### Metrics exposed

- `payd_http_requests_total` — total HTTP requests by status code and path
- `payd_http_request_duration_seconds` — request latency histogram (p50, p99, etc.)
- `payd_http_in_flight_requests` — concurrent in-flight requests
- `payd_payment_charge_attempts_total` — payment charge attempts by provider and outcome (success/failure)
- `payd_active_subscriptions` — current subscription count by status (updated every 30 seconds)

For details on interpreting the dashboard, see [docs/OPERATIONS.md](docs/OPERATIONS.md).

---

## Contributing

Contributions welcome. To add a payment provider:

1. Create `internal/provider/<name>/` with a struct implementing `PaymentProvider`
2. Implement all six interface methods (return `ErrNotSupported` for anything the provider can't do)
3. Add an entry to `providers.yml`
4. Register in `cmd/payd/main.go`
5. Write tests — `go test ./...` must pass

For all other contributions, open an issue first to discuss scope.

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

You are free to use, modify, and distribute billax, including in commercial products. If you fork or redistribute this project, you must retain the original copyright notice and attribution to the original authors. See the license file for full terms.
