DELETE FROM resource_limits WHERE name = 'ResourceLimitAgentSessionRunsPerSession';
DELETE FROM resource_limits WHERE name = 'ResourceLimitAgentCreditsPerUserPerMonth';
DROP TABLE IF EXISTS agent_credit_quotas;

DROP TRIGGER IF EXISTS agent_session_runs_notify_event ON agent_session_runs;
DROP TABLE IF EXISTS agent_session_messages;

DROP TABLE IF EXISTS agent_session_runs;
DROP TABLE IF EXISTS agent_sessions;
