-- Create CSRF tokens table for persistent CSRF protection

CREATE TABLE IF NOT EXISTS csrf_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(128) NOT NULL UNIQUE,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    used BOOLEAN NOT NULL DEFAULT FALSE
);

-- Index for fast token lookup
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_token ON csrf_tokens(token);

-- Index for session-based queries
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_session_id ON csrf_tokens(session_id) WHERE session_id IS NOT NULL;

-- Index for cleanup queries
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_expires_at ON csrf_tokens(expires_at);
