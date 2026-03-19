-- Migration 000007: Recurrency patterns, forecast transactions, and transaction recurrency columns

-- 1. Recurrency patterns table
CREATE SEQUENCE public.recurrency_patterns_id_seq
    START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;

CREATE TABLE public.recurrency_patterns (
    id bigint NOT NULL DEFAULT nextval('public.recurrency_patterns_id_seq'::regclass),
    user_id bigint NOT NULL,
    type character varying(20) NOT NULL,
    recurrency_type character varying(30) NOT NULL,
    cousin_id bigint,
    category character varying(100),
    avg_amount numeric(15,2),
    median_amount numeric(15,2),
    std_amount numeric(15,2),
    avg_day_of_month integer,
    occurrence_count integer,
    months_active integer,
    first_seen timestamp with time zone,
    last_seen timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    is_active boolean DEFAULT true,
    robust_cv numeric(8,4),
    circular_variance numeric(8,4),
    rayleigh_p numeric(10,6),
    month_coverage numeric(8,4),
    span_months integer,
    distinct_days integer,
    regime_start timestamp with time zone,
    regime_median numeric(15,2),
    regime_count integer,
    amount_trend numeric(10,6),
    parent_category character varying(100),
    CONSTRAINT recurrency_patterns_pkey PRIMARY KEY (id),
    CONSTRAINT check_identity CHECK ((cousin_id IS NOT NULL) OR (category IS NOT NULL) OR (parent_category IS NOT NULL)),
    CONSTRAINT check_recurrency_type CHECK ((recurrency_type)::text = ANY ((ARRAY['recurrent_fixed', 'recurrent_variable', 'irregular'])::text[])),
    CONSTRAINT check_type CHECK ((type)::text = ANY ((ARRAY['DEBIT', 'CREDIT'])::text[]))
);

ALTER SEQUENCE public.recurrency_patterns_id_seq OWNED BY public.recurrency_patterns.id;

ALTER TABLE public.recurrency_patterns
    ADD CONSTRAINT recurrency_patterns_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE public.recurrency_patterns
    ADD CONSTRAINT recurrency_patterns_cousin_id_fkey
    FOREIGN KEY (cousin_id) REFERENCES public.cousins(id) ON DELETE CASCADE;

CREATE INDEX idx_recurrency_user_id ON public.recurrency_patterns USING btree (user_id);

-- Unique indexes for recurrency pattern identity
CREATE UNIQUE INDEX uq_recurrency_user_type_cousin_category
    ON public.recurrency_patterns USING btree (user_id, type, cousin_id, category)
    WHERE (cousin_id IS NOT NULL) AND (category IS NOT NULL);

CREATE UNIQUE INDEX uq_recurrency_user_type_cousin_only
    ON public.recurrency_patterns USING btree (user_id, type, cousin_id)
    WHERE (cousin_id IS NOT NULL) AND (category IS NULL);

CREATE UNIQUE INDEX uq_recurrency_user_type_category_only
    ON public.recurrency_patterns USING btree (user_id, type, category)
    WHERE (cousin_id IS NULL) AND (category IS NOT NULL);

CREATE UNIQUE INDEX uq_recurrency_user_type_parent_category
    ON public.recurrency_patterns USING btree (user_id, type, parent_category)
    WHERE (parent_category IS NOT NULL) AND (cousin_id IS NULL) AND (category IS NULL);

-- 2. Forecast transactions table
CREATE SEQUENCE public.forecast_transactions_id_seq
    START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;

CREATE TABLE public.forecast_transactions (
    id bigint NOT NULL DEFAULT nextval('public.forecast_transactions_id_seq'::regclass),
    user_id bigint NOT NULL,
    recurrency_pattern_id bigint,
    type character varying(20) NOT NULL,
    recurrency_type character varying(30) NOT NULL,
    forecast_amount numeric(15,2) NOT NULL,
    forecast_low numeric(15,2),
    forecast_high numeric(15,2),
    forecast_date date,
    forecast_month date NOT NULL,
    cousin_id bigint,
    category character varying(100),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    parent_category character varying(100),
    uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    description text,
    cousin_name character varying(255),
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    generated_at timestamp with time zone,
    account_id character varying(255),
    CONSTRAINT forecast_transactions_pkey PRIMARY KEY (id),
    CONSTRAINT uq_forecast_uuid UNIQUE (uuid),
    CONSTRAINT check_forecast_recurrency_type CHECK ((recurrency_type)::text = ANY ((ARRAY['recurrent_fixed', 'recurrent_variable', 'irregular'])::text[])),
    CONSTRAINT check_forecast_type CHECK ((type)::text = ANY ((ARRAY['DEBIT', 'CREDIT'])::text[]))
);

ALTER SEQUENCE public.forecast_transactions_id_seq OWNED BY public.forecast_transactions.id;

ALTER TABLE public.forecast_transactions
    ADD CONSTRAINT forecast_transactions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE public.forecast_transactions
    ADD CONSTRAINT forecast_transactions_recurrency_pattern_id_fkey
    FOREIGN KEY (recurrency_pattern_id) REFERENCES public.recurrency_patterns(id) ON DELETE SET NULL;

ALTER TABLE public.forecast_transactions
    ADD CONSTRAINT forecast_transactions_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE SET NULL;

CREATE INDEX idx_forecast_user_id ON public.forecast_transactions USING btree (user_id);
CREATE INDEX idx_forecast_user_month ON public.forecast_transactions USING btree (user_id, forecast_month);
CREATE INDEX idx_forecast_parent_category ON public.forecast_transactions USING btree (parent_category) WHERE (parent_category IS NOT NULL);
CREATE INDEX idx_forecast_account_id ON public.forecast_transactions USING btree (account_id) WHERE (account_id IS NOT NULL);

-- 3. Transactions: add recurrency columns
ALTER TABLE public.transactions ADD COLUMN recurrency_type character varying(30);
ALTER TABLE public.transactions ADD COLUMN recurrency_pattern_id bigint;

ALTER TABLE public.transactions
    ADD CONSTRAINT check_transaction_recurrency_type
    CHECK ((recurrency_type IS NULL) OR ((recurrency_type)::text = ANY ((ARRAY['recurrent_fixed', 'recurrent_variable', 'irregular'])::text[])));

ALTER TABLE public.transactions
    ADD CONSTRAINT transactions_recurrency_pattern_id_fkey
    FOREIGN KEY (recurrency_pattern_id) REFERENCES public.recurrency_patterns(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_recurrency_type ON public.transactions USING btree (recurrency_type) WHERE (recurrency_type IS NOT NULL);
CREATE INDEX idx_transactions_recurrency_pattern_id ON public.transactions USING btree (recurrency_pattern_id) WHERE (recurrency_pattern_id IS NOT NULL);

-- 4. Clean up cousin names: replace underscores with spaces
UPDATE cousins
  SET name = REPLACE(name, '_', ' '),
      business_name = REPLACE(business_name, '_', ' ')
  WHERE name LIKE '%\_%' ESCAPE '\'
     OR business_name LIKE '%\_%' ESCAPE '\';

-- 5. Drop installment constraints from credit_card_data
ALTER TABLE public.credit_card_data DROP CONSTRAINT IF EXISTS check_installment_number;
ALTER TABLE public.credit_card_data DROP CONSTRAINT IF EXISTS check_total_installments;
ALTER TABLE public.credit_card_data DROP CONSTRAINT IF EXISTS check_installment_range;
