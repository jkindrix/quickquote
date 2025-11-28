-- Migration: 008_prompts
-- Description: Create prompts table for AI agent prompt/task configurations
-- Date: 2025-11-27

-- Prompts table stores AI agent configurations for voice calls
CREATE TABLE IF NOT EXISTS prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Basic info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- The main prompt/task text - instructions for the AI agent
    task TEXT NOT NULL,

    -- Voice settings
    voice VARCHAR(100) DEFAULT 'maya',
    language VARCHAR(20) DEFAULT 'en-US',

    -- Model and behavior
    model VARCHAR(50) DEFAULT 'base',
    temperature DECIMAL(3,2) DEFAULT 0.7,
    interruption_threshold INTEGER,
    max_duration INTEGER DEFAULT 30,

    -- Opening behavior
    first_sentence TEXT,
    wait_for_greeting BOOLEAN DEFAULT false,

    -- Transfer settings
    transfer_phone_number VARCHAR(20),
    transfer_list JSONB,

    -- Voicemail handling
    voicemail_action VARCHAR(50),
    voicemail_message TEXT,

    -- Recording and audio
    record BOOLEAN DEFAULT false,
    background_track VARCHAR(50),
    noise_cancellation BOOLEAN DEFAULT false,

    -- Knowledge and tools (arrays of IDs)
    knowledge_base_ids TEXT[],
    custom_tool_ids TEXT[],

    -- Post-call analysis
    summary_prompt TEXT,
    dispositions TEXT[],

    -- Organization
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT prompts_temperature_range CHECK (temperature >= 0 AND temperature <= 1),
    CONSTRAINT prompts_max_duration_positive CHECK (max_duration IS NULL OR max_duration > 0)
);

-- Partial unique index for non-deleted names
CREATE UNIQUE INDEX idx_prompts_name_unique ON prompts (name) WHERE deleted_at IS NULL;

-- Index for default prompt lookup
CREATE INDEX idx_prompts_default ON prompts (is_default) WHERE is_default = true AND is_active = true AND deleted_at IS NULL;

-- Index for active prompts listing
CREATE INDEX idx_prompts_active ON prompts (is_active, created_at DESC) WHERE deleted_at IS NULL;

-- Index for soft delete filtering
CREATE INDEX idx_prompts_deleted_at ON prompts (deleted_at) WHERE deleted_at IS NULL;

-- Add prompt_id to calls table to track which prompt was used
ALTER TABLE calls ADD COLUMN IF NOT EXISTS prompt_id UUID REFERENCES prompts(id);

-- Add initiated_by to track whether call was initiated via API or webhook
ALTER TABLE calls ADD COLUMN IF NOT EXISTS initiated_by VARCHAR(50);

-- Add bland_call_params to store the full parameters sent to Bland API
ALTER TABLE calls ADD COLUMN IF NOT EXISTS bland_call_params JSONB;

-- Index for calls by prompt
CREATE INDEX IF NOT EXISTS idx_calls_prompt_id ON calls (prompt_id) WHERE prompt_id IS NOT NULL;

-- Comment on table
COMMENT ON TABLE prompts IS 'AI agent prompt/task configurations for voice calls';
COMMENT ON COLUMN prompts.task IS 'The main prompt text - instructions telling the AI agent what to do';
COMMENT ON COLUMN prompts.voice IS 'Voice ID or preset name (maya, josh, etc.)';
COMMENT ON COLUMN prompts.temperature IS 'Controls randomness (0-1), higher = more creative';
COMMENT ON COLUMN prompts.transfer_list IS 'JSON object mapping departments to phone numbers';
COMMENT ON COLUMN prompts.knowledge_base_ids IS 'Array of Bland knowledge base vector IDs';
COMMENT ON COLUMN prompts.custom_tool_ids IS 'Array of Bland custom tool IDs';
