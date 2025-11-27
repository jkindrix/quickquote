-- Rollback query optimization indexes

-- Sessions indexes
DROP INDEX IF EXISTS idx_sessions_token_expires;
DROP INDEX IF EXISTS idx_sessions_previous_token;
DROP INDEX IF EXISTS idx_sessions_active;

-- Calls indexes
DROP INDEX IF EXISTS idx_calls_provider_call_id;
DROP INDEX IF EXISTS idx_calls_status_created;
DROP INDEX IF EXISTS idx_calls_pending_quote;
DROP INDEX IF EXISTS idx_calls_provider;

-- Quote jobs indexes
DROP INDEX IF EXISTS idx_quote_jobs_processing;
DROP INDEX IF EXISTS idx_quote_jobs_failed;

-- CSRF tokens indexes
DROP INDEX IF EXISTS idx_csrf_tokens_session_valid;

-- Users indexes
DROP INDEX IF EXISTS idx_users_email_lower;
