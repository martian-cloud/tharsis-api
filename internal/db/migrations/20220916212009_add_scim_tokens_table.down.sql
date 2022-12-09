DROP TABLE IF EXISTS scim_tokens;

ALTER table users
    DROP COLUMN IF EXISTS scim_external_id,
    DROP COLUMN IF EXISTS active;

ALTER TABLE teams
    DROP COLUMN IF EXISTS scim_external_id;
