-- Restores the original index. Must remain the only statement in this file; see up migration for why.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_email_outbox_items_on_ephemeral
    ON email_outbox_items (created_at)
    WHERE ephemeral = true AND status = 'completed';
