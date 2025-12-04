-- Parsa-Go Initial Schema

-- Users table
CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.users (
    id bigint NOT NULL DEFAULT nextval('public.users_id_seq'::regclass),
    email character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    first_name character varying(255),
    last_name character varying(255),
    avatar_url text,
    oauth_provider character varying(50),
    oauth_id character varying(255),
    password_hash character varying(255),
    provider_key text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_oauth_provider_oauth_id_key UNIQUE (oauth_provider, oauth_id),
    CONSTRAINT check_auth_method CHECK ((((oauth_provider IS NOT NULL) AND (oauth_id IS NOT NULL)) OR (password_hash IS NOT NULL)))
);

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

CREATE INDEX idx_users_email ON public.users USING btree (email);
CREATE INDEX idx_users_oauth ON public.users USING btree (oauth_provider, oauth_id);

-- Banks table
CREATE SEQUENCE public.banks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.banks (
    id bigint NOT NULL DEFAULT nextval('public.banks_id_seq'::regclass),
    name character varying(255) NOT NULL,
    connector character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    ui_name character varying(255),
    primary_color character varying(6) DEFAULT '1194F6'::character varying,
    CONSTRAINT banks_pkey PRIMARY KEY (id),
    CONSTRAINT banks_name_unique UNIQUE (name)
);

ALTER SEQUENCE public.banks_id_seq OWNED BY public.banks.id;

CREATE INDEX idx_banks_name ON public.banks USING btree (name);
CREATE INDEX idx_banks_connector ON public.banks USING btree (connector);

-- Items table: represents a connection/relationship with a financial institution
CREATE TABLE public.items (
    id character varying(255) NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT items_pkey PRIMARY KEY (id),
    CONSTRAINT items_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_items_user_id ON public.items USING btree (user_id);

-- Accounts table
CREATE TABLE public.accounts (
    id character varying(255) NOT NULL,
    user_id bigint NOT NULL,
    item_id character varying(255),
    name character varying(255) NOT NULL,
    account_type character varying(50) NOT NULL,
    subtype character varying(50),
    currency character varying(3) DEFAULT 'BRL'::character varying NOT NULL,
    balance numeric(15,2) DEFAULT 0.00 NOT NULL,
    bank_id bigint,
    provider_updated_at timestamp with time zone,
    provider_created_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    initial_balance numeric(15,2) DEFAULT 0.00 NOT NULL,
    is_open_finance_account boolean DEFAULT true NOT NULL,
    closed_at timestamp with time zone,
    "order" integer DEFAULT 90 NOT NULL,
    description text,
    removed boolean DEFAULT false NOT NULL,
    hidden_by_user boolean DEFAULT false NOT NULL,
    CONSTRAINT accounts_pkey PRIMARY KEY (id),
    CONSTRAINT accounts_subtype_check CHECK (((subtype)::text = ANY ((ARRAY['CHECKING_ACCOUNT'::character varying, 'SAVINGS_ACCOUNT'::character varying, 'CREDIT_CARD'::character varying])::text[]))),
    CONSTRAINT accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT accounts_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id) ON DELETE SET NULL,
    CONSTRAINT accounts_bank_id_fkey FOREIGN KEY (bank_id) REFERENCES public.banks(id) ON DELETE SET NULL
);

CREATE INDEX idx_accounts_user_id ON public.accounts USING btree (user_id);
CREATE INDEX idx_accounts_item_id ON public.accounts USING btree (item_id);
CREATE INDEX idx_accounts_bank_id ON public.accounts USING btree (bank_id);
CREATE INDEX idx_accounts_subtype ON public.accounts USING btree (subtype);
CREATE INDEX idx_accounts_match ON public.accounts USING btree (name, account_type, subtype);

-- Transactions table
CREATE TABLE public.transactions (
    id character varying(255) NOT NULL,
    account_id character varying(255) NOT NULL,
    amount numeric(15,2) NOT NULL,
    description text NOT NULL,
    category character varying(100),
    transaction_date timestamp with time zone NOT NULL,
    type character varying(20) DEFAULT 'DEBIT'::character varying NOT NULL,
    status character varying(20) DEFAULT 'POSTED'::character varying NOT NULL,
    provider_created_at timestamp with time zone,
    provider_updated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    considered boolean DEFAULT true NOT NULL,
    is_open_finance boolean DEFAULT true NOT NULL,
    tags text[],
    payment_method character varying(255),
    manipulated boolean DEFAULT false NOT NULL,
    cousin integer DEFAULT 0 NOT NULL,
    notes text,
    original_category character varying(100),
    original_description text,
    CONSTRAINT transactions_pkey PRIMARY KEY (id),
    CONSTRAINT check_transactions_type CHECK (((type)::text = ANY ((ARRAY['DEBIT'::character varying, 'CREDIT'::character varying])::text[]))),
    CONSTRAINT check_transactions_status CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'POSTED'::character varying])::text[]))),
    CONSTRAINT transactions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_transactions_account_id ON public.transactions USING btree (account_id);
CREATE INDEX idx_transactions_date ON public.transactions USING btree (transaction_date);
CREATE INDEX idx_transactions_category ON public.transactions USING btree (category);
CREATE INDEX idx_transactions_type ON public.transactions USING btree (type);
CREATE INDEX idx_transactions_status ON public.transactions USING btree (status);

-- Credit card data (installment info for credit card transactions)
CREATE SEQUENCE public.credit_card_data_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.credit_card_data (
    id bigint NOT NULL DEFAULT nextval('public.credit_card_data_id_seq'::regclass),
    transaction_id character varying(255) NOT NULL,
    purchase_date date NOT NULL,
    installment_number integer NOT NULL,
    total_installments integer NOT NULL,
    CONSTRAINT credit_card_data_pkey PRIMARY KEY (id),
    CONSTRAINT credit_card_data_transaction_id_key UNIQUE (transaction_id),
    CONSTRAINT check_installment_number CHECK ((installment_number > 0)),
    CONSTRAINT check_total_installments CHECK ((total_installments > 0)),
    CONSTRAINT check_installment_range CHECK ((installment_number <= total_installments)),
    CONSTRAINT credit_card_data_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id) ON DELETE CASCADE
);

ALTER SEQUENCE public.credit_card_data_id_seq OWNED BY public.credit_card_data.id;

CREATE INDEX idx_credit_card_data_transaction_id ON public.credit_card_data USING btree (transaction_id);
CREATE INDEX idx_credit_card_data_purchase_date ON public.credit_card_data USING btree (purchase_date);

-- Documents table (Brazilian CPF/CNPJ)
CREATE SEQUENCE public.documents_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.documents (
    id bigint NOT NULL DEFAULT nextval('public.documents_id_seq'::regclass),
    type character varying(10) NOT NULL,
    number character varying(20) NOT NULL,
    CONSTRAINT documents_pkey PRIMARY KEY (id),
    CONSTRAINT documents_number_key UNIQUE (number),
    CONSTRAINT documents_type_check CHECK (((type)::text = ANY ((ARRAY['cpf'::character varying, 'cnpj'::character varying])::text[])))
);

ALTER SEQUENCE public.documents_id_seq OWNED BY public.documents.id;

CREATE INDEX idx_documents_number ON public.documents USING btree (number);

-- Cousins table (merchants/individuals associated with transactions)
CREATE SEQUENCE public.cousins_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.cousins (
    id bigint NOT NULL DEFAULT nextval('public.cousins_id_seq'::regclass),
    document_id bigint,
    business_name character varying(255),
    name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT cousins_pkey PRIMARY KEY (id),
    CONSTRAINT cousins_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE SET NULL
);

ALTER SEQUENCE public.cousins_id_seq OWNED BY public.cousins.id;

CREATE INDEX idx_cousins_document_id ON public.cousins USING btree (document_id);
CREATE INDEX idx_cousins_name ON public.cousins USING btree (name);

