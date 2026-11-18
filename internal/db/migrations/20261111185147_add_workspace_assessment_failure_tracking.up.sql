-- Why the last assessment attempt could not produce a drift verdict: set when the run can't be
-- created (a renamed module, an expired credential), cleared when one is created again.
-- Nullable, so existing rows mean "did not fail".
ALTER TABLE workspace_assessments
    ADD COLUMN IF NOT EXISTS error_message VARCHAR;

-- Overrides the normal assessment interval, suppressing eligibility until this time. Stamped
-- on every completion: the base interval for a clean result, extended per consecutive recorded
-- failure (exponential backoff). Cleared once a run is created again.
ALTER TABLE workspace_assessments
    ADD COLUMN IF NOT EXISTS available_at TIMESTAMP;

-- Counts consecutive assessment-creation failures (see available_at). Resets to 0 once an
-- assessment run is created again.
ALTER TABLE workspace_assessments
    ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER NOT NULL DEFAULT 0;

-- Backfill: every row that completed before this column existed has available_at still NULL.
-- Stamp those as due now rather than carrying a permanent "legacy row" branch in the
-- eligibility query — going forward, every code path that sets completed_at also stamps
-- available_at in the same write, so this guard can never match a row created after this
-- migration runs.
UPDATE workspace_assessments
    SET available_at = now()
    WHERE completed_at IS NOT NULL AND available_at IS NULL;

