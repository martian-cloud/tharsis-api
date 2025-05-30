//go:build integration
// +build integration

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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// warmupStateVersionOutputs holds the inputs to and outputs from createWarmupStateVersionOutputs.
type warmupStateVersionOutputs struct {
	groups              []models.Group
	workspaces          []models.Workspace
	runs                []models.Run
	stateVersions       []models.StateVersion
	stateVersionOutputs []models.StateVersionOutput
}

func TestGetStateVersionOutputByID(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(20),
	})
	require.NoError(t, err)

	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "test-user",
	})
	require.NoError(t, err)

	stateVersionOutput, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &models.StateVersionOutput{
		Name:           "test-output",
		Value:          []byte("test-value"),
		Type:           []byte("string"),
		StateVersionID: stateVersion.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectOutput    bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "get state version output by ID",
			id:           stateVersionOutput.Metadata.ID,
			expectOutput: true,
		},
		{
			name: "resource with ID not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid ID will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualOutput, err := testClient.client.StateVersionOutputs.GetStateVersionOutputByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectOutput {
				require.NotNil(t, actualOutput)
				assert.Equal(t, test.id, actualOutput.Metadata.ID)
			} else {
				assert.Nil(t, actualOutput)
			}
		})
	}
}

func TestGetStateVersionOutputByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(20),
	})
	require.NoError(t, err)

	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "test-user",
	})
	require.NoError(t, err)

	stateVersionOutput, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &models.StateVersionOutput{
		Name:           "test-output",
		Value:          []byte("test-value"),
		Type:           []byte("string"),
		StateVersionID: stateVersion.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectOutput    bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "get state version output by TRN",
			trn:          stateVersionOutput.Metadata.TRN,
			expectOutput: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.StateVersionOutputModelType.BuildTRN(workspace.FullPath, stateVersion.GetGlobalID(), "unknown"),
		},
		{
			name:            "state version outputs TRN cannot have less than three parts",
			trn:             types.StateVersionOutputModelType.BuildTRN("unknown"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualOutput, err := testClient.client.StateVersionOutputs.GetStateVersionOutputByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectOutput {
				require.NotNil(t, actualOutput)
				assert.Equal(t,
					types.StateVersionOutputModelType.BuildTRN(workspace.FullPath, stateVersion.GetGlobalID(), stateVersionOutput.Name),
					actualOutput.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualOutput)
			}
		})
	}
}

func TestCreateStateVersionOutput(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupStateVersionOutputs(ctx, testClient, warmupStateVersionOutputs{
		groups:              standardWarmupGroupsForStateVersionOutputs,
		workspaces:          standardWarmupWorkspacesForStateVersionOutputs,
		runs:                standardWarmupRunsForStateVersionOutputs,
		stateVersions:       standardWarmupStateVersionsForStateVersionOutputs,
		stateVersionOutputs: []models.StateVersionOutput{},
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.StateVersionOutput
		expectCreated *models.StateVersionOutput
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		// Duplicates are not prohibited by the DB, so don't do a duplicate test case.
		{
			name: "positive, full",
			toCreate: &models.StateVersionOutput{
				Name:           "positive-full",
				Value:          []byte("positive-full-value"),
				Type:           []byte("positive-full-type"),
				Sensitive:      false,
				StateVersionID: warmupItems.stateVersions[0].Metadata.ID,
			},
			expectCreated: &models.StateVersionOutput{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:           "positive-full",
				Value:          []byte("positive-full-value"),
				Type:           []byte("positive-full-type"),
				Sensitive:      false,
				StateVersionID: warmupItems.stateVersions[0].Metadata.ID,
			},
		},

		{
			name: "non-existent state version ID",
			toCreate: &models.StateVersionOutput{
				Name:           "non-existent-state-version-id",
				Value:          []byte("non-existent-value"),
				Type:           []byte("non-existent-type"),
				StateVersionID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"state_version_outputs\" violates foreign key constraint \"fk_state_version_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective state version ID",
			toCreate: &models.StateVersionOutput{
				Name:           "invalid-state-version-id",
				StateVersionID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareStateVersionOutputs(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestGetStateVersionOutputs(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupStateVersionOutputs(ctx, testClient, warmupStateVersionOutputs{
		groups:              standardWarmupGroupsForStateVersionOutputs,
		workspaces:          standardWarmupWorkspacesForStateVersionOutputs,
		runs:                standardWarmupRunsForStateVersionOutputs,
		stateVersions:       standardWarmupStateVersionsForStateVersionOutputs,
		stateVersionOutputs: standardWarmupStateVersionOutputs,
	})
	createdHigh := currentTime()
	require.Nil(t, err)

	type testCase struct {
		expectMsg                 *string
		name                      string
		searchID                  string
		expectStateVersionOutputs []models.StateVersionOutput
	}

	testCases := []testCase{
		{
			name:                      "positive",
			searchID:                  warmupItems.stateVersions[0].Metadata.ID,
			expectStateVersionOutputs: warmupItems.stateVersionOutputs[0:2],
		},

		{
			name:                      "negative, non-existent state version ID",
			searchID:                  nonExistentID,
			expectStateVersionOutputs: []models.StateVersionOutput{},
			// expect state version outputs to be empty and error to be nil
		},

		{
			name:                      "defective-ID",
			searchID:                  invalidID,
			expectStateVersionOutputs: []models.StateVersionOutput{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualOutputs, err := testClient.client.StateVersionOutputs.GetStateVersionOutputs(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectStateVersionOutputs != nil {
				require.NotNil(t, actualOutputs)
				require.Equal(t, len(test.expectStateVersionOutputs), len(actualOutputs))
				for ix := range test.expectStateVersionOutputs {
					compareStateVersionOutputs(t, &test.expectStateVersionOutputs[ix], &actualOutputs[ix],
						false, &timeBounds{
							createLow:  &createdLow,
							createHigh: &createdHigh,
							updateLow:  &createdLow,
							updateHigh: &createdHigh,
						})
				}
			} else {
				assert.Nil(t, actualOutputs)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForStateVersionOutputs = []models.Group{
	{
		Description: "top level group 0 for testing state version output functions",
		FullPath:    "top-level-group-0-for-state-version-outputs",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
// Please note: The createWarmupStateVersionOutputs function supports only a single workspace.
var standardWarmupWorkspacesForStateVersionOutputs = []models.Workspace{
	{
		Description: "workspace 0 for testing state version output functions",
		FullPath:    "top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
		CreatedBy:   "someone-w0",
	},
}

// Standard warmup run(s) for tests in this module
// The ID fields will be replaced by the ID(s) during the create function.
var standardWarmupRunsForStateVersionOutputs = []models.Run{
	{
		WorkspaceID: "top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
		Comment:     "standard warmup run 0 for testing state version outputs",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
		Comment:     "standard warmup run 1 for testing state version outputs",
	},
}

// Standard warmup state versions for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
// Please note: Even though RunID is a pointer, it cannot be nil due to a not-null constraint.
var standardWarmupStateVersionsForStateVersionOutputs = []models.StateVersion{
	{
		WorkspaceID: "top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
		RunID:       ptr.String("standard warmup run 0 for testing state version outputs"),
		CreatedBy:   "someone-sv0",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
		RunID:       ptr.String("standard warmup run 1 for testing state version outputs"),
		CreatedBy:   "someone-sv1",
	},
}

// Standard warmup state version outputs for tests in this module.
// The ID fields will be replaced by the real IDs during the create function.
// The state version ID here is a concatenation of the workspace's full path, a colon, and the run's comment.
// There are 4 outputs: 2 from each run; 2 pairs of names.
var standardWarmupStateVersionOutputs = []models.StateVersionOutput{
	{
		Name:      "output-02",
		Value:     []byte("output-0-value"),
		Type:      []byte("output-0-type"),
		Sensitive: false,
		StateVersionID: fmt.Sprintf("%s:%s",
			"top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
			"standard warmup run 0 for testing state version outputs"),
	},
	{
		Name:      "output-13",
		Value:     []byte("output-1-value"),
		Type:      []byte("output-1-type"),
		Sensitive: true,
		StateVersionID: fmt.Sprintf("%s:%s",
			"top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
			"standard warmup run 0 for testing state version outputs"),
	},
	{
		Name:      "output-02",
		Value:     []byte("output-2-value"),
		Type:      []byte("output-2-type"),
		Sensitive: true,
		StateVersionID: fmt.Sprintf("%s:%s",
			"top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
			"standard warmup run 1 for testing state version outputs"),
	},
	{
		Name:      "output-13",
		Value:     []byte("output-3-value"),
		Type:      []byte("output-3-type"),
		Sensitive: false,
		StateVersionID: fmt.Sprintf("%s:%s",
			"top-level-group-0-for-state-version-outputs/workspace-0-for-state-version-outputs",
			"standard warmup run 1 for testing state version outputs"),
	},
}

// createWarmupStateVersionOutputs creates some warmup state version outputs for a test
// The warmup state version outputs to create can be standard or otherwise.
func createWarmupStateVersionOutputs(ctx context.Context, testClient *testClient,
	input warmupStateVersionOutputs,
) (*warmupStateVersionOutputs, error) {
	// It is necessary to create at least one group, workspace, run, and state version
	// in order to provide the necessary IDs for the state version outputs.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	workspaceMap := map[string]string{}
	for _, ws := range resultWorkspaces {
		workspaceMap[ws.FullPath] = ws.Metadata.ID
	}

	// Please note: This function supports only a single workspace.
	resultRuns, err := createInitialRuns(ctx, testClient, input.runs, resultWorkspaces[0].Metadata.ID)
	if err != nil {
		return nil, err
	}

	runMap := map[string]string{}
	for _, run := range resultRuns {
		runMap[run.Comment] = run.Metadata.ID
	}

	resultStateVersions, err := createInitialStateVersions(ctx, testClient, workspaceMap, runMap, input.stateVersions)
	if err != nil {
		return nil, err
	}

	stateVersionMap := map[string]string{}
	for _, sv := range resultStateVersions {
		stateVersionMap[fmt.Sprintf("%s:%s", sv.WorkspaceID, *sv.RunID)] = sv.Metadata.ID
	}

	resultStateVersionOutputs, err := createInitialStateVersionOutputs(ctx, testClient,
		workspaceMap, runMap, stateVersionMap, input.stateVersionOutputs)

	return &warmupStateVersionOutputs{
		groups:              resultGroups,
		workspaces:          resultWorkspaces,
		runs:                resultRuns,
		stateVersions:       resultStateVersions,
		stateVersionOutputs: resultStateVersionOutputs,
	}, err
}

// compareStateVersionOutputs compares two state version output objects,
// including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareStateVersionOutputs(t *testing.T, expected, actual *models.StateVersionOutput,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Value, actual.Value)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Sensitive, actual.Sensitive)
	assert.Equal(t, expected.StateVersionID, actual.StateVersionID)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
