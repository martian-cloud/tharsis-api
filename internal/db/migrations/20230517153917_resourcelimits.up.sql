CREATE TABLE IF NOT EXISTS resource_limits (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    value INTEGER NOT NULL
);
CREATE UNIQUE INDEX index_resource_limits_on_name ON resource_limits(name);

INSERT INTO resource_limits
    (id, version, created_at, updated_at, name, value)
VALUES
    ('04c35a50-303d-42c4-bade-c5c4d4da5ac3', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitSubgroupsPerParent', 1000), -- number of subgroups directly under one parent group
    ('e626308b-dc6a-4f5c-a8f9-c90223579cc2', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitGroupTreeDepth', 20), -- depth of the group tree
    ('e923f667-3dd1-4376-a973-1d5002249c65', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitWorkspacesPerGroup', 1000), -- number of workspaces directly under one group
    ('0b5e750d-30d8-462f-96b4-7c7730167857', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitServiceAccountsPerGroup', 1000), -- number of service accounts per group
    ('1a796b82-ff6e-4ed6-a135-66e3d987f1de', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitRunnerAgentsPerGroup', 1000), -- number of runner agents per group
    ('12b6c63f-d189-47c5-b9f2-c9c799c171b0', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVariablesPerNamespace', 1000), -- number of variables per group or workspace
    ('11f19e67-5e50-46a0-9e45-3d0fc6ac6c5b', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitGPGKeysPerGroup', 1000), -- number of GPG keys per group
    ('86e34c1a-24af-4ff9-86d8-e541768953b5', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentitiesPerGroup', 1000), -- number of managed identities per group
    ('6b99fe91-91eb-4375-917f-511a30ac7ad9', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentityAliasesPerManagedIdentity', 1000), -- number of managed identity aliases per managed identity
    ('87fe08b7-7ed1-4688-ae21-3b17dfaad198', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAssignedManagedIdentitiesPerWorkspace', 1000), -- number of assigned managed identities per workspace
    ('c22c7257-1c87-4dbd-9ace-0830837597c7', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitManagedIdentityAccessRulesPerManagedIdentity', 1000), -- number of managed identity access rules per managed identity
    ('ab1d78ae-f726-4486-b0fa-381e5507d987', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformModulesPerGroup', 1000), -- number of Terraform modules per group
    ('b46ad391-3621-4f2d-84d6-93f9a74daca4', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformModule', 1000), -- number of versions per Terraform module
    ('74151e8b-eae4-4b6f-972c-da3132d4b1be', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitAttestationsPerTerraformModule', 1000), -- number of attestations per Terraform module
    ('fd759edd-12d7-4960-9384-2cfa48353c51', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitTerraformProvidersPerGroup', 1000), -- number of Terraform providers per group
    ('f553accd-b634-4455-b0a9-718e496bb3c7', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVersionsPerTerraformProvider', 1000), -- number of versions per Terraform provider
    ('6fadb906-ef65-411c-9f20-60b7c5031d71', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitPlatformsPerTerraformProviderVersion', 1000), -- number of platforms per Terraform provider version
    ('efd32ed2-b2f5-40a2-a207-6b18eaa06c86', 1, CURRENT_TIMESTAMP(7), CURRENT_TIMESTAMP(7), 'ResourceLimitVCSProvidersPerGroup', 1000) -- number of VCS providers per group
ON CONFLICT DO NOTHING;
