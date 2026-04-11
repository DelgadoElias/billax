-- Migration 007: Add default provider configuration to tenants table

-- Add default_provider_name column to tenants
-- Defaults to 'mercadopago' for backward compatibility
-- Allows tenants to configure their preferred payment provider
ALTER TABLE tenants
ADD COLUMN default_provider_name TEXT NOT NULL DEFAULT 'mercadopago'
  CONSTRAINT valid_default_provider CHECK (default_provider_name IN ('mercadopago', 'stripe', 'helipagos'));

-- Index for fast lookups by default provider
CREATE INDEX idx_tenants_default_provider ON tenants(default_provider_name);
