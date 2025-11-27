-- Rollback session security fields

DROP INDEX IF EXISTS idx_sessions_last_active_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_agent;
ALTER TABLE sessions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE sessions DROP COLUMN IF EXISTS last_active_at;
