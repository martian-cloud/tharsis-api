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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    username VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    admin BOOLEAN NOT NULL
);
CREATE UNIQUE INDEX index_users_on_username ON users(username);
CREATE UNIQUE INDEX index_users_on_email ON users(email);

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
    description VARCHAR
    -- TODO: Eventually, add an org_id and the trimmings
);
CREATE UNIQUE INDEX index_teams_on_name ON teams(name);

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
    type VARCHAR NOT NULL,
    group_id UUID NOT NULL,
    data VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_managed_identities_on_name ON managed_identities(name, group_id);

CREATE TABLE managed_identity_rules (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    managed_identity_id UUID NOT NULL,
    run_stage VARCHAR NOT NULL,
    CONSTRAINT fk_managed_identity_id FOREIGN KEY(managed_identity_id) REFERENCES managed_identities(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX index_managed_identity_rules_on_run_stage ON managed_identity_rules(managed_identity_id, run_stage);

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
    run_id UUID NOT NULL,
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
