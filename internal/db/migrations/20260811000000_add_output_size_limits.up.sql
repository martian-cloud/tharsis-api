-- Output count limit per state version.
INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
    ('1138c015-adb1-4bce-8707-661981d65b2f', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitOutputsPerStateVersion', 400) -- max number of Terraform outputs per state version
ON CONFLICT DO NOTHING;
