package eventhandlers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	wsTestRunID       = "run-1"
	wsTestWorkspaceID = "ws-1"
)

func TestWorkspaceLockManager_HandleRunChanges(t *testing.T) {
	type testCase struct {
		name string
		run  *models.Run
		// node status changes carried by the run change (the transitions this command
		// produced), which the handler scans to decide whether to enqueue.
		nodeChanges []statemachine.NodeStatusChange
		// workspace returned by GetWorkspaceByID; nil means not found.
		workspace *models.Workspace
		// whether GetWorkspaceByID is expected to be called at all.
		expectGetWorkspace bool
		// whether UpdateWorkspace should be called, and the expected CurrentApplyRunID
		// / DirtyState on the workspace passed to it.
		expectUpdate            bool
		expectCurrentApplyRunID string
		expectDirtyState        bool
		// whether a work item should be enqueued.
		expectEnqueue bool
	}

	testCases := []testCase{
		{
			name: "speculative run enqueues on transition to queuing",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       nil, // speculative
				Plan:        models.Plan{Status: models.PlanPending},
				Status:      models.RunQueuing,
			},
			nodeChanges:        []statemachine.NodeStatusChange{statemachine.RunStatusChange{OldStatus: models.RunPending, NewStatus: models.RunQueuing}},
			expectGetWorkspace: false,
			expectEnqueue:      true, // transitioned to queuing
		},
		{
			name: "run that merely remains queuing without a transition does not enqueue",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       nil,
				// Plan node is still pending, but no run-status transition occurred in
				// this change set (e.g. a later no-op update) — must not re-enqueue.
				Plan:   models.Plan{Status: models.PlanPending},
				Status: models.RunQueuing,
			},
			nodeChanges:        nil,
			expectGetWorkspace: false,
			expectEnqueue:      false,
		},
		{
			name: "apply transition to queuing_apply enqueues",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyPending},
				Status:      models.RunQueuingApply,
			},
			nodeChanges:        []statemachine.NodeStatusChange{statemachine.RunStatusChange{OldStatus: models.RunPlanned, NewStatus: models.RunQueuingApply}},
			expectGetWorkspace: false, // not releasing (not complete, not planned)
			expectEnqueue:      true,
		},
		{
			name: "run with no queuing transition does nothing",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       nil,
				Plan:        models.Plan{Status: models.PlanRunning},
				Status:      models.RunPlanning,
			},
			expectGetWorkspace: false,
			expectEnqueue:      false,
		},
		{
			name: "completed non-speculative run releases workspace it holds and enqueues",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyFinished},
				Status:      models.RunApplied,
			},
			workspace:               &models.Workspace{CurrentApplyRunID: ptr.String(wsTestRunID)},
			expectGetWorkspace:      true,
			expectUpdate:            true,
			expectCurrentApplyRunID: "",
			expectEnqueue:           true,
		},
		{
			name: "completed run does not release a workspace held by a different run",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyFinished},
				Status:      models.RunApplied,
			},
			workspace:          &models.Workspace{CurrentApplyRunID: ptr.String("other-run")},
			expectGetWorkspace: true,
			expectUpdate:       false,
			expectEnqueue:      false,
		},
		{
			name: "manual run parked at planned with auto-apply off releases workspace",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyCreated},
				Status:      models.RunPlanned,
				AutoApply:   false,
			},
			workspace:               &models.Workspace{CurrentApplyRunID: ptr.String(wsTestRunID)},
			expectGetWorkspace:      true,
			expectUpdate:            true,
			expectCurrentApplyRunID: "",
			expectEnqueue:           true,
		},
		{
			name: "auto-apply run at planned does not release",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyCreated},
				Status:      models.RunPlanned,
				AutoApply:   true,
			},
			workspace:          &models.Workspace{CurrentApplyRunID: ptr.String(wsTestRunID)},
			expectGetWorkspace: false, // neither release nor force-cancel branch fires
			expectUpdate:       false,
			expectEnqueue:      false,
		},
		{
			name: "force-canceled apply marks workspace dirty",
			run: &models.Run{
				Metadata:      models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID:   wsTestWorkspaceID,
				Apply:         &models.Apply{Status: models.ApplyCanceled},
				Status:        models.RunCanceled,
				ForceCanceled: true,
			},
			// CurrentApplyRunID differs so the release branch does not update the
			// workspace; this isolates the dirty-state behavior. The run is canceled
			// (final), so GetWorkspaceByID is invoked by both the release check and the
			// force-cancel branch.
			workspace:               &models.Workspace{CurrentApplyRunID: ptr.String("other-run")},
			expectGetWorkspace:      true,
			expectUpdate:            true,
			expectCurrentApplyRunID: "other-run",
			expectDirtyState:        true,
			expectEnqueue:           false,
		},
		{
			name: "nil workspace from GetWorkspaceByID is handled without panic",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: wsTestRunID},
				WorkspaceID: wsTestWorkspaceID,
				Apply:       &models.Apply{Status: models.ApplyFinished},
				Status:      models.RunApplied,
			},
			workspace:          nil,
			expectGetWorkspace: true,
			expectUpdate:       false,
			expectEnqueue:      false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logr, _ := logger.NewForTest()

			mockWS := db.NewMockWorkspaces(t)
			mockWIQ := db.NewMockWorkItemsQueue(t)

			if test.expectGetWorkspace {
				mockWS.On("GetWorkspaceByID", mock.Anything, wsTestWorkspaceID).
					Return(test.workspace, nil)
			}

			if test.expectUpdate {
				mockWS.On("UpdateWorkspace", mock.Anything, mock.MatchedBy(func(ws *models.Workspace) bool {
					gotApplyRunID := ""
					if ws.CurrentApplyRunID != nil {
						gotApplyRunID = *ws.CurrentApplyRunID
					}
					return gotApplyRunID == test.expectCurrentApplyRunID &&
						ws.DirtyState == test.expectDirtyState
				})).Return(test.workspace, nil)
			}

			if test.expectEnqueue {
				mockWIQ.On("AddWorkItemToQueue", mock.Anything, mock.MatchedBy(func(in *db.AddWorkItemToQueueInput) bool {
					payload, ok := in.Payload.(*db.QueuePendingRunsForWorkspacePayload)
					return in.Type == db.QueuePendingRunsForWorkspaceType &&
						ok && payload.WorkspaceID == wsTestWorkspaceID
				})).Return(&db.WorkItem{}, nil)
			}

			dbClient := &db.Client{Workspaces: mockWS, WorkItemsQueue: mockWIQ}
			handler := NewWorkspaceLockManager(logr, dbClient)

			require.NoError(t, handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: test.run, NodeStatusChanges: test.nodeChanges}}))

			mockWS.AssertExpectations(t)
			mockWIQ.AssertExpectations(t)
		})
	}
}

func TestWorkspaceLockManager_HandleRunChanges_GetWorkspaceError(t *testing.T) {
	logr, _ := logger.NewForTest()

	mockWS := db.NewMockWorkspaces(t)
	mockWS.On("GetWorkspaceByID", mock.Anything, wsTestWorkspaceID).Return(nil, assert.AnError)
	mockWIQ := db.NewMockWorkItemsQueue(t)

	dbClient := &db.Client{Workspaces: mockWS, WorkItemsQueue: mockWIQ}
	handler := NewWorkspaceLockManager(logr, dbClient)

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: wsTestRunID},
		WorkspaceID: wsTestWorkspaceID,
		Apply:       &models.Apply{Status: models.ApplyFinished},
		Status:      models.RunApplied,
	}

	err := handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: run}})
	assert.ErrorIs(t, err, assert.AnError)
}
