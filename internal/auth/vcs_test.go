package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestVCSWorkspaceLinkCaller_GetSubject(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{Provider: &models.VCSProvider{
		ResourcePath: "rs1",
	}}
	assert.Equal(t, "rs1", caller.GetSubject())
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
		Link: &models.WorkspaceVCSProviderLink{
			WorkspaceID: ws.Metadata.ID,
		},
	}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expect      error
		name        string
		workspace   *models.Workspace
		perm        permissions.Permission
		constraints []func(*constraints)
	}{
		{
			name:        "link belongs to requested workspace",
			perm:        permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(ws.Metadata.ID)},
		},
		{
			name:        "access denied because link doesn't belong to requested workspace",
			perm:        permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "link is creating a run in its own workspace",
			perm:        permissions.CreateRunPermission,
			constraints: []func(*constraints){WithWorkspaceID(ws.Metadata.ID)},
		},
		{
			name:        "access denied because link is creating a run outside its own workspace",
			perm:        permissions.CreateRunPermission,
			constraints: []func(*constraints){WithWorkspaceID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:   "access denied because required constraint not provided",
			perm:   permissions.CreateRunPermission,
			expect: errMissingConstraints,
		},
		{
			name:   "access denied because permission is never available to caller",
			perm:   permissions.CreateGroupPermission,
			expect: authorizationError(ctx, false),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, caller.RequirePermission(ctx, test.perm, test.constraints...))
		})
	}
}

func TestVCSWorkspaceLinkCaller_RequireInheritedPermissions(t *testing.T) {
	caller := VCSWorkspaceLinkCaller{}
	err := caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), permissions.ApplyResourceType, nil)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}
