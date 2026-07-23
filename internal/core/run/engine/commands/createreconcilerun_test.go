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

func TestCreateReconcileRun_Execute(t *testing.T) {
	ctx := context.Background()
	input := &corerun.CreateRunInput{Subject: "u", WorkspaceID: "ws-1"}

	var gotInput *corerun.CreateRunInput
	cmd := &CreateReconcileRun{
		dbClient: &db.Client{},
		createRun: func(_ context.Context, in *corerun.CreateRunInput) (*models.Run, error) {
			gotInput = in
			return &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-new"},
				Status:   models.RunPending,
				Plan:     models.Plan{Status: models.PlanCreated},
			}, nil
		},
		variablesRetainFn: func(_ context.Context, _ string) error { return nil },
		createInput:       input,
	}

	runStore := store.NewRunStore(&db.Client{})
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	require.NotNil(t, cmd.Created)
	assert.Same(t, input, gotInput)
	assert.Equal(t, models.RunQueuing, cmd.Created.Status)
}

func TestCreateReconcileRun_Prepare(t *testing.T) {
	ctx := context.Background()

	// A reconcile run re-runs the configuration behind the current state without
	// destroying and without refresh-only, derived from the workspace's latest run.
	source := &models.Run{Metadata: models.ResourceMetadata{ID: "src"}, TerraformVersion: "1.4.0"}

	workspaces := db.NewMockWorkspaces(t)
	stateVersions := db.NewMockStateVersions(t)
	runs := db.NewMockRuns(t)
	artifactStore := coreworkspace.NewMockArtifactStore(t)
	dbClient := &db.Client{Workspaces: workspaces, StateVersions: stateVersions, Runs: runs}

	workspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{CurrentStateVersionID: "sv-1"}, nil)
	stateVersions.On("GetStateVersionByID", ctx, "sv-1").Return(&models.StateVersion{RunID: ptr.String("src")}, nil)
	runs.On("GetRunByID", ctx, "src").Return(source, nil)
	data, err := json.Marshal([]runvariables.Variable{})
	require.NoError(t, err)
	artifactStore.On("GetRunVariables", ctx, source).Return(io.NopCloser(bytes.NewReader(data)), nil)

	cmd := &CreateReconcileRun{
		dbClient:         dbClient,
		variablesBuilder: runvariables.NewBuilder(dbClient, secret.NewMockManager(t), artifactStore),
		uploadRunVariables: func(_ context.Context, _ string, _ []runvariables.Variable) (db.RetainObjectRefFunc, string, error) {
			return nil, "vars-key", nil
		},
		in: &CreateReconcileRunInput{Subject: "u", WorkspaceID: "ws-1"},
	}

	require.NoError(t, cmd.Prepare(ctx))
	require.NotNil(t, cmd.createInput)
	assert.False(t, cmd.createInput.IsDestroy)
	assert.False(t, cmd.createInput.RefreshOnly)
	assert.True(t, cmd.createInput.Refresh)
	assert.Equal(t, "vars-key", cmd.createInput.VariablesObjectStoreKey)
}
