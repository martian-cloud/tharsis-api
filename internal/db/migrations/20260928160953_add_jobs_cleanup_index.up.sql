-- Partial index for the cleanup poller, which only scans dispatched, final, not-yet-cleaned jobs.
-- Built CONCURRENTLY in its own migration so it doesn't lock the jobs table on large deployments.
CREATE INDEX CONCURRENTLY IF NOT EXISTS index_jobs_on_cleanup ON jobs(runner_id, created_at)
    WHERE dispatcher_data IS NOT NULL AND cleanup_completed_at IS NULL
      AND status IN ('finished', 'failed', 'canceled');
