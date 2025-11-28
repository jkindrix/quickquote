-- Migration: 009_bland_entities (rollback)
-- Description: Drop tables for Bland entities

-- Drop triggers first
DROP TRIGGER IF EXISTS update_knowledge_bases_updated_at ON knowledge_bases;
DROP TRIGGER IF EXISTS update_knowledge_base_documents_updated_at ON knowledge_base_documents;
DROP TRIGGER IF EXISTS update_pathways_updated_at ON pathways;
DROP TRIGGER IF EXISTS update_personas_updated_at ON personas;
DROP TRIGGER IF EXISTS update_memory_stores_updated_at ON memory_stores;
DROP TRIGGER IF EXISTS update_batches_updated_at ON batches;
DROP TRIGGER IF EXISTS update_usage_records_updated_at ON usage_records;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS usage_alerts;
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS batch_calls;
DROP TABLE IF EXISTS batches;
DROP TABLE IF EXISTS memory_stores;
DROP TABLE IF EXISTS personas;
DROP TABLE IF EXISTS pathway_versions;
DROP TABLE IF EXISTS pathways;
DROP TABLE IF EXISTS knowledge_base_documents;
DROP TABLE IF EXISTS knowledge_bases;
