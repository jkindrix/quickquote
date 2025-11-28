-- Migration: 010_prompts_analysis
-- Description: Add analysis_schema and keywords fields to prompts table

-- Analysis schema for structured data extraction from calls
ALTER TABLE prompts ADD COLUMN IF NOT EXISTS analysis_schema JSONB;

-- Keywords to boost transcription accuracy
ALTER TABLE prompts ADD COLUMN IF NOT EXISTS keywords TEXT[];

-- Comments
COMMENT ON COLUMN prompts.analysis_schema IS 'JSON schema defining structured data to extract from calls';
COMMENT ON COLUMN prompts.keywords IS 'Keywords to boost transcription accuracy for domain-specific terms';
