-- Index on workspace_assessments.updated_at to support the assessment scheduler's
-- GetWorkspaces OR-predicate, which checks completed_at <= ? OR (completed_at IS NULL AND
-- updated_at <= ?). Only completed_at was previously indexed, so the updated_at branch of the
-- OR had no index to use. Built CONCURRENTLY to avoid locking workspace_assessments against
-- writes; this statement must remain the only statement in this migration file since pgx's
-- multi-statement mode wraps multi-statement Exec calls in a transaction, which is incompatible
-- with CREATE INDEX CONCURRENTLY.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_workspace_assessments_on_updated_at
    ON workspace_assessments (updated_at);
