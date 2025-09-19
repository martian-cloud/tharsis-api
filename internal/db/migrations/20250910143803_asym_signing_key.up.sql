CREATE TABLE IF NOT EXISTS asym_signing_keys (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    public_key BYTEA,
    plugin_data BYTEA,
    plugin_type VARCHAR NOT NULL,
    pub_key_id VARCHAR NOT NULL,
    status VARCHAR NOT NULL
);

CREATE UNIQUE INDEX index_asym_signing_keys_on_status ON asym_signing_keys (status) WHERE (status = 'active' OR status = 'creating');

CREATE TRIGGER asym_signing_keys_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON asym_signing_keys FOR EACH ROW EXECUTE PROCEDURE notify_event();
