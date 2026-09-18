package models

import (
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*Role)(nil)

// Role defines a subject's ability to access or modify
// resources within Tharsis. It provides a set of permissions
// that dictate which resources can be viewed or modified.
type Role struct {
	Name        string
	Description string
	CreatedBy   string
	permissions []Permission
	Metadata    ResourceMetadata
	// SortOrder is a fixed display position used to order default roles ahead of custom roles
	SortOrder int
}

// GetID returns the Metadata ID.
func (r *Role) GetID() string {
	return r.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (r *Role) GetGlobalID() string {
	return gid.ToGlobalID(r.GetModelType(), r.Metadata.ID)
}

// GetModelType returns the model type.
func (r *Role) GetModelType() types.ModelType {
	return types.RoleModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Role) ResolveMetadata(key string) (*string, error) {
	val, err := r.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			return &r.Name, nil
		case "sort_order":
			resp := strconv.Itoa(r.SortOrder)
			return &resp, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// SetPermissions sets permissions for a role.
func (r *Role) SetPermissions(perms []Permission) {
	r.permissions = perms
}

// GetPermissions returns permissions for a role.
func (r *Role) GetPermissions() []Permission {
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
	uniquePerms := []Permission{}
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
	OwnerRoleID      DefaultRoleID = "623c83ea-23fe-4de6-874a-a99ccf6a76fc"
	DeployerRoleID   DefaultRoleID = "8aa7adba-b769-471f-8ebb-3215f33991cb"
	PublisherRoleID  DefaultRoleID = "028fa46b-23ba-443f-a24f-61edcde148ff"
	ViewerRoleID     DefaultRoleID = "52da70fd-37b0-4349-bb64-fb4659bcf5f5"
	MaintainerRoleID DefaultRoleID = "d1d61904-1255-4e4b-a5ee-4b9543e970b4"
)

// String returns the ID as a string.
func (d DefaultRoleID) String() string {
	return string(d)
}

// IsDefaultRole returns true if ID belongs to a default role.
func (d DefaultRoleID) IsDefaultRole() bool {
	switch d {
	case OwnerRoleID, DeployerRoleID, PublisherRoleID, ViewerRoleID, MaintainerRoleID:
		return true
	}

	return false
}

// Permissions returns the Permission set for a default Tharsis role.
func (d DefaultRoleID) Permissions() ([]Permission, bool) {
	perms, ok := defaultRolePermissions[d]
	return perms, ok
}

// defaultRolePermissions is a map of default role's ID to its Permission set.
var defaultRolePermissions = map[DefaultRoleID][]Permission{
	// Owner Role
	OwnerRoleID: {
		ViewGPGKeyPermission,
		CreateGPGKeyPermission,
		DeleteGPGKeyPermission,
		ViewGroupPermission,
		CreateGroupPermission,
		UpdateGroupPermission,
		DeleteGroupPermission,
		ViewWorkspacePermission,
		CreateWorkspacePermission,
		UpdateWorkspacePermission,
		DeleteWorkspacePermission,
		ViewWorkspaceRoleBindingPermission,
		CreateWorkspaceRoleBindingPermission,
		UpdateWorkspaceRoleBindingPermission,
		DeleteWorkspaceRoleBindingPermission,
		ViewNamespaceMembershipPermission,
		CreateNamespaceMembershipPermission,
		UpdateNamespaceMembershipPermission,
		DeleteNamespaceMembershipPermission,
		ViewRunPermission,
		CreateRunPermission,
		ViewJobPermission,
		ViewRunnerPermission,
		CreateRunnerPermission,
		UpdateRunnerPermission,
		DeleteRunnerPermission,
		ViewVariablePermission,
		CreateVariablePermission,
		UpdateVariablePermission,
		DeleteVariablePermission,
		ViewSensitiveVariableValuePermission,
		ViewTerraformProviderPermission,
		CreateTerraformProviderPermission,
		UpdateTerraformProviderPermission,
		DeleteTerraformProviderPermission,
		ViewTerraformModulePermission,
		CreateTerraformModulePermission,
		UpdateTerraformModulePermission,
		DeleteTerraformModulePermission,
		ViewStateVersionPermission,
		ViewStateVersionDataPermission,
		CreateStateVersionPermission,
		ViewConfigurationVersionPermission,
		CreateConfigurationVersionPermission,
		UpdateConfigurationVersionPermission,
		ViewServiceAccountPermission,
		CreateServiceAccountPermission,
		UpdateServiceAccountPermission,
		DeleteServiceAccountPermission,
		ViewManagedIdentityPermission,
		CreateManagedIdentityPermission,
		UpdateManagedIdentityPermission,
		DeleteManagedIdentityPermission,
		ViewVCSProviderPermission,
		CreateVCSProviderPermission,
		UpdateVCSProviderPermission,
		DeleteVCSProviderPermission,
		ViewTerraformProviderMirrorPermission,
		CreateTerraformProviderMirrorPermission,
		DeleteTerraformProviderMirrorPermission,
		ViewFederatedRegistryPermission,
		CreateFederatedRegistryPermission,
		UpdateFederatedRegistryPermission,
		DeleteFederatedRegistryPermission,
		ViewPackagePermission,
		CreatePackagePermission,
		UpdatePackagePermission,
		DeletePackagePermission,
		ViewPolicyPermission,
		CreatePolicyPermission,
		UpdatePolicyPermission,
		DeletePolicyPermission,
		ViewCleanupPolicyPermission,
		CreateCleanupPolicyPermission,
		UpdateCleanupPolicyPermission,
		DeleteCleanupPolicyPermission,
	},
	// Maintainer Role.
	// Everything Owner has except the ability to add, change, or remove namespace memberships —
	// the permission that governs conferring authority on other principals. This mirrors the
	// escalation boundary Deployer already respects (it too lacks Create/Update/DeleteNamespaceMembershipPermission),
	// extended to a role that otherwise carries Owner's full set.
	MaintainerRoleID: {
		ViewGPGKeyPermission,
		CreateGPGKeyPermission,
		DeleteGPGKeyPermission,
		ViewGroupPermission,
		CreateGroupPermission,
		UpdateGroupPermission,
		DeleteGroupPermission,
		ViewWorkspacePermission,
		CreateWorkspacePermission,
		UpdateWorkspacePermission,
		DeleteWorkspacePermission,
		ViewWorkspaceRoleBindingPermission,
		CreateWorkspaceRoleBindingPermission,
		UpdateWorkspaceRoleBindingPermission,
		DeleteWorkspaceRoleBindingPermission,
		ViewNamespaceMembershipPermission,
		ViewRunPermission,
		CreateRunPermission,
		ViewJobPermission,
		ViewRunnerPermission,
		CreateRunnerPermission,
		UpdateRunnerPermission,
		DeleteRunnerPermission,
		ViewVariablePermission,
		CreateVariablePermission,
		UpdateVariablePermission,
		DeleteVariablePermission,
		ViewSensitiveVariableValuePermission,
		ViewTerraformProviderPermission,
		CreateTerraformProviderPermission,
		UpdateTerraformProviderPermission,
		DeleteTerraformProviderPermission,
		ViewTerraformModulePermission,
		CreateTerraformModulePermission,
		UpdateTerraformModulePermission,
		DeleteTerraformModulePermission,
		ViewStateVersionPermission,
		ViewStateVersionDataPermission,
		CreateStateVersionPermission,
		ViewConfigurationVersionPermission,
		CreateConfigurationVersionPermission,
		UpdateConfigurationVersionPermission,
		ViewServiceAccountPermission,
		CreateServiceAccountPermission,
		UpdateServiceAccountPermission,
		DeleteServiceAccountPermission,
		ViewManagedIdentityPermission,
		CreateManagedIdentityPermission,
		UpdateManagedIdentityPermission,
		DeleteManagedIdentityPermission,
		ViewVCSProviderPermission,
		CreateVCSProviderPermission,
		UpdateVCSProviderPermission,
		DeleteVCSProviderPermission,
		ViewTerraformProviderMirrorPermission,
		CreateTerraformProviderMirrorPermission,
		DeleteTerraformProviderMirrorPermission,
		ViewFederatedRegistryPermission,
		CreateFederatedRegistryPermission,
		UpdateFederatedRegistryPermission,
		DeleteFederatedRegistryPermission,
		ViewPackagePermission,
		CreatePackagePermission,
		UpdatePackagePermission,
		DeletePackagePermission,
		ViewPolicyPermission,
		CreatePolicyPermission,
		UpdatePolicyPermission,
		DeletePolicyPermission,
		ViewCleanupPolicyPermission,
		CreateCleanupPolicyPermission,
		UpdateCleanupPolicyPermission,
		DeleteCleanupPolicyPermission,
	},
	// Deployer Role.
	DeployerRoleID: {
		ViewGPGKeyPermission,
		CreateGPGKeyPermission,
		DeleteGPGKeyPermission,
		ViewGroupPermission,
		CreateGroupPermission,
		UpdateGroupPermission,
		DeleteGroupPermission,
		ViewWorkspacePermission,
		CreateWorkspacePermission,
		UpdateWorkspacePermission,
		DeleteWorkspacePermission,
		ViewWorkspaceRoleBindingPermission,
		ViewNamespaceMembershipPermission,
		ViewRunPermission,
		CreateRunPermission,
		ViewJobPermission,
		ViewRunnerPermission,
		ViewVariablePermission,
		CreateVariablePermission,
		UpdateVariablePermission,
		DeleteVariablePermission,
		ViewSensitiveVariableValuePermission,
		ViewTerraformProviderPermission,
		CreateTerraformProviderPermission,
		UpdateTerraformProviderPermission,
		DeleteTerraformProviderPermission,
		ViewTerraformModulePermission,
		CreateTerraformModulePermission,
		UpdateTerraformModulePermission,
		DeleteTerraformModulePermission,
		ViewStateVersionDataPermission,
		ViewStateVersionPermission,
		CreateStateVersionPermission,
		ViewConfigurationVersionPermission,
		CreateConfigurationVersionPermission,
		UpdateConfigurationVersionPermission,
		ViewServiceAccountPermission,
		CreateServiceAccountPermission,
		UpdateServiceAccountPermission,
		DeleteServiceAccountPermission,
		ViewManagedIdentityPermission,
		ViewVCSProviderPermission,
		CreateVCSProviderPermission,
		UpdateVCSProviderPermission,
		DeleteVCSProviderPermission,
		ViewTerraformProviderMirrorPermission,
		CreateTerraformProviderMirrorPermission,
		DeleteTerraformProviderMirrorPermission,
		ViewFederatedRegistryPermission,
		CreateFederatedRegistryPermission,
		UpdateFederatedRegistryPermission,
		DeleteFederatedRegistryPermission,
		ViewPackagePermission,
		CreatePackagePermission,
		UpdatePackagePermission,
		DeletePackagePermission,
		ViewPolicyPermission,
		ViewCleanupPolicyPermission,
	},
	// Publisher Role.
	// Allows publishing terraform modules and providers with read access to all resources — no run/deployment permissions.
	PublisherRoleID: {
		ViewGPGKeyPermission,
		ViewGroupPermission,
		ViewWorkspacePermission,
		ViewWorkspaceRoleBindingPermission,
		ViewNamespaceMembershipPermission,
		ViewRunPermission,
		ViewJobPermission,
		ViewRunnerPermission,
		ViewVariablePermission,
		ViewTerraformModulePermission,
		CreateTerraformModulePermission,
		UpdateTerraformModulePermission,
		DeleteTerraformModulePermission,
		ViewTerraformProviderPermission,
		CreateTerraformProviderPermission,
		UpdateTerraformProviderPermission,
		DeleteTerraformProviderPermission,
		ViewStateVersionPermission,
		ViewConfigurationVersionPermission,
		ViewServiceAccountPermission,
		ViewManagedIdentityPermission,
		ViewVCSProviderPermission,
		ViewTerraformProviderMirrorPermission,
		ViewFederatedRegistryPermission,
		ViewPackagePermission,
		CreatePackagePermission,
		UpdatePackagePermission,
		DeletePackagePermission,
		ViewPolicyPermission,
		ViewCleanupPolicyPermission,
	},
	// Viewer Role.
	ViewerRoleID: {
		ViewGPGKeyPermission,
		ViewGroupPermission,
		ViewWorkspacePermission,
		ViewWorkspaceRoleBindingPermission,
		ViewNamespaceMembershipPermission,
		ViewRunPermission,
		ViewJobPermission,
		ViewRunnerPermission,
		ViewVariablePermission,
		ViewTerraformProviderPermission,
		ViewTerraformModulePermission,
		ViewStateVersionPermission,
		ViewConfigurationVersionPermission,
		ViewServiceAccountPermission,
		ViewManagedIdentityPermission,
		ViewVCSProviderPermission,
		ViewTerraformProviderMirrorPermission,
		ViewFederatedRegistryPermission,
		ViewPackagePermission,
		ViewPolicyPermission,
		ViewCleanupPolicyPermission,
	},
}
