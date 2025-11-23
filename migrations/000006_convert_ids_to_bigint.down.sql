-- Rollback migration: Convert BIGINT IDs back to UUID
-- This is a destructive rollback that will drop all existing data

-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS cousins CASCADE;
DROP TABLE IF EXISTS credit_card_data CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS banks CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Re-enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Recreate users table with UUID ID (original schema + password auth)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    avatar_url TEXT,
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    password_hash VARCHAR(255),
    provider_key TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(oauth_provider, oauth_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id);

-- Recreate banks table with UUID ID
CREATE TABLE banks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    connector VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_banks_name ON banks(name);
CREATE INDEX idx_banks_connector ON banks(connector);

-- Recreate accounts table with UUID ID and foreign keys
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    bank_id UUID REFERENCES banks(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_bank_id ON accounts(bank_id);

-- Recreate transactions table with UUID ID and foreign keys
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider_id VARCHAR(255),
    amount DECIMAL(15, 2) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(100),
    transaction_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
CREATE INDEX idx_transactions_category ON transactions(category);
CREATE INDEX idx_transactions_provider_id ON transactions(provider_id);

-- Recreate credit_card_data table with UUID ID and foreign keys
CREATE TABLE credit_card_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    purchase_date DATE NOT NULL,
    installment_number INT NOT NULL,
    total_installments INT NOT NULL,
    CONSTRAINT check_installment_number CHECK (installment_number > 0),
    CONSTRAINT check_total_installments CHECK (total_installments > 0),
    CONSTRAINT check_installment_range CHECK (installment_number <= total_installments)
);

CREATE INDEX idx_credit_card_data_transaction_id ON credit_card_data(transaction_id);
CREATE INDEX idx_credit_card_data_purchase_date ON credit_card_data(purchase_date);

-- Recreate documents table with UUID ID
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(10) NOT NULL CHECK (type IN ('cpf', 'cnpj')),
    number VARCHAR(20) NOT NULL UNIQUE
);

CREATE INDEX idx_documents_number ON documents(number);

