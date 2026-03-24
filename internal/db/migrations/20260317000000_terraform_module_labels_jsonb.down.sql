-- Rollback migration: Remove JSONB labels column from terraform_modules table

-- Drop index for JSONB labels column
DROP INDEX IF EXISTS index_terraform_modules_labels;

-- Remove labels column from terraform_modules table
ALTER TABLE terraform_modules DROP COLUMN IF EXISTS labels;
