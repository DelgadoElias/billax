# Operations Guide

This guide covers how to run, monitor, and troubleshoot billax in production.

## Observability Stack

### Starting the stack

```bash
docker-compose --profile observability up -d
```

This brings up:
- **billax** (http://localhost:8080) — the API
- **PostgreSQL** (localhost:5432) — the database
- **Prometheus** (http://localhost:9091) — metrics scraper
- **Grafana** (http://localhost:3000) — visualization dashboard

### Shutting down

```bash
docker-compose --profile observability down
```

Containers stop gracefully. Data persists in Docker volumes for next startup.

---

## Grafana Dashboard: "payd — Operational Overview"

The dashboard auto-provisions from `deploy/grafana/provisioning/dashboards/payd.json` and displays 9 panels.

### Panel 1: Request Rate

**What it shows**: HTTP requests per second over time, broken down by path.

**How to read it**:
- Normal traffic appears as a steady line
- Spikes indicate traffic bursts or DDoS
- The legend shows which paths are busiest

**What to do if abnormal**:
- If rate drops to zero → check billax logs for startup errors
- If rate spikes unexpectedly → check for integration test runs; if production, investigate client code changes
- If you see errors in the logs → skip to "Error Rate %" panel

### Panel 2: Error Rate %

**What it shows**: Percentage of requests returning 5xx (server errors).

**How to read it**:
- 0% is ideal
- 1-5% is acceptable during deployments or transient failures
- Above 5% indicates a serious issue

**Alarm thresholds** (recommended):
- Warning: 2%
- Critical: 5%

**What to do if abnormal**:
1. Click the panel to see full graph and exact error types
2. Check billax logs: `docker logs payd_app | grep error`
3. Check database health: `docker exec payd_db pg_isready`
4. If database is down, restart Postgres: `docker-compose restart postgres`
5. If errors persist, check application logs for stack traces

### Panel 3: P99 Latency (seconds)

**What it shows**: 99th percentile request latency (the slowest 1% of requests).

**How to read it**:
- Typical value: 0.1–0.5s
- Spikes indicate slow queries or external API calls

**Alarm thresholds** (recommended):
- Warning: 1s
- Critical: 2s

**What to do if abnormal**:
1. Run a sample request manually and time it: `time curl http://localhost:8080/health`
2. Check database slow log: `docker exec payd_db psql -U payd_app -d payd -c "SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"`
3. If a specific endpoint is slow, check its service implementation for N+1 queries
4. If database queries are slow, check table sizes and indexes

### Panel 4: P50 Latency (seconds)

**What it shows**: Median (50th percentile) request latency.

**How to read it**:
- Typical value: 0.01–0.1s
- Should be much lower than P99

**What to do if abnormal**:
- Use alongside P99 to understand latency distribution
- If P50 is high but P99 is reasonable → a few slow queries are normal
- If both are high → systematic issue (slow DB, high load)

### Panel 5: In-Flight Requests

**What it shows**: Number of requests currently being processed.

**How to read it**:
- Typical value: 1–20 (depends on concurrency settings)
- Sudden jump → traffic spike or requests hanging
- Stuck at high level → possible deadlock or slow query

**What to do if abnormal**:
- If stuck above 50 for >1 minute → likely hung requests; restart billax: `docker-compose restart payd_app`
- Check rate limit configuration — may be too low, causing queue buildup

### Panel 6: Payment Success Rate by Provider

**What it shows**: Percentage of successful payment attempts per provider.

**How to read it**:
- Mercado Pago line should stay at ~95%+ (occasional failures are normal)
- Below 80% indicates a serious provider issue

**Alarm thresholds** (recommended):
- Warning: 90%
- Critical: 80%

**What to do if abnormal**:
1. Check provider status page (e.g., status.mercadopago.com)
2. Verify provider credentials are correct: `docker exec payd_app cat providers.yml`
3. Check billax logs for provider-specific errors: `docker logs payd_app | grep -i "provider\|mercado"`
4. If provider is down, customers cannot pay; consider switching to fallback provider via config update
5. Monitor payment failure reasons in Panel 7

### Panel 7: Payment Attempts by Provider

**What it shows**: Total payment attempts (bar chart) split by success/failure outcome.

**How to read it**:
- Green bars = successful attempts
- Red bars = failed attempts
- Ratio should align with Panel 6 (success rate %)

**What to do if abnormal**:
- High red bar for a provider → correlate with Panel 6 for root cause
- Zero attempts → no subscriptions charging yet, or provider is disabled

### Panel 8: Active Subscriptions by Status

**What it shows**: Pie chart of subscription counts by status (trialing, active, past_due, canceled, expired).

**How to read it**:
- Total count = sum of all slices
- Large "canceled" slice may indicate churn issues
- Large "past_due" slice indicates payment failures

**What to do if abnormal**:
- If "active" is unexpectedly low → check if webhooks are updating subscription status
- If "expired" grows without bound → consider pruning stale subscriptions

### Panel 9: HTTP Status Distribution

**What it shows**: Bar chart of all requests by HTTP status code.

**How to read it**:
- 200/201 (green) should dominate
- 400/401 (yellow) indicate client errors (bad requests, auth failures)
- 500+ (red) indicate server errors

**How to read error codes**:
- 400 → validation error; check request body
- 401 → auth error; check API key
- 404 → resource not found
- 409 → conflict (e.g., duplicate idempotency key)
- 429 → rate limited; client is sending too many requests
- 500 → server error; check billax logs
- 503 → service unavailable; likely database issue

**What to do if abnormal**:
- If 400 are high → document common errors in API docs or improve validation messages
- If 401 are high → check API key distribution/rotation
- If 429 are high → increase `RATE_LIMIT_DEFAULT` or implement per-tenant limits
- If 500 are high → see Error Rate % panel

---

## Troubleshooting

### Billax won't start

**Symptoms**: Container exits immediately after start.

**Steps**:
1. Check logs: `docker logs payd_app`
2. Look for:
   - "failed to initialize database" → database not running or URL wrong
   - "failed to load provider capabilities" → `providers.yml` is invalid YAML
   - "cannot read body" → likely a panic during request parsing

**Fix**:
- Verify `DATABASE_URL` is correct
- Verify `providers.yml` exists and is valid YAML
- Ensure all required env vars are set (see `config/`)
- Restart: `docker-compose restart payd_app`

### High error rate (Panel 2)

**Symptoms**: "Error Rate %" is above 5%.

**Steps**:
1. Check Prometheus: what HTTP status codes are dominant? (Panel 9)
2. Grep logs for errors: `docker logs payd_app 2>&1 | grep -i error | head -20`
3. For 5xx errors, look for stack traces in the logs
4. For payment errors, check if provider is down (status.mercadopago.com)

**Common causes and fixes**:
- **Database unreachable** → Restart postgres: `docker-compose restart postgres`
- **Provider rate limited** → Add exponential backoff in connector; reduce charge frequency
- **Invalid provider config** → Check credentials in database: `SELECT * FROM provider_credentials`
- **Out of disk space** → Check: `docker exec payd_db df -h`; clean up old logs if needed

### Payment failures

**Symptoms**: Payments fail but error message is vague.

**Steps**:
1. Check payment status in database:
   ```sql
   SELECT id, status, failure_reason, provider_response
   FROM payments
   ORDER BY created_at DESC LIMIT 10;
   ```
2. Look at `failure_reason` — it comes from the provider
3. Look at `provider_response` — raw provider response (JSONB)

**Common causes**:
- **"card_declined"** → Cardholder issue, not billax
- **"insufficient_funds"** → Customer's account doesn't have enough
- **"rate_limit"** → Provider hit rate limit; retry later (idempotency key ensures safety)
- **"invalid_token"** → Token expired or customer removed payment method

### Slow requests (Panel 3/4)

**Symptoms**: P99 latency > 2s.

**Steps**:
1. Identify slow endpoint: click the panel, see legend by path
2. Check database slow log:
   ```bash
   docker exec payd_db psql -U payd_app -d payd -c \
     "SELECT query, calls, mean_exec_time FROM pg_stat_statements
      ORDER BY mean_exec_time DESC LIMIT 5;"
   ```
3. If queries are slow, check table sizes and missing indexes

**Common causes**:
- **N+1 queries** → Service loads plan, then for each subscription loads plan again. Fix: batch load or use JOINs
- **Unindexed column in WHERE** → Add index on the column
- **External API call** → Payment provider is slow; consider timeout and retry logic
- **Database under load** → Check CPU: `docker stats payd_db`; if high, scale up DB or optimize queries

### Metrics not updating

**Symptoms**: Grafana dashboard is blank or stuck at old values.

**Steps**:
1. Check Prometheus is scraping: http://localhost:9091/targets
   - Should see `payd` target with state "UP"
   - If "DOWN", check billax `/metrics` endpoint: `curl http://localhost:8080/metrics`
2. If `/metrics` returns 500, check billax logs for metric collection errors
3. If `/metrics` is empty, check metrics code in `internal/metrics/`

**Common causes**:
- **Metrics disabled** → Set `METRICS_ENABLED=true` in env
- **Metrics port wrong** → Check `METRICS_PORT` matches docker-compose config
- **No data yet** → Metrics initialize at zero; they need traffic or background jobs to populate

### Subscription metric stuck at 0

**Symptoms**: `payd_active_subscriptions` gauge always shows 0.

**Steps**:
1. Check background poller is running: `docker logs payd_app | grep "subscription metrics poller"`
2. Check for errors in the poller: `docker logs payd_app | grep -i "failed to update"`
3. Verify subscriptions exist in DB:
   ```bash
   docker exec payd_db psql -U payd_app -d payd -c "SELECT COUNT(*) FROM subscriptions;"
   ```

**Common causes**:
- **Poller not started** → Restart billax: `docker-compose restart payd_app`
- **Poller hitting an error** → Check logs for SQL errors; verify table permissions
- **No subscriptions created yet** → Create a test subscription and wait 30 seconds for the poller

---

## Scaling and Capacity

### Database

PostgreSQL settings in `docker-compose.yml`:
- Memory: 256MB default (fine for dev, increase for prod)
- Connections: 100 (sufficient for small deployments)
- WAL level: replica (enables point-in-time recovery)

**To increase capacity**:
1. Edit `docker-compose.yml` environment variables
2. Restart: `docker-compose restart postgres`
3. Monitor memory: `docker stats payd_db`

### Billax app

Concurrency limits:
- `RATE_LIMIT_DEFAULT=100` (requests/minute per tenant)
- HTTP server has no hard concurrency limit (Go runtime scales automatically)

**To increase throughput**:
1. Increase `RATE_LIMIT_DEFAULT` if hitting limits
2. Scale horizontally: run multiple instances behind a load balancer
3. Add caching layer (Redis) for frequently accessed plans/subscriptions

### Metrics storage

Prometheus stores data in-memory and on disk. Default: 15 days retention.

**To increase retention**:
1. Edit `docker-compose.yml` Prometheus command: add `--storage.tsdb.retention.time=90d`
2. Ensure disk space: `docker exec payd_prometheus df -h /prometheus`
3. Restart: `docker-compose restart prometheus`

---

## Alerting (Recommended)

Set up Grafana alerts for these conditions:

| Metric | Condition | Action |
|--------|-----------|--------|
| Error Rate | > 5% for 5 min | Page oncall |
| P99 Latency | > 2s for 10 min | Investigate DB/provider |
| Payment Success Rate | < 80% for 10 min | Check provider status |
| In-Flight Requests | > 100 for 2 min | Scale or investigate hang |
| Database CPU | > 80% for 10 min | Scale or optimize queries |

Grafana alerting docs: https://grafana.com/docs/grafana/latest/alerting/

---

## Additional Resources

- **Troubleshooting**: See README.md and provider-specific docs
- **Mercado Pago setup**: See docs/MERCADO_PAGO_SETUP.md
- **API reference**: See api/openapi.yaml
- **Provider credentials**: See docs/PROVIDER_CREDENTIALS.md
- **Database schema**: See migrations/
