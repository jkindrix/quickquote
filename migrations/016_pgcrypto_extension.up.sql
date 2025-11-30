-- Ensure pgcrypto is available for gen_random_uuid() usage
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
