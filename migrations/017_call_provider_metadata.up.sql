-- Add provider summary/disposition fields for richer call context
ALTER TABLE calls
    ADD COLUMN IF NOT EXISTS provider_summary TEXT,
    ADD COLUMN IF NOT EXISTS provider_disposition TEXT,
    ADD COLUMN IF NOT EXISTS provider_metadata JSONB DEFAULT '{}'::jsonb;

-- Backfill default metadata for existing rows
UPDATE calls
SET provider_metadata = '{}'::jsonb
WHERE provider_metadata IS NULL;
