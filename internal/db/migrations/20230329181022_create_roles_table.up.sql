CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    permissions JSONB NOT NULL
);
CREATE UNIQUE INDEX index_roles_on_name ON roles(name);

INSERT INTO roles
    (id, version, created_at, updated_at, created_by, name, description, permissions)
VALUES
    ('623c83ea-23fe-4de6-874a-a99ccf6a76fc', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'owner', 'Default owner role.', '[]'),
    ('8aa7adba-b769-471f-8ebb-3215f33991cb', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'deployer', 'Default deployer role.', '[]'),
    ('52da70fd-37b0-4349-bb64-fb4659bcf5f5', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'system', 'viewer', 'Default viewer role.', '[]')
ON CONFLICT DO NOTHING;

-- Create a new role_id column.
ALTER TABLE namespace_memberships
    ADD COLUMN role_id UUID,
    ADD CONSTRAINT fk_namespace_memberships_role_id FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE;

-- Backfill new role_id column.
WITH defaults AS (SELECT DISTINCT id, name FROM roles)
UPDATE namespace_memberships
SET role_id = CASE
    WHEN role = 'owner' THEN (SELECT id FROM defaults WHERE name = 'owner')
    WHEN role = 'deployer' THEN (SELECT id FROM defaults WHERE name = 'deployer')
    WHEN role = 'viewer' THEN (SELECT id FROM defaults WHERE name = 'viewer')
END;

-- DROP original role column.
ALTER TABLE namespace_memberships
    DROP COLUMN role,
    ALTER COLUMN role_id SET NOT NULL;

ALTER TABLE activity_events
ADD COLUMN IF NOT EXISTS role_target_id UUID,
    ADD CONSTRAINT fk_activity_events_role_target_id FOREIGN KEY(role_target_id) REFERENCES roles(id) ON DELETE CASCADE;
