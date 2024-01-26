-- This downward migration removes the log_streams table, etc. and puts back the old job_log_descriptors table, etc.

DROP TRIGGER runner_sessions_notify_event ON runner_sessions;
DROP TRIGGER log_streams_notify_event ON log_streams;

ALTER TABLE jobs DROP CONSTRAINT fk_runner_id;

DELETE FROM resource_limits WHERE id = '02baa4c6-5e83-4c18-95c7-9cc70eeb0417';

ALTER TABLE runners DROP COLUMN disabled;

-- Must delete all rows from the log_streams table if the job_id field is null.
DELETE FROM log_streams WHERE job_id IS NULL;

ALTER TABLE log_streams DROP CONSTRAINT fk_log_streams_runner_session_id;
DROP INDEX index_log_streams_on_runner_session_id;
ALTER TABLE log_streams DROP COLUMN completed;
ALTER TABLE log_streams DROP COLUMN runner_session_id;
ALTER TABLE log_streams ALTER COLUMN job_id SET NOT NULL;
ALTER INDEX index_log_streams_on_job_id RENAME TO index_job_log_descriptors_on_job_id;
ALTER TABLE log_streams RENAME TO job_log_descriptors;

DROP TABLE runner_sessions;

CREATE TRIGGER job_log_descriptors_notify_event
AFTER INSERT OR UPDATE OR DELETE ON job_log_descriptors
    FOR EACH ROW EXECUTE PROCEDURE notify_event();
