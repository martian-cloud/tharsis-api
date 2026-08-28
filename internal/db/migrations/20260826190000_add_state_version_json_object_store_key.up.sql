-- The `terraform show -json` rendering of a state version, stored as a separate object from the raw
-- state file. Nullable: state versions created before this column existed, or created by a client
-- that does not upload the rendering (an older job executor, the TFE state push endpoint, or the
-- GraphQL mutation), simply have no rendering.
ALTER TABLE state_versions ADD COLUMN IF NOT EXISTS json_object_store_key VARCHAR;
