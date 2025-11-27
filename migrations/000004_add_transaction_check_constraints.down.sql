-- Remove CHECK constraints for transactions.type and transactions.status columns
ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS check_transactions_type;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS check_transactions_status;

