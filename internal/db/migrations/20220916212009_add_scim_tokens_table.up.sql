CREATE TABLE IF NOT EXISTS scim_tokens (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    nonce UUID NOT NULL
);
CREATE UNIQUE INDEX index_scim_tokens_on_nonce ON scim_tokens(nonce);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS scim_external_id UUID,
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE UNIQUE INDEX index_users_on_scim_external_id ON users(scim_external_id);

ALTER TABLE teams
    ADD COLUMN IF NOT EXISTS scim_external_id UUID;

CREATE UNIQUE INDEX index_teams_on_scim_external_id ON teams(scim_external_id);
