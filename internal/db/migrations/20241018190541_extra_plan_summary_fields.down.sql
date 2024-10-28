ALTER TABLE plans
    DROP COLUMN IF EXISTS resource_imports,
    DROP COLUMN IF EXISTS resource_drift,
    DROP COLUMN IF EXISTS output_additions,
    DROP COLUMN IF EXISTS output_changes,
    DROP COLUMN IF EXISTS output_destructions,
    DROP COLUMN IF EXISTS diff_size;
