# Railway Deployment Guide

This guide walks you through deploying billax to Railway and configuring it for production use.

## Quick Deploy

Click the button below to deploy billax to Railway with a single click:

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new?templateId=https://github.com/DelgadoElias/billax)

The button creates a new Railway project with:
- billax application
- PostgreSQL 15 database
- All environment variables pre-configured

---

## What Happens During Deployment

1. **Project creation**: Railway creates a new project and links to your GitHub account
2. **Git integration**: Your fork of billax is connected to the project
3. **Database setup**: PostgreSQL 15 is provisioned with a new database `payd`
4. **App build**: Docker build runs using the Dockerfile
5. **Migrations**: You manually run SQL migrations (see below)
6. **App start**: The `./payd` binary runs with environment variables from railway.json

The deployment takes ~3-5 minutes from click to ready.

---

## Post-Deployment Setup

### 1. Verify the App is Running

In the Railway dashboard:
1. Go to "Deployments" tab
2. Wait for green checkmark next to the latest deployment
3. Click the service name to see logs
4. Verify no errors in the logs

Test the health endpoint (replace `payd-production.railway.app` with your Railway URL):

```bash
curl https://payd-production.railway.app/health
# {"status":"ok","version":"0.1.0"}
```

### 2. Get the Database Connection String

The `DATABASE_URL` is automatically set by Railway when you add the Postgres plugin. Verify it:

1. In Railway dashboard, go to the **Postgres** service
2. Click the **Connect** tab
3. Copy the connection string under "Postgres Connection URL"
4. In the **payd-app** service, go to **Variables** tab
5. Verify `DATABASE_URL` is populated (it should be auto-set)

### 3. Run Database Migrations

Migrations must be run before the app can start. They create the schema and tables.

**Option A: Via Railway CLI (recommended)**

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login
railway login

# Link to your project
railway link

# Run migrations in the remote database
railway run psql -U postgres -d payd < migrations/001_init.sql
railway run psql -U postgres -d payd < migrations/002_plan_slug_subscription_tags.sql
railway run psql -U postgres -d payd < migrations/003_planless_subscriptions.sql
```

**Option B: Via psql locally**

```bash
# From the Postgres connection string, extract credentials
DATABASE_URL="postgres://user:pass@host:5432/payd?sslmode=require"

# Extract components and run migrations
psql "$DATABASE_URL" < migrations/001_init.sql
psql "$DATABASE_URL" < migrations/002_plan_slug_subscription_tags.sql
psql "$DATABASE_URL" < migrations/003_planless_subscriptions.sql
```

After migrations, redeploy the app (go to Deployments → Redeploy latest).

### 4. Configure Payment Provider Credentials

billax supports payment providers via `provider_credentials` table. Before accepting payments, register your provider credentials.

#### For Mercado Pago:

1. Get credentials from Mercado Pago:
   - Log in to [developer.mercadopago.com](https://developer.mercadopago.com)
   - Navigate to "My integrations" → "Credentials"
   - Copy your **Access Token** (production or sandbox)
   - Copy your **Webhook Secret** (if using webhooks)

2. Register credentials in billax:

   ```bash
   curl -X POST https://payd-production.railway.app/v1/provider-credentials \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "provider_name": "mercadopago",
       "credentials": {
         "access_token": "YOUR_ACCESS_TOKEN",
         "webhook_secret": "YOUR_WEBHOOK_SECRET"
       }
     }'
   ```

3. To get an API key, first create a tenant:

   ```bash
   curl -X POST https://payd-production.railway.app/v1/tenants \
     -H "Content-Type: application/json" \
     -d '{
       "name": "My Company",
       "email": "admin@mycompany.com"
     }'
   ```

   This returns a tenant ID and API key. Use the API key to register credentials above.

For details on Mercado Pago setup, see [docs/MERCADO_PAGO_SETUP.md](MERCADO_PAGO_SETUP.md).

### 5. Create a Test Plan

Verify the API is working by creating a test plan:

```bash
curl -X POST https://payd-production.railway.app/v1/plans \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "test-plan",
    "name": "Test Plan",
    "amount": 100000,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1
  }'
```

If this returns 201 with a plan object, the API is working.

### 6. Set Custom Domain (Optional)

By default, your app gets a Railway subdomain (e.g., `payd-production.railway.app`).

To use a custom domain:
1. In Railway dashboard, go to **payd-app** service
2. Click **Settings** → **Domain**
3. Add your custom domain (e.g., `api.billing.mycompany.com`)
4. Update your DNS records as shown in Railway

### 7. Configure Webhooks (If Using Mercado Pago)

If you're using Mercado Pago, configure webhooks so MP notifies billax of payment events:

1. Log in to [developer.mercadopago.com](https://developer.mercadopago.com)
2. Go to "My integrations" → "Webhooks"
3. Add webhook URL: `https://YOUR_DOMAIN/webhooks/mercadopago`
4. Select events: `payment.created`, `payment.updated`
5. Save

billax will now receive payment status updates from Mercado Pago.

---

## Monitoring

### Logs

View real-time logs in the Railway dashboard:
1. Click the **payd-app** service
2. Go to **Logs** tab
3. Filter by severity or keyword as needed

### Metrics

billax exposes Prometheus metrics at `/metrics`. To access them:

```bash
curl https://payd-production.railway.app/metrics
```

For visualization, you can:
- Use a monitoring service like Datadog or New Relic (they can scrape `/metrics`)
- Set up a separate Prometheus instance to scrape the metrics
- Use Railway's built-in metrics (go to Dashboard → Metrics)

---

## Troubleshooting

### App won't start / keeps restarting

**Symptoms**: Deployment shows red X or "Failed" status.

**Steps**:
1. Check logs: Click service → Logs tab, look for error messages
2. Common errors:
   - `"failed to initialize database"` → DATABASE_URL not set or wrong, OR migrations not run
   - `"cannot read providers.yml"` → providers.yml missing in repo
   - `"listen tcp"` → PORT variable is wrong

**Fix**:
- Verify `DATABASE_URL` is set in the **payd-app** Variables
- Re-run migrations
- Restart the service: go to Deployments → Redeploy latest

### Database connection fails

**Symptoms**: Logs show `"failed to connect to database"` or timeouts.

**Steps**:
1. Verify Postgres is running: In Railway dashboard, check Postgres service status (should be green)
2. Verify `DATABASE_URL` is correct: In **payd-app** Variables, copy the URL
3. Test the connection locally:
   ```bash
   psql "$DATABASE_URL"
   ```
4. If that fails, the URL is wrong or the database is unreachable

**Fix**:
- If Postgres crashed, restart it: Dashboard → Postgres service → Redeploy
- If DATABASE_URL is wrong, copy it from Postgres service → Connect tab
- If you don't have psql installed, use Railway CLI: `railway run psql ...`

### Migrations failed

**Symptoms**: App runs but API returns 500 errors or "table doesn't exist".

**Steps**:
1. Verify migrations ran: `railway run psql -U postgres -d payd -c "SELECT * FROM plans LIMIT 1;"`
   - If error, migrations didn't run
2. Check for partial failures: `railway run psql -U postgres -d payd -c "\dt"` (list tables)
   - Should see: plans, subscriptions, payments, tenants, api_keys, provider_credentials, idempotency_keys

**Fix**:
- Re-run all migrations:
  ```bash
  railway run psql -U postgres -d payd < migrations/001_init.sql
  railway run psql -U postgres -d payd < migrations/002_plan_slug_subscription_tags.sql
  railway run psql -U postgres -d payd < migrations/003_planless_subscriptions.sql
  ```
- Restart the app: Deployments → Redeploy latest

### Payment provider errors

**Symptoms**: Payment creation fails with "invalid credentials" or "not found".

**Steps**:
1. Verify provider credentials are registered:
   ```bash
   curl -X GET https://payd-production.railway.app/v1/provider-credentials \
     -H "Authorization: Bearer YOUR_API_KEY"
   ```
2. If no credentials returned, register them (see step 4 above)
3. Check provider status: [status.mercadopago.com](https://status.mercadopago.com)

**Fix**:
- Verify credentials are correct (Access Token not expired, Webhook Secret matches)
- Re-register credentials if needed
- Check provider status page for outages

### API returns 401 Unauthorized

**Symptoms**: All requests return 401 even with valid API key.

**Steps**:
1. Verify API key format: should be `payd_live_<base64>` or `payd_test_<base64>`
2. Verify Authorization header: `Authorization: Bearer payd_live_...` (note the space)
3. Verify key hasn't expired: In your database, check api_keys table

**Fix**:
- Copy the API key from tenant creation response
- Use correct header format: `Authorization: Bearer KEY`
- If key expired, create a new tenant/key

### Custom domain not working

**Symptoms**: Custom domain returns "DNS error" or "Connection refused".

**Steps**:
1. Verify DNS records: Ask your registrar for current DNS records
2. Verify Railway domain settings: Dashboard → Service → Settings → Domain
3. Wait for DNS propagation (can take 1-10 minutes): `nslookup yourdomain.com`

**Fix**:
- Update DNS to point to Railway's CNAME (shown in Railway)
- Wait for DNS to propagate
- Test: `curl https://yourdomain.com/health`

---

## Environment Variables Reference

All variables are pre-configured in railway.json, but you can override them:

| Variable | Default | Description | Notes |
|----------|---------|-------------|-------|
| `DATABASE_URL` | (auto-set) | PostgreSQL DSN | Auto-set by Railway Postgres plugin |
| `APP_ENV` | `production` | Environment mode | `development` or `production` |
| `PORT` | `8080` | HTTP server port | Don't change; Railway sets it |
| `LOG_LEVEL` | `info` | Logging level | `debug`, `info`, `warn`, `error` |
| `RATE_LIMIT_DEFAULT` | `100` | Rate limit (req/min) | Per tenant, per minute |
| `METRICS_ENABLED` | `true` | Enable /metrics | Set to `false` to disable |
| `METRICS_PORT` | `9090` | Metrics port | Metrics endpoint port |
| `PROVIDERS_CONFIG_PATH` | `providers.yml` | Provider config file | Usually `providers.yml` |

To change a variable:
1. Go to **payd-app** → **Variables**
2. Edit the value
3. Save (triggers redeployment)

---

## Scaling and Limits

### Replica Count

By default, 1 replica runs. To scale horizontally:
1. Go to **payd-app** → **Settings**
2. Change "Num Replicas" to 2, 3, etc.
3. Save (new instances start automatically)

### Database Limits

Railway's PostgreSQL default plan supports:
- 10 GB storage
- 256 MB RAM
- Suitable for small to medium deployments

To upgrade:
1. Go to **Postgres** service
2. Click **Upgrade Plan**
3. Choose a larger plan (pricing increases)

### Traffic Limits

billax can handle ~100 requests/second per instance with default rate limiting.

To increase:
1. Increase `RATE_LIMIT_DEFAULT` (but watch database load)
2. Add more replicas (go to payd-app → Settings → Num Replicas)
3. Scale database if needed

---

## Costs

Railway charges based on:
- **Compute**: $5/month per instance (payd-app)
- **Database**: $15-30/month for PostgreSQL (depending on plan)
- **Bandwidth**: $0.10/GB egress (included: ingress, internal traffic)

Typical small deployment: ~$20-40/month.

To reduce costs:
- Use 1 replica instead of multiple
- Use PostgreSQL Starter plan instead of larger plans
- Monitor bandwidth usage (Dashboard → Usage)

---

## Additional Resources

- **Operations guide**: See [docs/OPERATIONS.md](OPERATIONS.md) for monitoring and troubleshooting
- **Provider setup**: See [docs/MERCADO_PAGO_SETUP.md](MERCADO_PAGO_SETUP.md)
- **API reference**: See [api/openapi.yaml](../api/openapi.yaml)
- **Railway docs**: [https://docs.railway.app](https://docs.railway.app)
