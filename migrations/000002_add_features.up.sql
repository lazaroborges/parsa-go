-- Migration 000002: Add bills, tags, cousin rules, and related features

-- Bills table (credit card bills)
CREATE TABLE public.bills (
    id character varying(255) NOT NULL,
    account_id character varying(255) NOT NULL,
    due_date timestamp with time zone NOT NULL,
    total_amount numeric(15,2) NOT NULL,
    provider_created_at timestamp with time zone,
    provider_updated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    is_open_finance boolean DEFAULT true NOT NULL,
    CONSTRAINT bills_pkey PRIMARY KEY (id),
    CONSTRAINT bills_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_bills_account_id ON public.bills USING btree (account_id);
CREATE INDEX idx_bills_due_date ON public.bills USING btree (due_date);

-- Tags table
CREATE TABLE public.tags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    name character varying(128) NOT NULL,
    color character varying(12) NOT NULL,
    display_order integer DEFAULT 99 NOT NULL,
    description character varying(255) DEFAULT ''::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT tags_pkey PRIMARY KEY (id),
    CONSTRAINT tags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tags_user_id ON public.tags USING btree (user_id);

-- Transaction tags junction table
CREATE TABLE public.transaction_tags (
    transaction_id character varying(255) NOT NULL,
    tag_id uuid NOT NULL,
    CONSTRAINT transaction_tags_pkey PRIMARY KEY (transaction_id, tag_id),
    CONSTRAINT transaction_tags_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id) ON DELETE CASCADE,
    CONSTRAINT transaction_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_transaction_tags_transaction_id ON public.transaction_tags USING btree (transaction_id);
CREATE INDEX idx_transaction_tags_tag_id ON public.transaction_tags USING btree (tag_id);

-- Cousin NER patterns table
CREATE SEQUENCE public.cousin_ner_patterns_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.cousin_ner_patterns (
    id bigint NOT NULL DEFAULT nextval('public.cousin_ner_patterns_id_seq'::regclass),
    cousin_id bigint NOT NULL,
    pattern character varying(255) NOT NULL,
    CONSTRAINT cousin_ner_patterns_pkey PRIMARY KEY (id),
    CONSTRAINT cousin_ner_patterns_cousin_id_fkey FOREIGN KEY (cousin_id) REFERENCES public.cousins(id) ON DELETE CASCADE
);

ALTER SEQUENCE public.cousin_ner_patterns_id_seq OWNED BY public.cousin_ner_patterns.id;

CREATE INDEX idx_cousin_ner_patterns_cousin_id ON public.cousin_ner_patterns USING btree (cousin_id);
CREATE INDEX idx_cousin_ner_patterns_pattern ON public.cousin_ner_patterns USING btree (pattern);

-- User cousin key values (rules) table
CREATE SEQUENCE public.user_ck_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.user_ck_values (
    id bigint NOT NULL DEFAULT nextval('public.user_ck_values_id_seq'::regclass),
    user_id bigint NOT NULL,
    cousin_id bigint NOT NULL,
    type character varying(20),
    category character varying(100),
    description text,
    notes text,
    considered boolean,
    dont_ask_again boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_ck_values_pkey PRIMARY KEY (id),
    CONSTRAINT user_ck_values_unique UNIQUE (user_id, cousin_id, type),
    CONSTRAINT user_ck_values_type_check CHECK (((type IS NULL) OR ((type)::text = ANY ((ARRAY['DEBIT'::character varying, 'CREDIT'::character varying])::text[])))),
    CONSTRAINT user_ck_values_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT user_ck_values_cousin_id_fkey FOREIGN KEY (cousin_id) REFERENCES public.cousins(id) ON DELETE CASCADE
);

ALTER SEQUENCE public.user_ck_values_id_seq OWNED BY public.user_ck_values.id;

CREATE INDEX idx_user_ck_values_user_id ON public.user_ck_values USING btree (user_id);
CREATE INDEX idx_user_ck_values_cousin_id ON public.user_ck_values USING btree (cousin_id);
CREATE INDEX idx_user_ck_values_lookup ON public.user_ck_values USING btree (user_id, cousin_id, type);

-- User cousin key value tags junction table
CREATE TABLE public.user_ck_value_tags (
    user_ck_value_id bigint NOT NULL,
    tag_id uuid NOT NULL,
    CONSTRAINT user_ck_value_tags_pkey PRIMARY KEY (user_ck_value_id, tag_id),
    CONSTRAINT user_ck_value_tags_rule_fkey FOREIGN KEY (user_ck_value_id) REFERENCES public.user_ck_values(id) ON DELETE CASCADE,
    CONSTRAINT user_ck_value_tags_tag_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_ck_value_tags_rule_id ON public.user_ck_value_tags USING btree (user_ck_value_id);

-- Add description_gk column to cousins
ALTER TABLE public.cousins ADD COLUMN description_gk boolean DEFAULT false;

-- Modify transactions.cousin: change from integer to bigint, remove default, drop NOT NULL, add FK
-- First drop the default and NOT NULL constraint, then alter the type, then set 0s to NULL, then add FK
ALTER TABLE public.transactions ALTER COLUMN cousin DROP DEFAULT;
ALTER TABLE public.transactions ALTER COLUMN cousin DROP NOT NULL;
ALTER TABLE public.transactions ALTER COLUMN cousin TYPE bigint USING cousin::bigint;
-- Set existing 0 values to NULL before adding FK
UPDATE public.transactions SET cousin = NULL WHERE cousin = 0;
ALTER TABLE public.transactions ADD CONSTRAINT transactions_cousin_fkey 
    FOREIGN KEY (cousin) REFERENCES public.cousins(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_cousin ON public.transactions USING btree (cousin);

-- Add new columns to transactions
ALTER TABLE public.transactions ADD COLUMN hasmagic boolean DEFAULT false NOT NULL;
ALTER TABLE public.transactions ADD COLUMN was_deleted boolean DEFAULT false;
ALTER TABLE public.transactions ADD COLUMN provider_category_id character varying(30) DEFAULT NULL;

-- Create notify function and trigger for cousin assignment
CREATE FUNCTION public.notify_cousin_assigned() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (OLD.cousin IS NULL OR OLD.cousin = 0) AND NEW.cousin IS NOT NULL AND NEW.cousin != 0 THEN
        PERFORM pg_notify(
            'cousin_assigned',
            json_build_object(
                'transaction_id', NEW.id,
                'cousin_id', NEW.cousin,
                'type', NEW.type,
                'account_id', NEW.account_id
            )::text
        );
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trigger_notify_cousin_assigned 
    AFTER UPDATE OF cousin ON public.transactions 
    FOR EACH ROW EXECUTE FUNCTION public.notify_cousin_assigned();
