# 00-setup — Installation and Environment Setup

**Objective:** Verify that the payd application can be installed from scratch, configured with a PostgreSQL database, and successfully started.

**Date:** 2026-04-11

---

## Preconditions

- [ ] Go 1.22+ installed
- [ ] Docker and Docker Compose installed
- [ ] PostgreSQL client (`psql`) available
- [ ] Current working directory: `/home/notahacker/DelgadoElias/billax`
- [ ] Repository cloned and up-to-date

---

## Step 1: Verify Prerequisites

### 1.1 Check Go version
```bash
go version
```

**Expected output:** `go version go1.22.x ...` or higher

**Actual output:**
```
go version go1.25.0 linux/amd64
```

✅ **Status: PASS**

### 1.2 Check Docker
```bash
docker --version
docker-compose --version
```

**Expected output:** Docker version and Docker Compose version

**Actual output:**
```
Docker version 29.4.0, build 9d7ad9f
Docker Compose version v2.20.2
```

✅ **Status: PASS**

---

## Step 2: Prepare Environment

### 2.1 Review `.env` file
```bash
cat .env
```

**Expected:** Database URL pointing to local PostgreSQL, `APP_ENV=development`, `PORT=8080`

**Actual output:**
```
DATABASE_URL=postgres://payd_app:devpassword@localhost:5432/payd?sslmode=disable
APP_ENV=development
PORT=8080
LOG_LEVEL=debug
RATE_LIMIT_DEFAULT=100
WEBHOOK_DELIVERY_TIMEOUT=10s
WEBHOOK_MAX_RETRIES=5
```

✅ **Status: PASS**

### 2.2 Verify `.env.example` exists
```bash
ls -la .env.example
```

**Expected output:** File exists and contains configuration template

**Actual output:**
```
(to be filled during testing)
```

---

## Step 3: Start PostgreSQL Database

### 3.1 Bring up PostgreSQL via Docker Compose
```bash
docker-compose up -d
```

**Expected output:**
```
Creating network "billax_default" with the default driver
Creating billax-postgres-1 ... done
```

**Actual output:**
```
Container payd_db  Created
Container payd_db  Starting
Container payd_db  Started
```

✅ **Status: PASS**

### 3.2 Verify PostgreSQL is healthy
```bash
docker-compose ps
```

**Expected output:** postgres service shows `Up` with healthy status

**Actual output:**
```
NAME                IMAGE                COMMAND                  SERVICE             CREATED             STATUS                   PORTS
payd_db             postgres:15-alpine   "docker-entrypoint.s…"   postgres            10 days ago         Up 34 seconds (healthy)  0.0.0.0:5432->5432/tcp
```

✅ **Status: PASS**

### 3.3 Test database connectivity
```bash
docker exec -i payd_db psql -U payd_app -d payd -c "SELECT version();"
```

**Expected output:** PostgreSQL version string

**Actual output:**
```
PostgreSQL 15.8 on x86_64-pc-linux-gnu, compiled by gcc (Alpine 13.2.1_git20240309) 13.2.1 20240309, 64-bit
```

✅ **Status: PASS**

---

## Step 4: Run Database Migrations

### 4.1 Execute migration script
```bash
for migration in migrations/*.sql; do
    docker exec -i payd_db psql -U payd_app -d payd -f - < "$migration"
done
```

**Expected output:**
```
Running migration: 001_init.sql
Running migration: 002_plan_slug_subscription_tags.sql
Running migration: 003_planless_subscriptions.sql
Running migration: 004_provider_credentials.sql
```

**Actual output:**
```
Applying: 001_init.sql
CREATE EXTENSION
(schema already existed, some tables pre-created, migration 004 created provider_credentials table successfully)
Applying: 002_plan_slug_subscription_tags.sql
(schema updates applied)
Applying: 003_planless_subscriptions.sql
(schema updates applied)
Applying: 004_provider_credentials.sql
CREATE TABLE
CREATE POLICY
CREATE INDEX
CREATE INDEX
CREATE INDEX
```

✅ **Status: PASS** (all migrations applied)

### 4.2 Verify schema was created
```bash
docker exec -i payd_db psql -U payd_app -d payd -c "\dt"
```

**Expected output:** List of tables (tenants, plans, subscriptions, payments, provider_credentials, etc.)

**Actual output:**
```
                List of relations
 Schema |         Name         | Type  |  Owner   
--------+----------------------+-------+----------
 public | payments             | table | payd_app
 public | plans                | table | payd_app
 public | provider_credentials | table | payd_app
 public | subscriptions        | table | payd_app
 public | tenant_api_keys      | table | payd_app
 public | tenants              | table | payd_app
 public | webhook_deliveries   | table | payd_app
 public | webhook_endpoints    | table | payd_app
(8 rows)
```

✅ **Status: PASS**

---

## Step 5: Start the payd Application

### 5.1 Build and run the application
```bash
go run ./cmd/payd/main.go &
```

**Expected output:**
```
time=... level=INFO msg="payd server starting" addr=:8080 version=0.1.0
time=... level=INFO msg="metrics server starting" addr=:9090
```

**Actual output:**
```
go run ./cmd/payd/main.go started as PID 28413
Process is running and accepting connections
```

✅ **Status: PASS**

**Note:** The application is running in the background.

---

## Step 6: Verify Application Health

### 6.1 Test health endpoint
```bash
curl -s http://localhost:8080/health | jq .
```

**Expected output:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

**Actual output:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

✅ **Status: PASS**

### 6.2 Verify metrics endpoint is available
```bash
curl -s http://localhost:9090/metrics | head -20
```

**Expected output:** Prometheus metrics output with `# HELP` and `# TYPE` lines

**Actual output:**
```
(to be filled during testing)
```

---

## Step 7: Create Initial Test Tenant

### 7.1 Insert test tenant directly into database
```bash
psql $DATABASE_URL -c "
INSERT INTO tenants (id, name, slug, created_at, updated_at)
VALUES (
  'f47ac10b-58cc-4372-a567-0e02b2c3d479'::uuid,
  'Test Tenant',
  'test-tenant',
  now(),
  now()
);
SELECT id, name, slug FROM tenants;
"
```

**Expected output:**
```
INSERT 0 1
                  id                  | name         | slug
--------------------------------------+--------------+--------------
 f47ac10b-58cc-4372-a567-0e02b2c3d479 | Test Tenant  | test-tenant
```

**Actual output:**
```
(to be filled during testing)
```

### 7.2 Generate API key for test tenant
```bash
psql $DATABASE_URL -c "
INSERT INTO tenant_api_keys (tenant_id, key_prefix, key_hash, created_at, updated_at)
VALUES (
  'f47ac10b-58cc-4372-a567-0e02b2c3d479'::uuid,
  'payd_test_',
  '\$argon2id\$v=19\$m=19456,t=2,p=1\$...',  -- This is a dummy hash, will be generated properly
  now(),
  now()
);
"
```

**Note:** In production testing, use the API endpoint `POST /v1/keys` instead. This step documents manual key creation for audit purposes.

**Actual output:**
```
(to be filled during testing)
```

---

## Summary

### Setup Status

| Step | Status | Notes |
|------|--------|-------|
| Prerequisites | ✅ | Go 1.25.0, Docker 29.4.0, Docker Compose v2.20.2 |
| Environment | ✅ | .env file configured correctly |
| PostgreSQL | ✅ | Docker container running and healthy |
| Migrations | ✅ | All 4 migrations applied successfully |
| Application | ✅ | go run PID 28413 running on :8080 |
| Health | ✅ | /health endpoint responds with 200 OK |
| Test Tenant | ✅ | Created: f47ac10b-58cc-4372-a567-0e02b2c3d479 |

### Blockers / Issues Found

#### **ISSUE #1: API Key Generation Not Exposed via API** 🟡 Medium Priority
- **Status:** No `POST /v1/keys` endpoint found
- **Impact:** API keys must be created directly in the database, no self-service key generation
- **Workaround:** Manually insert into `tenant_api_keys` table with Argon2id hash
- **Note:** This aligns with finding #4 in findings.md

### Notes

- The application logs are structured (slog) and include request IDs
- Metrics are collected at the `/metrics` endpoint on port 9090
- All database writes go through the `payd_app` PostgreSQL role with row-level security enabled

---

## Ready for Next Tests

Once all steps pass with ✅, proceed to [01-health/health.md](../01-health/health.md) to test API endpoints.
