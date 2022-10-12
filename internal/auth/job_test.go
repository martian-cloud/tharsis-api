package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestJobCaller_GetSubject(t *testing.T) {
	caller := JobCaller{JobID: "job1"}
	assert.Equal(t, "job1", caller.GetSubject())
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

func TestJobCaller_RequireAccessToNamespace(t *testing.T) {
	workspaceID := "ws1"

	// Test cases
	tests := []struct {
		name                            string
		workspace                       *models.Workspace
		requiredAccessLevel             models.Role
		requiredNamespace               string
		requiredNamespaceWorkspace      *models.Workspace
		requiredNamespaceWorkspaceError error
		expectErrorMsg                  string
	}{
		{
			name: "job has viewer access to workspace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel:             models.ViewerRole,
			requiredNamespace:               "ns1/ns11",
			requiredNamespaceWorkspace:      &models.Workspace{FullPath: "ns1/ns11"},
			requiredNamespaceWorkspaceError: nil,
		},
		{
			name: "job has viewer access to another workspace under the same root namespace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel:             models.ViewerRole,
			requiredNamespace:               "ns1/ns12",
			requiredNamespaceWorkspace:      &models.Workspace{FullPath: "ns1/ns12"},
			requiredNamespaceWorkspaceError: nil,
		},
		{
			name: "access denied because job is associated with workspace under a different root namespace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel:             models.ViewerRole,
			requiredNamespace:               "ns2",
			requiredNamespaceWorkspace:      &models.Workspace{FullPath: "ns2"},
			requiredNamespaceWorkspaceError: nil,
			expectErrorMsg:                  resourceNotFoundErrorMsg,
		},
		{
			name: "access denied because the namespace isn't a workspace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel:             models.ViewerRole,
			requiredNamespace:               "ns2",
			requiredNamespaceWorkspace:      nil,
			requiredNamespaceWorkspaceError: nil,
			expectErrorMsg:                  resourceNotFoundErrorMsg,
		},
		{
			name: "access denied because the namespace isn't a workspace with error",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel:             models.ViewerRole,
			requiredNamespace:               "ns2",
			requiredNamespaceWorkspace:      nil,
			requiredNamespaceWorkspaceError: errors.New("doesn't exist"),
			expectErrorMsg:                  resourceNotFoundErrorMsg,
		},
		{
			name: "access denied because job only have viewer access on workspace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1",
			},
			requiredAccessLevel: models.DeployerRole,
			requiredNamespace:   "ns1",
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockWorkspaces.On("GetWorkspaceByFullPath", mock.Anything, test.requiredNamespace).Return(test.requiredNamespaceWorkspace, test.requiredNamespaceWorkspaceError)
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(test.workspace, nil)

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: workspaceID, RunID: "run1"}

			err := caller.RequireAccessToNamespace(WithCaller(ctx, &caller), test.requiredNamespace, test.requiredAccessLevel)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireAccessToGroup(t *testing.T) {
	caller := JobCaller{JobID: "job1"}
	err := caller.RequireAccessToGroup(WithCaller(context.Background(), &caller), "group1", models.ViewerRole)
	assert.EqualError(t, err, resourceNotFoundErrorMsg)
}

func TestJobCaller_RequireAccessToInheritedGroupResource(t *testing.T) {
	workspaceID := "ws1"
	groupID := "group1"

	// Test cases
	tests := []struct {
		name           string
		workspace      *models.Workspace
		group          *models.Group
		expectErrorMsg string
	}{
		{
			name: "workspace is in requested group so access it granted",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11/ns111",
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1",
			},
		},
		{
			name: "workspace is not in requested group",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns2",
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1",
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name: "group not found",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns2",
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name: "workspace not found",
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1",
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(test.workspace, nil)
			mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
				Groups:     &mockGroups,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: workspaceID, RunID: "run1"}

			err := caller.RequireAccessToInheritedGroupResource(WithCaller(ctx, &caller), groupID)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireAccessToInheritedNamespaceResource(t *testing.T) {
	workspaceID := "ws1"

	// Test cases
	tests := []struct {
		name           string
		workspace      *models.Workspace
		namespace      string
		expectErrorMsg string
	}{
		{
			name: "workspace is in requested group so access it granted",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11/ns111",
			},
			namespace: "ns1",
		},
		{
			name: "workspace is not in requested group",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns2",
			},
			namespace:      "ns1",
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "workspace not found",
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(test.workspace, nil)

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: workspaceID, RunID: "run1"}

			err := caller.RequireAccessToInheritedNamespaceResource(WithCaller(ctx, &caller), test.namespace)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireAccessToWorkspace(t *testing.T) {
	jobWorkspaceID := "111-11-111"

	// Test cases
	tests := []struct {
		name                string
		jobWorkspacePath    string
		workspaceID         string
		workspace           *models.Workspace
		workspaceError      error
		requiredAccessLevel models.Role
		expectErrorMsg      string
	}{
		{
			name:                "deny access because access level is greater than viewer",
			workspaceID:         jobWorkspaceID,
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
		{
			name:                "grant viewer access because job is associated with workspace",
			workspaceID:         jobWorkspaceID,
			requiredAccessLevel: models.ViewerRole,
		},
		{
			name:                "deny because workspace does not exist",
			workspaceID:         "222-22-222",
			workspace:           nil,
			workspaceError:      nil,
			requiredAccessLevel: models.ViewerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
		{
			name:                "deny because workspace returned an error",
			workspaceID:         "222-22-222",
			workspace:           nil,
			workspaceError:      errors.New("something went wrong"),
			requiredAccessLevel: models.ViewerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
		{
			name:                "deny access because workspace is not under the same root namespace",
			jobWorkspacePath:    "rg1/ws1",
			workspaceID:         "222-22-222",
			workspace:           &models.Workspace{FullPath: "rg2/ws2"},
			requiredAccessLevel: models.ViewerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
		{
			name:                "grant viewer access because workspace is under the same root namespace",
			jobWorkspacePath:    "rg1/ws1",
			workspaceID:         "222-22-222",
			workspace:           &models.Workspace{FullPath: "rg1/ws2"},
			requiredAccessLevel: models.ViewerRole,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.workspaceID).Return(test.workspace, test.workspaceError).Once()
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspaceID).Return(&models.Workspace{FullPath: test.jobWorkspacePath}, nil).Once()

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: jobWorkspaceID, RunID: "run1"}

			err := caller.RequireAccessToWorkspace(WithCaller(ctx, &caller), test.workspaceID, test.requiredAccessLevel)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireViewerAccessToGroups(t *testing.T) {
	caller := JobCaller{JobID: "job1"}
	groups := []models.Group{{Metadata: models.ResourceMetadata{ID: "1"}}}
	err := caller.RequireViewerAccessToGroups(WithCaller(context.Background(), &caller), groups)
	assert.EqualError(t, err, resourceNotFoundErrorMsg)
}

func TestJobCaller_RequireViewerAccessToWorkspaces(t *testing.T) {
	jobWorkspace := models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "111-11-111",
		},
		FullPath: "rg1/ws1",
	}

	// Test cases
	tests := []struct {
		name               string
		expectErrorMsg     string
		requiredWorkspaces []models.Workspace
	}{
		{
			name:               "grant viewer access because job is associated with workspace",
			requiredWorkspaces: []models.Workspace{jobWorkspace},
		},
		{
			name:               "deny access because workspace is not under the same root namespace",
			requiredWorkspaces: []models.Workspace{jobWorkspace, {Metadata: models.ResourceMetadata{ID: "222-22-222"}, FullPath: "rg2/ws1"}},
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name:               "grant viewer access because workspace is under the same root namespace",
			requiredWorkspaces: []models.Workspace{jobWorkspace, {Metadata: models.ResourceMetadata{ID: "222-22-222"}, FullPath: "rg1/ws2"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspace.Metadata.ID).Return(&jobWorkspace, nil).Once()

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: jobWorkspace.Metadata.ID, RunID: "run1"}

			err := caller.RequireViewerAccessToWorkspaces(WithCaller(ctx, &caller), test.requiredWorkspaces)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireViewerAccessToNamespaces(t *testing.T) {
	jobWorkspace := models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "111-11-111",
		},
		FullPath: "rg1/ws1",
	}

	// Test cases
	tests := []struct {
		name               string
		expectErrorMsg     string
		requiredNamespaces []string
		workspaces         []models.Workspace
	}{
		{
			name:               "grant viewer access because job is associated with workspace",
			requiredNamespaces: []string{jobWorkspace.FullPath},
			workspaces:         []models.Workspace{{FullPath: jobWorkspace.FullPath}},
		},
		{
			name:               "deny access because workspace is not under the same root namespace",
			requiredNamespaces: []string{jobWorkspace.FullPath, "rg2/ws1"},
			workspaces:         []models.Workspace{{FullPath: jobWorkspace.FullPath}, {FullPath: "rg2/ws1"}},
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name:               "grant viewer access because workspace is under the same root namespace",
			requiredNamespaces: []string{jobWorkspace.FullPath, "rg1/ws2"},
			workspaces:         []models.Workspace{{FullPath: jobWorkspace.FullPath}, {FullPath: "rg1/ws2"}},
		},
		{
			name:               "deny access because namespace is not found",
			requiredNamespaces: []string{jobWorkspace.FullPath, "rg1/ws2"},
			workspaces:         []models.Workspace{{FullPath: jobWorkspace.FullPath}, {FullPath: "rg2/ws2"}},
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			workspaceMap := map[string]*models.Workspace{}

			for _, ws := range test.workspaces {
				wsCopy := ws
				workspaceMap[ws.FullPath] = &wsCopy
			}

			for _, ns := range test.requiredNamespaces {
				mockWorkspaces.On("GetWorkspaceByFullPath", mock.Anything, ns).Return(workspaceMap[ns], nil).Once()
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, jobWorkspace.Metadata.ID).Return(&jobWorkspace, nil).Once()

			dbClient := db.Client{
				Workspaces: &mockWorkspaces,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: "job1", WorkspaceID: jobWorkspace.Metadata.ID, RunID: "run1"}

			err := caller.RequireViewerAccessToNamespaces(WithCaller(ctx, &caller), test.requiredNamespaces)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireRunWriteAccess(t *testing.T) {
	runID := "run1"

	// Test cases
	tests := []struct {
		name           string
		runID          string
		expectErrorMsg string
	}{
		{
			name:  "grant access because job is associated with run",
			runID: runID,
		},
		{
			name:           "deny access because run is not associated with job",
			runID:          "run2",
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			caller := JobCaller{JobID: "job1", RunID: runID}

			err := caller.RequireRunWriteAccess(WithCaller(ctx, &caller), test.runID)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequirePlanWriteAccess(t *testing.T) {
	runID := "run1"
	planID := "plan1"
	jobID := "job1"

	// Test cases
	tests := []struct {
		name           string
		run            *models.Run
		job            *models.Job
		expectErrorMsg string
	}{
		{
			name: "job has permission to write to plan",
			run:  &models.Run{Metadata: models.ResourceMetadata{ID: runID}, PlanID: planID},
			job:  &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
		},
		{
			name:           "run plan ID does not match",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, PlanID: "plan2"},
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "job ID does not match",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, PlanID: planID},
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: "job2"}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "run not found",
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "latest job not found",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, PlanID: planID},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.MockRuns{}
			mockRuns.Test(t)

			mockJobs := db.MockJobs{}
			mockJobs.Test(t)

			mockRuns.On("GetRun", mock.Anything, runID).Return(test.run, nil)
			mockJobs.On("GetLatestJobByType", mock.Anything, runID, models.JobPlanType).Return(test.job, nil)

			dbClient := db.Client{
				Runs: &mockRuns,
				Jobs: &mockJobs,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: jobID, RunID: runID}

			err := caller.RequirePlanWriteAccess(WithCaller(ctx, &caller), planID)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireApplyWriteAccess(t *testing.T) {
	runID := "run1"
	applyID := "apply1"
	jobID := "job1"

	// Test cases
	tests := []struct {
		name           string
		run            *models.Run
		job            *models.Job
		expectErrorMsg string
	}{
		{
			name: "job has permission to write to apply",
			run:  &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ApplyID: applyID},
			job:  &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
		},
		{
			name:           "run apply ID does not match",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ApplyID: "apply2"},
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "job ID does not match",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ApplyID: applyID},
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: "job2"}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "run not found",
			job:            &models.Job{Metadata: models.ResourceMetadata{ID: jobID}},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "latest job not found",
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ApplyID: applyID},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.MockRuns{}
			mockRuns.Test(t)

			mockJobs := db.MockJobs{}
			mockJobs.Test(t)

			mockRuns.On("GetRun", mock.Anything, runID).Return(test.run, nil)
			mockJobs.On("GetLatestJobByType", mock.Anything, runID, models.JobApplyType).Return(test.job, nil)

			dbClient := db.Client{
				Runs: &mockRuns,
				Jobs: &mockJobs,
			}

			caller := JobCaller{dbClient: &dbClient, JobID: jobID, RunID: runID}

			err := caller.RequireApplyWriteAccess(WithCaller(ctx, &caller), applyID)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireJobWriteAccess(t *testing.T) {
	jobID := "job1"

	// Test cases
	tests := []struct {
		name           string
		jobID          string
		expectErrorMsg string
	}{
		{
			name:  "grant access because job ID matches",
			jobID: jobID,
		},
		{
			name:           "deny access because job ID does not match",
			jobID:          "job2",
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			caller := JobCaller{JobID: jobID}

			err := caller.RequireJobWriteAccess(WithCaller(ctx, &caller), test.jobID)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJobCaller_RequireTeamCreateAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestJobCaller_RequireTeamUpdateAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireTeamUpdateAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

func TestJobCaller_RequireTeamDeleteAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

func TestJobCaller_RequireUserCreateAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestJobCaller_RequireUserUpdateAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "a-fake-user-id"))
}

func TestJobCaller_RequireUserDeleteAccess(t *testing.T) {
	caller := JobCaller{}
	assert.NotNil(t, caller.RequireUserDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-user-id"))
}
