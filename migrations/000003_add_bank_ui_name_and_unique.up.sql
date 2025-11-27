-- Add ui_name column to banks table
ALTER TABLE banks ADD COLUMN ui_name VARCHAR(255);

-- Add unique constraint on name
ALTER TABLE banks ADD CONSTRAINT banks_name_unique UNIQUE (name);

