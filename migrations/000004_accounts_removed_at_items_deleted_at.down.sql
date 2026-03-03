-- Rollback migration 000004

ALTER TABLE public.transactions DROP COLUMN IF EXISTS notified_at;

ALTER TABLE public.items DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE public.accounts ADD COLUMN IF NOT EXISTS removed boolean DEFAULT false NOT NULL;
UPDATE public.accounts SET removed = true WHERE removed_at IS NOT NULL;
ALTER TABLE public.accounts DROP COLUMN IF EXISTS removed_at;
