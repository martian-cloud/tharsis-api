-- Partial index to quickly find completed, uncompacted log streams that are eligible for
-- compaction, ordered by updated_at. Built CONCURRENTLY to avoid locking log_streams against
-- writes; this statement must remain the only statement in this migration file since pgx's
-- multi-statement mode wraps multi-statement Exec calls in a transaction, which is incompatible
-- with CREATE INDEX CONCURRENTLY.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_log_streams_claimable_for_compaction
    ON log_streams (updated_at)
    WHERE completed = true AND compacted = false;
