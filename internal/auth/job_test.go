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

func TestJobCaller_GetRootNamespaceMemberships(t *testing.T) {
	caller := JobCaller{}
	namespaces, err := caller.GetRootNamespaceMemberships(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Must be a non-nil empty slice: a nil slice is treated as "no filter" by the membership
	// filter and would expose all resources. A job caller must deny by default.
	assert.NotNil(t, namespaces)
	assert.Empty(t, namespaces)
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
			run:         &models.Run{Plan: models.Plan{ID: "plan1"}},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       models.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
		},
		{
			name:            "access denied because requested plan ID does not match run plan ID",
			run:             &models.Run{Plan: models.Plan{ID: "plan1"}},
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because job IDs do not match",
			run:             &models.Run{Plan: models.Plan{ID: "plan1"}},
			job:             &models.Job{Metadata: models.ResourceMetadata{ID: invalid}},
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because run doesn't exist",
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because latest plan job doesn't exist",
			run:             &models.Run{Plan: models.Plan{ID: "plan1"}},
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "job has permission to write to apply",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1"},
			},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       models.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
		},
		{
			name: "access denied because requested apply ID does not match run apply ID",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1"},
			},
			perms:           models.UpdateApplyPermission,
			constraints:     []func(*constraints){WithApplyID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because job IDs do not match",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1"},
			},
			job:             &models.Job{Metadata: models.ResourceMetadata{ID: invalid}},
			perms:           models.UpdateApplyPermission,
			constraints:     []func(*constraints){WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because run doesn't exist",
			perms:           models.UpdateApplyPermission,
			constraints:     []func(*constraints){WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "access denied because latest apply job doesn't exist",
			run: &models.Run{
				Plan:  models.Plan{ID: "plan1"},
				Apply: &models.Apply{ID: "apply1"},
			},
			perms:           models.UpdateApplyPermission,
			constraints:     []func(*constraints){WithApplyID("apply1")},
			expectErrorCode: errors.ENotFound,
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

			stage := models.JobPlanType
			if constraints.applyID != nil {
				stage = models.JobApplyType
			}

			mockRuns.On("GetRunByID", mock.Anything, caller.RunID).Return(test.run, nil).Maybe()

			mockJobs.On("GetLatestJobByType", mock.Anything, caller.RunID, stage).Return(test.job, nil).Maybe()
			mockJobs.On("GetJobByID", mock.Anything, caller.JobID).Return(test.job, nil).Maybe()

			if constraints.workspaceID != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, *constraints.workspaceID).Return(test.workspace, nil).Maybe()
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, caller.WorkspaceID).Return(jobWorkspace, nil).Maybe()

			mockGroups.On("GetGroupByID", mock.Anything, "group-ID").Return(jobGroup, nil).Maybe()
			mockGroups.On("GetGroupByID", mock.Anything, "other-group-ID").Return(&models.Group{FullPath: "b"}, nil).Maybe()

			caller.dbClient = &db.Client{
				Runs:       mockRuns,
				Jobs:       mockJobs,
				Workspaces: mockWorkspaces,
				Groups:     mockGroups,
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

func TestJobCaller_RequireRole(t *testing.T) {
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{FullPath: "group/workspace"}, nil)

	caller := JobCaller{WorkspaceID: "ws-1", dbClient: &db.Client{Workspaces: mockWorkspaces}}
	err := caller.RequireRole(WithCaller(t.Context(), &caller), models.OwnerRoleID.String())
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
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
		// ViewConfigurationVersionPermission should use existing root namespace check, not output visibility
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
