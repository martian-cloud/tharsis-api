// Package services encapsulates the core business logic for Tharsis
package services

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/announcement"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/federatedregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/resourcelimit"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/role"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/version"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// modelFetcherFunc defines a function type that retrieves a model by its identifier
// Used for both GID and TRN-based lookups across different resource types
type modelFetcherFunc func(ctx context.Context, value string) (models.Model, error)

// Catalog provides a unified interface for accessing all Tharsis resources
// by their Global ID (GID) or Tharsis Resource Name (TRN)
type Catalog struct {
	ActivityEventService             activityevent.Service
	AnnouncementService              announcement.Service
	CLIService                       cli.Service
	FederatedRegistryService         federatedregistry.Service
	GPGKeyService                    gpgkey.Service
	GroupService                     group.Service
	JobService                       job.Service
	MaintenanceModeService           maintenance.Service
	ManagedIdentityService           managedidentity.Service
	NamespaceMembershipService       namespacemembership.Service
	ResourceLimitService             resourcelimit.Service
	RoleService                      role.Service
	RunnerService                    runner.Service
	RunService                       run.Service
	SCIMService                      scim.Service
	ServiceAccountService            serviceaccount.Service
	TeamService                      team.Service
	TerraformModuleRegistryService   moduleregistry.Service
	TerraformProviderMirrorService   providermirror.Service
	TerraformProviderRegistryService providerregistry.Service
	UserService                      user.Service
	VariableService                  variable.Service
	VCSService                       vcs.Service
	VersionService                   version.Service
	WorkspaceService                 workspace.Service
	gidFetchers                      map[string]modelFetcherFunc
	trnFetchers                      map[string]modelFetcherFunc
}

// FetchModel retrieves a resource model by its Global ID (GID) or Tharsis Resource Name (TRN)
// It automatically detects the identifier type and uses the appropriate fetcher
func (c *Catalog) FetchModel(ctx context.Context, value string) (models.Model, error) {
	if types.IsTRN(value) {
		modelName := types.GetModelNameFromTRN(value)
		fetchByTRN, ok := c.getModelFetcherByModelName(modelName)
		if !ok {
			return nil, errors.New("unsupported resource type: TRN with model type '%s' cannot be resolved", modelName)
		}

		return fetchByTRN(ctx, value)
	}

	parsedGID, err := gid.ParseGlobalID(value)
	if err != nil {
		return nil, errors.Wrap(err, "invalid identifier format: value is neither a valid TRN nor GID")
	}

	// If the value is not a TRN, fetch it using the appropriate method
	fetchByID, ok := c.getModelFetcherByGIDCode(parsedGID.Code)
	if !ok {
		return nil, errors.New("unsupported resource type: GID with code '%s' cannot be resolved", parsedGID.Code)
	}

	return fetchByID(ctx, parsedGID.ID)
}

// FetchModelID extracts a model's unique identifier from either a GID or TRN
// Returns the raw ID without fetching the entire model when possible
func (c *Catalog) FetchModelID(ctx context.Context, value string) (string, error) {
	// If the value is a TRN, fetch it using the appropriate method
	if types.IsTRN(value) {
		modelName := types.GetModelNameFromTRN(value)
		fetchByTRN, ok := c.getModelFetcherByModelName(modelName)
		if !ok {
			return "", errors.New("unsupported resource type: TRN with model type '%s' has no registered handler", modelName)
		}

		model, err := fetchByTRN(ctx, value)
		if err != nil {
			return "", errors.Wrap(err, "failed to retrieve resource by TRN")
		}

		return model.GetID(), nil
	}

	parsedGID, err := gid.ParseGlobalID(value)
	if err != nil {
		return "", errors.Wrap(err, "invalid identifier format: failed to parse as GID")
	}

	return parsedGID.ID, nil
}

// Init registers all service-specific model fetchers for both GID and TRN identifiers
// Must be called after creating a Catalog instance to enable model resolution
func (c *Catalog) Init() {
	// Add model fetchers
	// Methods are sorted alphabetically by service name

	// Announcement Service
	c.addModelFetchers(types.AnnouncementModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.AnnouncementService.GetAnnouncementByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.AnnouncementService.GetAnnouncementByTRN(ctx, value)
		},
	)

	// Federated Registry Service
	c.addModelFetchers(types.FederatedRegistryModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.FederatedRegistryService.GetFederatedRegistryByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.FederatedRegistryService.GetFederatedRegistryByTRN(ctx, value)
		},
	)

	// GPG Key Service
	c.addModelFetchers(types.GPGKeyModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.GPGKeyService.GetGPGKeyByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.GPGKeyService.GetGPGKeyByTRN(ctx, value)
		},
	)

	// Group Service
	c.addModelFetchers(types.GroupModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.GroupService.GetGroupByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.GroupService.GetGroupByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.NamespaceMembershipModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.NamespaceMembershipService.GetNamespaceMembershipByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.NamespaceMembershipService.GetNamespaceMembershipByTRN(ctx, value)
		},
	)

	// Job Service
	c.addModelFetchers(types.JobModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.JobService.GetJobByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.JobService.GetJobByTRN(ctx, value)
		},
	)

	// Managed Identity Service
	c.addModelFetchers(types.ManagedIdentityModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ManagedIdentityService.GetManagedIdentityByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ManagedIdentityService.GetManagedIdentityByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.ManagedIdentityAccessRuleModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ManagedIdentityService.GetManagedIdentityAccessRuleByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ManagedIdentityService.GetManagedIdentityAccessRuleByTRN(ctx, value)
		},
	)

	// Role Service
	c.addModelFetchers(types.RoleModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RoleService.GetRoleByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RoleService.GetRoleByTRN(ctx, value)
		},
	)

	// Run Service
	c.addModelFetchers(types.ApplyModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetApplyByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetApplyByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.PlanModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetPlanByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetPlanByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.RunModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetRunByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunService.GetRunByTRN(ctx, value)
		},
	)

	// Runner Service
	c.addModelFetchers(types.RunnerModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunnerService.GetRunnerByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunnerService.GetRunnerByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.RunnerSessionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunnerService.GetRunnerSessionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.RunnerService.GetRunnerSessionByTRN(ctx, value)
		},
	)

	// Service Account Service
	c.addModelFetchers(types.ServiceAccountModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ServiceAccountService.GetServiceAccountByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.ServiceAccountService.GetServiceAccountByTRN(ctx, value)
		},
	)

	// Team Service
	c.addModelFetchers(types.TeamModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TeamService.GetTeamByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TeamService.GetTeamByTRN(ctx, value)
		},
	)

	// Terraform Module Registry Service
	c.addModelFetchers(types.TerraformModuleModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.TerraformModuleVersionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleVersionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleVersionByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.TerraformModuleAttestationModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleAttestationByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformModuleRegistryService.GetModuleAttestationByTRN(ctx, value)
		},
	)

	// Terraform Provider Mirror Service
	c.addModelFetchers(types.TerraformProviderVersionMirrorModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderMirrorService.GetProviderVersionMirrorByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderMirrorService.GetProviderVersionMirrorByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.TerraformProviderPlatformMirrorModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderMirrorService.GetProviderPlatformMirrorByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderMirrorService.GetProviderPlatformMirrorByTRN(ctx, value)
		},
	)

	// Terraform Provider Registry Service
	c.addModelFetchers(types.TerraformProviderModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.TerraformProviderVersionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderVersionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderVersionByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.TerraformProviderPlatformModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderPlatformByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.TerraformProviderRegistryService.GetProviderPlatformByTRN(ctx, value)
		},
	)

	// User Service
	c.addModelFetchers(types.UserModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetUserByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetUserByTRN(ctx, value)
		},
	)

	// User Session - UserSession supports TRN resolution
	c.addModelFetchers(types.UserSessionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetUserSessionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetUserSessionByTRN(ctx, value)
		},
	)

	// Namespace Favorite
	c.addModelFetchers(types.NamespaceFavoriteModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetNamespaceFavoriteByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.UserService.GetNamespaceFavoriteByTRN(ctx, value)
		},
	)

	// Variable Service
	c.addModelFetchers(types.VariableModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VariableService.GetVariableByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VariableService.GetVariableByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.VariableVersionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VariableService.GetVariableVersionByID(ctx, value, false) // No sensitive vars
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VariableService.GetVariableVersionByTRN(ctx, value, false)
		},
	)

	// VCS Provider Service
	c.addModelFetchers(types.VCSProviderModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetVCSProviderByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetVCSProviderByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.WorkspaceVCSProviderLinkModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetWorkspaceVCSProviderLinkByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetWorkspaceVCSProviderLinkByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.VCSEventModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetVCSEventByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.VCSService.GetVCSEventByTRN(ctx, value)
		},
	)

	// Workspace Service
	c.addModelFetchers(types.ConfigurationVersionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetConfigurationVersionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetConfigurationVersionByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.StateVersionModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetStateVersionByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetStateVersionByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.StateVersionOutputModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetStateVersionOutputByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetStateVersionOutputByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.WorkspaceModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetWorkspaceByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetWorkspaceByTRN(ctx, value)
		},
	)

	c.addModelFetchers(types.WorkspaceAssessmentModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetWorkspaceAssessmentByID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return c.WorkspaceService.GetWorkspaceAssessmentByTRN(ctx, value)
		},
	)
}

// addModelFetchers registers a pair of fetcher functions for a specific model type
// Maps the GID fetcher to the model's GID code and TRN fetcher to the model's name
func (c *Catalog) addModelFetchers(modelType types.ModelType, fetchByGID, fetchByTRN modelFetcherFunc) {
	if c.gidFetchers == nil {
		c.gidFetchers = make(map[string]modelFetcherFunc)
	}

	if c.trnFetchers == nil {
		c.trnFetchers = make(map[string]modelFetcherFunc)
	}

	c.gidFetchers[modelType.GIDCode()] = fetchByGID
	c.trnFetchers[modelType.Name()] = fetchByTRN
}

// getModelFetcherByModelName retrieves the TRN-based fetcher function for a given model name
// Returns the fetcher function and a boolean indicating if the model type is supported
func (c *Catalog) getModelFetcherByModelName(modelName string) (modelFetcherFunc, bool) {
	fetcher, ok := c.trnFetchers[modelName]
	return fetcher, ok
}

// getModelFetcherByGIDCode retrieves the GID-based fetcher function for a given GID code
// Returns the fetcher function and a boolean indicating if the GID code is supported
func (c *Catalog) getModelFetcherByGIDCode(gidCode string) (modelFetcherFunc, bool) {
	fetcher, ok := c.gidFetchers[gidCode]
	return fetcher, ok
}
