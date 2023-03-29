CREATE TABLE IF NOT EXISTS runners (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    type VARCHAR NOT NULL,
    group_id UUID,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_runners_on_name ON runners(name)
WHERE (group_id IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS index_runners_on_group_id_name ON runners(group_id, name);

CREATE TABLE IF NOT EXISTS service_account_runner_relation (
    runner_id UUID NOT NULL,
    service_account_id UUID NOT NULL,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_account_id FOREIGN KEY(service_account_id) REFERENCES service_accounts(id) ON DELETE CASCADE,
    PRIMARY KEY(runner_id, service_account_id)
);

ALTER TABLE jobs
ADD COLUMN IF NOT EXISTS runner_path VARCHAR DEFAULT 'legacy-system-runner';

ALTER TABLE activity_events
ADD COLUMN IF NOT EXISTS runner_target_id UUID,
    ADD CONSTRAINT fk_activity_events_runner_target_id FOREIGN KEY(runner_target_id) REFERENCES runners(id) ON DELETE CASCADE;

CREATE TRIGGER runners_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON runners FOR EACH ROW EXECUTE PROCEDURE notify_event();
