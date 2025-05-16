package auth

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// VCSWorkspaceLinkCaller represents a VCS provider subject.
type VCSWorkspaceLinkCaller struct {
	Provider           *models.VCSProvider
	Link               *models.WorkspaceVCSProviderLink
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

// NewVCSWorkspaceLinkCaller returns a new VCS caller.
func NewVCSWorkspaceLinkCaller(
	provider *models.VCSProvider,
	link *models.WorkspaceVCSProviderLink,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
) *VCSWorkspaceLinkCaller {
	return &VCSWorkspaceLinkCaller{
		Provider:           provider,
		Link:               link,
		dbClient:           dbClient,
		maintenanceMonitor: maintenanceMonitor,
	}
}

// GetSubject returns the subject identifier for this caller.
func (v *VCSWorkspaceLinkCaller) GetSubject() string {
	return v.Provider.GetResourcePath()
}

// IsAdmin returns true if the caller is an admin.
func (v *VCSWorkspaceLinkCaller) IsAdmin() bool {
	return false
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (v *VCSWorkspaceLinkCaller) UnauthorizedError(_ context.Context, hasViewerAccess bool) error {
	forbiddedMsg := fmt.Sprintf("VCS workspace link %s is not authorized to perform the requested operation", v.GetSubject())

	// If subject has at least viewer permissions then return 403, if not, return 404
	if hasViewerAccess {
		return terrors.New(
			forbiddedMsg,
			terrors.WithErrorCode(terrors.EForbidden),
		)
	}

	return terrors.New(
		"either the requested resource does not exist or the %s",
		forbiddedMsg,
		terrors.WithErrorCode(terrors.ENotFound),
	)
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
func (v *VCSWorkspaceLinkCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	inMaintenance, err := v.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return err
	}

	if inMaintenance && perm.Action != models.ViewAction {
		// Server is in maintenance mode, only allow view permissions
		return errInMaintenanceMode
	}

	handlerFunc, ok := v.getPermissionHandler(perm)
	if !ok {
		return v.UnauthorizedError(ctx, false)
	}

	return handlerFunc(ctx, &perm, getConstraints(checks...))
}

// RequireAccessToInheritableResource will return an error if the caller doesn't have access to the specified resource type
func (v *VCSWorkspaceLinkCaller) RequireAccessToInheritableResource(ctx context.Context, _ types.ModelType, _ ...func(*constraints)) error {
	// Return an authorization error since VCS does not need any access to inherited resources.
	return v.UnauthorizedError(ctx, false)
}

// requireAccessToWorkspace will return an error if the caller doesn't have permission to view the specified workspace.
func (v *VCSWorkspaceLinkCaller) requireAccessToWorkspace(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.workspaceID == nil {
		return errMissingConstraints
	}

	if v.Link.WorkspaceID == *checks.workspaceID {
		// Allow since workspace on the link is
		// the one being accessed.
		return nil
	}

	// Deny all others.
	return v.UnauthorizedError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (v *VCSWorkspaceLinkCaller) getPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.ViewWorkspacePermission:              v.requireAccessToWorkspace,
		models.ViewRunPermission:                    v.requireAccessToWorkspace,
		models.CreateRunPermission:                  v.requireAccessToWorkspace, // Should only create runs for linked workspace.
		models.ViewConfigurationVersionPermission:   v.requireAccessToWorkspace,
		models.CreateConfigurationVersionPermission: v.requireAccessToWorkspace,
		models.UpdateConfigurationVersionPermission: v.requireAccessToWorkspace,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
