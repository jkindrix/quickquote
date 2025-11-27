-- Add session security fields for token rotation and activity tracking

-- Add last_active_at column to track session activity
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add IP address for session binding (optional security measure)
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);

-- Add user agent for security auditing
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS user_agent TEXT;

-- Create index on last_active_at for efficient cleanup queries
CREATE INDEX IF NOT EXISTS idx_sessions_last_active_at ON sessions(last_active_at);

-- Update existing sessions to have last_active_at set to created_at
UPDATE sessions SET last_active_at = created_at WHERE last_active_at IS NULL;
