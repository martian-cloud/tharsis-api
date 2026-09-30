ALTER TABLE jobs DROP COLUMN IF EXISTS cleanup_completed_at;
ALTER TABLE jobs DROP COLUMN IF EXISTS cleanup_claimed_at;
ALTER TABLE jobs DROP COLUMN IF EXISTS dispatcher_data;
ALTER TABLE jobs DROP COLUMN IF EXISTS resource_usage_limits;
ALTER TABLE jobs DROP COLUMN IF EXISTS resource_usage_metrics;
