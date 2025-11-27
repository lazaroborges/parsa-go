-- Remove unique constraint on name
ALTER TABLE banks DROP CONSTRAINT banks_name_unique;

-- Remove ui_name column
ALTER TABLE banks DROP COLUMN ui_name;

