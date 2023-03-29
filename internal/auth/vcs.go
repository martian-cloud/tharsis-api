package auth

import (
	"context"

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
func (v *VCSWorkspaceLinkCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces.
		RootNamespaceIDs: []string{},
	}, nil
}

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level.
func (v *VCSWorkspaceLinkCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	// VCSWorkspaceLinkCaller will only require access to workspaces.
	ws, err := v.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, namespacePath)
	if err != nil {
		return err
	}

	if ws != nil && v.Link.WorkspaceID == ws.Metadata.ID {
		// Allow since workspace being accessed is one
		// the link belongs to.
		return nil
	}

	// Deny access to all other namespaces.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have the required access level on the specified group.
func (v *VCSWorkspaceLinkCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	// Return an authorization error since VCS does not need any access to groups.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace.
func (v *VCSWorkspaceLinkCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	// Return an authorization error since VCS does not need any access to workspaces.
	return authorizationError(ctx, false)
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces.
func (v *VCSWorkspaceLinkCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	// Return an authorization error since VCS does not need any access to namespaces.
	return authorizationError(ctx, false)
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group.
func (v *VCSWorkspaceLinkCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	// Return an authorization error since VCS does not need any access to groups.
	return authorizationError(ctx, false)
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace.
func (v *VCSWorkspaceLinkCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	if v.Link.WorkspaceID == workspaceID {
		// Allow since workspace on the link is
		// the one being accessed.
		return nil
	}

	// Deny all others.
	return authorizationError(ctx, false)
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy.
func (v *VCSWorkspaceLinkCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	// Return an authorization error since VCS does not need any access to inherited group resources.
	return authorizationError(ctx, false)
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy.
func (v *VCSWorkspaceLinkCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	// Return an authorization error since VCS does not need any access to namespace resources.
	return authorizationError(ctx, false)
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state.
func (v *VCSWorkspaceLinkCaller) RequireRunWriteAccess(ctx context.Context, runID string) error {
	// Return an authorization error since VCS does not need any access to runs.
	return authorizationError(ctx, false)
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state.
func (v *VCSWorkspaceLinkCaller) RequirePlanWriteAccess(ctx context.Context, planID string) error {
	// Return an authorization error since VCS does not need any access to plans.
	return authorizationError(ctx, false)
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state.
func (v *VCSWorkspaceLinkCaller) RequireApplyWriteAccess(ctx context.Context, applyID string) error {
	// Return an authorization error since VCS does not need any access to plans.
	return authorizationError(ctx, false)
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job.
func (v *VCSWorkspaceLinkCaller) RequireJobWriteAccess(ctx context.Context, jobID string) error {
	// Return an authorization error since VCS does not need any access to jobs.
	return authorizationError(ctx, false)
}

// RequireTeamCreateAccess will return an error if the specified access is not allowed to the indicated team.
func (v *VCSWorkspaceLinkCaller) RequireTeamCreateAccess(ctx context.Context) error {
	// Return an authorization error since VCS does not need any access to create new teams.
	return authorizationError(ctx, false)
}

// RequireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (v *VCSWorkspaceLinkCaller) RequireTeamUpdateAccess(ctx context.Context, teamID string) error {
	// Return an authorization error since VCS does not need any access to update teams.
	return authorizationError(ctx, false)
}

// RequireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
func (v *VCSWorkspaceLinkCaller) RequireTeamDeleteAccess(ctx context.Context, teamID string) error {
	// Return an authorization error since VCS does not need any access to delete teams.
	return authorizationError(ctx, false)
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (v *VCSWorkspaceLinkCaller) RequireUserCreateAccess(ctx context.Context) error {
	// Return an authorization error since VCS does not need any access to create new users.
	return authorizationError(ctx, false)
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (v *VCSWorkspaceLinkCaller) RequireUserUpdateAccess(ctx context.Context, userID string) error {
	// Return an authorization error since VCS does not need any access to update users.
	return authorizationError(ctx, false)
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (v *VCSWorkspaceLinkCaller) RequireUserDeleteAccess(ctx context.Context, userID string) error {
	// Return an authorization error since VCS does not need any access to delete users.
	return authorizationError(ctx, false)
}

// RequireRunnerAccess will return an error if the caller is not allowed to claim a job as the specified runner
func (v *VCSWorkspaceLinkCaller) RequireRunnerAccess(ctx context.Context, runnerID string) error {
	// Return authorization error because vcs callers don't have runner access
	return authorizationError(ctx, false)
}
