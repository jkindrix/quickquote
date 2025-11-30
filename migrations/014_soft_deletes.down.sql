-- Rollback: 014_soft_deletes
-- Description: Remove soft delete columns from domain models
-- WARNING: This will permanently lose soft delete timestamps

-- Remove indexes first
DROP INDEX IF EXISTS idx_users_not_deleted;
DROP INDEX IF EXISTS idx_calls_not_deleted;
DROP INDEX IF EXISTS idx_knowledge_bases_not_deleted;
DROP INDEX IF EXISTS idx_personas_not_deleted;
DROP INDEX IF EXISTS idx_pathways_not_deleted;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_calls_deleted_at;

-- Remove deleted_at columns
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE calls DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE personas DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE pathways DROP COLUMN IF EXISTS deleted_at;
