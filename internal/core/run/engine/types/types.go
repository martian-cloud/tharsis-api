// Package types provides type definitions for run services.
package types

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// RunChange represents a change to a run
type RunChange struct {
	Run               *models.Run
	NodeStatusChanges []statemachine.NodeStatusChange
}

// RunChangeHandler handles run changes
type RunChangeHandler interface {
	HandleRunChanges(ctx context.Context, changes []RunChange) error
}

// RunChangeListener listens for run changes
type RunChangeListener func(change RunChange)

// RunStore manages run state and changes
type RunStore interface {
	RegisterListener(listener RunChangeListener) func()
	AddRun(run *models.Run)
	GetRunByID(ctx context.Context, runID string) (*models.Run, error)
	AddRunChanges(run *models.Run, changes ...statemachine.NodeStatusChange) error
	GetChanges() []RunChange
}

// Transformer transforms run changes
type Transformer interface {
	Transform(ctx context.Context, changeList []RunChange, runStore RunStore) error
}

// ExecuteInput contains the per-execution dependencies a command needs. Stable
// dependencies (the DB client, the admitter) are injected into commands by the
// factory; only the per-execution run store is passed here.
type ExecuteInput struct {
	RunStore RunStore
}

// Command executes run operations
type Command interface {
	Execute(ctx context.Context, input *ExecuteInput) error
}

// Preparer is an optional capability a Command may implement to run slow,
// read-only resolution (DB reads, registry/network I/O) BEFORE the transaction is
// opened, so that work does not hold the transaction open. Prepare runs once,
// outside the OLE retry loop; results are stashed on the command for Execute to
// consume.
type Preparer interface {
	Prepare(ctx context.Context) error
}
