-- Recreates index_email_outbox_items_on_ephemeral (dropped in the previous migration) with a
-- predicate (ephemeral IS TRUE AND status = 'completed') that matches the IS TRUE condition goqu
-- generates for the email cleaner's ClaimOutboxItems query; the old predicate (ephemeral = true
-- AND status = 'completed') could not be matched by the planner, so the index was never used.
-- Built CONCURRENTLY to avoid locking email_outbox_items against writes; this statement must
-- remain the only statement in this migration file since pgx's multi-statement mode wraps
-- multi-statement Exec calls in a transaction, which is incompatible with CREATE INDEX
-- CONCURRENTLY.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_email_outbox_items_on_ephemeral
    ON email_outbox_items (created_at)
    WHERE ephemeral IS TRUE AND status = 'completed';
