-- Migration: Add provider_credentials table for storing payment provider configuration
-- Purpose: Store per-tenant, per-provider credentials securely

CREATE TABLE provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_name TEXT NOT NULL,
    config JSONB NOT NULL,  -- {"access_token": "...", "webhook_secret": "..."}
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraint: only allow known providers
    CONSTRAINT valid_provider_name CHECK (
        provider_name IN ('mercadopago', 'stripe', 'helipagos')
    ),

    -- Constraint: one config per tenant per provider
    CONSTRAINT unique_tenant_provider UNIQUE (tenant_id, provider_name)
);

-- Enable Row Level Security
ALTER TABLE provider_credentials ENABLE ROW LEVEL SECURITY;

-- RLS policy: tenants can only see their own credentials
CREATE POLICY tenant_isolation ON provider_credentials
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes for fast lookups
CREATE INDEX idx_provider_credentials_tenant_id
    ON provider_credentials(tenant_id);
CREATE INDEX idx_provider_credentials_tenant_provider
    ON provider_credentials(tenant_id, provider_name);
CREATE INDEX idx_provider_credentials_provider
    ON provider_credentials(provider_name);
