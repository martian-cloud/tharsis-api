DELETE FROM resource_limits WHERE id = 'de6fbc64-92b0-4e3c-aadf-a7c64e814bed';

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS federated_registry_target_id;

DROP TABLE federated_registries;
