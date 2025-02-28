CREATE TABLE IF NOT EXISTS namespace_variable_versions (
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

CREATE TABLE latest_namespace_variable_versions (
    variable_id UUID NOT NULL,
    version_id UUID NOT NULL,
    CONSTRAINT fk_variable_id FOREIGN KEY(variable_id) REFERENCES namespace_variables(id) ON DELETE CASCADE,
    CONSTRAINT fk_version_id FOREIGN KEY(version_id) REFERENCES namespace_variable_versions(id) ON DELETE CASCADE,
    PRIMARY KEY(variable_id, version_id)
);

INSERT INTO namespace_variable_versions (id, version, created_at, updated_at, variable_id, key, value, hcl)
    SELECT gen_random_uuid(), 1, created_at, updated_at, id, key, value, hcl FROM namespace_variables;


INSERT INTO latest_namespace_variable_versions (variable_id, version_id)
    SELECT variable_id, id FROM namespace_variable_versions WHERE version = 1;

ALTER TABLE namespace_variables ADD COLUMN sensitive BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE namespace_variables ALTER COLUMN value DROP NOT NULL;
ALTER TABLE namespace_variables ALTER COLUMN hcl DROP NOT NULL;

-- TODO: Drop the hcl and value column from namespace_variables
