package auth

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
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

// IsAdmin returns true if the caller is an admin
func (j *JobCaller) IsAdmin() bool {
	return false
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (j *JobCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces
		RootNamespaceIDs: []string{},
	}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (j *JobCaller) RequirePermission(ctx context.Context, perm permissions.Permission, checks ...func(*constraints)) error {
	handlerFunc, ok := j.getPermissionHandler(perm)
	if !ok {
		return authorizationError(ctx, false)
	}

	return handlerFunc(ctx, &perm, getConstraints(checks...))
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (j *JobCaller) RequireAccessToInheritableResource(ctx context.Context, _ permissions.ResourceType, checks ...func(*constraints)) error {
	c := getConstraints(checks...)
	if c.groupID != nil {
		return j.requireAccessToInheritedGroupResource(ctx, *c.groupID)
	}
	if len(c.namespacePaths) > 0 {
		return j.requireAccessToInheritedNamespaceResource(ctx, c.namespacePaths)
	}

	return errMissingConstraints
}

// requireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (j *JobCaller) requireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
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

	if !workspace.IsDescendantOfGroup(group.FullPath) {
		return authorizationError(ctx, false)
	}

	return nil
}

// requireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (j *JobCaller) requireAccessToInheritedNamespaceResource(ctx context.Context, namespacePaths []string) error {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return authorizationError(ctx, false)
	}

	for _, ns := range namespacePaths {
		if !workspace.IsDescendantOfGroup(ns) {
			return authorizationError(ctx, false)
		}
	}

	return nil
}

// requireAccessToWorkspaces delegates the appropriate workspace check based on the Constraints.
func (j *JobCaller) requireAccessToWorkspaces(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.workspaceID != nil {
		return j.requireAccessToWorkspace(ctx, *checks.workspaceID)
	}

	if len(checks.namespacePaths) > 0 {
		return j.requireRootNamespaceAccess(ctx, checks.namespacePaths)
	}

	return errMissingConstraints
}

// requireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace
func (j *JobCaller) requireAccessToWorkspace(ctx context.Context, workspaceID string) error {
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

// requireRunAccess will return an error if the caller doesn't have permission to the run
func (j *JobCaller) requireRunAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.runID == nil && checks.workspaceID == nil {
		return errMissingConstraints
	}

	if checks.runID != nil && j.RunID == *checks.runID {
		// Job belongs to run.
		return nil
	}

	// TODO: revert to previous behavior (only compare workspaceIDs) after SDK has been
	// updated to not query for Run when only the current state version outputs are needed.
	return j.requireAccessToWorkspaces(ctx, nil, checks)
}

// requirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (j *JobCaller) requirePlanWriteAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.planID == nil {
		return errMissingConstraints
	}

	run, err := j.dbClient.Runs.GetRun(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.PlanID != *checks.planID {
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

// requireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (j *JobCaller) requireApplyWriteAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.applyID == nil {
		return errMissingConstraints
	}

	run, err := j.dbClient.Runs.GetRun(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.ApplyID != *checks.applyID {
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

// requireJobAccess will return an error if the caller doesn't have permission to the specified job
func (j *JobCaller) requireJobAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
	if checks.jobID == nil {
		return errMissingConstraints
	}

	if j.JobID == *checks.jobID {
		return nil
	}

	return authorizationError(ctx, false)
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

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (j *JobCaller) getPermissionHandler(perm permissions.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[permissions.Permission]permissionTypeHandler{
		permissions.ViewWorkspacePermission:            j.requireAccessToWorkspaces,
		permissions.ViewConfigurationVersionPermission: j.requireAccessToWorkspaces,
		permissions.ViewStateVersionPermission:         j.requireAccessToWorkspaces,
		permissions.CreateStateVersionPermission:       j.requireAccessToWorkspaces,
		permissions.ViewManagedIdentityPermission:      j.requireAccessToWorkspaces,
		permissions.ViewVariablePermission:             j.requireAccessToWorkspaces,
		permissions.ViewVariableValuePermission:        j.requireAccessToWorkspaces,
		permissions.ViewRunPermission:                  j.requireRunAccess, // View is automatically granted if action != View.
		permissions.ViewJobPermission:                  j.requireJobAccess, // View is automatically granted if action != View.
		permissions.UpdateJobPermission:                j.requireJobAccess,
		permissions.UpdatePlanPermission:               j.requirePlanWriteAccess,
		permissions.UpdateApplyPermission:              j.requireApplyWriteAccess,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
