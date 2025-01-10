ALTER TABLE runners
    ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS run_untagged_jobs BOOLEAN NOT NULL DEFAULT TRUE;

-- For pre-existing runners, add a tag to each runner based on the group's (internal) ID.
-- That way, group runners will continue to take priority over shared runners.
UPDATE runners
    SET tags = jsonb_build_array(runners.group_id)
    WHERE group_id IS NOT NULL AND NOT disabled;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS runner_tags JSONB;

-- Also, add a corresponding tag to all groups that have at least one runner.
UPDATE groups
    SET runner_tags = jsonb_build_array(groups.id)
    WHERE EXISTS (SELECT 1 FROM runners WHERE runners.group_id = groups.id AND NOT runners.disabled);

ALTER TABLE workspaces
    ADD COLUMN IF NOT EXISTS runner_tags JSONB;

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]';
CREATE INDEX IF NOT EXISTS index_jobs_on_tags ON jobs USING GIN (tags jsonb_path_ops);
