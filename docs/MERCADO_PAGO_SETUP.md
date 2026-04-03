# Mercado Pago Sandbox Setup Guide

Complete guide to setting up and testing payd with Mercado Pago Sandbox.

## 📋 Prerequisites

- Mercado Pago account (free at https://www.mercadopago.com.ar)
- payd deployed locally or on Railway
- curl or Postman for API testing
- ngrok (for local webhook testing) — https://ngrok.com/download

## 🔑 Step 1: Get Mercado Pago Credentials

### 1.1 Create a Mercado Pago Account
```
https://www.mercadopago.com.ar
Sign up with email
Verify email
```

### 1.2 Get Sandbox Credentials

**Option A: Dashboard (Recommended)**

1. Go to https://www.mercadopago.com.ar/developers/panel
2. Login with your account
3. On left sidebar: **Integrations**
4. Under **API Keys**:
   - **Access Token (Sandbox)**: Copy this value
   - Save it safely (starts with `APP_USR_...`)

**Option B: API (Alternative)**

```bash
curl -X POST https://api.mercadopago.com/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "grant_type": "client_credentials",
    "scope": "api"
  }'
```

### 1.3 Configure Webhook Secret

1. Dashboard → **Integrations** → **Webhooks**
2. Webhook URL: `http://localhost:8080/webhooks/mercadopago` (for local) or your Railway URL
3. Events: Select `payment`
4. Dashboard shows webhook secret (starts with `whsec_...`)

## 🧪 Step 2: Local Testing with Sandbox

### 2.1 Start your local environment

```bash
# Terminal 1: Start the app
chmod +x scripts/test-local.sh
./scripts/test-local.sh

# This will:
# - Start Postgres in Docker
# - Run migrations
# - Run tests
# - Start the app on http://localhost:8080
```

### 2.2 Configure credentials in payd

```bash
# Use your actual MP Sandbox credentials
curl -X POST http://localhost:8080/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer payd_live_test" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR_XXXXXXXXXXXXXXXXXXX",
    "webhook_secret": "whsec_XXXXXXXXXXXXXXXX"
  }'

# Expected response:
# HTTP 201 Created
# {"provider": "mercadopago", "message": "credentials configured"}
```

### 2.3 Create a test plan

```bash
curl -X POST http://localhost:8080/v1/plans \
  -H "Authorization: Bearer payd_live_test" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "test-plan",
    "name": "Test Plan",
    "amount": 10000,
    "currency": "ARS",
    "interval": "month",
    "interval_count": 1
  }'

# Save the returned plan_id
PLAN_ID="<uuid>"
```

### 2.4 Create a test subscription

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Authorization: Bearer payd_live_test" \
  -H "Idempotency-Key: test-sub-001" \
  -H "Content-Type: application/json" \
  -d "{
    \"plan_id\": \"$PLAN_ID\",
    \"external_customer_id\": \"test@example.com\",
    \"provider_name\": \"mercadopago\"
  }"

# Save the returned subscription_key
SUB_KEY="<uuid_v7>"
```

### 2.5 Create a test payment with Sandbox

```bash
curl -X POST http://localhost:8080/v1/subscriptions/$SUB_KEY/payments \
  -H "Authorization: Bearer payd_live_test" \
  -H "Idempotency-Key: test-payment-001" \
  -H "Content-Type: application/json" \
  -d '{
    "provider_name": "mercadopago"
  }'
```

**Expected response:**
```json
{
  "id": "pay_123abc",
  "status": "pending",
  "provider_name": "mercadopago",
  "provider_charge_id": "1234567890",
  "amount": 10000,
  "currency": "ARS",
  "payment_method": {
    "brand": "visa",
    "type": "credit_card",
    "last_four": "4242"
  }
}
```

### 2.6 Test Webhooks Locally with ngrok

**Terminal 2: Expose your local app**
```bash
ngrok http 8080

# You'll see:
# Forwarding: https://abcd1234.ngrok.io -> http://localhost:8080
```

**Update Mercado Pago Webhook**
1. Dashboard → **Integrations** → **Webhooks**
2. Webhook URL: `https://abcd1234.ngrok.io/webhooks/mercadopago`
3. Save

**Simulate Payment from MP Dashboard**
1. MP Dashboard → **Simulations**
2. Create a test payment
3. Check ngrok terminal to see webhook received

**Expected logs:**
```
Webhook received from mercadopago: payment_id=1234567890, event_type=payment
```

## 🚀 Step 3: Production Setup on Railway

### 3.1 Deploy to Railway

```bash
# If not done already
git push origin main

# Railway auto-deploys from GitHub
# Or manually:
railway up
```

### 3.2 Get Railway App URL

```bash
railway env | grep RAILWAY_PUBLIC_DOMAIN
# Returns something like: your-app.railway.app
```

### 3.3 Configure Mercado Pago Production

1. Create a **production account** on MP (separate from Sandbox)
2. Get production **Access Token**
3. Dashboard → **Integrations** → **Webhooks**
4. Webhook URL: `https://your-app.railway.app/webhooks/mercadopago`
5. Save webhook secret

### 3.4 Configure payd on Railway

```bash
# Via Railway Dashboard or CLI:
railway variables set \
  DATABASE_URL="postgres://..." \
  APP_ENV="production" \
  LOG_LEVEL="info" \
  PORT="8080"
```

### 3.5 Configure credentials on Railway

```bash
curl -X POST https://your-app.railway.app/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer payd_live_production_key" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR_PRODUCTION_TOKEN",
    "webhook_secret": "whsec_production_secret"
  }'
```

## 🧾 Test Card Numbers

For MP Sandbox, use these test cards:

| Card Type | Number | CVV | Expiry |
|-----------|--------|-----|--------|
| **Visa** | 4111 1111 1111 1111 | 123 | 11/25 |
| **Mastercard** | 5425 2334 3010 9903 | 123 | 11/25 |
| **Amex** | 3782 822463 10005 | 1234 | 11/25 |

**Approval email:** any@example.com
**CPF:** 12345678909

## ✅ Testing Checklist

### Credentials
- [ ] Access Token saved in payd
- [ ] Webhook Secret saved in payd
- [ ] Credentials validated with `GET /provider-credentials`

### Payments (Sandbox)
- [ ] Plan created
- [ ] Subscription created with plan
- [ ] Payment created successfully
- [ ] Payment status received from MP

### Webhooks (Sandbox)
- [ ] Webhook URL configured in MP
- [ ] ngrok/public URL working
- [ ] Webhook signature validates
- [ ] Webhook received logged by app
- [ ] Payment status updatable from webhook

### Production (Railway)
- [ ] App deployed and healthy (`/health` returns 200)
- [ ] Database connected (migrations ran)
- [ ] Production credentials configured
- [ ] Payment created in production
- [ ] Webhook received from production MP

## 🚨 Troubleshooting

### "Invalid credentials"
**Symptom:** 422 Unprocessable Entity when configuring credentials
**Solution:**
- Verify access_token and webhook_secret from MP Dashboard
- Check for typos or extra spaces
- Try getting a new token from MP

### "Webhook signature validation failed"
**Symptom:** 401 Unauthorized for webhook
**Solution:**
- Verify webhook_secret matches MP Dashboard
- Check headers: X-Signature and X-Request-ID
- Test with `curl` to verify headers

### "Payment failed with provider error"
**Symptom:** Payment creation returns 502 Bad Gateway
**Solution:**
- Check app logs: `railway logs --follow`
- Verify access_token is still valid (tokens expire)
- Check MP Sandbox is operational
- Verify rate limits not exceeded

### "Webhook never arrives"
**Symptom:** Payment created but no webhook received
**Solution:**
- Verify webhook URL in MP Dashboard
- Check ngrok is running (if testing locally)
- Check app logs for webhook attempts
- Test webhook manually from MP Dashboard

## 📚 References

- MP API Docs: https://www.mercadopago.com/developers/en/reference
- MP Sandbox: https://sandbox.mercadopago.com
- payd TESTING.md: See root directory for full testing guide
- Railway Docs: https://docs.railway.app

## 📞 Support

- **payd issues:** Check TESTING.md or GitHub issues
- **MP issues:** https://www.mercadopago.com/developers/support
- **Railway issues:** https://docs.railway.app/support
