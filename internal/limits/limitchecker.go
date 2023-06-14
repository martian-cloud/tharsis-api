// Package limits package
package limits

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ResourceLimitName is an enum for the names that will be used as keys when doing the checks.
type ResourceLimitName string

// ResourceLimitName constants
const (
	ResourceLimitSubgroupsPerParent                           ResourceLimitName = "SubgroupsPerParent"
	ResourceLimitGroupTreeDepth                               ResourceLimitName = "GroupTreeDepth"
	ResourceLimitWorkspacesPerGroup                           ResourceLimitName = "WorkspacesPerGroup"
	ResourceLimitServiceAccountsPerGroup                      ResourceLimitName = "ServiceAccountsPerGroup"
	ResourceLimitRunnerAgentsPerGroup                         ResourceLimitName = "RunnerAgentsPerGroup"
	ResourceLimitVariablesPerNamespace                        ResourceLimitName = "VariablesPerNamespace"
	ResourceLimitGPGKeysPerGroup                              ResourceLimitName = "GPGKeysPerGroup"
	ResourceLimitManagedIdentitiesPerGroup                    ResourceLimitName = "ManagedIdentitiesPerGroup"
	ResourceLimitManagedIdentityAliasesPerManagedIdentity     ResourceLimitName = "ManagedIdentityAliasesPerManagedIdentity"
	ResourceLimitAssignedManagedIdentitiesPerWorkspace        ResourceLimitName = "AssignedManagedIdentitiesPerWorkspace"
	ResourceLimitManagedIdentityAccessRulesPerManagedIdentity ResourceLimitName = "ManagedIdentityAccessRulesPerManagedIdentity"
	ResourceLimitTerraformModulesPerGroup                     ResourceLimitName = "TerraformModulesPerGroup"
	ResourceLimitVersionsPerTerraformModule                   ResourceLimitName = "VersionsPerTerraformModule"
	ResourceLimitAttestationsPerTerraformModule               ResourceLimitName = "AttestationsPerTerraformModule"
	ResourceLimitTerraformProvidersPerGroup                   ResourceLimitName = "TerraformProvidersPerGroup"
	ResourceLimitVersionsPerTerraformProvider                 ResourceLimitName = "VersionsPerTerraformProvider"
	ResourceLimitPlatformsPerTerraformProviderVersion         ResourceLimitName = "PlatformsPerTerraformProviderVersion"
	ResourceLimitVCSProvidersPerGroup                         ResourceLimitName = "VCSProvidersPerGroup"
)

// LimitChecker implements functionality related to resource limits.
type LimitChecker interface {
	CheckLimit(ctx context.Context, name ResourceLimitName, toCheck int32) error
}

type limitChecker struct {
	dbClient *db.Client
}

// NewLimitChecker creates an instance of LimitChecker
func NewLimitChecker(
	dbClient *db.Client,
) LimitChecker {
	return &limitChecker{
		dbClient: dbClient,
	}
}

// CheckLimit returns an error or nil based on a limit check.
// The returned error is already wrapped if appropriate.
// The toCheck argument is int32 rather than int, because most calls come from something.PageInfo.TotalCount.
func (c *limitChecker) CheckLimit(ctx context.Context, name ResourceLimitName, toCheck int32) error {
	limit, err := c.dbClient.ResourceLimits.GetResourceLimit(ctx, string(name))
	if err != nil {
		return err
	}
	if limit == nil {
		return errors.New(errors.EInvalid, "invalid resource limit name: %s", name)
	}

	if int(toCheck) > limit.Value {
		return errors.New(errors.EInvalid, "for limit %s: value %d exceeds limit of %d", name, toCheck, limit.Value)
	}

	// A valid limit value was found, and there is no violation.
	return nil
}
