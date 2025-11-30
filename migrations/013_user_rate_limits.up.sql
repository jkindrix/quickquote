-- Migration 013: User Rate Limits
-- Provides per-user API rate limiting with sliding window support

CREATE TABLE IF NOT EXISTS user_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    window_type VARCHAR(10) NOT NULL CHECK (window_type IN ('minute', 'hour', 'day')),
    request_count INTEGER NOT NULL DEFAULT 0,
    window_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each user can only have one record per window type
    UNIQUE (user_id, window_type)
);

-- Index for efficient window expiration cleanup
CREATE INDEX idx_user_rate_limits_window_end ON user_rate_limits(window_end);

-- Index for user lookups
CREATE INDEX idx_user_rate_limits_user_id ON user_rate_limits(user_id);

-- Comment on table
COMMENT ON TABLE user_rate_limits IS 'Tracks per-user API rate limits across minute, hour, and day windows';
COMMENT ON COLUMN user_rate_limits.window_type IS 'The time window type: minute, hour, or day';
COMMENT ON COLUMN user_rate_limits.request_count IS 'Number of requests made in the current window';
COMMENT ON COLUMN user_rate_limits.window_end IS 'When the current window expires and count resets';
