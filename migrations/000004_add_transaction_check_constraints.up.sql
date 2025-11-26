-- Add CHECK constraints for transactions.type and transactions.status columns
ALTER TABLE transactions
    ADD CONSTRAINT check_transactions_type CHECK (type IN ('DEBIT', 'CREDIT'));

ALTER TABLE transactions
    ADD CONSTRAINT check_transactions_status CHECK (status IN ('PENDING', 'POSTED'));

