package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ServiceAccountCaller represents a service account subject
type ServiceAccountCaller struct {
	authorizer         Authorizer
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
	ServiceAccountPath string
	ServiceAccountID   string
}

// NewServiceAccountCaller returns a new ServiceAccountCaller
func NewServiceAccountCaller(
	id,
	path string,
	authorizer Authorizer,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
) *ServiceAccountCaller {
	return &ServiceAccountCaller{
		ServiceAccountID:   id,
		ServiceAccountPath: path,
		authorizer:         authorizer,
		dbClient:           dbClient,
		maintenanceMonitor: maintenanceMonitor,
	}
}

// GetSubject returns the subject identifier for this caller
func (s *ServiceAccountCaller) GetSubject() string {
	return s.ServiceAccountPath
}

// IsAdmin returns true if the caller is an admin
func (s *ServiceAccountCaller) IsAdmin() bool {
	return false
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (s *ServiceAccountCaller) UnauthorizedError(_ context.Context, hasViewerAccess bool) error {
	// If subject has at least viewer permissions then return 403, if not, return 404
	if hasViewerAccess {
		return terrors.New(
			"service account %s is not authorized to perform the requested operation: ensure that the service account has been added as a member to the group/workspace with the role required to perform the requested operation",
			s.GetSubject(),
			terrors.WithErrorCode(terrors.EForbidden),
		)
	}

	return terrors.New(
		"either the requested resource does not exist or the service account %s is not authorized to perform the requested operation: ensure that the service account has been added as a member to the group/workspace with the role required to perform the requested operation",
		s.GetSubject(),
		terrors.WithErrorCode(terrors.ENotFound),
	)
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (s *ServiceAccountCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	rootNamespaces, err := s.authorizer.GetRootNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, ns := range rootNamespaces {
		ids = append(ids, ns.ID)
	}

	return &NamespaceAccessPolicy{AllowAll: false, RootNamespaceIDs: ids}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (s *ServiceAccountCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return err
	}

	if inMaintenance && perm.Action != models.ViewAction && perm.Action != models.ViewValueAction {
		// Server is in maintenance mode, only allow view permissions
		return errInMaintenanceMode
	}

	if handlerFunc, ok := s.getPermissionHandler(perm); ok {
		return handlerFunc(ctx, &perm, getConstraints(checks...))
	}

	return s.authorizer.RequireAccess(ctx, []models.Permission{perm}, checks...)
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (s *ServiceAccountCaller) RequireAccessToInheritableResource(ctx context.Context, modelType types.ModelType, checks ...func(*constraints)) error {

	// If the check is for a runner resource,
	// and if the service account is assigned to the runner,
	// that needs to count as the service account having viewer permission for the runner.
	if modelType.Equals(types.RunnerModelType) {
		if c := getConstraints(checks...); c.runnerID != nil {
			return s.requireRunnerAccess(ctx, &models.ViewRunnerPermission, c)
		}
	}

	return s.authorizer.RequireAccessToInheritableResource(ctx, []types.ModelType{modelType}, checks...)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (s *ServiceAccountCaller) getPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.ClaimJobPermission:            s.requireRunnerAccess,
		models.CreateRunnerSessionPermission: s.requireRunnerAccess,
		models.UpdateRunnerSessionPermission: s.requireRunnerAccess,
	}

	handler, ok := handlerMap[perm]
	return handler, ok
}

// requireRunnerAccess verifies that the service account is assigned to the specified runner
func (s *ServiceAccountCaller) requireRunnerAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.runnerID == nil {
		return errMissingConstraints
	}

	// Verify that service account is assigned to runner
	resp, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			RunnerID:          checks.runnerID,
			ServiceAccountIDs: []string{s.ServiceAccountID},
		},
	})
	if err != nil {
		return err
	}

	if len(resp.ServiceAccounts) > 0 {
		return nil
	}

	return s.UnauthorizedError(ctx, true)
}
