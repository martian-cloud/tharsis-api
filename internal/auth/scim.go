package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SCIMCaller represents a SCIM subject.
type SCIMCaller struct {
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

// NewSCIMCaller returns a new SCIM caller.
func NewSCIMCaller(dbClient *db.Client, maintenanceMonitor maintenance.Monitor) *SCIMCaller {
	return &SCIMCaller{dbClient, maintenanceMonitor}
}

// GetSubject returns the subject identifier for this caller.
func (s *SCIMCaller) GetSubject() string {
	return "scim"
}

// IsAdmin returns true if the caller is an admin.
func (s *SCIMCaller) IsAdmin() bool {
	return false
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (s *SCIMCaller) UnauthorizedError(_ context.Context, hasViewerAccess bool) error {
	forbiddedMsg := "SCIM caller is not authorized to perform the requested operation"

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
func (s *SCIMCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces.
		RootNamespaceIDs: []string{},
	}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions.
func (s *SCIMCaller) RequirePermission(ctx context.Context, perm permissions.Permission, checks ...func(*constraints)) error {
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return err
	}

	if inMaintenance && perm.Action != permissions.ViewAction {
		// Server is in maintenance mode, only allow view permissions
		return errInMaintenanceMode
	}

	handlerFunc, ok := s.getPermissionHandler(perm)
	if !ok {
		return s.UnauthorizedError(ctx, false)
	}

	return handlerFunc(ctx, &perm, getConstraints(checks...))
}

// RequireAccessToInheritableResource will return an error if the caller doesn't have access to the specified resource type.
func (s *SCIMCaller) RequireAccessToInheritableResource(ctx context.Context, _ permissions.ResourceType, _ ...func(*constraints)) error {
	// Return an authorization error since SCIM does not need any access to inherited resources.
	return s.UnauthorizedError(ctx, false)
}

// requireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
func (s *SCIMCaller) requireTeamDeleteAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.teamID == nil {
		return errMissingConstraints
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, *checks.teamID)
	if err != nil {
		return err
	}

	// Only allow deleting teams which are created via SCIM.
	if team != nil && team.SCIMExternalID != "" {
		return nil
	}

	return s.UnauthorizedError(ctx, false)
}

// requireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (s *SCIMCaller) requireUserDeleteAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.userID == nil {
		return errMissingConstraints
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, *checks.userID)
	if err != nil {
		return err
	}

	// Only allow deleting users created via SCIM.
	if user != nil && user.SCIMExternalID != "" {
		return nil
	}

	return s.UnauthorizedError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (s *SCIMCaller) getPermissionHandler(perm permissions.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[permissions.Permission]permissionTypeHandler{
		permissions.DeleteTeamPermission: s.requireTeamDeleteAccess,
		permissions.DeleteUserPermission: s.requireUserDeleteAccess,
		permissions.CreateTeamPermission: noopPermissionHandler,
		permissions.UpdateTeamPermission: noopPermissionHandler,
		permissions.CreateUserPermission: noopPermissionHandler,
		permissions.UpdateUserPermission: noopPermissionHandler,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
