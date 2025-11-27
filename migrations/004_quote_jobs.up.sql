-- Quote generation job queue for reliable async processing with retry support
CREATE TABLE IF NOT EXISTS quote_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,

    -- Job state
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, processing, completed, failed
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,

    -- Timing
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- When to next attempt
    started_at TIMESTAMPTZ,                           -- When processing started
    completed_at TIMESTAMPTZ,                         -- When job finished (success or final failure)

    -- Error tracking
    last_error TEXT,
    error_count INT NOT NULL DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Indexes for efficient job processing
CREATE INDEX IF NOT EXISTS idx_quote_jobs_status_scheduled ON quote_jobs(status, scheduled_at)
    WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_quote_jobs_call_id ON quote_jobs(call_id);
CREATE INDEX IF NOT EXISTS idx_quote_jobs_created_at ON quote_jobs(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_quote_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS trigger_quote_jobs_updated_at ON quote_jobs;
CREATE TRIGGER trigger_quote_jobs_updated_at
    BEFORE UPDATE ON quote_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_quote_jobs_updated_at();

-- Add quote_job_id to calls table to link to current/latest job
ALTER TABLE calls ADD COLUMN IF NOT EXISTS quote_job_id UUID REFERENCES quote_jobs(id);

COMMENT ON TABLE quote_jobs IS 'Queue for async quote generation with retry support';
COMMENT ON COLUMN quote_jobs.status IS 'Job status: pending, processing, completed, failed';
COMMENT ON COLUMN quote_jobs.scheduled_at IS 'When to next attempt processing (for retry backoff)';
COMMENT ON COLUMN quote_jobs.attempts IS 'Number of processing attempts made';
