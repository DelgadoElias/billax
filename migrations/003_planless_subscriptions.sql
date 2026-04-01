-- ============================================================================
-- Migration 003: Planless subscriptions + subscription-owned billing fields
-- ============================================================================

-- Make plan_id nullable (was NOT NULL REFERENCES plans(id))
ALTER TABLE subscriptions
    ALTER COLUMN plan_id DROP NOT NULL;

-- Add billing fields directly on subscriptions.
-- These are copied from the plan at creation time when a plan is provided,
-- or supplied directly by the caller for planless subscriptions.
ALTER TABLE subscriptions
    ADD COLUMN amount        BIGINT,
    ADD COLUMN currency      TEXT,
    ADD COLUMN interval      TEXT CHECK (interval IN ('day','week','month','year')),
    ADD COLUMN interval_count INT;

-- Backfill existing rows from their referenced plan.
-- This runs once; existing rows always have a plan_id.
UPDATE subscriptions s
SET
    amount         = p.amount,
    currency       = p.currency,
    interval       = p.interval,
    interval_count = p.interval_count
FROM plans p
WHERE s.plan_id = p.id;

-- Now enforce NOT NULL on the billing fields.
-- Every row — old or new — must have these values.
ALTER TABLE subscriptions
    ALTER COLUMN amount        SET NOT NULL,
    ALTER COLUMN currency      SET NOT NULL,
    ALTER COLUMN interval      SET NOT NULL,
    ALTER COLUMN interval_count SET NOT NULL;
