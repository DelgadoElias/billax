-- ============================================================================
-- Migration 002: Add slug to plans, tags to subscriptions, payment_method to payments
-- ============================================================================

-- Add slug to plans (idempotency anchor for payment buttons)
ALTER TABLE plans ADD COLUMN slug TEXT;

-- Backfill existing rows with a slug derived from name
UPDATE plans SET slug = LOWER(REGEXP_REPLACE(name, '\s+', '-', 'g')) WHERE slug IS NULL;

-- Now enforce NOT NULL
ALTER TABLE plans ALTER COLUMN slug SET NOT NULL;

-- Unique slug per tenant (this is the idempotency anchor for upserts)
ALTER TABLE plans ADD CONSTRAINT uq_plans_tenant_slug UNIQUE (tenant_id, slug);

CREATE INDEX idx_plans_tenant_slug ON plans(tenant_id, slug);

-- Add tags to subscriptions (for filtering and labeling)
ALTER TABLE subscriptions ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';

-- GIN index for array containment queries: WHERE tags @> ARRAY['premium']
CREATE INDEX idx_subscriptions_tags ON subscriptions USING GIN (tags);

-- Add payment_method to payments (non-sensitive payment metadata)
ALTER TABLE payments ADD COLUMN payment_method JSONB;
