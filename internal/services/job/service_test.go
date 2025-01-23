package job

import (
	"context"
	"encoding/json"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestClaimJob(t *testing.T) {
	jobID := "job-1"
	runnerID := "runner-1"
	workspaceID := "workspace-1"
	groupID := "group-1"
	token := "job-token"

	sampleQueuedJob := models.Job{
		Metadata: models.ResourceMetadata{
			ID: jobID,
		},
		WorkspaceID: workspaceID,
	}

	type testCase struct {
		existingRunner         *models.Runner
		expectResponse         *ClaimJobResponse
		name                   string
		authError              error
		expectErrorCode        errors.CodeType
		expectGetRunnersInput  *db.GetRunnersInput
		injectGetRunnersResult *db.RunnersResult
	}

	testCases := []testCase{
		{
			name: "successfully claim a job with shared runner",
			existingRunner: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID: runnerID,
				},
				Type: models.SharedRunnerType,
				Name: "shared-runner",
			},
			expectGetRunnersInput: &db.GetRunnersInput{
				Filter: &db.RunnerFilter{
					RunnerName: &runnerID,
				},
			},
			injectGetRunnersResult: &db.RunnersResult{
				Runners: []models.Runner{
					{
						Metadata: models.ResourceMetadata{
							ID: runnerID,
						},
						Type: models.SharedRunnerType,
						Name: "shared-runner",
					},
				},
			},
			expectResponse: &ClaimJobResponse{
				JobID: jobID,
				Token: token,
			},
		},
		{
			name: "successfully claim a job with an group runner",
			existingRunner: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID: runnerID,
				},
				Type:         models.GroupRunnerType,
				Name:         "group-runner",
				ResourcePath: "group-1/group-runner",
				GroupID:      &groupID,
			},
			expectGetRunnersInput: &db.GetRunnersInput{
				Filter: &db.RunnerFilter{
					RunnerName: &runnerID,
				},
			},
			injectGetRunnersResult: &db.RunnersResult{
				Runners: []models.Runner{
					{
						Metadata: models.ResourceMetadata{
							ID: runnerID,
						},
						Type:         models.GroupRunnerType,
						Name:         "group-runner",
						ResourcePath: "group-1/group-runner",
					},
				},
			},
			expectResponse: &ClaimJobResponse{
				JobID: jobID,
				Token: token,
			},
		},
		{
			name:            "runner not found",
			expectErrorCode: errors.ENotFound,
			expectGetRunnersInput: &db.GetRunnersInput{
				Filter: &db.RunnerFilter{
					RunnerName: &runnerID,
				},
			},
			injectGetRunnersResult: &db.RunnersResult{
				Runners: []models.Runner{},
			},
		},
		{
			name: "subject does not have permissions to claim job",
			existingRunner: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID: runnerID,
				},
				Type: models.SharedRunnerType,
				Name: "shared-runner",
			},
			expectGetRunnersInput: &db.GetRunnersInput{
				Filter: &db.RunnerFilter{
					RunnerName: &runnerID,
				},
			},
			injectGetRunnersResult: &db.RunnersResult{
				Runners: []models.Runner{
					{
						Metadata: models.ResourceMetadata{
							ID: runnerID,
						},
						Type: models.SharedRunnerType,
						Name: "shared-runner",
					},
				},
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockJobs := db.NewMockJobs(t)
			mockRunners := db.NewMockRunners(t)
			mockCaller := auth.NewMockCaller(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockTransactions := db.NewMockTransactions(t)
			mockJWSProvider := jws.NewMockProvider(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).
				Return(test.existingRunner, nil).Maybe()

			mockCaller.On("RequirePermission", mock.Anything, permissions.ClaimJobPermission, mock.Anything).
				Return(test.authError).Maybe()

			mockRunners.On("GetRunners", mock.Anything, test.expectGetRunnersInput).
				Return(test.injectGetRunnersResult, nil)

			if test.expectErrorCode == "" {
				// Mock jobs
				sortBy := db.JobSortableFieldCreatedAtAsc
				jobQueued := models.JobQueued
				jobsInput := &db.GetJobsInput{
					Sort: &sortBy,
					Filter: &db.JobFilter{
						JobStatus: &jobQueued,
						TagFilter: &db.JobTagFilter{TagSuperset: []string{}, ExcludeUntaggedJobs: ptr.Bool(true)},
					},
				}
				mockJobs.On("GetJobs", mock.Anything, jobsInput).Return(&db.JobsResult{Jobs: []models.Job{sampleQueuedJob}}, nil)

				mockJobs.On("GetJobCountForRunner", mock.Anything, runnerID).Return(1, nil)

				mockJobs.On("UpdateJob", mock.Anything, mock.Anything).Return(&sampleQueuedJob, nil)

				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).
					Return(&models.Workspace{
						Locked:   false,
						FullPath: "group-1/workspace-1",
					}, nil)

				// These are for the transaction opened by RunStateManager.
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
				mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

				mockJobs.On("GetJobByID", mock.Anything, mock.Anything).
					Return(&sampleQueuedJob, nil)

				mockCaller.On("GetSubject").Return("testSubject")

				mockJWSProvider.On("Sign", mock.Anything, mock.Anything).Return([]byte(token), nil)
			}

			dbClient := &db.Client{
				Jobs:         mockJobs,
				Runners:      mockRunners,
				Workspaces:   mockWorkspaces,
				Transactions: mockTransactions,
			}

			identityProvider := auth.NewIdentityProvider(mockJWSProvider, "http://tharsis.domain")

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:        dbClient,
				logger:          logger,
				idp:             identityProvider,
				eventManager:    events.NewEventManager(dbClient, logger),
				runStateManager: state.NewRunStateManager(dbClient, logger),
			}

			actualResponse, err := service.ClaimJob(auth.WithCaller(ctx, mockCaller), runnerID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResponse, actualResponse)
		})
	}
}

func TestGetNextAvailableJob(t *testing.T) {
	// Test cases
	tests := []struct {
		runner                *models.Runner
		workspaceMap          map[string]models.Workspace
		expectJobID           string
		name                  string
		runners               []models.Runner
		queuedJobs            []models.Job
		disallowQueuedJobs    bool
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
			name: "shared runner can get next job even if a group runner exists",
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
			expectJobID: "job1",
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
					Locked:   false,
					FullPath: "group1/group2/group3/ws1",
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
			name: "group runner can get next job even if there is a child group that has a runner",
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
			expectJobID: "job1",
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
		{
			name: "runner should not get next job because tags are not satisfied (job has a tag the runner lacks)",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
				Tags: []string{"extra-runner-tags-do-not-prevent-job-assignment"},
			},
			queuedJobs: []models.Job{
				{
					Metadata:    models.ResourceMetadata{ID: "job1"},
					WorkspaceID: "ws1",
					Tags:        []string{"job-tag-that-the-runner-lacks"},
				},
			},
			disallowQueuedJobs: true,
			runners:            []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: false,
				},
			},
		},
		{
			name: "runner should not get next job because runner does not run untagged jobs",
			runner: &models.Runner{
				Type:            models.SharedRunnerType,
				RunUntaggedJobs: false,
			},
			queuedJobs: []models.Job{
				{Metadata: models.ResourceMetadata{ID: "job1"}, WorkspaceID: "ws1"},
			},
			disallowQueuedJobs: true,
			runners:            []models.Runner{},
			workspaceMap: map[string]models.Workspace{
				"ws1": {
					Locked: false,
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

			// Mimic what the DB layer does with the tag filter.
			returnJobs := test.queuedJobs
			if test.disallowQueuedJobs {
				returnJobs = []models.Job{}
			}
			mockJobs.On("GetJobs", ctx, mock.Anything).Return(&db.JobsResult{
				Jobs: returnJobs,
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
		sendEventData  []*db.JobEventData
		expectedEvents []Event
		isAdmin        bool
	}{
		{
			name: "subscribe to job events for a workspace",
			input: &SubscribeToJobsInput{
				WorkspaceID: ptr.String("workspace1"),
			},
			sendEventData: []*db.JobEventData{
				{
					ID:          "job1",
					WorkspaceID: "workspace1",
				},
				{
					ID:          "job2",
					WorkspaceID: "workspace2",
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
			name: "subscribe to job events for a group runner",
			input: &SubscribeToJobsInput{
				RunnerID: ptr.String("runner1"),
			},
			sendEventData: []*db.JobEventData{
				{
					ID:       "job1",
					RunnerID: ptr.String("runner1"),
				},
				{
					ID:       "job2",
					RunnerID: ptr.String("runner2"),
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
			name: "not authorized to subscribe to job events for a group runner",
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
			sendEventData: []*db.JobEventData{
				{
					ID:       "job1",
					RunnerID: ptr.String("runner1"),
				},
				{
					ID:       "job2",
					RunnerID: ptr.String("runner2"),
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

			for _, e := range test.sendEventData {
				mockJobs.On("GetJobByID", mock.Anything, e.ID).Return(&models.Job{
					Metadata: models.ResourceMetadata{
						ID: e.ID,
					},
					WorkspaceID: e.WorkspaceID,
					RunnerID:    e.RunnerID,
				}, nil).Maybe()
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
				for _, e := range test.sendEventData {
					encoded, err := json.Marshal(e)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "jobs",
						Action: "UPDATE",
						ID:     e.ID,
						Data:   encoded,
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

func TestSubscribeToCancellationEvent(t *testing.T) {
	// Test cases
	tests := []struct {
		authError      error
		input          *CancellationSubscriptionsOptions
		name           string
		expectErrCode  errors.CodeType
		sendEvents     []*CancellationEvent
		expectedEvents []CancellationEvent
		isAdmin        bool
	}{
		{
			name: "subscribe to cancellation events for a job",
			input: &CancellationSubscriptionsOptions{
				JobID: "job1",
			},
			sendEvents: []*CancellationEvent{
				{
					Job: models.Job{
						Metadata: models.ResourceMetadata{
							ID: "job1",
						},
					},
				},
				{
					Job: models.Job{
						Metadata: models.ResourceMetadata{
							ID: "job2",
						},
					},
				},
			},
			expectedEvents: []CancellationEvent{
				{
					Job: models.Job{
						Metadata: models.ResourceMetadata{
							ID: "job1",
						},
						CancelRequested: true,
					},
				},
			},
		},
		{
			name: "not authorized to subscribe to events for a job",
			input: &CancellationSubscriptionsOptions{
				JobID: "job1",
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockEvents := db.NewMockEvents(t)

			mockJobs := db.NewMockJobs(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			mockCaller.On("RequirePermission", mock.Anything, permissions.ViewJobPermission, mock.Anything, mock.Anything).
				Return(test.authError).Maybe()

			// For the call to GetJob outside the wait-for loop.
			mockJobs.On("GetJobByID", mock.Anything, "job1").
				Return(&models.Job{
					Metadata: models.ResourceMetadata{
						ID: "job1",
					},
					CancelRequested: true,
				}, nil).Maybe()

			// For the calls to GetJob inside the wait-for loop.
			for _, e := range test.sendEvents {
				mockJobs.On("GetJobByID", mock.Anything, e.Job.Metadata.ID).
					Return(&models.Job{
						Metadata: models.ResourceMetadata{
							ID: e.Job.Metadata.ID,
						},
						CancelRequested: true,
					}, nil).Maybe()
			}

			dbClient := db.Client{
				Jobs:   mockJobs,
				Events: mockEvents,
			}

			logger, _ := logger.NewForTest()
			eventManager := events.NewEventManager(&dbClient, logger)
			eventManager.Start(ctx)

			service := &service{
				dbClient:     &dbClient,
				eventManager: eventManager,
				logger:       logger,
			}

			events, err := service.SubscribeToCancellationEvent(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			go func() {
				for _, e := range test.sendEvents {
					encoded, err := json.Marshal(e)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "jobs",
						Action: "UPDATE",
						ID:     e.Job.Metadata.ID,
						Data:   encoded,
					}
				}
			}()

			receivedEvents := []*CancellationEvent{}
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
