ALTER TABLE IF EXISTS managed_identities
    ADD COLUMN IF NOT EXISTS alias_source_id UUID,
    ALTER COLUMN type DROP NOT NULL,
    ALTER COLUMN data DROP NOT NULL,
    ADD CONSTRAINT fk_alias_source_id FOREIGN KEY(alias_source_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    ADD CONSTRAINT if_alias_source_id_is_null_require_not_null
        CHECK((alias_source_id IS NOT NULL) OR ((type IS NOT NULL) AND (data is NOT NULL)));
