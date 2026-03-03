-- Migration 000004: Replace accounts.removed with removed_at; add items.deleted_at; add transactions.notified_at

-- Accounts: migrate removed (boolean) -> removed_at (timestamptz)
ALTER TABLE public.accounts ADD COLUMN removed_at timestamp with time zone;
UPDATE public.accounts SET removed_at = CURRENT_TIMESTAMP WHERE removed = true;
ALTER TABLE public.accounts DROP COLUMN removed;

-- Items: add soft-delete column
ALTER TABLE public.items ADD COLUMN deleted_at timestamp with time zone;

-- Transactions: add notified_at (null for new rows; set to current time for existing rows)
ALTER TABLE public.transactions ADD COLUMN notified_at timestamp with time zone;
UPDATE public.transactions SET notified_at = CURRENT_TIMESTAMP;
