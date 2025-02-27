DROP TABLE IF EXISTS latest_namespace_variable_versions;
DROP TABLE IF EXISTS namespace_variable_versions;

DELETE FROM namespace_variables WHERE value IS NULL;

ALTER TABLE namespace_variables DROP COLUMN IF EXISTS sensitive;
ALTER TABLE namespace_variables ALTER COLUMN value SET NOT NULL;
ALTER TABLE namespace_variables ALTER COLUMN hcl SET NOT NULL;

-- TODO: Add the hcl and value column to namespace_variables
