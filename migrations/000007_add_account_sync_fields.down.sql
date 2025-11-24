-- Remove provider sync fields from accounts table
DROP INDEX IF EXISTS idx_accounts_subtype;
DROP INDEX IF EXISTS idx_accounts_user_provider;

ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_subtype_check,
    DROP COLUMN IF EXISTS provider_created_at,
    DROP COLUMN IF EXISTS provider_updated_at,
    DROP COLUMN IF EXISTS subtype,
    DROP COLUMN IF EXISTS provider_id;

