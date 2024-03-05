package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Role defines a subject's ability to access or modify
// resources within Tharsis. It provides a set of permissions
// that dictate which resources can be viewed or modified.
type Role struct {
	Name        string
	Description string
	CreatedBy   string
	permissions []permissions.Permission
	Metadata    ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Role) ResolveMetadata(key string) (string, error) {
	val, err := r.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			val = r.Name
		default:
			return "", err
		}
	}

	return val, nil
}

// SetPermissions sets permissions for a role.
func (r *Role) SetPermissions(perms []permissions.Permission) {
	r.permissions = perms
}

// GetPermissions returns permissions for a role.
func (r *Role) GetPermissions() []permissions.Permission {
	if perms, ok := DefaultRoleID(r.Metadata.ID).Permissions(); ok {
		return perms
	}

	return r.permissions
}

// Validate returns an error if the model is not valid
func (r *Role) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(r.Name); err != nil {
		return err
	}

	// Validate and deduplicate permissions.
	seen := map[string]struct{}{}
	uniquePerms := []permissions.Permission{}
	for _, perm := range r.permissions {
		// Make sure the permission can be assigned.
		if !perm.IsAssignable() {
			return errors.New(
				"Permission '%s' cannot be assigned to a role", perm,
				errors.WithErrorCode(errors.EInvalid),
			)
		}

		if _, ok := seen[perm.String()]; ok {
			continue
		}

		seen[perm.String()] = struct{}{}
		uniquePerms = append(uniquePerms, perm)
	}

	// Assign perms.
	r.permissions = uniquePerms

	// Verify description satisfies constraints
	return verifyValidDescription(r.Description)
}

// DefaultRoleID represents the static UUIDs for default Tharsis roles.
type DefaultRoleID string

// DefaultRoleID constants.
const (
	OwnerRoleID    DefaultRoleID = "623c83ea-23fe-4de6-874a-a99ccf6a76fc"
	DeployerRoleID DefaultRoleID = "8aa7adba-b769-471f-8ebb-3215f33991cb"
	ViewerRoleID   DefaultRoleID = "52da70fd-37b0-4349-bb64-fb4659bcf5f5"
)

// String returns the ID as a string.
func (d DefaultRoleID) String() string {
	return string(d)
}

// IsDefaultRole returns true if ID belongs to a default role.
func (d DefaultRoleID) IsDefaultRole() bool {
	switch d {
	case OwnerRoleID, DeployerRoleID, ViewerRoleID:
		return true
	}

	return false
}

// Permissions returns the Permission set for a default Tharsis role.
func (d DefaultRoleID) Permissions() ([]permissions.Permission, bool) {
	perms, ok := defaultRolePermissions[d]
	return perms, ok
}

// defaultRolePermissions is a map of default role's ID to its Permission set.
var defaultRolePermissions = map[DefaultRoleID][]permissions.Permission{
	// Owner Role
	OwnerRoleID: {
		permissions.ViewGPGKeyPermission,
		permissions.CreateGPGKeyPermission,
		permissions.DeleteGPGKeyPermission,
		permissions.ViewGroupPermission,
		permissions.CreateGroupPermission,
		permissions.UpdateGroupPermission,
		permissions.DeleteGroupPermission,
		permissions.ViewWorkspacePermission,
		permissions.CreateWorkspacePermission,
		permissions.UpdateWorkspacePermission,
		permissions.DeleteWorkspacePermission,
		permissions.ViewNamespaceMembershipPermission,
		permissions.CreateNamespaceMembershipPermission,
		permissions.UpdateNamespaceMembershipPermission,
		permissions.DeleteNamespaceMembershipPermission,
		permissions.ViewRunPermission,
		permissions.CreateRunPermission,
		permissions.ViewJobPermission,
		permissions.ViewRunnerPermission,
		permissions.CreateRunnerPermission,
		permissions.UpdateRunnerPermission,
		permissions.DeleteRunnerPermission,
		permissions.ViewVariablePermission,
		permissions.CreateVariablePermission,
		permissions.UpdateVariablePermission,
		permissions.DeleteVariablePermission,
		permissions.ViewVariableValuePermission,
		permissions.ViewTerraformProviderPermission,
		permissions.CreateTerraformProviderPermission,
		permissions.UpdateTerraformProviderPermission,
		permissions.DeleteTerraformProviderPermission,
		permissions.ViewTerraformModulePermission,
		permissions.CreateTerraformModulePermission,
		permissions.UpdateTerraformModulePermission,
		permissions.DeleteTerraformModulePermission,
		permissions.ViewStateVersionPermission,
		permissions.ViewStateVersionDataPermission,
		permissions.CreateStateVersionPermission,
		permissions.ViewConfigurationVersionPermission,
		permissions.CreateConfigurationVersionPermission,
		permissions.UpdateConfigurationVersionPermission,
		permissions.ViewServiceAccountPermission,
		permissions.CreateServiceAccountPermission,
		permissions.UpdateServiceAccountPermission,
		permissions.DeleteServiceAccountPermission,
		permissions.ViewManagedIdentityPermission,
		permissions.CreateManagedIdentityPermission,
		permissions.UpdateManagedIdentityPermission,
		permissions.DeleteManagedIdentityPermission,
		permissions.ViewVCSProviderPermission,
		permissions.CreateVCSProviderPermission,
		permissions.UpdateVCSProviderPermission,
		permissions.DeleteVCSProviderPermission,
		permissions.ViewTerraformProviderMirrorPermission,
		permissions.CreateTerraformProviderMirrorPermission,
		permissions.DeleteTerraformProviderMirrorPermission,
	},
	// Deployer Role.
	DeployerRoleID: {
		permissions.ViewGPGKeyPermission,
		permissions.CreateGPGKeyPermission,
		permissions.DeleteGPGKeyPermission,
		permissions.ViewGroupPermission,
		permissions.CreateGroupPermission,
		permissions.UpdateGroupPermission,
		permissions.DeleteGroupPermission,
		permissions.ViewWorkspacePermission,
		permissions.CreateWorkspacePermission,
		permissions.UpdateWorkspacePermission,
		permissions.DeleteWorkspacePermission,
		permissions.ViewNamespaceMembershipPermission,
		permissions.ViewRunPermission,
		permissions.CreateRunPermission,
		permissions.ViewJobPermission,
		permissions.ViewRunnerPermission,
		permissions.ViewVariablePermission,
		permissions.CreateVariablePermission,
		permissions.UpdateVariablePermission,
		permissions.DeleteVariablePermission,
		permissions.ViewVariableValuePermission,
		permissions.ViewTerraformProviderPermission,
		permissions.CreateTerraformProviderPermission,
		permissions.UpdateTerraformProviderPermission,
		permissions.DeleteTerraformProviderPermission,
		permissions.ViewTerraformModulePermission,
		permissions.CreateTerraformModulePermission,
		permissions.UpdateTerraformModulePermission,
		permissions.DeleteTerraformModulePermission,
		permissions.ViewStateVersionDataPermission,
		permissions.ViewStateVersionPermission,
		permissions.CreateStateVersionPermission,
		permissions.ViewConfigurationVersionPermission,
		permissions.CreateConfigurationVersionPermission,
		permissions.UpdateConfigurationVersionPermission,
		permissions.ViewServiceAccountPermission,
		permissions.CreateServiceAccountPermission,
		permissions.UpdateServiceAccountPermission,
		permissions.DeleteServiceAccountPermission,
		permissions.ViewManagedIdentityPermission,
		permissions.ViewVCSProviderPermission,
		permissions.CreateVCSProviderPermission,
		permissions.UpdateVCSProviderPermission,
		permissions.DeleteVCSProviderPermission,
		permissions.ViewTerraformProviderMirrorPermission,
		permissions.CreateTerraformProviderMirrorPermission,
		permissions.DeleteTerraformProviderMirrorPermission,
	},
	// Viewer Role.
	ViewerRoleID: {
		permissions.ViewGPGKeyPermission,
		permissions.ViewGroupPermission,
		permissions.ViewWorkspacePermission,
		permissions.ViewNamespaceMembershipPermission,
		permissions.ViewRunPermission,
		permissions.ViewJobPermission,
		permissions.ViewRunnerPermission,
		permissions.ViewVariablePermission,
		permissions.ViewTerraformProviderPermission,
		permissions.ViewTerraformModulePermission,
		permissions.ViewStateVersionPermission,
		permissions.ViewConfigurationVersionPermission,
		permissions.ViewServiceAccountPermission,
		permissions.ViewManagedIdentityPermission,
		permissions.ViewVCSProviderPermission,
		permissions.ViewTerraformProviderMirrorPermission,
	},
}
