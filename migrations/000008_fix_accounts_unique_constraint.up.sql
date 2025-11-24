-- Drop the partial unique index that cannot be used with ON CONFLICT
DROP INDEX IF EXISTS idx_accounts_user_provider;

-- Create a full unique index (without WHERE clause) so it can be used in ON CONFLICT
CREATE UNIQUE INDEX idx_accounts_user_provider ON accounts(user_id, provider_id);

