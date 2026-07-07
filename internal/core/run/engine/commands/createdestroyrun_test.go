package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	coreworkspace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
)

func TestCreateDestroyRun_Prepare(t *testing.T) {
	ctx := context.Background()

	source := &models.Run{
		Metadata:               models.ResourceMetadata{ID: "src"},
		TerraformVersion:       "1.4.0",
		ConfigurationVersionID: ptr.String("cv-1"),
		// ModuleSource nil -> ResolveModule returns a zero-value result.
	}

	workspaces := db.NewMockWorkspaces(t)
	stateVersions := db.NewMockStateVersions(t)
	runs := db.NewMockRuns(t)
	artifactStore := coreworkspace.NewMockArtifactStore(t)
	dbClient := &db.Client{Workspaces: workspaces, StateVersions: stateVersions, Runs: runs}

	// FindLatestApplyRunForWorkspace walks workspace -> state version -> run.
	workspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{CurrentStateVersionID: "sv-1"}, nil)
	stateVersions.On("GetStateVersionByID", ctx, "sv-1").Return(&models.StateVersion{RunID: ptr.String("src")}, nil)
	runs.On("GetRunByID", ctx, "src").Return(source, nil)
	// The source run's variables carry an inherited namespace path that must be cleared.
	vars := []runvariables.Variable{{Key: "k", Category: models.TerraformVariableCategory, NamespacePath: ptr.String("group")}}
	data, err := json.Marshal(vars)
	require.NoError(t, err)
	artifactStore.On("GetRunVariables", ctx, source).Return(io.NopCloser(bytes.NewReader(data)), nil)

	cmd := &CreateDestroyRun{
		dbClient:         dbClient,
		variablesBuilder: runvariables.NewBuilder(dbClient, secret.NewMockManager(t), artifactStore),
		in:               &CreateDestroyRunInput{Subject: "u", WorkspaceID: "ws-1"},
	}

	require.NoError(t, cmd.Prepare(ctx))
	require.NotNil(t, cmd.createInput)
	assert.True(t, cmd.createInput.IsDestroy)
	assert.True(t, cmd.createInput.Refresh)
	assert.Equal(t, "1.4.0", cmd.createInput.TerraformVersion)
	require.Len(t, cmd.createInput.Variables, 1)
	assert.Nil(t, cmd.createInput.Variables[0].NamespacePath, "inherited namespace path must be cleared")
}

func TestCreateDestroyRun_Execute(t *testing.T) {
	ctx := context.Background()
	input := &corerun.CreateRunInput{Subject: "u", WorkspaceID: "ws-1", IsDestroy: true}

	var gotInput *corerun.CreateRunInput
	cmd := &CreateDestroyRun{
		createRun: func(_ context.Context, in *corerun.CreateRunInput) (*models.Run, error) {
			gotInput = in
			return &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-new"},
				Status:   models.RunPending,
				Plan:     models.Plan{Status: models.PlanCreated},
			}, nil
		},
		createInput: input,
	}

	runStore := store.NewRunStore(&db.Client{})
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	require.NotNil(t, cmd.Created)
	assert.Same(t, input, gotInput)
	assert.Equal(t, models.RunQueuing, cmd.Created.Status)
}
