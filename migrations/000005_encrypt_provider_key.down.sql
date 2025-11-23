-- Revert provider_key column back to VARCHAR(255)
-- Warning: This will truncate any encrypted values longer than 255 characters
ALTER TABLE users
    ALTER COLUMN provider_key TYPE VARCHAR(255);
