package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// ServiceAccountCaller represents a service account subject
type ServiceAccountCaller struct {
	authorizer         Authorizer
	ServiceAccountPath string
	ServiceAccountID   string
	dbClient           *db.Client
}

// NewServiceAccountCaller returns a new ServiceAccountCaller
func NewServiceAccountCaller(id string, path string, authorizer Authorizer, dbClient *db.Client) *ServiceAccountCaller {
	return &ServiceAccountCaller{ServiceAccountID: id, ServiceAccountPath: path, authorizer: authorizer, dbClient: dbClient}
}

// GetSubject returns the subject identifier for this caller
func (s *ServiceAccountCaller) GetSubject() string {
	return s.ServiceAccountPath
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

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level
func (s *ServiceAccountCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	return s.authorizer.RequireAccessToNamespace(ctx, namespacePath, accessLevel)
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group
func (s *ServiceAccountCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	return s.authorizer.RequireAccessToGroup(ctx, groupID, accessLevel)
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace
func (s *ServiceAccountCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	return s.authorizer.RequireAccessToWorkspace(ctx, workspaceID, accessLevel)
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have viewer access to all the specified groups
func (s *ServiceAccountCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	return s.authorizer.RequireViewerAccessToGroups(ctx, groups)
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace
func (s *ServiceAccountCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	return s.authorizer.RequireViewerAccessToWorkspaces(ctx, workspaces)
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces
func (s *ServiceAccountCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	return s.authorizer.RequireViewerAccessToNamespaces(ctx, namespaces)
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (s *ServiceAccountCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	return s.authorizer.RequireAccessToInheritedGroupResource(ctx, groupID)
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (s *ServiceAccountCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	return s.authorizer.RequireAccessToInheritedNamespaceResource(ctx, namespace)
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state
func (s *ServiceAccountCaller) RequireRunWriteAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't have run write access
	return authorizationError(ctx, false)
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (s *ServiceAccountCaller) RequirePlanWriteAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't have plan write access
	return authorizationError(ctx, false)
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (s *ServiceAccountCaller) RequireApplyWriteAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't have apply write access
	return authorizationError(ctx, false)
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job
func (s *ServiceAccountCaller) RequireJobWriteAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't have job write access
	return authorizationError(ctx, false)
}

// RequireTeamCreateAccess will return an error if the specified access is not allowed to the indicated team.
// Currently, this method makes some simplifying assumptions that will need to change once orgs are implemented.
func (s *ServiceAccountCaller) RequireTeamCreateAccess(ctx context.Context) error {
	return authorizationError(ctx, true)
}

// RequireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
// Currently, this method makes some simplifying assumptions that will need to change once orgs are implemented.
func (s *ServiceAccountCaller) RequireTeamUpdateAccess(ctx context.Context, _ string) error {
	return authorizationError(ctx, true)
}

// RequireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
// Currently, this method makes some simplifying assumptions that will need to change once orgs are implemented.
func (s *ServiceAccountCaller) RequireTeamDeleteAccess(ctx context.Context, _ string) error {
	return authorizationError(ctx, true)
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (s *ServiceAccountCaller) RequireUserCreateAccess(ctx context.Context) error {
	// Return authorization error because services accounts don't need to modify users.
	return authorizationError(ctx, false)
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (s *ServiceAccountCaller) RequireUserUpdateAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't need to modify users.
	return authorizationError(ctx, false)
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (s *ServiceAccountCaller) RequireUserDeleteAccess(ctx context.Context, _ string) error {
	// Return authorization error because services accounts don't need to modify users.
	return authorizationError(ctx, false)
}

// RequireRunnerAccess will return an error if the caller is not allowed to claim a job as the specified runner
func (s *ServiceAccountCaller) RequireRunnerAccess(ctx context.Context, runnerID string) error {
	// Verify that service account is assigned to runner
	resp, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			RunnerID:          &runnerID,
			ServiceAccountIDs: []string{s.ServiceAccountID},
		},
	})
	if err != nil {
		return err
	}

	if len(resp.ServiceAccounts) > 0 {
		return nil
	}

	return authorizationError(ctx, true)
}
