-- Parsa-Go Initial Schema

-- Users table
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
    provider_key TEXT,  -- Encrypted API key for Open Finance provider
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(oauth_provider, oauth_id),
    CONSTRAINT check_auth_method CHECK (
        (oauth_provider IS NOT NULL AND oauth_id IS NOT NULL) OR
        (password_hash IS NOT NULL)
    )
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id);

-- Banks table (dormant until Pierre fixes their API)
CREATE TABLE banks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    connector VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_banks_name ON banks(name);
CREATE INDEX idx_banks_connector ON banks(connector);

-- Items table: represents a connection/relationship with a financial institution
-- One Item can have multiple Accounts (e.g., checking + credit card from same bank)
CREATE TABLE items (
    id VARCHAR(255) PRIMARY KEY,  -- Provider's itemId (UUID string)
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_items_user_id ON items(user_id);

-- Accounts table
-- Primary key is the provider's account ID (string UUID)
CREATE TABLE accounts (
    id VARCHAR(255) PRIMARY KEY,  -- Provider's account id
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id VARCHAR(255) REFERENCES items(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    subtype VARCHAR(50) CHECK (subtype IN ('CHECKING_ACCOUNT', 'SAVINGS_ACCOUNT', 'CREDIT_CARD', 'PAYMENT_ACCOUNT')),
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    bank_id BIGINT REFERENCES banks(id) ON DELETE SET NULL,
    provider_updated_at TIMESTAMP WITH TIME ZONE,
    provider_created_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_item_id ON accounts(item_id);
CREATE INDEX idx_accounts_bank_id ON accounts(bank_id);
CREATE INDEX idx_accounts_subtype ON accounts(subtype);
-- Index for transaction sync account matching (name, account_type, subtype)
CREATE INDEX idx_accounts_match ON accounts(name, account_type, subtype);

-- Transactions table
-- Primary key is the provider's transaction ID (string UUID)
CREATE TABLE transactions (
    id VARCHAR(255) PRIMARY KEY,  -- Provider's transaction id
    account_id VARCHAR(255) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount DECIMAL(15, 2) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(100),
    transaction_date TIMESTAMP WITH TIME ZONE NOT NULL,  -- Full timestamp from API
    type VARCHAR(20) NOT NULL DEFAULT 'DEBIT',
    status VARCHAR(20) NOT NULL DEFAULT 'POSTED',
    provider_created_at TIMESTAMP WITH TIME ZONE, -- NULL for now. 
    provider_updated_at TIMESTAMP WITH TIME ZONE, -- NULL for now.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_transactions_type CHECK (type IN ('DEBIT', 'CREDIT')),
    CONSTRAINT check_transactions_status CHECK (status IN ('PENDING', 'POSTED'))
);

CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
CREATE INDEX idx_transactions_category ON transactions(category);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_status ON transactions(status);

-- Credit card data (installment info for credit card transactions)
CREATE TABLE credit_card_data (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(255) NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
    purchase_date DATE NOT NULL,
    installment_number INT NOT NULL,
    total_installments INT NOT NULL,
    CONSTRAINT check_installment_number CHECK (installment_number > 0),
    CONSTRAINT check_total_installments CHECK (total_installments > 0),
    CONSTRAINT check_installment_range CHECK (installment_number <= total_installments)
);

CREATE INDEX idx_credit_card_data_transaction_id ON credit_card_data(transaction_id);
CREATE INDEX idx_credit_card_data_purchase_date ON credit_card_data(purchase_date);

-- Documents table (Brazilian CPF/CNPJ)
CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(10) NOT NULL CHECK (type IN ('cpf', 'cnpj')),
    number VARCHAR(20) NOT NULL UNIQUE
);

CREATE INDEX idx_documents_number ON documents(number);

-- Cousins table (merchants/individuals associated with transactions)
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
