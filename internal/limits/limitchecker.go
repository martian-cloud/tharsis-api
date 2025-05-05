// Package limits package
package limits

//go:generate go tool mockery --name LimitChecker --inpackage --case underscore

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// ResourceLimitTimePeriod is the time period used for time-based resource limits.
	// Only resources created within the last time period will account towards the limit.
	ResourceLimitTimePeriod = 1 * time.Hour
)

// ResourceLimitName is an enum for the names that will be used as keys when doing the checks.
type ResourceLimitName string

// ResourceLimitName constants
const (
	ResourceLimitSubgroupsPerParent                             ResourceLimitName = "ResourceLimitSubgroupsPerParent"
	ResourceLimitGroupTreeDepth                                 ResourceLimitName = "ResourceLimitGroupTreeDepth"
	ResourceLimitWorkspacesPerGroup                             ResourceLimitName = "ResourceLimitWorkspacesPerGroup"
	ResourceLimitServiceAccountsPerGroup                        ResourceLimitName = "ResourceLimitServiceAccountsPerGroup"
	ResourceLimitRunnerAgentsPerGroup                           ResourceLimitName = "ResourceLimitRunnerAgentsPerGroup"
	ResourceLimitVariablesPerNamespace                          ResourceLimitName = "ResourceLimitVariablesPerNamespace"
	ResourceLimitGPGKeysPerGroup                                ResourceLimitName = "ResourceLimitGPGKeysPerGroup"
	ResourceLimitManagedIdentitiesPerGroup                      ResourceLimitName = "ResourceLimitManagedIdentitiesPerGroup"
	ResourceLimitManagedIdentityAliasesPerManagedIdentity       ResourceLimitName = "ResourceLimitManagedIdentityAliasesPerManagedIdentity"
	ResourceLimitAssignedManagedIdentitiesPerWorkspace          ResourceLimitName = "ResourceLimitAssignedManagedIdentitiesPerWorkspace"
	ResourceLimitManagedIdentityAccessRulesPerManagedIdentity   ResourceLimitName = "ResourceLimitManagedIdentityAccessRulesPerManagedIdentity"
	ResourceLimitTerraformModulesPerGroup                       ResourceLimitName = "ResourceLimitTerraformModulesPerGroup"
	ResourceLimitVersionsPerTerraformModulePerTimePeriod        ResourceLimitName = "ResourceLimitVersionsPerTerraformModulePerTimePeriod"
	ResourceLimitAttestationsPerTerraformModulePerTimePeriod    ResourceLimitName = "ResourceLimitAttestationsPerTerraformModulePerTimePeriod"
	ResourceLimitTerraformProvidersPerGroup                     ResourceLimitName = "ResourceLimitTerraformProvidersPerGroup"
	ResourceLimitVersionsPerTerraformProviderPerTimePeriod      ResourceLimitName = "ResourceLimitVersionsPerTerraformProviderPerTimePeriod"
	ResourceLimitPlatformsPerTerraformProviderVersion           ResourceLimitName = "ResourceLimitPlatformsPerTerraformProviderVersion"
	ResourceLimitVCSProvidersPerGroup                           ResourceLimitName = "ResourceLimitVCSProvidersPerGroup"
	ResourceLimitTerraformProviderVersionMirrorsPerGroup        ResourceLimitName = "ResourceLimitTerraformProviderVersionMirrorsPerGroup"
	ResourceLimitRunnerSessionsPerRunner                        ResourceLimitName = "ResourceLimitRunnerSessionsPerRunner"
	ResourceLimitRunsPerWorkspacePerTimePeriod                  ResourceLimitName = "ResourceLimitRunsPerWorkspacePerTimePeriod"
	ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod ResourceLimitName = "ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod"
	ResourceLimitStateVersionsPerWorkspacePerTimePeriod         ResourceLimitName = "ResourceLimitStateVersionsPerWorkspacePerTimePeriod"
	ResourceLimitFederatedRegistriesPerGroup                    ResourceLimitName = "ResourceLimitFederatedRegistriesPerGroup"
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
		return errors.New("invalid resource limit name: %s", name, errors.WithErrorCode(errors.EInvalid))
	}

	if int(toCheck) > limit.Value {
		return errors.New("for limit %s: value %d exceeds limit of %d", name, toCheck, limit.Value, errors.WithErrorCode(errors.EInvalid))
	}

	// A valid limit value was found, and there is no violation.
	return nil
}
