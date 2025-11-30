-- Migration: 015_additional_indexes
-- Description: Add additional indexes for common query patterns
-- Focus on composite indexes and filtered indexes for better query performance

-- Calls: Composite index for status + created_at with soft delete filter
-- Commonly used query: List calls by status ordered by created_at (excludes deleted)
CREATE INDEX IF NOT EXISTS idx_calls_status_created_not_deleted
ON calls(status, created_at DESC)
WHERE deleted_at IS NULL;

-- Calls: Index on from_number for lookup by caller
CREATE INDEX IF NOT EXISTS idx_calls_from_number
ON calls(from_number)
WHERE deleted_at IS NULL;

-- Users: Index for email lookup (case-insensitive) excluding deleted
CREATE INDEX IF NOT EXISTS idx_users_email_not_deleted
ON users(email)
WHERE deleted_at IS NULL;

-- Sessions: Composite index for session validation queries
-- The most common query is: SELECT ... WHERE expires_at > NOW() AND (token = $1 OR previous_token = $1)
CREATE INDEX IF NOT EXISTS idx_sessions_expires_active
ON sessions(expires_at);

-- Quote jobs: Index for pending jobs ready to process
CREATE INDEX IF NOT EXISTS idx_quote_jobs_ready
ON quote_jobs(scheduled_at ASC)
WHERE status = 'pending';

-- CSRF tokens: Composite index for valid token lookup
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_valid
ON csrf_tokens(token, expires_at, used);

-- Knowledge bases: Index for active knowledge bases
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_active
ON knowledge_bases(created_at DESC)
WHERE status = 'active' AND deleted_at IS NULL;

-- Personas: Index for active personas (not default filtered)
CREATE INDEX IF NOT EXISTS idx_personas_active
ON personas(created_at DESC)
WHERE status = 'active' AND deleted_at IS NULL;

-- Pathways: Index for published pathways
CREATE INDEX IF NOT EXISTS idx_pathways_published
ON pathways(created_at DESC)
WHERE is_published = true AND status = 'active' AND deleted_at IS NULL;

-- Prompts: Ensure we have good coverage for active prompt queries
CREATE INDEX IF NOT EXISTS idx_prompts_active_name
ON prompts(name)
WHERE is_active = true AND deleted_at IS NULL;

-- Add comments explaining index purposes
COMMENT ON INDEX idx_calls_status_created_not_deleted IS 'Optimizes listing calls by status (common dashboard query)';
COMMENT ON INDEX idx_calls_from_number IS 'Optimizes lookup by caller phone number';
COMMENT ON INDEX idx_users_email_not_deleted IS 'Optimizes login/email lookup excluding deleted users';
COMMENT ON INDEX idx_quote_jobs_ready IS 'Optimizes job processor finding ready jobs';
COMMENT ON INDEX idx_csrf_tokens_valid IS 'Optimizes CSRF validation';
