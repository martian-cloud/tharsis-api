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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSCIMCaller_GetSubject(t *testing.T) {
	caller := SCIMCaller{}
	assert.Equal(t, "scim", caller.GetSubject())
}

func TestSCIMCaller_IsAdmin(t *testing.T) {
	caller := SCIMCaller{}
	assert.False(t, caller.IsAdmin())
}

func TestSCIMCaller_GetNamespaceAccessPolicy(t *testing.T) {
	expectedAccessPolicy := &NamespaceAccessPolicy{
		AllowAll:         false,
		RootNamespaceIDs: []string{},
	}

	caller := SCIMCaller{}
	accessPolicy, err := caller.GetNamespaceAccessPolicy(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
	assert.Equal(t, expectedAccessPolicy, accessPolicy)
}

func TestSCIMCaller_RequirePermissions(t *testing.T) {
	invalid := "invalid"
	teamID := "team-1"
	userID := "user-1"

	caller := SCIMCaller{}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expect            error
		team              *models.Team
		user              *models.User
		name              string
		perms             []permissions.Permission
		constraints       []func(*constraints)
		inMaintenanceMode bool
	}{
		{
			name: "viewing, creating, updating a team or a user; grant access",
			perms: []permissions.Permission{
				permissions.CreateTeamPermission,
				permissions.UpdateTeamPermission,
				permissions.CreateUserPermission,
				permissions.UpdateUserPermission,
			},
		},
		{
			name:        "deleting a team created by SCIM",
			team:        &models.Team{SCIMExternalID: "scim-team-1"},
			perms:       []permissions.Permission{permissions.DeleteTeamPermission},
			constraints: []func(*constraints){WithTeamID(teamID)},
		},
		{
			name:        "access denied because deleting a team not created by SCIM",
			team:        &models.Team{},
			perms:       []permissions.Permission{permissions.DeleteTeamPermission},
			constraints: []func(*constraints){WithTeamID(teamID)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because deleting a team that doesn't exist",
			perms:       []permissions.Permission{permissions.DeleteTeamPermission},
			constraints: []func(*constraints){WithTeamID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "deleting a user created by SCIM",
			user:        &models.User{SCIMExternalID: "scim-user-1"},
			perms:       []permissions.Permission{permissions.DeleteUserPermission},
			constraints: []func(*constraints){WithUserID(userID)},
		},
		{
			name:        "access denied because deleting a user not created by SCIM",
			user:        &models.User{},
			perms:       []permissions.Permission{permissions.DeleteUserPermission},
			constraints: []func(*constraints){WithUserID(userID)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because deleting a user that doesn't exist",
			perms:       []permissions.Permission{permissions.DeleteUserPermission},
			constraints: []func(*constraints){WithUserID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:   "access denied because required constraints not provided",
			perms:  []permissions.Permission{permissions.DeleteTeamPermission, permissions.DeleteUserPermission},
			expect: errMissingConstraints,
		},
		{
			name:   "access denied because permission is never available to caller",
			perms:  []permissions.Permission{permissions.CreateGroupPermission},
			expect: authorizationError(ctx, false),
		},
		{
			name:              "cannot have write permission when server in maintenance mode",
			perms:             []permissions.Permission{permissions.CreateTeamPermission},
			expect:            errInMaintenanceMode,
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.NewMockTeams(t)
			mockUsers := db.NewMockUsers(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			constraints := getConstraints(test.constraints...)

			if constraints.teamID != nil {
				mockTeams.On("GetTeamByID", mock.Anything, mock.Anything).Return(test.team, nil)
			}

			if constraints.userID != nil {
				mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(test.user, nil)
			}

			caller.maintenanceMonitor = mockMaintenanceMonitor
			caller.dbClient = &db.Client{Teams: mockTeams, Users: mockUsers}

			for _, perm := range test.perms {
				assert.Equal(t, test.expect, caller.RequirePermission(ctx, perm, test.constraints...))
			}
		})
	}
}

func TestSCIMCaller_RequireInheritedPermissions(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), permissions.RunnerResourceType, nil)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}
