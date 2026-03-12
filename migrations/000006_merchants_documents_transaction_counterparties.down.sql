-- Rollback migration 000006

-- 4. Cousins: remove merchant_id
ALTER TABLE public.cousins DROP CONSTRAINT IF EXISTS cousins_merchant_id_fkey;
DROP INDEX IF EXISTS public.idx_cousins_merchant_id;
ALTER TABLE public.cousins DROP COLUMN IF EXISTS merchant_id;

-- 3. Transactions: remove merchant_id and document_id
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_merchant_id_fkey;
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_document_id_fkey;
DROP INDEX IF EXISTS public.idx_transactions_merchant_id;
DROP INDEX IF EXISTS public.idx_transactions_document_id;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS merchant_id;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS document_id;

-- 2. Documents: remove business_name, restore number NOT NULL
DROP INDEX IF EXISTS public.idx_documents_business_name_lower;
DROP INDEX IF EXISTS public.idx_documents_number_unique;

-- Delete documents created from business_name only (no number) before restoring NOT NULL
DELETE FROM public.documents WHERE number IS NULL;

ALTER TABLE public.documents DROP COLUMN IF EXISTS business_name;
ALTER TABLE public.documents ALTER COLUMN number SET NOT NULL;

ALTER TABLE public.documents ADD CONSTRAINT documents_number_key UNIQUE (number);

-- 1. Drop merchants table
DROP TABLE IF EXISTS public.merchants;
DROP SEQUENCE IF EXISTS public.merchants_id_seq;
