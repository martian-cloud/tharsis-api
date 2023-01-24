ALTER TABLE IF EXISTS managed_identities
    DROP CONSTRAINT if_alias_source_id_is_null_require_not_null,
    DROP COLUMN IF EXISTS alias_source_id,
    ALTER COLUMN type SET NOT NULL,
    ALTER COLUMN data SET NOT NULL;
