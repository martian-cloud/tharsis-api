ALTER TABLE jobs
    DROP COLUMN IF EXISTS tags;

ALTER TABLE workspaces
    DROP COLUMN IF EXISTS runner_tags;

ALTER TABLE groups
    DROP COLUMN IF EXISTS runner_tags;

ALTER TABLE runners
    DROP COLUMN IF EXISTS run_untagged_jobs,
    DROP COLUMN IF EXISTS tags;
