-- Restore the janitor's orphan partial index to its pre-email_outbox_items form (with package_version_id,
-- without email_outbox_item_id) before dropping the FK column that references email_outbox_items.
DROP INDEX IF EXISTS index_object_store_refs_orphan;
CREATE INDEX IF NOT EXISTS index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL
  AND package_version_id IS NULL;
DROP INDEX IF EXISTS index_object_store_refs_on_email_outbox_item_id;
ALTER TABLE object_store_refs DROP COLUMN IF EXISTS email_outbox_item_id;

DROP TABLE IF EXISTS email_suppressions;
DROP TABLE IF EXISTS email_recipients;
DROP TABLE IF EXISTS email_outbox_items;
