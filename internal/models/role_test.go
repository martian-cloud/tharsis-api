package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkspaceRoleBindingPermissionsAreOwnerOnly asserts the escalation boundary for workspace role
// bindings: binding a role to a workspace confers that role's permissions on the workspace's job
// caller at the parent namespace, so only a role that can already confer access there may do it.
//
// Deployer must not hold the mutating permissions. Deployer already cannot add a human as a namespace
// member (it holds only ViewNamespaceMembershipPermission), and letting it confer authority on a
// workspace instead would contradict that in the more dangerous direction, since a workspace's
// authority is exercised by whoever can trigger a run in it.
func TestWorkspaceRoleBindingPermissionsAreOwnerOnly(t *testing.T) {
	mutating := []Permission{
		CreateWorkspaceRoleBindingPermission,
		UpdateWorkspaceRoleBindingPermission,
		DeleteWorkspaceRoleBindingPermission,
	}

	tests := []struct {
		name             string
		roleID           DefaultRoleID
		expectMutating   bool
		expectViewAccess bool
	}{
		{name: "owner holds the mutating permissions", roleID: OwnerRoleID, expectMutating: true, expectViewAccess: true},
		{name: "maintainer holds the mutating permissions", roleID: MaintainerRoleID, expectMutating: true, expectViewAccess: true},
		{name: "deployer cannot confer bindings", roleID: DeployerRoleID, expectMutating: false, expectViewAccess: true},
		{name: "publisher cannot confer bindings", roleID: PublisherRoleID, expectMutating: false, expectViewAccess: true},
		{name: "viewer cannot confer bindings", roleID: ViewerRoleID, expectMutating: false, expectViewAccess: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			perms, ok := test.roleID.Permissions()
			require.True(t, ok, "expected %s to be a default role", test.roleID)

			held := map[string]struct{}{}
			for _, p := range perms {
				held[p.String()] = struct{}{}
			}

			for _, want := range mutating {
				_, has := held[want.String()]
				assert.Equal(t, test.expectMutating, has, "permission %s", want)
			}

			_, hasView := held[ViewWorkspaceRoleBindingPermission.String()]
			assert.Equal(t, test.expectViewAccess, hasView, "permission %s", ViewWorkspaceRoleBindingPermission)
		})
	}
}

// TestWorkspaceRoleBindingPermissionsAreAssignable ensures the new permissions can be placed in a
// custom role. This is what allows an admin to build a narrow "controller" role — a deployer-like
// permission set plus CreateWorkspaceRoleBindingPermission — so a workspace can bind roles to the
// workspaces it creates without being granted full owner.
func TestWorkspaceRoleBindingPermissionsAreAssignable(t *testing.T) {
	for _, p := range []Permission{
		ViewWorkspaceRoleBindingPermission,
		CreateWorkspaceRoleBindingPermission,
		UpdateWorkspaceRoleBindingPermission,
		DeleteWorkspaceRoleBindingPermission,
	} {
		permCopy := p
		assert.True(t, permCopy.IsAssignable(), "permission %s should be assignable to a role", p)
	}
}

// TestDeployerIsSubsetOfOwner guards the ordering the binding gate relies on: a caller who holds
// Owner at a namespace can confer Deployer there, because Owner's permission set contains Deployer's.
func TestDeployerIsSubsetOfOwner(t *testing.T) {
	ownerPerms, ok := OwnerRoleID.Permissions()
	require.True(t, ok)
	deployerPerms, ok := DeployerRoleID.Permissions()
	require.True(t, ok)

	owned := map[string]struct{}{}
	for _, p := range ownerPerms {
		owned[p.String()] = struct{}{}
	}

	for _, p := range deployerPerms {
		_, has := owned[p.String()]
		assert.True(t, has, "owner is missing deployer permission %s", p)
	}
}

// TestMaintainerHasEveryOwnerPermissionExceptNamespaceMembershipMutation asserts Maintainer's
// defining property: it holds everything Owner holds, EXCEPT the three permissions that govern
// adding, changing, or removing a namespace membership. Those three are the permissions that let a
// subject confer authority on other principals, which is the one capability Maintainer must not
// have — everything else Owner can do, Maintainer can do too.
func TestMaintainerHasEveryOwnerPermissionExceptNamespaceMembershipMutation(t *testing.T) {
	excluded := []Permission{
		CreateNamespaceMembershipPermission,
		UpdateNamespaceMembershipPermission,
		DeleteNamespaceMembershipPermission,
	}

	ownerPerms, ok := OwnerRoleID.Permissions()
	require.True(t, ok)
	maintainerPerms, ok := MaintainerRoleID.Permissions()
	require.True(t, ok)

	maintainerSet := map[string]struct{}{}
	for _, p := range maintainerPerms {
		maintainerSet[p.String()] = struct{}{}
	}

	excludedSet := map[string]struct{}{}
	for _, p := range excluded {
		excludedSet[p.String()] = struct{}{}
	}

	// Every Owner permission not in the excluded set must be present in Maintainer.
	for _, p := range ownerPerms {
		permCopy := p
		if _, isExcluded := excludedSet[permCopy.String()]; isExcluded {
			continue
		}
		_, has := maintainerSet[permCopy.String()]
		assert.True(t, has, "maintainer is missing owner permission %s", permCopy)
	}

	// None of the excluded permissions may be present in Maintainer.
	for _, p := range excluded {
		permCopy := p
		_, has := maintainerSet[permCopy.String()]
		assert.False(t, has, "maintainer must not hold %s", permCopy)
	}

	// Maintainer must not hold any permission Owner doesn't (no set is broader than intended).
	ownerSet := map[string]struct{}{}
	for _, p := range ownerPerms {
		ownerSet[p.String()] = struct{}{}
	}
	for _, p := range maintainerPerms {
		permCopy := p
		_, has := ownerSet[permCopy.String()]
		assert.True(t, has, "maintainer holds permission %s that owner does not", permCopy)
	}

	// Maintainer still retains view access to namespace memberships.
	_, hasView := maintainerSet[ViewNamespaceMembershipPermission.String()]
	assert.True(t, hasView, "maintainer should retain ViewNamespaceMembershipPermission")
}

// TestMaintainerIsDefaultRole confirms the new role participates in the same default-role dispatch
// as the others: DefaultRoleID(id).Permissions() must resolve it (bypassing any stored permissions
// column, same as every other default role), and IsDefaultRole must recognize it.
func TestMaintainerIsDefaultRole(t *testing.T) {
	assert.True(t, MaintainerRoleID.IsDefaultRole())

	perms, ok := MaintainerRoleID.Permissions()
	assert.True(t, ok)
	assert.NotEmpty(t, perms)
}
