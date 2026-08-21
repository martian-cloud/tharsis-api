package transformers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// runsWaiting builds a GetRuns result carrying only a total count, which is all the admitter's ordering
// check reads: it asks for zero rows and counts the runs already waiting on the workspace.
func runsWaiting(count int32) *db.RunsResult {
	return &db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(count)}}
}

// noRunsWaiting builds a Runs mock reporting an empty queue, i.e. the run under test is the only one
// waiting on the workspace, so the admitter's ordering check admits it.
func noRunsWaiting(t *testing.T) *db.MockRuns {
	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsWaiting(0), nil).Maybe()
	return mockRuns
}

// queuingRun builds a non-speculative run waiting on the workspace.
func queuingRun(id string) *models.Run {
	return &models.Run{
		Metadata:    models.ResourceMetadata{ID: id},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}
}

func TestAdmissionTransformer_Transform_QueuesPendingPlan(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		// No apply node => speculative => always admitted, and never acquires the workspace.
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	// The admitter transitioned the plan to queued and recorded the change on the store.
	assert.Equal(t, models.PlanQueued, run.Plan.Status)

	storeChanges := runStore.GetChanges()
	assert.Len(t, storeChanges, 1)
}

func TestAdmissionTransformer_Transform_DoesNotQueueWhenWorkspaceLocked(t *testing.T) {
	ctx := context.Background()

	// Non-speculative run (has an apply node) so the workspace lock is enforced.
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, Locked: true}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	// Plan stays pending, nothing recorded on the store.
	assert.Equal(t, models.PlanPending, run.Plan.Status)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_IgnoresNonQueuingTransition(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanning,
		Plan:        models.Plan{Status: models.PlanRunning},
	}

	// No GetWorkspaceByID expectation: the admitter must not be called for a transition into a status
	// that is not waiting on the workspace (here a plan starting to run).
	mockWorkspaces := db.NewMockWorkspaces(t)
	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanning}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_QueuesPendingApply(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunApplyQueuing,
		// Apply admission requires a finished plan that produced changes.
		Plan:  models.Plan{Status: models.PlanFinished, HasChanges: true},
		Apply: &models.Apply{Status: models.ApplyPending},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	// The apply is non-speculative, so the workspace is acquired.
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.MatchedBy(func(ws *models.Workspace) bool {
		return ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == "run-1"
	})).Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: noRunsWaiting(t)}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunApplyQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.ApplyQueued, run.Apply.Status)
	assert.Len(t, runStore.GetChanges(), 1)
}

func TestAdmissionTransformer_Transform_SwallowsOptimisticLock(t *testing.T) {
	ctx := context.Background()

	// Non-speculative plan whose workspace acquisition fails with an OLE. The
	// transformer must swallow it, leaving the node pending and the store clean.
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
		Return(nil, errors.New("conflict", errors.WithErrorCode(errors.EOptimisticLock)))

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: noRunsWaiting(t)}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.PlanPending, run.Plan.Status)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_PropagatesNonOLEError(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(nil, errors.New("boom"))

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.Error(t, err)
}

// TestAdmissionTransformer_Transform_DefersToRunAlreadyWaiting is the regression test for the queue
// jump: a run whose queuing transition commits while the workspace happens to be free must not be
// admitted ahead of a run that was already waiting on that workspace. The node stays pending for the
// work item consumer, which admits the workspace's queuing runs in order.
func TestAdmissionTransformer_Transform_DefersToRunAlreadyWaiting(t *testing.T) {
	ctx := context.Background()

	run := queuingRun("run-2")

	// The workspace is free, so nothing but the ordering check can hold this run back. No
	// UpdateWorkspace expectation: acquiring the slot here would be the queue jump.
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsWaiting(1), nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.PlanPending, run.Plan.Status, "the run must stay pending behind the waiting run")
	assert.Empty(t, runStore.GetChanges())
}

// TestAdmissionTransformer_Transform_AdmitsWhenNoRunIsAlreadyWaiting asserts the inline admission that
// makes the uncontended path fast is preserved: an empty queue means this run is next, so it is admitted
// in the command's own transaction rather than waiting on the work item consumer.
func TestAdmissionTransformer_Transform_AdmitsWhenNoRunIsAlreadyWaiting(t *testing.T) {
	ctx := context.Background()

	run := queuingRun("run-1")

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.MatchedBy(func(ws *models.Workspace) bool {
		return ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == "run-1"
	})).Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsWaiting(0), nil).Once()

	dbClient := &db.Client{Workspaces: mockWorkspaces, Runs: mockRuns}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{NewStatus: models.RunPlanQueuing}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.PlanQueued, run.Plan.Status)
	assert.Len(t, runStore.GetChanges(), 1)
}
