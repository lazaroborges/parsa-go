-- Migration 000006: Merchants table, documents.business_name, transaction counterparty FKs

-- 1. New merchants table
CREATE SEQUENCE public.merchants_id_seq
    START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;

CREATE TABLE public.merchants (
    id bigint NOT NULL DEFAULT nextval('public.merchants_id_seq'::regclass),
    name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT merchants_pkey PRIMARY KEY (id)
);

ALTER SEQUENCE public.merchants_id_seq OWNED BY public.merchants.id;

CREATE UNIQUE INDEX idx_merchants_name_lower ON public.merchants USING btree (LOWER(name));

-- 2. Documents: add business_name, make number nullable
ALTER TABLE public.documents ADD COLUMN business_name character varying(255);

ALTER TABLE public.documents ALTER COLUMN number DROP NOT NULL;

-- Partial unique indexes: number unique when set, business_name unique when set
ALTER TABLE public.documents DROP CONSTRAINT IF EXISTS documents_number_key;
CREATE UNIQUE INDEX idx_documents_number_unique ON public.documents (number) WHERE number IS NOT NULL;

CREATE UNIQUE INDEX idx_documents_business_name_lower ON public.documents USING btree (LOWER(business_name)) WHERE business_name IS NOT NULL;

-- 3. Transactions: add merchant_id and document_id FKs
ALTER TABLE public.transactions ADD COLUMN merchant_id bigint;
ALTER TABLE public.transactions ADD COLUMN document_id bigint;

ALTER TABLE public.transactions
    ADD CONSTRAINT transactions_merchant_id_fkey
    FOREIGN KEY (merchant_id) REFERENCES public.merchants(id) ON DELETE SET NULL;

ALTER TABLE public.transactions
    ADD CONSTRAINT transactions_document_id_fkey
    FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_merchant_id ON public.transactions USING btree (merchant_id);
CREATE INDEX idx_transactions_document_id ON public.transactions USING btree (document_id);

-- 4. Cousins: add merchant_id FK
ALTER TABLE public.cousins ADD COLUMN merchant_id bigint;

ALTER TABLE public.cousins
    ADD CONSTRAINT cousins_merchant_id_fkey
    FOREIGN KEY (merchant_id) REFERENCES public.merchants(id) ON DELETE SET NULL;

CREATE INDEX idx_cousins_merchant_id ON public.cousins USING btree (merchant_id);
