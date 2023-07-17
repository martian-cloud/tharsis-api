CREATE TABLE IF NOT EXISTS terraform_provider_version_mirrors (
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

CREATE TABLE IF NOT EXISTS terraform_provider_platform_mirrors (
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

INSERT INTO resource_limits
    (id, version, created_at, updated_at, name, value)
VALUES
    ('1d26d247-4323-4ed4-adca-94516e5cf4f9', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformProviderVersionMirrorsPerGroup', 1000) -- number of terraform provider version mirrors per group
ON CONFLICT DO NOTHING;

ALTER TABLE activity_events
    ADD COLUMN IF NOT EXISTS terraform_provider_version_mirror_target_id UUID,
    ADD CONSTRAINT fk_activity_events_terraform_provider_version_mirror_target_id FOREIGN KEY(terraform_provider_version_mirror_target_id) REFERENCES terraform_provider_version_mirrors(id) ON DELETE CASCADE;
