-- Bills table: represents bills/boletos from Open Finance
CREATE TABLE public.bills (
    id character varying(255) NOT NULL,
    account_id character varying(255) NOT NULL,
    amount numeric(15,2) NOT NULL,
    due_date timestamp with time zone NOT NULL,
    status character varying(20) DEFAULT 'OPEN'::character varying NOT NULL,
    description text NOT NULL DEFAULT '',
    biller_name character varying(255) NOT NULL DEFAULT '',
    category character varying(100),
    barcode character varying(255),
    digitable_line character varying(255),
    payment_date timestamp with time zone,
    related_transaction_id character varying(255),
    provider_created_at timestamp with time zone,
    provider_updated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    is_open_finance boolean DEFAULT true NOT NULL,
    CONSTRAINT bills_pkey PRIMARY KEY (id),
    CONSTRAINT check_bills_status CHECK (((status)::text = ANY ((ARRAY['OPEN'::character varying, 'PAID'::character varying, 'OVERDUE'::character varying, 'CANCELLED'::character varying])::text[]))),
    CONSTRAINT bills_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE,
    CONSTRAINT bills_related_transaction_id_fkey FOREIGN KEY (related_transaction_id) REFERENCES public.transactions(id) ON DELETE SET NULL
);

CREATE INDEX idx_bills_account_id ON public.bills USING btree (account_id);
CREATE INDEX idx_bills_due_date ON public.bills USING btree (due_date);
CREATE INDEX idx_bills_status ON public.bills USING btree (status);
CREATE INDEX idx_bills_biller_name ON public.bills USING btree (biller_name);
