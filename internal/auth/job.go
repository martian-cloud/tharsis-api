package auth

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// JobCaller represents a job subject
type JobCaller struct {
	dbClient    *db.Client
	JobID       string
	WorkspaceID string
	RunID       string
}

// GetSubject returns the subject identifier for this caller
func (j *JobCaller) GetSubject() string {
	return j.JobID
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (j *JobCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces
		RootNamespaceIDs: []string{},
	}, nil
}

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level
func (j *JobCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	if accessLevel != models.ViewerRole {
		return authorizationError(ctx, false)
	}

	workspace, err := j.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, namespacePath)
	// If the namespace isn't a workspace or the workspace doesn't exist return an error
	if err != nil || workspace == nil {
		return authorizationError(ctx, false)
	}

	return j.requireRootNamespaceAccess(ctx, []string{workspace.FullPath})
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group
func (j *JobCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	// Return authorization error since job callers don't have access to groups
	return authorizationError(ctx, false)
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (j *JobCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return authorizationError(ctx, false)
	}

	// Job has access to view all parent group inherited resources
	group, err := j.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return authorizationError(ctx, false)
	}

	if !strings.HasPrefix(workspace.FullPath, group.FullPath+"/") {
		return authorizationError(ctx, false)
	}

	return nil
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (j *JobCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return authorizationError(ctx, false)
	}

	if !strings.HasPrefix(workspace.FullPath, namespace+"/") {
		return authorizationError(ctx, false)
	}

	return nil
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace
func (j *JobCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	if accessLevel != models.ViewerRole {
		return authorizationError(ctx, false)
	}

	if j.WorkspaceID == workspaceID {
		return nil
	}

	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil || workspace == nil {
		return authorizationError(ctx, false)
	}

	// we need to check if they have the same root namespace
	return j.requireRootNamespaceAccess(ctx, []string{workspace.FullPath})
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have viewer access to all the specified groups
func (j *JobCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	// Return authorization error since job callers don't have access to groups
	return authorizationError(ctx, false)
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces
func (j *JobCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	// Verify that all namespaces are workspaces
	for _, path := range namespaces {
		workspace, err := j.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, path)
		// If the namespace isn't a workspace or the workspace doesn't exist return an error
		if err != nil || workspace == nil {
			return authorizationError(ctx, false)
		}
	}

	return j.requireRootNamespaceAccess(ctx, namespaces)
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace
func (j *JobCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	if len(workspaces) == 1 && workspaces[0].Metadata.ID == j.WorkspaceID {
		return nil
	}

	namespacePaths := []string{}
	for _, ws := range workspaces {
		namespacePaths = append(namespacePaths, ws.FullPath)
	}

	return j.requireRootNamespaceAccess(ctx, namespacePaths)
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state
func (j *JobCaller) RequireRunWriteAccess(ctx context.Context, runID string) error {
	if j.RunID != runID {
		return authorizationError(ctx, false)
	}
	return nil
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (j *JobCaller) RequirePlanWriteAccess(ctx context.Context, planID string) error {
	run, err := j.dbClient.Runs.GetRun(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.PlanID != planID {
		return authorizationError(ctx, false)
	}

	// Get latest job associated with plan
	job, err := j.dbClient.Jobs.GetLatestJobByType(ctx, j.RunID, models.JobPlanType)
	if err != nil {
		return err
	}

	if job == nil || job.Metadata.ID != j.JobID {
		return authorizationError(ctx, false)
	}

	return nil
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (j *JobCaller) RequireApplyWriteAccess(ctx context.Context, applyID string) error {
	run, err := j.dbClient.Runs.GetRun(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.ApplyID != applyID {
		return authorizationError(ctx, false)
	}

	// Get latest job associated with plan
	job, err := j.dbClient.Jobs.GetLatestJobByType(ctx, j.RunID, models.JobApplyType)
	if err != nil {
		return err
	}

	if job == nil || job.Metadata.ID != j.JobID {
		return authorizationError(ctx, false)
	}

	return nil
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job
func (j *JobCaller) RequireJobWriteAccess(ctx context.Context, jobID string) error {
	if j.JobID != jobID {
		return authorizationError(ctx, false)
	}
	return nil
}

func (j *JobCaller) requireRootNamespaceAccess(ctx context.Context, namespacePaths []string) error {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return authorizationError(ctx, false)
	}

	// a workspace must belong under at least one group, so it is safe to assume
	// the first index is the root regardless of levels.
	rootNamespace := strings.Split(workspace.FullPath, "/")[0] + "/"

	for _, namespacePath := range namespacePaths {
		// TODO Advanced controls will need to eventually be added.
		// Currently, a job will have access to anything under the same root namespace
		if !strings.HasPrefix(namespacePath, rootNamespace) {
			return authorizationError(ctx, false)
		}
	}

	return nil
}

// RequireTeamCreateAccess will return an error if the specified access is not allowed to the indicated team.
func (j *JobCaller) RequireTeamCreateAccess(ctx context.Context) error {
	// Job callers won't ever have access to team info.
	return authorizationError(ctx, false)
}

// RequireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (j *JobCaller) RequireTeamUpdateAccess(ctx context.Context, teamID string) error {
	// Job callers won't ever have access to team info.
	return authorizationError(ctx, false)
}

// RequireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
func (j *JobCaller) RequireTeamDeleteAccess(ctx context.Context, teamID string) error {
	// Job callers won't ever have access to team info.
	return authorizationError(ctx, false)
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (j *JobCaller) RequireUserCreateAccess(ctx context.Context) error {
	// Job callers won't ever have access to user info.
	return authorizationError(ctx, false)
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (j *JobCaller) RequireUserUpdateAccess(ctx context.Context, userID string) error {
	// Job callers won't ever have access to user info.
	return authorizationError(ctx, false)
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (j *JobCaller) RequireUserDeleteAccess(ctx context.Context, userID string) error {
	// Job callers won't ever have access to user info.
	return authorizationError(ctx, false)
}
