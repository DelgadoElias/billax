-- Migration 009: Subscription lifecycle indexes and cross-tenant lookup functions
-- Problem: Lifecycle jobs need to find subscriptions across all tenants without auth context
-- Solution: Add indexes for lifecycle queries + SECURITY DEFINER functions to bypass RLS
--
-- Lifecycle jobs need to:
-- 1. Find subscriptions due for renewal (status=active, current_period_end <= now)
-- 2. Find expired trials (status=trialing, trial_ends_at <= now)
-- 3. Find past_due subscriptions pending expiry (status=past_due, current_period_end <= grace_period)

BEGIN;

-- Index for renewal lookup: active subscriptions ordered by period end
CREATE INDEX idx_subscriptions_lifecycle
    ON subscriptions(status, current_period_end)
    WHERE status IN ('active', 'past_due', 'trialing');

-- Index for trial expiry lookups
CREATE INDEX idx_subscriptions_trial_expiry
    ON subscriptions(trial_ends_at)
    WHERE status = 'trialing' AND trial_ends_at IS NOT NULL;

-- SECURITY DEFINER function: Find subscriptions due for renewal
-- Cross-tenant query for lifecycle job (runs as table owner, bypasses RLS)
CREATE OR REPLACE FUNCTION list_subscriptions_due_for_renewal(p_now TIMESTAMP WITH TIME ZONE)
RETURNS TABLE(id UUID, tenant_id UUID, subscription_key UUID, external_customer_id TEXT, status TEXT,
              provider_name TEXT, provider_subscription_id TEXT, current_period_end TIMESTAMP WITH TIME ZONE,
              trial_ends_at TIMESTAMP WITH TIME ZONE, amount BIGINT, currency TEXT,
              interval_type TEXT, interval_count INT)
LANGUAGE sql
SECURITY DEFINER
STABLE
AS $$
    SELECT
        s.id, s.tenant_id, s.subscription_key, s.external_customer_id, s.status,
        s.provider_name, s.provider_subscription_id, s.current_period_end,
        s.trial_ends_at, s.amount, s.currency, s.interval, s.interval_count
    FROM subscriptions s
    WHERE s.status = 'active'
      AND s.current_period_end <= p_now
      AND s.cancel_at_period_end = false
      AND s.provider_name IS NOT NULL
    ORDER BY s.current_period_end ASC
    LIMIT 100;
$$;

-- SECURITY DEFINER function: Find subscriptions with expired trials
-- Cross-tenant query for lifecycle job
CREATE OR REPLACE FUNCTION list_subscriptions_expired_trials(p_now TIMESTAMP WITH TIME ZONE)
RETURNS TABLE(id UUID, tenant_id UUID, subscription_key UUID, external_customer_id TEXT, status TEXT,
              provider_name TEXT, provider_subscription_id TEXT, current_period_start TIMESTAMP WITH TIME ZONE,
              current_period_end TIMESTAMP WITH TIME ZONE, trial_ends_at TIMESTAMP WITH TIME ZONE,
              amount BIGINT, currency TEXT, interval_type TEXT, interval_count INT)
LANGUAGE sql
SECURITY DEFINER
STABLE
AS $$
    SELECT
        s.id, s.tenant_id, s.subscription_key, s.external_customer_id, s.status,
        s.provider_name, s.provider_subscription_id, s.current_period_start, s.current_period_end,
        s.trial_ends_at, s.amount, s.currency, s.interval, s.interval_count
    FROM subscriptions s
    WHERE s.status = 'trialing'
      AND s.trial_ends_at <= p_now
    ORDER BY s.trial_ends_at ASC
    LIMIT 100;
$$;

-- SECURITY DEFINER function: Find past_due subscriptions pending expiry
-- Cross-tenant query for lifecycle job (grace period check)
CREATE OR REPLACE FUNCTION list_subscriptions_past_due_pending_expiry(p_grace_period_end TIMESTAMP WITH TIME ZONE)
RETURNS TABLE(id UUID, tenant_id UUID, subscription_key UUID, external_customer_id TEXT, status TEXT,
              provider_name TEXT, provider_subscription_id TEXT, current_period_end TIMESTAMP WITH TIME ZONE,
              amount BIGINT, currency TEXT, interval_type TEXT, interval_count INT)
LANGUAGE sql
SECURITY DEFINER
STABLE
AS $$
    SELECT
        s.id, s.tenant_id, s.subscription_key, s.external_customer_id, s.status,
        s.provider_name, s.provider_subscription_id, s.current_period_end,
        s.amount, s.currency, s.interval, s.interval_count
    FROM subscriptions s
    WHERE s.status = 'past_due'
      AND s.current_period_end <= p_grace_period_end
    ORDER BY s.current_period_end ASC
    LIMIT 100;
$$;

-- Grant execution to app role
GRANT EXECUTE ON FUNCTION list_subscriptions_due_for_renewal(TIMESTAMP WITH TIME ZONE) TO payd_app;
GRANT EXECUTE ON FUNCTION list_subscriptions_expired_trials(TIMESTAMP WITH TIME ZONE) TO payd_app;
GRANT EXECUTE ON FUNCTION list_subscriptions_past_due_pending_expiry(TIMESTAMP WITH TIME ZONE) TO payd_app;

COMMIT;
