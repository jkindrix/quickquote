-- Migration: 008_prompts (rollback)
-- Description: Remove prompts table and related columns

-- Remove indexes on calls table
DROP INDEX IF EXISTS idx_calls_prompt_id;

-- Remove columns from calls table
ALTER TABLE calls DROP COLUMN IF EXISTS bland_call_params;
ALTER TABLE calls DROP COLUMN IF EXISTS initiated_by;
ALTER TABLE calls DROP COLUMN IF EXISTS prompt_id;

-- Drop prompts table (this will cascade drop indexes)
DROP TABLE IF EXISTS prompts;
