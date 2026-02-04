ALTER TABLE groups ADD COLUMN provider_mirror_enabled BOOLEAN;
ALTER TABLE workspaces ADD COLUMN provider_mirror_enabled BOOLEAN;

ALTER TABLE jobs ADD COLUMN properties JSONB NOT NULL DEFAULT '{}';

-- Drop all mirrors since storage key format changed and createdBy now uses TRN
DELETE FROM terraform_provider_version_mirrors;
