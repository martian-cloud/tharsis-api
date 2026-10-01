-- Drops the original ephemeral cleanup index so it can be recreated under the same name with a
-- predicate the planner can match (see following migration). Must remain the only statement in
-- this file since DROP INDEX CONCURRENTLY cannot run inside a transaction.
DROP INDEX CONCURRENTLY IF EXISTS index_email_outbox_items_on_ephemeral;
