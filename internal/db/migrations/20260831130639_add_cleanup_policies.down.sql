DELETE FROM activity_events WHERE target_type = 'CLEANUP_POLICY';
ALTER TABLE activity_events DROP COLUMN IF EXISTS cleanup_policy_target_id;
DROP INDEX IF EXISTS index_activity_events_on_cleanup_policy_target_id;

DROP INDEX IF EXISTS index_runs_on_workspace_id_created_at;
DROP INDEX IF EXISTS index_terraform_module_versions_on_module_id_created_at;
DROP INDEX IF EXISTS index_terraform_provider_versions_on_provider_id_created_at;

DROP TABLE IF EXISTS cleanup_policies;

-- Revert VCS Providers index naming clash fix
DROP INDEX IF EXISTS index_vcs_providers_on_group_id;
DROP INDEX IF EXISTS index_vcs_providers_on_name_group_id;
CREATE UNIQUE INDEX index_vcs_providers_on_group_id ON vcs_providers(name, group_id);
