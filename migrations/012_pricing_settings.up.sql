-- Add pricing fallback settings
-- These are used when the Bland API pricing endpoint is unavailable

INSERT INTO settings (key, value, value_type, category, description) VALUES
    ('pricing_inbound_per_minute', '0.09', 'float', 'pricing', 'Inbound call cost per minute'),
    ('pricing_outbound_per_minute', '0.12', 'float', 'pricing', 'Outbound call cost per minute'),
    ('pricing_transcription_per_minute', '0.02', 'float', 'pricing', 'Transcription cost per minute'),
    ('pricing_analysis_per_call', '0.05', 'float', 'pricing', 'AI analysis cost per call'),
    ('pricing_phone_number_per_month', '2.00', 'float', 'pricing', 'Phone number monthly cost'),
    ('pricing_enhanced_model_premium', '0.02', 'float', 'pricing', 'Enhanced model additional cost per minute')
ON CONFLICT (key) DO NOTHING;
