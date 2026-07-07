package engine

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// fakeMaintenanceMonitor is a minimal maintenance.Monitor stub for engine tests.
type fakeMaintenanceMonitor struct {
	inMaintenance bool
	err           error
}

func (f *fakeMaintenanceMonitor) Start(context.Context) {}
func (f *fakeMaintenanceMonitor) InMaintenanceMode(context.Context) (bool, error) {
	return f.inMaintenance, f.err
}

// newWorkItemConsumerForTest wires a WorkItemConsumer with the given mock db client and command
// processor. The factory is a zero-value Factory: handleQueuePendingRunsForWorkspace
// only calls NewQueueRun, which simply constructs a command struct. NewWorkItemConsumer is
// used so the work-item handler registry is populated. The maintenance monitor defaults to
// not-in-maintenance; the handler tests below don't reach claimAndProcess, so it's unused there.
func newWorkItemConsumerForTest(dbClient *db.Client, processor CmdProcessor) *WorkItemConsumer {
	log, _ := logger.NewForTest()
	return NewWorkItemConsumer(log, dbClient, nil, processor, &commands.Factory{}, &fakeMaintenanceMonitor{})
}

func runsResult(runs ...*models.Run) *db.RunsResult {
	return &db.RunsResult{PageInfo: &pagination.PageInfo{}, Runs: runs}
}

func TestHandleQueuePendingRunsForWorkspace_SkipsSpeculativeRuns(t *testing.T) {
	ctx := context.Background()

	// Speculative run (no apply node) waiting in the queue. Speculative runs start
	// immediately at creation rather than through this handler, so it is skipped here
	// even on a free workspace.
	specRun := &models.Run{
		Metadata: models.ResourceMetadata{ID: "spec-run"},
		Status:   models.RunQueuing,
	}

	freeWS := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(freeWS, nil)

	mockRuns := db.NewMockRuns(t)
	// The query is restricted DB-side to the only statuses this handler acts on, so the
	// workspace's run history isn't fetched wholesale.
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.Filter != nil &&
			len(in.Filter.Statuses) == 2 &&
			in.Filter.Statuses[0] == models.RunQueuing &&
			in.Filter.Statuses[1] == models.RunQueuingApply
	})).Return(runsResult(specRun), nil)

	// No queue attempt: the speculative run is not this handler's responsibility.
	mockProcessor := NewMockCmdProcessor(t)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
	mockProcessor.AssertNotCalled(t, "ProcessCommand", mock.Anything, mock.Anything)
}

func TestHandleQueuePendingRunsForWorkspace_LockedWorkspaceStopsNonSpeculative(t *testing.T) {
	ctx := context.Background()

	// Non-speculative queuing run (has an apply node).
	nonSpec := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunQueuing,
		Apply:    &models.Apply{Status: models.ApplyCreated},
	}

	lockedWS := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, Locked: true}
	mockWorkspaces := db.NewMockWorkspaces(t)
	// Fetched once before the speculative loop and once after (re-fetch).
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(lockedWS, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(nonSpec), nil)

	// No ProcessCommand expectations: the locked workspace must stop non-speculative queuing.
	mockProcessor := NewMockCmdProcessor(t)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
	mockProcessor.AssertNotCalled(t, "ProcessCommand", mock.Anything, mock.Anything)
}

func TestHandleQueuePendingRunsForWorkspace_OccupiedWorkspaceStopsNonSpeculative(t *testing.T) {
	ctx := context.Background()

	nonSpec := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunQueuing,
		Apply:    &models.Apply{Status: models.ApplyCreated},
	}

	// Workspace is occupied by a different run.
	occupiedWS := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("other-run")}
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(occupiedWS, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(nonSpec), nil)

	mockProcessor := NewMockCmdProcessor(t)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
	mockProcessor.AssertNotCalled(t, "ProcessCommand", mock.Anything, mock.Anything)
}

func TestHandleQueuePendingRunsForWorkspace_ResumesParkedApply(t *testing.T) {
	ctx := context.Background()

	// A non-speculative run with an approved apply that was parked waiting for the workspace.
	parkedApply := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-apply"},
		Status:   models.RunQueuingApply,
		Apply:    &models.Apply{Status: models.ApplyPending},
	}

	freeWS := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(freeWS, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(parkedApply), nil)

	mockProcessor := NewMockCmdProcessor(t)
	mockProcessor.On("ProcessCommand", mock.Anything, mock.MatchedBy(func(cmd interface{}) bool {
		qr, ok := cmd.(*commands.QueueRun)
		return ok && qr.RunID == "run-apply"
	})).Return(nil).Once()

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
}

func TestHandleQueuePendingRunsForWorkspace_StartsNextPendingPlan(t *testing.T) {
	ctx := context.Background()

	// A queuing non-speculative run whose plan should be started on a free workspace.
	pending := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunQueuing,
		Apply:    &models.Apply{Status: models.ApplyCreated},
	}

	freeWS := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(freeWS, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(pending), nil)

	mockProcessor := NewMockCmdProcessor(t)
	mockProcessor.On("ProcessCommand", mock.Anything, mock.MatchedBy(func(cmd interface{}) bool {
		qr, ok := cmd.(*commands.QueueRun)
		return ok && qr.RunID == "run-1"
	})).Return(nil).Once()

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
}

func TestHandleQueuePendingRunsForWorkspace_WorkspaceNotFound(t *testing.T) {
	ctx := context.Background()

	// The actionable runs are queried first; the missing workspace then makes
	// handling a no-op that still succeeds.
	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(), nil)

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(nil, nil)

	mockProcessor := NewMockCmdProcessor(t)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleQueuePendingRunsForWorkspace(ctx, &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"})
	assert.NoError(t, err)
	mockProcessor.AssertNotCalled(t, "ProcessCommand", mock.Anything, mock.Anything)
}

func TestHandleDiscardStalePlannedRunsForWorkspace_DiscardsPlanned(t *testing.T) {
	ctx := context.Background()

	applyCompletedAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	planned := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-planned"},
		Status:   models.RunPlanned,
		Apply:    &models.Apply{Status: models.ApplyCreated},
	}

	mockRuns := db.NewMockRuns(t)
	// The handler restricts the query to planned runs in the workspace last updated (i.e.
	// that entered planned) before the apply completed (DB-side filters), so non-planned
	// and newer runs are never discarded.
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.Filter != nil &&
			in.Filter.WorkspaceID != nil && *in.Filter.WorkspaceID == "ws-1" &&
			len(in.Filter.Statuses) == 1 && in.Filter.Statuses[0] == models.RunPlanned &&
			in.Filter.UpdatedBefore != nil && in.Filter.UpdatedBefore.Equal(applyCompletedAt)
	})).Return(runsResult(planned), nil)

	mockProcessor := NewMockCmdProcessor(t)
	// Each planned run is discarded with the activity event suppressed (system-initiated).
	mockProcessor.On("ProcessCommand", mock.Anything, mock.MatchedBy(func(cmd interface{}) bool {
		_, ok := cmd.(*commands.DiscardRun)
		return ok
	})).Return(nil).Once()

	dbClient := &db.Client{Runs: mockRuns}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	err := s.handleDiscardStalePlannedRunsForWorkspace(ctx, &db.DiscardStalePlannedRunsForWorkspacePayload{WorkspaceID: "ws-1", ApplyCompletedAt: applyCompletedAt})
	assert.NoError(t, err)
	mockProcessor.AssertNumberOfCalls(t, "ProcessCommand", 1)
}

func TestClaimAndProcess_AcknowledgesAfterHandling(t *testing.T) {
	ctx := context.Background()

	workItem := db.WorkItem{
		ID:      "wi-1",
		Type:    db.QueuePendingRunsForWorkspaceType,
		Payload: &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-1"},
	}

	mockQueue := db.NewMockWorkItemsQueue(t)
	mockQueue.On("ClaimWorkItems", mock.Anything, mock.MatchedBy(func(in *db.ClaimWorkItemsInput) bool {
		return in.Type == db.QueuePendingRunsForWorkspaceType
	})).Return([]db.WorkItem{workItem}, nil)
	mockQueue.On("AcknowledgeWorkItem", mock.Anything, "wi-1").Return(nil).Once()

	// The work item resolves to a workspace that no longer exists, so handling is a no-op
	// that still succeeds and is acknowledged.
	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsResult(), nil)

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(nil, nil)

	mockProcessor := NewMockCmdProcessor(t)

	dbClient := &db.Client{
		WorkItemsQueue: mockQueue,
		Workspaces:     mockWorkspaces,
		Runs:           mockRuns,
	}
	s := newWorkItemConsumerForTest(dbClient, mockProcessor)

	s.claimAndProcess(ctx, db.QueuePendingRunsForWorkspaceType)

	mockQueue.AssertCalled(t, "AcknowledgeWorkItem", mock.Anything, "wi-1")
}

func TestClaimAndProcess_SkipsClaimWhileInMaintenance(t *testing.T) {
	ctx := context.Background()

	// No ClaimWorkItems expectation: NewMockWorkItemsQueue(t) fails the test if it's called,
	// which is exactly what we want to assert — claiming (a DB write) is skipped in maintenance.
	mockQueue := db.NewMockWorkItemsQueue(t)
	dbClient := &db.Client{WorkItemsQueue: mockQueue}

	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, dbClient, nil, NewMockCmdProcessor(t), &commands.Factory{},
		&fakeMaintenanceMonitor{inMaintenance: true})

	s.claimAndProcess(ctx, db.QueuePendingRunsForWorkspaceType)
}

func TestClaimAndProcess_ClaimsWhenNotInMaintenance(t *testing.T) {
	ctx := context.Background()

	// Not in maintenance: ClaimWorkItems is attempted (returns nothing, so no further work).
	mockQueue := db.NewMockWorkItemsQueue(t)
	mockQueue.On("ClaimWorkItems", mock.Anything, mock.Anything).Return([]db.WorkItem{}, nil).Once()
	dbClient := &db.Client{WorkItemsQueue: mockQueue}

	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, dbClient, nil, NewMockCmdProcessor(t), &commands.Factory{},
		&fakeMaintenanceMonitor{inMaintenance: false})

	s.claimAndProcess(ctx, db.QueuePendingRunsForWorkspaceType)

	mockQueue.AssertCalled(t, "ClaimWorkItems", mock.Anything, mock.Anything)
}
