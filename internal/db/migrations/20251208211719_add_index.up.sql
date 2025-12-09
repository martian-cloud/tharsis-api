CREATE INDEX IF NOT EXISTS index_state_versions_on_run_id ON state_versions(run_id);
CREATE INDEX IF NOT EXISTS index_state_versions_on_workspace_id ON state_versions(workspace_id);
CREATE INDEX IF NOT EXISTS index_state_version_outputs_on_state_version_id ON state_version_outputs(state_version_id);
