-- Add a varchar_pattern_ops index on namespaces.path so prefix LIKE queries
-- (namespaces.path LIKE 'some/path/%') used by the membership filter can use an index range
-- scan. The default UNIQUE index on path uses the database collation and cannot serve prefix
-- LIKE; equality (path = 'x') continues to use that unique index.
CREATE INDEX IF NOT EXISTS index_namespaces_on_path_pattern ON namespaces (path varchar_pattern_ops);
