-- Backfill the actor type/id/trn columns added in 20261014190000_add_actor_columns
-- from the legacy string column each one sits beside: created_by on every
-- table, plus run_nodes.apply_triggered_by, runs.force_canceled_by and
-- run_gates.overridden_by.
--
-- This is a separate migration so the ACCESS EXCLUSIVE locks taken by those
-- ALTER TABLEs are released before any row is rewritten. Reads keep running
-- throughout. Writes to runs, workspaces, runners and maintenance_mode wait until
-- this migration commits, because disabling their triggers (see below) locks
-- those tables against writes. On every other table the UPDATEs take only ROW
-- EXCLUSIVE locks, so only a write to a row this migration has already updated
-- waits.
--
-- The legacy column is due to be removed, so this resolves every subject it can,
-- accepting that a few rows may be attributed imperfectly. Each subject format a
-- caller writes is mapped to the principal that matches it now:
--
--   * 'system' and 'scim' map to their actor types, with a NULL id.
--   * An email (users) maps to the user that holds it now.
--   * A service account or VCS provider path ("<group path>/<name>") maps to the
--     record with that name in the group currently at that path.
--   * A job TRN ("trn:job:<workspace path>/<job global id>") maps to the job by
--     its global ID, which is stable even if the workspace has moved.
--   * A bare job ID (a UUID), which jobs recorded instead of their TRN before
--     February 2026, maps to the job with that ID.
--
-- A path shared by a service account and a VCS provider with the same name
-- names the VCS provider only on a run or configuration version created from a
-- VCS event, since VCS providers create nothing else; everywhere else it names
-- the service account.
--
-- A subject that matches no current principal (its principal was deleted or its
-- group has moved) still gets the actor type its format implies, with the
-- placeholder id 00000000-0000-0000-0000-000000000000 (models.UnmappedActorID).
-- That id marks an actor recorded before typed actors whose principal could not
-- be identified, so code that requires an id keeps working and the principal
-- simply resolves to nothing. The type comes from the format: an email is a user,
-- a job TRN or a UUID is a job, and a path is a service account, except that a
-- run or configuration version created from a VCS event is taken to be a VCS
-- provider's. Only a subject in none of these formats is left NULL.
--
-- Known imperfections, all from matching on what a string names today rather
-- than when the row was written:
--
--   * A deleted user whose email was later given to a new user (for example a
--     SCIM deprovision followed by a reprovision) is attributed to the new user.
--   * A deleted service account or VCS provider whose name was reused in the same
--     group, or a group moved away and replaced by another at the same path, is
--     attributed to the record at that path now.
--   * A VCS run triggered manually by a service account is attributed to a VCS
--     provider, because its configuration version has a VCS event, when the
--     service account has since been deleted (typed only) or shares its name
--     with a VCS provider in the same group.
--
-- Every row that gets a service account, VCS provider or job actor also gets
-- its TRN from the subject it recorded, whether or not the principal
-- was matched; see pg_temp.actor_trn below. A bare job ID names no workspace, so
-- a job matched by it gets a TRN built from the workspace path the job is in
-- now, and an unmatched one gets no TRN.
--
-- Run gate approvals also record the approver's user_id or service_account_id,
-- which identify the approver exactly, so those are used before the subject.
--
-- Every statement is restricted to rows whose actor type is still NULL, so this
-- migration is idempotent and can be re-run after later code populates more rows.
--
-- runs, workspaces, runners and maintenance_mode have notify_event triggers, so
-- without the statements below every backfilled row would publish an UPDATE
-- event carrying the whole row to every subscriber when this migration commits
-- at API startup. The rows change only in columns no subscriber reads, so the
-- triggers are disabled for the backfill and re-enabled at the end. ALTER TABLE
-- is transactional, so a failed migration leaves them enabled. It locks the
-- tables against writes until the migration commits, which is acceptable
-- because the backfill runs during a maintenance window.

ALTER TABLE runs DISABLE TRIGGER runs_notify_event;
ALTER TABLE workspaces DISABLE TRIGGER workspaces_notify_event;
ALTER TABLE runners DISABLE TRIGGER runners_notify_event;
ALTER TABLE maintenance_mode DISABLE TRIGGER maintenance_mode_notify_event;

-- Every subject a caller writes, mapped to its actor. actor_trn is set only
-- where the TRN cannot be built from the subject (see pg_temp.actor_trn).
-- Dropped when this migration's transaction commits.
CREATE TEMP TABLE actor_backfill_subjects (
    subject VARCHAR NOT NULL,
    actor_type VARCHAR NOT NULL,
    actor_id UUID,
    actor_trn VARCHAR
) ON COMMIT DROP;

INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id) VALUES
    ('system', 'system', NULL),
    ('scim', 'scim', NULL);

INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id)
    SELECT u.email, 'user', u.id FROM users u;

INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id)
    SELECT n.path || '/' || sa.name, 'service_account', sa.id
    FROM service_accounts sa JOIN namespaces n ON n.group_id = sa.group_id;

INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id)
    SELECT n.path || '/' || vp.name, 'vcs_provider', vp.id
    FROM vcs_providers vp JOIN namespaces n ON n.group_id = vp.group_id;

-- Jobs are keyed by global ID alone, which the UPDATEs below strip a job TRN
-- down to, so a job still matches after its workspace has moved. A global ID is
-- the unpadded base64url encoding of 'J_<uuid>' (see internal/gid); 'J_' plus a
-- UUID is 38 bytes, which encodes to 52 characters, so encode() does not insert
-- a line break. It contains no '@' or '/', so it cannot collide with an email or
-- a path.
INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id)
    SELECT rtrim(translate(encode(convert_to('J_' || j.id::text, 'UTF8'), 'base64'), '+/', '-_'), '='),
        'job', j.id
    FROM jobs j;

-- Before job TRNs, a job recorded its bare ID. A UUID contains no '@' or '/' and
-- is shorter than a global ID, so it cannot collide with any other subject. It
-- names no workspace, so the TRN is built from the workspace the job is in now.
INSERT INTO actor_backfill_subjects (subject, actor_type, actor_id, actor_trn)
    SELECT j.id::text, 'job', j.id,
        'trn:job:' || n.path || '/' || rtrim(translate(encode(convert_to('J_' || j.id::text, 'UTF8'), 'base64'), '+/', '-_'), '=')
    FROM jobs j LEFT JOIN namespaces n ON n.workspace_id = j.workspace_id;

-- A path shared by a service account and a VCS provider with the same name
-- names the VCS provider only on a run or configuration version created from a
-- VCS event (see above). Those VCS provider rows move here, for the runs and
-- configuration versions UPDATEs that resolve VCS event rows first; the service
-- account row stays for everything else. Dropped when this migration's
-- transaction commits.
CREATE TEMP TABLE actor_backfill_vcs_provider_subjects ON COMMIT DROP AS
    SELECT subject, actor_id FROM actor_backfill_subjects
    WHERE actor_type = 'vcs_provider' AND subject IN (
        SELECT subject FROM actor_backfill_subjects WHERE actor_type = 'service_account'
    );

DELETE FROM actor_backfill_subjects WHERE actor_type = 'vcs_provider' AND subject IN (
    SELECT subject FROM actor_backfill_vcs_provider_subjects
);

-- Any subject that still matches more than one principal is ambiguous
-- everywhere, so it is matched to none. The placeholder pass below then gives
-- it the actor type its format implies and the placeholder id.
DELETE FROM actor_backfill_subjects WHERE subject IN (
    SELECT subject FROM actor_backfill_subjects GROUP BY subject HAVING count(*) > 1
);

CREATE UNIQUE INDEX ON actor_backfill_subjects (subject);
CREATE UNIQUE INDEX ON actor_backfill_vcs_provider_subjects (subject);
ANALYZE actor_backfill_subjects;
ANALYZE actor_backfill_vcs_provider_subjects;

-- The actor type a subject's format implies, for subjects that match no current
-- principal. from_vcs_event is true when the row came from a VCS event, where a
-- path names a VCS provider rather than a service account. Dropped at the end of
-- this migration.
CREATE FUNCTION pg_temp.actor_type_from_subject(subject VARCHAR, from_vcs_event BOOLEAN) RETURNS VARCHAR AS $$
    SELECT CASE
        WHEN subject LIKE 'trn:job:%' THEN 'job'
        WHEN subject ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN 'job'
        WHEN subject LIKE '%@%' THEN 'user'
        WHEN subject LIKE '%/%' AND from_vcs_event THEN 'vcs_provider'
        WHEN subject LIKE '%/%' THEN 'service_account'
    END
$$ LANGUAGE sql IMMUTABLE;

-- The TRN for an actor of the given type, taken from the subject the
-- row recorded at the time: the principal's TRN. A service account or VCS
-- provider recorded its resource path, so the TRN is built from it; a job
-- recorded its TRN ("trn:job:<workspace path>/<job global id>") already, which is
-- kept only when it has that shape. Users, system and scim get no TRN (see
-- models.ActorType). Dropped at the end of this migration.
CREATE FUNCTION pg_temp.actor_trn(actor_type VARCHAR, subject VARCHAR) RETURNS VARCHAR AS $$
    SELECT CASE actor_type
        WHEN 'service_account' THEN 'trn:service_account:' || subject
        WHEN 'vcs_provider' THEN 'trn:vcs_provider:' || subject
        WHEN 'job' THEN substring(subject FROM '^(trn:job:.+/[^/]+)$')
    END
$$ LANGUAGE sql IMMUTABLE;

-- Approvals identify the approver by foreign key, which is exact.
UPDATE run_gate_approvals SET created_by_actor_type = 'user', created_by_actor_id = user_id
  WHERE created_by_actor_type IS NULL AND user_id IS NOT NULL;
UPDATE run_gate_approvals SET created_by_actor_type = 'service_account', created_by_actor_id = service_account_id,
    created_by_actor_trn = pg_temp.actor_trn('service_account', created_by)
  WHERE created_by_actor_type IS NULL AND service_account_id IS NOT NULL;

-- A path shared by a service account and a VCS provider names the VCS provider
-- on a run or configuration version created from a VCS event. These run before
-- the subject matches below, which then attribute the remaining rows with that
-- path to the service account.
UPDATE configuration_versions SET created_by_actor_type = 'vcs_provider', created_by_actor_id = v.actor_id,
    created_by_actor_trn = pg_temp.actor_trn('vcs_provider', configuration_versions.created_by)
  FROM actor_backfill_vcs_provider_subjects v
  WHERE configuration_versions.created_by_actor_type IS NULL AND configuration_versions.vcs_event_id IS NOT NULL
    AND v.subject = configuration_versions.created_by;

UPDATE runs SET created_by_actor_type = 'vcs_provider', created_by_actor_id = v.actor_id,
    created_by_actor_trn = pg_temp.actor_trn('vcs_provider', runs.created_by)
  FROM actor_backfill_vcs_provider_subjects v, configuration_versions cv
  WHERE runs.created_by_actor_type IS NULL AND cv.id = runs.configuration_version_id AND cv.vcs_event_id IS NOT NULL
    AND v.subject = runs.created_by;

UPDATE announcements SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, announcements.created_by)) FROM actor_backfill_subjects s
  WHERE announcements.created_by_actor_type IS NULL AND s.subject = regexp_replace(announcements.created_by, '^trn:job:.*/', '');

UPDATE configuration_versions SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, configuration_versions.created_by)) FROM actor_backfill_subjects s
  WHERE configuration_versions.created_by_actor_type IS NULL AND s.subject = regexp_replace(configuration_versions.created_by, '^trn:job:.*/', '');

UPDATE federated_registries SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, federated_registries.created_by)) FROM actor_backfill_subjects s
  WHERE federated_registries.created_by_actor_type IS NULL AND s.subject = regexp_replace(federated_registries.created_by, '^trn:job:.*/', '');

UPDATE gpg_keys SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, gpg_keys.created_by)) FROM actor_backfill_subjects s
  WHERE gpg_keys.created_by_actor_type IS NULL AND s.subject = regexp_replace(gpg_keys.created_by, '^trn:job:.*/', '');

UPDATE groups SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, groups.created_by)) FROM actor_backfill_subjects s
  WHERE groups.created_by_actor_type IS NULL AND s.subject = regexp_replace(groups.created_by, '^trn:job:.*/', '');

UPDATE maintenance_mode SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, maintenance_mode.created_by)) FROM actor_backfill_subjects s
  WHERE maintenance_mode.created_by_actor_type IS NULL AND s.subject = regexp_replace(maintenance_mode.created_by, '^trn:job:.*/', '');

UPDATE managed_identities SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, managed_identities.created_by)) FROM actor_backfill_subjects s
  WHERE managed_identities.created_by_actor_type IS NULL AND s.subject = regexp_replace(managed_identities.created_by, '^trn:job:.*/', '');

UPDATE package_versions SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, package_versions.created_by)) FROM actor_backfill_subjects s
  WHERE package_versions.created_by_actor_type IS NULL AND s.subject = regexp_replace(package_versions.created_by, '^trn:job:.*/', '');

UPDATE packages SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, packages.created_by)) FROM actor_backfill_subjects s
  WHERE packages.created_by_actor_type IS NULL AND s.subject = regexp_replace(packages.created_by, '^trn:job:.*/', '');

UPDATE policies SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, policies.created_by)) FROM actor_backfill_subjects s
  WHERE policies.created_by_actor_type IS NULL AND s.subject = regexp_replace(policies.created_by, '^trn:job:.*/', '');

UPDATE roles SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, roles.created_by)) FROM actor_backfill_subjects s
  WHERE roles.created_by_actor_type IS NULL AND s.subject = regexp_replace(roles.created_by, '^trn:job:.*/', '');

UPDATE run_gate_approvals SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, run_gate_approvals.created_by)) FROM actor_backfill_subjects s
  WHERE run_gate_approvals.created_by_actor_type IS NULL AND s.subject = regexp_replace(run_gate_approvals.created_by, '^trn:job:.*/', '');

UPDATE runners SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, runners.created_by)) FROM actor_backfill_subjects s
  WHERE runners.created_by_actor_type IS NULL AND s.subject = regexp_replace(runners.created_by, '^trn:job:.*/', '');

UPDATE runs SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, runs.created_by)) FROM actor_backfill_subjects s
  WHERE runs.created_by_actor_type IS NULL AND s.subject = regexp_replace(runs.created_by, '^trn:job:.*/', '');

UPDATE scim_tokens SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, scim_tokens.created_by)) FROM actor_backfill_subjects s
  WHERE scim_tokens.created_by_actor_type IS NULL AND s.subject = regexp_replace(scim_tokens.created_by, '^trn:job:.*/', '');

UPDATE service_accounts SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, service_accounts.created_by)) FROM actor_backfill_subjects s
  WHERE service_accounts.created_by_actor_type IS NULL AND s.subject = regexp_replace(service_accounts.created_by, '^trn:job:.*/', '');

UPDATE state_versions SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, state_versions.created_by)) FROM actor_backfill_subjects s
  WHERE state_versions.created_by_actor_type IS NULL AND s.subject = regexp_replace(state_versions.created_by, '^trn:job:.*/', '');

UPDATE terraform_module_attestations SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_module_attestations.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_module_attestations.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_module_attestations.created_by, '^trn:job:.*/', '');

UPDATE terraform_module_versions SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_module_versions.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_module_versions.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_module_versions.created_by, '^trn:job:.*/', '');

UPDATE terraform_modules SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_modules.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_modules.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_modules.created_by, '^trn:job:.*/', '');

UPDATE terraform_provider_platforms SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_provider_platforms.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_provider_platforms.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_provider_platforms.created_by, '^trn:job:.*/', '');

UPDATE terraform_provider_version_mirrors SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_provider_version_mirrors.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_provider_version_mirrors.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_provider_version_mirrors.created_by, '^trn:job:.*/', '');

UPDATE terraform_provider_versions SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_provider_versions.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_provider_versions.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_provider_versions.created_by, '^trn:job:.*/', '');

UPDATE terraform_providers SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, terraform_providers.created_by)) FROM actor_backfill_subjects s
  WHERE terraform_providers.created_by_actor_type IS NULL AND s.subject = regexp_replace(terraform_providers.created_by, '^trn:job:.*/', '');

UPDATE vcs_providers SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, vcs_providers.created_by)) FROM actor_backfill_subjects s
  WHERE vcs_providers.created_by_actor_type IS NULL AND s.subject = regexp_replace(vcs_providers.created_by, '^trn:job:.*/', '');

UPDATE workspace_role_bindings SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, workspace_role_bindings.created_by)) FROM actor_backfill_subjects s
  WHERE workspace_role_bindings.created_by_actor_type IS NULL AND s.subject = regexp_replace(workspace_role_bindings.created_by, '^trn:job:.*/', '');

UPDATE workspace_vcs_provider_links SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, workspace_vcs_provider_links.created_by)) FROM actor_backfill_subjects s
  WHERE workspace_vcs_provider_links.created_by_actor_type IS NULL AND s.subject = regexp_replace(workspace_vcs_provider_links.created_by, '^trn:job:.*/', '');

UPDATE workspaces SET created_by_actor_type = s.actor_type, created_by_actor_id = s.actor_id, created_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, workspaces.created_by)) FROM actor_backfill_subjects s
  WHERE workspaces.created_by_actor_type IS NULL AND s.subject = regexp_replace(workspaces.created_by, '^trn:job:.*/', '');

-- The same conversion for the other attribution columns. Only apply nodes have
-- an apply_triggered_by value, and only force-canceled runs and overridden gates
-- have the other two, so the remaining rows are skipped by the value match.

UPDATE run_nodes SET apply_triggered_by_actor_type = s.actor_type, apply_triggered_by_actor_id = s.actor_id, apply_triggered_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, run_nodes.apply_triggered_by)) FROM actor_backfill_subjects s
  WHERE run_nodes.apply_triggered_by_actor_type IS NULL AND s.subject = regexp_replace(run_nodes.apply_triggered_by, '^trn:job:.*/', '');

UPDATE runs SET force_canceled_by_actor_type = s.actor_type, force_canceled_by_actor_id = s.actor_id, force_canceled_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, runs.force_canceled_by)) FROM actor_backfill_subjects s
  WHERE runs.force_canceled_by_actor_type IS NULL AND s.subject = regexp_replace(runs.force_canceled_by, '^trn:job:.*/', '');

UPDATE run_gates SET overridden_by_actor_type = s.actor_type, overridden_by_actor_id = s.actor_id, overridden_by_actor_trn = COALESCE(s.actor_trn, pg_temp.actor_trn(s.actor_type, run_gates.overridden_by)) FROM actor_backfill_subjects s
  WHERE run_gates.overridden_by_actor_type IS NULL AND s.subject = regexp_replace(run_gates.overridden_by, '^trn:job:.*/', '');

-- Subjects that matched no current principal: type from the format, placeholder
-- id. VCS providers only ever create runs and configuration versions, so those
-- are the only rows where a path can name one. Their type depends on whether the
-- row came from a VCS event, so it is worked out once in a derived table and
-- used for both the type and the TRN, which must agree.

UPDATE configuration_versions SET created_by_actor_type = t.actor_type,
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(t.actor_type, configuration_versions.created_by)
  FROM (SELECT id, pg_temp.actor_type_from_subject(created_by, vcs_event_id IS NOT NULL) AS actor_type
        FROM configuration_versions WHERE created_by_actor_type IS NULL) t
  WHERE configuration_versions.id = t.id AND t.actor_type IS NOT NULL;

UPDATE runs SET created_by_actor_type = t.actor_type,
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(t.actor_type, runs.created_by)
  FROM (SELECT r.id, pg_temp.actor_type_from_subject(r.created_by, cv.vcs_event_id IS NOT NULL) AS actor_type
        FROM runs r LEFT JOIN configuration_versions cv ON cv.id = r.configuration_version_id
        WHERE r.created_by_actor_type IS NULL) t
  WHERE runs.id = t.id AND t.actor_type IS NOT NULL;

UPDATE announcements SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE federated_registries SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE gpg_keys SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE groups SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE maintenance_mode SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE managed_identities SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE package_versions SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE packages SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE policies SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE roles SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE run_gate_approvals SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE runners SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE scim_tokens SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE service_accounts SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE state_versions SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_module_attestations SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_module_versions SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_modules SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_provider_platforms SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_provider_version_mirrors SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_provider_versions SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE terraform_providers SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE vcs_providers SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE workspace_role_bindings SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE workspace_vcs_provider_links SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE workspaces SET created_by_actor_type = pg_temp.actor_type_from_subject(created_by, false),
    created_by_actor_id = '00000000-0000-0000-0000-000000000000',
    created_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(created_by, false), created_by)
  WHERE created_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(created_by, false) IS NOT NULL;

UPDATE run_nodes SET apply_triggered_by_actor_type = pg_temp.actor_type_from_subject(apply_triggered_by, false),
    apply_triggered_by_actor_id = '00000000-0000-0000-0000-000000000000',
    apply_triggered_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(apply_triggered_by, false), apply_triggered_by)
  WHERE apply_triggered_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(apply_triggered_by, false) IS NOT NULL;

UPDATE runs SET force_canceled_by_actor_type = pg_temp.actor_type_from_subject(force_canceled_by, false),
    force_canceled_by_actor_id = '00000000-0000-0000-0000-000000000000',
    force_canceled_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(force_canceled_by, false), force_canceled_by)
  WHERE force_canceled_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(force_canceled_by, false) IS NOT NULL;

UPDATE run_gates SET overridden_by_actor_type = pg_temp.actor_type_from_subject(overridden_by, false),
    overridden_by_actor_id = '00000000-0000-0000-0000-000000000000',
    overridden_by_actor_trn = pg_temp.actor_trn(pg_temp.actor_type_from_subject(overridden_by, false), overridden_by)
  WHERE overridden_by_actor_type IS NULL AND pg_temp.actor_type_from_subject(overridden_by, false) IS NOT NULL;

DROP FUNCTION pg_temp.actor_type_from_subject(VARCHAR, BOOLEAN);
DROP FUNCTION pg_temp.actor_trn(VARCHAR, VARCHAR);

ALTER TABLE runs ENABLE TRIGGER runs_notify_event;
ALTER TABLE workspaces ENABLE TRIGGER workspaces_notify_event;
ALTER TABLE runners ENABLE TRIGGER runners_notify_event;
ALTER TABLE maintenance_mode ENABLE TRIGGER maintenance_mode_notify_event;
