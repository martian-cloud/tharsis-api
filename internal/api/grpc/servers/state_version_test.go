package servers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	runsvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// TestGetRunStateVersion covers the RPC the post-apply policy job uses to find the state version its
// own run produced, rather than reading the workspace's current one (which belongs to whichever run
// wrote state last).
func TestGetRunStateVersion(t *testing.T) {
	// Real UUIDs: a global ID embeds one, so FetchModel rejects placeholder strings before it ever
	// reaches the run service.
	runID := "5f9c4f27-9d47-4d8a-9b0a-3f4c1a2b3c4d"
	runGID := gid.ToGlobalID(types.RunModelType, runID)
	workspaceID := "8a1b2c3d-4e5f-4a6b-8c9d-0e1f2a3b4c5d"
	stateVersionID := "c3d4e5f6-7a8b-4c9d-8e1f-2a3b4c5d6e7f"
	now := time.Now().UTC()

	// newServer wires a catalog holding only the run service: Init registers fetchers as closures, so
	// the unused services are never dereferenced while resolving a run.
	newServer := func(t *testing.T, stateVersions []models.StateVersion) *StateVersionServer {
		mockRunService := runsvc.NewMockService(t)
		mockRunService.On("GetRunByID", mock.Anything, runID).Return(&models.Run{
			Metadata:    models.ResourceMetadata{ID: runID},
			WorkspaceID: workspaceID,
		}, nil)
		mockRunService.On("GetStateVersionsByRunIDs", mock.Anything, []string{runID}).Return(stateVersions, nil)

		catalog := &services.Catalog{RunService: mockRunService}
		catalog.Init()

		return NewStateVersionServer(catalog)
	}

	t.Run("returns the run's state version", func(t *testing.T) {
		server := newServer(t, []models.StateVersion{
			{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
					// The protobuf conversion dereferences both timestamps.
					CreationTimestamp:    &now,
					LastUpdatedTimestamp: &now,
				},
				WorkspaceID: workspaceID,
				RunID:       &runID,
				CreatedBy:   "job",
			},
		})

		result, err := server.GetRunStateVersion(context.Background(), &pb.GetRunStateVersionRequest{RunId: runGID})
		require.NoError(t, err)

		// IDs cross the wire as global IDs, as everywhere else in this package.
		assert.Equal(t, gid.ToGlobalID(types.StateVersionModelType, stateVersionID), result.Metadata.Id)
		assert.Equal(t, gid.ToGlobalID(types.WorkspaceModelType, workspaceID), result.WorkspaceId)
		require.NotNil(t, result.RunId)
		assert.Equal(t, runGID, *result.RunId)
	})

	t.Run("reports not found when the run wrote no state", func(t *testing.T) {
		server := newServer(t, []models.StateVersion{})

		_, err := server.GetRunStateVersion(context.Background(), &pb.GetRunStateVersionRequest{RunId: runGID})
		require.Error(t, err)
		// ENotFound is what lets the caller treat "this run produced no state" as an expected outcome
		// instead of a failure; anything else must surface as an error.
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}
