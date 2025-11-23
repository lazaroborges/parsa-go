-- Remove provider_key from users table
DROP INDEX IF EXISTS idx_users_provider_key;

ALTER TABLE users
    DROP COLUMN provider_key;