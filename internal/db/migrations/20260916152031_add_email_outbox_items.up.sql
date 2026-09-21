CREATE TABLE IF NOT EXISTS email_outbox_items (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    ephemeral BOOLEAN NOT NULL,
    email_type VARCHAR NOT NULL,
    subject VARCHAR NOT NULL,
    payload BYTEA,
    payload_object_store_key VARCHAR,
    status VARCHAR NOT NULL,
    send_to_all_users BOOLEAN NOT NULL,
    recipient_user_ids JSONB NOT NULL DEFAULT '[]',
    recipient_team_ids JSONB NOT NULL DEFAULT '[]',
    send_at TIMESTAMP,
    claimed_at TIMESTAMP
);

-- Partial index for the recipient-materialization poller, which only scans items awaiting materialization.
CREATE INDEX IF NOT EXISTS index_email_outbox_items_on_status ON email_outbox_items(created_at)
    WHERE status = 'preparing';

-- Partial index for the cleanup poller, which only scans completed ephemeral rows.
CREATE INDEX IF NOT EXISTS index_email_outbox_items_on_ephemeral ON email_outbox_items(created_at)
    WHERE ephemeral = true AND status = 'completed';

CREATE TABLE IF NOT EXISTS email_recipients (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email_outbox_item_id UUID NOT NULL,
    address VARCHAR NOT NULL,
    delivery_status VARCHAR NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMP NOT NULL,
    last_attempt_at TIMESTAMP,
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    complained_at TIMESTAMP,
    failure_reason VARCHAR,
    CONSTRAINT fk_email_recipients_email_outbox_item_id FOREIGN KEY(email_outbox_item_id) REFERENCES email_outbox_items(id) ON DELETE CASCADE
);

-- Claim query: retryable rows whose available_at is due, ordered FIFO. available_at drives both
-- scheduling (not claimable until then) and the retry lease (a claim pushes it forward). The predicate
-- must list exactly the claimable statuses so the planner can prove the claim's IN (...) uses this index.
CREATE INDEX IF NOT EXISTS index_email_recipients_on_delivery_status_available_at
    ON email_recipients(delivery_status, available_at)
    WHERE delivery_status IN ('pending', 'accepted', 'soft_bounced', 'delayed');

-- FK index: "are all recipients final?" check + admin per-recipient view + ON DELETE CASCADE efficiency.
CREATE INDEX IF NOT EXISTS index_email_recipients_on_email_outbox_item_id ON email_recipients(email_outbox_item_id);

CREATE TABLE IF NOT EXISTS email_suppressions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    address VARCHAR NOT NULL,
    cause VARCHAR NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS index_email_suppressions_on_address ON email_suppressions(lower(address));

-- Track the email payload blob in object_store_refs (added in the object_store_refs migration) so the
-- janitor reclaims it when the owning email_outbox_items row is deleted. ON DELETE SET NULL nullifies the ref
-- on cascade delete, matching every other tracked resource -- no separate trigger needed.
ALTER TABLE object_store_refs
    ADD COLUMN IF NOT EXISTS email_outbox_item_id UUID REFERENCES email_outbox_items(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_email_outbox_item_id ON object_store_refs(email_outbox_item_id);

-- Recreate the janitor's orphan partial index to include the new FK column. The WHERE clause must list
-- every FK column or the janitor would treat an email-outbox-owned ref as orphaned.
DROP INDEX IF EXISTS index_object_store_refs_orphan;
CREATE INDEX IF NOT EXISTS index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL
  AND package_version_id IS NULL AND email_outbox_item_id IS NULL;

-- Wake the email background workers (materializer/sender/cleaner) promptly on new or updated items,
-- so their fallback poll interval can stay long without delaying delivery.
CREATE TRIGGER email_outbox_items_notify_event
    AFTER INSERT OR UPDATE ON email_outbox_items
    FOR EACH ROW EXECUTE PROCEDURE notify_event();
