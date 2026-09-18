package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestJobCaller_GetSubject(t *testing.T) {
	caller := JobCaller{JobTRN: "trn:job:group/workspace/job-id"}
	assert.Equal(t, "trn:job:group/workspace/job-id", caller.GetSubject())
}

func TestJobCaller_IsAdmin(t *testing.T) {
	caller := JobCaller{}
	assert.False(t, caller.IsAdminModeActivated(t.Context()))
}

// TestJobCaller_GetNamespacePermissions covers reporting the permissions granted by the job's
// workspace's WorkspaceRoleBinding, following the same inheritance direction as
// requireBoundRoleAccess: authority flows down from the workspace's parent group to the workspace
// and its descendants, never up or sideways.
func TestJobCaller_GetNamespacePermissions(t *testing.T) {
	parentGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: "parent-group-id"},
		FullPath: "root/parent",
	}

	siblingGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: "sibling-group-id"},
		FullPath: "root/sibling",
	}

	jobWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "ws1"},
		GroupID:  parentGroup.Metadata.ID,
		FullPath: "root/parent/ws1",
	}

	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)

	tests := []struct {
		name          string
		binding       *models.WorkspaceRoleBinding
		rolePerms     []models.Permission
		namespacePath string
		expectEmpty   bool
	}{
		{
			name:          "workspace has no binding",
			binding:       nil,
			namespacePath: parentGroup.FullPath,
			expectEmpty:   true,
		},
		{
			name:          "namespacePath is the workspace's direct parent group",
			binding:       &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:     deployerPerms,
			namespacePath: parentGroup.FullPath,
			expectEmpty:   false,
		},
		{
			name:          "namespacePath is a descendant of the parent group",
			binding:       &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:     deployerPerms,
			namespacePath: "root/parent/child-group",
			expectEmpty:   false,
		},
		{
			name:          "namespacePath is the workspace's own namespace, a descendant of the parent",
			binding:       &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:     deployerPerms,
			namespacePath: jobWorkspace.FullPath,
			expectEmpty:   false,
		},
		{
			name:          "namespacePath is a sibling namespace, neither the parent nor a descendant",
			binding:       &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:     deployerPerms,
			namespacePath: siblingGroup.FullPath,
			expectEmpty:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockRoles := db.NewMockRoles(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)

			caller := &JobCaller{WorkspaceID: jobWorkspace.Metadata.ID}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspace.Metadata.ID).Return(jobWorkspace, nil).Maybe()
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, jobWorkspace.Metadata.ID).
				Return(test.binding, nil).Maybe()

			if test.binding != nil {
				role := &models.Role{}
				role.SetPermissions(test.rolePerms)
				mockRoles.On("GetRoleByID", mock.Anything, test.binding.RoleID).Return(role, nil).Maybe()
			}

			caller.dbClient = &db.Client{
				Workspaces:            mockWorkspaces,
				Roles:                 mockRoles,
				WorkspaceRoleBindings: mockBindings,
			}

			perms, err := caller.GetNamespacePermissions(context.Background(), test.namespacePath)
			require.NoError(t, err)

			if test.expectEmpty {
				assert.Empty(t, perms)
				return
			}

			require.Len(t, perms, len(test.rolePerms))
			for i, p := range test.rolePerms {
				assert.Equal(t, p, *perms[i])
			}
		})
	}
}

func TestJobCaller_GetRootNamespaceMemberships(t *testing.T) {
	t.Run("returns a non-nil empty slice regardless of whether the workspace has a binding", func(t *testing.T) {
		caller := JobCaller{WorkspaceID: "ws1"}

		namespaces, err := caller.GetRootNamespaceMemberships(context.Background())
		require.NoError(t, err)
		// Must be a non-nil empty slice: a nil slice is treated as "no filter" by the membership
		// filter and would expose all resources. A job caller must deny by default. Bound-role
		// authority is enforced solely through RequirePermission, never through this method.
		assert.NotNil(t, namespaces)
		assert.Empty(t, namespaces)
	})
}

func TestJobCaller_RequirePermissions(t *testing.T) {
	invalid := "invalid"

	jobGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-ID",
		},
		FullPath: "a",
	}

	jobWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath: "a/ws1",
	}

	caller := JobCaller{
		JobID:       "job1",
		WorkspaceID: jobWorkspace.Metadata.ID,
		RunID:       "run1",
	}

	mockResolver := namespace.NewMockInheritedSettingResolver(t)
	mockResolver.On("GetOutputVisibility", mock.Anything, mock.Anything).Return(&namespace.OutputVisibilitySetting{
		Value: models.OutputVisibilityRootGroup,
	}, nil).Maybe()
	caller.inheritedSettingResolver = mockResolver

	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expectErrorCode errors.CodeType
		run             *models.Run
		job             *models.Job
		name            string
		workspace       *models.Workspace
		group           *models.Group
		perms           models.Permission
		constraints     []func(*constraints)
	}{
		{
			name:        "job is associated with workspace",
			perms:       models.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(jobWorkspace.Metadata.ID)},
		},
		{
			name:        "requested workspace is under same root namespace as job's workspace",
			workspace:   &models.Workspace{FullPath: "a/ws-2"},
			perms:       models.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID("ws2")},
		},
		{
			name:        "job can view state data because it's for the workspace that contains the job",
			workspace:   &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}},
			perms:       models.CreateStateVersionPermission,
			constraints: []func(*constraints){WithWorkspaceID("ws1")},
		},
		{
			name:            "access denied because job cannot view state data for another workspace",
			workspace:       &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws2"}, FullPath: "a/ws-2"},
			perms:           models.CreateStateVersionPermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because workspace is not under same root namespace",
			workspace:       &models.Workspace{FullPath: "b/ws-2"},
			perms:           models.ViewWorkspacePermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			// requireRunAccess: a runID constraint matching the job's own run is granted directly,
			// without even consulting the workspace.
			name:        "job can view its own run by run ID",
			perms:       models.ViewRunPermission,
			constraints: []func(*constraints){WithRunID(caller.RunID)},
		},
		{
			// requireRunAccess falls through to requireAccessToWorkspacesInGroupHierarchy when the
			// runID constraint doesn't match the job's own run (or isn't given) — same root
			// namespace as the job's own workspace is enough.
			name:        "job can view a run for a workspace under the same root namespace",
			workspace:   &models.Workspace{FullPath: "a/ws-2"},
			perms:       models.ViewRunPermission,
			constraints: []func(*constraints){WithWorkspaceID("ws2")},
		},
		{
			name:            "access denied because job cannot view a run for a workspace outside its root namespace",
			workspace:       &models.Workspace{FullPath: "b/ws-2"},
			perms:           models.ViewRunPermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			// Neither a runID nor a workspaceID constraint: requireRunAccess itself returns
			// errMissingConstraints, but ViewRunPermission is a bound-role-fallback permission, so
			// RequirePermission doesn't propagate that error directly — it falls through to
			// requireBoundRoleAccess (which also can't resolve a target with no constraints) and
			// then to viewerAccessDenial, which treats "no constraints at all" as no viewer access.
			name:            "access denied because view run has no runID or workspaceID constraint",
			perms:           models.ViewRunPermission,
			expectErrorCode: errors.ENotFound,
		},
		{
			// See getPermissionHandler: ViewVariablePermission is now scoped to the job's own workspace.
			name:            "access denied because job cannot view variables for a sibling workspace in the same root namespace",
			workspace:       &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws2"}, FullPath: "a/ws-2"},
			perms:           models.ViewVariablePermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "job can view variables for its own workspace",
			perms:       models.ViewVariablePermission,
			constraints: []func(*constraints){WithWorkspaceID(jobWorkspace.Metadata.ID)},
		},
		{
			// Regression test: requireAccessToJobWorkspace previously ignored namespace-path
			// constraints, which is what the variable service actually uses.
			name:        "job can view variables for its own workspace via namespace path",
			perms:       models.ViewVariablePermission,
			constraints: []func(*constraints){WithNamespacePath(jobWorkspace.FullPath)},
		},
		{
			name:            "access denied because job cannot view variables for a sibling workspace's namespace path",
			perms:           models.ViewVariablePermission,
			constraints:     []func(*constraints){WithNamespacePath("a/ws-2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "job can view managed identities for its own workspace",
			perms:       models.ViewManagedIdentityPermission,
			constraints: []func(*constraints){WithWorkspaceID(jobWorkspace.Metadata.ID)},
		},
		{
			name:        "job can view managed identities for its own workspace via namespace path",
			perms:       models.ViewManagedIdentityPermission,
			constraints: []func(*constraints){WithNamespacePath(jobWorkspace.FullPath)},
		},
		{
			name:            "access denied because job cannot view managed identities for a sibling workspace",
			perms:           models.ViewManagedIdentityPermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			workspace:       &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws2"}, FullPath: "a/ws-2"},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because job cannot view managed identities for a sibling workspace's namespace path",
			perms:           models.ViewManagedIdentityPermission,
			constraints:     []func(*constraints){WithNamespacePath("a/ws-2")},
			expectErrorCode: errors.ENotFound,
		},
		{
			// ViewConfigurationVersionPermission only uses WithWorkspaceID in the service layer.
			name:        "job can view configuration versions for its own workspace",
			perms:       models.ViewConfigurationVersionPermission,
			constraints: []func(*constraints){WithWorkspaceID(jobWorkspace.Metadata.ID)},
		},
		{
			name:            "access denied because job cannot view configuration versions for a sibling workspace",
			perms:           models.ViewConfigurationVersionPermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			workspace:       &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws2"}, FullPath: "a/ws-2"},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because workspace doesn't exist",
			perms:           models.ViewWorkspacePermission,
			constraints:     []func(*constraints){WithWorkspaceID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "job is requesting access to itself",
			perms:       models.UpdateJobPermission,
			constraints: []func(*constraints){WithJobID(caller.JobID)},
		},
		{
			name:            "access denied because job is requesting access to another job",
			perms:           models.UpdateJobPermission,
			constraints:     []func(*constraints){WithJobID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "job has permission to write to plan",
			run:         &models.Run{Plan: models.Plan{ID: "plan1", LatestJobID: &caller.JobID}},
			perms:       models.UpdateRunPermission,
			constraints: []func(*constraints){WithRunID(caller.RunID), WithPlanID("plan1")},
		},
		{
			name:            "access denied because requested plan ID does not match run plan ID",
			run:             &models.Run{Plan: models.Plan{ID: "plan1", LatestJobID: &caller.JobID}},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPlanID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because run ID does not match caller run ID (plan)",
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(invalid), WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because plan latest job is not the caller's job",
			run:             &models.Run{Plan: models.Plan{ID: "plan1", LatestJobID: &invalid}},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because run doesn't exist (plan)",
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because plan has no latest job",
			run:             &models.Run{Plan: models.Plan{ID: "plan1"}},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "job has permission to write to apply",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1", LatestJobID: &caller.JobID},
			},
			perms:       models.UpdateRunPermission,
			constraints: []func(*constraints){WithRunID(caller.RunID), WithApplyID("apply1")},
		},
		{
			name: "access denied because requested apply ID does not match run apply ID",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1", LatestJobID: &caller.JobID},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithApplyID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because apply latest job is not the caller's job",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1", LatestJobID: &invalid},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because run doesn't exist (apply)",
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because apply has no latest job",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1"},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "job has permission to write to policy check",
			run: &models.Run{
				Plan:       models.Plan{ID: "plan1"},
				TaskStages: []*models.RunTaskStage{{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{{ID: "pc1", LatestJobID: &caller.JobID}}}},
			},
			perms:       models.UpdateRunPermission,
			constraints: []func(*constraints){WithRunID(caller.RunID), WithPolicyCheckID("pc1")},
		},
		{
			name: "access denied because requested policy check ID is not on the run",
			run: &models.Run{
				Plan:       models.Plan{ID: "plan1"},
				TaskStages: []*models.RunTaskStage{{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{{ID: "pc1", LatestJobID: &caller.JobID}}}},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPolicyCheckID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because policy check latest job is not the caller's job",
			run: &models.Run{
				Plan:       models.Plan{ID: "plan1"},
				TaskStages: []*models.RunTaskStage{{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{{ID: "pc1", LatestJobID: &invalid}}}},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPolicyCheckID("pc1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because policy check has no latest job",
			run: &models.Run{
				Plan:       models.Plan{ID: "plan1"},
				TaskStages: []*models.RunTaskStage{{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{{ID: "pc1"}}}},
			},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID), WithPolicyCheckID("pc1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because no resource constraint specified with run ID",
			run:             &models.Run{Plan: models.Plan{ID: "plan1"}},
			perms:           models.UpdateRunPermission,
			constraints:     []func(*constraints){WithRunID(caller.RunID)},
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "access denied because no permissions specified",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because permission is never available to caller",
			perms:           models.CreateWorkspacePermission,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "delete variable access denied, no constraints, not found",
			workspace:       &models.Workspace{FullPath: "a/ws1"},
			perms:           models.DeleteVariablePermission,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "delete variable access denied, matching workspace ID, found but forbidden",
			workspace:       &models.Workspace{FullPath: "a/ws1"},
			perms:           models.DeleteVariablePermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws1")},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "delete variable access denied, matching group, found but forbidden",
			workspace:       &models.Workspace{FullPath: "a/ws1"},
			perms:           models.DeleteVariablePermission,
			constraints:     []func(*constraints){WithGroupID("group-ID")},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "delete variable access denied, matching namespace path, found but forbidden",
			workspace:       &models.Workspace{FullPath: "a/ws1"},
			perms:           models.DeleteVariablePermission,
			constraints:     []func(*constraints){WithNamespacePath("a")},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:        "create provider mirror when enabled and group in hierarchy",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithNamespacePath("a")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{"providerMirrorEnabled": "true"},
			},
		},
		{
			name:        "create provider mirror with groupID when enabled",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithGroupID("group-ID")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{"providerMirrorEnabled": "true"},
			},
		},
		{
			name:        "create provider mirror access denied because not enabled",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithNamespacePath("a")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{"providerMirrorEnabled": "false"},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:        "create provider mirror access denied because property missing",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithNamespacePath("a")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:        "create provider mirror access denied because group not in hierarchy",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithNamespacePath("b")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{"providerMirrorEnabled": "true"},
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "create provider mirror access denied with groupID not in hierarchy",
			perms:       models.CreateTerraformProviderMirrorPermission,
			constraints: []func(*constraints){WithGroupID("other-group-ID")},
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: caller.JobID},
				Properties: map[string]string{"providerMirrorEnabled": "true"},
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRuns := db.NewMockRuns(t)
			mockJobs := db.NewMockJobs(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)

			constraints := getConstraints(test.constraints...)

			mockRuns.On("GetRunByID", mock.Anything, caller.RunID).Return(test.run, nil).Maybe()

			mockJobs.On("GetJobByID", mock.Anything, caller.JobID).Return(test.job, nil).Maybe()

			if constraints.workspaceID != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, *constraints.workspaceID).Return(test.workspace, nil).Maybe()
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, caller.WorkspaceID).Return(jobWorkspace, nil).Maybe()

			mockGroups.On("GetGroupByID", mock.Anything, "group-ID").Return(jobGroup, nil).Maybe()
			mockGroups.On("GetGroupByID", mock.Anything, "other-group-ID").Return(&models.Group{FullPath: "b"}, nil).Maybe()

			// No workspace has a role binding in this test; every case here exercises a
			// job-mechanics handler or the plain deny path, never the bound-role fallthrough.
			mockBindings := db.NewMockWorkspaceRoleBindings(t)
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

			caller.dbClient = &db.Client{
				Runs:                  mockRuns,
				Jobs:                  mockJobs,
				Workspaces:            mockWorkspaces,
				Groups:                mockGroups,
				WorkspaceRoleBindings: mockBindings,
			}

			err := caller.RequirePermission(ctx, test.perms, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestJobCaller_RequireInheritedPermissions(t *testing.T) {
	invalid := "invalid"

	caller := JobCaller{WorkspaceID: "ws1"}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expectErrorCode errors.CodeType
		workspace       *models.Workspace
		group           *models.Group
		name            string
		modelType       types.ModelType
		constraints     []func(*constraints)
	}{
		{
			name:        "workspace is in requested group",
			workspace:   &models.Workspace{FullPath: "a/ws1"},
			group:       &models.Group{FullPath: "a"},
			modelType:   types.ManagedIdentityModelType,
			constraints: []func(*constraints){WithGroupID("group1")},
		},
		{
			name:            "access denied because workspace is not in requested group",
			workspace:       &models.Workspace{FullPath: "b/ws1"},
			group:           &models.Group{FullPath: "a"},
			modelType:       types.ManagedIdentityModelType,
			constraints:     []func(*constraints){WithGroupID("group1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because workspace not found",
			modelType:       types.ManagedIdentityModelType,
			constraints:     []func(*constraints){WithGroupID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because group not found",
			workspace:       &models.Workspace{},
			modelType:       types.ManagedIdentityModelType,
			constraints:     []func(*constraints){WithGroupID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "workspace is in requested namespace path",
			workspace:   &models.Workspace{FullPath: "a/ws1"},
			modelType:   types.ManagedIdentityModelType,
			constraints: []func(*constraints){WithNamespacePath("a")},
		},
		{
			name:            "access denied because workspace not found",
			modelType:       types.ManagedIdentityModelType,
			constraints:     []func(*constraints){WithNamespacePath("a")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because required constraints not provided",
			modelType:       types.ManagedIdentityModelType,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockGroups := db.NewMockGroups(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			constraints := getConstraints(test.constraints...)

			if constraints.groupID != nil {
				mockGroups.On("GetGroupByID", mock.Anything, *constraints.groupID).Return(test.group, nil).Maybe()
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, caller.WorkspaceID).Return(test.workspace, nil).Maybe()

			caller.dbClient = &db.Client{Groups: mockGroups, Workspaces: mockWorkspaces}

			err := caller.RequireAccessToInheritableResource(ctx, test.modelType, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestJobCaller_OutputVisibilityAccess(t *testing.T) {
	requestingWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "requesting-ws"},
		FullPath: "cloud/networking/ws-requester",
		GroupID:  "networking-group-id",
	}

	t.Run("self-access is always allowed", func(t *testing.T) {
		targetWSID := "requesting-ws"

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWSID})
		require.NoError(t, err)
	})

	t.Run("BLOCK_ACCESS denies all access", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityBlockAccess,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("DIRECT_GROUP_ONLY allows same group", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupOnly,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("DIRECT_GROUP_ONLY denies different group", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/applications/ws-target",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupOnly,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("DIRECT_GROUP_AND_SUBGROUPS allows same group", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupAndSubgroups,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("DIRECT_GROUP_AND_SUBGROUPS allows child subgroup", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		// Requesting workspace is in a child subgroup of networking
		childWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "requesting-ws"},
			FullPath: "cloud/networking/inner/ws-deep",
			GroupID:  "inner-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(childWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupAndSubgroups,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("DIRECT_GROUP_AND_SUBGROUPS denies different group branch", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		// Requesting workspace is in a completely different branch (applications)
		differentBranchWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "requesting-ws"},
			FullPath: "cloud/applications/ws-frontend",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(differentBranchWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupAndSubgroups,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("ROOT_GROUP allows same root group", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/applications/ws-target",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityRootGroup,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("ROOT_GROUP denies different root group", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "operations/monitoring/ws-target",
			GroupID:  "monitoring-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityRootGroup,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("GLOBAL allows any workspace", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "operations/monitoring/ws-target",
			GroupID:  "monitoring-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityGlobal,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("nil perm falls back to existing root namespace check", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/applications/ws-target",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		// nil perm — should use existing root namespace logic (same root group = allowed)
		err := caller.requireAccessToWorkspacesInGroupHierarchy(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})

	t.Run("non-output perm uses existing check", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/applications/ws-target",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		// Exercises requireAccessToWorkspacesInGroupHierarchy directly (same root group = allowed).
		// No permission currently routes here via getPermissionHandler that also has an output
		// visibility variant; kept as a standalone test of the function's own behavior.
		err := caller.requireAccessToWorkspacesInGroupHierarchy(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err) // same root group = allowed
	})

	t.Run("ViewWorkspacePermission uses visibility check for cross-root access", func(t *testing.T) {
		// A workspace in a different root group — would be denied by old root namespace check,
		// but allowed because target has GLOBAL.
		// This verifies ViewWorkspacePermission goes through the visibility path.
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "operations/monitoring/ws-target",
			GroupID:  "monitoring-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "target-ws").Return(targetWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityGlobal,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		// ViewWorkspacePermission with GLOBAL allows cross-root access
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{workspaceID: &targetWorkspace.Metadata.ID})
		require.NoError(t, err)
	})
}

func TestJobCaller_OutputVisibilityAccess_NamespacePaths(t *testing.T) {
	requestingWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "requesting-ws"},
		FullPath: "cloud/networking/ws-requester",
		GroupID:  "networking-group-id",
	}

	t.Run("namespacePath resolves to workspace and visibility allows", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/networking/ws-target",
			GroupID:  "networking-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, "trn:workspace:cloud/networking/ws-target").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupOnly,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{namespacePaths: []string{"cloud/networking/ws-target"}})
		require.NoError(t, err)
	})

	t.Run("namespacePath resolves to workspace and visibility denies", func(t *testing.T) {
		targetWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "target-ws"},
			FullPath: "cloud/applications/ws-target",
			GroupID:  "applications-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, "trn:workspace:cloud/applications/ws-target").Return(targetWorkspace, nil)
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)
		mockResolver.On("GetOutputVisibility", mock.Anything, targetWorkspace).Return(&namespace.OutputVisibilitySetting{
			Value: models.OutputVisibilityDirectGroupOnly,
		}, nil)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{namespacePaths: []string{"cloud/applications/ws-target"}})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("namespacePath does not resolve to workspace — returns unauthorized", func(t *testing.T) {
		mockWorkspaces := db.NewMockWorkspaces(t)
		// GetWorkspaceByTRN returns nil — path doesn't resolve to a workspace
		mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, "trn:workspace:cloud/networking").Return(nil, nil)
		// UnauthorizedError needs to load the requesting workspace for the error message
		mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "requesting-ws").Return(requestingWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{namespacePaths: []string{"cloud/networking"}})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("namespacePath matches requesting workspace — self-access", func(t *testing.T) {
		// The target path resolves to a workspace with the same ID as the requester
		selfWorkspace := &models.Workspace{
			Metadata: models.ResourceMetadata{ID: "requesting-ws"},
			FullPath: "cloud/networking/ws-requester",
			GroupID:  "networking-group-id",
		}

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, "trn:workspace:cloud/networking/ws-requester").Return(selfWorkspace, nil)

		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{namespacePaths: []string{"cloud/networking/ws-requester"}})
		require.NoError(t, err)
	})

	t.Run("namespacePath DB error propagates instead of falling back", func(t *testing.T) {
		mockWorkspaces := db.NewMockWorkspaces(t)
		// GetWorkspaceByTRN returns a DB error — should propagate, not fall back
		mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, "trn:workspace:cloud/networking/ws-target").Return(nil, errors.New("connection refused", errors.WithErrorCode(errors.EInternal)))

		mockResolver := namespace.NewMockInheritedSettingResolver(t)

		caller := &JobCaller{
			WorkspaceID:              "requesting-ws",
			dbClient:                 &db.Client{Workspaces: mockWorkspaces},
			inheritedSettingResolver: mockResolver,
		}

		ctx := WithCaller(t.Context(), caller)
		err := caller.requireWorkspaceOutputVisibility(ctx, nil, &constraints{namespacePaths: []string{"cloud/networking/ws-target"}})
		require.Error(t, err)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

// TestJobCaller_RequirePermission_BoundRoleFallthrough covers the workspace role binding
// fallthrough: for a permission with no job-mechanics handler in getPermissionHandler, a job whose
// workspace has a WorkspaceRoleBinding may still be authorized if the bound role covers the
// permission at the target namespace, which must be the workspace's DIRECT PARENT group or a
// descendant of it.
func TestJobCaller_RequirePermission_BoundRoleFallthrough(t *testing.T) {
	parentGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: "parent-group-id"},
		FullPath: "root/parent",
	}

	siblingGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: "sibling-group-id"},
		FullPath: "root/sibling",
	}

	jobWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "ws1"},
		GroupID:  parentGroup.Metadata.ID,
		FullPath: "root/parent/ws1",
	}

	// CreateGroupPermission has no job-mechanics handler in getPermissionHandler, so it can only be
	// satisfied through the bound-role fallthrough. That makes it a clean permission to test the
	// fallthrough with, without any job-mechanics handler being able to mask a bug in it.
	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)
	viewerPerms, ok := models.ViewerRoleID.Permissions()
	require.True(t, ok)

	tests := []struct {
		name            string
		binding         *models.WorkspaceRoleBinding
		rolePerms       []models.Permission
		targetGroup     *models.Group
		targetNamespace string // used instead of a group constraint when set
		expectErrorCode errors.CodeType
	}{
		{
			name:            "job is authorized via the bound role at the direct parent namespace",
			binding:         &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:       deployerPerms,
			targetGroup:     parentGroup,
			expectErrorCode: "",
		},
		{
			name:            "job is authorized via the bound role at a descendant of the parent namespace",
			binding:         &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:       deployerPerms,
			targetNamespace: "root/parent/child-group",
			expectErrorCode: "",
		},
		{
			// After the bound-role fallthrough denies, RequirePermission falls back to its normal
			// viewer-access check. The job's workspace IS a descendant of the target group, so it
			// has viewer access there and the failure is EForbidden, not ENotFound — this is
			// unrelated existing behavior, not something the binding fallthrough changes.
			name:            "job is denied when the workspace has no binding",
			binding:         nil,
			targetGroup:     parentGroup,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "job is denied when the bound role lacks the requested permission",
			binding:         &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:       viewerPerms,
			targetGroup:     parentGroup,
			expectErrorCode: errors.EForbidden,
		},
		{
			// The target group here is NOT an ancestor of the job's workspace, so the fallback
			// viewer-access check also denies, and denies with no viewer access — ENotFound.
			name:            "job is denied when the target is a sibling namespace, not the parent or a descendant",
			binding:         &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:       deployerPerms,
			targetGroup:     siblingGroup,
			expectErrorCode: errors.ENotFound,
		},
		{
			// The workspace's OWN namespace is itself a descendant of its parent, so the bound
			// role's authority — which covers the parent and everything beneath it — legitimately
			// extends to the workspace's own namespace too. This is intended: a bound Deployer role
			// lets the job manage resources (e.g. sibling workspaces) anywhere in the subtree rooted
			// at the parent, the job's own workspace included. This is authorized, not denied.
			name:            "the bound role's authority extends to the workspace's own namespace, since it is a descendant of the parent",
			binding:         &models.WorkspaceRoleBinding{RoleID: "role-1"},
			rolePerms:       deployerPerms,
			targetNamespace: jobWorkspace.FullPath,
			expectErrorCode: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			mockRoles := db.NewMockRoles(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)

			caller := &JobCaller{WorkspaceID: jobWorkspace.Metadata.ID}
			ctx := WithCaller(context.Background(), caller)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspace.Metadata.ID).Return(jobWorkspace, nil).Maybe()
			mockGroups.On("GetGroupByID", mock.Anything, parentGroup.Metadata.ID).Return(parentGroup, nil).Maybe()
			mockGroups.On("GetGroupByID", mock.Anything, siblingGroup.Metadata.ID).Return(siblingGroup, nil).Maybe()
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, jobWorkspace.Metadata.ID).
				Return(test.binding, nil).Maybe()

			if test.binding != nil && test.rolePerms != nil {
				role := &models.Role{}
				role.SetPermissions(test.rolePerms)
				mockRoles.On("GetRoleByID", mock.Anything, test.binding.RoleID).Return(role, nil).Maybe()
			}

			caller.dbClient = &db.Client{
				Workspaces:            mockWorkspaces,
				Groups:                mockGroups,
				Roles:                 mockRoles,
				WorkspaceRoleBindings: mockBindings,
			}

			var checkOpt func(*constraints)
			if test.targetNamespace != "" {
				checkOpt = WithNamespacePath(test.targetNamespace)
			} else {
				checkOpt = WithGroupID(test.targetGroup.Metadata.ID)
			}

			err := caller.RequirePermission(ctx, models.CreateGroupPermission, checkOpt)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				require.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestJobCaller_RequirePermission_BoundRoleSupersedesOutputVisibility covers the deliberate
// precedence for assignable permissions that also have job-mechanics handling: the job-mechanics
// handler runs first, but if it denies, RequirePermission falls through to requireBoundRoleAccess
// (see getSupersedableJobPermissionHandler), and a sufficiently-permissioned bound role supersedes
// the narrower handler's denial.
//
// Concretely: a workspace bound to Deployer can read state version outputs from ANY workspace in
// its parent group's subtree, even one that has explicitly set output visibility to block_access.
// Deployer already grants broader access to that subtree than output visibility is trying to
// narrow, so letting one workspace opt out of visibility to a strictly narrower slice of the same
// authority would be inconsistent — the bound role's authority level supersedes the setting.
func TestJobCaller_RequirePermission_BoundRoleSupersedesOutputVisibility(t *testing.T) {
	parentGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: "parent-group-id"},
		FullPath: "root/parent",
	}

	requestingWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "ws1"},
		GroupID:  parentGroup.Metadata.ID,
		FullPath: "root/parent/ws1",
	}

	// A sibling workspace under the SAME parent, so it is within the bound role's namespace scope.
	blockedTargetWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: "target-ws"},
		GroupID:  parentGroup.Metadata.ID,
		FullPath: "root/parent/target-ws",
	}

	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)
	require.Contains(t, deployerPerms, models.ViewStateVersionPermission)

	viewerPerms, ok := models.ViewerRoleID.Permissions()
	require.True(t, ok)

	tests := []struct {
		name            string
		rolePerms       []models.Permission
		visibility      models.NamespaceOutputVisibilityLevel
		expectErrorCode errors.CodeType
	}{
		{
			name:       "bound Deployer role supersedes block_access on a sibling workspace",
			rolePerms:  deployerPerms,
			visibility: models.OutputVisibilityBlockAccess,
		},
		{
			// Sanity check: when visibility already allows it, no fallback is needed either way.
			name:       "global visibility allows access regardless of any binding",
			rolePerms:  viewerPerms,
			visibility: models.OutputVisibilityGlobal,
		},
		{
			// The bound role does NOT contain ViewStateVersionPermission (Viewer doesn't either,
			// but this uses a role that plainly lacks it), so block_access still wins: the
			// fallback only supersedes the setting when the role actually covers the permission.
			name:            "bound role lacking the permission does not supersede block_access",
			rolePerms:       []models.Permission{models.ViewGroupPermission},
			visibility:      models.OutputVisibilityBlockAccess,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)
			mockRoles := db.NewMockRoles(t)
			mockResolver := namespace.NewMockInheritedSettingResolver(t)

			caller := &JobCaller{
				WorkspaceID:              requestingWorkspace.Metadata.ID,
				inheritedSettingResolver: mockResolver,
			}
			ctx := WithCaller(context.Background(), caller)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, blockedTargetWorkspace.Metadata.ID).
				Return(blockedTargetWorkspace, nil)
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, requestingWorkspace.Metadata.ID).
				Return(requestingWorkspace, nil).Maybe()
			mockGroups.On("GetGroupByID", mock.Anything, parentGroup.Metadata.ID).Return(parentGroup, nil).Maybe()

			mockResolver.On("GetOutputVisibility", mock.Anything, blockedTargetWorkspace).
				Return(&namespace.OutputVisibilitySetting{Value: test.visibility}, nil).Maybe()

			role := &models.Role{}
			role.SetPermissions(test.rolePerms)
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, requestingWorkspace.Metadata.ID).
				Return(&models.WorkspaceRoleBinding{RoleID: "role-1"}, nil).Maybe()
			mockRoles.On("GetRoleByID", mock.Anything, "role-1").Return(role, nil).Maybe()

			caller.dbClient = &db.Client{
				Workspaces:            mockWorkspaces,
				Groups:                mockGroups,
				WorkspaceRoleBindings: mockBindings,
				Roles:                 mockRoles,
			}

			err := caller.RequirePermission(ctx, models.ViewStateVersionPermission, WithWorkspaceID(blockedTargetWorkspace.Metadata.ID))

			if test.expectErrorCode != "" {
				require.Error(t, err)
				require.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestJobCaller_RequirePermission_CoreJobPermissionsNeverFallThrough covers the other half of the
// split: permissions handled by getCoreJobPermissionHandler are not assignable to a role at all, so
// their handler's result must be final. requireBoundRoleAccess must never even be consulted for
// them, regardless of what a bound role happens to contain.
func TestJobCaller_RequirePermission_CoreJobPermissionsNeverFallThrough(t *testing.T) {
	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)

	caller := &JobCaller{JobID: "job-1", WorkspaceID: "ws1"}
	ctx := WithCaller(context.Background(), caller)

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockBindings := db.NewMockWorkspaceRoleBindings(t)
	mockRoles := db.NewMockRoles(t)

	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}, FullPath: "root/ws1"}, nil).Maybe()

	// A binding exists and its role (deliberately, hypothetically) contains UpdateJobPermission, to
	// prove that even so, the fallthrough is never consulted for a core job permission.
	binding := &models.WorkspaceRoleBinding{RoleID: "role-1"}
	role := &models.Role{}
	role.SetPermissions(append(deployerPerms, models.UpdateJobPermission))

	mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, "ws1").Return(binding, nil).Maybe()
	mockRoles.On("GetRoleByID", mock.Anything, "role-1").Return(role, nil).Maybe()

	caller.dbClient = &db.Client{
		Workspaces:            mockWorkspaces,
		WorkspaceRoleBindings: mockBindings,
		Roles:                 mockRoles,
	}

	// Requesting access to a DIFFERENT job ID must still be denied by requireJobAccess, regardless
	// of what the bound role contains.
	err := caller.RequirePermission(ctx, models.UpdateJobPermission, WithJobID("some-other-job"))
	require.Error(t, err)
	require.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

// TestJobCaller_ResolveBoundRoleCheckTargetPath covers resolveBoundRoleCheckTargetPath directly: it
// must resolve exactly one target path from whichever single constraint kind is set (workspace ID,
// group ID, or a lone namespace path), return an empty path when nothing resolves or more than one
// namespace path is given, and propagate DB errors rather than treating them as "no match."
func TestJobCaller_ResolveBoundRoleCheckTargetPath(t *testing.T) {
	ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}, FullPath: "root/parent/ws1"}
	group := &models.Group{Metadata: models.ResourceMetadata{ID: "group1"}, FullPath: "root/parent"}

	tests := []struct {
		name            string
		checks          *constraints
		setupMocks      func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups)
		expectPath      string
		expectErrorCode errors.CodeType
	}{
		{
			name:   "workspace ID constraint resolves to the workspace's full path",
			checks: &constraints{workspaceID: &ws.Metadata.ID},
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, _ *db.MockGroups) {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)
			},
			expectPath: ws.FullPath,
		},
		{
			name:   "workspace ID constraint resolves to empty path when the workspace does not exist",
			checks: &constraints{workspaceID: &ws.Metadata.ID},
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, _ *db.MockGroups) {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(nil, nil)
			},
			expectPath: "",
		},
		{
			name:   "workspace ID constraint propagates a DB error",
			checks: &constraints{workspaceID: &ws.Metadata.ID},
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, _ *db.MockGroups) {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).
					Return(nil, errors.New("connection refused", errors.WithErrorCode(errors.EInternal)))
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name:   "group ID constraint resolves to the group's full path",
			checks: &constraints{groupID: &group.Metadata.ID},
			setupMocks: func(_ *db.MockWorkspaces, mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, group.Metadata.ID).Return(group, nil)
			},
			expectPath: group.FullPath,
		},
		{
			name:   "group ID constraint resolves to empty path when the group does not exist",
			checks: &constraints{groupID: &group.Metadata.ID},
			setupMocks: func(_ *db.MockWorkspaces, mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, group.Metadata.ID).Return(nil, nil)
			},
			expectPath: "",
		},
		{
			name:   "group ID constraint propagates a DB error",
			checks: &constraints{groupID: &group.Metadata.ID},
			setupMocks: func(_ *db.MockWorkspaces, mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, group.Metadata.ID).
					Return(nil, errors.New("connection refused", errors.WithErrorCode(errors.EInternal)))
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "a single namespace path constraint is returned as-is",
			checks:     &constraints{namespacePaths: []string{"root/parent/child"}},
			setupMocks: func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectPath: "root/parent/child",
		},
		{
			name:       "more than one namespace path resolves to empty, since a bound-role check only supports a single target",
			checks:     &constraints{namespacePaths: []string{"root/parent/child-1", "root/parent/child-2"}},
			setupMocks: func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectPath: "",
		},
		{
			name:       "no constraints set at all resolves to empty",
			checks:     &constraints{},
			setupMocks: func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectPath: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			test.setupMocks(mockWorkspaces, mockGroups)

			caller := &JobCaller{
				dbClient: &db.Client{Workspaces: mockWorkspaces, Groups: mockGroups},
			}

			path, err := caller.resolveBoundRoleCheckTargetPath(context.Background(), test.checks)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				require.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectPath, path)
		})
	}
}

// TestJobCaller_RequireBoundRoleAccess_DBErrors covers requireBoundRoleAccess's error propagation:
// a DB error from any of the lookups it performs (the binding itself, the job's workspace, the
// workspace's parent group, or the bound role) must propagate as-is rather than being swallowed
// into an UnauthorizedError. Silently converting a DB error into a deny would make outages look
// like permission problems and would be indistinguishable from a real deny in logs/metrics.
func TestJobCaller_RequireBoundRoleAccess_DBErrors(t *testing.T) {
	dbErr := errors.New("connection refused", errors.WithErrorCode(errors.EInternal))

	ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}, GroupID: "group1", FullPath: "root/parent/ws1"}
	parentGroup := &models.Group{Metadata: models.ResourceMetadata{ID: "group1"}, FullPath: "root/parent"}
	binding := &models.WorkspaceRoleBinding{RoleID: "role-1"}

	tests := []struct {
		name       string
		setupMocks func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups, mockBindings *db.MockWorkspaceRoleBindings, mockRoles *db.MockRoles)
	}{
		{
			name: "binding lookup error propagates",
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups, mockBindings *db.MockWorkspaceRoleBindings, _ *db.MockRoles) {
				mockGroups.On("GetGroupByID", mock.Anything, parentGroup.Metadata.ID).Return(parentGroup, nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil).Maybe()
				mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, ws.Metadata.ID).Return(nil, dbErr)
			},
		},
		{
			name: "workspace lookup error propagates",
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups, mockBindings *db.MockWorkspaceRoleBindings, _ *db.MockRoles) {
				mockGroups.On("GetGroupByID", mock.Anything, parentGroup.Metadata.ID).Return(parentGroup, nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(nil, dbErr)
				mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, ws.Metadata.ID).Return(binding, nil)
			},
		},
		{
			name: "role lookup error propagates",
			setupMocks: func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups, mockBindings *db.MockWorkspaceRoleBindings, mockRoles *db.MockRoles) {
				mockGroups.On("GetGroupByID", mock.Anything, parentGroup.Metadata.ID).Return(parentGroup, nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)
				mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, ws.Metadata.ID).Return(binding, nil)
				mockRoles.On("GetRoleByID", mock.Anything, binding.RoleID).Return(nil, dbErr)
			},
		},
	}

	// Every case targets the parent group directly via a group ID constraint, resolved by
	// resolveBoundRoleCheckTargetPath's own GetGroupByID call.
	targetChecksByTest := map[string]*constraints{
		"binding lookup error propagates":   {groupID: &parentGroup.Metadata.ID},
		"workspace lookup error propagates": {groupID: &parentGroup.Metadata.ID},
		"role lookup error propagates":      {groupID: &parentGroup.Metadata.ID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)
			mockRoles := db.NewMockRoles(t)
			test.setupMocks(mockWorkspaces, mockGroups, mockBindings, mockRoles)

			caller := &JobCaller{
				WorkspaceID: ws.Metadata.ID,
				dbClient: &db.Client{
					Workspaces:            mockWorkspaces,
					Groups:                mockGroups,
					WorkspaceRoleBindings: mockBindings,
					Roles:                 mockRoles,
				},
			}

			perm := models.CreateGroupPermission
			err := caller.requireBoundRoleAccess(context.Background(), &perm, targetChecksByTest[test.name])

			require.Error(t, err)
			require.Equal(t, errors.EInternal, errors.ErrorCode(err))
		})
	}
}

// TestJobCaller_GetCoreJobPermissionHandler locks down exactly which permissions are core
// job-mechanics permissions: those NOT assignable to a role (see models.registerPermission), whose
// handler's result RequirePermission always treats as final, never falling through to
// requireBoundRoleAccess. If a permission is ever added here, it must not be assignable — otherwise
// a WorkspaceRoleBinding could never actually grant it despite being allowed to contain it.
func TestJobCaller_GetCoreJobPermissionHandler(t *testing.T) {
	caller := &JobCaller{}

	corePermissions := []models.Permission{
		models.UpdateJobPermission,
		models.IssueFederatedRegistryTokenPermission,
		models.UpdateRunPermission,
	}

	for _, perm := range corePermissions {
		permCopy := perm
		t.Run(permCopy.String(), func(t *testing.T) {
			handler, ok := caller.getCoreJobPermissionHandler(permCopy)
			require.True(t, ok, "expected %s to have a core job-mechanics handler", permCopy)
			require.NotNil(t, handler)

			// A core job-mechanics permission must never be assignable to a role: if it were, a
			// WorkspaceRoleBinding could contain it but RequirePermission would still never grant
			// it through the bound-role path, which would make the permission silently unusable
			// via a binding despite Role.Validate() accepting it into a custom role.
			assert.False(t, permCopy.IsAssignable(), "core job permission %s must not be assignable to a role", permCopy)
		})
	}

	// A permission with a bound-role fallback handler (assignable) must not also be treated as core.
	_, ok := caller.getCoreJobPermissionHandler(models.ViewWorkspacePermission)
	assert.False(t, ok, "an assignable, fallback-eligible permission must not have a core handler")

	// A permission with no handler at all in either map must not spuriously match.
	_, ok = caller.getCoreJobPermissionHandler(models.CreateGroupPermission)
	assert.False(t, ok)
}

// TestJobCaller_getSupersedableJobPermissionHandler locks down exactly which permissions have a
// job-mechanics handler that a bound role's authority can supersede on denial (see RequirePermission
// and getSupersedableJobPermissionHandler's doc comment). Every one of these must be assignable to
// a role, since the whole point of the fallback is that a WorkspaceRoleBinding might grant it.
func TestJobCaller_getSupersedableJobPermissionHandler(t *testing.T) {
	caller := &JobCaller{}

	fallbackPermissions := []models.Permission{
		models.ViewWorkspacePermission,
		models.ViewStateVersionPermission,
		models.ViewVariablePermission,
		models.ViewManagedIdentityPermission,
		models.ViewConfigurationVersionPermission,
		models.ViewStateVersionDataPermission,
		models.CreateStateVersionPermission,
		models.ViewSensitiveVariableValuePermission,
		models.ViewRunPermission,
		models.ViewJobPermission,
		models.CreateTerraformProviderMirrorPermission,
	}

	for _, perm := range fallbackPermissions {
		permCopy := perm
		t.Run(permCopy.String(), func(t *testing.T) {
			handler, ok := caller.getSupersedableJobPermissionHandler(permCopy)
			require.True(t, ok, "expected %s to have a supersedable job-mechanics handler", permCopy)
			require.NotNil(t, handler)

			// Every fallback-eligible permission must be assignable, since a WorkspaceRoleBinding
			// can only ever grant permissions that a role is allowed to hold in the first place.
			assert.True(t, permCopy.IsAssignable(), "bound-role fallback permission %s must be assignable to a role", permCopy)
		})
	}

	// A core job-mechanics permission must not also appear in the fallback map — the split between
	// the two handler maps is meant to be a partition (see RequirePermission), not overlapping.
	for _, perm := range []models.Permission{models.UpdateJobPermission, models.IssueFederatedRegistryTokenPermission, models.UpdateRunPermission} {
		_, ok := caller.getSupersedableJobPermissionHandler(perm)
		assert.False(t, ok, "core job permission %s must not also have a bound-role fallback handler", perm)
	}

	// A permission with no handler at all in either map must not spuriously match.
	_, ok := caller.getSupersedableJobPermissionHandler(models.CreateGroupPermission)
	assert.False(t, ok)
}

// TestJobCaller_ViewerAccessDenial covers viewerAccessDenial directly: the final denial
// RequirePermission falls back to once neither a job-mechanics handler nor a bound role satisfied
// the request. It must choose EForbidden when the job has at least viewer access to the constrained
// resource, and ENotFound otherwise, and it must treat "no constraints at all" as no viewer access
// (there being nothing to check access against).
func TestJobCaller_ViewerAccessDenial(t *testing.T) {
	jobWorkspace := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}, FullPath: "root/ws1"}
	siblingWorkspaceID := "ws2"

	tests := []struct {
		name            string
		checks          *constraints
		setupMocks      func(mockWorkspaces *db.MockWorkspaces, mockGroups *db.MockGroups)
		expectErrorCode errors.CodeType
	}{
		{
			name:            "no constraints at all denies with no viewer access",
			checks:          &constraints{},
			setupMocks:      func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "workspace constraint matching the job's own workspace has viewer access",
			checks:          &constraints{workspaceID: &jobWorkspace.Metadata.ID},
			setupMocks:      func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "workspace constraint for a different workspace has no viewer access",
			checks:          &constraints{workspaceID: &siblingWorkspaceID},
			setupMocks:      func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:   "group constraint the job's workspace descends from has viewer access",
			checks: &constraints{groupID: strPtr("group-1")},
			setupMocks: func(_ *db.MockWorkspaces, mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").Return(&models.Group{FullPath: "root"}, nil)
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:   "group constraint the job's workspace does not descend from has no viewer access",
			checks: &constraints{groupID: strPtr("group-1")},
			setupMocks: func(_ *db.MockWorkspaces, mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").Return(&models.Group{FullPath: "unrelated"}, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "namespace path constraint the job's workspace descends from has viewer access",
			checks:          &constraints{namespacePaths: []string{"root"}},
			setupMocks:      func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "namespace path constraint the job's workspace does not descend from has no viewer access",
			checks:          &constraints{namespacePaths: []string{"unrelated"}},
			setupMocks:      func(_ *db.MockWorkspaces, _ *db.MockGroups) {},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			// UnauthorizedError always loads the job's own workspace to build its message.
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspace.Metadata.ID).Return(jobWorkspace, nil).Maybe()
			test.setupMocks(mockWorkspaces, mockGroups)

			caller := &JobCaller{
				WorkspaceID: jobWorkspace.Metadata.ID,
				dbClient:    &db.Client{Workspaces: mockWorkspaces, Groups: mockGroups},
			}

			err := caller.viewerAccessDenial(context.Background(), test.checks)

			require.Error(t, err)
			require.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
		})
	}
}

func strPtr(s string) *string { return &s }
