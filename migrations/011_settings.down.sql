-- Rollback settings table
DROP TRIGGER IF EXISTS settings_updated_at ON settings;
DROP FUNCTION IF EXISTS update_settings_updated_at();
DROP TABLE IF EXISTS settings;
