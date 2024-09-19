UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformModule' WHERE id = 'b46ad391-3621-4f2d-84d6-93f9a74daca4';

UPDATE resource_limits
    SET name = 'ResourceLimitAttestationsPerTerraformModule' WHERE id = '74151e8b-eae4-4b6f-972c-da3132d4b1be';

UPDATE resource_limits
    SET name = 'ResourceLimitVersionsPerTerraformProvider' WHERE id = 'f553accd-b634-4455-b0a9-718e496bb3c7';

DELETE FROM resource_limits WHERE id = '246822db-b982-45e6-8cd4-18b5cadf83a3';
DELETE FROM resource_limits WHERE id = '3e479306-6a52-4d47-a14c-2488464a2ad4';
DELETE FROM resource_limits WHERE id = '9388d4a3-ed04-4ce4-899f-d1dfd4187834';
