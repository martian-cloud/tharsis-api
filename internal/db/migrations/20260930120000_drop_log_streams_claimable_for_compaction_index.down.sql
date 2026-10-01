-- Restores the original index. Must remain the only statement in this file; see up migration for why.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_log_streams_claimable_for_compaction
    ON log_streams (updated_at)
    WHERE completed = true AND compacted = false;
