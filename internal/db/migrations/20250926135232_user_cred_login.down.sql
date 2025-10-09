DROP INDEX IF EXISTS index_activity_events_created_at_id;
DROP INDEX IF EXISTS index_runs_created_at_id;
DROP INDEX IF EXISTS index_workspaces_created_at_id;
DROP INDEX IF EXISTS index_user_sessions_on_oauth_code;

ALTER TABLE user_sessions
    DROP COLUMN IF EXISTS oauth_code,
    DROP COLUMN IF EXISTS oauth_code_challenge,
    DROP COLUMN IF EXISTS oauth_code_challenge_method,
    DROP COLUMN IF EXISTS oauth_code_expiration,
    DROP COLUMN IF EXISTS oauth_redirect_uri;

ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
