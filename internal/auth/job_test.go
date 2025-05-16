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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestJobCaller_GetSubject(t *testing.T) {
	caller := JobCaller{JobID: "job1"}
	assert.Equal(t, "job1", caller.GetSubject())
}

func TestJobCaller_IsAdmin(t *testing.T) {
	caller := JobCaller{}
	assert.False(t, caller.IsAdmin())
}

func TestJobCaller_GetNamespaceAccessPolicy(t *testing.T) {
	caller := JobCaller{}
	policy, err := caller.GetNamespaceAccessPolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &NamespaceAccessPolicy{
		AllowAll:         false,
		RootNamespaceIDs: []string{},
	}, policy)
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

	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expectErrorCode errors.CodeType
		run             *models.Run
		job             *models.Job
		name            string
		workspace       *models.Workspace
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
			run:         &models.Run{PlanID: "plan1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       models.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
		},
		{
			name:            "access denied because requested plan ID does not match run plan ID",
			run:             &models.Run{PlanID: "plan1"},
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because job IDs do not match",
			run:             &models.Run{PlanID: "plan1"},
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
			run:             &models.Run{PlanID: "plan1"},
			perms:           models.UpdatePlanPermission,
			constraints:     []func(*constraints){WithPlanID("plan1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "job has permission to write to apply",
			run:         &models.Run{ApplyID: "apply1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       models.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
		},
		{
			name:            "access denied because requested apply ID does not match run apply ID",
			run:             &models.Run{ApplyID: "apply1"},
			perms:           models.UpdateApplyPermission,
			constraints:     []func(*constraints){WithApplyID(invalid)},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because job IDs do not match",
			run:             &models.Run{ApplyID: "apply1"},
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
			name:            "access denied because latest apply job doesn't exist",
			run:             &models.Run{ApplyID: "apply1"},
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
			if constraints.workspaceID != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, *constraints.workspaceID).Return(test.workspace, nil).Maybe()
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, caller.WorkspaceID).Return(jobWorkspace, nil).Maybe()

			mockGroups.On("GetGroupByID", mock.Anything, "group-ID").Return(jobGroup, nil).Maybe()

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
