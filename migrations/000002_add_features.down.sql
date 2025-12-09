-- Rollback migration 000002

DROP TRIGGER IF EXISTS trigger_notify_cousin_assigned ON public.transactions;
DROP FUNCTION IF EXISTS public.notify_cousin_assigned();

ALTER TABLE public.transactions DROP COLUMN IF EXISTS provider_category_id;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS was_deleted;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS hasmagic;

DROP INDEX IF EXISTS idx_transactions_cousin;
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_cousin_fkey;
UPDATE public.transactions SET cousin = 0 WHERE cousin IS NULL;
ALTER TABLE public.transactions ALTER COLUMN cousin TYPE integer USING cousin::integer;
ALTER TABLE public.transactions ALTER COLUMN cousin SET DEFAULT 0;

ALTER TABLE public.cousins DROP COLUMN IF EXISTS description_gk;

DROP TABLE IF EXISTS public.user_ck_value_tags;
DROP TABLE IF EXISTS public.user_ck_values;
DROP SEQUENCE IF EXISTS public.user_ck_values_id_seq;

DROP TABLE IF EXISTS public.cousin_ner_patterns;
DROP SEQUENCE IF EXISTS public.cousin_ner_patterns_id_seq;

DROP TABLE IF EXISTS public.transaction_tags;
DROP TABLE IF EXISTS public.tags;
DROP TABLE IF EXISTS public.bills;