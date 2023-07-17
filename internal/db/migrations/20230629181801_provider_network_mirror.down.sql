DELETE FROM activity_events WHERE target_type = 'TERRAFORM_PROVIDER_VERSION_MIRROR';

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS terraform_provider_version_mirror_target_id;

DELETE FROM resource_limits WHERE id = '1d26d247-4323-4ed4-adca-94516e5cf4f9';

DROP TABLE IF EXISTS terraform_provider_platform_mirrors;
DROP TABLE IF EXISTS terraform_provider_version_mirrors;
