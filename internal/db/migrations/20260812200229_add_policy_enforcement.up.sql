-- Policy enforcement: the OPA policy set registry, its policies and per-run snapshot, the
-- policy-check run node columns, and the run rules / approval gates that govern soft-mandatory
-- overrides. Aggregated into a single migration.

-- ============================================================================================
-- Package registry (group-owned, versioned packages; today OPA policy bundles).
-- ============================================================================================

CREATE TABLE IF NOT EXISTS packages (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    group_id UUID NOT NULL,
    root_group_id UUID NOT NULL,
    kind VARCHAR NOT NULL,
    visibility VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    allow_mutable_versions BOOLEAN NOT NULL,
    CONSTRAINT fk_packages_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_packages_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_packages_on_name ON packages(group_id, name);
CREATE INDEX IF NOT EXISTS index_packages_on_root_group_id ON packages(root_group_id);

CREATE TABLE IF NOT EXISTS package_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    package_id UUID NOT NULL,
    semantic_version VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    error VARCHAR,
    object_store_key VARCHAR,
    sha_sum bytea NOT NULL,
    size INTEGER NOT NULL,
    latest BOOLEAN NOT NULL,
    upload_started_at TIMESTAMP,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_package_versions_package_id FOREIGN KEY(package_id) REFERENCES packages(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_package_versions_on_semantic_version ON package_versions(package_id, semantic_version);
CREATE UNIQUE INDEX IF NOT EXISTS index_package_versions_on_latest ON package_versions(package_id, latest) WHERE latest = true;

-- Track package version objects in object_store_refs (added in the object_store_refs migration) so
-- the janitor reclaims them when a package version is deleted. The FK column is added here because
-- package_versions is created above in this same migration.
ALTER TABLE object_store_refs
    ADD COLUMN IF NOT EXISTS package_version_id UUID REFERENCES package_versions(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_package_version_id ON object_store_refs(package_version_id);

-- Recreate the janitor's orphan partial index to include the new FK column. The WHERE clause must
-- list every FK column or the janitor would treat a package-version-owned ref as orphaned.
DROP INDEX IF EXISTS index_object_store_refs_orphan;
CREATE INDEX IF NOT EXISTS index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL
  AND package_version_id IS NULL;

-- ============================================================================================
-- Policy set policies. Every policy is owned by exactly one group. PackageSource is a
-- fully-qualified "<group-path>/<package-name>" reference resolved at run creation time.
-- Scope is a JSONB array of ScopeRule objects controlling which runs the policy applies to;
-- an empty array means the policy applies to all workspaces under the owning group.
-- ============================================================================================

CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    kind VARCHAR NOT NULL,
    kind_data JSONB,
    scope JSONB NOT NULL DEFAULT '[]',
    required_approvals INTEGER NOT NULL DEFAULT 0,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_policies_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
-- Policy names are unique within an owning group.
CREATE UNIQUE INDEX IF NOT EXISTS index_policies_on_group ON policies(group_id, name);

-- Approver principals for soft-mandatory policies (who may approve an override), replacing the
-- former standalone approval rules. required_approvals lives on the policy above.
CREATE TABLE IF NOT EXISTS policy_allowed_users (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_users_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_users_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- A principal is an approver for a policy once, so (policy_id, principal) is unique. The composite
-- doubles as the policy_id FK index, and is the index every read wants anyway: the approver lists are
-- always loaded and replaced by policy (db/policy.go scanChildIDs / UpdatePolicy). The trailing
-- single-column index covers the cascade from a deleted principal, which the composite cannot serve.
-- Same shape as team_members, which pairs a unique (user_id, team_id) with an index on team_id.
CREATE UNIQUE INDEX IF NOT EXISTS index_policy_allowed_users_on_policy_id_user_id ON policy_allowed_users(policy_id, user_id);
CREATE INDEX IF NOT EXISTS index_policy_allowed_users_on_user_id ON policy_allowed_users(user_id);

CREATE TABLE IF NOT EXISTS policy_allowed_service_accounts (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_sas_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_sas_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_policy_allowed_service_accounts_on_policy_id_sa_id ON policy_allowed_service_accounts(policy_id, service_account_id);
CREATE INDEX IF NOT EXISTS index_policy_allowed_service_accounts_on_sa_id ON policy_allowed_service_accounts(service_account_id);

CREATE TABLE IF NOT EXISTS policy_allowed_teams (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_teams_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_teams_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_policy_allowed_teams_on_policy_id_team_id ON policy_allowed_teams(policy_id, team_id);
CREATE INDEX IF NOT EXISTS index_policy_allowed_teams_on_team_id ON policy_allowed_teams(team_id);


ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS policy_check_type VARCHAR;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS stage_name VARCHAR;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS policy_check_policies JSONB;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS policy_check_messages_summary JSONB;

-- An advisory policy failure never blocks a run, so the run's status cannot record that one happened.
-- has_advisory_failures caches "at least one advisory policy failed" on the run row -- recomputed from the
-- checks whenever one reports outcomes or is retried -- so a list of runs can flag findings without
-- loading every check's policies. Every policy on an assessment run is forced to advisory, which is the
-- case that most needs it.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS has_advisory_failures BOOLEAN NOT NULL DEFAULT FALSE;

-- ============================================================================================
-- Job type-specific data. job_data is a nullable JSONB blob holding the payload for job types
-- that need one (today only the OPA policy-eval job, which stores its policy check node id so
-- the runner can report outcomes without scanning the run). Plan/apply jobs leave it NULL.
-- ============================================================================================

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS job_data JSONB;

-- A policy check's OPA evaluation jobs (including retries) are matched on the policy check id stored
-- in job_data (OPAJobData.policyCheckID); the PolicyCheck.jobs GraphQL field runs that filter. This
-- partial expression index backs it and only covers jobs that carry a policy check id (OPA jobs),
-- keeping it small. The query's "job_data->>'policyCheckID' = ?" predicate matches the same
-- expression, so the planner uses it.
CREATE INDEX IF NOT EXISTS index_jobs_on_policy_check_id
    ON jobs ((job_data->>'policyCheckID'))
    WHERE (job_data->>'policyCheckID') IS NOT NULL;

-- ============================================================================================
-- Run gates: one gate per run policy check that parks at awaiting_override because one or more
-- soft-mandatory policies failed. Approvers live on the policy (see above); the gate captures a
-- durable snapshot of the per-policy approval requirements at creation.
-- ============================================================================================

-- type is the kind of policy check the gate governs (e.g. "opa_policy"). approval_rules is a JSONB
-- list with one entry per soft-failed policy: {name, requiredApprovals, allowedSubjects}, where
-- name is the run's PolicyCheckPolicy ID (no FK — the run's JSONB snapshot is the source of truth,
-- and the UI joins on name for display). approval_rules is the durable/display record. workspace_id
-- is denormalized from the run so gates can be filtered by namespace membership without joining
-- through runs.
CREATE TABLE IF NOT EXISTS run_gates (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    run_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    policy_check_id VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    approval_rules JSONB NOT NULL,
    status VARCHAR NOT NULL,
    -- Set only when status is 'overridden': the subject that bypassed the approval requirements and
    -- the reason they gave. A gate that reached 'approved' by collecting its approvals leaves both
    -- null, which is what distinguishes the two ways a gate is cleared.
    overridden_by VARCHAR,
    override_comment VARCHAR,
    CONSTRAINT fk_run_gates_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gates_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_run_gates_on_run_id ON run_gates(run_id);
CREATE INDEX IF NOT EXISTS index_run_gates_on_workspace_id ON run_gates(workspace_id);
-- One gate per policy-check node, enforced rather than assumed: the API exposes the gate as a
-- one-to-one field on the check, the creation guard skips a check that already has one, and a
-- retried check deletes its gate before a new one can be created.
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gates_on_policy_check_id ON run_gates(policy_check_id);

-- Query-index tables derived from the gate snapshot at creation, so the approvals inbox can
-- efficiently find gates where a principal is an eligible approver.
CREATE TABLE IF NOT EXISTS run_gate_allowed_users (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_users_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_users_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- These are query-index tables seeded once per gate from a deduped set (db/rungate.go
-- allowedSubjectIDsByType), so the unique constraint records that invariant rather than changing what
-- a caller can do: one row per (gate, principal). The single-column index serves the inbox lookup,
-- which searches by principal; the composite serves the gate_id cascade, which fires on every
-- policy-check retry and not only when the run is removed.
CREATE INDEX IF NOT EXISTS index_run_gate_allowed_users_on_user_id ON run_gate_allowed_users(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gate_allowed_users_on_gate_id_user_id ON run_gate_allowed_users(gate_id, user_id);

CREATE TABLE IF NOT EXISTS run_gate_allowed_service_accounts (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_service_accounts_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_service_accounts_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_run_gate_allowed_service_accounts_on_sa_id ON run_gate_allowed_service_accounts(service_account_id);
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gate_allowed_service_accounts_on_gate_id_sa_id ON run_gate_allowed_service_accounts(gate_id, service_account_id);

CREATE TABLE IF NOT EXISTS run_gate_allowed_teams (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_teams_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_teams_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_run_gate_allowed_teams_on_team_id ON run_gate_allowed_teams(team_id);
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gate_allowed_teams_on_gate_id_team_id ON run_gate_allowed_teams(gate_id, team_id);

CREATE TABLE IF NOT EXISTS run_gate_approvals (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    run_gate_id UUID NOT NULL,
    user_id UUID,
    service_account_id UUID,
    created_by VARCHAR NOT NULL,
    decision VARCHAR NOT NULL,
    comment VARCHAR,
    covered_rules JSONB NOT NULL DEFAULT '[]',
    CONSTRAINT fk_run_gate_approvals_run_gate_id FOREIGN KEY(run_gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_approvals_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_run_gate_approvals_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS index_run_gate_approvals_on_run_gate_id ON run_gate_approvals(run_gate_id);
-- The approver FKs are ON DELETE SET NULL, so deleting a user or service account has to find their
-- decisions. This table keeps a row per decision for the life of the run, so it is the one child
-- table here that grows without bound. Partial: a decision is made by a user or a service account,
-- never both, so half the rows are NULL in each column and no lookup ever passes NULL.
CREATE INDEX IF NOT EXISTS index_run_gate_approvals_on_user_id ON run_gate_approvals(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_run_gate_approvals_on_sa_id ON run_gate_approvals(service_account_id) WHERE service_account_id IS NOT NULL;
-- A principal decides once per gate: the decision command replaces a prior decision from the same
-- principal rather than inserting a second row, so at most one row per (gate, principal) may exist.
-- These partial unique indexes enforce that invariant at the schema level -- without them, a duplicate
-- row could inflate a rule's approval count past what its approver list can ever provide and clear a
-- gate that should still be waiting. Partial because exactly one of user_id/service_account_id is set
-- per row (never both, never neither), so a NULL column carries no principal to constrain.
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gate_approvals_on_gate_id_user_id ON run_gate_approvals(run_gate_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS index_run_gate_approvals_on_gate_id_sa_id ON run_gate_approvals(run_gate_id, service_account_id) WHERE service_account_id IS NOT NULL;

-- ============================================================================================
-- Activity event target columns for policy-enforcement types.
-- ============================================================================================

ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS package_target_id UUID;
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS package_version_target_id UUID;
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS policy_target_id UUID;
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS run_gate_target_id UUID;

ALTER TABLE activity_events
    ADD CONSTRAINT fk_activity_events_package_target_id FOREIGN KEY(package_target_id) REFERENCES packages(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_activity_events_package_version_target_id FOREIGN KEY(package_version_target_id) REFERENCES package_versions(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_activity_events_policy_target_id FOREIGN KEY(policy_target_id) REFERENCES policies(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_activity_events_run_gate_target_id FOREIGN KEY(run_gate_target_id) REFERENCES run_gates(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS index_activity_events_on_package_target_id ON activity_events(package_target_id) WHERE package_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_package_version_target_id ON activity_events(package_version_target_id) WHERE package_version_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_policy_target_id ON activity_events(policy_target_id) WHERE policy_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_run_gate_target_id ON activity_events(run_gate_target_id) WHERE run_gate_target_id IS NOT NULL;

-- ============================================================================================
-- Run admission: split the queuing run statuses per phase.
--
-- run_refactor introduced queuing / queuing_apply to mean "this node is ready but has not been admitted
-- to the workspace yet", as distinct from plan_queued / apply_queued ("admitted; its job is waiting for a
-- runner"). Policy enforcement adds workspace-gated task stages, so that pair becomes one *_queuing
-- status per gated node: pre_plan_queuing, plan_queuing, pre_apply_queuing, apply_queuing.
--
-- queuing and queuing_apply already mean exactly plan_queuing and apply_queuing, so they are renamed
-- rather than reinterpreted. The pre_plan / pre_apply statuses are new and have no existing rows.
-- ============================================================================================

UPDATE runs SET status = 'plan_queuing' WHERE status = 'queuing';
UPDATE runs SET status = 'apply_queuing' WHERE status = 'queuing_apply';

-- Both admission sweeps select runs by their queuing statuses, so both are backed by a partial index
-- scoped to that set. It stays tiny (queuing is transient) and cheap to maintain on a hot table.
--
-- These predicates are the SQL copy of models.QueuingRunStatuses, which is the Go-side definition every
-- caller reads. A predicate cannot call into Go, so adding a workspace-gated node means adding its
-- status there AND extending both indexes in a new migration.

-- The run reconciler's cross-workspace sweep, which filters on updated_at with no workspace_id. Recreated
-- rather than altered: run_refactor scoped it to the two old statuses.
DROP INDEX IF EXISTS index_runs_on_updated_at_queuing;
CREATE INDEX IF NOT EXISTS index_runs_on_updated_at_queuing
    ON runs(updated_at)
    WHERE status IN ('pre_plan_queuing', 'plan_queuing', 'pre_apply_queuing', 'apply_queuing');

-- The work item consumer's workspace-scoped, oldest-first sweep. run_refactor's
-- index_runs_on_workspace_id_status_updated_at cannot serve it: that one is ordered by updated_at, and
-- this sweep admits in created_at order so runs are queued in the order they were requested.
CREATE INDEX IF NOT EXISTS index_runs_on_workspace_id_created_at_queuing
    ON runs(workspace_id, created_at)
    WHERE status IN ('pre_plan_queuing', 'plan_queuing', 'pre_apply_queuing', 'apply_queuing');

-- Resource limits for the policy-enforcement feature (the resource_limits table is created in the init migration).
INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
    ('887b98dc-2afe-4ba3-b938-2b902c56198a', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPackagesPerGroup', 1000), -- number of packages per group
    ('3b021da6-ad87-4b04-a75d-ddc202d70bd8', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPoliciesPerGroup', 100), -- number of policies per group or workspace
    ('b2ba9fb6-06a7-45e1-a82b-9140aba91cb6', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerPackagePerTimePeriod', 100) -- number of package versions per package per time period
ON CONFLICT DO NOTHING;
