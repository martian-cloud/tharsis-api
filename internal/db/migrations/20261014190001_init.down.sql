-- This is the consolidated baseline, so reverting it would drop the entire schema and every row in it.
-- It shares its version with the last migration it replaced, whose down migration was a no-op, so a
-- "down 1" run on an existing deployment (make migrate-down, or the db-migrate:down CI jobs) lands here.
-- Refuse instead of wiping the database. To reset a local database, use make migrate-reset, which runs
-- migrate drop.
--
-- golang-migrate marks the version dirty before running this file, so a refused revert leaves
-- schema_migrations at version -1 and dirty, with the schema untouched. Recover with
-- "migrate force 20261014190001".
DO $$
BEGIN
    RAISE EXCEPTION 'refusing to revert the baseline migration: it would drop the entire schema; use migrate drop to reset a database';
END
$$;
