-- Add new fields to users table
ALTER TABLE users
    ADD COLUMN first_name VARCHAR(255),
    ADD COLUMN last_name VARCHAR(255),
    ADD COLUMN avatar_url TEXT;

-- Create banks table
CREATE TABLE banks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    connector VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_banks_name ON banks(name);
CREATE INDEX idx_banks_connector ON banks(connector);

-- Add bank_id to accounts table
ALTER TABLE accounts
    ADD COLUMN bank_id UUID REFERENCES banks(id) ON DELETE SET NULL;

CREATE INDEX idx_accounts_bank_id ON accounts(bank_id);

-- Add provider_id to transactions table
ALTER TABLE transactions
    ADD COLUMN provider_id VARCHAR(255);

CREATE INDEX idx_transactions_provider_id ON transactions(provider_id);

-- Create credit_card_data table
CREATE TABLE credit_card_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    purchase_date DATE NOT NULL,
    installment_number INT NOT NULL,
    total_installments INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_installment_number CHECK (installment_number > 0),
    CONSTRAINT check_total_installments CHECK (total_installments > 0),
    CONSTRAINT check_installment_range CHECK (installment_number <= total_installments)
);

CREATE INDEX idx_credit_card_data_transaction_id ON credit_card_data(transaction_id);
CREATE INDEX idx_credit_card_data_purchase_date ON credit_card_data(purchase_date);

-- Create documents table (for Brazilian CPF/CNPJ)
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(10) NOT NULL CHECK (type IN ('cpf', 'cnpj')),
    number VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, type)
);

CREATE INDEX idx_documents_user_id ON documents(user_id);
CREATE INDEX idx_documents_number ON documents(number);

