# Provider Credentials Management

This package handles secure storage and retrieval of payment provider credentials (API keys, webhooks secrets, etc.) on a per-tenant, per-provider basis.

## Overview

Credentials are stored in the `provider_credentials` table with:
- **Tenant isolation** via PostgreSQL RLS
- **Validation** with the provider's `ValidateConfig()` method before storage
- **JSONB storage** for flexibility across different providers
- **Audit logging** of configuration changes (no secrets logged)

## Architecture

```
┌─────────────────────────────┐
│ HTTP Handler (handler.go)   │
│ POST /provider-credentials/ │
└────────────┬────────────────┘
             ↓
┌─────────────────────────────┐
│ Service (service.go)        │
│ ├─ SetProviderConfig()      │
│ ├─ GetProviderConfig()      │
│ ├─ ValidateAndFetch()       │
│ └─ ListProviders()          │
└────────────┬────────────────┘
             ↓
┌─────────────────────────────┐
│ Repository (repository.go)  │
│ ├─ Set(upsert)              │
│ ├─ Get()                    │
│ ├─ List()                   │
│ └─ Delete()                 │
└────────────┬────────────────┘
             ↓
┌─────────────────────────────┐
│ PostgreSQL (RLS enforced)   │
│ provider_credentials table  │
└─────────────────────────────┘
```

## API Usage

### 1. Configure Mercado Pago credentials

```bash
curl -X POST https://your-instance/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer payd_live_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "APP_USR_1234567890...",
    "webhook_secret": "whsec_abcd1234..."
  }'
```

**Response (201 Created):**
```json
{
  "provider": "mercadopago",
  "message": "credentials configured"
}
```

### 2. List configured providers

```bash
curl -X GET https://your-instance/v1/provider-credentials \
  -H "Authorization: Bearer payd_live_xxx"
```

**Response (200 OK):**
```json
{
  "providers": ["mercadopago", "stripe"]
}
```

### 3. Check if provider is configured

```bash
curl -X GET https://your-instance/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer payd_live_xxx"
```

**Response (200 OK):**
```json
{
  "provider": "mercadopago",
  "configured": true
}
```

### 4. Delete credentials

```bash
curl -X DELETE https://your-instance/v1/provider-credentials/mercadopago \
  -H "Authorization: Bearer payd_live_xxx"
```

**Response (204 No Content)**

## Service Methods

### SetProviderConfig
```go
err := credSvc.SetProviderConfig(ctx, tenantID, "mercadopago", config)
```

**Flow:**
1. Validates provider is registered in the adapter
2. Calls `provider.ValidateConfig()` to check credentials are valid
3. Stores in DB (upsert on `UNIQUE(tenant_id, provider_name)`)
4. Logs the change (no secret values)

**Returns:** `ErrInvalidInput` if validation fails

### GetProviderConfig
```go
config, err := credSvc.GetProviderConfig(ctx, tenantID, "mercadopago")
// Returns: map[string]string with keys like "access_token", "webhook_secret"
```

**Returns:** `ErrInvalidInput` if not found (privacy: doesn't leak existence)

### ValidateAndFetch
```go
config, err := credSvc.ValidateAndFetch(ctx, tenantID, "mercadopago")
```

**Flow:**
1. Verifies provider is registered
2. Fetches config from DB
3. Returns config for use in API calls

Used by payment service to fetch credentials before creating charges.

### ListProviders
```go
providers, err := credSvc.ListProviders(ctx, tenantID)
// Returns: []string{"mercadopago", "stripe"}
```

### DeleteProviderConfig
```go
err := credSvc.DeleteProviderConfig(ctx, tenantID, "mercadopago")
```

**Returns:** `ErrNotFound` if credentials don't exist

## Database Schema

```sql
CREATE TABLE provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_name TEXT NOT NULL CHECK (provider_name IN ('mercadopago', 'stripe', 'helipagos')),
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, provider_name)
);

ALTER TABLE provider_credentials ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON provider_credentials
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

### Example record

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "660e8400-e29b-41d4-a716-446655440000",
  "provider_name": "mercadopago",
  "config": {
    "access_token": "APP_USR_1234567890...",
    "webhook_secret": "whsec_abcd1234..."
  },
  "created_at": "2026-04-03T16:00:00Z",
  "updated_at": "2026-04-03T16:00:00Z"
}
```

## Integration with Payment Creation

**Current flow (before integration):**
```
POST /v1/subscriptions/{key}/payments
{
  "provider_name": "mercadopago",
  "access_token": "...",           // ← In request body (insecure!)
  "webhook_secret": "..."          // ← In request body (insecure!)
}
```

**Future flow (after integration):**
```
POST /v1/subscriptions/{key}/payments
{
  "provider_name": "mercadopago"   // ← Only provider, no credentials!
}
```

**PaymentService will:**
1. Extract `tenantID` from auth context
2. Call `credSvc.ValidateAndFetch(ctx, tenantID, "mercadopago")`
3. Get credentials from DB
4. Pass to provider adapter

## Security Considerations

### ✅ Strengths

1. **RLS at database layer** — Tenants cannot access each other's credentials
2. **Validation before storage** — Invalid configs rejected immediately
3. **No secrets in logs** — Config values never logged
4. **No secrets in responses** — API returns `{configured: true}`, never the actual values
5. **Encrypted in transit** — HTTPS enforced by application constraints
6. **Immutable after set** — Updates go through same validation flow

### ⚠️ Considerations

1. **Database access** — Anyone with DB access can read credentials. Mitigate:
   - Use database-level encryption at rest (RDS encryption)
   - Limit DB access to application servers
   - Use VPC security groups

2. **Secret rotation** — Credentials must be rotated regularly:
   - POST `/v1/provider-credentials/{provider}` with new values
   - Old credentials become invalid immediately
   - Recommend: annual rotation, immediate rotation on key compromise

3. **Audit trail** — Changes logged but not payload:
   ```
   "provider credentials updated"
   tenant_id: <uuid>
   provider: mercadopago
   ```
   No `access_token` or `webhook_secret` logged.

## Testing

Unit tests use a mock repository:

```go
func setupTestService(t *testing.T) (*CredentialsService, uuid.UUID) {
    registry := provider.NewRegistry()
    registry.Register(mercadopago.New())
    adapter := provider.NewAdapter(registry, provider.CapabilitiesConfig{})
    mockRepo := newMockRepository()
    svc := NewService(mockRepo, adapter)
    tenantID := uuid.New()
    return svc, tenantID
}
```

**Coverage:**
- ✅ Valid config storage
- ✅ Missing token/secret rejection
- ✅ Unknown provider rejection
- ✅ Retrieval of stored config
- ✅ Non-existence returns `ErrInvalidInput`
- ✅ List all providers
- ✅ Delete provider config
- ✅ Nil tenant ID rejection

Run tests:
```bash
go test ./internal/providercredentials... -v
```

## Future Enhancements

- [ ] **Secret encryption at rest** — Use AWS KMS or HashiCorp Vault
- [ ] **Credential rotation tracking** — Audit table with rotation history
- [ ] **API key expiration** — Set `expires_at` field, enforce refresh
- [ ] **Separate webhook secrets** — Store separately from API credentials
- [ ] **Provider-specific validation schemas** — Custom validators per provider type

## References

- [Provider Interface](../provider/provider.go)
- [PaymentProvider Implementation](../provider/mercadopago/provider.go)
- [Handler Example](handler.go)
- [Service Implementation](service.go)
- [Database RLS](../migrations/004_provider_credentials.sql)
