-- Restore DELETE on the notify-event triggers the up migration narrowed.
DROP TRIGGER IF EXISTS jobs_notify_event ON jobs;
CREATE TRIGGER jobs_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON jobs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS runs_notify_event ON runs;
CREATE TRIGGER runs_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS workspaces_notify_event ON workspaces;
CREATE TRIGGER workspaces_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON workspaces
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS log_streams_notify_event ON log_streams;
CREATE TRIGGER log_streams_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON log_streams
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS runner_sessions_notify_event ON runner_sessions;
CREATE TRIGGER runner_sessions_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON runner_sessions
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS asym_signing_keys_notify_event ON asym_signing_keys;
CREATE TRIGGER asym_signing_keys_notify_event
    AFTER INSERT OR UPDATE OR DELETE ON asym_signing_keys
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

-- Drop the workspace_assessments notify-event trigger added by the up migration
-- (the table had no trigger before run_refactor).
DROP TRIGGER IF EXISTS workspace_assessments_notify_event ON workspace_assessments;

-- Drop trigger
DROP TRIGGER IF EXISTS work_items_queue_notify_event ON work_items_queue;

-- Drop work_items_queue table
DROP TABLE IF EXISTS work_items_queue;

-- Restore the non-cascading run foreign key on jobs.
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS fk_run_id;
ALTER TABLE jobs ADD CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id);

-- Restore the non-cascading run foreign key on state_versions.
ALTER TABLE state_versions DROP CONSTRAINT IF EXISTS fk_run_id;
ALTER TABLE state_versions ADD CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id);

-- Remove current_apply_run_id from workspaces
ALTER TABLE workspaces DROP COLUMN IF EXISTS current_apply_run_id;

-- Restore current_job_id (dropped by the up migration): column, foreign key, index.
-- The prior value is lossy, so the column is left NULL.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS current_job_id UUID;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_current_job_id' AND conrelid = 'workspaces'::regclass
    ) THEN
        ALTER TABLE workspaces ADD CONSTRAINT fk_current_job_id FOREIGN KEY(current_job_id) REFERENCES jobs(id) ON DELETE SET NULL;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS index_workspaces_on_current_job_id ON workspaces(current_job_id);

-- Recreate the pre-refactor plans/applies tables and the runs.plan_id/apply_id
-- columns (all dropped by the up migration), then restore their data from run_nodes
-- (the source of truth while the new schema was applied) so the prior schema's code
-- can operate after rollback. run_nodes has no workspace_id, so it is taken from the
-- owning run.
CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    status VARCHAR NOT NULL,
    has_changes BOOLEAN NOT NULL,
    resource_additions INTEGER NOT NULL,
    resource_changes INTEGER NOT NULL,
    resource_destructions INTEGER NOT NULL,
    resource_imports INTEGER NOT NULL DEFAULT 0,
    resource_drift INTEGER NOT NULL DEFAULT 0,
    output_additions INTEGER NOT NULL DEFAULT 0,
    output_changes INTEGER NOT NULL DEFAULT 0,
    output_destructions INTEGER NOT NULL DEFAULT 0,
    diff_size INTEGER NOT NULL DEFAULT 0,
    error_message VARCHAR,
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_plans_on_workspace_id ON plans(workspace_id);

CREATE TABLE IF NOT EXISTS applies (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    status VARCHAR NOT NULL,
    triggered_by VARCHAR,
    comment VARCHAR NOT NULL,
    error_message VARCHAR,
    CONSTRAINT fk_workspace_Id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_applies_on_workspace_id ON applies(workspace_id);

ALTER TABLE runs ADD COLUMN IF NOT EXISTS plan_id UUID;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS apply_id UUID;

-- Restore plans from plan nodes.
INSERT INTO plans (
    id, version, created_at, updated_at, workspace_id, status, has_changes,
    resource_additions, resource_changes, resource_destructions, resource_imports,
    resource_drift, output_additions, output_changes, output_destructions, diff_size,
    error_message
)
SELECT
    n.id,
    -- run_nodes no longer stores a version; the legacy plans/applies rows always used 1.
    1,
    -- run_nodes no longer stores its own timestamps; nodes are created with their run, so use
    -- the run's created_at/updated_at to satisfy the legacy NOT NULL columns.
    r.created_at,
    r.updated_at,
    r.workspace_id,
    -- The pre-refactor schema has no created plan status (old plans began at queued); a plan
    -- node still in created (run not yet admitted, no plan job) maps back to pending, the
    -- closest pre-refactor "waiting" state the old code understands.
    CASE WHEN n.status = 'created' THEN 'pending' ELSE n.status END,
    COALESCE(n.plan_has_changes, FALSE),
    n.plan_resource_additions,
    n.plan_resource_changes,
    n.plan_resource_destructions,
    n.plan_resource_imports,
    n.plan_resource_drift,
    n.plan_output_additions,
    n.plan_output_changes,
    n.plan_output_destructions,
    COALESCE(n.plan_diff_size, 0),
    n.error_message
FROM run_nodes n
JOIN runs r ON r.id = n.run_id
WHERE n.type = 'plan'
ON CONFLICT (id) DO UPDATE SET
    version = EXCLUDED.version,
    updated_at = EXCLUDED.updated_at,
    status = EXCLUDED.status,
    has_changes = EXCLUDED.has_changes,
    resource_additions = EXCLUDED.resource_additions,
    resource_changes = EXCLUDED.resource_changes,
    resource_destructions = EXCLUDED.resource_destructions,
    resource_imports = EXCLUDED.resource_imports,
    resource_drift = EXCLUDED.resource_drift,
    output_additions = EXCLUDED.output_additions,
    output_changes = EXCLUDED.output_changes,
    output_destructions = EXCLUDED.output_destructions,
    diff_size = EXCLUDED.diff_size,
    error_message = EXCLUDED.error_message;

-- Restore applies from apply nodes.
INSERT INTO applies (
    id, version, created_at, updated_at, workspace_id, status, triggered_by, comment,
    error_message
)
SELECT
    n.id,
    -- run_nodes no longer stores a version; the legacy plans/applies rows always used 1.
    1,
    -- run_nodes no longer stores its own timestamps; nodes are created with their run, so use
    -- the run's created_at/updated_at to satisfy the legacy NOT NULL columns.
    r.created_at,
    r.updated_at,
    r.workspace_id,
    -- The pre-refactor schema has no skipped status; a skipped apply (never started
    -- before the run ended) maps back to created, the status it held in that schema.
    CASE WHEN n.status = 'skipped' THEN 'created' ELSE n.status END,
    n.apply_triggered_by,
    COALESCE(n.apply_comment, ''),
    n.error_message
FROM run_nodes n
JOIN runs r ON r.id = n.run_id
WHERE n.type = 'apply'
ON CONFLICT (id) DO UPDATE SET
    version = EXCLUDED.version,
    updated_at = EXCLUDED.updated_at,
    status = EXCLUDED.status,
    triggered_by = EXCLUDED.triggered_by,
    comment = EXCLUDED.comment,
    error_message = EXCLUDED.error_message;

-- Point each run at its restored plan/apply rows.
UPDATE runs r SET plan_id = n.id
FROM run_nodes n
WHERE n.run_id = r.id AND n.type = 'plan';

UPDATE runs r SET apply_id = n.id
FROM run_nodes n
WHERE n.run_id = r.id AND n.type = 'apply';

-- Restore the foreign keys and indexes on runs.plan_id/apply_id (added after the
-- columns are populated so the constraints validate).
ALTER TABLE runs DROP CONSTRAINT IF EXISTS fk_plan;
ALTER TABLE runs ADD CONSTRAINT fk_plan FOREIGN KEY(plan_id) REFERENCES plans(id);
ALTER TABLE runs DROP CONSTRAINT IF EXISTS fk_apply;
ALTER TABLE runs ADD CONSTRAINT fk_apply FOREIGN KEY(apply_id) REFERENCES applies(id);
CREATE INDEX IF NOT EXISTS index_runs_on_plan_id ON runs(plan_id);
CREATE INDEX IF NOT EXISTS index_runs_on_apply_id ON runs(apply_id);

-- Restore the single-column workspace_id index that the up migration replaced with the
-- composite (workspace_id, status, updated_at) supervisor index.
DROP INDEX IF EXISTS index_runs_on_workspace_id_status_updated_at;
DROP INDEX IF EXISTS index_runs_on_updated_at_queuing;
CREATE INDEX IF NOT EXISTS index_runs_on_workspace_id ON runs(workspace_id);

-- Restore the single-column jobs indexes that the up migration replaced with the
-- composite (status, created_at, id) and (runner_id, status) dispatch indexes.
DROP INDEX IF EXISTS index_jobs_on_status_created_at;
DROP INDEX IF EXISTS index_jobs_on_runner_id_status;
CREATE INDEX IF NOT EXISTS index_jobs_on_status ON jobs(status);
CREATE INDEX IF NOT EXISTS index_jobs_on_runner_id ON jobs(runner_id);

-- Restore runs.has_changes (dropped by the up migration) from the plan node, since
-- the prior schema's code reads this column directly.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS has_changes BOOLEAN;
UPDATE runs r SET has_changes = COALESCE(
    (SELECT n.plan_has_changes FROM run_nodes n WHERE n.run_id = r.id AND n.type = 'plan'),
    FALSE
);
ALTER TABLE runs ALTER COLUMN has_changes SET NOT NULL;

-- The pre-refactor schema has no queuing/queuing_apply run statuses; map them back to
-- the statuses those runs held before the refactor (pending / planned).
UPDATE runs SET status = 'pending' WHERE status = 'queuing';
UPDATE runs SET status = 'planned' WHERE status = 'queuing_apply';

-- The pre-refactor schema has no discarded run status; map it back to planned, the phase a
-- discarded run had completed before being discarded.
UPDATE runs SET status = 'planned' WHERE status = 'discarded';

-- Drop run_nodes table
DROP TABLE IF EXISTS run_nodes;

-- The refactor recorded run cancel/discard as UPDATE activity events on a RUN target
-- (ActivityEventUpdateRunPayload). The pre-refactor code doesn't produce or understand
-- these, so remove them when reverting. Runs never produced UPDATE activity events before
-- the refactor, so this only removes those rows.
DELETE FROM activity_events WHERE action = 'UPDATE' AND target_type = 'RUN';
