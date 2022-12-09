CREATE TABLE IF NOT EXISTS vcs_providers (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    type VARCHAR NOT NULL,
    hostname VARCHAR NOT NULL,
    oauth_client_id VARCHAR NOT NULL,
    oauth_client_secret VARCHAR NOT NULL,
    oauth_state UUID,
    oauth_access_token VARCHAR,
    oauth_refresh_token VARCHAR,
    oauth_access_token_expires_at TIMESTAMP,
    auto_create_webhooks BOOLEAN NOT NULL,
    group_id UUID NOT NULL,
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

ALTER TABLE activity_events
    ADD COLUMN IF NOT EXISTS vcs_provider_target_id UUID,
    ADD CONSTRAINT fk_activity_events_vcs_provider_target_id FOREIGN KEY(vcs_provider_target_id) REFERENCES vcs_providers(id) ON DELETE CASCADE;

ALTER TABLE configuration_versions
    ADD COLUMN IF NOT EXISTS vcs_event_id UUID,
    ADD CONSTRAINT fk_vcs_event_id FOREIGN KEY(vcs_event_id) REFERENCES vcs_events(id);
