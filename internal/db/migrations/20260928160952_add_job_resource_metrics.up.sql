ALTER TABLE jobs ADD COLUMN IF NOT EXISTS resource_usage_metrics JSONB;

-- resource_usage_limits holds the configured resource ceilings the job ran under, as JSON; null when
-- no limits were configured. Reported alongside usage so the UI can show usage against its limit.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS resource_usage_limits JSONB;

-- dispatcher_data holds opaque dispatcher-specific data for the job's runtime (e.g. a Kubernetes pod
-- name or Docker container ID) as a JSON string map. It is null until the runner reports a successful
-- dispatch, including for dispatchers whose CleanupJob is a no-op because the runtime self-cleans.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS dispatcher_data JSONB;

-- cleanup_claimed_at leases a job to a single runner's cleanup poller (FOR UPDATE SKIP LOCKED), so two
-- runners never reap the same job; a stale claim from a crashed runner is reclaimable once the lease lapses.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cleanup_claimed_at TIMESTAMP;

-- cleanup_completed_at is stamped once a runner confirms the job's runtime was torn down, so the row is
-- never returned to the cleanup poller again.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cleanup_completed_at TIMESTAMP;
