-- Add session token rotation fields for secure token invalidation

-- Add previous_token for grace period during rotation
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS previous_token VARCHAR(128);

-- Add rotated_at timestamp to track when token was last rotated
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMP WITH TIME ZONE;

-- Create index on previous_token for efficient lookup during grace period
CREATE INDEX IF NOT EXISTS idx_sessions_previous_token ON sessions(previous_token) WHERE previous_token IS NOT NULL;

-- Create index on rotated_at for efficient cleanup of expired grace periods
CREATE INDEX IF NOT EXISTS idx_sessions_rotated_at ON sessions(rotated_at) WHERE rotated_at IS NOT NULL;
