-- Split job/runner-session logs across multiple object-store files ("chunks").
-- Each chunk references one object-store file and records its absolute offset within the stream.

-- Flag set when a stream is truncated at the server-authoritative maximum log size.
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS truncated BOOLEAN NOT NULL DEFAULT FALSE;
-- Flag set once a stream's logs live in the single static-key consolidated object
-- (logstreams/{id}.txt). A completed pre-chunking stream is already final and single-file there, so
-- mark it compacted rather than retrofitting chunk rows for it; reads then go straight to the
-- consolidated object. In-flight pre-chunking streams are left uncompacted: with no chunk rows yet,
-- reads fall back to the consolidated object, and the next WriteLogs adopts that object as chunk 0
-- before appending new chunks.
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS compacted BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE log_streams SET compacted = TRUE WHERE completed = TRUE;
-- Set when an instance begins compacting a stream. Gates compaction so multiple horizontally-scaled
-- instances don't compact the same stream concurrently (a stale value is reclaimable after a TTL).
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS compaction_started_at TIMESTAMP;

CREATE TABLE IF NOT EXISTS log_stream_chunks (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    log_stream_id UUID NOT NULL,
    chunk_index INTEGER NOT NULL,
    start_offset INTEGER NOT NULL,
    size INTEGER NOT NULL,
    object_key VARCHAR NOT NULL,
    sealed BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_log_stream_chunks_log_stream_id FOREIGN KEY(log_stream_id) REFERENCES log_streams(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_log_stream_chunks_on_stream_id_and_index ON log_stream_chunks(log_stream_id, chunk_index);
CREATE INDEX IF NOT EXISTS index_log_stream_chunks_on_stream_id_and_offset ON log_stream_chunks(log_stream_id, start_offset);
