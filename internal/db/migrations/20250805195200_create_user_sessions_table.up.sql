CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    user_agent VARCHAR NOT NULL,
    refresh_token_id UUID NOT NULL,
    expiration TIMESTAMP NOT NULL,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_user_sessions_on_refresh_token_id ON user_sessions(refresh_token_id);
CREATE INDEX IF NOT EXISTS index_user_sessions_on_user_id_refresh_token_id ON user_sessions(user_id, refresh_token_id);
CREATE INDEX IF NOT EXISTS index_user_sessions_on_user_id_expiration ON user_sessions(user_id, expiration);
