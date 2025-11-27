-- Rollback session token rotation fields

DROP INDEX IF EXISTS idx_sessions_rotated_at;
DROP INDEX IF EXISTS idx_sessions_previous_token;
ALTER TABLE sessions DROP COLUMN IF EXISTS rotated_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS previous_token;
