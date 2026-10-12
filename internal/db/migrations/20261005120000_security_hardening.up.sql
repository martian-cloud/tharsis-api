-- The managed identities a plan or apply node was authorized to use: the identities assigned to the
-- workspace when the node's stage passed its managed identity access rules. A job may only create
-- credentials for an identity on its own node, so an identity assigned mid-run is not usable by that
-- run.
--
-- Existing nodes are backfilled below for the runs that are not final. A NULL list authorizes no
-- identities. An empty array means none were authorized.
ALTER TABLE run_nodes ADD COLUMN IF NOT EXISTS managed_identity_ids JSONB;

-- Records the managed identities of plan and apply nodes that were created before the
-- managed_identity_ids column existed, so every node a job can still run on has a recorded list and
-- a NULL list no longer needs to be treated as "any identity assigned to the workspace".
--
-- Runs that can never run another job (applied, planned_and_finished) are skipped, except the applied
-- run behind each workspace's current state version: an assessment run reuses that run's plan
-- identities. Canceled and errored runs are included because their plan or apply can be retried
-- (RetryRunNode), and discarded runs because they can be undiscarded (UndiscardRun). Each node gets
-- the identities currently assigned to the run's workspace, which is what it was authorized for
-- before the identities were recorded. An apply that hasn't been started yet gets its list replaced
-- by the apply-stage rules when it starts.
UPDATE run_nodes rn
SET managed_identity_ids = COALESCE(
    (
        SELECT jsonb_agg(wmir.managed_identity_id::text ORDER BY wmir.managed_identity_id)
        FROM workspace_managed_identity_relation wmir
        WHERE wmir.workspace_id = r.workspace_id
    ),
    '[]'::jsonb
)
FROM runs r
WHERE rn.run_id = r.id
  AND rn.type IN ('plan', 'apply')
  AND rn.managed_identity_ids IS NULL
  AND (
    r.status NOT IN ('applied', 'planned_and_finished')
    OR r.id IN (
        SELECT sv.run_id
        FROM workspaces w
        JOIN state_versions sv ON sv.id = w.current_state_version_id
        WHERE sv.run_id IS NOT NULL
    )
  );

-- The issuer URL of the IDP whose SCIM client manages the team. A SCIM client may only see, update,
-- delete or adopt teams of its own IDP. Existing SCIM teams are left NULL, so no SCIM client can manage
-- them until this is backfilled.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS scim_idp_issuer_url VARCHAR;

-- The issuer URL of the IDP whose SCIM client manages the user. A SCIM client may only see, update or
-- delete users of its own IDP. Existing SCIM users are left NULL, so no SCIM client can manage them until
-- this is backfilled.
ALTER TABLE users ADD COLUMN IF NOT EXISTS scim_idp_issuer_url VARCHAR;

-- A SCIM external ID is only unique within the IDP whose SCIM client manages the user or team. Rows
-- managed by no IDP (scim_idp_issuer_url NULL) don't conflict, since NULLs are distinct.
DROP INDEX IF EXISTS index_users_on_scim_external_id;
CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_scim_idp_issuer_url_scim_external_id ON users(scim_idp_issuer_url, scim_external_id);

DROP INDEX IF EXISTS index_teams_on_scim_external_id;
CREATE UNIQUE INDEX IF NOT EXISTS index_teams_on_scim_idp_issuer_url_scim_external_id ON teams(scim_idp_issuer_url, scim_external_id);

-- Write lease: set while a WriteLogs request rewrites the chunk's object, so a concurrent writer can't
-- overwrite it with a stale prefix. NULL (or a time in the past) means the chunk is free to claim.
ALTER TABLE log_stream_chunks ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMP;
-- CRC-32 (IEEE) of the chunk's committed bytes [0, size). NULL for chunks written before checksums were
-- tracked (and chunks continuing from one), which are not verified.
ALTER TABLE log_stream_chunks ADD COLUMN IF NOT EXISTS checksum BIGINT;

-- Max number of concurrent jobs a runner can execute; 0 means no limit.
ALTER TABLE runners ADD COLUMN IF NOT EXISTS job_limit INTEGER NOT NULL DEFAULT 0;
