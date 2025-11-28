-- Settings table for persisting application configuration
-- This replaces hardcoded defaults and allows runtime configuration changes

CREATE TABLE IF NOT EXISTS settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    value_type VARCHAR(50) NOT NULL DEFAULT 'string', -- string, int, float, bool, json
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookups by key and category
CREATE INDEX idx_settings_key ON settings(key);
CREATE INDEX idx_settings_category ON settings(category);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER settings_updated_at
    BEFORE UPDATE ON settings
    FOR EACH ROW
    EXECUTE FUNCTION update_settings_updated_at();

-- Insert default settings (these become the source of truth, not code)
INSERT INTO settings (key, value, value_type, category, description) VALUES
    -- Business Identity
    ('business_name', 'QuickQuote', 'string', 'business', 'Company name shown to callers'),
    ('project_types', 'web_app,mobile_app,api,ecommerce,custom_software,integration', 'string', 'business', 'Comma-separated list of project types offered'),

    -- Voice Configuration
    ('voice', 'maya', 'string', 'voice', 'Voice ID for AI agent'),
    ('voice_stability', '0.75', 'float', 'voice', 'Voice stability (0-1)'),
    ('voice_similarity_boost', '0.80', 'float', 'voice', 'Voice similarity/clarity (0-1)'),
    ('voice_style', '0.3', 'float', 'voice', 'Voice expressiveness (0-1)'),
    ('voice_speaker_boost', 'true', 'bool', 'voice', 'Enhanced speaker clarity'),

    -- AI Model Settings
    ('model', 'enhanced', 'string', 'ai', 'AI model: base or enhanced'),
    ('language', 'en-US', 'string', 'ai', 'Language code'),
    ('temperature', '0.6', 'float', 'ai', 'AI temperature (0-1)'),
    ('interruption_threshold', '100', 'int', 'ai', 'Response delay in ms (50-500)'),

    -- Call Behavior
    ('wait_for_greeting', 'true', 'bool', 'call', 'Wait for caller to speak first'),
    ('noise_cancellation', 'true', 'bool', 'call', 'Enable noise filtering'),
    ('background_track', 'office', 'string', 'call', 'Background audio: none, office, cafe, restaurant'),
    ('max_duration_minutes', '15', 'int', 'call', 'Max call duration in minutes'),
    ('record_calls', 'true', 'bool', 'call', 'Record calls for review'),
    ('quality_preset', 'default', 'string', 'call', 'Quality preset: default, high_quality, fast_response, accessibility'),

    -- Custom Greeting
    ('custom_greeting', '', 'string', 'call', 'Custom opening message (empty for default)')
ON CONFLICT (key) DO NOTHING;
