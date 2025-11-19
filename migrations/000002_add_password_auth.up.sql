-- Add password authentication support
ALTER TABLE users
    ADD COLUMN password_hash VARCHAR(255),
    ALTER COLUMN oauth_provider DROP NOT NULL,
    ALTER COLUMN oauth_id DROP NOT NULL;

-- Add check constraint to ensure either OAuth or password is present
ALTER TABLE users ADD CONSTRAINT check_auth_method
    CHECK (
        (oauth_provider IS NOT NULL AND oauth_id IS NOT NULL) OR
        (password_hash IS NOT NULL)
    );
