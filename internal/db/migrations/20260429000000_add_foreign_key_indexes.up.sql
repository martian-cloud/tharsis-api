-- Add indexes for foreign key columns that are missing a covering index.
-- This improves performance of CASCADE deletes and JOIN queries on FK columns.

-- groups
CREATE INDEX IF NOT EXISTS index_groups_on_parent_id ON groups(parent_id);

-- workspaces
CREATE INDEX IF NOT EXISTS index_workspaces_on_group_id ON workspaces(group_id);
CREATE INDEX IF NOT EXISTS index_workspaces_on_current_job_id ON workspaces(current_job_id);
CREATE INDEX IF NOT EXISTS index_workspaces_on_current_state_version_id ON workspaces(current_state_version_id);

-- vcs_events
CREATE INDEX IF NOT EXISTS index_vcs_events_on_workspace_id ON vcs_events(workspace_id);

-- configuration_versions
CREATE INDEX IF NOT EXISTS index_configuration_versions_on_workspace_id ON configuration_versions(workspace_id);
CREATE INDEX IF NOT EXISTS index_configuration_versions_on_vcs_event_id ON configuration_versions(vcs_event_id);

-- plans
CREATE INDEX IF NOT EXISTS index_plans_on_workspace_id ON plans(workspace_id);

-- applies
CREATE INDEX IF NOT EXISTS index_applies_on_workspace_id ON applies(workspace_id);

-- runs
CREATE INDEX IF NOT EXISTS index_runs_on_configuration_version_id ON runs(configuration_version_id);

-- user_external_identities
CREATE INDEX IF NOT EXISTS index_user_external_identities_on_user_id ON user_external_identities(user_id);

-- service_accounts
CREATE INDEX IF NOT EXISTS index_service_accounts_on_group_id ON service_accounts(group_id);

-- team_members
CREATE INDEX IF NOT EXISTS index_team_members_on_team_id ON team_members(team_id);

-- managed_identities
CREATE INDEX IF NOT EXISTS index_managed_identities_on_group_id ON managed_identities(group_id);
CREATE INDEX IF NOT EXISTS index_managed_identities_on_alias_source_id ON managed_identities(alias_source_id);

-- managed_identity_rules
CREATE INDEX IF NOT EXISTS index_managed_identity_rules_on_managed_identity_id ON managed_identity_rules(managed_identity_id);

-- managed_identity_rule_allowed_users
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_users_on_rule_id ON managed_identity_rule_allowed_users(rule_id);
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_users_on_user_id ON managed_identity_rule_allowed_users(user_id);

-- managed_identity_rule_allowed_service_accounts
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_service_accounts_on_rule_id ON managed_identity_rule_allowed_service_accounts(rule_id);
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_service_accounts_on_sa_id ON managed_identity_rule_allowed_service_accounts(service_account_id);

-- managed_identity_rule_allowed_teams
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_teams_on_rule_id ON managed_identity_rule_allowed_teams(rule_id);
CREATE INDEX IF NOT EXISTS index_managed_identity_rule_allowed_teams_on_team_id ON managed_identity_rule_allowed_teams(team_id);

-- workspace_managed_identity_relation
CREATE INDEX IF NOT EXISTS index_workspace_managed_identity_relation_on_workspace_id ON workspace_managed_identity_relation(workspace_id);

-- namespace_memberships
CREATE INDEX IF NOT EXISTS index_namespace_memberships_on_namespace_id ON namespace_memberships(namespace_id);
CREATE INDEX IF NOT EXISTS index_namespace_memberships_on_role_id ON namespace_memberships(role_id);

-- namespace_variable_versions
CREATE INDEX IF NOT EXISTS index_namespace_variable_versions_on_variable_id ON namespace_variable_versions(variable_id);

-- latest_namespace_variable_versions
CREATE INDEX IF NOT EXISTS index_latest_namespace_variable_versions_on_version_id ON latest_namespace_variable_versions(version_id);

-- terraform_providers
CREATE INDEX IF NOT EXISTS index_terraform_providers_on_group_id ON terraform_providers(group_id);

-- vcs_providers
CREATE INDEX IF NOT EXISTS index_vcs_providers_on_group_id ON vcs_providers(group_id);

-- workspace_vcs_provider_links
CREATE INDEX IF NOT EXISTS index_workspace_vcs_provider_links_on_provider_id ON workspace_vcs_provider_links(provider_id);

-- terraform_modules
CREATE INDEX IF NOT EXISTS index_terraform_modules_on_group_id ON terraform_modules(group_id);

-- service_account_runner_relation
CREATE INDEX IF NOT EXISTS index_service_account_runner_relation_on_service_account_id ON service_account_runner_relation(service_account_id);

-- terraform_provider_version_mirrors
CREATE INDEX IF NOT EXISTS index_terraform_provider_version_mirrors_on_group_id ON terraform_provider_version_mirrors(group_id);

-- federated_registries
CREATE INDEX IF NOT EXISTS index_federated_registries_on_group_id ON federated_registries(group_id);

-- activity_events (target columns — partial indexes since only one target is set per event)
CREATE INDEX IF NOT EXISTS index_activity_events_on_gpg_key_target_id ON activity_events(gpg_key_target_id) WHERE gpg_key_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_group_target_id ON activity_events(group_target_id) WHERE group_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_managed_identity_target_id ON activity_events(managed_identity_target_id) WHERE managed_identity_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_managed_identity_rule_target_id ON activity_events(managed_identity_rule_target_id) WHERE managed_identity_rule_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_namespace_membership_target_id ON activity_events(namespace_membership_target_id) WHERE namespace_membership_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_run_target_id ON activity_events(run_target_id) WHERE run_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_service_account_target_id ON activity_events(service_account_target_id) WHERE service_account_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_state_version_target_id ON activity_events(state_version_target_id) WHERE state_version_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_team_target_id ON activity_events(team_target_id) WHERE team_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_terraform_provider_target_id ON activity_events(terraform_provider_target_id) WHERE terraform_provider_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_tf_provider_version_target_id ON activity_events(terraform_provider_version_target_id) WHERE terraform_provider_version_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_variable_target_id ON activity_events(variable_target_id) WHERE variable_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_workspace_target_id ON activity_events(workspace_target_id) WHERE workspace_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_vcs_provider_target_id ON activity_events(vcs_provider_target_id) WHERE vcs_provider_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_terraform_module_target_id ON activity_events(terraform_module_target_id) WHERE terraform_module_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_tf_module_version_target_id ON activity_events(terraform_module_version_target_id) WHERE terraform_module_version_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_runner_target_id ON activity_events(runner_target_id) WHERE runner_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_role_target_id ON activity_events(role_target_id) WHERE role_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_tf_provider_version_mirror_target_id ON activity_events(terraform_provider_version_mirror_target_id) WHERE terraform_provider_version_mirror_target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activity_events_on_federated_registry_target_id ON activity_events(federated_registry_target_id) WHERE federated_registry_target_id IS NOT NULL;

-- workspace_assessments
CREATE INDEX IF NOT EXISTS index_workspace_assessments_on_completed_run_id ON workspace_assessments(completed_run_id);

-- notification_preferences
CREATE INDEX IF NOT EXISTS index_notification_preferences_on_namespace_id ON notification_preferences(namespace_id);

-- namespace_favorites
CREATE INDEX IF NOT EXISTS index_namespace_favorites_on_group_id ON namespace_favorites(group_id);
CREATE INDEX IF NOT EXISTS index_namespace_favorites_on_workspace_id ON namespace_favorites(workspace_id);

-- agent_session_messages
CREATE INDEX IF NOT EXISTS index_agent_session_messages_on_parent_id ON agent_session_messages(parent_id);
