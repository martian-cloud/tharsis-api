-- The created_by_actor_type, created_by_actor_id and created_by_actor_trn columns found on many tables
-- record the principal that created a row as a structured actor. The type is one of user,
-- service_account, job, vcs_provider, system or scim. The id is the UUID of the referenced record and is
-- NULL for system and scim, which have no backing record. The trn is a display snapshot so clients can
-- render the actor without looking up its principal: the TRN of a service account, VCS provider or job,
-- and NULL for users, system and scim (a user TRN names its username, which identifies a person, and a
-- copy on every row would outlive the user's deletion). They sit alongside created_by, which keeps its
-- free-form value for backward compatibility. There is deliberately no foreign key: the actor must
-- outlive the principal it refers to. A NULL type means the row has no structured actor. The
-- apply_triggered_by_actor_*, force_canceled_by_actor_* and overridden_by_actor_* columns use the same
-- layout.

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    parent_id UUID,
    created_by VARCHAR NOT NULL,
    runner_tags JSONB,
    drift_detection_enabled BOOLEAN,
    provider_mirror_enabled BOOLEAN,
    output_visibility VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_parent_id FOREIGN KEY(parent_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE INDEX index_groups_on_parent_id ON groups(parent_id);

CREATE TABLE runners (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    type VARCHAR NOT NULL,
    group_id UUID,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB NOT NULL DEFAULT '[]',
    run_untagged_jobs BOOLEAN NOT NULL DEFAULT TRUE,
    -- Max number of concurrent jobs a runner can execute; 0 means no limit.
    job_limit INTEGER NOT NULL DEFAULT 0,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_runners_on_name ON runners(name) WHERE (group_id IS NULL);
CREATE UNIQUE INDEX index_runners_on_group_id_name ON runners(group_id, name);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    current_state_version_id UUID,
    group_id UUID NOT NULL,
    dirty_state BOOLEAN NOT NULL,
    max_job_duration INTEGER NOT NULL,
    created_by VARCHAR NOT NULL,
    locked BOOLEAN NOT NULL,
    terraform_version VARCHAR NOT NULL,
    prevent_destroy_plan BOOLEAN DEFAULT false,
    runner_tags JSONB,
    drift_detection_enabled BOOLEAN,
    labels JSONB DEFAULT '{}',
    provider_mirror_enabled BOOLEAN,
    -- The run currently holding the workspace for its apply phase.
    current_apply_run_id UUID,
    output_visibility VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE INDEX index_workspaces_created_at_id ON workspaces(created_at DESC, id DESC);
CREATE INDEX index_workspaces_labels ON workspaces USING GIN (labels jsonb_ops);
CREATE INDEX index_workspaces_on_group_id ON workspaces(group_id);
CREATE INDEX index_workspaces_on_current_state_version_id ON workspaces(current_state_version_id);
CREATE INDEX index_workspaces_on_current_apply_run_id ON workspaces(current_apply_run_id);
-- Serves the assessment scheduler's GetWorkspaces query.
CREATE INDEX index_workspaces_on_locked_current_state_version_id
    ON workspaces (locked, current_state_version_id)
    WHERE locked IS FALSE AND current_state_version_id IS NOT NULL;

CREATE TABLE vcs_events(
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    commit_id VARCHAR,
    source_reference_name VARCHAR,
    workspace_id UUID NOT NULL,
    type VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    repository_url VARCHAR NOT NULL,
    error_message VARCHAR,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX index_vcs_events_on_workspace_id ON vcs_events(workspace_id);

CREATE TABLE configuration_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    speculative BOOLEAN NOT NULL,
    workspace_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    vcs_event_id UUID,
    object_store_key VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_vcs_event_id FOREIGN KEY(vcs_event_id) REFERENCES vcs_events(id)
);
CREATE INDEX index_configuration_versions_on_workspace_id ON configuration_versions(workspace_id);
CREATE INDEX index_configuration_versions_on_vcs_event_id ON configuration_versions(vcs_event_id);

CREATE TABLE runs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    is_destroy BOOLEAN NOT NULL,
    workspace_id UUID NOT NULL,
    configuration_version_id UUID,
    created_by VARCHAR NOT NULL,
    module_source VARCHAR,
    module_version VARCHAR,
    force_canceled_by VARCHAR,
    force_cancel_available_at TIMESTAMP,
    force_canceled BOOLEAN NOT NULL,
    comment VARCHAR NOT NULL,
    auto_apply BOOLEAN NOT NULL,
    terraform_version VARCHAR NOT NULL DEFAULT '1.2.2',
    module_digest bytea,
    targets JSONB NOT NULL DEFAULT '[]',
    refresh BOOLEAN NOT NULL DEFAULT TRUE,
    refresh_only BOOLEAN NOT NULL DEFAULT FALSE,
    is_assessment_run BOOLEAN NOT NULL DEFAULT FALSE,
    variables_object_store_key VARCHAR,
    -- Caches "at least one advisory policy failed" on the run row, recomputed from the checks whenever
    -- one reports outcomes or is retried, so a list of runs can flag findings without loading every
    -- check's policies.
    has_advisory_failures BOOLEAN NOT NULL DEFAULT FALSE,
    -- Immutable key/value pairs (with an optional link) set at run creation, letting a run be traced
    -- back to what created it (e.g. commit, repository, triggering job).
    annotations JSONB NOT NULL DEFAULT '[]',
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    force_canceled_by_actor_type VARCHAR,
    force_canceled_by_actor_id UUID,
    force_canceled_by_actor_trn VARCHAR,
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_configuration_version FOREIGN KEY(configuration_version_id) REFERENCES configuration_versions(id)
);
CREATE INDEX index_runs_created_at_id ON runs(created_at DESC, id DESC);
CREATE INDEX index_runs_on_configuration_version_id ON runs(configuration_version_id);
-- Serves the supervisor's workspace-scoped run queries (workspace_id + status IN, and workspace_id +
-- status + updated_at range).
CREATE INDEX index_runs_on_workspace_id_status_updated_at ON runs(workspace_id, status, updated_at);
-- Pruners page a workspace's runs in fixed order.
CREATE INDEX index_runs_on_workspace_id_created_at ON runs(workspace_id, created_at DESC);
-- The next two indexes serve the admission sweeps, which select runs by their queuing statuses. Their
-- predicates are the SQL copy of models.QueuingRunStatuses: adding a workspace-gated node means adding
-- its status there AND extending both indexes in a new migration.

-- The run reconciler's cross-workspace sweep, which filters on updated_at with no workspace_id.
CREATE INDEX index_runs_on_updated_at_queuing
    ON runs(updated_at)
    WHERE status IN ('pre_plan_queuing', 'plan_queuing', 'pre_apply_queuing', 'apply_queuing');
-- The work item consumer's workspace-scoped, oldest-first sweep, which admits in created_at order.
CREATE INDEX index_runs_on_workspace_id_created_at_queuing
    ON runs(workspace_id, created_at)
    WHERE status IN ('pre_plan_queuing', 'plan_queuing', 'pre_apply_queuing', 'apply_queuing');

CREATE TABLE run_nodes (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL,
    type VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    sort_order INTEGER NOT NULL,
    latest_job_id UUID,
    error_message VARCHAR,
    -- All plan_* columns are nullable: only plan nodes populate them; rows for other node
    -- types (e.g. apply) store NULL ("not applicable") rather than a misleading 0.
    plan_has_changes BOOLEAN,
    plan_diff_size INTEGER,
    plan_resource_additions INTEGER,
    plan_resource_changes INTEGER,
    plan_resource_destructions INTEGER,
    plan_resource_imports INTEGER,
    plan_resource_drift INTEGER,
    plan_output_additions INTEGER,
    plan_output_changes INTEGER,
    plan_output_destructions INTEGER,
    apply_triggered_by VARCHAR,
    apply_comment VARCHAR,
    plan_cache_object_store_key VARCHAR,
    plan_json_object_store_key VARCHAR,
    plan_diff_object_store_key VARCHAR,
    policy_check_type VARCHAR,
    stage_name VARCHAR,
    policy_check_policies JSONB,
    policy_check_messages_summary JSONB,
    -- The managed identities a plan or apply node was authorized to use: the identities assigned to the
    -- workspace when the node's stage passed its managed identity access rules. A job may only create
    -- credentials for an identity on its own node, so an identity assigned mid-run is not usable by that
    -- run. A NULL list authorizes no identities. An empty array means none were authorized.
    managed_identity_ids JSONB,
    apply_triggered_by_actor_type VARCHAR,
    apply_triggered_by_actor_id UUID,
    apply_triggered_by_actor_trn VARCHAR,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
);
-- (run_id, type) also serves run_id-only lookups (including the FK cascade) via its
-- leading column, so no separate run_id index is needed.
CREATE INDEX index_run_nodes_on_run_id_type ON run_nodes(run_id, type);

CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    workspace_id UUID NOT NULL,
    run_id UUID NOT NULL,
    runner_id UUID,
    cancel_requested_at TIMESTAMP,
    queued_at TIMESTAMP,
    pending_at TIMESTAMP,
    running_at TIMESTAMP,
    finished_at TIMESTAMP,
    max_job_duration INTEGER NOT NULL,
    runner_path VARCHAR DEFAULT 'legacy-system-runner',
    tags JSONB NOT NULL DEFAULT '[]',
    properties JSONB NOT NULL DEFAULT '{}',
    force_canceled BOOLEAN NOT NULL DEFAULT FALSE,
    outdated_job_protocol_version BOOLEAN NOT NULL DEFAULT FALSE,
    -- Payload for job types that need one (today only the OPA policy-eval job, which stores its policy
    -- check node id so the runner can report outcomes without scanning the run). Plan/apply jobs leave
    -- it NULL.
    job_data JSONB,
    resource_usage_metrics JSONB,
    -- The configured resource ceilings the job ran under; NULL when no limits were configured.
    resource_usage_limits JSONB,
    -- Opaque dispatcher-specific data for the job's runtime (e.g. a Kubernetes pod name or Docker
    -- container ID) as a JSON string map. NULL until the runner reports a successful dispatch.
    dispatcher_data JSONB,
    -- Leases a job to a single runner's cleanup poller (FOR UPDATE SKIP LOCKED); a stale claim from a
    -- crashed runner is reclaimable once the lease lapses.
    cleanup_claimed_at TIMESTAMP,
    -- Stamped once a runner confirms the job's runtime was torn down.
    cleanup_completed_at TIMESTAMP,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE SET NULL
);
CREATE INDEX index_jobs_on_run_id_and_type ON jobs(run_id, type);
CREATE INDEX index_jobs_on_workspace_id ON jobs(workspace_id);
CREATE INDEX index_jobs_on_tags ON jobs USING GIN (tags jsonb_path_ops);
-- Serves the job dispatch query, which filters by status and orders by created_at, id.
CREATE INDEX index_jobs_on_status_created_at ON jobs(status, created_at, id);
-- Serves GetJobCountForRunner and the fk_runner_id ON DELETE SET NULL scan.
CREATE INDEX index_jobs_on_runner_id_status ON jobs(runner_id, status);
-- Matches a policy check's OPA evaluation jobs on the policy check id stored in job_data.
CREATE INDEX index_jobs_on_policy_check_id
    ON jobs ((job_data->>'policyCheckID'))
    WHERE (job_data->>'policyCheckID') IS NOT NULL;
-- The cleanup poller only scans dispatched, final, not-yet-cleaned jobs.
CREATE INDEX index_jobs_on_cleanup ON jobs(runner_id, created_at)
    WHERE dispatcher_data IS NOT NULL AND cleanup_completed_at IS NULL
      AND status IN ('finished', 'failed', 'canceled');

CREATE TABLE state_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    run_id UUID,
    created_by VARCHAR NOT NULL,
    object_store_key VARCHAR,
    -- The `terraform show -json` rendering of the state version, stored as a separate object from the
    -- raw state file. NULL when the client did not upload a rendering.
    json_object_store_key VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
);
CREATE INDEX index_state_versions_on_run_id ON state_versions(run_id);
CREATE INDEX index_state_versions_on_workspace_id ON state_versions(workspace_id);

ALTER TABLE workspaces ADD CONSTRAINT fk_current_state_version_id FOREIGN KEY(current_state_version_id) REFERENCES state_versions(id) ON DELETE SET NULL,
ADD CONSTRAINT fk_current_apply_run_id FOREIGN KEY(current_apply_run_id) REFERENCES runs(id) ON DELETE SET NULL;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    username VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    admin BOOLEAN NOT NULL,
    scim_external_id UUID,
    active BOOLEAN NOT NULL DEFAULT true,
    password_hash BYTEA,
    admin_mode_expiration TIMESTAMP,
    -- The issuer URL of the IDP whose SCIM client manages the user. A SCIM client may only see, update
    -- or delete users of its own IDP.
    scim_idp_issuer_url VARCHAR
);
CREATE UNIQUE INDEX index_users_on_username ON users(username);
CREATE UNIQUE INDEX index_users_on_email ON users(email);
-- A SCIM external ID is only unique within the IDP whose SCIM client manages the user. Rows managed by
-- no IDP (scim_idp_issuer_url NULL) don't conflict, since NULLs are distinct.
CREATE UNIQUE INDEX index_users_on_scim_idp_issuer_url_scim_external_id ON users(scim_idp_issuer_url, scim_external_id);

CREATE TABLE user_external_identities (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    issuer VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_user_external_identities_on_issuer_external_id ON user_external_identities(issuer, external_id);
CREATE INDEX index_user_external_identities_on_user_id ON user_external_identities(user_id);

CREATE TABLE namespaces (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    path VARCHAR NOT NULL UNIQUE,
    group_id UUID UNIQUE,
    workspace_id UUID UNIQUE,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
-- Lets prefix LIKE queries (namespaces.path LIKE 'some/path/%') use an index range scan. The UNIQUE
-- index on path uses the database collation and cannot serve prefix LIKE.
CREATE INDEX index_namespaces_on_path_pattern ON namespaces (path varchar_pattern_ops);

CREATE TABLE service_accounts (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    group_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    oidc_trust_policies JSON NOT NULL,
    client_secret_hash VARCHAR,
    client_secret_expires_at TIMESTAMP,
    secret_expiration_email_sent_at TIMESTAMP,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_service_accounts_on_name ON service_accounts(name, group_id);
CREATE INDEX index_service_accounts_on_group_id ON service_accounts(group_id);

CREATE TABLE teams (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    scim_external_id UUID,
    -- The issuer URL of the IDP whose SCIM client manages the team. A SCIM client may only see, update,
    -- delete or adopt teams of its own IDP.
    scim_idp_issuer_url VARCHAR
    -- TODO: Eventually, add an org_id and the trimmings
);
CREATE UNIQUE INDEX index_teams_on_name ON teams(name);
-- A SCIM external ID is only unique within the IDP whose SCIM client manages the team.
CREATE UNIQUE INDEX index_teams_on_scim_idp_issuer_url_scim_external_id ON teams(scim_idp_issuer_url, scim_external_id);

CREATE TABLE team_members (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    team_id UUID NOT NULL,
    is_maintainer BOOLEAN NOT NULL,
    CONSTRAINT fk_team_members_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_team_members_on_user_id_team_id ON team_members(user_id, team_id);
CREATE INDEX index_team_members_on_team_id ON team_members(team_id);

CREATE TABLE managed_identities (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    type VARCHAR,
    group_id UUID NOT NULL,
    data VARCHAR,
    created_by VARCHAR NOT NULL,
    alias_source_id UUID,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_alias_source_id FOREIGN KEY(alias_source_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    CONSTRAINT if_alias_source_id_is_null_require_not_null CHECK((alias_source_id IS NOT NULL) OR ((type IS NOT NULL) AND (data is NOT NULL)))
);
CREATE UNIQUE INDEX index_managed_identities_on_name ON managed_identities(name, group_id);
CREATE INDEX index_managed_identities_on_group_id ON managed_identities(group_id);
CREATE INDEX index_managed_identities_on_alias_source_id ON managed_identities(alias_source_id);

CREATE TABLE managed_identity_rules (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    managed_identity_id UUID NOT NULL,
    run_stage VARCHAR NOT NULL,
    type VARCHAR NOT NULL DEFAULT 'eligible_principals',
    module_attestation_policies JSONB,
    verify_state_lineage BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_managed_identity_id FOREIGN KEY(managed_identity_id) REFERENCES managed_identities(id) ON DELETE CASCADE
);
CREATE INDEX index_managed_identity_rules_on_managed_identity_id ON managed_identity_rules(managed_identity_id);

CREATE TABLE managed_identity_rule_allowed_users (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX index_managed_identity_rule_allowed_users_on_rule_id ON managed_identity_rule_allowed_users(rule_id);
CREATE INDEX index_managed_identity_rule_allowed_users_on_user_id ON managed_identity_rule_allowed_users(user_id);

CREATE TABLE managed_identity_rule_allowed_service_accounts (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);
CREATE INDEX index_managed_identity_rule_allowed_service_accounts_on_rule_id ON managed_identity_rule_allowed_service_accounts(rule_id);
CREATE INDEX index_managed_identity_rule_allowed_service_accounts_on_sa_id ON managed_identity_rule_allowed_service_accounts(service_account_id);

CREATE TABLE managed_identity_rule_allowed_teams (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE INDEX index_managed_identity_rule_allowed_teams_on_rule_id ON managed_identity_rule_allowed_teams(rule_id);
CREATE INDEX index_managed_identity_rule_allowed_teams_on_team_id ON managed_identity_rule_allowed_teams(team_id);

CREATE TABLE workspace_managed_identity_relation (
    managed_identity_id UUID,
    workspace_id UUID,
    CONSTRAINT fk_managed_identity_id FOREIGN KEY(managed_identity_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    PRIMARY KEY(managed_identity_id, workspace_id)
);
CREATE INDEX index_workspace_managed_identity_relation_on_workspace_id ON workspace_managed_identity_relation(workspace_id);

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    permissions JSONB NOT NULL,
    -- Fixed display position for the default roles when no explicit sort is requested. Custom roles
    -- default to 100, sorting after all default roles.
    sort_order SMALLINT NOT NULL DEFAULT 100,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR
);
CREATE UNIQUE INDEX index_roles_on_name ON roles(name);
CREATE INDEX index_roles_on_sort_order ON roles(sort_order);

-- The permissions column is empty for the default roles because Role.GetPermissions() bypasses the
-- stored column for any DefaultRoleID and resolves from the Go-side map instead (see models/role.go).
INSERT INTO roles
(id, version, created_at, updated_at, created_by, name, description, permissions, sort_order, created_by_actor_type)
VALUES
    ('52da70fd-37b0-4349-bb64-fb4659bcf5f5', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'viewer', 'Allows read-only access to all resources.', '[]', 0, 'system'),
    ('028fa46b-23ba-443f-a24f-61edcde148ff', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'publisher', 'Allows publishing Terraform modules and providers.', '[]', 1, 'system'),
    ('8aa7adba-b769-471f-8ebb-3215f33991cb', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'deployer', 'Allows managing most resources and deploying runs, but only has view access to namespace memberships, managed identities, runners, and policies.', '[]', 2, 'system'),
    ('d1d61904-1255-4e4b-a5ee-4b9543e970b4', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'maintainer', 'Allows managing all resources, but cannot add, change, or remove namespace memberships.', '[]', 3, 'system'),
    ('623c83ea-23fe-4de6-874a-a99ccf6a76fc', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'owner', 'Allows managing all resources, including namespace memberships.', '[]', 4, 'system');

CREATE TABLE namespace_memberships (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    namespace_id UUID NOT NULL,
    user_id UUID,
    service_account_id UUID,
    team_id UUID,
    role_id UUID NOT NULL,
    CONSTRAINT fk_namespace_memberships_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_role_id FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_memberships_on_user_id_namespace_id ON namespace_memberships(user_id, namespace_id);
CREATE UNIQUE INDEX index_namespace_memberships_on_service_account_id_namespace_id ON namespace_memberships(service_account_id, namespace_id);
CREATE UNIQUE INDEX index_namespace_memberships_on_team_id_namespace_id ON namespace_memberships(team_id, namespace_id);
CREATE INDEX index_namespace_memberships_on_namespace_id ON namespace_memberships(namespace_id);
CREATE INDEX index_namespace_memberships_on_role_id ON namespace_memberships(role_id);

CREATE TABLE namespace_variables (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    namespace_id UUID NOT NULL,
    key VARCHAR NOT NULL,
    value VARCHAR,
    category VARCHAR NOT NULL,
    hcl BOOLEAN,
    sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_variables_on_namespace_id_category_key ON namespace_variables(namespace_id, category, key);

CREATE TABLE runner_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    runner_id UUID NOT NULL,
    last_contacted_at TIMESTAMP NOT NULL,
    error_count INTEGER NOT NULL,
    internal BOOLEAN NOT NULL,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE CASCADE
);
CREATE INDEX index_runner_sessions_on_runner_id ON runner_sessions(runner_id);

CREATE TABLE log_streams (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    job_id UUID,
    size INTEGER NOT NULL,
    runner_session_id UUID,
    completed BOOLEAN NOT NULL DEFAULT TRUE,
    -- Set when a stream is truncated at the server-authoritative maximum log size.
    truncated BOOLEAN NOT NULL DEFAULT FALSE,
    -- Set once a stream's logs live in the single static-key consolidated object (logstreams/{id}.txt).
    compacted BOOLEAN NOT NULL DEFAULT FALSE,
    -- Set when an instance begins compacting a stream. Gates compaction so multiple horizontally-scaled
    -- instances don't compact the same stream concurrently (a stale value is reclaimable after a TTL).
    compaction_started_at TIMESTAMP,
    object_store_key VARCHAR,
    CONSTRAINT fk_job_id FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_log_streams_runner_session_id FOREIGN KEY(runner_session_id) REFERENCES runner_sessions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_log_streams_on_job_id ON log_streams(job_id);
CREATE UNIQUE INDEX index_log_streams_on_runner_session_id ON log_streams(runner_session_id);
-- Finds completed, uncompacted log streams that are eligible for compaction. The IS TRUE / IS FALSE
-- predicate matches the conditions goqu generates for ClaimLogStreamsForCompaction.
CREATE INDEX index_log_streams_claimable_for_compaction
    ON log_streams (updated_at)
    WHERE completed IS TRUE AND compacted IS FALSE;

-- Job/runner-session logs are split across multiple object-store files ("chunks"). Each chunk
-- references one object-store file and records its absolute offset within the stream.
CREATE TABLE log_stream_chunks (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    log_stream_id UUID NOT NULL,
    chunk_index INTEGER NOT NULL,
    start_offset INTEGER NOT NULL,
    size INTEGER NOT NULL,
    -- The NOT NULL constraint keeps the name it had before this column was renamed from object_key,
    -- so the schema matches databases migrated through that rename.
    object_store_key VARCHAR CONSTRAINT log_stream_chunks_object_key_not_null NOT NULL,
    sealed BOOLEAN NOT NULL DEFAULT FALSE,
    -- Write lease: set while a WriteLogs request rewrites the chunk's object, so a concurrent writer
    -- can't overwrite it with a stale prefix. NULL (or a time in the past) means the chunk is free to
    -- claim.
    lease_expires_at TIMESTAMP,
    -- CRC-32 (IEEE) of the chunk's committed bytes [0, size). NULL for chunks that are not verified.
    checksum BIGINT,
    CONSTRAINT fk_log_stream_chunks_log_stream_id FOREIGN KEY(log_stream_id) REFERENCES log_streams(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_log_stream_chunks_on_stream_id_and_index ON log_stream_chunks(log_stream_id, chunk_index);
CREATE INDEX index_log_stream_chunks_on_stream_id_and_offset ON log_stream_chunks(log_stream_id, start_offset);

CREATE TABLE state_version_outputs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    value VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    sensitive BOOLEAN NOT NULL,
    state_version_id UUID NOT NULL,
    CONSTRAINT fk_state_version_id FOREIGN KEY(state_version_id) REFERENCES state_versions(id) ON DELETE CASCADE
);
CREATE INDEX index_state_version_outputs_on_name ON state_version_outputs(name);
CREATE INDEX index_state_version_outputs_on_state_version_id ON state_version_outputs(state_version_id);

CREATE TABLE gpg_keys (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID NOT NULL,
    gpg_key_id BIGINT NOT NULL,
    fingerprint VARCHAR NOT NULL,
    ascii_armor VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_gpg_keys_on_key_id ON gpg_keys(group_id, fingerprint);

CREATE TABLE terraform_providers (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID NOT NULL,
    root_group_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    private BOOLEAN NOT NULL,
    repo_url VARCHAR NOT NULL DEFAULT '',
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_providers_on_name ON terraform_providers(root_group_id, name);
CREATE INDEX index_terraform_providers_on_group_id ON terraform_providers(group_id);

CREATE TABLE terraform_provider_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    provider_id UUID NOT NULL,
    provider_sem_version VARCHAR NOT NULL,
    gpg_key_id BIGINT,
    gpg_ascii_armor VARCHAR,
    protocols JSON NOT NULL,
    sha_sums_uploaded BOOLEAN NOT NULL,
    sha_sums_sig_uploaded BOOLEAN NOT NULL,
    created_by VARCHAR NOT NULL,
    readme_uploaded BOOLEAN NOT NULL DEFAULT false,
    latest BOOLEAN NOT NULL DEFAULT false,
    readme_object_store_key VARCHAR,
    shasums_object_store_key VARCHAR,
    shasums_signature_object_store_key VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_provider_id FOREIGN KEY(provider_id) REFERENCES terraform_providers(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_versions_on_version ON terraform_provider_versions(provider_id, provider_sem_version);
CREATE UNIQUE INDEX index_terraform_provider_versions_on_latest ON terraform_provider_versions(provider_id, latest) WHERE latest = true;
CREATE INDEX index_terraform_provider_versions_on_provider_id_created_at ON terraform_provider_versions(provider_id, created_at DESC);

CREATE TABLE terraform_provider_platforms (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    provider_version_id UUID NOT NULL,
    os VARCHAR NOT NULL,
    arch VARCHAR NOT NULL,
    sha_sum VARCHAR NOT NULL,
    filename VARCHAR NOT NULL,
    binary_uploaded BOOLEAN NOT NULL,
    created_by VARCHAR NOT NULL,
    binary_object_store_key VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_provider_version_id FOREIGN KEY(provider_version_id) REFERENCES terraform_provider_versions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_platforms_on_os_arch ON terraform_provider_platforms(provider_version_id, os, arch);

CREATE TABLE scim_tokens (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    nonce UUID NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR
);
CREATE UNIQUE INDEX index_scim_tokens_on_nonce ON scim_tokens(nonce);

CREATE TABLE vcs_providers (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    type VARCHAR NOT NULL,
    oauth_client_id VARCHAR NOT NULL,
    oauth_client_secret VARCHAR NOT NULL,
    oauth_state UUID,
    oauth_access_token VARCHAR,
    oauth_refresh_token VARCHAR,
    oauth_access_token_expires_at TIMESTAMP,
    auto_create_webhooks BOOLEAN NOT NULL,
    group_id UUID NOT NULL,
    url VARCHAR NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_vcs_providers_on_name_group_id ON vcs_providers(name, group_id);
CREATE INDEX index_vcs_providers_on_group_id ON vcs_providers(group_id);

CREATE TABLE workspace_vcs_provider_links (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    workspace_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    token_nonce UUID NOT NULL,
    repository_path VARCHAR NOT NULL,
    auto_speculative_plan BOOLEAN NOT NULL,
    webhook_id VARCHAR,
    module_directory VARCHAR,
    branch VARCHAR NOT NULL,
    tag_regex VARCHAR,
    glob_patterns JSON,
    webhook_disabled BOOLEAN NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_provider_id FOREIGN KEY(provider_id) REFERENCES vcs_providers(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_workspace_vcs_provider_links_on_workspace_id ON workspace_vcs_provider_links(workspace_id);
CREATE INDEX index_workspace_vcs_provider_links_on_provider_id ON workspace_vcs_provider_links(provider_id);

CREATE TABLE terraform_modules (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID NOT NULL,
    root_group_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    repo_url VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    system VARCHAR NOT NULL,
    private BOOLEAN NOT NULL,
    labels JSONB DEFAULT '{}',
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_modules_on_name_system ON terraform_modules(root_group_id, name, system);
CREATE INDEX index_terraform_modules_labels ON terraform_modules USING GIN (labels jsonb_ops);
CREATE INDEX index_terraform_modules_on_group_id ON terraform_modules(group_id);

CREATE TABLE terraform_module_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    module_id UUID NOT NULL,
    semantic_version VARCHAR NOT NULL,
    latest BOOLEAN NOT NULL,
    submodules JSONB,
    examples JSONB,
    status VARCHAR NOT NULL,
    error VARCHAR,
    diagnostics VARCHAR,
    upload_started_at TIMESTAMP,
    sha_sum bytea NOT NULL,
    created_by VARCHAR NOT NULL,
    package_object_store_key VARCHAR,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_module_id FOREIGN KEY(module_id) REFERENCES terraform_modules(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_module_versions_on_semantic_version ON terraform_module_versions(module_id, semantic_version);
CREATE UNIQUE INDEX index_terraform_module_versions_on_latest ON terraform_module_versions(module_id, latest) WHERE latest = true;
CREATE INDEX index_terraform_module_versions_on_module_id_created_at ON terraform_module_versions(module_id, created_at DESC);

CREATE TABLE terraform_module_attestations (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    description VARCHAR,
    module_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    schema_type VARCHAR NOT NULL,
    predicate_type VARCHAR NOT NULL,
    digests JSONB NOT NULL,
    data VARCHAR NOT NULL,
    data_sha_sum bytea NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_module_id FOREIGN KEY(module_id) REFERENCES terraform_modules(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX index_terraform_module_attestations_on_module_and_data_sha_sum ON terraform_module_attestations(module_id, data_sha_sum);
CREATE INDEX index_terraform_module_attestations_on_digests ON terraform_module_attestations USING GIN (digests jsonb_path_ops);

CREATE TABLE service_account_runner_relation (
    runner_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    PRIMARY KEY(runner_id, service_account_id)
);
CREATE INDEX index_service_account_runner_relation_on_service_account_id ON service_account_runner_relation(service_account_id);

CREATE TABLE terraform_provider_version_mirrors (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    semantic_version VARCHAR NOT NULL,
    registry_namespace VARCHAR NOT NULL,
    registry_hostname VARCHAR NOT NULL,
    digests JSONB,
    group_id UUID NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_version_mirrors_on_type_version_namespace_hostname ON terraform_provider_version_mirrors(type, semantic_version, registry_namespace, registry_hostname, group_id);
CREATE INDEX index_terraform_provider_version_mirrors_on_group_id ON terraform_provider_version_mirrors(group_id);

CREATE TABLE federated_registries (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    hostname VARCHAR NOT NULL,
    audience VARCHAR NOT NULL,
    group_id UUID NOT NULL,
    created_by VARCHAR NOT NULL DEFAULT '',
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_federated_registries_on_hostname_group_id ON federated_registries(hostname, group_id);
CREATE INDEX index_federated_registries_on_group_id ON federated_registries(group_id);

CREATE TABLE resource_limits (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    value INTEGER NOT NULL
);
CREATE UNIQUE INDEX index_resource_limits_on_name ON resource_limits(name);

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
    ('04c35a50-303d-42c4-bade-c5c4d4da5ac3', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitSubgroupsPerParent', 1000), -- number of subgroups directly under one parent group
    ('e626308b-dc6a-4f5c-a8f9-c90223579cc2', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitGroupTreeDepth', 20), -- depth of the group tree
    ('e923f667-3dd1-4376-a973-1d5002249c65', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitWorkspacesPerGroup', 1000), -- number of workspaces directly under one group
    ('0b5e750d-30d8-462f-96b4-7c7730167857', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitServiceAccountsPerGroup', 1000), -- number of service accounts per group
    ('1a796b82-ff6e-4ed6-a135-66e3d987f1de', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunnerAgentsPerGroup', 1000), -- number of runner agents per group
    ('12b6c63f-d189-47c5-b9f2-c9c799c171b0', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVariablesPerNamespace', 1000), -- number of variables per group or workspace
    ('11f19e67-5e50-46a0-9e45-3d0fc6ac6c5b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitGPGKeysPerGroup', 1000), -- number of GPG keys per group
    ('86e34c1a-24af-4ff9-86d8-e541768953b5', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentitiesPerGroup', 1000), -- number of managed identities per group
    ('6b99fe91-91eb-4375-917f-511a30ac7ad9', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentityAliasesPerManagedIdentity', 1000), -- number of managed identity aliases per managed identity
    ('87fe08b7-7ed1-4688-ae21-3b17dfaad198', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAssignedManagedIdentitiesPerWorkspace', 1000), -- number of assigned managed identities per workspace
    ('c22c7257-1c87-4dbd-9ace-0830837597c7', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentityAccessRulesPerManagedIdentity', 1000), -- number of managed identity access rules per managed identity
    ('ab1d78ae-f726-4486-b0fa-381e5507d987', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformModulesPerGroup', 1000), -- number of Terraform modules per group
    ('fd759edd-12d7-4960-9384-2cfa48353c51', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformProvidersPerGroup', 1000), -- number of Terraform providers per group
    ('6fadb906-ef65-411c-9f20-60b7c5031d71', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPlatformsPerTerraformProviderVersion', 1000), -- number of platforms per Terraform provider version
    ('efd32ed2-b2f5-40a2-a207-6b18eaa06c86', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVCSProvidersPerGroup', 1000), -- number of VCS providers per group
    ('1d26d247-4323-4ed4-adca-94516e5cf4f9', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformProviderVersionMirrorsPerGroup', 1000), -- number of terraform provider version mirrors per group
    ('02baa4c6-5e83-4c18-95c7-9cc70eeb0417', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunnerSessionsPerRunner', 100), -- number of active runner sessions per runner
    ('246822db-b982-45e6-8cd4-18b5cadf83a3', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunsPerWorkspacePerTimePeriod', 100), -- number of runs per workspace per time period
    ('3e479306-6a52-4d47-a14c-2488464a2ad4', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod', 100), -- number of configuration versions per workspace per time period
    ('9388d4a3-ed04-4ce4-899f-d1dfd4187834', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitStateVersionsPerWorkspacePerTimePeriod', 100), -- number of state versions per workspace per time period
    ('1ac2099c-6c8d-48b8-bbf1-a618474dc07b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformModulePerTimePeriod', 100), -- number of versions per Terraform module per time period
    ('fdc8fba1-3dfc-4aa5-b8bb-ca38eaf42f0b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAttestationsPerTerraformModulePerTimePeriod', 100), -- number of attestations per Terraform module per time period
    ('87243e0a-d4c9-4b4c-8104-f7b1b37bda49', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformProviderPerTimePeriod', 100), -- number of versions per Terraform provider per time period
    ('de6fbc64-92b0-4e3c-aadf-a7c64e814bed', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitFederatedRegistriesPerGroup', 100), -- number of federated registries per group
    ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAgentCreditsPerUserPerMonth', 5000), -- number of agent credits per user per month
    ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAgentSessionRunsPerSession', 50), -- number of agent session runs per session
    ('1138c015-adb1-4bce-8707-661981d65b2f', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitOutputsPerStateVersion', 400), -- max number of Terraform outputs per state version
    ('887b98dc-2afe-4ba3-b938-2b902c56198a', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPackagesPerGroup', 1000), -- number of packages per group
    ('3b021da6-ad87-4b04-a75d-ddc202d70bd8', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPoliciesPerGroup', 100), -- number of policies per group or workspace
    ('b2ba9fb6-06a7-45e1-a82b-9140aba91cb6', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerPackagePerTimePeriod', 100); -- number of package versions per package per time period

CREATE TABLE terraform_provider_platform_mirrors (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    os VARCHAR NOT NULL,
    architecture VARCHAR NOT NULL,
    version_mirror_id UUID NOT NULL,
    object_store_key VARCHAR,
    CONSTRAINT fk_version_mirror_id FOREIGN KEY (version_mirror_id) REFERENCES terraform_provider_version_mirrors(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_platform_mirrors_on_os_arch ON terraform_provider_platform_mirrors(version_mirror_id, os, architecture);

CREATE TABLE maintenance_mode (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR
);

CREATE TABLE namespace_variable_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    variable_id UUID NOT NULL,
    key VARCHAR NOT NULL,
    value VARCHAR,
    hcl BOOLEAN NOT NULL,
    secret_data bytea,
    CONSTRAINT fk_variable_id FOREIGN KEY(variable_id) REFERENCES namespace_variables(id) ON DELETE CASCADE
);
CREATE INDEX index_namespace_variable_versions_on_variable_id ON namespace_variable_versions(variable_id);

CREATE TABLE latest_namespace_variable_versions (
    variable_id UUID NOT NULL,
    version_id UUID NOT NULL,
    CONSTRAINT fk_variable_id FOREIGN KEY(variable_id) REFERENCES namespace_variables(id) ON DELETE CASCADE,
    CONSTRAINT fk_version_id FOREIGN KEY(version_id) REFERENCES namespace_variable_versions(id) ON DELETE CASCADE,
    PRIMARY KEY(variable_id, version_id)
);
CREATE INDEX index_latest_namespace_variable_versions_on_version_id ON latest_namespace_variable_versions(version_id);

CREATE TABLE workspace_assessments (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    has_drift BOOLEAN NOT NULL,
    requires_notification BOOLEAN NOT NULL,
    completed_run_id UUID,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_completed_run_id FOREIGN KEY(completed_run_id) REFERENCES runs(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX index_workspace_assessments_on_workspace_id ON workspace_assessments(workspace_id);
CREATE INDEX index_workspace_assessments_on_completed_at ON workspace_assessments(completed_at);
CREATE INDEX index_workspace_assessments_on_completed_run_id ON workspace_assessments(completed_run_id);
-- Serves the updated_at branch of the assessment scheduler's GetWorkspaces OR-predicate.
CREATE INDEX index_workspace_assessments_on_updated_at ON workspace_assessments (updated_at);

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    namespace_id UUID,
    scope VARCHAR(255) NOT NULL,
    custom_events JSONB,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_user_id_namespace_id ON notification_preferences(user_id, namespace_id) NULLS NOT DISTINCT;
CREATE INDEX index_notification_preferences_on_namespace_id ON notification_preferences(namespace_id);

CREATE TABLE announcements (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    message TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    created_by VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    dismissible BOOLEAN NOT NULL DEFAULT false,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR
);
CREATE INDEX index_announcements_time_range ON announcements (start_time, end_time);

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    user_agent VARCHAR NOT NULL,
    refresh_token_id UUID NOT NULL,
    expiration TIMESTAMP NOT NULL,
    oauth_code VARCHAR,
    oauth_code_challenge VARCHAR,
    oauth_code_challenge_method VARCHAR,
    oauth_code_expiration TIMESTAMP,
    oauth_redirect_uri VARCHAR,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX index_user_sessions_on_refresh_token_id ON user_sessions(refresh_token_id);
CREATE INDEX index_user_sessions_on_user_id_refresh_token_id ON user_sessions(user_id, refresh_token_id);
CREATE INDEX index_user_sessions_on_user_id_expiration ON user_sessions(user_id, expiration);
CREATE UNIQUE INDEX index_user_sessions_on_oauth_code ON user_sessions(oauth_code);

CREATE TABLE asym_signing_keys (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    public_key BYTEA,
    plugin_data BYTEA,
    plugin_type VARCHAR NOT NULL,
    pub_key_id VARCHAR NOT NULL,
    status VARCHAR NOT NULL
);
CREATE UNIQUE INDEX index_asym_signing_keys_on_status ON asym_signing_keys (status) WHERE (status = 'active' OR status = 'creating');

CREATE TABLE namespace_favorites (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    group_id UUID,
    workspace_id UUID,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_favorites_on_user_id_group_id ON namespace_favorites(user_id, group_id) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX index_namespace_favorites_on_user_id_workspace_id ON namespace_favorites(user_id, workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX index_namespace_favorites_on_group_id ON namespace_favorites(group_id);
CREATE INDEX index_namespace_favorites_on_workspace_id ON namespace_favorites(workspace_id);

CREATE TABLE agent_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    total_credits DOUBLE PRECISION NOT NULL DEFAULT 0,
    CONSTRAINT fk_agent_sessions_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX index_agent_sessions_on_user_id ON agent_sessions(user_id);

CREATE TABLE agent_session_runs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    session_id UUID NOT NULL,
    previous_run_id UUID,
    last_message_id UUID,
    status TEXT NOT NULL,
    error_message TEXT,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_agent_session_runs_session_id
        FOREIGN KEY(session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_runs_previous_run_id
        FOREIGN KEY(previous_run_id) REFERENCES agent_session_runs(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_agent_session_runs_on_previous_run_id ON agent_session_runs(previous_run_id) WHERE previous_run_id IS NOT NULL;
CREATE INDEX index_agent_session_runs_on_session_id_created_at ON agent_session_runs(session_id, created_at, id);

CREATE TABLE agent_session_messages (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    session_id UUID NOT NULL,
    run_id UUID NOT NULL,
    parent_id UUID,
    role TEXT NOT NULL,
    content JSONB,
    tool_content_object_store_key VARCHAR,
    CONSTRAINT fk_agent_session_messages_session_id
        FOREIGN KEY(session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_messages_run_id
        FOREIGN KEY(run_id) REFERENCES agent_session_runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_messages_parent_id
        FOREIGN KEY(parent_id) REFERENCES agent_session_messages(id) ON DELETE SET NULL
);
CREATE INDEX index_agent_session_messages_on_session_id ON agent_session_messages(session_id);
CREATE INDEX index_agent_session_messages_on_run_id_created_at ON agent_session_messages(run_id, created_at);
CREATE INDEX index_agent_session_messages_on_parent_id ON agent_session_messages(parent_id);

CREATE TABLE agent_credit_quotas (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    month_date DATE NOT NULL,
    total_credits DOUBLE PRECISION NOT NULL DEFAULT 0,
    CONSTRAINT fk_agent_credit_quotas_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_agent_credit_quotas_user_month UNIQUE(user_id, month_date)
);
CREATE INDEX index_agent_credit_quotas_on_user_id ON agent_credit_quotas(user_id);

CREATE TABLE work_items_queue (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    available_at TIMESTAMP NOT NULL,
    claim_count INTEGER NOT NULL DEFAULT 0,
    type VARCHAR NOT NULL,
    payload JSONB NOT NULL
);
CREATE INDEX index_work_items_queue_on_type_available_at_created_at ON work_items_queue(type, available_at, created_at);

-- Package registry: group-owned, versioned packages (today OPA policy bundles).
CREATE TABLE packages (
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
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_packages_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_packages_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_packages_on_name ON packages(group_id, name);
CREATE INDEX index_packages_on_root_group_id ON packages(root_group_id);

CREATE TABLE package_versions (
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
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_package_versions_package_id FOREIGN KEY(package_id) REFERENCES packages(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_package_versions_on_semantic_version ON package_versions(package_id, semantic_version);
CREATE UNIQUE INDEX index_package_versions_on_latest ON package_versions(package_id, latest) WHERE latest = true;

-- Policy set policies. Every policy is owned by exactly one group. Scope is a JSONB array of ScopeRule
-- objects controlling which runs the policy applies to; an empty array means the policy applies to all
-- workspaces under the owning group.
CREATE TABLE policies (
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
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_policies_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
-- Policy names are unique within an owning group.
CREATE UNIQUE INDEX index_policies_on_group ON policies(group_id, name);

-- Approver principals for soft-mandatory policies (who may approve an override). A principal is an
-- approver for a policy once, so (policy_id, principal) is unique; the composite doubles as the
-- policy_id FK index. The single-column index covers the cascade from a deleted principal.
CREATE TABLE policy_allowed_users (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_users_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_users_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_policy_allowed_users_on_policy_id_user_id ON policy_allowed_users(policy_id, user_id);
CREATE INDEX index_policy_allowed_users_on_user_id ON policy_allowed_users(user_id);

CREATE TABLE policy_allowed_service_accounts (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_sas_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_sas_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_policy_allowed_service_accounts_on_policy_id_sa_id ON policy_allowed_service_accounts(policy_id, service_account_id);
CREATE INDEX index_policy_allowed_service_accounts_on_sa_id ON policy_allowed_service_accounts(service_account_id);

CREATE TABLE policy_allowed_teams (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_policy_allowed_teams_policy_id FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_policy_allowed_teams_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_policy_allowed_teams_on_policy_id_team_id ON policy_allowed_teams(policy_id, team_id);
CREATE INDEX index_policy_allowed_teams_on_team_id ON policy_allowed_teams(team_id);

-- Run gates: one gate per run policy check that parks at awaiting_override because one or more
-- soft-mandatory policies failed. type is the kind of policy check the gate governs (e.g. "opa_policy").
-- approval_rules is a JSONB list with one entry per soft-failed policy: {name, requiredApprovals,
-- allowedSubjects}, where name is the run's PolicyCheckPolicy ID (no FK — the run's JSONB snapshot is the
-- source of truth). workspace_id is denormalized from the run so gates can be filtered by namespace
-- membership without joining through runs.
CREATE TABLE run_gates (
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
    overridden_by_actor_type VARCHAR,
    overridden_by_actor_id UUID,
    overridden_by_actor_trn VARCHAR,
    CONSTRAINT fk_run_gates_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gates_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX index_run_gates_on_run_id ON run_gates(run_id);
CREATE INDEX index_run_gates_on_workspace_id ON run_gates(workspace_id);
-- One gate per policy-check node, enforced rather than assumed.
CREATE UNIQUE INDEX index_run_gates_on_policy_check_id ON run_gates(policy_check_id);

-- Query-index tables derived from the gate snapshot at creation, so the approvals inbox can efficiently
-- find gates where a principal is an eligible approver. One row per (gate, principal). The
-- single-column index serves the inbox lookup by principal; the composite serves the gate_id cascade.
CREATE TABLE run_gate_allowed_users (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_users_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_users_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX index_run_gate_allowed_users_on_user_id ON run_gate_allowed_users(user_id);
CREATE UNIQUE INDEX index_run_gate_allowed_users_on_gate_id_user_id ON run_gate_allowed_users(gate_id, user_id);

CREATE TABLE run_gate_allowed_service_accounts (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_service_accounts_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_service_accounts_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);
CREATE INDEX index_run_gate_allowed_service_accounts_on_sa_id ON run_gate_allowed_service_accounts(service_account_id);
CREATE UNIQUE INDEX index_run_gate_allowed_service_accounts_on_gate_id_sa_id ON run_gate_allowed_service_accounts(gate_id, service_account_id);

CREATE TABLE run_gate_allowed_teams (
    id UUID PRIMARY KEY,
    gate_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_run_gate_allowed_teams_gate_id FOREIGN KEY(gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_allowed_teams_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);
CREATE INDEX index_run_gate_allowed_teams_on_team_id ON run_gate_allowed_teams(team_id);
CREATE UNIQUE INDEX index_run_gate_allowed_teams_on_gate_id_team_id ON run_gate_allowed_teams(gate_id, team_id);

CREATE TABLE run_gate_approvals (
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
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_run_gate_approvals_run_gate_id FOREIGN KEY(run_gate_id) REFERENCES run_gates(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_gate_approvals_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_run_gate_approvals_sa_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE SET NULL
);
CREATE INDEX index_run_gate_approvals_on_run_gate_id ON run_gate_approvals(run_gate_id);
-- The approver FKs are ON DELETE SET NULL, so deleting a user or service account has to find their
-- decisions. Partial: a decision is made by a user or a service account, never both.
CREATE INDEX index_run_gate_approvals_on_user_id ON run_gate_approvals(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX index_run_gate_approvals_on_sa_id ON run_gate_approvals(service_account_id) WHERE service_account_id IS NOT NULL;
-- A principal decides once per gate: without these, a duplicate row could inflate a rule's approval
-- count past what its approver list can ever provide and clear a gate that should still be waiting.
CREATE UNIQUE INDEX index_run_gate_approvals_on_gate_id_user_id ON run_gate_approvals(run_gate_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX index_run_gate_approvals_on_gate_id_sa_id ON run_gate_approvals(run_gate_id, service_account_id) WHERE service_account_id IS NOT NULL;

CREATE TABLE cleanup_policies (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID,
    workspace_id UUID,
    disabled BOOLEAN NOT NULL,
    kind VARCHAR NOT NULL,
    kind_data JSONB NOT NULL,
    sweep_claimed_at TIMESTAMP,
    sweep_cursor VARCHAR,
    last_sweep_completed_at TIMESTAMP,
    CONSTRAINT fk_cleanup_policies_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_cleanup_policies_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_cleanup_policies_on_group_id_kind ON cleanup_policies(group_id, kind) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX index_cleanup_policies_on_workspace_id_kind ON cleanup_policies(workspace_id, kind) WHERE workspace_id IS NOT NULL;
CREATE INDEX index_cleanup_policies_on_group_id ON cleanup_policies(group_id);
CREATE INDEX index_cleanup_policies_on_workspace_id ON cleanup_policies(workspace_id);
-- The sweeper claims the policies that are due, the ones never claimed first.
CREATE INDEX index_cleanup_policies_on_sweep_claimed_at ON cleanup_policies(sweep_claimed_at NULLS FIRST);

CREATE TABLE workspace_role_bindings (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    created_by_actor_type VARCHAR,
    created_by_actor_id UUID,
    created_by_actor_trn VARCHAR,
    CONSTRAINT fk_workspace_role_bindings_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    -- RESTRICT rather than CASCADE: silently dropping a binding when its role is deleted would
    -- quietly de-privilege a workspace and break its deployments. Failing the role deletion forces
    -- an admin to unbind first, which surfaces the dependency. Role deletion is admin-only and rare.
    CONSTRAINT fk_workspace_role_bindings_role_id FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE RESTRICT
);
-- One binding per workspace. A binding always applies at the workspace's direct parent namespace, so
-- a second row for the same workspace would have no distinct meaning.
CREATE UNIQUE INDEX index_workspace_role_bindings_on_workspace_id ON workspace_role_bindings(workspace_id);
CREATE INDEX index_workspace_role_bindings_on_role_id ON workspace_role_bindings(role_id);

CREATE TABLE email_outbox_items (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    ephemeral BOOLEAN NOT NULL,
    email_type VARCHAR NOT NULL,
    subject VARCHAR NOT NULL,
    payload BYTEA,
    payload_object_store_key VARCHAR,
    status VARCHAR NOT NULL,
    send_to_all_users BOOLEAN NOT NULL,
    recipient_user_ids JSONB NOT NULL DEFAULT '[]',
    recipient_team_ids JSONB NOT NULL DEFAULT '[]',
    send_at TIMESTAMP,
    claimed_at TIMESTAMP
);
-- The recipient-materialization poller only scans items awaiting materialization.
CREATE INDEX index_email_outbox_items_on_status ON email_outbox_items(created_at)
    WHERE status = 'preparing';
-- The cleanup poller only scans completed ephemeral rows. The IS TRUE predicate matches the condition
-- goqu generates for the email cleaner's ClaimOutboxItems query.
CREATE INDEX index_email_outbox_items_on_ephemeral
    ON email_outbox_items (created_at)
    WHERE ephemeral IS TRUE AND status = 'completed';

CREATE TABLE email_recipients (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email_outbox_item_id UUID NOT NULL,
    address VARCHAR NOT NULL,
    delivery_status VARCHAR NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMP NOT NULL,
    last_attempt_at TIMESTAMP,
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    complained_at TIMESTAMP,
    failure_reason VARCHAR,
    CONSTRAINT fk_email_recipients_email_outbox_item_id FOREIGN KEY(email_outbox_item_id) REFERENCES email_outbox_items(id) ON DELETE CASCADE
);
-- Claim query: retryable rows whose available_at is due, ordered FIFO. available_at drives both
-- scheduling (not claimable until then) and the retry lease (a claim pushes it forward). The predicate
-- must list exactly the claimable statuses so the planner can prove the claim's IN (...) uses this index.
CREATE INDEX index_email_recipients_on_delivery_status_available_at
    ON email_recipients(delivery_status, available_at)
    WHERE delivery_status IN ('pending', 'accepted', 'soft_bounced', 'delayed');
CREATE INDEX index_email_recipients_on_email_outbox_item_id ON email_recipients(email_outbox_item_id);

CREATE TABLE email_suppressions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    address VARCHAR NOT NULL,
    cause VARCHAR NOT NULL
);
CREATE UNIQUE INDEX index_email_suppressions_on_address ON email_suppressions(lower(address));

-- Tracks every object-store object against the resource that owns it, so the janitor can reclaim the
-- object once its owner is deleted (each FK is ON DELETE SET NULL).
CREATE TABLE object_store_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    available_at TIMESTAMP NOT NULL DEFAULT now(),
    claim_count INTEGER NOT NULL DEFAULT 0,
    object_key TEXT NOT NULL,
    run_id UUID REFERENCES runs(id) ON DELETE SET NULL,
    state_version_id UUID REFERENCES state_versions(id) ON DELETE SET NULL,
    configuration_version_id UUID REFERENCES configuration_versions(id) ON DELETE SET NULL,
    log_stream_id UUID REFERENCES log_streams(id) ON DELETE SET NULL,
    log_stream_chunk_id UUID REFERENCES log_stream_chunks(id) ON DELETE SET NULL,
    module_version_id UUID REFERENCES terraform_module_versions(id) ON DELETE SET NULL,
    provider_version_id UUID REFERENCES terraform_provider_versions(id) ON DELETE SET NULL,
    provider_platform_id UUID REFERENCES terraform_provider_platforms(id) ON DELETE SET NULL,
    provider_mirror_platform_id UUID REFERENCES terraform_provider_platform_mirrors(id) ON DELETE SET NULL,
    agent_session_id UUID REFERENCES agent_sessions(id) ON DELETE SET NULL,
    package_version_id UUID REFERENCES package_versions(id) ON DELETE SET NULL,
    email_outbox_item_id UUID REFERENCES email_outbox_items(id) ON DELETE SET NULL,
    CONSTRAINT object_store_refs_unique_key UNIQUE (object_key)
);
CREATE INDEX index_object_store_refs_on_run_id ON object_store_refs(run_id);
CREATE INDEX index_object_store_refs_on_state_version_id ON object_store_refs(state_version_id);
CREATE INDEX index_object_store_refs_on_configuration_version_id ON object_store_refs(configuration_version_id);
CREATE INDEX index_object_store_refs_on_log_stream_id ON object_store_refs(log_stream_id);
CREATE INDEX index_object_store_refs_on_log_stream_chunk_id ON object_store_refs(log_stream_chunk_id);
CREATE INDEX index_object_store_refs_on_module_version_id ON object_store_refs(module_version_id);
CREATE INDEX index_object_store_refs_on_provider_version_id ON object_store_refs(provider_version_id);
CREATE INDEX index_object_store_refs_on_provider_platform_id ON object_store_refs(provider_platform_id);
CREATE INDEX index_object_store_refs_on_provider_mirror_platform_id ON object_store_refs(provider_mirror_platform_id);
CREATE INDEX index_object_store_refs_on_agent_session_id ON object_store_refs(agent_session_id);
CREATE INDEX index_object_store_refs_on_package_version_id ON object_store_refs(package_version_id);
CREATE INDEX index_object_store_refs_on_email_outbox_item_id ON object_store_refs(email_outbox_item_id);
-- Partial index for the janitor's orphan sweep: rows where every FK is NULL are candidates for deletion.
-- The WHERE clause must list every FK column or the janitor would treat a ref owned by the missing
-- column's resource as orphaned.
CREATE INDEX index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL
  AND package_version_id IS NULL AND email_outbox_item_id IS NULL;

CREATE TABLE activity_events (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID,
    service_account_id UUID,
    namespace_id UUID,
    action VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    gpg_key_target_id UUID,
    group_target_id UUID,
    managed_identity_target_id UUID,
    managed_identity_rule_target_id UUID,
    namespace_membership_target_id UUID,
    run_target_id UUID,
    service_account_target_id UUID,
    state_version_target_id UUID,
    team_target_id UUID,
    terraform_provider_target_id UUID,
    terraform_provider_version_target_id UUID,
    variable_target_id UUID,
    workspace_target_id UUID,
    vcs_provider_target_id UUID,
    terraform_module_target_id UUID,
    terraform_module_version_target_id UUID,
    runner_target_id UUID,
    role_target_id UUID,
    payload JSONB,
    terraform_provider_version_mirror_target_id UUID,
    federated_registry_target_id UUID,
    package_target_id UUID,
    package_version_target_id UUID,
    policy_target_id UUID,
    run_gate_target_id UUID,
    cleanup_policy_target_id UUID REFERENCES cleanup_policies(id) ON DELETE CASCADE,
    workspace_role_binding_target_id UUID REFERENCES workspace_role_bindings(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_gpg_key_target_id FOREIGN KEY(gpg_key_target_id) REFERENCES gpg_keys(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_group_target_id FOREIGN KEY(group_target_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_managed_identity_target_id FOREIGN KEY(managed_identity_target_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_managed_identity_rule_target_id FOREIGN KEY(managed_identity_rule_target_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_namespace_membership_target_id FOREIGN KEY(namespace_membership_target_id) REFERENCES namespace_memberships(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_run_target_id FOREIGN KEY(run_target_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_service_account_target_id FOREIGN KEY(service_account_target_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_state_version_target_id FOREIGN KEY(state_version_target_id) REFERENCES state_versions(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_team_target_id FOREIGN KEY(team_target_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_terraform_provider_target_id FOREIGN KEY(terraform_provider_target_id) REFERENCES terraform_providers(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_terraform_provider_version_target_id FOREIGN KEY(terraform_provider_version_target_id) REFERENCES terraform_provider_versions(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_variable_target_id FOREIGN KEY(variable_target_id) REFERENCES namespace_variables(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_workspace_target_id FOREIGN KEY(workspace_target_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_vcs_provider_target_id FOREIGN KEY(vcs_provider_target_id) REFERENCES vcs_providers(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_terraform_module_target_id FOREIGN KEY(terraform_module_target_id) REFERENCES terraform_modules(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_terraform_module_version_target_id FOREIGN KEY(terraform_module_version_target_id) REFERENCES terraform_module_versions(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_runner_target_id FOREIGN KEY(runner_target_id) REFERENCES runners(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_role_target_id FOREIGN KEY(role_target_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_terraform_provider_version_mirror_target_id FOREIGN KEY(terraform_provider_version_mirror_target_id) REFERENCES terraform_provider_version_mirrors(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_federated_registry_target_id FOREIGN KEY(federated_registry_target_id) REFERENCES federated_registries(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_package_target_id FOREIGN KEY(package_target_id) REFERENCES packages(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_package_version_target_id FOREIGN KEY(package_version_target_id) REFERENCES package_versions(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_policy_target_id FOREIGN KEY(policy_target_id) REFERENCES policies(id) ON DELETE CASCADE,
    CONSTRAINT fk_activity_events_run_gate_target_id FOREIGN KEY(run_gate_target_id) REFERENCES run_gates(id) ON DELETE CASCADE
);
CREATE INDEX index_activity_events_on_user_id ON activity_events(user_id);
CREATE INDEX index_activity_events_on_service_account_id ON activity_events(service_account_id);
CREATE INDEX index_activity_events_on_namespace_id ON activity_events(namespace_id);
CREATE INDEX index_activity_events_created_at_id ON activity_events(created_at DESC, id DESC);
-- Target columns use partial indexes since only one target is set per event.
CREATE INDEX index_activity_events_on_gpg_key_target_id ON activity_events(gpg_key_target_id) WHERE gpg_key_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_group_target_id ON activity_events(group_target_id) WHERE group_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_managed_identity_target_id ON activity_events(managed_identity_target_id) WHERE managed_identity_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_managed_identity_rule_target_id ON activity_events(managed_identity_rule_target_id) WHERE managed_identity_rule_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_namespace_membership_target_id ON activity_events(namespace_membership_target_id) WHERE namespace_membership_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_run_target_id ON activity_events(run_target_id) WHERE run_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_service_account_target_id ON activity_events(service_account_target_id) WHERE service_account_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_state_version_target_id ON activity_events(state_version_target_id) WHERE state_version_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_team_target_id ON activity_events(team_target_id) WHERE team_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_terraform_provider_target_id ON activity_events(terraform_provider_target_id) WHERE terraform_provider_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_tf_provider_version_target_id ON activity_events(terraform_provider_version_target_id) WHERE terraform_provider_version_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_variable_target_id ON activity_events(variable_target_id) WHERE variable_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_workspace_target_id ON activity_events(workspace_target_id) WHERE workspace_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_vcs_provider_target_id ON activity_events(vcs_provider_target_id) WHERE vcs_provider_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_terraform_module_target_id ON activity_events(terraform_module_target_id) WHERE terraform_module_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_tf_module_version_target_id ON activity_events(terraform_module_version_target_id) WHERE terraform_module_version_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_runner_target_id ON activity_events(runner_target_id) WHERE runner_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_role_target_id ON activity_events(role_target_id) WHERE role_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_tf_provider_version_mirror_target_id ON activity_events(terraform_provider_version_mirror_target_id) WHERE terraform_provider_version_mirror_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_federated_registry_target_id ON activity_events(federated_registry_target_id) WHERE federated_registry_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_package_target_id ON activity_events(package_target_id) WHERE package_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_package_version_target_id ON activity_events(package_version_target_id) WHERE package_version_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_policy_target_id ON activity_events(policy_target_id) WHERE policy_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_run_gate_target_id ON activity_events(run_gate_target_id) WHERE run_gate_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_cleanup_policy_target_id ON activity_events(cleanup_policy_target_id) WHERE cleanup_policy_target_id IS NOT NULL;
CREATE INDEX index_activity_events_on_workspace_role_binding_target_id ON activity_events(workspace_role_binding_target_id) WHERE workspace_role_binding_target_id IS NOT NULL;

CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $$
DECLARE
    row RECORD;
    notification json;
BEGIN -- Convert the old or new row to JSON, based on the kind of action.
    -- Action = DELETE?             -> OLD row
    -- Action = INSERT or UPDATE?   -> NEW row
    IF (TG_OP = 'DELETE') THEN row = OLD;
    ELSE row = NEW;
    END IF;

    -- Construct the notification as a JSON string.
    notification = json_build_object(
        'table',
        TG_TABLE_NAME,
        'action',
        TG_OP,
        'id',
        row.id,
        'data',
        row_to_json(row)
    );

    -- Execute pg_notify(channel, notification)
    PERFORM pg_notify('events', notification::text);

    -- Result is ignored since this is an AFTER trigger
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Notify-event triggers omit DELETE for tables whose delete events are not consumed by any subscriber.
-- asym_signing_keys is included: its subscriber re-syncs the full key set from the database on each
-- event, and signing keys are only ever deleted once fully decommissioned (already expired) or while
-- still in the pending "creating" state (no public key, so never in the key set). The maintenance_mode
-- and runners triggers keep DELETE because their subscribers react to deletions.
CREATE TRIGGER jobs_notify_event
    AFTER INSERT OR UPDATE ON jobs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runs_notify_event
    AFTER INSERT OR UPDATE ON runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER workspaces_notify_event
    AFTER INSERT OR UPDATE ON workspaces
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runners_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON runners
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER maintenance_mode_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON maintenance_mode
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER log_streams_notify_event
    AFTER INSERT OR UPDATE ON log_streams
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runner_sessions_notify_event
    AFTER INSERT OR UPDATE ON runner_sessions
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER asym_signing_keys_notify_event
    AFTER INSERT OR UPDATE ON asym_signing_keys
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER workspace_assessments_notify_event
    AFTER INSERT OR UPDATE ON workspace_assessments
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER agent_session_runs_notify_event
    AFTER INSERT OR UPDATE ON agent_session_runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER work_items_queue_notify_event
    AFTER INSERT ON work_items_queue
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

-- Wakes the email background workers (materializer/sender/cleaner) promptly on new or updated items,
-- so their fallback poll interval can stay long without delaying delivery.
CREATE TRIGGER email_outbox_items_notify_event
    AFTER INSERT OR UPDATE ON email_outbox_items
    FOR EACH ROW EXECUTE PROCEDURE notify_event();
