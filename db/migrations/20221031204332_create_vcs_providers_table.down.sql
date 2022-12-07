DELETE FROM activity_events WHERE target_type = 'VCS_PROVIDER';

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS vcs_provider_target_id;

ALTER TABLE configuration_versions
    DROP CONSTRAINT fk_vcs_event_id,
    DROP COLUMN IF EXISTS vcs_event_id;

DROP TABLE IF EXISTS workspace_vcs_provider_links;
DROP TABLE IF EXISTS vcs_providers;
DROP TABLE IF EXISTS vcs_events;
