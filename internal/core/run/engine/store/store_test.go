package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func testRun(id string) *models.Run {
	return &models.Run{
		Metadata: models.ResourceMetadata{ID: id},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-" + id, Status: models.PlanCreated},
	}
}

func planQueued() statemachine.PlanStatusChange {
	return statemachine.PlanStatusChange{OldStatus: models.PlanCreated, NewStatus: models.PlanQueued}
}

func TestRunStore_AddRun_Idempotent(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")

	g.AddRun(r)
	g.AddRun(r) // a second add for the same ID is ignored

	assert.Len(t, g.GetRuns(), 1)
}

func TestRunStore_GetRunByID_CacheHit(t *testing.T) {
	// A cached run is returned without consulting the DB. The mock has no
	// expectations, so a stray DB call would fail the test.
	mockRuns := db.NewMockRuns(t)
	g := NewRunStore(&db.Client{Runs: mockRuns})
	r := testRun("r1")
	g.AddRun(r)

	got, err := g.GetRunByID(context.Background(), "r1")
	require.NoError(t, err)
	assert.Same(t, r, got)
}

func TestRunStore_GetRunByID_DBMissThenCached(t *testing.T) {
	mockRuns := db.NewMockRuns(t)
	r := testRun("r1")
	// .Once(): a second lookup must come from the cache, not the DB.
	mockRuns.On("GetRunByID", mock.Anything, "r1").Return(r, nil).Once()

	g := NewRunStore(&db.Client{Runs: mockRuns})

	got, err := g.GetRunByID(context.Background(), "r1")
	require.NoError(t, err)
	assert.Same(t, r, got)

	cached, err := g.GetRunByID(context.Background(), "r1")
	require.NoError(t, err)
	assert.Same(t, r, cached)
}

func TestRunStore_GetRunByID_NotFound(t *testing.T) {
	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRunByID", mock.Anything, "missing").Return(nil, nil)

	g := NewRunStore(&db.Client{Runs: mockRuns})

	_, err := g.GetRunByID(context.Background(), "missing")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestRunStore_RegisterListener_FiresOnUpdate(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")
	g.AddRun(r)

	var received []types.RunChange
	g.RegisterListener(func(c types.RunChange) { received = append(received, c) })

	require.NoError(t, g.AddRunChanges(r, planQueued()))

	require.Len(t, received, 1)
	assert.Equal(t, r, received[0].Run)
	require.Len(t, received[0].NodeStatusChanges, 1)
}

// TestRunStore_RemoveListener_ByIdentity is the regression test for listener
// removal: removing listeners out of registration order must remove exactly the
// intended listeners. (A captured-index implementation would mis-remove once the
// slice shifted.)
func TestRunStore_RemoveListener_ByIdentity(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")
	g.AddRun(r)

	var a, b, c int
	removeA := g.RegisterListener(func(types.RunChange) { a++ })
	removeB := g.RegisterListener(func(types.RunChange) { b++ })
	g.RegisterListener(func(types.RunChange) { c++ })

	// Remove the first two out of order; only the third should remain.
	removeA()
	removeB()
	removeA() // removing an already-removed listener is a no-op (must not panic)

	require.NoError(t, g.AddRunChanges(r, planQueued()))

	assert.Equal(t, 0, a, "removed listener A must not fire")
	assert.Equal(t, 0, b, "removed listener B must not fire")
	assert.Equal(t, 1, c, "surviving listener C must fire exactly once")
}

func TestRunStore_AddRunChanges_NotInCache(t *testing.T) {
	g := NewRunStore(&db.Client{})
	err := g.AddRunChanges(testRun("not-added"), planQueued())
	assert.Error(t, err)
}

func TestRunStore_UpdateRun_MergesChangesByRunID(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")
	g.AddRun(r)

	require.NoError(t, g.AddRunChanges(r, planQueued()))
	require.NoError(t, g.AddRunChanges(r, statemachine.PlanStatusChange{OldStatus: models.PlanQueued, NewStatus: models.PlanRunning}))

	changes := g.GetChanges()
	require.Len(t, changes, 1, "changes for the same run are merged into one entry")
	assert.Equal(t, "r1", changes[0].Run.Metadata.ID)
	assert.Len(t, changes[0].NodeStatusChanges, 2)
}

// TestRunStore_GetChanges_IsolatedFromLaterMerges is the regression test for the
// double job-creation bug: GetChanges hands its result to the transform fixpoint
// loop, which iterates it while transformers record new changes. A merge performed
// after the call (addChanges appends into the accumulated entries) must NOT become
// visible through the previously returned list — otherwise a change recorded
// mid-iteration is delivered to the remaining transformers immediately AND again on
// the next iteration (e.g. creating two jobs for one plan-queued transition).
func TestRunStore_GetChanges_IsolatedFromLaterMerges(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")
	g.AddRun(r)

	require.NoError(t, g.AddRunChanges(r, planQueued()))

	snapshot := g.GetChanges()
	require.Len(t, snapshot, 1)
	require.Len(t, snapshot[0].NodeStatusChanges, 1)

	// A merge for the same run after the call must not leak into the snapshot.
	require.NoError(t, g.AddRunChanges(r, statemachine.PlanStatusChange{OldStatus: models.PlanQueued, NewStatus: models.PlanRunning}))

	assert.Len(t, snapshot[0].NodeStatusChanges, 1, "previously returned changes must not grow on later merges")

	// A fresh call reflects the merged set.
	merged := g.GetChanges()
	require.Len(t, merged, 1)
	assert.Len(t, merged[0].NodeStatusChanges, 2)
}

func TestRunStore_GetChangedNodeIDsForRun(t *testing.T) {
	g := NewRunStore(&db.Client{})
	r := testRun("r1")
	g.AddRun(r) // snapshots a copy of the run as the baseline

	// Unchanged run => no changed nodes.
	ids, err := g.GetChangedNodeIDsForRun("r1")
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Mutating the plan node should surface its ID as changed.
	r.Plan.Status = models.PlanQueued
	ids, err = g.GetChangedNodeIDsForRun("r1")
	require.NoError(t, err)
	assert.Contains(t, ids, r.Plan.ID)

	// Unknown run => error.
	_, err = g.GetChangedNodeIDsForRun("missing")
	assert.Error(t, err)
}
