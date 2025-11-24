-- Add provider sync fields to accounts table
ALTER TABLE accounts
    ADD COLUMN provider_id VARCHAR(255),
    ADD COLUMN subtype VARCHAR(50) CHECK (subtype IN ('CHECKING_ACCOUNT', 'SAVINGS_ACCOUNT', 'CREDIT_CARD', 'PAYMENT_ACCOUNT')),
    ADD COLUMN provider_updated_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN provider_created_at TIMESTAMP WITH TIME ZONE;

-- Create unique constraint for provider_id per user (for upsert)
-- Note: Using a full unique index (without WHERE clause) so it can be used in ON CONFLICT
CREATE UNIQUE INDEX idx_accounts_user_provider ON accounts(user_id, provider_id);

-- Create index for subtype filtering
CREATE INDEX idx_accounts_subtype ON accounts(subtype);

