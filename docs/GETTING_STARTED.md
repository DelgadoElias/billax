# Getting Started with billax

This guide walks you through setting up billax and implementing a complete billing flow: create a plan → create a subscription → charge a payment.

---

## Prerequisites

- Docker and Docker Compose
- `curl` or Postman for API testing
- Go 1.21+ (for local development)
- A Mercado Pago account (for production payments)

---

## 1. Start billax locally

Clone and start the full stack:

```bash
git clone https://github.com/DelgadoElias/billax.git
cd billax

# Start app, database, Prometheus, Grafana
docker-compose --profile observability up -d

# Apply migrations
docker exec payd_db psql -U payd_app -d payd < migrations/001_init.sql
docker exec payd_db psql -U payd_app -d payd < migrations/002_plan_slug_subscription_tags.sql
docker exec payd_db psql -U payd_app -d payd < migrations/003_planless_subscriptions.sql
docker exec payd_db psql -U payd_app -d payd < migrations/005_subscription_idempotency.sql

# Verify app is running
curl http://localhost:8080/health
# {"status":"ok","version":"0.1.0"}
```

### Access your services

- **billax API**: http://localhost:8080
- **Grafana dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9091

---

## 2. Get your test API key

billax comes with a pre-seeded test tenant and API key for development.

```bash
# From the Docker logs or environment
export API_KEY="payd_test_pF+3gggDxi4kpvzqKofHD2C9IJuGdy"
export BASE_URL="http://localhost:8080"
```

Test the key:

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE_URL/v1/me
# {"tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"}
```

---

## 3. Create your first plan

A plan is a template for recurring billing. For example, a "Pro Monthly" plan charges $29.99/month.

```bash
curl -s -X POST $BASE_URL/v1/plans \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "pro-monthly",
    "name": "Pro Monthly",
    "description": "Pro plan, renewed monthly",
    "amount": 2999,
    "currency": "USD",
    "interval": "month",
    "interval_count": 1,
    "trial_days": 14
  }'
```

**Key points:**
- `slug` must be unique per tenant and immutable (used in URLs, upsert for idempotency)
- `amount` is in minor currency units (cents for USD, centavos for ARS) — 2999 = $29.99
- `currency` is ISO 4217 code
- `interval` and `interval_count` define the billing cycle (month, week, etc.)
- `trial_days` is optional; 0 = no trial

Response (HTTP 201):
```json
{
  "id": "plan_01j3k4l5m6n7o8p9q0r1",
  "slug": "pro-monthly",
  "name": "Pro Monthly",
  "description": "Pro plan, renewed monthly",
  "amount": 2999,
  "currency": "USD",
  "interval": "month",
  "interval_count": 1,
  "trial_days": 14,
  "is_active": true,
  "created_at": "2026-04-11T12:00:00Z",
  "updated_at": "2026-04-11T12:00:00Z"
}
```

**Check idempotency:** Call the same request again — you'll get HTTP 200 (same plan, no duplicate).

---

## 4. Create a subscription

A subscription ties a customer to a plan. When a customer signs up for the Pro plan, you create a subscription.

```bash
# Generate idempotency key (unique per subscription request)
IDEMPOTENCY_KEY=$(uuidgen)

curl -s -X POST $BASE_URL/v1/subscriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_slug": "pro-monthly",
    "external_customer_id": "cust_acme_001",
    "provider_name": "mercadopago",
    "tags": ["premium", "early-adopter"],
    "metadata": {
      "company": "ACME Corp",
      "industry": "Tech"
    }
  }'
```

**Key points:**
- `plan_slug` references the plan created above
- `external_customer_id` is YOUR system's customer ID (email, UUID, etc.)
- `provider_name` is which payment processor will charge this subscription (mercadopago, stripe, etc.)
- `Idempotency-Key` header ensures the request is idempotent — same key = same subscription, safe to retry forever
- `tags` are for filtering subscriptions later
- `metadata` is arbitrary JSON for your own use

Response (HTTP 201):
```json
{
  "id": "sub_01j3k4l5m6n7o8p9q0r1",
  "subscription_key": "550e8400-e29b-41d3-a567-426614174000",
  "status": "trialing",
  "plan_id": "plan_01j3k4l5m6n7o8p9q0r1",
  "amount": 2999,
  "currency": "USD",
  "interval": "month",
  "interval_count": 1,
  "current_period_start": "2026-04-11T12:00:00Z",
  "current_period_end": "2026-05-11T12:00:00Z",
  "trial_ends_at": "2026-04-25T12:00:00Z",
  "external_customer_id": "cust_acme_001",
  "provider_name": "mercadopago",
  "tags": ["premium", "early-adopter"],
  "metadata": {"company": "ACME Corp", "industry": "Tech"},
  "payments": [],
  "created_at": "2026-04-11T12:00:00Z"
}
```

Save the subscription key for the next step:
```bash
export SUBSCRIPTION_KEY="550e8400-e29b-41d3-a567-426614174000"
```

---

## 5. Store Mercado Pago credentials

To charge payments, billax needs your Mercado Pago API credentials. Store them once; they're reused for all charges.

```bash
# From your Mercado Pago dashboard
curl -s -X POST $BASE_URL/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR-1234567890123456-041110-a84c8b13a30d5fc6a3e2332a9f34b8e8-123456789",
    "webhook_secret": "ccb58bd94631f19f75ed7f23ebe9cc0cf47575919f22959c36e7961a71859d49"
  }'
```

Response (HTTP 200):
```json
{
  "provider": "mercadopago",
  "stored": true
}
```

Credentials are stored in the database, scoped to your tenant. They're automatically retrieved when you create payments.

---

## 6. Create a payment

Now charge the subscription via Mercado Pago. This is the full end-to-end flow:

```bash
# Generate idempotency key for the payment
PAYMENT_KEY=$(uuidgen)

curl -s -X POST $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $PAYMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "provider_name": "mercadopago",
    "amount": 2999,
    "currency": "USD",
    "description": "Pro Monthly — April 2026"
  }'
```

**Key points:**
- The URL includes the `subscription_key` (from step 4)
- `amount` must match the subscription amount (usually)
- `provider_name` specifies which payment processor routes the charge
- `Idempotency-Key` header prevents duplicate charges — retry safely with the same key
- Credentials are fetched automatically from the database (no need to pass them in the request)

Response (HTTP 201):
```json
{
  "id": "pay_01j3k4l5m6n7o8p9q0r1",
  "subscription_id": "sub_01j3k4l5m6n7o8p9q0r1",
  "idempotency_key": "...",
  "provider_name": "mercadopago",
  "provider_charge_id": "1234567890",
  "amount": 2999,
  "currency": "USD",
  "status": "pending",
  "failure_reason": null,
  "payment_method": {
    "brand": "visa",
    "type": "credit_card",
    "last_four": "1234"
  },
  "created_at": "2026-04-11T12:00:00Z"
}
```

The payment has been sent to Mercado Pago. Status will be `pending` until the customer completes payment flow, then transitions to `succeeded` or `failed`.

**Retry the same payment:**
```bash
curl -s -X POST $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: $PAYMENT_KEY" \
  -d '...'

# Response (HTTP 200 — same payment, no duplicate charge):
# {...}
```

---

## 7. Check payment status

Retrieve the subscription with all its payments:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY
```

Response includes the last 10 payments:
```json
{
  "id": "sub_01j...",
  "subscription_key": "550e8400-...",
  "status": "active",
  "payments": [
    {
      "id": "pay_01j...",
      "provider_charge_id": "1234567890",
      "status": "succeeded",
      "amount": 2999,
      "payment_method": {
        "brand": "visa",
        "type": "credit_card",
        "last_four": "1234"
      },
      "created_at": "2026-04-11T12:30:00Z"
    }
  ]
}
```

Get full payment history (paginated):

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/payments?limit=20"
```

---

## 8. Update subscription amount (pay-per-use)

If your pricing is variable, update the subscription amount before the next billing cycle:

```bash
# Customer upgrades; charge more next month
curl -s -X PATCH $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"amount": 4999}'

# Response (HTTP 200):
# {
#   "id": "sub_01j...",
#   "amount": 4999,
#   ...
# }
```

The next payment will use the new amount automatically.

---

## 9. Handle webhooks (production)

In production, Mercado Pago sends webhook notifications when payments settle, customers chargeback, refunds are issued, etc.

Configure Mercado Pago to send webhooks to:
```
https://your-domain.com/webhooks/mercadopago
```

billax validates the webhook signature and processes events:
- Payment settled → update payment status
- Refund issued → create refund record
- Chargeback → dispute management

See [docs/OPERATIONS.md](OPERATIONS.md) for webhook configuration and troubleshooting.

---

## 10. Monitor with Grafana

Open Grafana (http://localhost:3000) to see real-time metrics:

- **Request Rate**: requests/sec across all endpoints
- **Error Rate**: failed requests as percentage
- **Payment Charge Attempts**: by provider and outcome
- **Active Subscriptions**: total subscriptions by status (active, trialing, canceled, etc.)

Your test charges should appear on the "Payment Charge Attempts" panel.

---

## Next Steps

1. **Create more plans** — add plans for different tiers (Basic, Pro, Enterprise)
2. **Implement webhooks** — handle settlement notifications from Mercado Pago
3. **Add more providers** — integrate Stripe, Helipagos, or build a custom connector
4. **Load testing** — verify your setup handles your expected volume (>100 RPS target)
5. **Read [docs/OPERATIONS.md](OPERATIONS.md)** — production deployment, scaling, troubleshooting

---

## Common tasks

### List all subscriptions

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?limit=100"
```

### Filter subscriptions by tag

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/subscriptions?tag=premium&tag=early-adopter"
```

### Cancel a subscription

```bash
curl -s -X POST $BASE_URL/v1/subscriptions/$SUBSCRIPTION_KEY/cancel \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"at_period_end": true}'

# at_period_end=true: cancel at the end of current billing period
# at_period_end=false: cancel immediately
```

### Check your tenant info

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE_URL/v1/me
# {"tenant_id": "..."}
```

### View all API keys for your tenant

```bash
curl -s -H "Authorization: Bearer $API_KEY" $BASE_URL/v1/keys
```

---

## Troubleshooting

**"Subscription creation returns 500"**

Check that migration 005 is applied:
```bash
docker exec payd_db psql -U payd_app -d payd -c "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='subscriptions' AND column_name='idempotency_key')"
# Should return: t
```

**"Payment creation fails with 'invalid_input'"**

Ensure:
- Subscription exists and key is correct
- Amount is a positive integer (in minor units)
- Currency is valid ISO 4217 code
- Mercado Pago credentials are stored and valid

**"Rate limit exceeded"**

billax limits you to 100 requests/minute by default. This is per tenant and configurable. Wait a minute and retry, or contact support to increase your limit.

**"Authentication fails"**

Verify:
- `Authorization: Bearer <key>` header is present
- Key format is correct (`payd_live_...` or `payd_test_...`)
- Key hasn't expired

---

## Further reading

- [docs/TESTING.md](TESTING.md) — detailed API testing examples
- [docs/OPERATIONS.md](OPERATIONS.md) — production deployment and monitoring
- [docs/RAILWAY_DEPLOYMENT.md](RAILWAY_DEPLOYMENT.md) — deploy to Railway with one click
- [README.md](../README.md) — architecture and design principles
