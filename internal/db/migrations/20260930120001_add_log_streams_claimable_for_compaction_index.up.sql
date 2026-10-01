-- Recreates index_log_streams_claimable_for_compaction (dropped in the previous migration) with a
-- predicate (completed IS TRUE AND compacted IS FALSE) that matches the IS TRUE / IS FALSE
-- conditions goqu generates for ClaimLogStreamsForCompaction; the old predicate (completed = true
-- AND compacted = false) could not be matched by the planner, causing a sequential scan. Built
-- CONCURRENTLY to avoid locking log_streams against writes; this statement must remain the only
-- statement in this migration file since pgx's multi-statement mode wraps multi-statement Exec
-- calls in a transaction, which is incompatible with CREATE INDEX CONCURRENTLY.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_log_streams_claimable_for_compaction
    ON log_streams (updated_at)
    WHERE completed IS TRUE AND compacted IS FALSE;
