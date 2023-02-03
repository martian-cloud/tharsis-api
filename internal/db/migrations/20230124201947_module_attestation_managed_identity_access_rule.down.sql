ALTER TABLE managed_identity_rules
    DROP COLUMN IF EXISTS type;

ALTER TABLE managed_identity_rules
    DROP COLUMN IF EXISTS module_attestation_policies;

CREATE UNIQUE INDEX IF NOT EXISTS index_managed_identity_rules_on_run_stage ON managed_identity_rules(managed_identity_id, run_stage);

ALTER TABLE runs
    DROP COLUMN IF EXISTS module_digest;
