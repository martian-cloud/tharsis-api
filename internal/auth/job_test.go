package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
		expect      error
		run         *models.Run
		job         *models.Job
		name        string
		workspace   *models.Workspace
		perms       permissions.Permission
		constraints []func(*constraints)
	}{
		{
			name:        "job is associated with workspace",
			perms:       permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(jobWorkspace.Metadata.ID)},
		},
		{
			name:        "requested workspace is under same root namespace as job's workspace",
			workspace:   &models.Workspace{FullPath: "a/ws-2"},
			perms:       permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID("ws2")},
		},
		{
			name:        "access denied because workspace is not under same root namespace",
			workspace:   &models.Workspace{FullPath: "b/ws-2"},
			perms:       permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID("ws2")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because workspace doesn't exist",
			perms:       permissions.ViewWorkspacePermission,
			constraints: []func(*constraints){WithWorkspaceID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "job is requesting access to itself",
			perms:       permissions.UpdateJobPermission,
			constraints: []func(*constraints){WithJobID(caller.JobID)},
		},
		{
			name:        "access denied because job is requesting access to another job",
			perms:       permissions.UpdateJobPermission,
			constraints: []func(*constraints){WithJobID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "job has permission to write to plan",
			run:         &models.Run{PlanID: "plan1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       permissions.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
		},
		{
			name:        "access denied because requested plan ID does not match run plan ID",
			run:         &models.Run{PlanID: "plan1"},
			perms:       permissions.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because job IDs do not match",
			run:         &models.Run{PlanID: "plan1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: invalid}},
			perms:       permissions.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because run doesn't exist",
			perms:       permissions.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because latest plan job doesn't exist",
			run:         &models.Run{PlanID: "plan1"},
			perms:       permissions.UpdatePlanPermission,
			constraints: []func(*constraints){WithPlanID("plan1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "job has permission to write to apply",
			run:         &models.Run{ApplyID: "apply1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: caller.JobID}},
			perms:       permissions.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
		},
		{
			name:        "access denied because requested apply ID does not match run apply ID",
			run:         &models.Run{ApplyID: "apply1"},
			perms:       permissions.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID(invalid)},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because job IDs do not match",
			run:         &models.Run{ApplyID: "apply1"},
			job:         &models.Job{Metadata: models.ResourceMetadata{ID: invalid}},
			perms:       permissions.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because run doesn't exist",
			perms:       permissions.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:        "access denied because latest apply job doesn't exist",
			run:         &models.Run{ApplyID: "apply1"},
			perms:       permissions.UpdateApplyPermission,
			constraints: []func(*constraints){WithApplyID("apply1")},
			expect:      authorizationError(ctx, false),
		},
		{
			name:   "access denied because no permissions specified",
			expect: authorizationError(ctx, false),
		},
		{
			name:   "access denied because permission is never available to caller",
			perms:  permissions.CreateWorkspacePermission,
			expect: authorizationError(ctx, false),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRuns := db.NewMockRuns(t)
			mockJobs := db.NewMockJobs(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			constraints := getConstraints(test.constraints...)

			stage := models.JobPlanType
			if constraints.applyID != nil {
				stage = models.JobApplyType
			}

			mockRuns.On("GetRun", mock.Anything, caller.RunID).Return(test.run, nil).Maybe()

			mockJobs.On("GetLatestJobByType", mock.Anything, caller.RunID, stage).Return(test.job, nil).Maybe()

			if constraints.workspaceID != nil && caller.WorkspaceID != *constraints.workspaceID {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, *constraints.workspaceID).Return(test.workspace, nil)
			}

			if len(constraints.namespacePaths) > 0 {
				for _, p := range constraints.namespacePaths {
					mockWorkspaces.On("GetWorkspaceByFullPath", mock.Anything, p).Return(test.workspace, nil)
				}
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, caller.WorkspaceID).Return(jobWorkspace, nil).Maybe()

			caller.dbClient = &db.Client{
				Runs:       mockRuns,
				Jobs:       mockJobs,
				Workspaces: mockWorkspaces,
			}

			assert.Equal(t, test.expect, caller.RequirePermission(ctx, test.perms, test.constraints...))
		})
	}
}

func TestJobCaller_RequireInheritedPermissions(t *testing.T) {
	invalid := "invalid"

	caller := JobCaller{WorkspaceID: "ws1"}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expect       error
		workspace    *models.Workspace
		group        *models.Group
		name         string
		resourceType permissions.ResourceType
		constraints  []func(*constraints)
	}{
		{
			name:         "workspace is in requested group",
			workspace:    &models.Workspace{FullPath: "a/ws1"},
			group:        &models.Group{FullPath: "a"},
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithGroupID("group1")},
		},
		{
			name:         "access denied because workspace is not in requested group",
			workspace:    &models.Workspace{FullPath: "b/ws1"},
			group:        &models.Group{FullPath: "a"},
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithGroupID("group1")},
			expect:       authorizationError(ctx, false),
		},
		{
			name:         "access denied because workspace not found",
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithGroupID(invalid)},
			expect:       authorizationError(ctx, false),
		},
		{
			name:         "access denied because group not found",
			workspace:    &models.Workspace{},
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithGroupID(invalid)},
			expect:       authorizationError(ctx, false),
		},
		{
			name:         "workspace is in requested namespace path",
			workspace:    &models.Workspace{FullPath: "a/ws1"},
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithNamespacePath("a")},
		},
		{
			name:         "access denied because workspace not found",
			resourceType: permissions.ManagedIdentityResourceType,
			constraints:  []func(*constraints){WithNamespacePath("a")},
			expect:       authorizationError(ctx, false),
		},
		{
			name:         "access denied because required constraints not provided",
			resourceType: permissions.ManagedIdentityResourceType,
			expect:       errMissingConstraints,
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
			assert.Equal(t, test.expect, caller.RequireAccessToInheritableResource(ctx, test.resourceType, test.constraints...))
		})
	}
}
