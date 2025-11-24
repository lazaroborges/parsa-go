-- Revert to partial unique index
DROP INDEX IF EXISTS idx_accounts_user_provider;

-- Recreate the partial index with WHERE clause
CREATE UNIQUE INDEX idx_accounts_user_provider ON accounts(user_id, provider_id) WHERE provider_id IS NOT NULL;

