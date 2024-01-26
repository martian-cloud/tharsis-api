-- This upward migration renames the job_log_descriptors table to log_streams and adds runner_sessions.

DROP TRIGGER job_log_descriptors_notify_event ON job_log_descriptors;

CREATE TABLE IF NOT EXISTS runner_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    runner_id UUID NOT NULL,
    last_contacted_at TIMESTAMP NOT NULL,
    error_count INTEGER NOT NULL,
    internal BOOLEAN NOT NULL,
    CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE CASCADE
);
CREATE INDEX index_runner_sessions_on_runner_id ON runner_sessions(runner_id);

ALTER TABLE job_log_descriptors RENAME TO log_streams;
ALTER INDEX index_job_log_descriptors_on_job_id RENAME TO index_log_streams_on_job_id;
ALTER TABLE log_streams ALTER COLUMN job_id DROP NOT NULL;
ALTER TABLE log_streams ADD COLUMN runner_session_id UUID;
ALTER TABLE log_streams ADD COLUMN completed BOOLEAN NOT NULL DEFAULT TRUE;
CREATE UNIQUE INDEX index_log_streams_on_runner_session_id ON log_streams(runner_session_id);
ALTER TABLE log_streams ADD CONSTRAINT fk_log_streams_runner_session_id FOREIGN KEY(runner_session_id)
    REFERENCES runner_sessions(id) ON DELETE CASCADE;

ALTER TABLE runners ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
('02baa4c6-5e83-4c18-95c7-9cc70eeb0417', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunnerSessionsPerRunner', 100) -- number of active runner sessions per runner
ON CONFLICT DO NOTHING;

-- Must set runner_id in jobs table to null if the value does not exist in the runners table.
UPDATE jobs SET runner_id = NULL WHERE runner_id NOT IN (SELECT id FROM runners);

ALTER TABLE jobs ADD CONSTRAINT fk_runner_id FOREIGN KEY(runner_id) REFERENCES runners(id) ON DELETE SET NULL;

CREATE TRIGGER log_streams_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON log_streams FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TRIGGER runner_sessions_notify_event
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON runner_sessions FOR EACH ROW EXECUTE PROCEDURE notify_event();
