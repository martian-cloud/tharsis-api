package admission

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestWorkspaceAdmits(t *testing.T) {
	// A run with no apply node is speculative; one with an apply node is not.
	speculativeRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}
	nonSpeculativeRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Apply: &models.Apply{}}

	tests := []struct {
		name string
		ws   *models.Workspace
		run  *models.Run
		want bool
	}{
		{"speculative run admitted even when the workspace is locked", &models.Workspace{Locked: true}, speculativeRun, true},
		{"speculative run admitted while an apply is in progress", &models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, speculativeRun, true},
		{"non-speculative run blocked when the workspace is locked", &models.Workspace{Locked: true}, nonSpeculativeRun, false},
		{"non-speculative run blocked when another run holds the workspace", &models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, nonSpeculativeRun, false},
		{"non-speculative run admitted when the workspace is free", &models.Workspace{}, nonSpeculativeRun, true},
		{"non-speculative run admitted when it already holds the workspace", &models.Workspace{CurrentApplyRunID: ptr.String("run-1")}, nonSpeculativeRun, true},
	}

	a := &Admitter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.workspaceAdmits(tt.ws, tt.run))
		})
	}
}

// TestTryStartNextRunTransition_NotQueuing verifies that a run not waiting on the workspace is a no-op
// that returns before any workspace lookup, so a nil db client is never dereferenced. The planned case is
// the one that matters: advancing such a run would begin its apply phase, which is a human's decision,
// not admission's.
func TestTryStartNextRunTransition_NotQueuing(t *testing.T) {
	runs := map[string]*models.Run{
		"planned, awaiting manual approval": {
			Status: models.RunPlanned,
			Plan:   models.Plan{Status: models.PlanFinished, HasChanges: true},
			Apply:  &models.Apply{Status: models.ApplyCreated},
		},
		"blocked on a pre-apply override": {
			Status: models.RunPreApplyAwaitingDecision,
			Plan:   models.Plan{Status: models.PlanFinished, HasChanges: true},
			Apply:  &models.Apply{Status: models.ApplyCreated},
			TaskStages: []*models.RunTaskStage{
				{StageName: models.RunTaskStageNamePreApply, Status: models.RunTaskStageAwaitingOverride},
			},
		},
		"already running its plan": {
			Status: models.RunPlanning,
			Plan:   models.Plan{Status: models.PlanRunning},
		},
		"finished": {
			Status: models.RunPlannedAndFinished,
			Plan:   models.Plan{Status: models.PlanFinished},
		},
	}

	for name, run := range runs {
		t.Run(name, func(t *testing.T) {
			a := &Admitter{} // nil db client: not queuing means no workspace lookup
			started, changes, err := a.TryStartNextRunTransition(context.Background(), run)
			assert.NoError(t, err)
			assert.False(t, started)
			assert.Nil(t, changes)
		})
	}
}

// TestTryStartNextRunTransition_QueuingStatusesAreConsidered asserts that every workspace-gated node is
// covered by the single queuing-status check, so a new gated node cannot be silently left unadmitted. The
// workspace is missing, so each run gets as far as the lookup and then stops — reaching the lookup at all
// is what is being asserted, since a non-queuing run returns before it.
func TestTryStartNextRunTransition_QueuingStatusesAreConsidered(t *testing.T) {
	for _, status := range models.QueuingRunStatuses {
		t.Run(string(status), func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(nil, nil).Once()

			a := New(&db.Client{Workspaces: mockWorkspaces})
			run := &models.Run{
				Metadata:    models.ResourceMetadata{ID: "run-1"},
				WorkspaceID: "ws-1",
				Status:      status,
			}

			started, _, err := a.TryStartNextRunTransition(context.Background(), run)
			assert.NoError(t, err)
			assert.False(t, started, "a missing workspace admits nothing")
		})
	}
}

func TestWorkspaceAvailable(t *testing.T) {
	assert.True(t, WorkspaceAvailable(&models.Workspace{}), "unlocked and unoccupied is available")
	assert.False(t, WorkspaceAvailable(&models.Workspace{Locked: true}), "locked is not available")
	assert.False(t, WorkspaceAvailable(&models.Workspace{CurrentApplyRunID: ptr.String("run-1")}), "occupied is not available")
}

func TestWorkspaceAvailableForRun(t *testing.T) {
	run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}

	assert.True(t, workspaceAvailableForRun(&models.Workspace{}, run), "empty CurrentApplyRunID is available")
	assert.True(t, workspaceAvailableForRun(&models.Workspace{CurrentApplyRunID: ptr.String("run-1")}, run), "held by this run is available for it")
	assert.False(t, workspaceAvailableForRun(&models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, run), "held by another run is not available")
	assert.False(t, workspaceAvailableForRun(&models.Workspace{Locked: true, CurrentApplyRunID: ptr.String("run-1")}, run), "locked is not available even for the holding run")
}

// waitingRun builds a non-speculative run parked in plan_queuing on ws-1.
func waitingRun(id string) *models.Run {
	return &models.Run{
		Metadata:    models.ResourceMetadata{ID: id},
		WorkspaceID: "ws-1",
		Status:      models.RunPlanQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}
}

// runsWaiting builds a GetRuns result carrying only a total count, which is all the ordering check
// reads: it asks for zero rows and counts the runs already waiting on the workspace.
func runsWaiting(count int32) *db.RunsResult {
	return &db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(count)}}
}

// TestTryStartNextRunTransitionIfNextInLine covers the ordering guard: the run is admitted only when
// nothing was already waiting on the workspace. The query counts the runs ahead of it (the run's own
// queuing transition is not written yet, so it is not among them), and any non-zero count means the
// work item consumer must do the admitting, in queue order.
func TestTryStartNextRunTransitionIfNextInLine(t *testing.T) {
	tests := []struct {
		name string
		// waitingAhead is the number of runs the query finds already waiting on the workspace.
		waitingAhead int32
		wantStarted  bool
	}{
		{"admitted when nothing was already waiting", 0, true},
		{"deferred when one run was already waiting", 1, false},
		{"deferred when several runs were already waiting", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := waitingRun("run-1")

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
				Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
			if tt.wantStarted {
				// Only an admitted run acquires the slot.
				mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
					Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)
			}

			mockRuns := db.NewMockRuns(t)
			mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(runsWaiting(tt.waitingAhead), nil)

			a := New(&db.Client{Workspaces: mockWorkspaces, Runs: mockRuns})

			started, changes, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStarted, started)
			if tt.wantStarted {
				assert.NotEmpty(t, changes)
			} else {
				assert.Empty(t, changes)
				assert.Equal(t, models.PlanPending, run.Plan.Status, "a deferred run's node stays pending")
			}
		})
	}
}

// TestTryStartNextRunTransitionIfNextInLine_SkipsOrderingCheckForSlotHolder asserts a run that already
// holds the workspace is admitted without consulting the queue. It is not competing for the slot, so it
// cannot jump the queue — and deferring it would strand it, because the work item consumer only admits
// when the workspace is free, which it is not while this run holds it. The Runs mock has no
// expectations, so any ordering query would fail the test.
func TestTryStartNextRunTransitionIfNextInLine_SkipsOrderingCheckForSlotHolder(t *testing.T) {
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1", CreationTimestamp: nil},
		WorkspaceID: "ws-1",
		Status:      models.RunApplyQueuing,
		Plan:        models.Plan{Status: models.PlanFinished, HasChanges: true},
		Apply:       &models.Apply{Status: models.ApplyPending},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	// The run holds the slot, so it is neither re-acquired (no UpdateWorkspace) nor order-checked.
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: db.NewMockRuns(t)})

	started, changes, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
	assert.NoError(t, err)
	assert.True(t, started)
	assert.NotEmpty(t, changes)
	assert.Equal(t, models.ApplyQueued, run.Apply.Status)
}

// TestTryStartNextRunTransitionIfNextInLine_SkipsOrderingCheckForSpeculativeRun asserts a speculative
// run is admitted without an ordering query: it never occupies the slot, so it is not in line for one.
func TestTryStartNextRunTransitionIfNextInLine_SkipsOrderingCheckForSpeculativeRun(t *testing.T) {
	run := waitingRun("run-1")
	run.Apply = nil // no apply node => speculative

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("other-run")}, nil)

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: db.NewMockRuns(t)})

	started, _, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
	assert.NoError(t, err)
	assert.True(t, started)
	assert.Equal(t, models.PlanQueued, run.Plan.Status)
}

// TestTryStartNextRunTransition_NoOrderingCheck asserts the plain entry point performs no ordering
// query, since its callers (the work item consumer, via QueueRun) have already selected the workspace's
// oldest queuing run. The Runs mock has no expectations, so a query would fail the test.
func TestTryStartNextRunTransition_NoOrderingCheck(t *testing.T) {
	run := waitingRun("run-1")

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: db.NewMockRuns(t)})

	started, _, err := a.TryStartNextRunTransition(context.Background(), run)
	assert.NoError(t, err)
	assert.True(t, started)
	assert.Equal(t, models.PlanQueued, run.Plan.Status)
}

// TestIsNextInLine_QueryShape pins the ordering query, which is what makes a bare count sufficient:
// the workspace's runs in a queuing status, bounded by the current time, with no rows fetched. The run
// under test carries a stale LastUpdatedTimestamp to assert the bound comes from the clock rather than
// from the run, whose own updated_at predates the transition being admitted.
func TestIsNextInLine_QueryShape(t *testing.T) {
	before := time.Now().UTC()

	run := waitingRun("run-1")
	stale := before.Add(-time.Hour)
	run.Metadata.LastUpdatedTimestamp = &stale

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(input *db.GetRunsInput) bool {
		if input.Filter == nil || input.PaginationOptions == nil || input.PaginationOptions.First == nil {
			return false
		}
		return *input.PaginationOptions.First == 0 &&
			input.Filter.WorkspaceID != nil && *input.Filter.WorkspaceID == "ws-1" &&
			assert.ObjectsAreEqual(models.QueuingRunStatuses, input.Filter.Statuses) &&
			input.Filter.UpdatedBefore != nil && !input.Filter.UpdatedBefore.Before(before)
	})).Return(runsWaiting(0), nil).Once()

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: mockRuns})

	started, _, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
	assert.NoError(t, err)
	assert.True(t, started)
}

// TestIsNextInLine_CountError asserts a failed count is propagated rather than silently admitting the
// run, which could jump the queue. The count is lazy, so it fails on invocation, not on the query.
func TestIsNextInLine_CountError(t *testing.T) {
	run := waitingRun("run-1")

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	failingCount := func(context.Context) (int32, error) { return 0, errors.New("count failed") }

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: failingCount}}, nil)

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: mockRuns})

	started, _, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
	assert.Error(t, err)
	assert.False(t, started)
}

// TestIsNextInLine_QueryError asserts a failed ordering query is propagated rather than silently
// admitting the run, which could jump the queue.
func TestIsNextInLine_QueryError(t *testing.T) {
	run := waitingRun("run-1")

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	a := New(&db.Client{Workspaces: mockWorkspaces, Runs: mockRuns})

	started, _, err := a.TryStartNextRunTransitionIfNextInLine(context.Background(), run)
	assert.Error(t, err)
	assert.False(t, started)
}
