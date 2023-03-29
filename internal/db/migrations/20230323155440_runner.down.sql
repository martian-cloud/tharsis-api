DROP TRIGGER runners_notify_event ON runners;

ALTER TABLE jobs DROP COLUMN IF EXISTS runner_path;

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS runner_target_id;

DROP TABLE service_account_runner_relation;
DROP TABLE runners;
