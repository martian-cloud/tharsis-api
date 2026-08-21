-- Reverse of the policy-enforcement migration, in dependency order.

-- ============================================================================================
-- Run admission: restore run_refactor's queuing / queuing_apply statuses.
--
-- Rolling back lands on the run_refactor schema, which has queuing / queuing_apply — so plan_queuing
-- and apply_queuing simply take their old names back. The policy-stage statuses have no counterpart
-- there and cannot be mapped onto one, so runs holding them are canceled (see below).
-- ============================================================================================

-- Restore run_refactor's two-status predicate and drop the index this migration added.
DROP INDEX IF EXISTS index_runs_on_workspace_id_created_at_queuing;
DROP INDEX IF EXISTS index_runs_on_updated_at_queuing;
CREATE INDEX IF NOT EXISTS index_runs_on_updated_at_queuing
    ON runs(updated_at)
    WHERE status IN ('queuing', 'queuing_apply');

-- The exact inverse of the up migration's rename. plan_queued / apply_queued are untouched: they mean the
-- same thing on both sides.
UPDATE runs SET status = 'queuing' WHERE status = 'plan_queuing';
UPDATE runs SET status = 'queuing_apply' WHERE status = 'apply_queuing';

-- Every policy-stage run status is canceled. None of them exist in the prior schema, and there is no
-- honest mapping onto one that does: queuing / queuing_apply mean specifically "this run's plan (or
-- apply) node is pending, waiting for the workspace slot" -- the invariant the prior admitter relies on
-- to find the node to start (see the run_refactor backfill that introduced them) -- and a run in a
-- policy stage does not satisfy it, because the node its stage gates is deliberately still 'created'.
-- Renaming onto those statuses therefore either strands the run in a queuing status the prior code can
-- never advance, or, since this rollback deletes the task_stage and policy_check nodes further down and
-- the unsatisfied gate with them, leaves the run reading as an approved apply waiting for the slot --
-- so the prior code applies a run whose policy gate never cleared, which is exactly what the gate
-- existed to prevent. Canceling is the one outcome the prior schema can express for an in-flight run
-- that cannot continue.
--
-- The statements below select these runs by the same status list, and the one that rewrites runs.status
-- comes last so the others still see it. Keep the lists identical.

-- Release the workspace slot these runs hold. A run whose stage was running (or which was admitted and
-- then parked) owns current_apply_run_id, and a canceled run never releases it on its own, so without
-- this the workspace admits nothing ever again.
UPDATE workspaces SET current_apply_run_id = NULL
WHERE current_apply_run_id IN (
    SELECT id FROM runs WHERE status IN (
        'pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
        'post_plan_running', 'post_plan_awaiting_decision', 'post_plan_completed',
        'pre_apply_queuing', 'pre_apply_running', 'pre_apply_awaiting_decision', 'pre_apply_completed'));

-- Settle the plan and apply nodes the way the state machine settles them on a cancel -- a node that was
-- active becomes canceled, one that never started becomes skipped -- so the run does not read as
-- canceled with a plan still claiming to be queued or pending.
UPDATE run_nodes SET status = 'canceled'
WHERE type IN ('plan', 'apply')
  AND status IN ('pending', 'queued', 'running')
  AND run_id IN (
    SELECT id FROM runs WHERE status IN (
        'pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
        'post_plan_running', 'post_plan_awaiting_decision', 'post_plan_completed',
        'pre_apply_queuing', 'pre_apply_running', 'pre_apply_awaiting_decision', 'pre_apply_completed'));

UPDATE run_nodes SET status = 'skipped'
WHERE type IN ('plan', 'apply')
  AND status = 'created'
  AND run_id IN (
    SELECT id FROM runs WHERE status IN (
        'pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
        'post_plan_running', 'post_plan_awaiting_decision', 'post_plan_completed',
        'pre_apply_queuing', 'pre_apply_running', 'pre_apply_awaiting_decision', 'pre_apply_completed'));

-- Cancel any policy-eval job still in flight. Its policy_check node is deleted below, so a runner
-- claiming the job afterwards would report against a node that no longer exists.
UPDATE jobs SET status = 'canceled'
WHERE type = 'opa'
  AND status IN ('pending', 'queued', 'running')
  AND run_id IN (
    SELECT id FROM runs WHERE status IN (
        'pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
        'post_plan_running', 'post_plan_awaiting_decision', 'post_plan_completed',
        'pre_apply_queuing', 'pre_apply_running', 'pre_apply_awaiting_decision', 'pre_apply_completed'));

-- Last: rewrites the status the statements above select by.
UPDATE runs SET status = 'canceled'
WHERE status IN ('pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
                 'post_plan_running', 'post_plan_awaiting_decision', 'post_plan_completed',
                 'pre_apply_queuing', 'pre_apply_running', 'pre_apply_awaiting_decision', 'pre_apply_completed');

-- Remove activity events for policy-enforcement target types before dropping their columns.
DELETE FROM activity_events WHERE target_type IN ('PACKAGE', 'PACKAGE_VERSION', 'POLICY', 'RUN_GATE');

-- Drop activity event target columns added by this migration.
ALTER TABLE activity_events
    DROP CONSTRAINT IF EXISTS fk_activity_events_run_gate_target_id,
    DROP CONSTRAINT IF EXISTS fk_activity_events_policy_target_id,
    DROP CONSTRAINT IF EXISTS fk_activity_events_package_version_target_id,
    DROP CONSTRAINT IF EXISTS fk_activity_events_package_target_id;

ALTER TABLE activity_events DROP COLUMN IF EXISTS run_gate_target_id;
ALTER TABLE activity_events DROP COLUMN IF EXISTS policy_target_id;
ALTER TABLE activity_events DROP COLUMN IF EXISTS package_version_target_id;
ALTER TABLE activity_events DROP COLUMN IF EXISTS package_target_id;

-- Run gate approvals.
DROP TABLE IF EXISTS run_gate_approvals;
DROP TABLE IF EXISTS run_gate_allowed_teams;
DROP TABLE IF EXISTS run_gate_allowed_service_accounts;
DROP TABLE IF EXISTS run_gate_allowed_users;
DROP TABLE IF EXISTS run_gates;

-- Resource limits seeded by this migration.
DELETE FROM resource_limits WHERE name IN ('ResourceLimitPackagesPerGroup', 'ResourceLimitPoliciesPerGroup', 'ResourceLimitVersionsPerPackagePerTimePeriod');

-- Job type-specific data column and its policy-check-id index.
DROP INDEX IF EXISTS index_jobs_on_policy_check_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS job_data;

-- Task-stage and policy-check run nodes. The task_stage and policy_check node types are introduced by
-- this migration, so rolling it back has to remove the rows and not just their columns. Dropping the
-- columns alone would leave rows the prior code has no node type for, and on a re-apply they would come
-- back with stage_name / policy_check_type NULL — which hydrates as an empty stage name (db/run.go
-- treats stage_name as nullable), collapsing every check onto one nameless stage, or dereferencing a
-- nil stage when no task_stage row matches "".
--
-- policy_check rows go first: they are children of task_stage rows (matched on stage_name, no FK).
-- Nothing else references these ids — run_gates.policy_check_id and jobs.job_data->>'policyCheckID'
-- both live in objects this migration drops above, and run_nodes.latest_job_id has no FK.
DELETE FROM run_nodes WHERE type = 'policy_check';
DELETE FROM run_nodes WHERE type = 'task_stage';

ALTER TABLE runs DROP COLUMN IF EXISTS has_advisory_failures;

-- sort_order predates this migration (run_refactor), so it is left in place.
ALTER TABLE run_nodes DROP COLUMN IF EXISTS policy_check_messages_summary;
ALTER TABLE run_nodes DROP COLUMN IF EXISTS policy_check_policies;
ALTER TABLE run_nodes DROP COLUMN IF EXISTS stage_name;
ALTER TABLE run_nodes DROP COLUMN IF EXISTS policy_check_type;

-- Policy set policies (+ approver child tables).
DROP TABLE IF EXISTS policy_allowed_teams;
DROP TABLE IF EXISTS policy_allowed_service_accounts;
DROP TABLE IF EXISTS policy_allowed_users;
DROP TABLE IF EXISTS policies;

-- Package version object tracking. Remove the FK column and restore the orphan partial index to its
-- object_store_refs-migration form (without package_version_id) before package_versions is dropped
-- below, since the column references it.
DROP INDEX IF EXISTS index_object_store_refs_orphan;
CREATE INDEX IF NOT EXISTS index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL;
DROP INDEX IF EXISTS index_object_store_refs_on_package_version_id;
ALTER TABLE object_store_refs DROP COLUMN IF EXISTS package_version_id;

-- Policy set registry.
DROP TABLE IF EXISTS package_versions;
DROP TABLE IF EXISTS packages;
