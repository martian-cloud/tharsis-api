ALTER TABLE vcs_providers ADD COLUMN url VARCHAR;

UPDATE vcs_providers SET url = 'https://' || hostname;

ALTER TABLE vcs_providers
    DROP COLUMN hostname,
    ALTER COLUMN url SET NOT NULL;
