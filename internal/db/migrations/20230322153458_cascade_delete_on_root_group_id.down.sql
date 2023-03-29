-- In order to drop the CASCADE DELETE clause to the foreign key constraints
-- for root_group_id for TF providers and TF modules, it is necessary to drop
-- the constraints and then add them back.

ALTER TABLE IF EXISTS terraform_modules
    DROP CONSTRAINT IF EXISTS fk_root_group_id,
    ADD CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id);

ALTER TABLE IF EXISTS terraform_providers
    DROP CONSTRAINT IF EXISTS fk_root_group_id,
    ADD CONSTRAINT fk_root_group_id FOREIGN KEY(root_group_id) REFERENCES groups(id);

-- The End.
