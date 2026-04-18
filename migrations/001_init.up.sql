-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE: tenants
-- ============================================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- TABLE: tenant_api_keys
-- ============================================================================
CREATE TABLE tenant_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{read,write}',
    description TEXT,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, key_prefix)
);

CREATE INDEX idx_tenant_api_keys_key_prefix ON tenant_api_keys(key_prefix);
CREATE INDEX idx_tenant_api_keys_tenant_id ON tenant_api_keys(tenant_id);

-- ============================================================================
-- TABLE: plans
-- ============================================================================
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'ARS',
    interval TEXT NOT NULL CHECK (interval IN ('day','week','month','year')),
    interval_count INT NOT NULL DEFAULT 1,
    trial_days INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE plans ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_plans ON plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_plans_tenant_id ON plans(tenant_id);

-- ============================================================================
-- TABLE: subscriptions
-- ============================================================================
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    subscription_key UUID NOT NULL UNIQUE,
    external_customer_id TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('trialing','active','past_due','canceled','expired')),
    provider_name TEXT,
    provider_subscription_id TEXT,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    trial_ends_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_subscriptions ON subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_subscription_key ON subscriptions(subscription_key);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- ============================================================================
-- TABLE: payments
-- ============================================================================
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    idempotency_key TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    provider_charge_id TEXT,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','succeeded','failed','refunded')),
    failure_reason TEXT,
    provider_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, idempotency_key)
);

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_payments ON payments
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX idx_payments_status ON payments(status);

-- ============================================================================
-- TABLE: webhook_endpoints
-- ============================================================================
CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    events TEXT[] NOT NULL DEFAULT '{*}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE webhook_endpoints ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_webhook_endpoints ON webhook_endpoints
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_webhook_endpoints_tenant_id ON webhook_endpoints(tenant_id);

-- ============================================================================
-- TABLE: webhook_deliveries
-- ============================================================================
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','succeeded','failed')),
    attempt_count INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ,
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_webhook_deliveries ON webhook_deliveries
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_webhook_deliveries_tenant_id ON webhook_deliveries(tenant_id);

-- ============================================================================
-- ROLE: payd_app (limited permissions)
-- ============================================================================
-- Note: This is idempotent and database-agnostic.
-- On Neon or other managed services, the role may already exist and this will be a no-op.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'payd_app') THEN
        CREATE ROLE payd_app WITH LOGIN PASSWORD 'P@ssw0rd!DevRole2026';
    END IF;
EXCEPTION WHEN OTHERS THEN
    -- If role creation fails (e.g., on Neon), continue
    NULL;
END
$$;

-- Grant permissions on current database (works regardless of DB name)
-- Note: GRANT CONNECT is not required if role already exists
DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO payd_app', current_database());
EXCEPTION WHEN OTHERS THEN
    -- On Neon or managed services, GRANT CONNECT may fail; that's OK
    NULL;
END
$$;

GRANT USAGE ON SCHEMA public TO payd_app;

-- Grant SELECT, INSERT, UPDATE, DELETE on all tables (no DDL)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO payd_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO payd_app;

-- Ensure RLS is enforced for payd_app role (if not already set)
DO $$
BEGIN
    ALTER ROLE payd_app SET row_security = on;
EXCEPTION WHEN OTHERS THEN
    -- If ALTER fails (permissions), continue; RLS is a nice-to-have
    NULL;
END
$$;
