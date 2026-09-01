CREATE TABLE IF NOT EXISTS cleanup_policies (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    group_id UUID,
    workspace_id UUID,
    disabled BOOLEAN NOT NULL,
    kind VARCHAR NOT NULL,
    kind_data JSONB NOT NULL,
    sweep_claimed_at TIMESTAMP,
    sweep_cursor VARCHAR,
    last_sweep_completed_at TIMESTAMP,
    CONSTRAINT fk_cleanup_policies_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_cleanup_policies_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS index_cleanup_policies_on_group_id_kind ON cleanup_policies(group_id, kind) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS index_cleanup_policies_on_workspace_id_kind ON cleanup_policies(workspace_id, kind) WHERE workspace_id IS NOT NULL;

-- FK indexes for cleanup_policies so ON DELETE CASCADE is efficient.
CREATE INDEX IF NOT EXISTS index_cleanup_policies_on_group_id ON cleanup_policies(group_id);
CREATE INDEX IF NOT EXISTS index_cleanup_policies_on_workspace_id ON cleanup_policies(workspace_id);

-- The sweeper claims the policies that are due, the ones never claimed first, so it needs this ordering.
CREATE INDEX IF NOT EXISTS index_cleanup_policies_on_sweep_claimed_at ON cleanup_policies(sweep_claimed_at NULLS FIRST);

ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS cleanup_policy_target_id UUID REFERENCES cleanup_policies(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS index_activity_events_on_cleanup_policy_target_id ON activity_events(cleanup_policy_target_id) WHERE cleanup_policy_target_id IS NOT NULL;

-- The pruners page a namespace's resources in fixed order; these cover both the filter and the sort.
CREATE INDEX IF NOT EXISTS index_runs_on_workspace_id_created_at ON runs(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS index_terraform_module_versions_on_module_id_created_at ON terraform_module_versions(module_id, created_at DESC);
CREATE INDEX IF NOT EXISTS index_terraform_provider_versions_on_provider_id_created_at ON terraform_provider_versions(provider_id, created_at DESC);

-- Fix VCS Providers index naming clash
DROP INDEX IF EXISTS index_vcs_providers_on_group_id;
CREATE UNIQUE INDEX index_vcs_providers_on_name_group_id ON vcs_providers(name, group_id);
CREATE INDEX index_vcs_providers_on_group_id ON vcs_providers(group_id);
