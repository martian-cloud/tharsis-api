UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformModulePerTimePeriod' WHERE id = 'b46ad391-3621-4f2d-84d6-93f9a74daca4';

UPDATE resource_limits
    SET name = 'ResourceLimitAttestationsPerTerraformModulePerTimePeriod' WHERE id = '74151e8b-eae4-4b6f-972c-da3132d4b1be';

UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformProviderPerTimePeriod' WHERE id = 'f553accd-b634-4455-b0a9-718e496bb3c7';

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
('246822db-b982-45e6-8cd4-18b5cadf83a3', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunsPerWorkspacePerTimePeriod', 100), -- number of runs per workspace per time period
('3e479306-6a52-4d47-a14c-2488464a2ad4', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod', 100), -- number of configuration versions per workspace per time period
('9388d4a3-ed04-4ce4-899f-d1dfd4187834', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitStateVersionsPerWorkspacePerTimePeriod', 100) -- number of state versions per workspace per time period
ON CONFLICT DO NOTHING;
