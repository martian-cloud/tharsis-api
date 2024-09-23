INSERT INTO resource_limits
(id, version, created_at, updated_at, name, value)
VALUES
('b46ad391-3621-4f2d-84d6-93f9a74daca4', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformModule', 1000), -- number of versions per Terraform module
('74151e8b-eae4-4b6f-972c-da3132d4b1be', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAttestationsPerTerraformModule', 1000), -- number of attestations per Terraform module
('f553accd-b634-4455-b0a9-718e496bb3c7', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformProvider', 1000) -- number of versions per Terraform provider
ON CONFLICT DO NOTHING;
