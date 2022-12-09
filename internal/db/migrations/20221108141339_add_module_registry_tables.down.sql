DELETE FROM activity_events WHERE target_type = 'TERRAFORM_MODULE';
DELETE FROM activity_events WHERE target_type = 'TERRAFORM_MODULE_VERSION';

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS terraform_module_target_id;

ALTER TABLE activity_events
    DROP COLUMN IF EXISTS terraform_module_version_target_id;

DROP TABLE IF EXISTS terraform_module_versions;
DROP TABLE IF EXISTS terraform_modules;
