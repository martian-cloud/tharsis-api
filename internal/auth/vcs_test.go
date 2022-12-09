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

func TestVCSWorkspaceLinkCaller_GetNamespaceAccessPolicy(t *testing.T) {
	expectedAccessPolicy := &NamespaceAccessPolicy{
		AllowAll:         false,
		RootNamespaceIDs: []string{},
	}

	caller := VCSWorkspaceLinkCaller{}
	accessPolicy, err := caller.GetNamespaceAccessPolicy(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
	assert.Equal(t, expectedAccessPolicy, accessPolicy)
}

func TestVCSWorkspaceLinkCaller_RequireAccessToNamespace(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{
		Link: &models.WorkspaceVCSProviderLink{
			WorkspaceID: "workspace-id",
		},
	}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expect    error
		workspace *models.Workspace
		name      string
	}{
		{
			name: "positive: requested workspace is the one workspace vcs provider link belongs to; grant access",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "workspace-id",
				},
			},
		},
		{
			name:   "negative: requested workspace doesn't exist; deny access",
			expect: authorizationError(ctx, false),
		},
		{
			name: "negative: requested workspace ID doesn't match the link's workspace ID, deny access",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "another-workspace-id",
				},
			},
			expect: authorizationError(ctx, false),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.On("GetWorkspaceByFullPath", mock.Anything, mock.Anything).Return(test.workspace, nil)
			caller.dbClient = &db.Client{Workspaces: &mockWorkspaces}

			err := caller.RequireAccessToNamespace(ctx, "workspace-id", models.DeployerRole)
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

func TestVCSWorkspaceLinkCaller_RequireViewerAccessToGroups(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireViewerAccessToGroups(WithCaller(context.Background(), &caller), []models.Group{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireViewerAccessToWorkspaces(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireViewerAccessToWorkspaces(WithCaller(context.Background(), &caller), []models.Workspace{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireViewerAccessToNamespaces(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireViewerAccessToNamespaces(WithCaller(context.Background(), &caller), []string{})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireAccessToGroup(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireAccessToGroup(WithCaller(context.Background(), &caller), "1", models.DeployerRole)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireAccessToWorkspace(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{
		Link: &models.WorkspaceVCSProviderLink{
			WorkspaceID: "workspace-id",
		},
	}
	// Allow access.
	err := caller.RequireAccessToWorkspace(WithCaller(context.Background(), &caller), "workspace-id", models.DeployerRole)
	assert.Nil(t, err)
	// Deny access.
	err = caller.RequireAccessToWorkspace(WithCaller(context.Background(), &caller), "another-workspace-id", models.DeployerRole)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireAccessToInheritedGroupResource(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireAccessToInheritedGroupResource(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireAccessToInheritedNamespaceResource(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireAccessToInheritedNamespaceResource(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireRunWriteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireRunWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequirePlanWriteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequirePlanWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireApplyWriteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireApplyWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireJobWriteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireJobWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireTeamCreateAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller))
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireTeamUpdateAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireTeamUpdateAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireTeamDeleteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireUserCreateAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller))
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireUserUpdateAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestVCSWorkspaceLinkCaller_RequireUserDeleteAccess(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireUserDeleteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}
