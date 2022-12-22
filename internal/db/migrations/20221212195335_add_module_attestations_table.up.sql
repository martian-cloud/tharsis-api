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
