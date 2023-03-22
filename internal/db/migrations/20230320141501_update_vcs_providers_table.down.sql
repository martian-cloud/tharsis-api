ALTER TABLE vcs_providers ADD COLUMN hostname VARCHAR;

UPDATE vcs_providers set hostname = substring(url from '.*://([^/]*)');

ALTER TABLE vcs_providers
    DROP COLUMN url,
    ALTER COLUMN hostname SET NOT NULL;
