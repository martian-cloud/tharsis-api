CREATE TABLE federated_registries (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    hostname VARCHAR NOT NULL,
    audience VARCHAR NOT NULL,
    group_id UUID NOT NULL,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_federated_registries_on_hostname_group_id ON federated_registries(hostname, group_id);

INSERT INTO resource_limits (id, version, created_at, updated_at, name, value)
VALUES (
        'de6fbc64-92b0-4e3c-aadf-a7c64e814bed',
        1,
        CURRENT_TIMESTAMP(7),
        CURRENT_TIMESTAMP(7),
        'ResourceLimitFederatedRegistriesPerGroup',
        100
    ) -- number of federated registries per group
    ON CONFLICT DO NOTHING;

ALTER TABLE activity_events
ADD COLUMN IF NOT EXISTS federated_registry_target_id UUID,
    ADD CONSTRAINT fk_activity_events_federated_registry_target_id FOREIGN KEY(federated_registry_target_id) REFERENCES federated_registries(id) ON DELETE CASCADE;
