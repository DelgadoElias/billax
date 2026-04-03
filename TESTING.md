# Testing Guide: Local → Sandbox → Production

This guide walks through testing the payd billing system at different levels, culminating in deployment to Railway with Mercado Pago Sandbox.

## ✅ Level 1: Unit Tests (Local, no DB/MP required)

### Run all tests
```bash
go test ./... -v
```

### Run specific tests
```bash
# Provider credentials tests
go test ./internal/providercredentials/... -v

# Mercado Pago connector tests
go test ./internal/provider/mercadopago/... -v
```

**Result:** 48 tests passing, validates:
- ✅ Config validation
- ✅ Amount conversion
- ✅ Webhook signature validation
- ✅ Retry logic with backoff
- ✅ RLS isolation

---

## ✅ Level 2: Integration Tests (Local with Docker Postgres)

### Prerequisites
```bash
# Docker Desktop installed
docker --version

# Docker Compose
docker-compose --version
```

### Start services
```bash
docker-compose up -d
```

This starts:
- PostgreSQL 15 on localhost:5432
- Mocked Mercado Pago API (optional: httptest)

### Run migrations
```bash
# The app auto-runs migrations on startup
# Or manually:
psql postgres://payd_app:password@localhost:5432/payd -f migrations/001_init.sql
psql postgres://payd_app:password@localhost:5432/payd -f migrations/002_plan_slug_subscription_tags.sql
psql postgres://payd_app:password@localhost:5432/payd -f migrations/003_planless_subscriptions.sql
psql postgres://payd_app:password@localhost:5432/payd -f migrations/004_provider_credentials.sql
```

### Start the app
```bash
export DATABASE_URL="postgres://payd_app:password@localhost:5432/payd?sslmode=disable"
export APP_ENV=development
export PORT=8080

go run ./cmd/payd/main.go
```

### Test the flow (in another terminal)

**1. Create a tenant**
```bash
# Create tenant (this would normally be an admin operation)
psql $DATABASE_URL << EOF
INSERT INTO tenants (id, name, api_key_hash)
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Tenant', 'hash');
EOF

TENANT_ID="550e8400-e29b-41d4-a716-446655440001"
API_KEY="payd_live_test_key"  # This is a placeholder; normally from /v1/keys endpoint
```

**2. Configure Mercado Pago credentials**
```bash
curl -X POST http://localhost:8080/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "TEST_ACCESS_TOKEN",
    "webhook_secret": "TEST_WEBHOOK_SECRET"
  }'

# Expected: 201 Created
# Response: {"provider": "mercadopago", "message": "credentials configured"}
```

**3. List configured providers**
```bash
curl -X GET http://localhost:8080/v1/provider-credentials \
  -H "Authorization: Bearer $API_KEY"

# Expected: 200 OK
# Response: {"providers": ["mercadopago"]}
```

**4. Create a plan**
```bash
curl -X POST http://localhost:8080/v1/plans \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "monthly-pro",
    "name": "Monthly Pro",
    "amount": 9999,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1
  }'

# Expected: 201 Created
# Store the plan ID from response
PLAN_ID="<uuid>"
```

**5. Create a subscription**
```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: sub-001" \
  -H "Content-Type: application/json" \
  -d "{
    \"plan_id\": \"$PLAN_ID\",
    \"external_customer_id\": \"customer@example.com\",
    \"provider_name\": \"mercadopago\"
  }"

# Expected: 201 Created
# Store subscription key from response
SUB_KEY="<uuid_v7>"
```

**6. Create a payment (triggers Mercado Pago API)**
```bash
curl -X POST http://localhost:8080/v1/subscriptions/$SUB_KEY/payments \
  -H "Authorization: Bearer $API_KEY" \
  -H "Idempotency-Key: payment-001" \
  -H "Content-Type: application/json" \
  -d '{
    "provider_name": "mercadopago"
  }'

# Expected: 201 Created
# Response: {
#   "id": "pay_xxx",
#   "status": "pending|succeeded|failed",
#   "provider_charge_id": "mp_payment_id",
#   "amount": 9999,
#   "currency": "ARS"
# }
```

### Cleanup
```bash
docker-compose down
```

---

## ✅ Level 3: Sandbox Testing (Local with Real MP Sandbox)

### Get Mercado Pago Sandbox Credentials

1. **Create account** → https://www.mercadopago.com.ar/developers
2. **Login** → Dashboard
3. **Settings** → Integrations
4. **Copy:**
   - `Access Token` (Sandbox or Production mode)
   - Keep note of `Public Key` (for frontend)

### Update credentials
```bash
curl -X POST http://localhost:8080/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR_XXXXXXX",
    "webhook_secret": "whsec_xxxxx"
  }'
```

### Test payment creation
The same curl from Level 2 step 6, but now hits real MP Sandbox API.

**Expected flow:**
1. App validates credentials with MP
2. MP returns payment ID and status
3. Payment is stored in DB
4. Response sent to client

### Test webhook (from MP)

**1. Configure webhook URL in MP Dashboard**
- Settings → Webhooks
- URL: `http://your-tunnel.ngrok.io/webhooks/mercadopago`
- Events: `payment`

**2. Use ngrok to expose local server**
```bash
# Install ngrok: https://ngrok.com/download
ngrok http 8080

# Copy the forwarding URL: https://xxxxx.ngrok.io
# Configure in MP Dashboard with: https://xxxxx.ngrok.io/webhooks/mercadopago
```

**3. Process a payment in MP Sandbox and observe webhook**
- MP Sandbox sends webhook to your ngrok URL
- App validates signature
- App logs webhook received
- Response: 200 OK sent to MP

---

## 🚀 Level 4: Production Deployment on Railway

### Prerequisites
- Railway account (https://railway.app)
- GitHub repository with payd code
- Mercado Pago production account (with approved credentials)

### Step 1: Create Railway Database

```bash
# Via Railway dashboard:
# 1. New Project
# 2. Add Service → Postgres
# 3. Configure:
#    - Postgres version: 15
#    - Database: payd
#    - User: payd_app
#    - Password: (auto-generated)
# 4. Get DATABASE_URL from environment
```

### Step 2: Run Migrations on Railway

```bash
# Via Railway CLI or dashboard:
railway run psql $DATABASE_URL < migrations/001_init.sql
railway run psql $DATABASE_URL < migrations/002_plan_slug_subscription_tags.sql
railway run psql $DATABASE_URL < migrations/003_planless_subscriptions.sql
railway run psql $DATABASE_URL < migrations/004_provider_credentials.sql
```

### Step 3: Deploy App to Railway

```bash
# Option A: Connect GitHub repo (easiest)
# 1. Push code to GitHub
# 2. Railway Dashboard → New Project → Connect GitHub Repo
# 3. Select payd repo
# 4. Railway auto-detects Dockerfile and deploys

# Option B: Manual deployment
railway link  # Connect to your Railway project
railway up    # Deploy
```

### Step 4: Configure Environment Variables

Railway Dashboard → payd service → Variables:

```env
DATABASE_URL=postgres://payd_app:PASSWORD@host:port/payd?sslmode=require
APP_ENV=production
PORT=8080
LOG_LEVEL=info
PROVIDERS_CONFIG_PATH=providers.yml
RATE_LIMIT_DEFAULT=100
```

### Step 5: Configure Mercado Pago

1. **Production account on MP** → Settings → Integrations
2. **Copy production access_token and webhook_secret**
3. **Configure webhook URL** → `https://your-railway-app.railway.app/webhooks/mercadopago`
4. **Test in MP Dashboard** → Send test webhook

### Step 6: Test on Production

```bash
# Get your Railway app URL
RAILWAY_URL="https://your-railway-app.railway.app"
API_KEY="payd_live_production_key"  # From admin panel

# 1. Check health
curl $RAILWAY_URL/health

# 2. Configure credentials
curl -X POST $RAILWAY_URL/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR_PRODUCTION",
    "webhook_secret": "whsec_production"
  }'

# 3. Create plan, subscription, payment
# (Same as Level 2, just with $RAILWAY_URL instead of localhost:8080)
```

---

## 🧪 Testing Checklist

### Before Sandbox
- [ ] All tests pass locally (`go test ./...`)
- [ ] App starts without errors
- [ ] Health endpoint works
- [ ] Credentials can be stored and retrieved

### Before Railway Deployment
- [ ] Credentials work with real MP Sandbox
- [ ] Payment creation succeeds
- [ ] Webhook signature validation works
- [ ] Idempotency prevents duplicate payments
- [ ] Migrations run successfully

### Production (Railway)
- [ ] Database connection via DATABASE_URL
- [ ] App starts and health check passes
- [ ] Credentials configured with production MP
- [ ] End-to-end payment flow works
- [ ] Webhook reception and validation works
- [ ] Logs are accessible in Railway dashboard

---

## 🚨 Common Issues

### "Invalid credentials"
```
Solution: Verify access_token and webhook_secret match MP account
         Check MP Dashboard → Settings → Integrations for correct values
```

### "Webhook signature validation failed"
```
Solution: Ensure webhook_secret matches MP Dashboard
         Check X-Signature and X-Request-ID headers are present
         Verify webhook secret is URL-encoded if needed
```

### "Rate limited (429)"
```
Solution: App auto-retries with exponential backoff
         If persists: check MP rate limits on your account
         Production may need higher rate limits from MP
```

### "Database connection refused"
```
Solution: Verify DATABASE_URL is correct
         Check PostgreSQL is running and accessible
         On Railway: verify Postgres service is running
```

---

## 📊 Monitoring in Production

### Railway Dashboard
- Logs → see app output and errors
- Metrics → CPU, memory, network
- Deployments → see deploy history

### App Logs
```bash
railway logs --follow
```

### Database Health
```bash
railway run psql $DATABASE_URL -c "SELECT 1;"
```

### Payment Status
```bash
# Query payments from DB
railway run psql $DATABASE_URL << EOF
SELECT id, status, provider_name, created_at FROM payments ORDER BY created_at DESC LIMIT 10;
EOF
```

---

## 🎯 Next Steps After Deployment

1. **Monitor first 24 hours** — Watch logs for errors
2. **Test webhook recovery** — Manually resend failed webhooks
3. **Set up alerting** — Slack/email alerts on errors
4. **Implement dashboard** — Monitor payment status
5. **Document API** — Create API docs for tenants

---

## References

- Railway Docs: https://docs.railway.app
- Mercado Pago Sandbox: https://sandbox.mercadopago.com
- ngrok: https://ngrok.com (for local webhook testing)
- PostgreSQL: https://www.postgresql.org/docs/15/
