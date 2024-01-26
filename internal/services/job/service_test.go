package job

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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

			mockRunners.On("GetRunnerByID", mock.Anything, mock.Anything).Return(test.runner, nil)

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

			job, err := jobService.getNextAvailableJob(ctx, test.runner.Metadata.ID)
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

func TestSubscribeToJobs(t *testing.T) {
	// Test cases
	tests := []struct {
		authError      error
		input          *SubscribeToJobsInput
		name           string
		expectErrCode  errors.CodeType
		runner         *models.Runner
		sendEvents     []Event
		expectedEvents []Event
		isAdmin        bool
	}{
		{
			name: "subscribe to job events for a workspace",
			input: &SubscribeToJobsInput{
				WorkspaceID: ptr.String("workspace1"),
			},
			sendEvents: []Event{
				{
					Job: &models.Job{
						Metadata:    models.ResourceMetadata{ID: "job1"},
						WorkspaceID: "workspace1",
					},
				},
				{
					Job: &models.Job{
						Metadata:    models.ResourceMetadata{ID: "job2"},
						WorkspaceID: "workspace2",
					},
				},
			},
			expectedEvents: []Event{
				{
					Job: &models.Job{
						Metadata:    models.ResourceMetadata{ID: "job1"},
						WorkspaceID: "workspace1",
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name: "not authorized to subscribe to job events for a workspace",
			input: &SubscribeToJobsInput{
				WorkspaceID: ptr.String("workspace1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to job events for an group runner",
			input: &SubscribeToJobsInput{
				RunnerID: ptr.String("runner1"),
			},
			sendEvents: []Event{
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job1"},
						RunnerID: ptr.String("runner1"),
					},
				},
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job2"},
						RunnerID: ptr.String("runner2"),
					},
				},
			},
			expectedEvents: []Event{
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job1"},
						RunnerID: ptr.String("runner1"),
					},
					Action: "UPDATE",
				},
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
		},
		{
			name: "not authorized to subscribe to job events for an group runner",
			input: &SubscribeToJobsInput{
				RunnerID: ptr.String("runner1"),
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to job events for a shared runner",
			input: &SubscribeToJobsInput{
				RunnerID: ptr.String("runner1"),
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name: "not authorized to subscribe to job events for a shared runner",
			input: &SubscribeToJobsInput{
				RunnerID: ptr.String("runner1"),
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name:    "subscribe to all job events",
			input:   &SubscribeToJobsInput{},
			isAdmin: true,
			sendEvents: []Event{
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job1"},
						RunnerID: ptr.String("runner1"),
					},
				},
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job2"},
						RunnerID: ptr.String("runner2"),
					},
				},
			},
			expectedEvents: []Event{
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job1"},
						RunnerID: ptr.String("runner1"),
					},
					Action: "UPDATE",
				},
				{
					Job: &models.Job{
						Metadata: models.ResourceMetadata{ID: "job2"},
						RunnerID: ptr.String("runner2"),
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name:          "not authorized to subscribe to all job events",
			input:         &SubscribeToJobsInput{},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockJobs := db.NewMockJobs(t)
			mockEvents := db.NewMockEvents(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			if test.input.RunnerID != nil {
				mockRunners.On("GetRunnerByID", mock.Anything, *test.input.RunnerID).Return(test.runner, nil)
			}

			if test.input.WorkspaceID != nil {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewWorkspacePermission, mock.Anything).
					Return(test.authError)
			} else if test.input.RunnerID != nil {
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(test.authError).Maybe()
				mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()
			} else {
				mockCaller.On("IsAdmin").Return(test.isAdmin)
			}

			for _, e := range test.sendEvents {
				mockJobs.On("GetJobByID", mock.Anything, e.Job.Metadata.ID).Return(e.Job, nil).Maybe()
			}

			dbClient := db.Client{
				Runners: mockRunners,
				Jobs:    mockJobs,
				Events:  mockEvents,
			}

			logger, _ := logger.NewForTest()
			eventManager := events.NewEventManager(&dbClient, logger)
			eventManager.Start(ctx)

			service := &service{
				dbClient:     &dbClient,
				eventManager: eventManager,
				logger:       logger,
			}

			events, err := service.SubscribeToJobs(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*Event{}

			go func() {
				for _, e := range test.sendEvents {
					mockEventChannel <- db.Event{
						Table:  "jobs",
						Action: "UPDATE",
						ID:     e.Job.Metadata.ID,
					}
				}
			}()

			if len(test.expectedEvents) > 0 {
				for e := range events {
					eCopy := e
					receivedEvents = append(receivedEvents, eCopy)

					if len(receivedEvents) == len(test.expectedEvents) {
						break
					}
				}
			}

			require.Equal(t, len(test.expectedEvents), len(receivedEvents))
			for i, e := range test.expectedEvents {
				assert.Equal(t, e, *receivedEvents[i])
			}
		})
	}
}

func TestGetJobs(t *testing.T) {
	workspaceID := "ws1"
	runnerID := "r1"
	groupID := "g1"

	sampleJob := models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job1",
		},
		WorkspaceID: workspaceID,
		RunnerID:    &runnerID,
	}

	type testCase struct {
		authError             error
		workspaceID           *string
		runnerID              *string
		name                  string
		injectRunner          *models.Runner
		injectRunnerPermError error
		expectErrorCode       errors.CodeType
		expectJob             bool
		isAdmin               bool
	}

	tests := []testCase{
		{
			name:        "non admin should be able to get jobs for a workspace",
			workspaceID: &workspaceID,
			expectJob:   true,
		},
		{
			name:      "admin should be able to get jobs for any workspace/runner",
			expectJob: true,
			isAdmin:   true,
		},
		{
			name:            "non admin does not have access to workspace",
			workspaceID:     &workspaceID,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "only admin can get jobs for any workspace/runner",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:     "non admin should be able to get jobs for a runner",
			runnerID: &runnerID,
			injectRunner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: &groupID,
			},
			expectJob: true,
		},
		{
			name:            "non admin should be able to get jobs for a runner except that the runner does not exist",
			runnerID:        &runnerID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:     "non admin does not have access to runner",
			runnerID: &runnerID,
			injectRunner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: &groupID,
			},
			injectRunnerPermError: errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:       errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockJobs := db.NewMockJobs(t)
			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)

			if test.workspaceID != nil {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewWorkspacePermission, mock.Anything).
					Return(test.authError)
			}
			if test.runnerID != nil {
				mockRunners.On("GetRunnerByID", mock.Anything, mock.Anything).
					Return(test.injectRunner, nil)
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
					Return(test.injectRunnerPermError).Maybe()
			}
			mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()

			if test.authError == nil {
				dbInput := &db.GetJobsInput{
					Filter: &db.JobFilter{
						WorkspaceID: test.workspaceID,
						RunnerID:    test.runnerID,
					},
				}

				dbResult := &db.JobsResult{Jobs: []models.Job{}}

				if test.expectJob {
					dbResult.Jobs = append(dbResult.Jobs, sampleJob)
				}

				mockJobs.On("GetJobs", mock.Anything, dbInput).Return(dbResult, nil).Maybe()
			}

			dbClient := &db.Client{
				Jobs:    mockJobs,
				Runners: mockRunners,
			}

			jobService := service{
				dbClient: dbClient,
			}

			jobsResult, err := jobService.GetJobs(auth.WithCaller(ctx, mockCaller),
				&GetJobsInput{
					WorkspaceID: test.workspaceID,
					RunnerID:    test.runnerID,
				})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, jobsResult)
			assert.Equal(t, []models.Job{sampleJob}, jobsResult.Jobs)
		})
	}
}
