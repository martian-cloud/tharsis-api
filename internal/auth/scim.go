package auth

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// SCIMCaller represents a SCIM subject.
type SCIMCaller struct {
	dbClient *db.Client
}

// NewSCIMCaller returns a new SCIM caller.
func NewSCIMCaller(dbClient *db.Client) *SCIMCaller {
	return &SCIMCaller{dbClient}
}

// GetSubject returns the subject identifier for this caller.
func (s *SCIMCaller) GetSubject() string {
	return "scim"
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller.
func (s *SCIMCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces.
		RootNamespaceIDs: []string{},
	}, nil
}

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level.
func (s *SCIMCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	// Return an authorization error since SCIM does not need access to namespaces.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have the required access level on the specified group.
func (s *SCIMCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	// Return an authorization error since SCIM does not need any access to groups.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace.
func (s *SCIMCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	// Return an authorization error since SCIM does not need any access to workspaces.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces.
func (s *SCIMCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	// Return an authorization error since SCIM does not need any access to namespaces.
	return authorizationError(ctx, false)
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group.
func (s *SCIMCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	// Return an authorization error since SCIM does not need any access to groups.
	return authorizationError(ctx, false)
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace.
func (s *SCIMCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	// Return an authorization error since SCIM does not need any access to workspaces.
	return authorizationError(ctx, false)
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy.
func (s *SCIMCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	// Return an authorization error since SCIM does not need any access to group resources.
	return authorizationError(ctx, false)
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy.
func (s *SCIMCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	// Return an authorization error since SCIM does not need any access to namespace resources.
	return authorizationError(ctx, false)
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state.
func (s *SCIMCaller) RequireRunWriteAccess(ctx context.Context, runID string) error {
	// Return an authorization error since SCIM does not need any access to runs.
	return authorizationError(ctx, false)
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state.
func (s *SCIMCaller) RequirePlanWriteAccess(ctx context.Context, planID string) error {
	// Return an authorization error since SCIM does not need any access to plans.
	return authorizationError(ctx, false)
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state.
func (s *SCIMCaller) RequireApplyWriteAccess(ctx context.Context, applyID string) error {
	// Return an authorization error since SCIM does not need any access to plans.
	return authorizationError(ctx, false)
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job.
func (s *SCIMCaller) RequireJobWriteAccess(ctx context.Context, jobID string) error {
	// Return an authorization error since SCIM does not need any access to jobs.
	return authorizationError(ctx, false)
}

// RequireTeamCreateAccess will return an error if the specified access is not allowed to the indicated team.
func (s *SCIMCaller) RequireTeamCreateAccess(ctx context.Context) error {
	// SCIM is allowed to create new teams.
	return nil
}

// RequireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (s *SCIMCaller) RequireTeamUpdateAccess(ctx context.Context, teamID string) error {
	// SCIM is allowed to update teams.
	return nil
}

// RequireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
func (s *SCIMCaller) RequireTeamDeleteAccess(ctx context.Context, teamID string) error {
	team, err := s.dbClient.Teams.GetTeamByID(ctx, teamID)
	if err != nil {
		return err
	}

	// Only allow deleting teams which are created via SCIM.
	if team != nil && team.SCIMExternalID != "" {
		return nil
	}

	return authorizationError(ctx, false)
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (s *SCIMCaller) RequireUserCreateAccess(ctx context.Context) error {
	// SCIM caller is allowed to create new users.
	return nil
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (s *SCIMCaller) RequireUserUpdateAccess(ctx context.Context, userID string) error {
	// SCIM caller is allowed to update users.
	return nil
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (s *SCIMCaller) RequireUserDeleteAccess(ctx context.Context, userID string) error {
	user, err := s.dbClient.Users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Only allow deleting users created via SCIM.
	if user != nil && user.SCIMExternalID != "" {
		return nil
	}

	return authorizationError(ctx, false)
}

// RequireRunnerAccess will return an error if the caller is not allowed to claim a job as the specified runner
func (s *SCIMCaller) RequireRunnerAccess(ctx context.Context, runnerID string) error {
	// Return authorization error because SCIM callers don't have runner access
	return authorizationError(ctx, false)
}
