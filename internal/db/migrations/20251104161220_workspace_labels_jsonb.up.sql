-- Add JSONB labels column to workspaces table
-- This migration implements the JSONB labels column as specified in the design document

-- Add JSONB labels column to workspaces table with default empty object
ALTER TABLE workspaces ADD COLUMN labels JSONB DEFAULT '{}';

-- Create GIN index on labels column for efficient JSONB containment queries
CREATE INDEX index_workspaces_labels ON workspaces USING GIN (labels jsonb_ops);
