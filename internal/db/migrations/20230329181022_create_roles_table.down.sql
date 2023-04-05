-- Recreate role column.
ALTER TABLE namespace_memberships
    ADD COLUMN IF NOT EXISTS role VARCHAR;

-- Delete any custom roles, so namespace memberships are dropped as well.
DELETE FROM roles WHERE name NOT IN ('owner', 'deployer', 'viewer');

-- Backfill role column.
UPDATE namespace_memberships
SET role = CASE
    WHEN role_id = '623c83ea-23fe-4de6-874a-a99ccf6a76fc' THEN 'owner'
    WHEN role_id = '8aa7adba-b769-471f-8ebb-3215f33991cb' THEN 'deployer'
    WHEN role_id = '52da70fd-37b0-4349-bb64-fb4659bcf5f5' THEN 'viewer'
END;

-- DROP original role_id column.
ALTER TABLE namespace_memberships
    DROP COLUMN IF EXISTS role_id,
    ALTER COLUMN role SET NOT NULL;

DELETE FROM activity_events WHERE target_type = 'ROLE';

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS role_target_id;

DROP TABLE IF EXISTS roles;
