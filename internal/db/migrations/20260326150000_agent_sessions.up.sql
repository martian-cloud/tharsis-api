CREATE TABLE IF NOT EXISTS agent_sessions (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    total_credits DOUBLE PRECISION NOT NULL DEFAULT 0,
    CONSTRAINT fk_agent_sessions_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS index_agent_sessions_on_user_id ON agent_sessions(user_id);

CREATE TABLE IF NOT EXISTS agent_session_runs (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    session_id UUID NOT NULL,
    previous_run_id UUID,
    last_message_id UUID,
    status TEXT NOT NULL,
    error_message TEXT,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_agent_session_runs_session_id
        FOREIGN KEY(session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_runs_previous_run_id
        FOREIGN KEY(previous_run_id) REFERENCES agent_session_runs(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_agent_session_runs_on_previous_run_id ON agent_session_runs(previous_run_id) WHERE previous_run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_agent_session_runs_on_session_id_created_at ON agent_session_runs(session_id, created_at, id);

CREATE TABLE IF NOT EXISTS agent_session_messages (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    session_id UUID NOT NULL,
    run_id UUID NOT NULL,
    parent_id UUID,
    role TEXT NOT NULL,
    content JSONB,
    CONSTRAINT fk_agent_session_messages_session_id
        FOREIGN KEY(session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_messages_run_id
        FOREIGN KEY(run_id) REFERENCES agent_session_runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_agent_session_messages_parent_id
        FOREIGN KEY(parent_id) REFERENCES agent_session_messages(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS index_agent_session_messages_on_session_id ON agent_session_messages(session_id);
CREATE INDEX IF NOT EXISTS index_agent_session_messages_on_run_id_created_at ON agent_session_messages(run_id, created_at);

CREATE TRIGGER agent_session_runs_notify_event
    AFTER INSERT OR UPDATE ON agent_session_runs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

CREATE TABLE IF NOT EXISTS agent_credit_quotas (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    month_date DATE NOT NULL,
    total_credits DOUBLE PRECISION NOT NULL DEFAULT 0,
    CONSTRAINT fk_agent_credit_quotas_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_agent_credit_quotas_user_month UNIQUE(user_id, month_date)
);
CREATE INDEX IF NOT EXISTS index_agent_credit_quotas_on_user_id ON agent_credit_quotas(user_id);

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
    ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAgentCreditsPerUserPerMonth', 5000)
ON CONFLICT DO NOTHING;

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
    ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAgentSessionRunsPerSession', 50)
ON CONFLICT DO NOTHING;
