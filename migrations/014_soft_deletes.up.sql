-- Migration: 014_soft_deletes
-- Description: Add soft delete support to domain models
-- This adds deleted_at columns to enable soft deletes instead of hard deletes

-- Add deleted_at to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add deleted_at to calls table (if not already present)
ALTER TABLE calls ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add deleted_at to knowledge_bases table
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add deleted_at to personas table
ALTER TABLE personas ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add deleted_at to pathways table
ALTER TABLE pathways ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add indexes for efficient soft delete queries (filter out deleted records)
-- These partial indexes only index non-deleted records, making queries efficient
CREATE INDEX IF NOT EXISTS idx_users_not_deleted ON users(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_calls_not_deleted ON calls(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_not_deleted ON knowledge_bases(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_personas_not_deleted ON personas(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pathways_not_deleted ON pathways(id) WHERE deleted_at IS NULL;

-- Add index for querying deleted records (for admin/audit purposes)
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_calls_deleted_at ON calls(deleted_at) WHERE deleted_at IS NOT NULL;

-- Add comments explaining soft delete behavior
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp - NULL means active, non-NULL means deleted';
COMMENT ON COLUMN calls.deleted_at IS 'Soft delete timestamp - NULL means active, non-NULL means deleted';
COMMENT ON COLUMN knowledge_bases.deleted_at IS 'Soft delete timestamp - NULL means active, non-NULL means deleted';
COMMENT ON COLUMN personas.deleted_at IS 'Soft delete timestamp - NULL means active, non-NULL means deleted';
COMMENT ON COLUMN pathways.deleted_at IS 'Soft delete timestamp - NULL means active, non-NULL means deleted';
