CREATE TABLE IF NOT EXISTS maintenance_mode (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    message VARCHAR NOT NULL
);

CREATE TRIGGER maintenance_mode_notify_event
AFTER INSERT OR UPDATE OR DELETE ON maintenance_mode
    FOR EACH ROW EXECUTE PROCEDURE notify_event();
