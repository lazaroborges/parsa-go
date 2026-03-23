-- Revert amount_trend back to original type
-- Guard against out-of-range values before narrowing the column
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM recurrency_patterns
    WHERE amount_trend IS NOT NULL
      AND ABS(amount_trend) >= 10000
  ) THEN
    RAISE EXCEPTION
      'Rollback blocked: amount_trend has values outside NUMERIC(10,6) range';
  END IF;
END $$;

ALTER TABLE recurrency_patterns
    ALTER COLUMN amount_trend TYPE NUMERIC(10, 6);
