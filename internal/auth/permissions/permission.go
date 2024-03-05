// Package permissions contains the permission sets
// and other related functionalities that dictate
// the level of access a subject has to a Tharsis
// resource.
package permissions

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// All possible Permissions.
var (
	ViewGPGKeyPermission                    = Permission{ResourceType: GPGKeyResourceType, Action: ViewAction}
	CreateGPGKeyPermission                  = Permission{ResourceType: GPGKeyResourceType, Action: CreateAction}
	DeleteGPGKeyPermission                  = Permission{ResourceType: GPGKeyResourceType, Action: DeleteAction}
	ViewGroupPermission                     = Permission{ResourceType: GroupResourceType, Action: ViewAction}
	CreateGroupPermission                   = Permission{ResourceType: GroupResourceType, Action: CreateAction}
	UpdateGroupPermission                   = Permission{ResourceType: GroupResourceType, Action: UpdateAction}
	DeleteGroupPermission                   = Permission{ResourceType: GroupResourceType, Action: DeleteAction}
	ViewNamespaceMembershipPermission       = Permission{ResourceType: NamespaceMembershipResourceType, Action: ViewAction}
	CreateNamespaceMembershipPermission     = Permission{ResourceType: NamespaceMembershipResourceType, Action: CreateAction}
	UpdateNamespaceMembershipPermission     = Permission{ResourceType: NamespaceMembershipResourceType, Action: UpdateAction}
	DeleteNamespaceMembershipPermission     = Permission{ResourceType: NamespaceMembershipResourceType, Action: DeleteAction}
	ViewWorkspacePermission                 = Permission{ResourceType: WorkspaceResourceType, Action: ViewAction}
	CreateWorkspacePermission               = Permission{ResourceType: WorkspaceResourceType, Action: CreateAction}
	UpdateWorkspacePermission               = Permission{ResourceType: WorkspaceResourceType, Action: UpdateAction}
	DeleteWorkspacePermission               = Permission{ResourceType: WorkspaceResourceType, Action: DeleteAction}
	CreateTeamPermission                    = Permission{ResourceType: TeamResourceType, Action: CreateAction}
	UpdateTeamPermission                    = Permission{ResourceType: TeamResourceType, Action: UpdateAction}
	DeleteTeamPermission                    = Permission{ResourceType: TeamResourceType, Action: DeleteAction}
	ViewRunPermission                       = Permission{ResourceType: RunResourceType, Action: ViewAction}
	CreateRunPermission                     = Permission{ResourceType: RunResourceType, Action: CreateAction}
	ViewJobPermission                       = Permission{ResourceType: JobResourceType, Action: ViewAction}
	ClaimJobPermission                      = Permission{ResourceType: JobResourceType, Action: ClaimAction}    // Specifically for claiming jobs.
	UpdateJobPermission                     = Permission{ResourceType: JobResourceType, Action: UpdateAction}   // Write job perm.
	UpdatePlanPermission                    = Permission{ResourceType: PlanResourceType, Action: UpdateAction}  // Write plan perm.
	UpdateApplyPermission                   = Permission{ResourceType: ApplyResourceType, Action: UpdateAction} // Write apply perm.
	ViewRunnerPermission                    = Permission{ResourceType: RunnerResourceType, Action: ViewAction}
	CreateRunnerPermission                  = Permission{ResourceType: RunnerResourceType, Action: CreateAction}
	UpdateRunnerPermission                  = Permission{ResourceType: RunnerResourceType, Action: UpdateAction}
	DeleteRunnerPermission                  = Permission{ResourceType: RunnerResourceType, Action: DeleteAction}
	CreateRunnerSessionPermission           = Permission{ResourceType: RunnerSessionResourceType, Action: CreateAction}
	UpdateRunnerSessionPermission           = Permission{ResourceType: RunnerSessionResourceType, Action: UpdateAction}
	CreateUserPermission                    = Permission{ResourceType: UserResourceType, Action: CreateAction}
	UpdateUserPermission                    = Permission{ResourceType: UserResourceType, Action: UpdateAction}
	DeleteUserPermission                    = Permission{ResourceType: UserResourceType, Action: DeleteAction}
	ViewVariableValuePermission             = Permission{ResourceType: VariableResourceType, Action: ViewValueAction} // Viewing variable values.
	ViewVariablePermission                  = Permission{ResourceType: VariableResourceType, Action: ViewAction}
	CreateVariablePermission                = Permission{ResourceType: VariableResourceType, Action: CreateAction}
	UpdateVariablePermission                = Permission{ResourceType: VariableResourceType, Action: UpdateAction}
	DeleteVariablePermission                = Permission{ResourceType: VariableResourceType, Action: DeleteAction}
	ViewTerraformProviderPermission         = Permission{ResourceType: TerraformProviderResourceType, Action: ViewAction}
	CreateTerraformProviderPermission       = Permission{ResourceType: TerraformProviderResourceType, Action: CreateAction}
	UpdateTerraformProviderPermission       = Permission{ResourceType: TerraformProviderResourceType, Action: UpdateAction}
	DeleteTerraformProviderPermission       = Permission{ResourceType: TerraformProviderResourceType, Action: DeleteAction}
	ViewTerraformModulePermission           = Permission{ResourceType: TerraformModuleResourceType, Action: ViewAction}
	CreateTerraformModulePermission         = Permission{ResourceType: TerraformModuleResourceType, Action: CreateAction}
	UpdateTerraformModulePermission         = Permission{ResourceType: TerraformModuleResourceType, Action: UpdateAction}
	DeleteTerraformModulePermission         = Permission{ResourceType: TerraformModuleResourceType, Action: DeleteAction}
	ViewStateVersionPermission              = Permission{ResourceType: StateVersionResourceType, Action: ViewAction}
	ViewStateVersionDataPermission          = Permission{ResourceType: StateVersionResourceType, Action: ViewValueAction}
	CreateStateVersionPermission            = Permission{ResourceType: StateVersionResourceType, Action: CreateAction}
	ViewConfigurationVersionPermission      = Permission{ResourceType: ConfigurationVersionResourceType, Action: ViewAction}
	CreateConfigurationVersionPermission    = Permission{ResourceType: ConfigurationVersionResourceType, Action: CreateAction}
	UpdateConfigurationVersionPermission    = Permission{ResourceType: ConfigurationVersionResourceType, Action: UpdateAction}
	ViewServiceAccountPermission            = Permission{ResourceType: ServiceAccountResourceType, Action: ViewAction}
	CreateServiceAccountPermission          = Permission{ResourceType: ServiceAccountResourceType, Action: CreateAction}
	UpdateServiceAccountPermission          = Permission{ResourceType: ServiceAccountResourceType, Action: UpdateAction}
	DeleteServiceAccountPermission          = Permission{ResourceType: ServiceAccountResourceType, Action: DeleteAction}
	ViewManagedIdentityPermission           = Permission{ResourceType: ManagedIdentityResourceType, Action: ViewAction}
	CreateManagedIdentityPermission         = Permission{ResourceType: ManagedIdentityResourceType, Action: CreateAction}
	UpdateManagedIdentityPermission         = Permission{ResourceType: ManagedIdentityResourceType, Action: UpdateAction}
	DeleteManagedIdentityPermission         = Permission{ResourceType: ManagedIdentityResourceType, Action: DeleteAction}
	ViewVCSProviderPermission               = Permission{ResourceType: VCSProviderResourceType, Action: ViewAction}
	CreateVCSProviderPermission             = Permission{ResourceType: VCSProviderResourceType, Action: CreateAction}
	UpdateVCSProviderPermission             = Permission{ResourceType: VCSProviderResourceType, Action: UpdateAction}
	DeleteVCSProviderPermission             = Permission{ResourceType: VCSProviderResourceType, Action: DeleteAction}
	ViewTerraformProviderMirrorPermission   = Permission{ResourceType: TerraformProviderMirrorResourceType, Action: ViewAction}
	CreateTerraformProviderMirrorPermission = Permission{ResourceType: TerraformProviderMirrorResourceType, Action: CreateAction}
	DeleteTerraformProviderMirrorPermission = Permission{ResourceType: TerraformProviderMirrorResourceType, Action: DeleteAction}
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
	ResourceType ResourceType `json:"resourceType"`
	Action       Action       `json:"action"`
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
			return nil, errors.New("invalid permission: %s", p, errors.WithErrorCode(errors.EInvalid))
		}

		perm := Permission{
			ResourceType: ResourceType(strings.TrimSpace(pair[0])),
			Action:       Action(strings.TrimSpace(pair[1])),
		}

		parsedPerms = append(parsedPerms, perm)
	}

	return parsedPerms, nil
}
