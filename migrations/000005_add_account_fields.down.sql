ALTER TABLE accounts DROP COLUMN IF EXISTS hidden_by_user;
ALTER TABLE accounts DROP COLUMN IF EXISTS removed;
ALTER TABLE accounts DROP COLUMN IF EXISTS description;
ALTER TABLE accounts DROP COLUMN IF EXISTS "order";
ALTER TABLE accounts DROP COLUMN IF EXISTS closed_at;
ALTER TABLE accounts DROP COLUMN IF EXISTS is_open_finance_account;
ALTER TABLE accounts DROP COLUMN IF EXISTS initial_balance;