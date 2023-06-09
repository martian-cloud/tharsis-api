package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// VCSWorkspaceLinkCaller represents a VCS provider subject.
type VCSWorkspaceLinkCaller struct {
	Provider *models.VCSProvider
	Link     *models.WorkspaceVCSProviderLink
	dbClient *db.Client
}

// NewVCSWorkspaceLinkCaller returns a new VCS caller.
func NewVCSWorkspaceLinkCaller(provider *models.VCSProvider, link *models.WorkspaceVCSProviderLink, dbClient *db.Client) *VCSWorkspaceLinkCaller {
	return &VCSWorkspaceLinkCaller{
		Provider: provider,
		Link:     link,
		dbClient: dbClient,
	}
}

// GetSubject returns the subject identifier for this caller.
func (v *VCSWorkspaceLinkCaller) GetSubject() string {
	return v.Provider.ResourcePath
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller.
func (v *VCSWorkspaceLinkCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces.
		RootNamespaceIDs: []string{},
	}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions.
func (v *VCSWorkspaceLinkCaller) RequirePermission(ctx context.Context, perm permissions.Permission, checks ...func(*constraints)) error {
	handlerFunc, ok := v.getPermissionHandler(perm)
	if !ok {
		return authorizationError(ctx, false)
	}

	return handlerFunc(ctx, &perm, getConstraints(checks...))
}

// RequireAccessToInheritableResource will return an error if the caller doesn't have access to the specified resource type
func (v *VCSWorkspaceLinkCaller) RequireAccessToInheritableResource(ctx context.Context, _ permissions.ResourceType, _ ...func(*constraints)) error {
	// Return an authorization error since VCS does not need any access to inherited resources.
	return authorizationError(ctx, false)
}

// requireAccessToWorkspace will return an error if the caller doesn't have permission to view the specified workspace.
func (v *VCSWorkspaceLinkCaller) requireAccessToWorkspace(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.workspaceID == nil {
		return errMissingConstraints
	}

	if v.Link.WorkspaceID == *checks.workspaceID {
		// Allow since workspace on the link is
		// the one being accessed.
		return nil
	}

	// Deny all others.
	return authorizationError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (v *VCSWorkspaceLinkCaller) getPermissionHandler(perm permissions.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[permissions.Permission]permissionTypeHandler{
		permissions.ViewWorkspacePermission:              v.requireAccessToWorkspace,
		permissions.ViewRunPermission:                    v.requireAccessToWorkspace,
		permissions.CreateRunPermission:                  v.requireAccessToWorkspace, // Should only create runs for linked workspace.
		permissions.ViewConfigurationVersionPermission:   v.requireAccessToWorkspace,
		permissions.CreateConfigurationVersionPermission: v.requireAccessToWorkspace,
		permissions.UpdateConfigurationVersionPermission: v.requireAccessToWorkspace,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
