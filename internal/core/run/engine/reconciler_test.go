package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func newReconcilerForTest(dbClient *db.Client) *Reconciler {
	log, _ := logger.NewForTest()
	return NewReconciler(log, dbClient, &fakeMaintenanceMonitor{})
}

func queuingRun(wsID string) *models.Run {
	return &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-" + wsID},
		WorkspaceID: wsID,
		Status:      models.RunQueuing,
	}
}

// matchEnqueueForWorkspace matches an AddWorkItemToQueue call carrying a
// QUEUE_PENDING_RUNS_FOR_WORKSPACE payload for the given workspace.
func matchEnqueueForWorkspace(wsID string) any {
	return mock.MatchedBy(func(in *db.AddWorkItemToQueueInput) bool {
		p, ok := in.Payload.(*db.QueuePendingRunsForWorkspacePayload)
		return ok && in.Type == db.QueuePendingRunsForWorkspaceType && p.WorkspaceID == wsID
	})
}

func TestReconcile_DedupesWorkspacesAndFiltersStaleQueuingRuns(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	// The sweep queries both queuing statuses with a staleness cutoff and no workspace filter.
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.Filter != nil &&
			in.Filter.WorkspaceID == nil &&
			len(in.Filter.Statuses) == 2 &&
			in.Filter.Statuses[0] == models.RunQueuing &&
			in.Filter.Statuses[1] == models.RunQueuingApply &&
			in.Filter.UpdatedBefore != nil
	})).Return(&db.RunsResult{
		PageInfo: &pagination.PageInfo{HasNextPage: false},
		// Two runs share ws-1; ws-2 has one.
		Runs: []*models.Run{queuingRun("ws-1"), queuingRun("ws-1"), queuingRun("ws-2")},
	}, nil)

	mockWIQ := db.NewMockWorkItemsQueue(t)
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-1")).Return(&db.WorkItem{}, nil).Once()
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-2")).Return(&db.WorkItem{}, nil).Once()

	dbClient := &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ}
	err := newReconcilerForTest(dbClient).reconcile(ctx)
	assert.NoError(t, err)
}

func TestReconcile_PaginatesAndDedupesAcrossPages(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	// Page 1 (no cursor): ws-1, with another page to follow.
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.PaginationOptions != nil && in.PaginationOptions.After == nil
	})).Return(&db.RunsResult{
		PageInfo: &pagination.PageInfo{
			HasNextPage: true,
			Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
				c := "cursor-1"
				return &c, nil
			},
		},
		Runs: []*models.Run{queuingRun("ws-1")},
	}, nil)
	// Page 2 (cursor advanced): ws-1 again (already enqueued) plus ws-2; last page.
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.PaginationOptions != nil && in.PaginationOptions.After != nil && *in.PaginationOptions.After == "cursor-1"
	})).Return(&db.RunsResult{
		PageInfo: &pagination.PageInfo{HasNextPage: false},
		Runs:     []*models.Run{queuingRun("ws-1"), queuingRun("ws-2")},
	}, nil)

	mockWIQ := db.NewMockWorkItemsQueue(t)
	// ws-1 is enqueued exactly once despite appearing on both pages; ws-2 once.
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-1")).Return(&db.WorkItem{}, nil).Once()
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-2")).Return(&db.WorkItem{}, nil).Once()

	dbClient := &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ}
	err := newReconcilerForTest(dbClient).reconcile(ctx)
	assert.NoError(t, err)
}

func TestReconcile_NoQueuingRunsEnqueuesNothing(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
		PageInfo: &pagination.PageInfo{HasNextPage: false},
		Runs:     []*models.Run{},
	}, nil)

	// No AddWorkItemToQueue expectations: NewMockWorkItemsQueue(t) fails the test if called.
	mockWIQ := db.NewMockWorkItemsQueue(t)

	dbClient := &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ}
	err := newReconcilerForTest(dbClient).reconcile(ctx)
	assert.NoError(t, err)
}

func TestReconcile_EnqueueErrorDoesNotAbortSweep(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
		PageInfo: &pagination.PageInfo{HasNextPage: false},
		Runs:     []*models.Run{queuingRun("ws-1"), queuingRun("ws-2")},
	}, nil)

	mockWIQ := db.NewMockWorkItemsQueue(t)
	// ws-1's enqueue fails; ws-2 must still be attempted.
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-1")).
		Return(nil, errors.New("boom")).Once()
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, matchEnqueueForWorkspace("ws-2")).
		Return(&db.WorkItem{}, nil).Once()

	dbClient := &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ}
	err := newReconcilerForTest(dbClient).reconcile(ctx)
	assert.NoError(t, err)
}

func TestReconcile_GetRunsErrorReturned(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	mockWIQ := db.NewMockWorkItemsQueue(t)

	dbClient := &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ}
	err := newReconcilerForTest(dbClient).reconcile(ctx)
	assert.Error(t, err)
}

func TestReconcile_SkipsSweepWhileInMaintenance(t *testing.T) {
	ctx := context.Background()

	// No GetRuns/AddWorkItemToQueue expectations: the mocks fail the test if touched, asserting
	// the sweep is skipped entirely (no reads, no enqueue writes) while in maintenance.
	mockRuns := db.NewMockRuns(t)
	mockWIQ := db.NewMockWorkItemsQueue(t)

	log, _ := logger.NewForTest()
	r := NewReconciler(log, &db.Client{Runs: mockRuns, WorkItemsQueue: mockWIQ},
		&fakeMaintenanceMonitor{inMaintenance: true})

	err := r.reconcile(ctx)
	assert.NoError(t, err)
}
