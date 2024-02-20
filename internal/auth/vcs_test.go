package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestVCSWorkspaceLinkCaller_GetSubject(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{Provider: &models.VCSProvider{
		ResourcePath: "rs1",
	}}
	assert.Equal(t, "rs1", caller.GetSubject())
}

func TestVCSWorkspaceLinkCaller_IsAdmin(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	assert.False(t, caller.IsAdmin())
}

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

func TestVCSWorkspaceLinkCaller_RequirePermissions(t *testing.T) {
	invalid := "invalid"

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws-1",
		},
	}

	caller := VCSWorkspaceLinkCaller{
		Provider: &models.VCSProvider{
			ResourcePath: "group1/vcs-provider",
		},
		Link: &models.WorkspaceVCSProviderLink{
			WorkspaceID: ws.Metadata.ID,
		},
	}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expectErrorCode   errors.CodeType
		name              string
		workspace         *models.Workspace
		perm              permissions.Permission
		constraints       []func(*constraints)
		inMaintenanceMode bool
	}{
		{
			name:        "link belongs to requested workspace",
			perm:        permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(ws.Metadata.ID)},
		},
		{
			name:            "access denied because link doesn't belong to requested workspace",
			perm:            permissions.ViewWorkspacePermission,
			constraints:     []func(*constraints){WithWorkspaceID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "link is creating a run in its own workspace",
			perm:        permissions.CreateRunPermission,
			constraints: []func(*constraints){WithWorkspaceID(ws.Metadata.ID)},
		},
		{
			name:            "access denied because link is creating a run outside its own workspace",
			perm:            permissions.CreateRunPermission,
			constraints:     []func(*constraints){WithWorkspaceID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because required constraint not provided",
			perm:            permissions.CreateRunPermission,
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "access denied because permission is never available to caller",
			perm:            permissions.CreateGroupPermission,
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "cannot have write permission when system is in maintenance mode",
			perm: permissions.CreateRunPermission,
			constraints: []func(*constraints){
				WithWorkspaceID(ws.Metadata.ID),
			},
			expectErrorCode:   errors.EServiceUnavailable,
			inMaintenanceMode: true,
		},
		{
			name: "can have read permission when system is in maintenance mode",
			perm: permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){
				WithWorkspaceID(ws.Metadata.ID),
			},
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			caller.maintenanceMonitor = mockMaintenanceMonitor

			err := caller.RequirePermission(ctx, test.perm, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestVCSWorkspaceLinkCaller_RequireInheritedPermissions(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{
		Provider: &models.VCSProvider{
			ResourcePath: "group1/vcs-provider",
		},
	}
	err := caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), permissions.ApplyResourceType, nil)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}
