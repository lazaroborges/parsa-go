-- Add provider_key to users table
ALTER TABLE users
    ADD COLUMN provider_key VARCHAR(255);

CREATE INDEX idx_users_provider_key ON users(provider_key);