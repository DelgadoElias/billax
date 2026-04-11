-- Migration: Add idempotency_key column to subscriptions table
-- Purpose: Enable idempotent subscription creation

ALTER TABLE subscriptions
ADD COLUMN idempotency_key TEXT;

-- Add unique constraint for idempotency (tenant_id, idempotency_key)
-- This ensures that creating a subscription with the same idempotency key returns the same subscription
ALTER TABLE subscriptions
ADD CONSTRAINT subscriptions_tenant_idempotency_key_key UNIQUE (tenant_id, idempotency_key);

-- Create index for fast lookups by idempotency key
CREATE INDEX idx_subscriptions_tenant_idempotency_key
    ON subscriptions(tenant_id, idempotency_key);
