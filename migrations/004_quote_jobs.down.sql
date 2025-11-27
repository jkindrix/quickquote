-- Rollback quote generation job queue

-- Remove foreign key from calls
ALTER TABLE calls DROP COLUMN IF EXISTS quote_job_id;

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_quote_jobs_updated_at ON quote_jobs;
DROP FUNCTION IF EXISTS update_quote_jobs_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_quote_jobs_status_scheduled;
DROP INDEX IF EXISTS idx_quote_jobs_call_id;
DROP INDEX IF EXISTS idx_quote_jobs_created_at;

-- Drop table
DROP TABLE IF EXISTS quote_jobs;
