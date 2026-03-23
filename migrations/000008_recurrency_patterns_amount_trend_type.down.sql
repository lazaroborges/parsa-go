-- Revert amount_trend back to original type
ALTER TABLE recurrency_patterns
    ALTER COLUMN amount_trend TYPE NUMERIC(10, 6);
