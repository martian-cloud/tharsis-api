package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestUserCaller_GetSubject(t *testing.T) {
	caller := UserCaller{User: &models.User{Email: "user@email"}}
	assert.Equal(t, "user@email", caller.GetSubject())
}

func TestUserCaller_IsAdmin(t *testing.T) {
	caller := UserCaller{User: &models.User{}}
	assert.False(t, caller.IsAdmin())

	caller.User.Admin = true
	assert.True(t, caller.IsAdmin())
}

func TestUserCaller_GetNamespaceAccessPolicy(t *testing.T) {
	caller := UserCaller{User: &models.User{}}
	ctx := WithCaller(context.Background(), &caller)

	// Admin case.
	caller.User.Admin = true
	policy, err := caller.GetNamespaceAccessPolicy(ctx)
	assert.Nil(t, err)
	assert.Equal(t, &NamespaceAccessPolicy{AllowAll: true}, policy)

	// Non-admin case.
	caller.User.Admin = false
	membershipNamespaceID := "nm-1"

	mockAuthorizer := NewMockAuthorizer(t)
	mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{{ID: membershipNamespaceID}}, nil)
	caller.authorizer = mockAuthorizer

	policy, err = caller.GetNamespaceAccessPolicy(ctx)
	assert.Nil(t, err)
	assert.Equal(t, &NamespaceAccessPolicy{AllowAll: false, RootNamespaceIDs: []string{membershipNamespaceID}}, policy)
}

func TestUserCaller_RequirePermissions(t *testing.T) {
	teamID := "team1"
	caller := UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Email: "user@email"}}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name              string
		expect            error
		teamMember        *models.TeamMember
		perm              permissions.Permission
		constraints       []func(*constraints)
		isAdmin           bool
		withAuthorizer    bool
		inMaintenanceMode bool
	}{
		{
			name:           "access is granted by the authorizer",
			perm:           permissions.ViewGroupPermission,
			constraints:    []func(*constraints){WithNamespacePath("namespace")},
			withAuthorizer: true,
		},
		{
			name:           "access denied by the authorizer because a permission is not satisfied",
			perm:           permissions.DeleteGroupPermission,
			constraints:    []func(*constraints){WithNamespacePath("namespace")},
			expect:         authorizationError(ctx, false),
			withAuthorizer: true,
		},
		{
			name:    "permissions are only granted since user is admin",
			perm:    permissions.CreateTeamPermission,
			isAdmin: true,
		},
		{
			name:        "access forbidden because user must be an admin",
			perm:        permissions.CreateTeamPermission,
			constraints: []func(*constraints){WithGroupID("team-1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "team update allowed since user is an admin",
			perm:        permissions.UpdateTeamPermission,
			constraints: []func(*constraints){WithTeamID(teamID)},
			isAdmin:     true,
		},
		{
			name:        "access denied because user is not an admin or a team maintainer",
			teamMember:  &models.TeamMember{IsMaintainer: false},
			perm:        permissions.UpdateTeamPermission,
			constraints: []func(*constraints){WithTeamID(teamID)},
			expect:      authorizationError(ctx, true),
		},
		{
			name:        "access denied because team member not found",
			perm:        permissions.UpdateTeamPermission,
			constraints: []func(*constraints){WithTeamID(teamID)},
			expect:      authorizationError(ctx, true),
		},
		{
			name:        "access denied because user is not a maintainer",
			teamMember:  &models.TeamMember{IsMaintainer: false},
			perm:        permissions.UpdateTeamPermission,
			constraints: []func(*constraints){WithTeamID(teamID)},
			expect:      authorizationError(ctx, true),
		},
		{
			name:           "access denied because required constraints are not specified",
			perm:           permissions.ViewWorkspacePermission,
			expect:         errMissingConstraints,
			withAuthorizer: true,
		},
		{
			name:              "cannot have write permission when system is in maintenance mode",
			perm:              permissions.CreateWorkspacePermission,
			expect:            errInMaintenanceMode,
			inMaintenanceMode: true,
		},
		{
			name:              "can have read permission when system is in maintenance mode",
			perm:              permissions.ViewWorkspacePermission,
			constraints:       []func(*constraints){WithWorkspaceID("ws-1")},
			withAuthorizer:    true,
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			if test.perm == permissions.UpdateTeamPermission && !test.isAdmin {
				mockTeamMembers.On("GetTeamMember", mock.Anything, caller.User.Metadata.ID, teamID).Return(test.teamMember, nil)
			}

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{test.perm}, mock.Anything).Return(requireAccessAuthorizerFunc)
			}

			caller.User.Admin = test.isAdmin
			caller.authorizer = mockAuthorizer
			caller.maintenanceMonitor = mockMaintenanceMonitor
			caller.dbClient = &db.Client{TeamMembers: mockTeamMembers}

			assert.Equal(t, test.expect, caller.RequirePermission(ctx, test.perm, test.constraints...))
		})
	}
}

func TestUserCaller_RequireInheritedPermissions(t *testing.T) {
	caller := UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Email: "user@email"}}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name           string
		expect         error
		resourceType   permissions.ResourceType
		constraints    []func(*constraints)
		isAdmin        bool
		withAuthorizer bool
	}{
		{
			name:           "multiple permissions granted by the authorizer",
			resourceType:   permissions.ManagedIdentityResourceType,
			constraints:    []func(*constraints){WithGroupID("group1")},
			withAuthorizer: true,
		},
		{
			name:           "access denied by the authorizer because a permission is not satisfied",
			resourceType:   permissions.ApplyResourceType, // Just using an invalid resource here to deny access.
			constraints:    []func(*constraints){WithWorkspaceID("ws2")},
			expect:         authorizationError(ctx, false),
			withAuthorizer: true,
		},
		{
			name:         "permissions granted since user is admin",
			resourceType: permissions.GPGKeyResourceType,
			isAdmin:      true,
		},
		{
			name:           "access denied because required constraints are not specified",
			resourceType:   permissions.TerraformModuleResourceType,
			expect:         errMissingConstraints,
			withAuthorizer: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccessToInheritableResource", mock.Anything, []permissions.ResourceType{test.resourceType}, mock.Anything).Return(requireInheritedAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer
			caller.User.Admin = test.isAdmin
			assert.Equal(t, test.expect, caller.RequireAccessToInheritableResource(ctx, test.resourceType, test.constraints...))
		})
	}
}

// The End.
