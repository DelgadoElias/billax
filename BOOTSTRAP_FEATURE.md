# Bootstrap Feature - Auto-Create First Tenant on Startup

**Status**: ✅ **IMPLEMENTED & TESTED**

The bootstrap feature automatically creates the first tenant and admin backoffice user when the application starts, similar to Grafana's `GF_SECURITY_ADMIN_PASSWORD` pattern. This eliminates the need to manually call the signup endpoint in fresh deployments.

---

## Configuration

Bootstrap is **disabled by default**. To enable it, set these three environment variables:

| Variable | Description | Required |
|---|---|---|
| `BOOTSTRAP_TENANT_NAME` | Tenant display name (e.g., "Acme Corp") | Yes (to activate) |
| `BOOTSTRAP_TENANT_EMAIL` | Tenant email & admin backoffice user email | Yes (to activate) |
| `BOOTSTRAP_ADMIN_PASSWORD` | Admin backoffice user password | Yes (to activate) |
| `BOOTSTRAP_TENANT_SLUG` | Tenant URL slug (e.g., "acme-corp") | No (auto-generated from name) |

**Activation Logic**: Bootstrap runs **only if all three required variables are set**. If none are set, bootstrap silently skips. If only 1-2 are set, a warning is logged and bootstrap is skipped (does not fail startup).

---

## Docker Compose Example

### Default (Bootstrap Disabled)
```yaml
services:
  payd:
    environment:
      DATABASE_URL: postgres://payd_app:devpassword@postgres:5432/payd?sslmode=disable
      BOOTSTRAP_TENANT_NAME: ""        # empty = disabled
      BOOTSTRAP_TENANT_EMAIL: ""
      BOOTSTRAP_ADMIN_PASSWORD: ""
```

### With Bootstrap Enabled
```yaml
services:
  payd:
    environment:
      DATABASE_URL: postgres://payd_app:devpassword@postgres:5432/payd?sslmode=disable
      BOOTSTRAP_TENANT_NAME: "Acme Corp"
      BOOTSTRAP_TENANT_EMAIL: "admin@acme.com"
      BOOTSTRAP_ADMIN_PASSWORD: "SecurePass123!"
      BOOTSTRAP_TENANT_SLUG: "acme-corp"     # optional
```

### Railway Deployment Example
```bash
# Set environment variables in Railway dashboard:
export BOOTSTRAP_TENANT_NAME="Acme Corp"
export BOOTSTRAP_TENANT_EMAIL="admin@acme.com"
export BOOTSTRAP_ADMIN_PASSWORD="SecurePass123!"
export BOOTSTRAP_TENANT_SLUG="acme-corp"

# Then deploy:
railway up
```

---

## Idempotence & Safety

The bootstrap feature is **fully idempotent** — it's safe to restart the application with the same configuration:

| Scenario | Behavior | Log Output |
|---|---|---|
| **First startup, vars set** | Creates tenant + admin user | ✅ "bootstrap: creando tenant inicial" → "bootstrap: tenant creado" → "bootstrap: completado" |
| **Restart with same vars** | Detects existing tenant, skips user creation | ✅ "bootstrap: tenant ya existe, usando el existente" → "bootstrap: usuario admin ya existe, skipping" |
| **Bootstrap disabled (empty vars)** | No bootstrap action taken | ✅ Silent return, no logs |
| **Partial vars (1-2 set)** | Skips bootstrap with warning | ⚠️ "bootstrap skipped: ... deben estar todas configuradas" |

---

## Startup Flow

```
APPLICATION START
  ↓
Load Configuration (including BOOTSTRAP_* vars)
  ↓
Run Database Migrations (auto-apply schema)
  ↓
Connect to Database
  ↓
BOOTSTRAP PHASE (if configured)
  ├─ Check if all 3 vars are present
  ├─ If none → silent return (bootstrap disabled)
  ├─ If partial → log warning, skip
  ├─ If all present →
  │  ├─ Create Tenant (or get existing if conflict)
  │  ├─ Create Backoffice Admin User (skip if already exists)
  │  └─ Log "bootstrap: completado"
  ↓
Load Provider Capabilities
  ↓
Initialize Repositories & Services
  ↓
Start HTTP Server (port 8080)
  ↓
Start Metrics Server (port 9090)
  ↓
Start Background Jobs (lifecycle runner)
  ↓
Ready for Requests ✅
```

---

## Testing the Feature

### Test 1: Fresh Start with Bootstrap

```bash
# 1. Clean database
docker compose down -v

# 2. Start with bootstrap enabled
export BOOTSTRAP_TENANT_NAME="Test Company"
export BOOTSTRAP_TENANT_EMAIL="admin@test.com"
export BOOTSTRAP_ADMIN_PASSWORD="Test@1234"

docker compose up payd

# 3. Check logs for bootstrap messages
docker compose logs payd | grep bootstrap
# Output:
# time=... level=INFO msg="bootstrap: creando tenant inicial"
# time=... level=INFO msg="bootstrap: tenant creado" id=... slug=...
# time=... level=INFO msg="bootstrap: completado"

# 4. Verify login works immediately
curl -X POST http://localhost:8080/v1/backoffice/check-email \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@test.com"}'
# Response: {"single_tenant": true, "tenant_slug": "..."}

curl -X POST http://localhost:8080/v1/backoffice/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@test.com", "password":"Test@1234", "tenant_slug":"test-company"}'
# Response: {"token": "...", "user": {...}}
```

### Test 2: Idempotence (Restart)

```bash
# 1. Restart the app with same bootstrap vars
docker compose restart payd

# 2. Check logs - should show "ya existe" messages
docker compose logs payd | grep bootstrap
# Output:
# time=... level=INFO msg="bootstrap: creando tenant inicial"
# time=... level=INFO msg="bootstrap: tenant ya existe, usando el existente"
# time=... level=INFO msg="bootstrap: usuario admin ya existe, skipping"

# 3. Verify no duplicates - login still works
curl -X POST http://localhost:8080/v1/backoffice/login ...
# Same user returned, no duplicates created ✅
```

### Test 3: Bootstrap Disabled

```bash
# 1. Start with empty bootstrap vars (default)
BOOTSTRAP_TENANT_NAME="" \
BOOTSTRAP_TENANT_EMAIL="" \
BOOTSTRAP_ADMIN_PASSWORD="" \
docker compose up payd

# 2. Check logs - NO bootstrap messages
docker compose logs payd | grep bootstrap
# (No output - bootstrap silently disabled)

# 3. Verify normal behavior - signup endpoint still works
curl -X POST http://localhost:8080/v1/signup ...
```

---

## Implementation Details

### Files Modified

1. **`internal/config/config.go`**
   - Added 4 new optional fields to `Config` struct
   - Parse environment variables with `os.Getenv()` (no defaults)

2. **`cmd/payd/main.go`**
   - Added `bootstrapTenant()` function
   - Call after database connection, before provider loading
   - Handles `ErrConflict` gracefully for idempotence
   - Imports: `errors`, `strings`, `pgxpool`, `apperrors`

3. **`docker-compose.yml`**
   - Added 4 bootstrap environment variables (empty by default)
   - Includes comments explaining usage

### Error Handling

- **Tenant creation failure** (non-conflict): Log error and return (fail safe)
- **Tenant already exists**: Retrieve existing tenant by email and continue
- **Admin user creation failure** (non-conflict): Log error and return
- **Admin user already exists**: Log "skipping" and continue (idempotent)
- **Partial configuration**: Log warning and skip bootstrap (does not fail startup)

---

## Security Considerations

1. **Password Requirements**: Admin password must meet the configured policy (minimum 8 chars, uppercase, lowercase, digit, special char)
2. **Email Validation**: Email is validated in tenant creation
3. **Tenant Isolation**: Backoffice user is created in tenant context (RLS-compliant)
4. **No Secrets in Logs**: Bootstrap logs only contain names and slugs, never passwords
5. **One-Time Setup**: Should be configured once at first deployment; change password afterward if needed

---

## Production Checklist

- ✅ Generate strong passwords (minimum 12 characters with mixed case, numbers, symbols)
- ✅ Use environment variable files (`.env` files) not committed to git
- ✅ For Railway/Cloud: Use dashboard to set secret environment variables
- ✅ Verify login works after first deployment
- ✅ Change admin password post-deployment if using a default
- ✅ Monitor logs during first startup to confirm bootstrap completed
- ✅ Disable bootstrap (set empty strings) for subsequent restarts if desired

---

## Troubleshooting

### Bootstrap runs every restart
**Issue**: Admin user creation fails every restart  
**Solution**: Check logs for error message; likely a password policy mismatch. Update `BOOTSTRAP_ADMIN_PASSWORD` to meet policy requirements.

### "bootstrap skipped" warning in logs
**Issue**: Only 1-2 of the required vars are set  
**Solution**: Set all 3 required vars or set all 3 to empty strings (to disable).

### Login fails after bootstrap
**Issue**: Check-email or login endpoint returns 404  
**Solution**: Verify tenant slug matches `BOOTSTRAP_TENANT_SLUG` or auto-generated slug from name. Check logs for actual slug created.

### Duplicate user error on restart
**Issue**: "duplicate key value violates unique constraint"  
**Solution**: This is now handled gracefully in v0.1.0+; update to latest version.

---

## Related Files

- Configuration: `internal/config/config.go`
- Application Startup: `cmd/payd/main.go`
- Tenant Management: `internal/tenant/`
- Backoffice Users: `internal/backoffice/`
- Login Endpoints: `internal/backoffice/handler.go`

