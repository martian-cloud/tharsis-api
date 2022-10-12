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
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id)
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
    CONSTRAINT fk_provider_id FOREIGN KEY(provider_id) REFERENCES terraform_providers(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_terraform_provider_versions_on_version ON terraform_provider_versions(provider_id, provider_sem_version);

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
