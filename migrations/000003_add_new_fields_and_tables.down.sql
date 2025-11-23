-- Drop documents table
DROP TABLE IF EXISTS documents;

-- Drop credit_card_data table
DROP TABLE IF EXISTS credit_card_data;

-- Remove provider_id from transactions table
DROP INDEX IF EXISTS idx_transactions_provider_id;
ALTER TABLE transactions
    DROP COLUMN IF EXISTS provider_id;

-- Remove bank_id from accounts table
DROP INDEX IF EXISTS idx_accounts_bank_id;
ALTER TABLE accounts
    DROP COLUMN IF EXISTS bank_id;

-- Drop banks table
DROP TABLE IF EXISTS banks;

-- Remove new fields from users table
ALTER TABLE users
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;

