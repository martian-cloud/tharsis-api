package job

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetNextAvailableJob(t *testing.T) {
	// Test cases
	tests := []struct {
		runner                *models.Runner
		workspaceMap          map[string]models.Workspace
		expectJobID           string
		name                  string
		runners               []models.Runner
		queuedJobs            []models.Job
		currentRunnerJobCount int
	}{
		{
			name: "shared runner should get next job because no group runner exists",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: false,
				},
			},
			expectJobID: "job1",
		},
		{
			name: "shared runner should not get next job because a group runner exists",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{
				{ResourcePath: "a/runner1"},
			},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: false,
				},
			},
		},
		{
			name: "shared runner should not get next job because workspace is locked",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: true,
				},
			},
		},
		{
			name: "shared runner should not get next job because runner is over job limit",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners:               []models.Runner{},
			currentRunnerJobCount: runnerJobsLimit,
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: false,
				},
			},
		},
		{
			name: "group runner should get next job because it's in the same group as the workspace",
			runner: &models.Runner{
				Type:         models.GroupRunnerType,
				ResourcePath: "group1/runner1",
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked:   false,
					FullPath: "group1/ws1",
				},
			},
			expectJobID: "job1",
		},
		{
			name: "group runner should get next job because there are no runners in a child group with higher precedence",
			runner: &models.Runner{
				Type:         models.GroupRunnerType,
				ResourcePath: "group1/runner1",
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{
				{
					Type:         models.GroupRunnerType,
					ResourcePath: "group1/runner1",
				},
			},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked:   false,
					FullPath: "group1/group2/group3/ws1",
				},
			},
			expectJobID: "job1",
		},
		{
			name: "group runner should not get next job because there is a child group that has a runner with higher precedence",
			runner: &models.Runner{
				Type:         models.GroupRunnerType,
				ResourcePath: "group1/runner1",
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{
				{
					Type:         models.GroupRunnerType,
					ResourcePath: "group1/runner1",
				},
				{
					Type:         models.GroupRunnerType,
					ResourcePath: "group1/group2/runner1",
				},
			},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked:   false,
					FullPath: "group1/group2/group3/ws1",
				},
			},
		},
		{
			name: "group runner should not get next job because the workspace is in a different group hierarchy",
			runner: &models.Runner{
				Type:         models.GroupRunnerType,
				ResourcePath: "group1/group2/runner1",
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			runners: []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked:   false,
					FullPath: "group1/group3/ws1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockJobs := db.NewMockJobs(t)
			mockWorkspace := db.NewMockWorkspaces(t)
			mockRunners := db.NewMockRunners(t)

			mockJobs.On("GetJobs", ctx, mock.Anything).Return(&db.JobsResult{
				Jobs: test.queuedJobs,
			}, nil)

			for _, j := range test.queuedJobs {
				ws, ok := test.workspaceMap[j.WorkspaceID]
				if !ok {
					t.Fatalf("workspaceMap is missing workspace with ID %s", j.WorkspaceID)
				}
				mockWorkspace.On("GetWorkspaceByID", ctx, j.WorkspaceID).Return(&ws, nil).Maybe()
			}

			mockRunners.On("GetRunners", ctx, mock.Anything).Return(&db.RunnersResult{
				Runners: test.runners,
			}, nil).Maybe()

			mockJobs.On("GetJobCountForRunner", ctx, test.runner.Metadata.ID).Return(test.currentRunnerJobCount, nil).Maybe()

			logger, _ := logger.NewForTest()
			jobService := service{
				logger: logger,
				dbClient: &db.Client{
					Jobs:       mockJobs,
					Workspaces: mockWorkspace,
					Runners:    mockRunners,
				},
			}

			job, err := jobService.getNextAvailableJob(ctx, test.runner)
			if err != nil {
				t.Fatal(err)
			}

			if test.expectJobID == "" {
				require.Nil(t, job)
			} else {
				require.NotNil(t, job)
				assert.Equal(t, test.expectJobID, job.Metadata.ID)
			}
		})
	}
}
