package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// JobCaller represents a job subject
type JobCaller struct {
	dbClient                 *db.Client
	inheritedSettingResolver namespace.InheritedSettingResolver
	JobID                    string
	JobTRN                   string
	WorkspaceID              string
	RunID                    string
}

// GetSubject returns the subject identifier for this caller
func (j *JobCaller) GetSubject() string {
	return j.JobTRN
}

// IsAdminModeActivated returns true if the caller is an admin
func (j *JobCaller) IsAdminModeActivated(_ context.Context) bool {
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
		"job in workspace %s is not authorized to perform the requested operation: a job only has read access to resources in its group, a workspace role binding must be created to perform write operations.",
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

// GetNamespacePermissions returns the permissions granted by the job's workspace's role binding if it has one
func (j *JobCaller) GetNamespacePermissions(ctx context.Context, namespacePath string) ([]*models.Permission, error) {
	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if workspace == nil {
		return []*models.Permission{}, nil
	}

	parentGroupPath := workspace.GetGroupPath()

	if namespacePath != parentGroupPath && !utils.IsDescendantOfPath(namespacePath, parentGroupPath) {
		return []*models.Permission{}, nil
	}

	binding, err := j.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByWorkspaceID(ctx, j.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if binding == nil {
		return []*models.Permission{}, nil
	}

	role, err := j.dbClient.Roles.GetRoleByID(ctx, binding.RoleID)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return []*models.Permission{}, nil
	}

	rolePerms := role.GetPermissions()
	perms := make([]*models.Permission, 0, len(rolePerms))
	for _, p := range rolePerms {
		permCopy := p
		perms = append(perms, &permCopy)
	}

	return perms, nil
}

// GetRootNamespaceMemberships returns a non-nil empty slice; a job caller has no root namespace
// memberships
func (j *JobCaller) GetRootNamespaceMemberships(_ context.Context) ([]models.MembershipNamespace, error) {
	return []models.MembershipNamespace{}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (j *JobCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	// First check non-assignable permissions which take precedence
	if handlerFunc, ok := j.getCoreJobPermissionHandler(perm); ok {
		return handlerFunc(ctx, &perm, getConstraints(checks...))
	}

	c := getConstraints(checks...)

	if handlerFunc, ok := j.getSupersedableJobPermissionHandler(perm); ok {
		if handlerErr := handlerFunc(ctx, &perm, c); handlerErr == nil {
			return nil
		}

		if boundErr := j.requireBoundRoleAccess(ctx, &perm, c); boundErr == nil {
			return nil
		}

		return j.viewerAccessDenial(ctx, c)
	}

	// No handler at all for this permission. Try the bound-role fallthrough before falling all the
	// way through to a flat deny.
	if boundErr := j.requireBoundRoleAccess(ctx, &perm, c); boundErr == nil {
		return nil
	}

	return j.viewerAccessDenial(ctx, c)
}

// viewerAccessDenial builds the final denial for a permission that neither a job-mechanics handler
// nor a bound role satisfied, choosing EForbidden vs ENotFound based on whether the job has at least
// viewer access to the constrained resource.
func (j *JobCaller) viewerAccessDenial(ctx context.Context, c *constraints) error {
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

// requireWorkspaceOutputVisibility enforces the output visibility setting when a job attempts
// to view another workspace or its state. ViewWorkspacePermission is included because the
// Terraform provider resolves the workspace (GetWorkspaceByTRN) before reading its outputs.
func (j *JobCaller) requireWorkspaceOutputVisibility(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.workspaceID != nil {
		if j.WorkspaceID == *checks.workspaceID {
			return nil // self-access is always allowed
		}
		workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, *checks.workspaceID)
		if err != nil {
			return err
		}
		if workspace == nil {
			return j.UnauthorizedError(ctx, false)
		}
		return j.checkOutputVisibility(ctx, workspace)
	}

	if len(checks.namespacePaths) > 0 {
		for _, nsPath := range checks.namespacePaths {
			ws, err := j.dbClient.Workspaces.GetWorkspaceByTRN(ctx, trn.TypeWorkspace.Build(nsPath))
			if err != nil {
				return err
			}
			if ws == nil {
				return j.UnauthorizedError(ctx, false)
			}
			if j.WorkspaceID == ws.Metadata.ID {
				continue // self-access is always allowed
			}
			if err := j.checkOutputVisibility(ctx, ws); err != nil {
				return err
			}
		}
		return nil
	}

	return errMissingConstraints
}

// requireAccessToWorkspacesInGroupHierarchy checks that the job has access to a workspace
// within the same root namespace (group hierarchy).
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
	if checks.workspaceID != nil {
		if *checks.workspaceID != j.WorkspaceID {
			return j.UnauthorizedError(ctx, false)
		}
		return nil
	}

	if len(checks.namespacePaths) > 0 {
		workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
		if err != nil {
			return err
		}

		if workspace == nil {
			return j.UnauthorizedError(ctx, false)
		}

		for _, namespacePath := range checks.namespacePaths {
			// Must match exactly, unlike requireAccessToWorkspacesInGroupHierarchy.
			if namespacePath != workspace.FullPath {
				return j.UnauthorizedError(ctx, false)
			}
		}

		return nil
	}

	return errMissingConstraints
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

// requireRunWriteAccess authorizes a run's job to update one of the run's nodes — its plan,
// apply, or policy check — selected by which resource ID constraint is set. The caller's job must
// belong to the run identified by the run ID constraint, the referenced resource must exist on
// that run, and the run's latest job of the matching type must be this caller's job.
func (j *JobCaller) requireRunWriteAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.runID == nil {
		return errMissingConstraints
	}

	if j.RunID != *checks.runID {
		return j.UnauthorizedError(ctx, false)
	}

	run, err := j.dbClient.Runs.GetRunByID(ctx, j.RunID)
	if err != nil {
		return err
	}

	if run == nil {
		return j.UnauthorizedError(ctx, false)
	}

	// Each updatable run node records the ID of the job currently driving it. The caller is
	// authorized only if the referenced node exists on the run and the caller owns that node's
	// latest job.
	var latestJobID *string
	switch {
	case checks.planID != nil:
		// run.Plan is a value (always present); a mismatched ID fails closed.
		if run.Plan.GetID() != *checks.planID {
			return j.UnauthorizedError(ctx, false)
		}
		latestJobID = run.Plan.LatestJobID
	case checks.applyID != nil:
		if run.Apply == nil || run.Apply.GetID() != *checks.applyID {
			return j.UnauthorizedError(ctx, false)
		}
		latestJobID = run.Apply.LatestJobID
	case checks.policyCheckID != nil:
		check := run.PolicyCheckByID(*checks.policyCheckID)
		if check == nil {
			return j.UnauthorizedError(ctx, false)
		}
		latestJobID = check.LatestJobID
	default:
		return errMissingConstraints
	}

	if latestJobID == nil || *latestJobID != j.JobID {
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

// checkOutputVisibility checks if the job's workspace is allowed to read outputs from the target workspace
// based on the target workspace's output visibility setting.
func (j *JobCaller) checkOutputVisibility(ctx context.Context, targetWorkspace *models.Workspace) error {
	setting, err := j.inheritedSettingResolver.GetOutputVisibility(ctx, targetWorkspace)
	if err != nil {
		return err
	}

	switch setting.Value {
	case models.OutputVisibilityGlobal:
		return nil
	case models.OutputVisibilityBlockAccess:
		return j.UnauthorizedError(ctx, false)
	case models.OutputVisibilityRootGroup,
		models.OutputVisibilityDirectGroupOnly,
		models.OutputVisibilityDirectGroupAndSubgroups:
		// These cases require loading the requesting workspace
		requestingWS, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
		if err != nil {
			return err
		}
		if requestingWS == nil {
			return j.UnauthorizedError(ctx, false)
		}

		switch setting.Value {
		case models.OutputVisibilityRootGroup:
			// Same root group
			if requestingWS.GetRootGroupPath() != targetWorkspace.GetRootGroupPath() {
				return j.UnauthorizedError(ctx, false)
			}
		case models.OutputVisibilityDirectGroupOnly:
			// Same immediate parent group
			if requestingWS.GroupID != targetWorkspace.GroupID {
				return j.UnauthorizedError(ctx, false)
			}
		case models.OutputVisibilityDirectGroupAndSubgroups:
			// Same group or any descendant subgroup
			targetGroupPath := targetWorkspace.GetGroupPath()
			requestingGroupPath := requestingWS.GetGroupPath()
			if requestingGroupPath != targetGroupPath && !utils.IsDescendantOfPath(requestingGroupPath, targetGroupPath) {
				return j.UnauthorizedError(ctx, false)
			}
		}
		return nil
	default:
		// Unknown value — deny by default
		return j.UnauthorizedError(ctx, false)
	}
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

// requireBoundRoleAccess checks whether the job's workspace has a WorkspaceRoleBinding whose role
// covers perm at the target namespace named by checks. RequirePermission consults it in two cases:
// as the sole check for a permission with no job-mechanics handler at all, and as a fallback after a
// job-mechanics handler for an assignable permission denies (see
// getSupersedableJobPermissionHandler) — a bound role's authority can supersede that narrower
// check. It is never consulted for a permission handled by getCoreJobPermissionHandler; those
// permissions are not assignable to a role and their handler's result is always final.
func (j *JobCaller) requireBoundRoleAccess(ctx context.Context, perm *models.Permission, checks *constraints) error {
	targetPath, err := j.resolveBoundRoleCheckTargetPath(ctx, checks)
	if err != nil || targetPath == "" {
		return j.UnauthorizedError(ctx, false)
	}

	binding, err := j.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByWorkspaceID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if binding == nil {
		return j.UnauthorizedError(ctx, false)
	}

	workspace, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, j.WorkspaceID)
	if err != nil {
		return err
	}

	if workspace == nil {
		return j.UnauthorizedError(ctx, false)
	}

	parentGroupPath := workspace.GetGroupPath()

	// The target must be the parent namespace itself or a descendant of it. This is the inheritance
	// direction: authority flows down from the parent, never up to it from the workspace, and never
	// sideways to an unrelated namespace.
	if targetPath != parentGroupPath && !utils.IsDescendantOfPath(targetPath, parentGroupPath) {
		return j.UnauthorizedError(ctx, false)
	}

	role, err := j.dbClient.Roles.GetRoleByID(ctx, binding.RoleID)
	if err != nil {
		return err
	}

	if role == nil {
		return j.UnauthorizedError(ctx, false)
	}

	for _, p := range role.GetPermissions() {
		if p.GTE(perm) {
			return nil
		}
	}

	return j.UnauthorizedError(ctx, false)
}

// resolveBoundRoleCheckTargetPath extracts the single namespace path the caller is asking about from
// constraints. A bound-role check only makes sense against exactly one target namespace, unlike the
// membership authorizer's RequireAccess which can be asked about several at once — job-mechanics
// permissions only ever set one constraint kind at a time in practice, so this returns an empty path
// rather than attempting to check several against a single role.
func (j *JobCaller) resolveBoundRoleCheckTargetPath(ctx context.Context, checks *constraints) (string, error) {
	switch {
	case checks.workspaceID != nil:
		ws, err := j.dbClient.Workspaces.GetWorkspaceByID(ctx, *checks.workspaceID)
		if err != nil {
			return "", err
		}
		if ws == nil {
			return "", nil
		}
		return ws.FullPath, nil
	case checks.groupID != nil:
		group, err := j.dbClient.Groups.GetGroupByID(ctx, *checks.groupID)
		if err != nil {
			return "", err
		}
		if group == nil {
			return "", nil
		}
		return group.FullPath, nil
	case len(checks.namespacePaths) == 1:
		return checks.namespacePaths[0], nil
	default:
		return "", nil
	}
}

// getCoreJobPermissionHandler returns the handler for a core job-mechanics permission: one that is
// NOT assignable to a role (see models.registerPermission's assignable flag) and therefore can never
// be granted through a WorkspaceRoleBinding. These govern the job's own identity and the run nodes
// it drives, and their result is always final — RequirePermission never falls through to
// requireBoundRoleAccess for them.
func (j *JobCaller) getCoreJobPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.UpdateJobPermission:                   j.requireJobAccess,
		models.IssueFederatedRegistryTokenPermission: j.requireJobAccess,
		models.UpdateRunPermission:                   j.requireRunWriteAccess,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}

// getSupersedableJobPermissionHandler returns the handler for a permission that IS assignable to a
// role and therefore could legitimately be granted through a WorkspaceRoleBinding, but also has
// job-mechanics meaning of its own (self-access to the job's own workspace, output visibility, or
// run access). The job-mechanics handler is tried first; if it denies, RequirePermission falls
// through to requireBoundRoleAccess before the final flat deny, so a bound role's broader authority
// can supersede the narrower job-mechanics check — see RequirePermission for the output-visibility
// case this exists for.
func (j *JobCaller) getSupersedableJobPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.ViewWorkspacePermission:                 j.requireWorkspaceOutputVisibility,
		models.ViewStateVersionPermission:              j.requireWorkspaceOutputVisibility,
		models.ViewVariablePermission:                  j.requireAccessToJobWorkspace,
		models.ViewManagedIdentityPermission:           j.requireAccessToJobWorkspace,
		models.ViewConfigurationVersionPermission:      j.requireAccessToJobWorkspace,
		models.ViewStateVersionDataPermission:          j.requireAccessToJobWorkspace,
		models.CreateStateVersionPermission:            j.requireAccessToJobWorkspace,
		models.ViewSensitiveVariableValuePermission:    j.requireAccessToJobWorkspace,
		models.ViewRunPermission:                       j.requireRunAccess, // View is automatically granted if action != View.
		models.ViewJobPermission:                       j.requireJobAccess, // View is automatically granted if action != View.
		models.CreateTerraformProviderMirrorPermission: j.requireProviderMirrorAccess,
	}

	handlerFunc, ok := handlerMap[perm]
	return handlerFunc, ok
}
