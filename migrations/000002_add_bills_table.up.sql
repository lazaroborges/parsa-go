-- Bills table: represents past due credit card bills (faturas vencidas)
-- Data from Pierre Finance GET /tools/api/get-bills endpoint
CREATE TABLE public.bills (
    id character varying(255) NOT NULL,
    account_id character varying(255) NOT NULL,
    due_date timestamp with time zone NOT NULL,
    close_date timestamp with time zone,
    total_amount numeric(15,2) NOT NULL,
    minimum_payment numeric(15,2),
    status character varying(20) DEFAULT 'OVERDUE'::character varying NOT NULL,
    is_overdue boolean DEFAULT true NOT NULL,
    provider_created_at timestamp with time zone,
    provider_updated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    is_open_finance boolean DEFAULT true NOT NULL,
    CONSTRAINT bills_pkey PRIMARY KEY (id),
    CONSTRAINT check_bills_status CHECK (((status)::text = ANY ((ARRAY['OPEN'::character varying, 'CLOSED'::character varying, 'OVERDUE'::character varying, 'PAID'::character varying])::text[]))),
    CONSTRAINT bills_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_bills_account_id ON public.bills USING btree (account_id);
CREATE INDEX idx_bills_due_date ON public.bills USING btree (due_date);
CREATE INDEX idx_bills_status ON public.bills USING btree (status);
CREATE INDEX idx_bills_is_overdue ON public.bills USING btree (is_overdue);
