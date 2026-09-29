-- Must remain the only statement in this file; see up migration for why.
DROP INDEX CONCURRENTLY IF EXISTS index_workspace_assessments_on_updated_at;
