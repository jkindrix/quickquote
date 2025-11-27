-- Migration: 003_provider_abstraction
-- Purpose: Abstract voice provider to support Bland, Vapi, Retell, and others
-- This migration renames bland_call_id to provider_call_id and adds a provider column

-- Rename bland_call_id to provider_call_id for provider-agnostic naming
ALTER TABLE calls RENAME COLUMN bland_call_id TO provider_call_id;

-- Add provider column to track which voice provider the call came from
-- Default to 'bland' for existing records
ALTER TABLE calls ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'bland';

-- Update index for the new column name
DROP INDEX IF EXISTS idx_calls_bland_call_id;
CREATE INDEX idx_calls_provider_call_id ON calls(provider_call_id);

-- Create index on provider for filtering
CREATE INDEX idx_calls_provider ON calls(provider);

-- Add comment explaining the abstraction
COMMENT ON COLUMN calls.provider_call_id IS 'Unique call ID from the voice provider (Bland, Vapi, Retell, etc.)';
COMMENT ON COLUMN calls.provider IS 'Voice provider type: bland, vapi, retell, livekit, custom';
