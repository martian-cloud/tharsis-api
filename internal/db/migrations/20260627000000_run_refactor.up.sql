-- Create run_nodes table
CREATE TABLE IF NOT EXISTS run_nodes (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL,
    type VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    sort_order INTEGER NOT NULL,
    latest_job_id UUID,
    error_message VARCHAR,
    -- All plan_* columns are nullable: only plan nodes populate them; rows for other node
    -- types (e.g. apply) store NULL ("not applicable") rather than a misleading 0.
    plan_has_changes BOOLEAN,
    plan_diff_size INTEGER,
    plan_resource_additions INTEGER,
    plan_resource_changes INTEGER,
    plan_resource_destructions INTEGER,
    plan_resource_imports INTEGER,
    plan_resource_drift INTEGER,
    plan_output_additions INTEGER,
    plan_output_changes INTEGER,
    plan_output_destructions INTEGER,
    apply_triggered_by VARCHAR,
    apply_comment VARCHAR,
    CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
);

-- (run_id, type) also serves run_id-only lookups (including the FK cascade) via its
-- leading column, so no separate run_id index is needed.
CREATE INDEX IF NOT EXISTS index_run_nodes_on_run_id_type ON run_nodes(run_id, type);

-- Add current_apply_run_id to workspaces. It references the run currently holding the
-- workspace for its apply phase. Mirror the foreign key and index that the replaced
-- current_job_id column carried: null it out if the referenced run is deleted, and index
-- it for lookups. (DROP COLUMN in the down migration drops both automatically.)
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS current_apply_run_id UUID;
ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS fk_current_apply_run_id;
ALTER TABLE workspaces ADD CONSTRAINT fk_current_apply_run_id FOREIGN KEY(current_apply_run_id) REFERENCES runs(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS index_workspaces_on_current_apply_run_id ON workspaces(current_apply_run_id);

-- Cascade-delete a run's jobs when the run is deleted.
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS fk_run_id;
ALTER TABLE jobs ADD CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE;

-- Cascade-delete a run's state versions when the run is deleted.
ALTER TABLE state_versions DROP CONSTRAINT IF EXISTS fk_run_id;
ALTER TABLE state_versions ADD CONSTRAINT fk_run_id FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE;

-- Create work_items_queue table
CREATE TABLE IF NOT EXISTS work_items_queue (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    available_at TIMESTAMP NOT NULL,
    claim_count INTEGER NOT NULL DEFAULT 0,
    type VARCHAR NOT NULL,
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS index_work_items_queue_on_type_available_at_created_at ON work_items_queue(type, available_at, created_at);

DROP TRIGGER IF EXISTS work_items_queue_notify_event ON work_items_queue;
CREATE TRIGGER work_items_queue_notify_event
    AFTER INSERT ON work_items_queue
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

-- Narrow notify-event triggers to drop DELETE for tables whose delete events are
-- not consumed by any subscriber, so the database doesn't emit unused delete
-- notifications. asym_signing_keys is included: its subscriber re-syncs the full key
-- set from the database on each event, and signing keys are only ever deleted
-- internally once fully decommissioned (already expired) or while still in the
-- pending "creating" state (no public key, so never in the key set) — the key set
-- self-heals on the next create/update event. The maintenance_mode and runners
-- triggers keep DELETE because their subscribers react to deletions.
DROP TRIGGER IF EXISTS jobs_notify_event ON jobs;
CREATE TRIGGER jobs_notify_event
    AFTER INSERT OR UPDATE ON jobs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS runs_notify_event ON runs;
CREATE TRIGGER runs_notify_event
    AFTER INSERT OR UPDATE ON runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS workspaces_notify_event ON workspaces;
CREATE TRIGGER workspaces_notify_event
    AFTER INSERT OR UPDATE ON workspaces
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS log_streams_notify_event ON log_streams;
CREATE TRIGGER log_streams_notify_event
    AFTER INSERT OR UPDATE ON log_streams
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS runner_sessions_notify_event ON runner_sessions;
CREATE TRIGGER runner_sessions_notify_event
    AFTER INSERT OR UPDATE ON runner_sessions
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

DROP TRIGGER IF EXISTS asym_signing_keys_notify_event ON asym_signing_keys;
CREATE TRIGGER asym_signing_keys_notify_event
    AFTER INSERT OR UPDATE ON asym_signing_keys
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

-- Add a notify-event trigger to workspace_assessments so subscribers (e.g. workspace
-- event subscriptions) are notified when an assessment is created or updated. DELETE is
-- omitted to match the other narrowed triggers above; assessment deletes aren't consumed.
DROP TRIGGER IF EXISTS workspace_assessments_notify_event ON workspace_assessments;
CREATE TRIGGER workspace_assessments_notify_event
    AFTER INSERT OR UPDATE ON workspace_assessments
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

-- Backfill run_nodes from existing plans/applies

-- Insert plan task nodes (preserve existing plan ID)
INSERT INTO run_nodes (id, run_id, type, status, sort_order, latest_job_id, error_message, plan_has_changes, plan_diff_size, plan_resource_additions, plan_resource_changes, plan_resource_destructions, plan_resource_imports, plan_resource_drift, plan_output_additions, plan_output_changes, plan_output_destructions)
SELECT
    p.id,
    r.id,
    'plan',
    p.status,
    0,
    (SELECT j.id FROM jobs j WHERE j.run_id = r.id AND j.type = 'plan' ORDER BY j.created_at DESC LIMIT 1),
    p.error_message,
    p.has_changes,
    p.diff_size,
    p.resource_additions,
    p.resource_changes,
    p.resource_destructions,
    p.resource_imports,
    p.resource_drift,
    p.output_additions,
    p.output_changes,
    p.output_destructions
FROM runs r
JOIN plans p ON r.plan_id = p.id;

-- Insert apply task nodes (preserve existing apply ID)
INSERT INTO run_nodes (id, run_id, type, status, sort_order, latest_job_id, error_message, apply_triggered_by, apply_comment)
SELECT
    a.id,
    r.id,
    'apply',
    a.status,
    1,
    (SELECT j.id FROM jobs j WHERE j.run_id = r.id AND j.type = 'apply' ORDER BY j.created_at DESC LIMIT 1),
    a.error_message,
    a.triggered_by,
    a.comment
FROM runs r
JOIN applies a ON r.apply_id = a.id
WHERE r.apply_id IS NOT NULL;

-- An apply that never started before its run reached a final state is now explicitly
-- skipped (the pre-refactor schema left it in created forever). Backfill migrated apply
-- nodes so historical runs read consistently with the new state machine.
UPDATE run_nodes n SET status = 'skipped'
FROM runs r
WHERE n.run_id = r.id AND n.type = 'apply' AND n.status = 'created'
  AND r.status IN ('planned_and_finished', 'errored', 'canceled', 'discarded');

-- A run whose plan/apply node is pending (waiting to be queued) is now projected as
-- queuing / queuing_apply instead of lingering in pending / planned. Backfill in-flight
-- runs so the supervisor's status filters still pick them up after the migration.
UPDATE runs r SET status = 'queuing'
WHERE r.status = 'pending'
  AND EXISTS (SELECT 1 FROM run_nodes n WHERE n.run_id = r.id AND n.type = 'plan' AND n.status = 'pending');
UPDATE runs r SET status = 'queuing_apply'
WHERE r.status = 'planned'
  AND EXISTS (SELECT 1 FROM run_nodes n WHERE n.run_id = r.id AND n.type = 'apply' AND n.status = 'pending');

-- Backfill current_apply_run_id from current_job_id, but only when that job is an
-- apply job. The new field specifically means "the run currently holding the workspace
-- for apply", so a workspace that was merely mid-plan (current_job_id pointing at a plan
-- job) must NOT be marked apply-locked — doing so would stall admission for it until the
-- run resolved. A plan-job workspace simply gets a NULL current_apply_run_id.
UPDATE workspaces w
SET current_apply_run_id = (
    SELECT r.id FROM runs r
    JOIN jobs j ON j.run_id = r.id
    WHERE j.id = w.current_job_id AND j.type = 'apply'
    LIMIT 1
)
WHERE w.current_job_id IS NOT NULL;

-- Drop current_job_id now that current_apply_run_id replaces it. Dropping the
-- column also drops fk_current_job_id and index_workspaces_on_current_job_id.
ALTER TABLE workspaces DROP COLUMN IF EXISTS current_job_id;

-- Drop runs.has_changes; it is now derived from the plan node (run_nodes.plan_has_changes).
ALTER TABLE runs DROP COLUMN IF EXISTS has_changes;

-- Drop the legacy plans/applies tables and the runs.plan_id/apply_id columns now that
-- run_nodes is the source of truth. Dropping the columns also drops their fk_plan/
-- fk_apply constraints and the index_runs_on_plan_id/index_runs_on_apply_id indexes,
-- after which the tables have no remaining references and can be dropped.
ALTER TABLE runs DROP COLUMN IF EXISTS plan_id;
ALTER TABLE runs DROP COLUMN IF EXISTS apply_id;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS applies;

-- Composite index for the supervisor's workspace-scoped run queries: queuing pending
-- runs (workspace_id + status IN) and discarding stale planned runs (workspace_id +
-- status + updated_at range). Its leading column also serves workspace_id-only lookups
-- and the FK cascade, making the old single-column index redundant.
CREATE INDEX IF NOT EXISTS index_runs_on_workspace_id_status_updated_at ON runs(workspace_id, status, updated_at);
DROP INDEX IF EXISTS index_runs_on_workspace_id;

-- Partial index for the run reconciler's cross-workspace sweep, which filters by
-- status + updated_at with no workspace_id (so the composite index above can't apply).
-- Scoped to the transient queuing states so it stays tiny and cheap to maintain on a
-- hot, frequently-updated table.
CREATE INDEX IF NOT EXISTS index_runs_on_updated_at_queuing
    ON runs(updated_at)
    WHERE status IN ('queuing', 'queuing_apply');

-- Composite index for the job dispatch query (getNextAvailableJob -> GetJobs), which
-- filters jobs by status (e.g. 'queued') and orders by created_at, id. The old
-- single-column status index can satisfy the filter but not the ordering, forcing a
-- separate sort step; this composite serves the filter, ordering, and LIMIT 1
-- directly, which matters during a burst of pipeline activity that floods the queue.
-- Its leading column also serves status-only lookups, making the old single-column
-- index redundant.
CREATE INDEX IF NOT EXISTS index_jobs_on_status_created_at ON jobs(status, created_at, id);
DROP INDEX IF EXISTS index_jobs_on_status;

-- Composite index for GetJobCountForRunner, which counts jobs by runner_id filtered on
-- status (pending, running). Its leading column also serves runner_id-only lookups and
-- the fk_runner_id ON DELETE SET NULL enforcement scan, making the old single-column
-- index redundant.
CREATE INDEX IF NOT EXISTS index_jobs_on_runner_id_status ON jobs(runner_id, status);
DROP INDEX IF EXISTS index_jobs_on_runner_id;
