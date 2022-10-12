package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

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

func TestSCIMCaller_RequireAccessToNamespace(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToNamespace(WithCaller(context.Background(), &caller), "1", models.DeployerRole)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireViewerAccessToGroups(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireViewerAccessToGroups(WithCaller(context.Background(), &caller), []models.Group{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireViewerAccessToWorkspaces(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireViewerAccessToWorkspaces(WithCaller(context.Background(), &caller), []models.Workspace{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireViewerAccessToNamespaces(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireViewerAccessToNamespaces(WithCaller(context.Background(), &caller), []string{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireAccessToGroup(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToGroup(WithCaller(context.Background(), &caller), "1", models.DeployerRole)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireAccessToWorkspace(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToWorkspace(WithCaller(context.Background(), &caller), "1", models.DeployerRole)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireAccessToInheritedGroupResource(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToInheritedGroupResource(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireAccessToInheritedNamespaceResource(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToInheritedNamespaceResource(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireRunWriteAccess(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireRunWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequirePlanWriteAccess(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequirePlanWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireApplyWriteAccess(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireApplyWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireJobWriteAccess(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireJobWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_RequireTeamCreateAccess(t *testing.T) {
	caller := SCIMCaller{}
	assert.Nil(t, caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestSCIMCaller_RequireTeamUpdateAccess(t *testing.T) {
	caller := SCIMCaller{}
	assert.Nil(t, caller.RequireTeamUpdateAccess(WithCaller(context.Background(), &caller), "1"))
}

func TestSCIMCaller_RequireTeamDeleteAccess(t *testing.T) {
	caller := SCIMCaller{}
	ctx := WithCaller(context.Background(), &caller)

	tests := []struct {
		expect error
		team   *models.Team
		name   string
	}{
		{
			name: "positive: team with SCIMExternalID. Grant access.",
			team: &models.Team{
				Name:           "positive-test-team",
				Description:    "positive team description",
				SCIMExternalID: "positive-scim-id",
			},
			// expect errors to be nil
		},
		{
			name: "negative: team without SCIMExternalID. Deny access.",
			team: &models.Team{
				Name:        "negative-test-team",
				Description: "negative team description",
			},
			expect: authorizationError(ctx, false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			mockTeam := db.MockTeams{}
			mockTeam.On("GetTeamByID", mock.Anything, mock.Anything).Return(test.team, nil)
			caller.dbClient = &db.Client{Teams: &mockTeam}

			err := caller.RequireTeamDeleteAccess(ctx, "a-fake-team-ID")
			if test.expect == nil {
				// Positive case.
				assert.Nil(t, err)
			} else {
				// Negative case.
				assert.EqualError(t, err, test.expect.Error())
			}
		})
	}
}

func TestSCIMCaller_RequireUserCreateAccess(t *testing.T) {
	caller := SCIMCaller{}
	assert.Nil(t, caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestSCIMCaller_RequireUserUpdateAccess(t *testing.T) {
	caller := SCIMCaller{}
	assert.Nil(t, caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "1"))
}

func TestSCIMCaller_RequireUserDeleteAccess(t *testing.T) {
	caller := SCIMCaller{}
	ctx := WithCaller(context.Background(), &caller)

	tests := []struct {
		expect error
		user   *models.User
		name   string
	}{
		{
			name: "positive: user with SCIMExternalID. Grant access.",
			user: &models.User{
				Username:       "positive-test-user",
				SCIMExternalID: "positive-scim-id",
			},
			// expect errors to be nil
		},
		{
			name: "negative: user without SCIMExternalID. Deny access.",
			user: &models.User{
				Username: "negative-test-user",
			},
			expect: authorizationError(ctx, false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			mockUsers := db.MockUsers{}
			mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(test.user, nil)
			caller.dbClient = &db.Client{Users: &mockUsers}

			err := caller.RequireUserDeleteAccess(ctx, "a-fake-user-ID")
			if test.expect == nil {
				// Positive case.
				assert.Nil(t, err)
			} else {
				// Negative case.
				assert.EqualError(t, err, test.expect.Error())
			}
		})
	}
}
