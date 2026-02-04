package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// JobCaller represents a job subject
type JobCaller struct {
	dbClient    *db.Client
	JobID       string
	JobTRN      string
	WorkspaceID string
	RunID       string
}

// GetSubject returns the subject identifier for this caller
func (j *JobCaller) GetSubject() string {
	return j.JobTRN
}

// IsAdmin returns true if the caller is an admin
func (j *JobCaller) IsAdmin() bool {
	return false
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (j *JobCaller) UnauthorizedError(ctx context.Context, hasViewerAccess bool) error {
	// Get workspace path
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	workspacePath := "unknown workspace"
	if workspace != nil {
		workspacePath = workspace.FullPath
	}

	forbiddedMsg := fmt.Sprintf(
		"job in workspace %s is not authorized to perform the requested operation: a job only has read access to resources in its group, a Tharsis Managed Identity must be assigned to the workspace to peform write operations.",
		workspacePath,
	)

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

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (j *JobCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		AllowAll: false,
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces
		RootNamespaceIDs: []string{},
	}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (j *JobCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	handlerFunc, ok := j.getPermissionHandler(perm)
	if !ok {
		// Handler not found so we need to check if the job has viewer access to determine which error to return
		c := getConstraints(checks...)

		// If no constraints are provided, we can't determine if the job has viewer access
		if c.workspaceID == nil && c.groupID == nil && len(c.namespacePaths) == 0 {
			return j.UnauthorizedError(ctx, false)
		}

		hasViewerAccess := true

		if c.workspaceID != nil && j.WorkspaceID != *c.workspaceID {
			// Job doesn't have access to the workspace
			hasViewerAccess = false
		}

		if c.groupID != nil {
			if err := j.requireAccessToInheritedGroupResource(ctx, *c.groupID); err != nil {
				// Job doesn't have access to the group
				hasViewerAccess = false
			}
		}
		if len(c.namespacePaths) > 0 {
			if err := j.requireAccessToInheritedNamespaceResource(ctx, c.namespacePaths); err != nil {
				// Job doesn't have access to one of the namespaces
				hasViewerAccess = false
			}
		}

		return j.UnauthorizedError(ctx, hasViewerAccess)
	}

	return handlerFunc(ctx, &perm, getConstraints(checks...))
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (j *JobCaller) RequireAccessToInheritableResource(ctx context.Context, _ types.ModelType, checks ...func(*constraints)) error {
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
		return j.UnauthorizedError(ctx, false)
	}

	// Job has access to view all parent group inherited resources
	group, err := j.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return j.UnauthorizedError(ctx, false)
	}

	if !workspace.IsDescendantOfGroup(group.FullPath) {
		return j.UnauthorizedError(ctx, false)
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
		return j.UnauthorizedError(ctx, false)
	}

	for _, ns := range namespacePaths {
		if !workspace.IsDescendantOfGroup(ns) {
			return j.UnauthorizedError(ctx, false)
		}
	}

	return nil
}

// requireAccessToWorkspacesInGroupHierarchy delegates the appropriate workspace check based on the Constraints.
func (j *JobCaller) requireAccessToWorkspacesInGroupHierarchy(ctx context.Context, _ *models.Permission, checks *constraints) error {
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
		return j.UnauthorizedError(ctx, false)
	}

	// we need to check if they have the same root namespace
	return j.requireRootNamespaceAccess(ctx, []string{workspace.FullPath})
}

// requireAccessToJobWorkspace will return an error if the job caller isn't in the requested workspace
func (j *JobCaller) requireAccessToJobWorkspace(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.workspaceID == nil {
		return errMissingConstraints
	}

	if *checks.workspaceID != j.WorkspaceID {
		return j.UnauthorizedError(ctx, false)
	}

	return nil
}

// requireRunAccess will return an error if the caller doesn't have permission to the run
func (j *JobCaller) requireRunAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.runID == nil && checks.workspaceID == nil {
		return errMissingConstraints
	}

	if checks.runID != nil && j.RunID == *checks.runID {
		// Job belongs to run.
		return nil
	}

	// TODO: revert to previous behavior (only compare workspaceIDs) after SDK has been
	// updated to not query for Run when only the current state version outputs are needed.
	return j.requireAccessToWorkspacesInGroupHierarchy(ctx, nil, checks)
}

// requirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (j *JobCaller) requirePlanWriteAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.planID == nil {
		return errMissingConstraints
	}

	run, err := j.dbClient.Runs.GetRunByID(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.PlanID != *checks.planID {
		return j.UnauthorizedError(ctx, false)
	}

	// Get latest job associated with plan
	job, err := j.dbClient.Jobs.GetLatestJobByType(ctx, j.RunID, models.JobPlanType)
	if err != nil {
		return err
	}

	if job == nil || job.Metadata.ID != j.JobID {
		return j.UnauthorizedError(ctx, false)
	}

	return nil
}

// requireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (j *JobCaller) requireApplyWriteAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.applyID == nil {
		return errMissingConstraints
	}

	run, err := j.dbClient.Runs.GetRunByID(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil || run.ApplyID != *checks.applyID {
		return j.UnauthorizedError(ctx, false)
	}

	// Get latest job associated with plan
	job, err := j.dbClient.Jobs.GetLatestJobByType(ctx, j.RunID, models.JobApplyType)
	if err != nil {
		return err
	}

	if job == nil || job.Metadata.ID != j.JobID {
		return j.UnauthorizedError(ctx, false)
	}

	return nil
}

// requireJobAccess will return an error if the caller doesn't have permission to the specified job
func (j *JobCaller) requireJobAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.jobID == nil {
		return errMissingConstraints
	}

	if j.JobID == *checks.jobID {
		return nil
	}

	return j.UnauthorizedError(ctx, false)
}

func (j *JobCaller) requireRootNamespaceAccess(ctx context.Context, namespacePaths []string) error {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return j.UnauthorizedError(ctx, false)
	}

	// a workspace must belong under at least one group, so it is safe to assume
	// the first index is the root regardless of levels.
	rootNamespace := strings.Split(workspace.FullPath, "/")[0]

	for _, namespacePath := range namespacePaths {
		// TODO Advanced controls will need to eventually be added.
		// Currently, a job will have access to anything under the same root namespace
		if namespacePath != rootNamespace && !strings.HasPrefix(namespacePath, rootNamespace+"/") {
			return j.UnauthorizedError(ctx, false)
		}
	}

	return nil
}

// requireProviderMirrorAccess allows creating provider mirrors if the workspace has provider mirror enabled.
func (j *JobCaller) requireProviderMirrorAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.groupID == nil && len(checks.namespacePaths) == 0 {
		return errMissingConstraints
	}

	job, err := j.dbClient.Jobs.GetJobByID(ctx, j.JobID)
	if err != nil {
		return err
	}

	if job == nil {
		return j.UnauthorizedError(ctx, false)
	}

	// Check if provider mirror is enabled from job properties
	propValue, ok := job.Properties[models.JobPropertyProviderMirrorEnabled]
	if !ok {
		return j.UnauthorizedError(ctx, true)
	}

	enabled, err := strconv.ParseBool(propValue)
	if err != nil {
		return terrors.Wrap(err, "invalid value for job property %s", models.JobPropertyProviderMirrorEnabled)
	}

	if !enabled {
		return j.UnauthorizedError(ctx, true)
	}

	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return j.UnauthorizedError(ctx, false)
	}

	rootGroupPath := workspace.GetRootGroupPath()

	// Check namespace paths if provided
	if len(checks.namespacePaths) > 0 {
		for _, ns := range checks.namespacePaths {
			if ns != rootGroupPath {
				return j.UnauthorizedError(ctx, false)
			}
		}

		return nil
	}

	if checks.groupID == nil {
		return j.UnauthorizedError(ctx, false)
	}

	// Fall back to groupID
	group, err := j.dbClient.Groups.GetGroupByID(ctx, *checks.groupID)
	if err != nil {
		return err
	}

	if group == nil || group.FullPath != rootGroupPath {
		return j.UnauthorizedError(ctx, false)
	}

	return nil
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (j *JobCaller) getPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.ViewWorkspacePermission:                 j.requireAccessToWorkspacesInGroupHierarchy,
		models.ViewConfigurationVersionPermission:      j.requireAccessToWorkspacesInGroupHierarchy,
		models.ViewStateVersionPermission:              j.requireAccessToWorkspacesInGroupHierarchy,
		models.ViewManagedIdentityPermission:           j.requireAccessToWorkspacesInGroupHierarchy,
		models.ViewVariablePermission:                  j.requireAccessToWorkspacesInGroupHierarchy,
		models.ViewStateVersionDataPermission:          j.requireAccessToJobWorkspace,
		models.CreateStateVersionPermission:            j.requireAccessToJobWorkspace,
		models.ViewVariableValuePermission:             j.requireAccessToJobWorkspace,
		models.ViewRunPermission:                       j.requireRunAccess, // View is automatically granted if action != View.
		models.ViewJobPermission:                       j.requireJobAccess, // View is automatically granted if action != View.
		models.UpdateJobPermission:                     j.requireJobAccess,
		models.IssueFederatedRegistryTokenPermission:   j.requireJobAccess,
		models.UpdatePlanPermission:                    j.requirePlanWriteAccess,
		models.UpdateApplyPermission:                   j.requireApplyWriteAccess,
		models.CreateTerraformProviderMirrorPermission: j.requireProviderMirrorAccess,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
