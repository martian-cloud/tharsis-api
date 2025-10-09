ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash BYTEA;

ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS oauth_code VARCHAR;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS oauth_code_challenge VARCHAR;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS oauth_code_challenge_method VARCHAR;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS oauth_code_expiration TIMESTAMP;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS oauth_redirect_uri VARCHAR;

CREATE UNIQUE INDEX index_user_sessions_on_oauth_code ON user_sessions(oauth_code);

CREATE INDEX IF NOT EXISTS index_activity_events_created_at_id ON activity_events(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS index_runs_created_at_id ON runs(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS index_workspaces_created_at_id ON workspaces(created_at DESC, id DESC);
