CREATE TABLE IF NOT EXISTS workspace_role_bindings (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_by VARCHAR NOT NULL,
    CONSTRAINT fk_workspace_role_bindings_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    -- RESTRICT rather than CASCADE: silently dropping a binding when its role is deleted would
    -- quietly de-privilege a workspace and break its deployments. Failing the role deletion forces
    -- an admin to unbind first, which surfaces the dependency. Role deletion is admin-only and rare.
    CONSTRAINT fk_workspace_role_bindings_role_id FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE RESTRICT
);

-- One binding per workspace. A binding always applies at the workspace's direct parent namespace, so
-- a second row for the same workspace would have no distinct meaning.
CREATE UNIQUE INDEX IF NOT EXISTS index_workspace_role_bindings_on_workspace_id ON workspace_role_bindings(workspace_id);

-- FK index so the ON DELETE RESTRICT check on roles does not table scan.
CREATE INDEX IF NOT EXISTS index_workspace_role_bindings_on_role_id ON workspace_role_bindings(role_id);

ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS workspace_role_binding_target_id UUID REFERENCES workspace_role_bindings(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS index_activity_events_on_workspace_role_binding_target_id ON activity_events(workspace_role_binding_target_id) WHERE workspace_role_binding_target_id IS NOT NULL;

-- Fixed display position for the default roles when no explicit sort is requested (custom roles
-- default to 100, sorting after all default roles, ties broken by the existing id tiebreaker).
ALTER TABLE roles ADD COLUMN IF NOT EXISTS sort_order SMALLINT NOT NULL DEFAULT 100;
CREATE INDEX IF NOT EXISTS index_roles_on_sort_order ON roles(sort_order);

-- Maintainer default role: everything Owner has except the ability to add, change, or remove
-- namespace memberships — the permission that governs conferring authority on other principals.
-- Same escalation boundary Deployer already respects, extended to a role that otherwise carries
-- Owner's full set. Permissions column is empty because Role.GetPermissions() bypasses the stored
-- column for any DefaultRoleID and resolves from the Go-side map instead (see models/role.go).
INSERT INTO roles (
    id,
    version,
    created_at,
    updated_at,
    created_by,
    name,
    description,
    permissions,
    sort_order
)
VALUES (
    'd1d61904-1255-4e4b-a5ee-4b9543e970b4',
    1,
    CURRENT_TIMESTAMP(7),
    CURRENT_TIMESTAMP(7),
    'system',
    'maintainer',
    'Allows managing all resources, but cannot add, change, or remove namespace memberships.',
    '[]',
    3
) ON CONFLICT DO NOTHING;

-- Refresh the original default roles' descriptions to describe what they actually grant, matching
-- the style used for publisher and maintainer, instead of the original generic placeholders. Also
-- assign their fixed sort positions: viewer, publisher, deployer, maintainer, owner.
UPDATE roles SET description = 'Allows managing all resources, including namespace memberships.', sort_order = 4
    WHERE id = '623c83ea-23fe-4de6-874a-a99ccf6a76fc';
UPDATE roles SET description = 'Allows managing most resources and deploying runs, but only has view access to namespace memberships, managed identities, runners, and policies.', sort_order = 2
    WHERE id = '8aa7adba-b769-471f-8ebb-3215f33991cb';
UPDATE roles SET description = 'Allows read-only access to all resources.', sort_order = 0
    WHERE id = '52da70fd-37b0-4349-bb64-fb4659bcf5f5';
UPDATE roles SET sort_order = 1
    WHERE id = '028fa46b-23ba-443f-a24f-61edcde148ff';
