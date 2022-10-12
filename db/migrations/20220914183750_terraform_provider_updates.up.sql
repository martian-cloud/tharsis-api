ALTER TABLE terraform_providers ADD COLUMN IF NOT EXISTS repo_url VARCHAR NOT NULL DEFAULT '';
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS readme_uploaded BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE terraform_provider_versions ADD COLUMN IF NOT EXISTS latest BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_provider_versions_on_latest ON terraform_provider_versions(provider_id, latest) WHERE latest = true;
