DROP TABLE IF EXISTS log_stream_chunks;
ALTER TABLE log_streams DROP COLUMN IF EXISTS compaction_started_at;
ALTER TABLE log_streams DROP COLUMN IF EXISTS compacted;
ALTER TABLE log_streams DROP COLUMN IF EXISTS truncated;
