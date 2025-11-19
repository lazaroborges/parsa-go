-- Remove password authentication support
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS check_auth_method,
    DROP COLUMN IF EXISTS password_hash,
    ALTER COLUMN oauth_provider SET NOT NULL,
    ALTER COLUMN oauth_id SET NOT NULL;
