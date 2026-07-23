CREATE TABLE IF NOT EXISTS object_store_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    available_at TIMESTAMP NOT NULL DEFAULT now(),
    claim_count INTEGER NOT NULL DEFAULT 0,
    object_key TEXT NOT NULL,
    run_id UUID REFERENCES runs(id) ON DELETE SET NULL,
    state_version_id UUID REFERENCES state_versions(id) ON DELETE SET NULL,
    configuration_version_id UUID REFERENCES configuration_versions(id) ON DELETE SET NULL,
    log_stream_id UUID REFERENCES log_streams(id) ON DELETE SET NULL,
    log_stream_chunk_id UUID REFERENCES log_stream_chunks(id) ON DELETE SET NULL,
    module_version_id UUID REFERENCES terraform_module_versions(id) ON DELETE SET NULL,
    provider_version_id UUID REFERENCES terraform_provider_versions(id) ON DELETE SET NULL,
    provider_platform_id UUID REFERENCES terraform_provider_platforms(id) ON DELETE SET NULL,
    provider_mirror_platform_id UUID REFERENCES terraform_provider_platform_mirrors(id) ON DELETE SET NULL,
    agent_session_id UUID REFERENCES agent_sessions(id) ON DELETE SET NULL,
    CONSTRAINT object_store_refs_unique_key UNIQUE (object_key)
);

CREATE INDEX IF NOT EXISTS index_object_store_refs_on_run_id ON object_store_refs(run_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_state_version_id ON object_store_refs(state_version_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_configuration_version_id ON object_store_refs(configuration_version_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_log_stream_id ON object_store_refs(log_stream_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_log_stream_chunk_id ON object_store_refs(log_stream_chunk_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_module_version_id ON object_store_refs(module_version_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_provider_version_id ON object_store_refs(provider_version_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_provider_platform_id ON object_store_refs(provider_platform_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_provider_mirror_platform_id ON object_store_refs(provider_mirror_platform_id);
CREATE INDEX IF NOT EXISTS index_object_store_refs_on_agent_session_id ON object_store_refs(agent_session_id);

-- Partial index for the janitor's orphan sweep: rows where every FK is NULL are candidates for deletion.
CREATE INDEX IF NOT EXISTS index_object_store_refs_orphan ON object_store_refs (available_at, created_at)
WHERE run_id IS NULL AND state_version_id IS NULL AND configuration_version_id IS NULL
  AND log_stream_id IS NULL AND log_stream_chunk_id IS NULL AND module_version_id IS NULL
  AND provider_version_id IS NULL AND provider_platform_id IS NULL
  AND provider_mirror_platform_id IS NULL AND agent_session_id IS NULL;

-- Key columns on resource tables. Nullable; backfilled below for existing rows.
-- New rows always have the column set at write time.
ALTER TABLE configuration_versions ADD COLUMN IF NOT EXISTS object_store_key VARCHAR;
ALTER TABLE state_versions ADD COLUMN IF NOT EXISTS object_store_key VARCHAR;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS variables_object_store_key VARCHAR;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS plan_cache_object_store_key VARCHAR;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS plan_json_object_store_key VARCHAR;
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS plan_diff_object_store_key VARCHAR;
ALTER TABLE terraform_module_versions ADD COLUMN IF NOT EXISTS package_object_store_key VARCHAR;
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS readme_object_store_key VARCHAR;
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS shasums_object_store_key VARCHAR;
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS shasums_signature_object_store_key VARCHAR;
ALTER TABLE terraform_provider_platforms ADD COLUMN IF NOT EXISTS binary_object_store_key VARCHAR;
ALTER TABLE terraform_provider_platform_mirrors ADD COLUMN IF NOT EXISTS object_store_key VARCHAR;
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS object_store_key VARCHAR;
ALTER TABLE agent_session_messages ADD COLUMN IF NOT EXISTS tool_content_object_store_key VARCHAR;
ALTER TABLE log_stream_chunks RENAME COLUMN object_key TO object_store_key;

-- Backfill object store key fields on resource tables so the application never needs a fallback.
UPDATE runs
    SET variables_object_store_key = format('workspaces/%s/runs/%s/variables.json', workspace_id, id)
    WHERE variables_object_store_key IS NULL;

UPDATE run_nodes n SET
    plan_cache_object_store_key = format('workspaces/%s/runs/%s/plan/%s', r.workspace_id, r.id, n.id),
    plan_json_object_store_key  = format('workspaces/%s/runs/%s/plan/%s.json', r.workspace_id, r.id, n.id),
    plan_diff_object_store_key  = format('workspaces/%s/runs/%s/plan/diff_%s.json', r.workspace_id, r.id, n.id)
    FROM runs r
    WHERE n.run_id = r.id AND n.type = 'plan'
      AND (n.plan_cache_object_store_key IS NULL OR n.plan_json_object_store_key IS NULL OR n.plan_diff_object_store_key IS NULL);

UPDATE configuration_versions
    SET object_store_key = format('workspaces/%s/configuration_versions/%s.tar.gz', workspace_id, id)
    WHERE object_store_key IS NULL;

UPDATE state_versions
    SET object_store_key = format('workspaces/%s/state_versions/%s.json', workspace_id, id)
    WHERE object_store_key IS NULL;

UPDATE log_streams
    SET object_store_key = format('logstreams/%s.txt', id)
    WHERE object_store_key IS NULL;

UPDATE terraform_provider_platform_mirrors
    SET object_store_key = format('provider-mirror/providers/%s.zip', id)
    WHERE object_store_key IS NULL;

UPDATE terraform_module_versions
    SET package_object_store_key = format('registry/modules/%s/%s/package.tar.gz', module_id, id)
    WHERE package_object_store_key IS NULL;

UPDATE terraform_provider_versions
    SET readme_object_store_key            = format('registry/providers/%s/%s/README', provider_id, id),
        shasums_object_store_key           = format('registry/providers/%s/%s/SHA256SUMS', provider_id, id),
        shasums_signature_object_store_key = format('registry/providers/%s/%s/SHA256SUMS.sig', provider_id, id)
    WHERE readme_object_store_key IS NULL
       OR shasums_object_store_key IS NULL
       OR shasums_signature_object_store_key IS NULL;

UPDATE terraform_provider_platforms pp
    SET binary_object_store_key = format(
        'registry/providers/%s/%s/platforms/%s_%s/terraform-provider-%s_%s_%s_%s.zip',
        pv.provider_id, pp.provider_version_id,
        pp.os, pp.arch,
        p.name, pv.provider_sem_version,
        pp.os, pp.arch
    )
    FROM terraform_provider_versions pv
    JOIN terraform_providers p ON p.id = pv.provider_id
    WHERE pp.provider_version_id = pv.id
      AND pp.binary_object_store_key IS NULL;

-- Backfill: insert refs for all existing S3 objects using their deterministic keys so the
-- janitor can clean up legacy objects when resources are deleted.

-- Run variables
INSERT INTO object_store_refs (object_key, run_id)
SELECT format('workspaces/%s/runs/%s/variables.json', workspace_id, id), id
FROM runs
ON CONFLICT (object_key) DO NOTHING;

-- Plan artifacts (join run_nodes since plans table was dropped in run_refactor migration)
INSERT INTO object_store_refs (object_key, run_id)
SELECT format('workspaces/%s/runs/%s/plan/%s', r.workspace_id, r.id, n.id), r.id
FROM runs r JOIN run_nodes n ON n.run_id = r.id AND n.type = 'plan'
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, run_id)
SELECT format('workspaces/%s/runs/%s/plan/%s.json', r.workspace_id, r.id, n.id), r.id
FROM runs r JOIN run_nodes n ON n.run_id = r.id AND n.type = 'plan'
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, run_id)
SELECT format('workspaces/%s/runs/%s/plan/diff_%s.json', r.workspace_id, r.id, n.id), r.id
FROM runs r JOIN run_nodes n ON n.run_id = r.id AND n.type = 'plan'
ON CONFLICT (object_key) DO NOTHING;

-- Configuration versions
INSERT INTO object_store_refs (object_key, configuration_version_id)
SELECT format('workspaces/%s/configuration_versions/%s.tar.gz', workspace_id, id), id
FROM configuration_versions
ON CONFLICT (object_key) DO NOTHING;

-- State versions
INSERT INTO object_store_refs (object_key, state_version_id)
SELECT format('workspaces/%s/state_versions/%s.json', workspace_id, id), id
FROM state_versions
ON CONFLICT (object_key) DO NOTHING;

-- Log streams (consolidated/legacy single-file key)
INSERT INTO object_store_refs (object_key, log_stream_id)
SELECT format('logstreams/%s.txt', id), id
FROM log_streams
ON CONFLICT (object_key) DO NOTHING;

-- Log stream chunks (already have UUID keys stored on the row)
INSERT INTO object_store_refs (object_key, log_stream_chunk_id)
SELECT object_store_key, id
FROM log_stream_chunks
ON CONFLICT (object_key) DO NOTHING;

-- Module version packages
INSERT INTO object_store_refs (object_key, module_version_id)
SELECT format('registry/modules/%s/%s/package.tar.gz', module_id, id), id
FROM terraform_module_versions
ON CONFLICT (object_key) DO NOTHING;

-- Module version metadata: root + one object per submodule/example. Child paths follow a fixed
-- convention ('root', 'modules/<name>', 'examples/<name>') and the names are persisted on the
-- version's submodules/examples JSONB arrays, so every key is reconstructable here.
INSERT INTO object_store_refs (object_key, module_version_id)
SELECT format('registry/modules/%s/%s/metadata/root', module_id, id), id
FROM terraform_module_versions
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, module_version_id)
SELECT format('registry/modules/%s/%s/metadata/modules/%s', mv.module_id, mv.id, sub), mv.id
FROM terraform_module_versions mv,
     jsonb_array_elements_text(CASE WHEN jsonb_typeof(mv.submodules) = 'array' THEN mv.submodules ELSE '[]'::jsonb END) AS sub
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, module_version_id)
SELECT format('registry/modules/%s/%s/metadata/examples/%s', mv.module_id, mv.id, ex), mv.id
FROM terraform_module_versions mv,
     jsonb_array_elements_text(CASE WHEN jsonb_typeof(mv.examples) = 'array' THEN mv.examples ELSE '[]'::jsonb END) AS ex
ON CONFLICT (object_key) DO NOTHING;

-- Provider version files
INSERT INTO object_store_refs (object_key, provider_version_id)
SELECT format('registry/providers/%s/%s/README', provider_id, id), id
FROM terraform_provider_versions
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, provider_version_id)
SELECT format('registry/providers/%s/%s/SHA256SUMS', provider_id, id), id
FROM terraform_provider_versions
ON CONFLICT (object_key) DO NOTHING;

INSERT INTO object_store_refs (object_key, provider_version_id)
SELECT format('registry/providers/%s/%s/SHA256SUMS.sig', provider_id, id), id
FROM terraform_provider_versions
ON CONFLICT (object_key) DO NOTHING;

-- Provider platform binaries
INSERT INTO object_store_refs (object_key, provider_platform_id)
SELECT format(
    'registry/providers/%s/%s/platforms/%s_%s/terraform-provider-%s_%s_%s_%s.zip',
    pv.provider_id,
    pp.provider_version_id,
    pp.os,
    pp.arch,
    p.name,
    pv.provider_sem_version,
    pp.os,
    pp.arch
), pp.id
FROM terraform_provider_platforms pp
JOIN terraform_provider_versions pv ON pv.id = pp.provider_version_id
JOIN terraform_providers p ON p.id = pv.provider_id
ON CONFLICT (object_key) DO NOTHING;

-- Provider mirror platforms
INSERT INTO object_store_refs (object_key, provider_mirror_platform_id)
SELECT format('provider-mirror/providers/%s.zip', id), id
FROM terraform_provider_platform_mirrors
ON CONFLICT (object_key) DO NOTHING;

-- Agent session history files
INSERT INTO object_store_refs (object_key, agent_session_id)
SELECT format('agent-sessions/%s/history', id), id
FROM agent_sessions
ON CONFLICT (object_key) DO NOTHING;

-- Backfill legacy per-message tool content keys (from before UUID-key migration).
UPDATE agent_session_messages
SET tool_content_object_store_key = format('agent-sessions/%s/messages/%s/tool-content', session_id, id)
WHERE tool_content_object_store_key IS NULL AND role = 'tool';

INSERT INTO object_store_refs (object_key, agent_session_id)
SELECT format('agent-sessions/%s/messages/%s/tool-content', session_id, id), session_id
FROM agent_session_messages
WHERE role = 'tool'
ON CONFLICT (object_key) DO NOTHING;

-- Agent session traces (one per agent session run; trace ID == run ID)
INSERT INTO object_store_refs (object_key, agent_session_id)
SELECT format('agent-sessions/%s/traces/%s', session_id, id), session_id
FROM agent_session_runs
ON CONFLICT (object_key) DO NOTHING;

-- Legacy per-job log objects (predating log streams): workspaces/<ws_id>/runs/<run_id>/logs/<job_id>.txt
-- Linked to run_id so the janitor cleans them up when the run is deleted.
INSERT INTO object_store_refs (object_key, run_id)
SELECT format('workspaces/%s/runs/%s/logs/%s.txt', workspace_id, run_id, id), run_id
FROM jobs
ON CONFLICT (object_key) DO NOTHING;
