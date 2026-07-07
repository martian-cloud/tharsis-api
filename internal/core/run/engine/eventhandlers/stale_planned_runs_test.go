package eventhandlers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestStalePlannedRunDiscarder_HandleRunChanges(t *testing.T) {
	logr, _ := logger.NewForTest()

	runChange := func(workspaceID string, changes ...statemachine.NodeStatusChange) types.RunChange {
		return types.RunChange{
			Run:               &models.Run{Metadata: models.ResourceMetadata{ID: "run-" + workspaceID}, WorkspaceID: workspaceID},
			NodeStatusChanges: changes,
		}
	}

	t.Run("enqueues a work item stamped with the apply-completion cutoff", func(t *testing.T) {
		before := time.Now().UTC()

		mockWIQ := db.NewMockWorkItemsQueue(t)
		mockWIQ.On("AddWorkItemToQueue", mock.Anything, mock.MatchedBy(func(in *db.AddWorkItemToQueueInput) bool {
			payload, ok := in.Payload.(*db.DiscardStalePlannedRunsForWorkspacePayload)
			// The cutoff is captured as "now" when the apply completes, so it must be set
			// and not earlier than the moment just before handling.
			return in.Type == db.DiscardStalePlannedRunsForWorkspaceType &&
				ok && payload.WorkspaceID == "ws-1" &&
				!payload.ApplyCompletedAt.IsZero() && !payload.ApplyCompletedAt.Before(before)
		})).Return(&db.WorkItem{}, nil).Once()

		handler := NewStalePlannedRunDiscarder(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{
			runChange("ws-1", statemachine.RunStatusChange{OldStatus: models.RunApplying, NewStatus: models.RunApplied}),
		})
		require.NoError(t, err)
		mockWIQ.AssertExpectations(t)
	})

	t.Run("does not enqueue when no run was applied", func(t *testing.T) {
		// NewMockWorkItemsQueue(t) fails the test on any unexpected call.
		mockWIQ := db.NewMockWorkItemsQueue(t)
		handler := NewStalePlannedRunDiscarder(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{
			// A cancel/other run transition, and a plan/apply node change — none is RunApplied.
			runChange("ws-1", statemachine.RunStatusChange{OldStatus: models.RunPlanning, NewStatus: models.RunCanceled}),
			runChange("ws-2", statemachine.ApplyStatusChange{OldStatus: models.ApplyRunning, NewStatus: models.ApplyFinished}),
		})
		require.NoError(t, err)
		mockWIQ.AssertNotCalled(t, "AddWorkItemToQueue", mock.Anything, mock.Anything)
	})

	t.Run("enqueues at most once per workspace", func(t *testing.T) {
		mockWIQ := db.NewMockWorkItemsQueue(t)
		mockWIQ.On("AddWorkItemToQueue", mock.Anything, mock.Anything).Return(&db.WorkItem{}, nil).Once()

		handler := NewStalePlannedRunDiscarder(logr, &db.Client{WorkItemsQueue: mockWIQ})

		// Two applied runs in the same workspace should enqueue a single work item.
		err := handler.HandleRunChanges(context.Background(), []types.RunChange{
			runChange("ws-1", statemachine.RunStatusChange{OldStatus: models.RunApplying, NewStatus: models.RunApplied}),
			runChange("ws-1", statemachine.RunStatusChange{OldStatus: models.RunApplying, NewStatus: models.RunApplied}),
		})
		require.NoError(t, err)
		mockWIQ.AssertNumberOfCalls(t, "AddWorkItemToQueue", 1)
	})
}
