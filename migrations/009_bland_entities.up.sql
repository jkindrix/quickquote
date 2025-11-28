-- Migration: 009_bland_entities
-- Description: Create tables for local caching of Bland entities (knowledge bases, pathways, personas)

-- Knowledge Bases table
CREATE TABLE IF NOT EXISTS knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bland_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    vector_db_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    document_count INTEGER NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    sync_error TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_bland_id ON knowledge_bases(bland_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_status ON knowledge_bases(status);
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_name ON knowledge_bases(name);

-- Knowledge Base Documents table
CREATE TABLE IF NOT EXISTS knowledge_base_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    bland_doc_id VARCHAR(255),
    name VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    content_hash VARCHAR(64),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    chunk_count INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kb_documents_knowledge_base_id ON knowledge_base_documents(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_kb_documents_status ON knowledge_base_documents(status);
CREATE INDEX IF NOT EXISTS idx_kb_documents_bland_doc_id ON knowledge_base_documents(bland_doc_id);

-- Pathways table
CREATE TABLE IF NOT EXISTS pathways (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bland_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    nodes JSONB NOT NULL DEFAULT '[]',
    edges JSONB NOT NULL DEFAULT '[]',
    start_node_id VARCHAR(255),
    last_synced_at TIMESTAMP WITH TIME ZONE,
    sync_error TEXT,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pathways_bland_id ON pathways(bland_id);
CREATE INDEX IF NOT EXISTS idx_pathways_status ON pathways(status);
CREATE INDEX IF NOT EXISTS idx_pathways_name ON pathways(name);
CREATE INDEX IF NOT EXISTS idx_pathways_is_published ON pathways(is_published);

-- Pathway Versions table (for version history)
CREATE TABLE IF NOT EXISTS pathway_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pathway_id UUID NOT NULL REFERENCES pathways(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    nodes JSONB NOT NULL,
    edges JSONB NOT NULL,
    change_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    UNIQUE(pathway_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pathway_versions_pathway_id ON pathway_versions(pathway_id);

-- Personas table
CREATE TABLE IF NOT EXISTS personas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bland_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    voice VARCHAR(100),
    language VARCHAR(20) DEFAULT 'en-US',
    voice_settings JSONB DEFAULT '{}',
    personality TEXT,
    background_story TEXT,
    system_prompt TEXT,
    behavior JSONB DEFAULT '{}',
    knowledge_bases JSONB DEFAULT '[]',
    tools JSONB DEFAULT '[]',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    sync_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_personas_bland_id ON personas(bland_id);
CREATE INDEX IF NOT EXISTS idx_personas_status ON personas(status);
CREATE INDEX IF NOT EXISTS idx_personas_name ON personas(name);
CREATE INDEX IF NOT EXISTS idx_personas_is_default ON personas(is_default);

-- Memory Stores table (for local caching of Bland memory)
CREATE TABLE IF NOT EXISTS memory_stores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(50) NOT NULL,
    bland_memory_id VARCHAR(255),
    customer_name VARCHAR(255),
    context_data JSONB DEFAULT '{}',
    interaction_count INTEGER NOT NULL DEFAULT 0,
    last_interaction_at TIMESTAMP WITH TIME ZONE,
    tags JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_memory_stores_phone_number ON memory_stores(phone_number);
CREATE INDEX IF NOT EXISTS idx_memory_stores_bland_memory_id ON memory_stores(bland_memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_stores_customer_name ON memory_stores(customer_name);

-- Batches table (for tracking batch call campaigns)
CREATE TABLE IF NOT EXISTS batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bland_batch_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_calls INTEGER NOT NULL DEFAULT 0,
    completed_calls INTEGER NOT NULL DEFAULT 0,
    failed_calls INTEGER NOT NULL DEFAULT 0,
    success_rate DECIMAL(5,2),
    campaign_config JSONB DEFAULT '{}',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_batches_bland_batch_id ON batches(bland_batch_id);
CREATE INDEX IF NOT EXISTS idx_batches_status ON batches(status);
CREATE INDEX IF NOT EXISTS idx_batches_name ON batches(name);

-- Batch Calls table (for tracking individual calls in a batch)
CREATE TABLE IF NOT EXISTS batch_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    call_id UUID REFERENCES calls(id),
    bland_call_id VARCHAR(255),
    phone_number VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempt_number INTEGER NOT NULL DEFAULT 1,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    outcome VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_batch_calls_batch_id ON batch_calls(batch_id);
CREATE INDEX IF NOT EXISTS idx_batch_calls_status ON batch_calls(status);
CREATE INDEX IF NOT EXISTS idx_batch_calls_phone_number ON batch_calls(phone_number);
CREATE INDEX IF NOT EXISTS idx_batch_calls_bland_call_id ON batch_calls(bland_call_id);

-- Usage Tracking table
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_date DATE NOT NULL,
    call_count INTEGER NOT NULL DEFAULT 0,
    total_minutes DECIMAL(10,2) NOT NULL DEFAULT 0,
    call_cost DECIMAL(10,4) NOT NULL DEFAULT 0,
    sms_count INTEGER NOT NULL DEFAULT 0,
    sms_cost DECIMAL(10,4) NOT NULL DEFAULT 0,
    transcription_cost DECIMAL(10,4) NOT NULL DEFAULT 0,
    analysis_cost DECIMAL(10,4) NOT NULL DEFAULT 0,
    total_cost DECIMAL(10,4) NOT NULL DEFAULT 0,
    api_requests INTEGER NOT NULL DEFAULT 0,
    api_errors INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_date ON usage_records(record_date);

-- Usage Alerts table
CREATE TABLE IF NOT EXISTS usage_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_type VARCHAR(50) NOT NULL,
    threshold DECIMAL(10,4) NOT NULL,
    threshold_type VARCHAR(20) NOT NULL, -- percentage, absolute
    current_value DECIMAL(10,4),
    message TEXT NOT NULL,
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_usage_alerts_alert_type ON usage_alerts(alert_type);
CREATE INDEX IF NOT EXISTS idx_usage_alerts_acknowledged ON usage_alerts(acknowledged);

-- Create triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to all new tables
DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOR table_name IN
        SELECT unnest(ARRAY['knowledge_bases', 'knowledge_base_documents', 'pathways',
                           'personas', 'memory_stores', 'batches', 'usage_records'])
    LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
            CREATE TRIGGER update_%s_updated_at
                BEFORE UPDATE ON %s
                FOR EACH ROW
                EXECUTE FUNCTION update_updated_at_column();
        ', table_name, table_name, table_name, table_name);
    END LOOP;
END;
$$;

-- Add comments for documentation
COMMENT ON TABLE knowledge_bases IS 'Local cache of Bland AI knowledge bases';
COMMENT ON TABLE knowledge_base_documents IS 'Documents within knowledge bases';
COMMENT ON TABLE pathways IS 'Local cache of Bland AI conversational pathways';
COMMENT ON TABLE pathway_versions IS 'Version history for pathways';
COMMENT ON TABLE personas IS 'Local cache of Bland AI personas';
COMMENT ON TABLE memory_stores IS 'Local cache of customer memory/context';
COMMENT ON TABLE batches IS 'Batch call campaigns';
COMMENT ON TABLE batch_calls IS 'Individual calls within a batch campaign';
COMMENT ON TABLE usage_records IS 'Daily usage and cost tracking';
COMMENT ON TABLE usage_alerts IS 'Usage threshold alerts';
