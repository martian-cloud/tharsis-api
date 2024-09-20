DELETE FROM resource_limits WHERE id = '1ac2099c-6c8d-48b8-bbf1-a618474dc07b';
DELETE FROM resource_limits WHERE id = 'fdc8fba1-3dfc-4aa5-b8bb-ca38eaf42f0b';
DELETE FROM resource_limits WHERE id = '87243e0a-d4c9-4b4c-8104-f7b1b37bda49';

UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformModulePerTimePeriod' WHERE id = 'b46ad391-3621-4f2d-84d6-93f9a74daca4';

UPDATE resource_limits
    SET name = 'ResourceLimitAttestationsPerTerraformModulePerTimePeriod' WHERE id = '74151e8b-eae4-4b6f-972c-da3132d4b1be';

UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformProviderPerTimePeriod' WHERE id = 'f553accd-b634-4455-b0a9-718e496bb3c7';
