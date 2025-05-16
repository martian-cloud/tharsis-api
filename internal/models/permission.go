package models

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// All possible Permissions.
var (
	ViewGPGKeyPermission                    = Permission{ResourceType: types.GPGKeyModelType.Name(), Action: ViewAction}
	CreateGPGKeyPermission                  = Permission{ResourceType: types.GPGKeyModelType.Name(), Action: CreateAction}
	DeleteGPGKeyPermission                  = Permission{ResourceType: types.GPGKeyModelType.Name(), Action: DeleteAction}
	ViewGroupPermission                     = Permission{ResourceType: types.GroupModelType.Name(), Action: ViewAction}
	CreateGroupPermission                   = Permission{ResourceType: types.GroupModelType.Name(), Action: CreateAction}
	UpdateGroupPermission                   = Permission{ResourceType: types.GroupModelType.Name(), Action: UpdateAction}
	DeleteGroupPermission                   = Permission{ResourceType: types.GroupModelType.Name(), Action: DeleteAction}
	ViewNamespaceMembershipPermission       = Permission{ResourceType: types.NamespaceMembershipModelType.Name(), Action: ViewAction}
	CreateNamespaceMembershipPermission     = Permission{ResourceType: types.NamespaceMembershipModelType.Name(), Action: CreateAction}
	UpdateNamespaceMembershipPermission     = Permission{ResourceType: types.NamespaceMembershipModelType.Name(), Action: UpdateAction}
	DeleteNamespaceMembershipPermission     = Permission{ResourceType: types.NamespaceMembershipModelType.Name(), Action: DeleteAction}
	ViewWorkspacePermission                 = Permission{ResourceType: types.WorkspaceModelType.Name(), Action: ViewAction}
	CreateWorkspacePermission               = Permission{ResourceType: types.WorkspaceModelType.Name(), Action: CreateAction}
	UpdateWorkspacePermission               = Permission{ResourceType: types.WorkspaceModelType.Name(), Action: UpdateAction}
	DeleteWorkspacePermission               = Permission{ResourceType: types.WorkspaceModelType.Name(), Action: DeleteAction}
	CreateTeamPermission                    = Permission{ResourceType: types.TeamModelType.Name(), Action: CreateAction}
	UpdateTeamPermission                    = Permission{ResourceType: types.TeamModelType.Name(), Action: UpdateAction}
	DeleteTeamPermission                    = Permission{ResourceType: types.TeamModelType.Name(), Action: DeleteAction}
	ViewRunPermission                       = Permission{ResourceType: types.RunModelType.Name(), Action: ViewAction}
	CreateRunPermission                     = Permission{ResourceType: types.RunModelType.Name(), Action: CreateAction}
	ViewJobPermission                       = Permission{ResourceType: types.JobModelType.Name(), Action: ViewAction}
	ClaimJobPermission                      = Permission{ResourceType: types.JobModelType.Name(), Action: ClaimAction}    // Specifically for claiming jobs.
	UpdateJobPermission                     = Permission{ResourceType: types.JobModelType.Name(), Action: UpdateAction}   // Write job perm.
	UpdatePlanPermission                    = Permission{ResourceType: types.PlanModelType.Name(), Action: UpdateAction}  // Write plan perm.
	UpdateApplyPermission                   = Permission{ResourceType: types.ApplyModelType.Name(), Action: UpdateAction} // Write apply perm.
	ViewRunnerPermission                    = Permission{ResourceType: types.RunnerModelType.Name(), Action: ViewAction}
	CreateRunnerPermission                  = Permission{ResourceType: types.RunnerModelType.Name(), Action: CreateAction}
	UpdateRunnerPermission                  = Permission{ResourceType: types.RunnerModelType.Name(), Action: UpdateAction}
	DeleteRunnerPermission                  = Permission{ResourceType: types.RunnerModelType.Name(), Action: DeleteAction}
	CreateRunnerSessionPermission           = Permission{ResourceType: types.RunnerSessionModelType.Name(), Action: CreateAction}
	UpdateRunnerSessionPermission           = Permission{ResourceType: types.RunnerSessionModelType.Name(), Action: UpdateAction}
	CreateUserPermission                    = Permission{ResourceType: types.UserModelType.Name(), Action: CreateAction}
	UpdateUserPermission                    = Permission{ResourceType: types.UserModelType.Name(), Action: UpdateAction}
	DeleteUserPermission                    = Permission{ResourceType: types.UserModelType.Name(), Action: DeleteAction}
	ViewVariableValuePermission             = Permission{ResourceType: types.VariableModelType.Name(), Action: ViewValueAction} // Viewing variable values.
	ViewVariablePermission                  = Permission{ResourceType: types.VariableModelType.Name(), Action: ViewAction}
	CreateVariablePermission                = Permission{ResourceType: types.VariableModelType.Name(), Action: CreateAction}
	UpdateVariablePermission                = Permission{ResourceType: types.VariableModelType.Name(), Action: UpdateAction}
	DeleteVariablePermission                = Permission{ResourceType: types.VariableModelType.Name(), Action: DeleteAction}
	ViewTerraformProviderPermission         = Permission{ResourceType: types.TerraformProviderModelType.Name(), Action: ViewAction}
	CreateTerraformProviderPermission       = Permission{ResourceType: types.TerraformProviderModelType.Name(), Action: CreateAction}
	UpdateTerraformProviderPermission       = Permission{ResourceType: types.TerraformProviderModelType.Name(), Action: UpdateAction}
	DeleteTerraformProviderPermission       = Permission{ResourceType: types.TerraformProviderModelType.Name(), Action: DeleteAction}
	ViewTerraformModulePermission           = Permission{ResourceType: types.TerraformModuleModelType.Name(), Action: ViewAction}
	CreateTerraformModulePermission         = Permission{ResourceType: types.TerraformModuleModelType.Name(), Action: CreateAction}
	UpdateTerraformModulePermission         = Permission{ResourceType: types.TerraformModuleModelType.Name(), Action: UpdateAction}
	DeleteTerraformModulePermission         = Permission{ResourceType: types.TerraformModuleModelType.Name(), Action: DeleteAction}
	ViewStateVersionPermission              = Permission{ResourceType: types.StateVersionModelType.Name(), Action: ViewAction}
	ViewStateVersionDataPermission          = Permission{ResourceType: types.StateVersionModelType.Name(), Action: ViewValueAction}
	CreateStateVersionPermission            = Permission{ResourceType: types.StateVersionModelType.Name(), Action: CreateAction}
	ViewConfigurationVersionPermission      = Permission{ResourceType: types.ConfigurationVersionModelType.Name(), Action: ViewAction}
	CreateConfigurationVersionPermission    = Permission{ResourceType: types.ConfigurationVersionModelType.Name(), Action: CreateAction}
	UpdateConfigurationVersionPermission    = Permission{ResourceType: types.ConfigurationVersionModelType.Name(), Action: UpdateAction}
	ViewServiceAccountPermission            = Permission{ResourceType: types.ServiceAccountModelType.Name(), Action: ViewAction}
	CreateServiceAccountPermission          = Permission{ResourceType: types.ServiceAccountModelType.Name(), Action: CreateAction}
	UpdateServiceAccountPermission          = Permission{ResourceType: types.ServiceAccountModelType.Name(), Action: UpdateAction}
	DeleteServiceAccountPermission          = Permission{ResourceType: types.ServiceAccountModelType.Name(), Action: DeleteAction}
	ViewManagedIdentityPermission           = Permission{ResourceType: types.ManagedIdentityModelType.Name(), Action: ViewAction}
	CreateManagedIdentityPermission         = Permission{ResourceType: types.ManagedIdentityModelType.Name(), Action: CreateAction}
	UpdateManagedIdentityPermission         = Permission{ResourceType: types.ManagedIdentityModelType.Name(), Action: UpdateAction}
	DeleteManagedIdentityPermission         = Permission{ResourceType: types.ManagedIdentityModelType.Name(), Action: DeleteAction}
	ViewVCSProviderPermission               = Permission{ResourceType: types.VCSProviderModelType.Name(), Action: ViewAction}
	CreateVCSProviderPermission             = Permission{ResourceType: types.VCSProviderModelType.Name(), Action: CreateAction}
	UpdateVCSProviderPermission             = Permission{ResourceType: types.VCSProviderModelType.Name(), Action: UpdateAction}
	DeleteVCSProviderPermission             = Permission{ResourceType: types.VCSProviderModelType.Name(), Action: DeleteAction}
	ViewTerraformProviderMirrorPermission   = Permission{ResourceType: types.TerraformProviderMirrorModelType.Name(), Action: ViewAction}
	CreateTerraformProviderMirrorPermission = Permission{ResourceType: types.TerraformProviderMirrorModelType.Name(), Action: CreateAction}
	DeleteTerraformProviderMirrorPermission = Permission{ResourceType: types.TerraformProviderMirrorModelType.Name(), Action: DeleteAction}
	ViewFederatedRegistryPermission         = Permission{ResourceType: types.FederatedRegistryModelType.Name(), Action: ViewAction}
	CreateFederatedRegistryPermission       = Permission{ResourceType: types.FederatedRegistryModelType.Name(), Action: CreateAction}
	UpdateFederatedRegistryPermission       = Permission{ResourceType: types.FederatedRegistryModelType.Name(), Action: UpdateAction}
	DeleteFederatedRegistryPermission       = Permission{ResourceType: types.FederatedRegistryModelType.Name(), Action: DeleteAction}
	CreateFederatedRegistryTokenPermission  = Permission{ResourceType: types.FederatedRegistryModelType.Name(), Action: CreateAction}
)

// assignablePermissions contains all the permissions that
// may be assigned to Tharsis roles.
var assignablePermissions = map[Permission]struct{}{
	ViewGPGKeyPermission:                    {},
	CreateGPGKeyPermission:                  {},
	DeleteGPGKeyPermission:                  {},
	ViewGroupPermission:                     {},
	CreateGroupPermission:                   {},
	UpdateGroupPermission:                   {},
	DeleteGroupPermission:                   {},
	ViewNamespaceMembershipPermission:       {},
	CreateNamespaceMembershipPermission:     {},
	UpdateNamespaceMembershipPermission:     {},
	DeleteNamespaceMembershipPermission:     {},
	ViewWorkspacePermission:                 {},
	CreateWorkspacePermission:               {},
	UpdateWorkspacePermission:               {},
	DeleteWorkspacePermission:               {},
	ViewRunPermission:                       {},
	CreateRunPermission:                     {},
	ViewJobPermission:                       {},
	ViewRunnerPermission:                    {},
	CreateRunnerPermission:                  {},
	UpdateRunnerPermission:                  {},
	DeleteRunnerPermission:                  {},
	CreateUserPermission:                    {},
	UpdateUserPermission:                    {},
	DeleteUserPermission:                    {},
	ViewVariableValuePermission:             {},
	ViewVariablePermission:                  {},
	CreateVariablePermission:                {},
	UpdateVariablePermission:                {},
	DeleteVariablePermission:                {},
	ViewTerraformProviderPermission:         {},
	CreateTerraformProviderPermission:       {},
	UpdateTerraformProviderPermission:       {},
	DeleteTerraformProviderPermission:       {},
	ViewTerraformModulePermission:           {},
	CreateTerraformModulePermission:         {},
	UpdateTerraformModulePermission:         {},
	DeleteTerraformModulePermission:         {},
	ViewStateVersionPermission:              {},
	ViewStateVersionDataPermission:          {},
	CreateStateVersionPermission:            {},
	ViewConfigurationVersionPermission:      {},
	CreateConfigurationVersionPermission:    {},
	UpdateConfigurationVersionPermission:    {},
	ViewServiceAccountPermission:            {},
	CreateServiceAccountPermission:          {},
	UpdateServiceAccountPermission:          {},
	DeleteServiceAccountPermission:          {},
	ViewManagedIdentityPermission:           {},
	CreateManagedIdentityPermission:         {},
	UpdateManagedIdentityPermission:         {},
	DeleteManagedIdentityPermission:         {},
	ViewVCSProviderPermission:               {},
	CreateVCSProviderPermission:             {},
	UpdateVCSProviderPermission:             {},
	DeleteVCSProviderPermission:             {},
	ViewTerraformProviderMirrorPermission:   {},
	CreateTerraformProviderMirrorPermission: {},
	DeleteTerraformProviderMirrorPermission: {},
	ViewFederatedRegistryPermission:         {},
	CreateFederatedRegistryPermission:       {},
	UpdateFederatedRegistryPermission:       {},
	DeleteFederatedRegistryPermission:       {},
}

// Action is an enum representing a CRUD action.
type Action string

// Action constants.
const (
	ViewAction      Action = "view"
	ViewValueAction Action = "view_value"
	CreateAction    Action = "create"
	UpdateAction    Action = "update"
	DeleteAction    Action = "delete"
	ClaimAction     Action = "claim"
)

// HasViewerAccess returns true if Action is viewer access or greater.
func (p Action) HasViewerAccess() bool {
	switch p {
	case ViewAction,
		CreateAction,
		UpdateAction,
		DeleteAction,
		ViewValueAction:
		return true
	}

	return false
}

// Permission represents a level of access a subject has to a Tharsis resource.
type Permission struct {
	ResourceType string `json:"resourceType"`
	Action       Action `json:"action"`
}

// String returns the Permission as <resource_type:action> string.
func (p *Permission) String() string {
	return fmt.Sprintf("%s:%s", p.ResourceType, p.Action)
}

// GTE returns true if permission available is >= wanted permission.
func (p *Permission) GTE(want *Permission) bool {
	if p.String() == want.String() {
		// Both permissions are equal.
		return true
	}

	if p.ResourceType != want.ResourceType {
		// This permission is for a different resource type.
		return false
	}

	if want.Action == ViewAction {
		// Return true if permission grants ViewerAccess.
		return p.Action.HasViewerAccess()
	}

	// Permission action doesn't exist for resource type.
	return false
}

// IsAssignable returns true if permission is assignable to a role.
func (p *Permission) IsAssignable() bool {
	_, ok := assignablePermissions[*p]
	return ok
}

// GetAssignablePermissions returns a list of assignable permissions.
func GetAssignablePermissions() []string {
	assignable := []string{}
	for perm := range assignablePermissions {
		assignable = append(assignable, perm.String())
	}

	sort.Strings(assignable)
	return assignable
}

// ParsePermissions parses and normalizes a slice of
// permission strings and extracts a Permission that adheres
// to the format resource_type:action.
func ParsePermissions(perms []string) ([]Permission, error) {
	var parsedPerms []Permission
	for _, p := range perms {
		if strings.TrimSpace(p) == "" {
			// Skip any empty perms.
			continue
		}

		// Make sure there are exactly two parts.
		pair := strings.Split(p, ":")
		if len(pair) != 2 {
			return nil, errors.New("permission must be in format 'resource_type:action', got %s", p, errors.WithErrorCode(errors.EInvalid))
		}

		perm := Permission{
			ResourceType: strings.TrimSpace(pair[0]),
			Action:       Action(strings.TrimSpace(pair[1])),
		}

		if !perm.IsAssignable() {
			return nil, errors.New("permission is not assignable: %s", p, errors.WithErrorCode(errors.EInvalid))
		}

		parsedPerms = append(parsedPerms, perm)
	}

	return parsedPerms, nil
}
