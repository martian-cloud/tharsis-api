-- Add JSONB labels column to terraform_modules table

-- Add JSONB labels column with default empty object
ALTER TABLE terraform_modules ADD COLUMN labels JSONB DEFAULT '{}';

-- Create GIN index on labels column for efficient JSONB containment queries
CREATE INDEX index_terraform_modules_labels ON terraform_modules USING GIN (labels jsonb_ops);
