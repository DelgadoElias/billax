-- Migration 010: Backoffice users table for UI authentication
-- Problem: Need email+password login for backoffice UI, separate from API key auth
-- Solution: Create backoffice_users table with bcrypt hashes and role-based access
--
-- Design:
-- - Each tenant can have multiple backoffice users (admin + members)
-- - Admin (role='admin') can invite/manage other users
-- - Members (role='member') can only access their own profile
-- - Passwords stored as bcrypt hashes (never plaintext)
-- - JWT tokens generated on login with 24h TTL
-- - RLS enforced per tenant (tenant_isolation policy)

BEGIN;

CREATE TABLE backoffice_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,  -- bcrypt hash (60 chars)
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    name TEXT NOT NULL DEFAULT '',
    must_change_password BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique email per tenant (cannot have duplicate emails across tenants due to RLS)
CREATE UNIQUE INDEX idx_backoffice_users_email_tenant
    ON backoffice_users(tenant_id, email);

-- Enable RLS and set tenant isolation policy
ALTER TABLE backoffice_users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON backoffice_users
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Grant permissions to app role
GRANT SELECT, INSERT, UPDATE, DELETE ON backoffice_users TO payd_app;

COMMIT;
