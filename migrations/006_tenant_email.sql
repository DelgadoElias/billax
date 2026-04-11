-- Migration 006: Add email column to tenants table
-- Allows self-service signup with email as contact info

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS email TEXT;

-- Unique index on email for fast lookup and constraint enforcement
-- WHERE email IS NOT NULL allows multiple null emails (legacy tenants)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email) WHERE email IS NOT NULL;
