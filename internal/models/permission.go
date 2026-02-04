package models

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var (
	permissionRegistry    = map[string]struct{}{}
	assignablePermissions = map[Permission]struct{}{}
)

// All possible Permissions.
var (
	ViewGPGKeyPermission                    = registerPermission(types.GPGKeyModelType, ViewAction, true)
	CreateGPGKeyPermission                  = registerPermission(types.GPGKeyModelType, CreateAction, true)
	DeleteGPGKeyPermission                  = registerPermission(types.GPGKeyModelType, DeleteAction, true)
	ViewGroupPermission                     = registerPermission(types.GroupModelType, ViewAction, true)
	CreateGroupPermission                   = registerPermission(types.GroupModelType, CreateAction, true)
	UpdateGroupPermission                   = registerPermission(types.GroupModelType, UpdateAction, true)
	DeleteGroupPermission                   = registerPermission(types.GroupModelType, DeleteAction, true)
	ViewNamespaceMembershipPermission       = registerPermission(types.NamespaceMembershipModelType, ViewAction, true)
	CreateNamespaceMembershipPermission     = registerPermission(types.NamespaceMembershipModelType, CreateAction, true)
	UpdateNamespaceMembershipPermission     = registerPermission(types.NamespaceMembershipModelType, UpdateAction, true)
	DeleteNamespaceMembershipPermission     = registerPermission(types.NamespaceMembershipModelType, DeleteAction, true)
	ViewWorkspacePermission                 = registerPermission(types.WorkspaceModelType, ViewAction, true)
	CreateWorkspacePermission               = registerPermission(types.WorkspaceModelType, CreateAction, true)
	UpdateWorkspacePermission               = registerPermission(types.WorkspaceModelType, UpdateAction, true)
	DeleteWorkspacePermission               = registerPermission(types.WorkspaceModelType, DeleteAction, true)
	CreateTeamPermission                    = registerPermission(types.TeamModelType, CreateAction, false)
	UpdateTeamPermission                    = registerPermission(types.TeamModelType, UpdateAction, false)
	DeleteTeamPermission                    = registerPermission(types.TeamModelType, DeleteAction, false)
	ViewRunPermission                       = registerPermission(types.RunModelType, ViewAction, true)
	CreateRunPermission                     = registerPermission(types.RunModelType, CreateAction, true)
	ViewJobPermission                       = registerPermission(types.JobModelType, ViewAction, true)
	ClaimJobPermission                      = registerPermission(types.JobModelType, ClaimAction, false)    // Specifically for claiming jobs.
	UpdateJobPermission                     = registerPermission(types.JobModelType, UpdateAction, false)   // Write job perm.
	UpdatePlanPermission                    = registerPermission(types.PlanModelType, UpdateAction, false)  // Write plan perm.
	UpdateApplyPermission                   = registerPermission(types.ApplyModelType, UpdateAction, false) // Write apply perm.
	ViewRunnerPermission                    = registerPermission(types.RunnerModelType, ViewAction, true)
	CreateRunnerPermission                  = registerPermission(types.RunnerModelType, CreateAction, true)
	UpdateRunnerPermission                  = registerPermission(types.RunnerModelType, UpdateAction, true)
	DeleteRunnerPermission                  = registerPermission(types.RunnerModelType, DeleteAction, true)
	CreateRunnerSessionPermission           = registerPermission(types.RunnerSessionModelType, CreateAction, false)
	UpdateRunnerSessionPermission           = registerPermission(types.RunnerSessionModelType, UpdateAction, false)
	CreateUserPermission                    = registerPermission(types.UserModelType, CreateAction, true)
	UpdateUserPermission                    = registerPermission(types.UserModelType, UpdateAction, true)
	DeleteUserPermission                    = registerPermission(types.UserModelType, DeleteAction, true)
	ViewVariableValuePermission             = registerPermission(types.VariableModelType, ViewValueAction, true) // Viewing variable values.
	ViewVariablePermission                  = registerPermission(types.VariableModelType, ViewAction, true)
	CreateVariablePermission                = registerPermission(types.VariableModelType, CreateAction, true)
	UpdateVariablePermission                = registerPermission(types.VariableModelType, UpdateAction, true)
	DeleteVariablePermission                = registerPermission(types.VariableModelType, DeleteAction, true)
	ViewTerraformProviderPermission         = registerPermission(types.TerraformProviderModelType, ViewAction, true)
	CreateTerraformProviderPermission       = registerPermission(types.TerraformProviderModelType, CreateAction, true)
	UpdateTerraformProviderPermission       = registerPermission(types.TerraformProviderModelType, UpdateAction, true)
	DeleteTerraformProviderPermission       = registerPermission(types.TerraformProviderModelType, DeleteAction, true)
	ViewTerraformModulePermission           = registerPermission(types.TerraformModuleModelType, ViewAction, true)
	CreateTerraformModulePermission         = registerPermission(types.TerraformModuleModelType, CreateAction, true)
	UpdateTerraformModulePermission         = registerPermission(types.TerraformModuleModelType, UpdateAction, true)
	DeleteTerraformModulePermission         = registerPermission(types.TerraformModuleModelType, DeleteAction, true)
	ViewStateVersionPermission              = registerPermission(types.StateVersionModelType, ViewAction, true)
	ViewStateVersionDataPermission          = registerPermission(types.StateVersionModelType, ViewValueAction, true)
	CreateStateVersionPermission            = registerPermission(types.StateVersionModelType, CreateAction, true)
	ViewConfigurationVersionPermission      = registerPermission(types.ConfigurationVersionModelType, ViewAction, true)
	CreateConfigurationVersionPermission    = registerPermission(types.ConfigurationVersionModelType, CreateAction, true)
	UpdateConfigurationVersionPermission    = registerPermission(types.ConfigurationVersionModelType, UpdateAction, true)
	ViewServiceAccountPermission            = registerPermission(types.ServiceAccountModelType, ViewAction, true)
	CreateServiceAccountPermission          = registerPermission(types.ServiceAccountModelType, CreateAction, true)
	UpdateServiceAccountPermission          = registerPermission(types.ServiceAccountModelType, UpdateAction, true)
	DeleteServiceAccountPermission          = registerPermission(types.ServiceAccountModelType, DeleteAction, true)
	ViewManagedIdentityPermission           = registerPermission(types.ManagedIdentityModelType, ViewAction, true)
	CreateManagedIdentityPermission         = registerPermission(types.ManagedIdentityModelType, CreateAction, true)
	UpdateManagedIdentityPermission         = registerPermission(types.ManagedIdentityModelType, UpdateAction, true)
	DeleteManagedIdentityPermission         = registerPermission(types.ManagedIdentityModelType, DeleteAction, true)
	ViewVCSProviderPermission               = registerPermission(types.VCSProviderModelType, ViewAction, true)
	CreateVCSProviderPermission             = registerPermission(types.VCSProviderModelType, CreateAction, true)
	UpdateVCSProviderPermission             = registerPermission(types.VCSProviderModelType, UpdateAction, true)
	DeleteVCSProviderPermission             = registerPermission(types.VCSProviderModelType, DeleteAction, true)
	ViewTerraformProviderMirrorPermission   = registerPermission(types.TerraformProviderMirrorModelType, ViewAction, true)
	CreateTerraformProviderMirrorPermission = registerPermission(types.TerraformProviderMirrorModelType, CreateAction, true)
	DeleteTerraformProviderMirrorPermission = registerPermission(types.TerraformProviderMirrorModelType, DeleteAction, true)
	ViewFederatedRegistryPermission         = registerPermission(types.FederatedRegistryModelType, ViewAction, true)
	CreateFederatedRegistryPermission       = registerPermission(types.FederatedRegistryModelType, CreateAction, true)
	UpdateFederatedRegistryPermission       = registerPermission(types.FederatedRegistryModelType, UpdateAction, true)
	DeleteFederatedRegistryPermission       = registerPermission(types.FederatedRegistryModelType, DeleteAction, true)
	IssueFederatedRegistryTokenPermission   = registerPermission(types.FederatedRegistryModelType, IssueTokenAction, false)
)

// Action is an enum representing a CRUD action.
type Action string

// Action constants.
const (
	ViewAction       Action = "view"
	ViewValueAction  Action = "view_value"
	CreateAction     Action = "create"
	UpdateAction     Action = "update"
	DeleteAction     Action = "delete"
	ClaimAction      Action = "claim"
	IssueTokenAction Action = "issue_token"
)

// HasViewerAccess returns true if Action is viewer access or greater.
func (p Action) HasViewerAccess() bool {
	switch p {
	case ViewAction,
		CreateAction,
		UpdateAction,
		DeleteAction,
		ViewValueAction,
		IssueTokenAction:
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

// registerPermission creates a new Permission and registers it. Panics if a duplicate is detected.
func registerPermission(resourceType types.ModelType, action Action, assignable bool) Permission {
	p := Permission{ResourceType: resourceType.Name(), Action: action}
	key := p.String()

	if _, exists := permissionRegistry[key]; exists {
		panic("duplicate permission: " + key)
	}

	permissionRegistry[key] = struct{}{}

	if assignable {
		assignablePermissions[p] = struct{}{}
	}

	return p
}
