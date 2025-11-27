-- Rollback CSRF tokens table

DROP INDEX IF EXISTS idx_csrf_tokens_expires_at;
DROP INDEX IF EXISTS idx_csrf_tokens_session_id;
DROP INDEX IF EXISTS idx_csrf_tokens_token;
DROP TABLE IF EXISTS csrf_tokens;
