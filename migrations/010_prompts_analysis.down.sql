-- Rollback: 010_prompts_analysis

ALTER TABLE prompts DROP COLUMN IF EXISTS analysis_schema;
ALTER TABLE prompts DROP COLUMN IF EXISTS keywords;
