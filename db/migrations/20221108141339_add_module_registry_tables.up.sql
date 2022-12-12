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
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id)
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

ALTER TABLE activity_events
    ADD COLUMN IF NOT EXISTS terraform_module_target_id UUID,
    ADD CONSTRAINT fk_activity_events_terraform_module_target_id FOREIGN KEY(terraform_module_target_id) REFERENCES terraform_modules(id) ON DELETE CASCADE;

ALTER TABLE activity_events
    ADD COLUMN IF NOT EXISTS terraform_module_version_target_id UUID,
    ADD CONSTRAINT fk_activity_events_terraform_module_version_target_id FOREIGN KEY(terraform_module_version_target_id) REFERENCES terraform_module_versions(id) ON DELETE CASCADE;
