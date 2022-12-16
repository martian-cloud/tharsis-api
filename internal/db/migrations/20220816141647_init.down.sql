DROP TRIGGER jobs_notify_event ON jobs;
DROP TRIGGER runs_notify_event ON runs;
DROP TRIGGER workspaces_notify_event ON workspaces;
DROP TRIGGER job_log_descriptors_notify_event ON job_log_descriptors;

DROP FUNCTION notify_event;


ALTER TABLE workspaces DROP CONSTRAINT fk_current_job_id;
ALTER TABLE workspaces DROP CONSTRAINT fk_current_state_version_id;

DROP TABLE namespace_variables;
DROP TABLE job_log_descriptors;
DROP TABLE jobs;
DROP TABLE workspace_managed_identity_relation;
DROP TABLE managed_identity_rule_allowed_users;
DROP TABLE managed_identity_rule_allowed_service_accounts;
DROP TABLE managed_identity_rule_allowed_teams;
DROP TABLE managed_identity_rules;
DROP TABLE managed_identities;
DROP TABLE team_members;
DROP TABLE state_version_outputs;
DROP TABLE state_versions;
DROP TABLE runs;
DROP TABLE plans;
DROP TABLE applies;
DROP TABLE configuration_versions;
DROP TABLE namespace_memberships;
DROP TABLE teams;
DROP TABLE service_accounts;
DROP TABLE namespaces;
DROP TABlE user_external_identities;
DROP TABLE users;
DROP TABLE workspaces;
DROP TABLE groups;
