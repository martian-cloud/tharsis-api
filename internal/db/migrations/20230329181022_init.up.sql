CREATE TABLE groups (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    parent_id UUID,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_parent_id FOREIGN KEY(parent_id) REFERENCES groups(id) ON DELETE CASCADE
);

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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    username VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    admin BOOLEAN NOT NULL,
    scim_external_id UUID,
    active BOOLEAN NOT NULL DEFAULT true
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

CREATE TABLE namespace_memberships (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    namespace_id UUID NOT NULL,
    user_id UUID,
    service_account_id UUID,
    team_id UUID,
    role VARCHAR NOT NULL,
    CONSTRAINT fk_namespace_memberships_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_team_id FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_memberships_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_memberships_on_user_id_namespace_id ON namespace_memberships(user_id, namespace_id);
CREATE UNIQUE INDEX index_namespace_memberships_on_service_account_id_namespace_id ON namespace_memberships(service_account_id, namespace_id);
CREATE UNIQUE INDEX index_namespace_memberships_on_team_id_namespace_id ON namespace_memberships(team_id, namespace_id);

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
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
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
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
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
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_configuration_version FOREIGN KEY(configuration_version_id) REFERENCES configuration_versions(id),
    CONSTRAINT fk_plan FOREIGN KEY(plan_id) REFERENCES plans(id),
    CONSTRAINT fk_apply FOREIGN KEY(apply_id) REFERENCES applies(id)
);
CREATE INDEX index_runs_on_plan_id ON runs(plan_id);
CREATE INDEX index_runs_on_apply_id ON runs(apply_id);
CREATE INDEX index_runs_on_workspace_id ON runs(workspace_id);

CREATE TABLE namespace_variables (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    namespace_id UUID NOT NULL,
    key VARCHAR NOT NULL,
    value VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    hcl BOOLEAN NOT NULL,
    CONSTRAINT fk_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_variables_on_namespace_id_category_key ON namespace_variables(namespace_id, category, key);

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
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id)
);
CREATE INDEX index_jobs_on_run_id_and_type ON jobs(run_id, type);
CREATE INDEX index_jobs_on_workspace_id ON jobs(workspace_id);
CREATE INDEX index_jobs_on_runner_id ON jobs(runner_id);

CREATE TABLE job_log_descriptors (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    job_id UUID NOT NULL,
    size INTEGER NOT NULL,
    CONSTRAINT fk_job_id FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_job_log_descriptors_on_job_id ON job_log_descriptors(job_id);

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

ALTER TABLE workspaces ADD CONSTRAINT fk_current_job_id FOREIGN KEY(current_job_id) REFERENCES jobs(id) ON DELETE SET NULL;
ALTER TABLE workspaces ADD CONSTRAINT fk_current_state_version_id FOREIGN KEY(current_state_version_id) REFERENCES state_versions(id) ON DELETE SET NULL;

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

CREATE TABLE IF NOT EXISTS gpg_keys (
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

CREATE TABLE IF NOT EXISTS terraform_providers (
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

CREATE TABLE IF NOT EXISTS terraform_provider_versions (
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
CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_provider_versions_on_latest ON terraform_provider_versions(provider_id, latest) WHERE latest = true;

CREATE TABLE IF NOT EXISTS terraform_provider_platforms (
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

CREATE TABLE IF NOT EXISTS scim_tokens (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    nonce UUID NOT NULL
);
CREATE UNIQUE INDEX index_scim_tokens_on_nonce ON scim_tokens(nonce);

CREATE TABLE IF NOT EXISTS vcs_providers (
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

CREATE TABLE IF NOT EXISTS workspace_vcs_provider_links (
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

CREATE TABLE IF NOT EXISTS vcs_events(
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

ALTER TABLE configuration_versions
    ADD COLUMN IF NOT EXISTS vcs_event_id UUID,
    ADD CONSTRAINT fk_vcs_event_id FOREIGN KEY(vcs_event_id) REFERENCES vcs_events(id);

CREATE TABLE IF NOT EXISTS terraform_modules (
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
CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_modules_on_name_system ON terraform_modules(root_group_id, name, system);

CREATE TABLE IF NOT EXISTS terraform_module_versions (
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
CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_module_versions_on_semantic_version ON terraform_module_versions(module_id, semantic_version);
CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_module_versions_on_latest ON terraform_module_versions(module_id, latest) WHERE latest = true;

CREATE TABLE IF NOT EXISTS terraform_module_attestations (
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

CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_module_attestations_on_module_and_data_sha_sum ON terraform_module_attestations(module_id, data_sha_sum);
CREATE INDEX IF NOT EXISTS index_terraform_module_attestations_on_digests ON terraform_module_attestations USING GIN (digests jsonb_path_ops);

CREATE TABLE IF NOT EXISTS runners (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    type VARCHAR NOT NULL,
    group_id UUID,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_runners_on_name ON runners(name)
WHERE (group_id IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS index_runners_on_group_id_name ON runners(group_id, name);

CREATE TABLE IF NOT EXISTS service_account_runner_relation (
    runner_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    PRIMARY KEY(runner_id, service_account_id)
);

CREATE TABLE IF NOT EXISTS roles (
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

ALTER TABLE namespace_memberships
    DROP COLUMN role,
    ADD COLUMN role_id UUID NOT NULL,
    ADD CONSTRAINT fk_namespace_memberships_role_id FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS activity_events (
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
    CONSTRAINT fk_activity_events_role_target_id FOREIGN KEY(role_target_id) REFERENCES roles(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_activity_events_on_user_id ON activity_events(user_id);
CREATE INDEX IF NOT EXISTS index_activity_events_on_service_account_id ON activity_events(service_account_id);
CREATE INDEX IF NOT EXISTS index_activity_events_on_namespace_id ON activity_events(namespace_id);

CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $$

    DECLARE
        row RECORD;
        notification json;

    BEGIN

        -- Convert the old or new row to JSON, based on the kind of action.
        -- Action = DELETE?             -> OLD row
        -- Action = INSERT or UPDATE?   -> NEW row
        IF (TG_OP = 'DELETE') THEN
            row = OLD;
        ELSE
            row = NEW;
        END IF;

        -- Construct the notification as a JSON string.
        notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', row.id);


        -- Execute pg_notify(channel, notification)
        PERFORM pg_notify('events',notification::text);

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

CREATE TRIGGER job_log_descriptors_notify_event
AFTER INSERT OR UPDATE OR DELETE ON job_log_descriptors
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runners_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON runners FOR EACH ROW EXECUTE PROCEDURE notify_event();
