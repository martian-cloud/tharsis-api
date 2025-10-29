//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestStateVersionOutputs_CreateStateVersionOutput(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, workspace, and run for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-version-output",
		Description: "test group for state version output",
		FullPath:    "test-group-state-version-output",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-version-output",
		Description:    "test workspace for state version output",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Status:      models.RunPending,
	})
	require.NoError(t, err)

	// Create a state version first (required dependency)
	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		RunID:       &run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		output          *models.StateVersionOutput
	}

	testCases := []testCase{
		{
			name: "successfully create state version output",
			output: &models.StateVersionOutput{
				Name:           "test_output",
				StateVersionID: stateVersion.Metadata.ID,
				Value:          []byte(`"test-value"`),
				Type:           []byte(`"string"`),
				Sensitive:      false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			output, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, test.output)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, test.output.Name, output.Name)
			assert.Equal(t, test.output.StateVersionID, output.StateVersionID)
			assert.Equal(t, test.output.Sensitive, output.Sensitive)
		})
	}
}

func TestStateVersionOutputs_GetStateVersionOutputByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-output-get-by-id",
		Description: "test group for state output get by id",
		FullPath:    "test-group-state-output-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-output-get-by-id",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a state version for testing
	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a state version output for testing
	createdOutput, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &models.StateVersionOutput{
		Name:           "test-output-get-by-id",
		Value:          []byte("test-value"),
		Type:           []byte("string"),
		StateVersionID: stateVersion.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectOutput    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdOutput.Metadata.ID,
			expectOutput: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			output, err := testClient.client.StateVersionOutputs.GetStateVersionOutputByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectOutput {
				require.NotNil(t, output)
				assert.Equal(t, test.id, output.Metadata.ID)
			} else {
				assert.Nil(t, output)
			}
		})
	}
}

func TestStateVersionOutputs_GetStateVersionOutputByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-output-get-by-trn",
		Description: "test group for state output get by trn",
		FullPath:    "test-group-state-output-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-output-get-by-trn",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a state version for testing
	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a state version output for testing
	createdOutput, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &models.StateVersionOutput{
		Name:           "test-output-get-by-trn",
		Value:          []byte("test-value"),
		Type:           []byte("string"),
		StateVersionID: stateVersion.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectOutput    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdOutput.Metadata.TRN,
			expectOutput: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:state_version_output:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			output, err := testClient.client.StateVersionOutputs.GetStateVersionOutputByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectOutput {
				require.NotNil(t, output)
				assert.Equal(t, createdOutput.Metadata.ID, output.Metadata.ID)
			} else {
				assert.Nil(t, output)
			}
		})
	}
}

func TestStateVersionOutputs_GetStateVersionOutputs(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-outputs-list",
		Description: "test group for state outputs list",
		FullPath:    "test-group-state-outputs-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-outputs-list",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a state version for testing
	stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test state version outputs
	outputs := []models.StateVersionOutput{
		{
			Name:           "test-output-1",
			Value:          []byte("test-value-1"),
			Type:           []byte("string"),
			StateVersionID: stateVersion.Metadata.ID,
		},
		{
			Name:           "test-output-2",
			Value:          []byte("test-value-2"),
			Type:           []byte("number"),
			StateVersionID: stateVersion.Metadata.ID,
		},
	}

	createdOutputs := []models.StateVersionOutput{}
	for _, output := range outputs {
		created, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &output)
		require.NoError(t, err)
		createdOutputs = append(createdOutputs, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		stateVersionID  string
		expectCount     int
	}

	testCases := []testCase{
		{
			name:           "get all outputs for state version",
			stateVersionID: stateVersion.Metadata.ID,
			expectCount:    len(createdOutputs),
		},
		{
			name:           "get outputs for non-existent state version",
			stateVersionID: nonExistentID,
			expectCount:    0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.StateVersionOutputs.GetStateVersionOutputs(ctx, test.stateVersionID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, test.expectCount)
		})
	}
}
