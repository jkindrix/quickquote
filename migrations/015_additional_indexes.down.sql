-- Rollback: 015_additional_indexes
-- Description: Remove additional indexes

DROP INDEX IF EXISTS idx_calls_status_created_not_deleted;
DROP INDEX IF EXISTS idx_calls_from_number;
DROP INDEX IF EXISTS idx_users_email_not_deleted;
DROP INDEX IF EXISTS idx_sessions_expires_active;
DROP INDEX IF EXISTS idx_quote_jobs_ready;
DROP INDEX IF EXISTS idx_csrf_tokens_valid;
DROP INDEX IF EXISTS idx_knowledge_bases_active;
DROP INDEX IF EXISTS idx_personas_active;
DROP INDEX IF EXISTS idx_pathways_published;
DROP INDEX IF EXISTS idx_prompts_active_name;
