# Testing Guide — billax

This guide covers unit testing, integration testing, and manual API testing for billax.

---

## Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests for a specific package:

```bash
go test ./internal/subscription/...
go test ./internal/provider/mercadopago/...
```

### Provider connector tests

Each payment connector includes unit tests with mocked HTTP responses. No real API calls are made.

```bash
# Test Mercado Pago connector (46 tests covering payment creation, refunds, webhooks)
go test ./internal/provider/mercadopago/...

# All tests pass without network:
# ok    github.com/DelgadoElias/billax/internal/provider/mercadopago  1.234s
```

---

## Integration Tests

Integration tests require a live PostgreSQL database. They run with the `integration` build tag.

```bash
# Start database if not running
docker-compose up -d payd_db

# Run integration tests
go test -tags=integration ./...
```

Integration tests use `testcontainers-go` to spin up ephemeral Postgres containers if you prefer not to manage your own.

---

## Manual API Testing

### Prerequisites

1. Start the server:
   ```bash
   docker-compose --profile observability up -d
   # Apply migrations (see README)
   ```

2. Get test API key and tenant ID:
   ```bash
   # From docker logs or .env
   API_KEY=payd_test_pF+3gggDxi4kpvzqKofHD2C9IJuGdy
   TENANT=f47ac10b-58cc-4372-a567-0e02b2c3d479
   BASE_URL=http://localhost:8080
   ```

### Health check

```bash
curl -s $BASE_URL/health
# {"status":"ok","version":"0.1.0"}
```

### Authentication

All `/v1/*` endpoints require the `Authorization: Bearer` header:

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE_URL/v1/me
# {"tenant_id": "f47ac10b-..."}
```

### Plans — Create and list

Create a plan:

```bash
curl -s -X POST $BASE_URL/v1/plans \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "starter",
    "name": "Starter Plan",
    "amount": 99900,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1,
    "trial_days": 14
  }'

# Response (HTTP 201):
# {
#   "id": "plan_01j...",
#   "slug": "starter",
#   "name": "Starter Plan",
#   "amount": 99900,
#   "currency": "ARS",
#   "interval": "month",
#   "interval_count": 1,
#   "trial_days": 14,
#   "is_active": true,
#   "created_at": "2026-04-11T12:00:00Z",
#   "updated_at": "2026-04-11T12:00:00Z"
# }
```

List plans:

```bash
curl -s -H "Authorization: Bearer $API_KEY" "$BASE_URL/v1/plans?limit=10"

# Response (HTTP 200):
# {
#   "plans": [
#     {
#       "id": "plan_01j...",
#       "slug": "starter",
#       ...
#     }
#   ],
#   "next_cursor": null
# }
```

### Subscriptions — Plan-based

Create a subscription (plan-based):

```bash
IDEMPOTENCY_KEY=$(uuidgen)

curl -s -X POST $BASE_URL/v1/subscriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_slug": "starter",
    "external_customer_id": "cust_acme_001",
    "provider_name": "mercadopago",
    "tags": ["trial", "acme"],
    "metadata": {"company": "ACME Corp"}
  }'

# Response (HTTP 201):
# {
#   "id": "sub_01j...",
#   "subscription_key": "550e8400-e29b-...",
#   "status": "trialing",
#   "plan_id": "plan_01j...",
#   "amount": 99900,
#   "currency": "ARS",
#   "interval": "month",
#   "interval_count": 1,
#   "current_period_start": "2026-04-11T12:00:00Z",
#   "current_period_end": "2026-05-11T12:00:00Z",
#   "trial_ends_at": "2026-04-25T12:00:00Z",
#   "external_customer_id": "cust_acme_001",
#   "tags": ["trial", "acme"],
#   "payments": [],
#   "created_at": "2026-04-11T12:00:00Z"
# }

# Save subscription key for later
SUBSCRIPTION_KEY="550e8400-e29b-..."
```

Retry the same request (same Idempotency-Key):

```bash
curl -s -X POST $BASE_URL/v1/subscriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{...}'

# Response (HTTP 200 — same subscription returned):
# {
#   "id": "sub_01j...",
#   "subscription_key": "550e8400-e29b-...",
#   ...
# }
```

List subscriptions:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?tag=trial&limit=10"

# Response (HTTP 200):
# {
#   "subscriptions": [
#     {
#       "id": "sub_01j...",
#       "subscription_key": "550e8400-...",
#       "status": "trialing",
#       ...
#       "payments": []
#     }
#   ],
#   "next_cursor": null
# }
```

Get a subscription by key:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY"

# Response (HTTP 200 — includes last 10 payments):
# {
#   "id": "sub_01j...",
#   "subscription_key": "550e8400-...",
#   "status": "trialing",
#   ...
#   "payments": []
# }
```

### Subscriptions — Planless

Create a custom subscription without a plan:

```bash
curl -s -X POST $BASE_URL/v1/subscriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: custom-sub-001" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50000,
    "currency": "ARS",
    "interval": "week",
    "interval_count": 2,
    "external_customer_id": "enterprise_client",
    "provider_name": "mercadopago",
    "tags": ["enterprise", "bi-weekly"]
  }'

# Response (HTTP 201):
# {
#   "id": "sub_01j...",
#   "subscription_key": "...",
#   "status": "active",
#   "plan_id": null,
#   "amount": 50000,
#   "currency": "ARS",
#   "interval": "week",
#   "interval_count": 2,
#   ...
# }
```

### Pay-per-use — Update subscription amount

Update a subscription's amount (for variable billing):

```bash
curl -s -X PATCH $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"amount": 150000}'

# Response (HTTP 200):
# {
#   "id": "sub_01j...",
#   "subscription_key": "...",
#   "amount": 150000,
#   ...
# }
```

### Payments — Create via Mercado Pago

Store Mercado Pago credentials (one-time setup):

```bash
curl -s -X POST $BASE_URL/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "TEST-1234567890-...",
    "webhook_secret": "..."
  }'

# Response (HTTP 200):
# {"provider": "mercadopago", "stored": true}
```

Create a payment (credentials fetched from database):

```bash
PAYMENT_KEY=$(uuidgen)

curl -s -X POST $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $PAYMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "provider_name": "mercadopago",
    "amount": 99900,
    "currency": "ARS",
    "description": "Starter Plan — April 2026"
  }'

# Response (HTTP 201 on first call):
# {
#   "id": "pay_01j...",
#   "subscription_id": "sub_01j...",
#   "idempotency_key": "...",
#   "provider_name": "mercadopago",
#   "provider_charge_id": "1234567890",
#   "amount": 99900,
#   "currency": "ARS",
#   "status": "pending",
#   "payment_method": {
#     "brand": "visa",
#     "type": "credit_card",
#     "last_four": "1234"
#   },
#   "created_at": "2026-04-11T12:00:00Z"
# }
```

Retry the same payment (same Idempotency-Key):

```bash
curl -s -X POST $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $PAYMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{...}'

# Response (HTTP 200 — same payment returned, no duplicate charge):
# {
#   "id": "pay_01j...",
#   ...
# }
```

List payments for subscription:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments?limit=10"

# Response (HTTP 200):
# {
#   "payments": [
#     {
#       "id": "pay_01j...",
#       "status": "succeeded",
#       ...
#     }
#   ],
#   "next_cursor": null
# }
```

### Error handling

Test validation error:

```bash
curl -s -X POST $BASE_URL/v1/plans \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "bad",
    "name": "Bad Plan",
    "amount": -100,
    "currency": "INVALID",
    "interval": "wrong"
  }'

# Response (HTTP 400):
# {
#   "error": {
#     "code": "invalid_input",
#     "message": "validation failed",
#     "request_id": "req_01j...",
#     "fields": [
#       {
#         "field": "amount",
#         "message": "must be greater than 0"
#       },
#       {
#         "field": "currency",
#         "message": "must be a supported ISO 4217 currency code (got \"INVALID\")"
#       },
#       {
#         "field": "interval",
#         "message": "must be one of: day, week, month, year"
#       }
#     ]
#   }
# }
```

Test missing auth:

```bash
curl -s $BASE_URL/v1/plans

# Response (HTTP 401):
# {
#   "error": {
#     "code": "invalid_request",
#     "message": "missing Authorization header",
#     "request_id": "req_01j..."
#   }
# }
```

---

## Test Checklist

Use this checklist to verify core functionality works end-to-end:

- [ ] Health check returns ok
- [ ] Plans can be created with POST
- [ ] Plan creation is idempotent (same slug = same plan, 201 then 200)
- [ ] Plans can be listed and filtered
- [ ] Plan-based subscriptions can be created
- [ ] Subscription creation is idempotent (same Idempotency-Key = same subscription)
- [ ] Planless subscriptions can be created with custom billing fields
- [ ] Subscriptions can be listed and filtered by tag
- [ ] Subscription GET includes last 10 payments embedded
- [ ] Subscription amount can be updated (pay-per-use)
- [ ] Mercado Pago credentials can be stored via API
- [ ] Payments can be created and routed through Mercado Pago
- [ ] Payment creation is idempotent (same key = same payment, no duplicate charge)
- [ ] Payment list returns all payments for a subscription
- [ ] Validation errors return HTTP 400 with `fields` array
- [ ] Invalid auth returns HTTP 401
- [ ] Rate limiting is enforced (100 req/min by default)
- [ ] All endpoints respect tenant isolation (RLS)

---

## Observability during testing

### Grafana dashboard

While testing, monitor the Grafana dashboard:
- **Request Rate**: should show spikes for test calls
- **Error Rate**: should be 0% for successful tests
- **Latency**: should be <100ms for most calls
- **Payment Attempts**: should show successful charges
- **Active Subscriptions**: should increase as you create subscriptions

```
http://localhost:3000 (admin/admin)
```

### Logs

Watch structured logs in real-time:

```bash
docker logs -f payd_app
```

Look for:
- Request IDs matching your curl output
- Successful payment routing to Mercado Pago
- Any validation or database errors

### Metrics

Query Prometheus directly:

```bash
# Active subscriptions
curl -s 'http://localhost:9091/api/v1/query?query=payd_active_subscriptions' | jq

# Payment success rate by provider
curl -s 'http://localhost:9091/api/v1/query?query=rate(payd_payment_charge_attempts_total[5m])' | jq
```

---

## Troubleshooting

### "address already in use"
```bash
# Kill existing process on port 8080
lsof -ti :8080 | xargs kill -9
docker-compose up -d  # restart containers
```

### "database connection refused"
```bash
# Verify Postgres is running
docker-compose logs payd_db

# Restart it if needed
docker-compose restart payd_db
```

### "subscription creation returns 500"
Check migration 005 is applied:
```bash
docker exec payd_db psql -U payd_app -d payd -c \
  "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='subscriptions' AND column_name='idempotency_key')"

# Should return: t (true)
# If false, apply the migration:
docker exec payd_db psql -U payd_app -d payd < migrations/005_subscription_idempotency.sql
```

### "Mercado Pago returns payer_email error"
Ensure you're using the latest code with the nested `payer: {email}` format. This was fixed in April 2026. Pull the latest main branch.

---

## Next Steps

1. **Integration tests** — add comprehensive tests to `internal/*/integration_test.go`
2. **Load testing** — use `ab` or `wrk` to verify >100 RPS capacity
3. **Webhook testing** — simulate Mercado Pago webhook delivery for settlement notifications
4. **Multi-tenant testing** — verify strict isolation between tenants
5. **Provider failover** — test switching between Mercado Pago and other connectors
