-- Must remain the only statement in this file; see up migration for why.
DROP INDEX CONCURRENTLY IF EXISTS index_email_outbox_items_on_ephemeral;
