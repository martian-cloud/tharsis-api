-- Partial index to speed up the assessment scheduler's GetWorkspaces query, which filters on
-- locked = false AND current_state_version_id IS NOT NULL. Neither column was previously
-- indexed for this predicate shape (current_state_version_id has only a plain single-column
-- index from a prior migration, unfiltered). Built CONCURRENTLY to avoid locking workspaces
-- against writes; this statement must remain the only statement in this migration file since
-- pgx's multi-statement mode wraps multi-statement Exec calls in a transaction, which is
-- incompatible with CREATE INDEX CONCURRENTLY.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_workspaces_on_locked_current_state_version_id
    ON workspaces (locked, current_state_version_id)
    WHERE locked IS FALSE AND current_state_version_id IS NOT NULL;
