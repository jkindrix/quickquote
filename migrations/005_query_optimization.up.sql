-- Query optimization: Additional indexes and partial indexes for common query patterns

-- ============================================================================
-- Sessions Table Optimizations
-- ============================================================================

-- Composite index for token lookup with expiry check (most common session query)
-- Note: We include expires_at in the index for covering queries, not as a partial filter
-- Partial indexes with NOW() are evaluated at creation time, making them useless for time-based filtering
CREATE INDEX IF NOT EXISTS idx_sessions_token_expires ON sessions(token, expires_at);

-- Index for previous token lookup during rotation grace period
CREATE INDEX IF NOT EXISTS idx_sessions_previous_token ON sessions(previous_token)
    WHERE previous_token IS NOT NULL;

-- Index for active sessions queries (user_id + expires_at for efficient filtering)
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(user_id, expires_at DESC);

-- ============================================================================
-- Calls Table Optimizations
-- ============================================================================

-- Composite index for provider call ID lookup (webhook processing)
CREATE INDEX IF NOT EXISTS idx_calls_provider_call_id ON calls(provider_call_id);

-- Composite index for status + created_at (used in List with status filter)
CREATE INDEX IF NOT EXISTS idx_calls_status_created ON calls(status, created_at DESC);

-- Partial index for calls needing quote generation
CREATE INDEX IF NOT EXISTS idx_calls_pending_quote ON calls(created_at DESC)
    WHERE quote_summary IS NULL AND status = 'completed';

-- Index for provider-based queries
CREATE INDEX IF NOT EXISTS idx_calls_provider ON calls(provider);

-- ============================================================================
-- Quote Jobs Table Optimizations
-- ============================================================================

-- Partial index for processing jobs (stuck job detection)
CREATE INDEX IF NOT EXISTS idx_quote_jobs_processing ON quote_jobs(started_at)
    WHERE status = 'processing';

-- Partial index for failed jobs (retry/monitoring)
CREATE INDEX IF NOT EXISTS idx_quote_jobs_failed ON quote_jobs(completed_at DESC)
    WHERE status = 'failed';

-- ============================================================================
-- CSRF Tokens Table Optimizations
-- ============================================================================

-- Composite index for session token lookup with expiry and used check
-- Note: Can't use NOW() in partial index - include expires_at for efficient filtering at query time
-- Partial filter on NOT used and session_id IS NOT NULL (static conditions)
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_session_valid ON csrf_tokens(session_id, expires_at DESC, created_at DESC)
    WHERE NOT used AND session_id IS NOT NULL;

-- ============================================================================
-- Users Table Optimizations
-- ============================================================================

-- The existing idx_users_email is already optimal for email lookups
-- Adding a LOWER index for case-insensitive email searches (if needed in future)
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email));

-- ============================================================================
-- Statistics and Maintenance
-- ============================================================================

-- Update statistics for query planner
ANALYZE users;
ANALYZE sessions;
ANALYZE calls;
ANALYZE quote_jobs;
ANALYZE csrf_tokens;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON INDEX idx_sessions_token_expires IS 'Optimizes token validation queries that check expiry';
COMMENT ON INDEX idx_sessions_previous_token IS 'Optimizes lookups of rotated tokens during grace period';
COMMENT ON INDEX idx_sessions_active IS 'Optimizes queries for active session counts';
COMMENT ON INDEX idx_calls_status_created IS 'Optimizes paginated call lists with status filter';
COMMENT ON INDEX idx_calls_pending_quote IS 'Optimizes finding calls that need quote generation';
COMMENT ON INDEX idx_quote_jobs_processing IS 'Optimizes detection of stuck jobs';
COMMENT ON INDEX idx_quote_jobs_failed IS 'Optimizes monitoring of failed jobs';
COMMENT ON INDEX idx_csrf_tokens_session_valid IS 'Optimizes GetOrCreate token lookups';
