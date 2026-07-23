package commands

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	coreworkspace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// stubCreateRun returns a createRunFunc that records the input it received and
// returns the given run (in RunPending state, ready to be queued) or error. It lets
// the command tests exercise Execute without wiring every corerun.Create collaborator.
func stubCreateRun(captured **corerun.CreateRunInput, err error) createRunFunc {
	return func(_ context.Context, in *corerun.CreateRunInput) (*models.Run, error) {
		if captured != nil {
			*captured = in
		}
		if err != nil {
			return nil, err
		}
		return &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-new"},
			Status:   models.RunPending,
			Plan:     models.Plan{Status: models.PlanCreated},
		}, nil
	}
}

func TestCreateRun_Prepare(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		refresh     *bool
		wantRefresh bool
	}{
		{name: "nil refresh defaults to true", refresh: nil, wantRefresh: true},
		{name: "explicit false is honored", refresh: ptr.Bool(false), wantRefresh: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspaces := db.NewMockWorkspaces(t)
			variables := db.NewMockVariables(t)
			dbClient := &db.Client{Workspaces: workspaces, Variables: variables}

			// variablesBuilder.Build merges the workspace's inherited variables.
			workspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{FullPath: "group/ws"}, nil)
			variables.On("GetVariables", ctx, mock.Anything).Return(&db.VariableResult{Variables: nil}, nil)

			cmd := &CreateRun{
				dbClient:         dbClient,
				variablesBuilder: runvariables.NewBuilder(dbClient, secret.NewMockManager(t), coreworkspace.NewMockArtifactStore(t)),
				moduleResolver:   nil, // ModuleSource nil -> ResolveModule returns a zero-value result, resolver unused.
				uploadRunVariables: func(_ context.Context, _ string, _ []runvariables.Variable) (db.RetainObjectRefFunc, string, error) {
					return nil, "vars-key", nil
				},
				in: &NewRunInput{Subject: "u@example.com", WorkspaceID: "ws-1", Refresh: test.refresh},
			}

			require.NoError(t, cmd.Prepare(ctx))
			require.NotNil(t, cmd.createInput)
			assert.Equal(t, test.wantRefresh, cmd.createInput.Refresh)
			assert.Equal(t, "vars-key", cmd.createInput.VariablesObjectStoreKey)
		})
	}
}

func TestCreateRun_Execute(t *testing.T) {
	tests := []struct {
		name       string
		createErr  error
		linkRefErr error
		wantErr    bool
	}{
		{name: "creates the run and queues it"},
		{name: "propagates a create error", createErr: errors.New("boom"), wantErr: true},
		{name: "LinkRef error is returned", linkRefErr: errors.New("link error"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			input := &corerun.CreateRunInput{Subject: "u", WorkspaceID: "ws-1"}

			var gotInput *corerun.CreateRunInput
			cmd := &CreateRun{
				dbClient:  &db.Client{},
				createRun: stubCreateRun(&gotInput, test.createErr),
				variablesRetainFn: func(_ context.Context, _ string) error {
					return test.linkRefErr
				},
				createInput: input,
			}

			runStore := store.NewRunStore(&db.Client{})
			err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})

			if test.wantErr {
				require.Error(t, err)
				assert.Nil(t, cmd.Created)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd.Created)
			assert.Same(t, input, gotInput, "the command passes its resolved input to createRun")
			assert.Equal(t, models.RunQueuing, cmd.Created.Status)
			got, gErr := runStore.GetRunByID(ctx, "run-new")
			require.NoError(t, gErr)
			assert.Equal(t, cmd.Created, got)
		})
	}
}
