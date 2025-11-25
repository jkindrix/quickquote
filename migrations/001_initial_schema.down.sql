-- Rollback initial schema

DROP TRIGGER IF EXISTS update_calls_updated_at ON calls;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS calls;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
