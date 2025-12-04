-- Drop all tables in reverse dependency order

DROP TABLE IF EXISTS public.cousins CASCADE;
DROP TABLE IF EXISTS public.documents CASCADE;
DROP TABLE IF EXISTS public.credit_card_data CASCADE;
DROP TABLE IF EXISTS public.transactions CASCADE;
DROP TABLE IF EXISTS public.accounts CASCADE;
DROP TABLE IF EXISTS public.items CASCADE;
DROP TABLE IF EXISTS public.banks CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;

