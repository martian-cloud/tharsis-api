-- Rollback migration: Remove JSONB labels column from workspaces table

-- Drop index for JSONB labels column
DROP INDEX IF EXISTS index_workspaces_labels;

-- Remove labels column from workspaces table
ALTER TABLE workspaces DROP COLUMN IF EXISTS labels;
