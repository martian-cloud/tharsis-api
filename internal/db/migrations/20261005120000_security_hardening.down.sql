ALTER TABLE runners DROP COLUMN IF EXISTS job_limit;

ALTER TABLE log_stream_chunks DROP COLUMN IF EXISTS checksum;
ALTER TABLE log_stream_chunks DROP COLUMN IF EXISTS lease_expires_at;

DROP INDEX IF EXISTS index_teams_on_scim_idp_issuer_url_scim_external_id;
CREATE UNIQUE INDEX IF NOT EXISTS index_teams_on_scim_external_id ON teams(scim_external_id);

DROP INDEX IF EXISTS index_users_on_scim_idp_issuer_url_scim_external_id;
CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_scim_external_id ON users(scim_external_id);

ALTER TABLE users DROP COLUMN IF EXISTS scim_idp_issuer_url;
ALTER TABLE teams DROP COLUMN IF EXISTS scim_idp_issuer_url;
ALTER TABLE run_nodes DROP COLUMN IF EXISTS managed_identity_ids;
