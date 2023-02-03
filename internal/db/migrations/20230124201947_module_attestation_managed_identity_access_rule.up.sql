ALTER TABLE managed_identity_rules
    ADD COLUMN IF NOT EXISTS type VARCHAR NOT NULL DEFAULT 'eligible_principals',
    ADD COLUMN IF NOT EXISTS module_attestation_policies JSONB;

DROP INDEX IF EXISTS index_managed_identity_rules_on_run_stage;

ALTER TABLE runs
    ADD COLUMN IF NOT EXISTS module_digest bytea;
