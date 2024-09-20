UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformModule' WHERE id = 'b46ad391-3621-4f2d-84d6-93f9a74daca4';

UPDATE resource_limits
    SET name = 'ResourceLimitAttestationsPerTerraformModule' WHERE id = '74151e8b-eae4-4b6f-972c-da3132d4b1be';

UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformProvider' WHERE id = 'f553accd-b634-4455-b0a9-718e496bb3c7';

INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
('1ac2099c-6c8d-48b8-bbf1-a618474dc07b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformModulePerTimePeriod', 100), -- number of versions per Terraform module per time period
('fdc8fba1-3dfc-4aa5-b8bb-ca38eaf42f0b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAttestationsPerTerraformModulePerTimePeriod', 100), -- number of attestations per Terraform module per time period
('87243e0a-d4c9-4b4c-8104-f7b1b37bda49', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformProviderPerTimePeriod', 100) -- number of versions per Terraform provider per time period
ON CONFLICT DO NOTHING;
