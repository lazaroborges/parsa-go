-- Make connector column nullable since we may only have bank name from transactions
ALTER TABLE banks ALTER COLUMN connector DROP NOT NULL;

