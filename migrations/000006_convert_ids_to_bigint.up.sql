-- Migration to convert all ID columns from UUID to BIGINT
-- This is a destructive migration that will drop all existing data

-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS credit_card_data CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS banks CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop UUID extension as we no longer need it
DROP EXTENSION IF EXISTS "uuid-ossp";

-- Recreate users table with BIGINT ID
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
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

-- Recreate banks table with BIGINT ID
CREATE TABLE banks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    connector VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_banks_name ON banks(name);
CREATE INDEX idx_banks_connector ON banks(connector);

-- Recreate accounts table with BIGINT ID and foreign keys
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    bank_id BIGINT REFERENCES banks(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_bank_id ON accounts(bank_id);

-- Recreate transactions table with BIGINT ID and foreign keys
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
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

-- Recreate credit_card_data table with BIGINT ID and foreign keys
CREATE TABLE credit_card_data (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    purchase_date DATE NOT NULL,
    installment_number INT NOT NULL,
    total_installments INT NOT NULL,
    CONSTRAINT check_installment_number CHECK (installment_number > 0),
    CONSTRAINT check_total_installments CHECK (total_installments > 0),
    CONSTRAINT check_installment_range CHECK (installment_number <= total_installments)
);

CREATE INDEX idx_credit_card_data_transaction_id ON credit_card_data(transaction_id);
CREATE INDEX idx_credit_card_data_purchase_date ON credit_card_data(purchase_date);

-- Recreate documents table with BIGINT ID
CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(10) NOT NULL CHECK (type IN ('cpf', 'cnpj')),
    number VARCHAR(20) NOT NULL UNIQUE
);

CREATE INDEX idx_documents_number ON documents(number);

-- Create cousins table (merchants/individuals)
CREATE TABLE cousins (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
    business_name VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cousins_document_id ON cousins(document_id);
CREATE INDEX idx_cousins_name ON cousins(name);

