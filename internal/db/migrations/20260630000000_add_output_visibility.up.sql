ALTER TABLE groups ADD COLUMN IF NOT EXISTS output_visibility VARCHAR;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS output_visibility VARCHAR;

-- Set default output visibility for existing root groups (no parent) that don't have an explicit value.
-- ROOT_GROUP preserves today's behavior (any workspace in the same root group can read outputs).
UPDATE groups SET output_visibility = 'root_group' WHERE parent_id IS NULL AND output_visibility IS NULL;
