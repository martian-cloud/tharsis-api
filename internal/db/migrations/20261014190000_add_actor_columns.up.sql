-- created_by_actor_type and created_by_actor_id record the principal that
-- created a row as a structured actor. The type is one of user,
-- service_account, job, vcs_provider, system or scim. The id is the UUID of the
-- referenced record and is NULL for system and scim, which have no backing
-- record. Both sit alongside
-- created_by, which keeps its existing free-form value for backward
-- compatibility.
--
-- created_by_actor_trn is a display snapshot so clients can render the actor
-- without looking up its principal: the TRN of a service account, VCS provider
-- or job, which names its resource path (for a job, its workspace path and
-- ID). It is NULL for users, system
-- and scim. A user never gets a TRN because a user TRN names its username, which identifies a
-- person, and a copy on every row would outlive the user's deletion.
--
-- The id is a native UUID so comparisons and any future index work on 16-byte
-- values rather than text. There is deliberately no foreign key: the actor must
-- outlive the principal it refers to.
--
-- Nullable with no default, so each ADD COLUMN is a metadata-only change. A NULL
-- type means the row has no structured actor, which is the case for every
-- pre-existing row the backfill in 20261014190001_backfill_actor_columns cannot
-- resolve. The API reports no actor for those, only the legacy created_by
-- value, and "WHERE created_by_actor_type IS NULL" reports how much un-migrated
-- data remains.
--
-- The backfill lives in its own migration so these ALTER TABLEs, which take an
-- ACCESS EXCLUSIVE lock, commit immediately instead of holding that lock on
-- every table until the backfill UPDATEs finish.
--
-- No indexes are added here. Add one on (created_by_actor_id,
-- created_by_actor_type) only for tables that are actually queried by actor, so
-- writes elsewhere do not pay for index maintenance.
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE configuration_versions ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE federated_registries ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE gpg_keys ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE maintenance_mode ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE managed_identities ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE package_versions ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE policies ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE run_gate_approvals ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE runners ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE scim_tokens ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE service_accounts ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE state_versions ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_module_attestations ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_module_versions ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_modules ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_provider_platforms ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_provider_version_mirrors ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE terraform_providers ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE vcs_providers ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE workspace_role_bindings ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE workspace_vcs_provider_links ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS created_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS created_by_actor_id UUID, ADD COLUMN IF NOT EXISTS created_by_actor_trn VARCHAR;

-- apply_triggered_by_actor_type and apply_triggered_by_actor_id record the
-- principal that triggered an apply as a structured actor, using the same
-- type/id/trn layout as the created_by_actor_* columns. They sit
-- alongside apply_triggered_by, which keeps its existing free-form value.
-- Nullable, like the columns above: a NULL type means the apply has no
-- structured actor, which is the case for every apply triggered before this
-- migration that the backfill cannot resolve.
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS apply_triggered_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS apply_triggered_by_actor_id UUID, ADD COLUMN IF NOT EXISTS apply_triggered_by_actor_trn VARCHAR;

-- The same layout for the principal that force-canceled a run and the principal
-- that overrode a run gate, alongside force_canceled_by and overridden_by.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS force_canceled_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS force_canceled_by_actor_id UUID, ADD COLUMN IF NOT EXISTS force_canceled_by_actor_trn VARCHAR;
ALTER TABLE run_gates ADD COLUMN IF NOT EXISTS overridden_by_actor_type VARCHAR, ADD COLUMN IF NOT EXISTS overridden_by_actor_id UUID, ADD COLUMN IF NOT EXISTS overridden_by_actor_trn VARCHAR;
