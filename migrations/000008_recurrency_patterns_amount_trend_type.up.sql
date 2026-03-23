-- Migration 000008: Widen amount_trend column to NUMERIC(15,2)
-- The enrichment engine produces values that exceed NUMERIC(10,6) constraints.

ALTER TABLE recurrency_patterns
    ALTER COLUMN amount_trend TYPE NUMERIC(15, 2)
    USING ROUND(amount_trend, 2);
