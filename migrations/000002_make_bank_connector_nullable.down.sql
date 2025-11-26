-- Restore NOT NULL constraint (will fail if there are NULL values)
ALTER TABLE banks ALTER COLUMN connector SET NOT NULL;

