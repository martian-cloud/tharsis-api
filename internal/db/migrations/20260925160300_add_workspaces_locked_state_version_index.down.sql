-- Must remain the only statement in this file; see up migration for why.
DROP INDEX CONCURRENTLY IF EXISTS index_workspaces_on_locked_current_state_version_id;
