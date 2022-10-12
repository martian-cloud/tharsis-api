//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestListen(t *testing.T) {

	// Context with cancel.
	ctx, cancel := context.WithCancel(context.Background())
	// The defer cancel() call is a few lines down.

	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Note: This must be stated _AFTER_ stating the closing of the test client,
	// so that at execution time the context gets cancelled first.
	// If done in the usual order, the whole test sequence hangs.
	defer cancel()

	warmupGroups, err := createWarmupItemsForEvents(ctx, testClient, standardWarmupGroupForEvents)
	require.Nil(t, err)

	jobDuration := int32((time.Hour * 12).Minutes())
	var createdWorkspace *models.Workspace
	var updatedWorkspace *models.Workspace

	chEvent, chFatal := testClient.client.Events.Listen(ctx)

	type testCase struct {
		action         func() error
		name           string
		expectedEvents []Event
	}

	testCases := []testCase{

		{
			name: "create workspace",
			action: func() error {
				createdWorkspace, err = testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
					Name:           "empty-workspace",
					GroupID:        warmupGroups[0].Metadata.ID,
					Description:    "this is an almost empty workspace",
					MaxJobDuration: &jobDuration,
					CreatedBy:      "empty-workspace-creator",
				})
				return err
			},
			expectedEvents: []Event{
				{
					Table:  "workspaces",
					Action: "INSERT",
				},
			},
		},

		{
			name: "update workspace",
			action: func() error {
				toUpdate := *createdWorkspace
				toUpdate.Description = "this is an updated description"
				updatedWorkspace, err = testClient.client.Workspaces.UpdateWorkspace(ctx, &toUpdate)
				return err
			},
			expectedEvents: []Event{
				{
					Table:  "workspaces",
					Action: "UPDATE",
				},
			},
		},

		{
			name: "delete workspace",
			action: func() error {
				err := testClient.client.Workspaces.DeleteWorkspace(ctx, updatedWorkspace)
				return err
			},
			expectedEvents: []Event{
				{
					Table:  "workspaces",
					Action: "DELETE",
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
			for {
				select {
				case gotEvent := <-chEvent:
					actualEvents = append(actualEvents, gotEvent)
				case gotFatal := <-chFatal:
					fatalErrors = append(fatalErrors, gotFatal)
				}

				if len(actualEvents) >= expectedCount || len(fatalErrors) > 0 {
					break
				}
			}

			// Check for fatal errors.
			assert.Zero(t, len(fatalErrors))

			// Check errors and events.
			assert.Equal(t, expectedCount, len(actualEvents))
			for ix := range test.expectedEvents {
				assert.Equal(t, test.expectedEvents[ix].Table, actualEvents[ix].Table)
				assert.Equal(t, test.expectedEvents[ix].Action, actualEvents[ix].Action)
				assert.Equal(t, createdWorkspace.Metadata.ID, actualEvents[ix].ID)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupForEvents = []models.Group{
	{
		Description: "top level group 0 for testing workspace functions",
		FullPath:    "top-level-group-0-for-workspaces",
		CreatedBy:   "someone-1",
	},
}

// createWarmupItemsForEvents creates some warmup groups for a test.
// The warmup groups and workspaces to create can be standard or otherwise.
func createWarmupItemsForEvents(ctx context.Context, testClient *testClient,
	newGroups []models.Group) ([]models.Group, error) {

	resultGroups, _, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, err
	}

	return resultGroups, nil
}

// The End.
