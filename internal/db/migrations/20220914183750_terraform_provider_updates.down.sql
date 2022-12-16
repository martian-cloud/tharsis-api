DROP INDEX IF EXISTS index_terraform_provider_versions_on_latest;

ALTER TABLE terraform_providers DROP COLUMN IF EXISTS repo_url;
ALTER TABLE terraform_provider_versions DROP COLUMN IF EXISTS readme_uploaded;
ALTER TABLE terraform_provider_versions DROP COLUMN IF EXISTS latest;
