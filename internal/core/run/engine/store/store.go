// Package store caches run handles and brokers run-change events for the run
// engine. It lets commands read and mutate a run in memory for the duration of
// an engine pass instead of round-tripping the DB, and notifies registered
// listeners as those mutations are recorded.
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// listenerEntry pairs a listener with a stable id so it can be removed by identity
// rather than by a captured slice index (which goes stale once another listener is
// removed).
type listenerEntry struct {
	id string
	fn types.RunChangeListener
}

// RunStore caches the live run handles for an engine pass and records the
// resulting change stream. The runs it hands out are shared, mutable pointers
// (see GetRunByID), so callers mutate run state in place and the store diffs
// those mutations against pristine copies to surface changed nodes.
type RunStore struct {
	dbClient  *db.Client
	cache     map[string]*models.Run
	copies    map[string]*models.Run
	listeners []listenerEntry
	changes   []types.RunChange
}

// NewRunStore creates a new RunStore
func NewRunStore(dbClient *db.Client) *RunStore {
	return &RunStore{
		dbClient: dbClient,
		cache:    map[string]*models.Run{},
		copies:   map[string]*models.Run{},
	}
}

// RegisterListener registers a listener for run changes. Listeners receive the raw,
// pre-merge change stream: they may see multiple changes for the same run (one per
// UpdateRun call), unlike GetChanges, which returns the merged set with one entry per
// run. The returned function removes that specific listener (by id), so removals are
// safe regardless of order.
func (s *RunStore) RegisterListener(listener types.RunChangeListener) func() {
	id := uuid.NewString()
	s.listeners = append(s.listeners, listenerEntry{id: id, fn: listener})
	return func() {
		for i, e := range s.listeners {
			if e.id == id {
				s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
				return
			}
		}
	}
}

// GetRuns returns the live run handles currently cached in the store. Like
// GetRunByID, these are shared, mutable pointers, not copies.
func (s *RunStore) GetRuns() []*models.Run {
	response := []*models.Run{}
	for _, run := range s.cache {
		response = append(response, run)
	}
	return response
}

// AddRun caches a run handle (a no-op if one is already cached for that ID).
// It also stashes a pristine Copy of the run so GetChangedNodeIDsForRun can
// later diff the live, mutated handle against the state at insertion time.
func (s *RunStore) AddRun(run *models.Run) {
	if _, ok := s.cache[run.Metadata.ID]; ok {
		return
	}

	s.cache[run.Metadata.ID] = run
	s.copies[run.Metadata.ID] = run.Copy()
}

// GetRunByID returns the run with the given ID, loading it from the DB and
// caching it on first access. The returned *models.Run is the store's live
// handle, NOT a copy: callers may mutate it directly (e.g. drive state
// transitions) and those mutations are visible to everyone else holding the
// handle this pass and are what GetChangedNodeIDsForRun diffs against the
// pristine copy taken in AddRun. Because the handle is shared and scoped to the
// engine pass, callers must not treat it as a read-only snapshot or retain it
// beyond the pass.
func (s *RunStore) GetRunByID(ctx context.Context, runID string) (*models.Run, error) {
	if run, ok := s.cache[runID]; ok {
		return run, nil
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, errors.New("run not found: %s", runID, errors.WithErrorCode(errors.ENotFound))
	}

	s.AddRun(run)

	return run, nil
}

// AddRunChanges records status changes for a run
func (s *RunStore) AddRunChanges(run *models.Run, changes ...statemachine.NodeStatusChange) error {
	if _, ok := s.cache[run.Metadata.ID]; !ok {
		return fmt.Errorf("run %s not found in store", run.Metadata.ID)
	}

	s.addChanges([]types.RunChange{{
		Run:               run,
		NodeStatusChanges: changes,
	}})

	return nil
}

// GetChanges returns all run changes. It returns a copy: addChanges merges later
// changes into the accumulated entries in place, and callers (e.g. the transform
// fixpoint loop) iterate the returned list while transformers record new changes.
// Without the copy those mid-iteration merges would leak into the caller's view and
// changes would be delivered twice (e.g. two jobs created for one plan-queued
// transition). Listener delivery is the only channel for changes recorded after the
// call.
func (s *RunStore) GetChanges() []types.RunChange {
	changes := make([]types.RunChange, len(s.changes))
	copy(changes, s.changes)
	return changes
}

// GetChangedNodeIDsForRun returns the IDs of nodes that have changed for a given run.
func (s *RunStore) GetChangedNodeIDsForRun(runID string) ([]string, error) {
	run, ok := s.cache[runID]
	if !ok {
		return nil, fmt.Errorf("run %s not found in cache", runID)
	}
	original, ok := s.copies[runID]
	if !ok {
		return nil, fmt.Errorf("original run %s not found in cache", runID)
	}

	return run.Diff(original), nil
}

func (s *RunStore) addChanges(changesToAdd []types.RunChange) {
	if len(changesToAdd) == 0 {
		return
	}

	// Fire listeners
	for _, entry := range s.listeners {
		for _, change := range changesToAdd {
			entry.fn(types.RunChange{
				Run:               change.Run,
				NodeStatusChanges: change.NodeStatusChanges,
			})
		}
	}

	// Merge into the accumulated changes, keeping one entry per run in first-seen
	// order so GetChanges returns a deterministic sequence for the change handlers.
	index := map[string]int{}
	for i, change := range s.changes {
		index[change.Run.Metadata.ID] = i
	}
	for _, newChange := range changesToAdd {
		if i, ok := index[newChange.Run.Metadata.ID]; ok {
			s.changes[i].NodeStatusChanges = append(s.changes[i].NodeStatusChanges, newChange.NodeStatusChanges...)
		} else {
			index[newChange.Run.Metadata.ID] = len(s.changes)
			s.changes = append(s.changes, newChange)
		}
	}
}
