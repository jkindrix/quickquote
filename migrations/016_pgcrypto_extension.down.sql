-- Dropping pgcrypto is optional; keep extension if other objects depend on it.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN
        DROP EXTENSION pgcrypto;
    END IF;
END $$;
