-- Add annotations to runs. Annotations are immutable key/value pairs (with an optional link) set at
-- run creation, letting a run be traced back to what created it (e.g. commit, repository, triggering
-- job). Stored as a JSONB array, mirroring the existing targets column and Phobos pipeline annotations.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS annotations JSONB NOT NULL DEFAULT '[]';
