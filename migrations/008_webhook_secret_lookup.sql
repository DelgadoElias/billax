-- Migration 008: Webhook secret reverse lookup for public webhook endpoints
-- Problem: provider_credentials table has RLS, but webhook routing needs to find tenant by secret
-- Solution: Add separate webhook_secret column + SECURITY DEFINER function to bypass RLS
--
-- This allows:
-- 1. Fast indexed lookup (UNIQUE constraint prevents secret collisions)
-- 2. RLS bypass via SECURITY DEFINER (webhook handler discovers tenant without auth)
-- 3. Backward compatibility (existing plaintext credentials auto-backfilled)

BEGIN;

-- Add webhook_secret column for reverse lookup (separate from encrypted config)
ALTER TABLE provider_credentials ADD COLUMN webhook_secret TEXT;

-- Index for fast lookup by secret (webhook handler uses this)
CREATE UNIQUE INDEX idx_provider_credentials_webhook_secret
    ON provider_credentials(webhook_secret)
    WHERE webhook_secret IS NOT NULL;

-- Backfill from existing plaintext credentials (config->>'webhook_secret')
-- Only for rows that are NOT encrypted (no '_enc' marker in config)
UPDATE provider_credentials
SET webhook_secret = config->>'webhook_secret'
WHERE config->>'webhook_secret' IS NOT NULL
  AND config->>'_enc' IS NULL;

-- SECURITY DEFINER function: bypasses RLS for webhook routing
-- This runs as the table owner (superuser), NOT as payd_app
-- It safely returns only the columns webhook handler needs (tenant_id, provider_name, config)
-- The app layer handles decryption of config JSONB if needed
CREATE OR REPLACE FUNCTION lookup_tenant_by_webhook_secret(p_secret TEXT)
RETURNS TABLE(tenant_id UUID, provider_name TEXT, config JSONB)
LANGUAGE sql
SECURITY DEFINER
STABLE
AS $$
    SELECT tenant_id, provider_name, config
    FROM provider_credentials
    WHERE webhook_secret = p_secret
    LIMIT 1;
$$;

-- Grant execution to app role (payd_app)
GRANT EXECUTE ON FUNCTION lookup_tenant_by_webhook_secret(TEXT) TO payd_app;

COMMIT;
