-- Rollback migration: 003_provider_abstraction
-- Revert to Bland-specific naming

-- Drop the new indexes
DROP INDEX IF EXISTS idx_calls_provider;
DROP INDEX IF EXISTS idx_calls_provider_call_id;

-- Remove the provider column
ALTER TABLE calls DROP COLUMN provider;

-- Rename back to bland_call_id
ALTER TABLE calls RENAME COLUMN provider_call_id TO bland_call_id;

-- Recreate the original index
CREATE INDEX idx_calls_bland_call_id ON calls(bland_call_id);
