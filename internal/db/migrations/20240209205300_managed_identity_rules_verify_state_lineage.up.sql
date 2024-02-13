ALTER TABLE managed_identity_rules ADD COLUMN IF NOT EXISTS verify_state_lineage BOOLEAN NOT NULL DEFAULT FALSE;
