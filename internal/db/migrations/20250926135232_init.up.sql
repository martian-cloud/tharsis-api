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
    CONSTRAINT fk_parent_id FOREIGN KEY(parent_id) REFERENCES groups(id) ON DELETE CASCADE
);

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
    current_job_id UUID,
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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE INDEX index_workspaces_created_at_id ON workspaces(created_at DESC, id DESC);

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
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_vcs_event_id FOREIGN KEY(vcs_event_id) REFERENCES vcs_events(id)
);

CREATE TABLE plans (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    status VARCHAR NOT NULL,
    has_changes BOOLEAN NOT NULL,
    resource_additions INTEGER NOT NULL,
    resource_changes INTEGER NOT NULL,
    resource_destructions INTEGER NOT NULL,
    resource_imports INTEGER NOT NULL DEFAULT 0,
    resource_drift INTEGER NOT NULL DEFAULT 0,
    output_additions INTEGER NOT NULL DEFAULT 0,
    output_changes INTEGER NOT NULL DEFAULT 0,
    output_destructions INTEGER NOT NULL DEFAULT 0,
    diff_size INTEGER NOT NULL DEFAULT 0,
    error_message VARCHAR,
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE TABLE applies (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    status VARCHAR NOT NULL,
    triggered_by VARCHAR,
    comment VARCHAR NOT NULL,
    error_message VARCHAR,
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE TABLE runs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    is_destroy BOOLEAN NOT NULL,
    has_changes BOOLEAN NOT NULL,
    workspace_id UUID NOT NULL,
    configuration_version_id UUID,
    plan_id UUID,
    apply_id UUID,
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
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_configuration_version FOREIGN KEY(configuration_version_id) REFERENCES configuration_versions(id),
    CONSTRAINT fk_plan FOREIGN KEY(plan_id) REFERENCES plans(id),
    CONSTRAINT fk_apply FOREIGN KEY(apply_id) REFERENCES applies(id)
);
CREATE INDEX index_runs_on_plan_id ON runs(plan_id);
CREATE INDEX index_runs_on_apply_id ON runs(apply_id);
CREATE INDEX index_runs_on_workspace_id ON runs(workspace_id);
CREATE INDEX index_runs_created_at_id ON runs(created_at DESC, id DESC);

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
    cancel_requested BOOLEAN NOT NULL,
    cancel_requested_at TIMESTAMP,
    queued_at TIMESTAMP,
    pending_at TIMESTAMP,
    running_at TIMESTAMP,
    finished_at TIMESTAMP,
    max_job_duration INTEGER NOT NULL,
    runner_path VARCHAR DEFAULT 'legacy-system-runner',
    tags JSONB NOT NULL DEFAULT '[]',
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id),
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE SET NULL
);
CREATE INDEX index_jobs_on_run_id_and_type ON jobs(run_id, type);
CREATE INDEX index_jobs_on_workspace_id ON jobs(workspace_id);
CREATE INDEX index_jobs_on_runner_id ON jobs(runner_id);
CREATE INDEX index_jobs_on_status ON jobs(status);
CREATE INDEX index_jobs_on_tags ON jobs USING GIN (tags jsonb_path_ops);

CREATE TABLE state_versions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    run_id UUID,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id)
);

ALTER TABLE workspaces ADD CONSTRAINT fk_current_job_id FOREIGN KEY(current_job_id) REFERENCES jobs(id) ON DELETE SET NULL,
ADD CONSTRAINT fk_current_state_version_id FOREIGN KEY(current_state_version_id) REFERENCES state_versions(id) ON DELETE SET NULL;

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
    password_hash BYTEA
);
CREATE UNIQUE INDEX index_users_on_username ON users(username);
CREATE UNIQUE INDEX index_users_on_email ON users(email);
CREATE UNIQUE INDEX index_users_on_scim_external_id ON users(scim_external_id);

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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_service_accounts_on_name ON service_accounts(name, group_id);

CREATE TABLE teams (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    scim_external_id UUID
    -- TODO: Eventually, add an org_id and the trimmings
);
CREATE UNIQUE INDEX index_teams_on_name ON teams(name);
CREATE UNIQUE INDEX index_teams_on_scim_external_id ON teams(scim_external_id);

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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_alias_source_id FOREIGN KEY(alias_source_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    CONSTRAINT if_alias_source_id_is_null_require_not_null CHECK((alias_source_id IS NOT NULL) OR ((type IS NOT NULL) AND (data is NOT NULL)))
);
CREATE UNIQUE INDEX index_managed_identities_on_name ON managed_identities(name, group_id);

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

CREATE TABLE managed_identity_rule_allowed_users (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE managed_identity_rule_allowed_service_accounts (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE
);

CREATE TABLE managed_identity_rule_allowed_teams (
    id UUID PRIMARY KEY,
    rule_id UUID NOT NULL,
    team_id UUID NOT NULL,
    CONSTRAINT fk_rule_id FOREIGN KEY(rule_id) REFERENCES managed_identity_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE
);

CREATE TABLE workspace_managed_identity_relation (
    managed_identity_id UUID,
    workspace_id UUID,
    CONSTRAINT fk_managed_identity_id FOREIGN KEY(managed_identity_id) REFERENCES managed_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    PRIMARY KEY(managed_identity_id, workspace_id)
);

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    permissions JSONB NOT NULL
);
CREATE UNIQUE INDEX index_roles_on_name ON roles(name);

INSERT INTO roles
(id, version, created_at, updated_at, created_by, name, description, permissions)
VALUES
    ('623c83ea-23fe-4de6-874a-a99ccf6a76fc', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'owner', 'Default owner role.', '[]'),
    ('8aa7adba-b769-471f-8ebb-3215f33991cb', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'deployer', 'Default deployer role.', '[]'),
    ('52da70fd-37b0-4349-bb64-fb4659bcf5f5', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'viewer', 'Default viewer role.', '[]')
ON CONFLICT DO NOTHING;

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
    CONSTRAINT fk_job_id FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_log_streams_runner_session_id FOREIGN KEY(runner_session_id) REFERENCES runner_sessions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_log_streams_on_job_id ON log_streams(job_id);
CREATE UNIQUE INDEX index_log_streams_on_runner_session_id ON log_streams(runner_session_id);

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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_providers_on_name ON terraform_providers(root_group_id, name);

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
    CONSTRAINT fk_provider_id FOREIGN KEY(provider_id) REFERENCES terraform_providers(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_versions_on_version ON terraform_provider_versions(provider_id, provider_sem_version);
CREATE UNIQUE INDEX index_terraform_provider_versions_on_latest ON terraform_provider_versions(provider_id, latest) WHERE latest = true;

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
    CONSTRAINT fk_provider_version_id FOREIGN KEY(provider_version_id) REFERENCES terraform_provider_versions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_platforms_on_os_arch ON terraform_provider_platforms(provider_version_id, os, arch);

CREATE TABLE scim_tokens (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    nonce UUID NOT NULL
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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_vcs_providers_on_group_id ON vcs_providers(name, group_id);

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
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_provider_id FOREIGN KEY(provider_id) REFERENCES vcs_providers(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_workspace_vcs_provider_links_on_workspace_id ON workspace_vcs_provider_links(workspace_id);

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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_modules_on_name_system ON terraform_modules(root_group_id, name, system);

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
    CONSTRAINT fk_module_id FOREIGN KEY(module_id) REFERENCES terraform_modules(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_module_versions_on_semantic_version ON terraform_module_versions(module_id, semantic_version);
CREATE UNIQUE INDEX index_terraform_module_versions_on_latest ON terraform_module_versions(module_id, latest) WHERE latest = true;

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
    CONSTRAINT fk_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_version_mirrors_on_type_version_namespace_hostname ON terraform_provider_version_mirrors(type, semantic_version, registry_namespace, registry_hostname, group_id);

CREATE TABLE federated_registries (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    hostname VARCHAR NOT NULL,
    audience VARCHAR NOT NULL,
    group_id UUID NOT NULL,
    created_by VARCHAR NOT NULL DEFAULT '',
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_federated_registries_on_hostname_group_id ON federated_registries(hostname, group_id);

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
    CONSTRAINT fk_activity_events_federated_registry_target_id FOREIGN KEY(federated_registry_target_id) REFERENCES federated_registries(id) ON DELETE CASCADE
);
CREATE INDEX index_activity_events_on_user_id ON activity_events(user_id);
CREATE INDEX index_activity_events_on_service_account_id ON activity_events(service_account_id);
CREATE INDEX index_activity_events_on_namespace_id ON activity_events(namespace_id);
CREATE INDEX index_activity_events_created_at_id ON activity_events(created_at DESC, id DESC);

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
    ('de6fbc64-92b0-4e3c-aadf-a7c64e814bed', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitFederatedRegistriesPerGroup', 100)
ON CONFLICT DO NOTHING;

CREATE TABLE terraform_provider_platform_mirrors (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    os VARCHAR NOT NULL,
    architecture VARCHAR NOT NULL,
    version_mirror_id UUID NOT NULL,
    CONSTRAINT fk_version_mirror_id FOREIGN KEY (version_mirror_id) REFERENCES terraform_provider_version_mirrors(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_platform_mirrors_on_os_arch ON terraform_provider_platform_mirrors(version_mirror_id, os, architecture);

CREATE TABLE maintenance_mode (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL
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

INSERT INTO namespace_variable_versions (id, version, created_at, updated_at, variable_id, key, value, hcl)
SELECT gen_random_uuid(), 1, created_at, updated_at, id, key, value, hcl FROM namespace_variables;

CREATE TABLE latest_namespace_variable_versions (
    variable_id UUID NOT NULL,
    version_id UUID NOT NULL,
    CONSTRAINT fk_variable_id FOREIGN KEY(variable_id) REFERENCES namespace_variables(id) ON DELETE CASCADE,
    CONSTRAINT fk_version_id FOREIGN KEY(version_id) REFERENCES namespace_variable_versions(id) ON DELETE CASCADE,
    PRIMARY KEY(variable_id, version_id)
);

INSERT INTO latest_namespace_variable_versions (variable_id, version_id)
SELECT variable_id, id FROM namespace_variable_versions WHERE version = 1;

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
    dismissible BOOLEAN NOT NULL DEFAULT false
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

CREATE TRIGGER jobs_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON jobs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runs_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER workspaces_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON workspaces
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runners_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON runners FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER maintenance_mode_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON maintenance_mode FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER log_streams_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON log_streams FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runner_sessions_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON runner_sessions FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER asym_signing_keys_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON asym_signing_keys FOR EACH ROW EXECUTE PROCEDURE notify_event();
