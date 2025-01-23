//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestListen(t *testing.T) {

	// Context with cancel.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Some common/shared foundational resources.
	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-1",
		Description: "this is a common/shared foundational group",
		CreatedBy:   "test-group-1-creator",
	})
	require.Nil(t, err)
	require.NotNil(t, group1)

	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-1",
		Description:    "this is a common/shared foundational workspace",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "test-workspace-1-creator",
		MaxJobDuration: ptr.Int32(int32(10)),
	})
	require.Nil(t, err)
	require.NotNil(t, workspace1)

	var runnerIDForJobs string
	var jobID *string
	var createdJob *models.Job

	var createdLogStream *models.LogStream

	var runID *string
	var workspaceIDForRuns *string
	var createdRun *models.Run

	var runnerIDForRunnerSessions string
	var createdRunnerSession *models.RunnerSession
	var updatedRunnerSession *models.RunnerSession
	var runnerSessionID string

	listenCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	chEvent, chFatal := testClient.client.Events.Listen(listenCtx)

	type testCase struct {
		action            func() error
		name              string
		expectedEvents    []Event
		expectedEventData []any
	}

	testCases := []testCase{

		{
			name: "create and update job",
			action: func() error {
				runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
					Name:        "runner-for-jobs",
					Description: "runner description for jobs",
					GroupID:     &group1.Metadata.ID,
					Type:        models.GroupRunnerType,
				})
				if err != nil {
					return fmt.Errorf("failed to create runner for job test: %w", err)
				}

				// Save the runner ID for expected results.
				runnerIDForJobs = runner1.Metadata.ID

				run1, rErr := testClient.client.Runs.CreateRun(ctx, &models.Run{
					WorkspaceID: workspace1.Metadata.ID,
					CreatedBy:   "test-run-1-creator",
				})
				require.Nil(t, rErr)
				require.NotNil(t, run1)

				// Save the IDs for expected results, because the checker will check it.
				runID = &run1.Metadata.ID
				workspaceIDForRuns = &run1.WorkspaceID

				var cErr error
				createdJob, cErr = testClient.client.Jobs.CreateJob(ctx, &models.Job{
					WorkspaceID: workspace1.Metadata.ID,
					RunID:       run1.Metadata.ID,
					RunnerID:    &runner1.Metadata.ID,
				})
				if cErr != nil {
					return fmt.Errorf("failed to create job: %w", cErr)
				}
				assert.NotNil(t, createdJob)

				// Save the job ID for expected results.
				jobID = &createdJob.Metadata.ID

				toUpdate := createdJob
				toUpdate.Status = models.JobFinished
				_, uErr := testClient.client.Jobs.UpdateJob(ctx, toUpdate)
				if uErr != nil {
					return fmt.Errorf("failed to update job: %w", uErr)
				}

				return nil
			},
			expectedEvents: []Event{
				{
					Table:  "runners",
					Action: "INSERT",
				},
				{
					Table:  "runs",
					Action: "INSERT",
				},
				{
					Table:  "jobs",
					Action: "INSERT",
				},
				{
					Table:  "jobs",
					Action: "UPDATE",
				},
			},
			expectedEventData: []any{
				struct{}{}, // no need to check the runner
				&RunEventData{
					// IDs will be filled in later.
				},
				&JobEventData{
					// IDs will be filled in later.
					WorkspaceID: workspace1.Metadata.ID,
				},
				&JobEventData{
					// IDs will be filled in later.
					WorkspaceID: workspace1.Metadata.ID,
				},
			},
		},

		{
			name: "create and update log stream",
			action: func() error {
				createdLogStream, err = testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
					Size:      42,
					Completed: false,
				})
				if err != nil {
					return fmt.Errorf("failed to create log stream: %w", err)
				}

				toUpdate := *createdLogStream
				toUpdate.Size = 97
				toUpdate.Completed = true
				_, err = testClient.client.LogStreams.UpdateLogStream(ctx, &toUpdate)
				if err != nil {
					return fmt.Errorf("failed to update log stream: %w", err)
				}

				return nil
			},
			expectedEvents: []Event{
				{
					Table:  "log_streams",
					Action: "INSERT",
				},
				{
					Table:  "log_streams",
					Action: "UPDATE",
				},
			},
			expectedEventData: []any{
				&LogStreamEventData{
					Size:      42,
					Completed: false,
				},
				&LogStreamEventData{
					Size:      97,
					Completed: true,
				},
			},
		},

		{
			name: "create, update, and delete a run",
			action: func() error {
				var cErr error
				createdRun, cErr = testClient.client.Runs.CreateRun(ctx, &models.Run{
					WorkspaceID: workspace1.Metadata.ID,
					CreatedBy:   "run-creator",
				})
				if cErr != nil {
					return fmt.Errorf("failed to create run: %w", cErr)
				}
				assert.NotNil(t, createdRun)

				// Save the run ID for expected results.
				runID = &createdRun.Metadata.ID
				workspaceIDForRuns = &createdRun.WorkspaceID

				toUpdate := createdRun
				toUpdate.Status = models.RunApplied
				_, uErr := testClient.client.Runs.UpdateRun(ctx, toUpdate)
				if uErr != nil {
					return fmt.Errorf("failed to update run: %w", uErr)
				}

				return nil
			},
			expectedEvents: []Event{
				{
					Table:  "runs",
					Action: "INSERT",
				},
				{
					Table:  "runs",
					Action: "UPDATE",
				},
			},
			expectedEventData: []any{
				&RunEventData{
					// IDs will be filled in later.
				},
				&RunEventData{
					// IDs will be filled in later.
				},
			},
		},

		{
			name: "create, update, and delete a runner session",
			action: func() error {
				runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
					Name:        "runner-for-runner-sessions",
					Description: "runner description for runner sessions",
					GroupID:     &group1.Metadata.ID,
					Type:        models.GroupRunnerType,
				})
				if err != nil {
					return fmt.Errorf("failed to create runner for runner session test: %w", err)
				}

				// Save the runner ID for expected results.
				runnerIDForRunnerSessions = runner1.Metadata.ID

				var cErr error
				createdRunnerSession, cErr = testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
					RunnerID:   runner1.Metadata.ID,
					ErrorCount: 4,
				})
				if cErr != nil {
					return fmt.Errorf("failed to create runner session: %w", cErr)
				}

				// Save the ID for expected results.
				runnerSessionID = createdRunnerSession.Metadata.ID

				toUpdate := *createdRunnerSession
				toUpdate.ErrorCount = 76
				var uErr error
				updatedRunnerSession, uErr = testClient.client.RunnerSessions.UpdateRunnerSession(ctx, &toUpdate)
				if uErr != nil {
					return fmt.Errorf("failed to update runner session: %w", uErr)
				}

				dErr := testClient.client.RunnerSessions.DeleteRunnerSession(ctx, updatedRunnerSession)
				if dErr != nil {
					return fmt.Errorf("failed to delete runner session: %w", dErr)
				}

				return nil
			},
			expectedEvents: []Event{
				{
					Table:  "runners",
					Action: "INSERT",
				},
				{
					Table:  "runner_sessions",
					Action: "INSERT",
				},
				{
					Table:  "runner_sessions",
					Action: "UPDATE",
				},
				{
					Table:  "runner_sessions",
					Action: "DELETE",
				},
			},
			expectedEventData: []any{
				struct{}{}, // no need to check the runner
				&RunnerSessionEventData{
					// ID will be filled in later.
					RunnerID: runnerIDForRunnerSessions,
				},
				&RunnerSessionEventData{
					// ID will be filled in later.
					RunnerID: runnerIDForRunnerSessions,
				},
				&RunnerSessionEventData{
					// ID will be filled in later.
					RunnerID: runnerIDForRunnerSessions,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// Execute the action.
			gotActionError := test.action()
			assert.Nil(t, gotActionError)

			// Listen for action errors, fatal errors, and events.
			expectedCount := len(test.expectedEvents)
			actualEvents := []Event{}
			fatalErrors := []error{}

			// Wait for events, errors, or timeout.
			keepWaiting := true
			for keepWaiting {
				select {
				case gotEvent := <-chEvent:
					actualEvents = append(actualEvents, gotEvent)
				case gotFatal := <-chFatal:
					fatalErrors = append(fatalErrors, gotFatal)
				case <-ctx.Done():
					keepWaiting = false
				}

				if len(actualEvents) >= expectedCount || len(fatalErrors) > 0 {
					keepWaiting = false
				}
			}

			// Check for fatal errors.
			assert.Zero(t, len(fatalErrors))

			// Check errors and events.
			assert.Equal(t, expectedCount, len(actualEvents))
			for ix := range test.expectedEvents {
				assert.Equal(t, len(test.expectedEvents), len(test.expectedEventData),
					"test coding error: expectedEvents and expectedEventData must have the same length")
				assert.Equal(t, test.expectedEvents[ix].Table, actualEvents[ix].Table)
				assert.Equal(t, test.expectedEvents[ix].Action, actualEvents[ix].Action)

				switch test.expectedEvents[ix].Table {
				case eventTableJobs:
					typedExpected, ok := test.expectedEventData[ix].(*JobEventData)
					assert.True(t, ok)
					// Must paste in the IDs after the job has been created.
					typedExpected.ID = *jobID
					typedExpected.RunnerID = &runnerIDForJobs

					actualJobEventData, err := actualEvents[ix].ToJobEventData()
					assert.Nil(t, err)

					assert.Equal(t, typedExpected, actualJobEventData)
				case eventTableLogStreams:
					typedExpected, ok := test.expectedEventData[ix].(*LogStreamEventData)
					assert.True(t, ok)

					actualLogStreamEventData, err := actualEvents[ix].ToLogStreamEventData()
					assert.Nil(t, err)

					assert.Equal(t, typedExpected, actualLogStreamEventData)
				case eventTableRuns:
					typedExpected, ok := test.expectedEventData[ix].(*RunEventData)
					assert.True(t, ok)
					// Must paste in the IDs after the run has been created.
					typedExpected.ID = *runID
					typedExpected.WorkspaceID = *workspaceIDForRuns

					actualRunEventData, err := actualEvents[ix].ToRunEventData()
					assert.Nil(t, err)

					assert.Equal(t, typedExpected, actualRunEventData)
				case eventTableRunnerSessions:
					typedExpected, ok := test.expectedEventData[ix].(*RunnerSessionEventData)
					assert.True(t, ok)
					// Must paste in the runner session ID and runner ID after the runner session has been created.
					typedExpected.ID = runnerSessionID
					typedExpected.RunnerID = runnerIDForRunnerSessions

					actualRunnerSessionEventData, err := actualEvents[ix].ToRunnerSessionEventData()
					assert.Nil(t, err)

					assert.Equal(t, typedExpected, actualRunnerSessionEventData)
				}
			}
		})
	}
}
