-- Rollback migration 000007

-- 4. Restore installment constraints on credit_card_data
ALTER TABLE public.credit_card_data ADD CONSTRAINT check_installment_number CHECK (installment_number > 0);
ALTER TABLE public.credit_card_data ADD CONSTRAINT check_total_installments CHECK (total_installments > 0);
ALTER TABLE public.credit_card_data ADD CONSTRAINT check_installment_range CHECK (installment_number <= total_installments);

-- 3. Transactions: remove recurrency columns
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_recurrency_pattern_id_fkey;
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS check_transaction_recurrency_type;
DROP INDEX IF EXISTS public.idx_transactions_recurrency_type;
DROP INDEX IF EXISTS public.idx_transactions_recurrency_pattern_id;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS recurrency_pattern_id;
ALTER TABLE public.transactions DROP COLUMN IF EXISTS recurrency_type;

-- 2. Drop forecast transactions
DROP TABLE IF EXISTS public.forecast_transactions;
DROP SEQUENCE IF EXISTS public.forecast_transactions_id_seq;

-- 1. Drop recurrency patterns
DROP TABLE IF EXISTS public.recurrency_patterns;
DROP SEQUENCE IF EXISTS public.recurrency_patterns_id_seq;
