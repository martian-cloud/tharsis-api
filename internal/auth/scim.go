package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SCIMCaller represents a SCIM subject.
type SCIMCaller struct {
	idpIssuerURL       string
	maintenanceMonitor maintenance.Monitor
}

// NewSCIMCaller returns a new SCIM caller.
func NewSCIMCaller(
	maintenanceMonitor maintenance.Monitor,
	idpIssuerURL string,
) *SCIMCaller {
	return &SCIMCaller{
		maintenanceMonitor: maintenanceMonitor,
		idpIssuerURL:       idpIssuerURL,
	}
}

// GetSubject returns the subject identifier for this caller.
func (s *SCIMCaller) GetSubject() string {
	return "scim"
}

// GetActor returns the actor for this caller.
func (s *SCIMCaller) GetActor() models.Actor {
	return models.SCIMActor()
}

// GetIDPIssuerURL returns the IDP issuer URL
func (s *SCIMCaller) GetIDPIssuerURL() string {
	return s.idpIssuerURL
}

// Manages returns true if a user or team with the given SCIM IDP issuer URL is managed by this caller's IDP.
// Users and teams not managed by SCIM have no issuer, so they never match.
func (s *SCIMCaller) Manages(scimIDPIssuerURL string) bool {
	return s.idpIssuerURL != "" && scimIDPIssuerURL == s.idpIssuerURL
}

// IsAdminModeActivated returns true if the caller is an admin.
func (s *SCIMCaller) IsAdminModeActivated(_ context.Context) bool {
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

// GetNamespacePermissions returns an empty set. A SCIM caller is not backed by namespace
// memberships and has no permission set to report.
func (s *SCIMCaller) GetNamespacePermissions(_ context.Context, _ string) ([]*models.Permission, error) {
	return []*models.Permission{}, nil
}

// GetRootNamespaceMemberships returns a non-nil empty slice; a SCIM caller has no root namespace
// memberships. It must be non-nil so the membership filter is applied and denies access — a nil
// slice would be treated as "no filter".
func (s *SCIMCaller) GetRootNamespaceMemberships(_ context.Context) ([]models.MembershipNamespace, error) {
	return []models.MembershipNamespace{}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions.
func (s *SCIMCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return err
	}

	if inMaintenance && perm.Action != models.ViewAction {
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
func (s *SCIMCaller) RequireAccessToInheritableResource(ctx context.Context, _ types.ModelType, _ ...func(*constraints)) error {
	// Return an authorization error since SCIM does not need any access to inherited resources.
	return s.UnauthorizedError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (s *SCIMCaller) getPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	// Only the SCIM-specific permissions are granted. The generic team and user permissions are
	// checked by the team and user services, which are reachable over GraphQL and gRPC and don't
	// limit a SCIM caller to its own IDP's resources, so granting them would let a SCIM token
	// create, modify or delete any team. The SCIM service checks that the caller's IDP manages each
	// user or team it acts on.
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.ManageSCIMTeamPermission: noopPermissionHandler,
		models.ManageSCIMUserPermission: noopPermissionHandler,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
