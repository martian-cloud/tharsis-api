package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// fakeCommand is a test double implementing types.Command and types.Preparer. It
// records the order in which Prepare and Execute are invoked relative to the tx.
type fakeCommand struct {
	prepareCalls int
	executeCalls int
	prepareErr   error
	executeErr   error

	// hooks let a test observe state at call time (e.g. whether the tx had begun).
	onPrepare func()
	onExecute func()
}

func (c *fakeCommand) Prepare(_ context.Context) error {
	c.prepareCalls++
	if c.onPrepare != nil {
		c.onPrepare()
	}
	return c.prepareErr
}

func (c *fakeCommand) Execute(_ context.Context, _ *types.ExecuteInput) error {
	c.executeCalls++
	if c.onExecute != nil {
		c.onExecute()
	}
	return c.executeErr
}

func TestProcessCommand_Success(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.NewForTest()

	var (
		txBegun     bool
		prepareSeen bool
	)

	mockTransactions := db.NewMockTransactions(t)
	mockTransactions.On("BeginTx", mock.Anything).Run(func(_ mock.Arguments) {
		txBegun = true
	}).Return(ctx, nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)
	// RollbackTx is always deferred; after a successful commit it is a no-op on the db side.
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

	dbClient := &db.Client{Transactions: mockTransactions}

	cmd := &fakeCommand{
		onPrepare: func() {
			// Prepare must run before the tx is opened.
			prepareSeen = true
			assert.False(t, txBegun, "Prepare should run before BeginTx")
		},
		onExecute: func() {
			// Execute must run inside the tx.
			assert.True(t, txBegun, "Execute should run inside the tx")
		},
	}

	processor := NewCmdProcessor(log, dbClient, nil, nil, nil)

	err := processor.ProcessCommand(ctx, cmd)
	assert.NoError(t, err)

	assert.True(t, prepareSeen)
	assert.Equal(t, 1, cmd.prepareCalls, "Prepare should run exactly once")
	assert.Equal(t, 1, cmd.executeCalls, "Execute should run exactly once")
	mockTransactions.AssertCalled(t, "CommitTx", mock.Anything)
}

func TestProcessCommand_PrepareError(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.NewForTest()

	// No transaction mock expectations: a Prepare failure must short-circuit before
	// the tx is ever opened.
	mockTransactions := db.NewMockTransactions(t)
	dbClient := &db.Client{Transactions: mockTransactions}

	wantErr := errors.New("prepare failed")
	cmd := &fakeCommand{prepareErr: wantErr}

	processor := NewCmdProcessor(log, dbClient, nil, nil, nil)

	err := processor.ProcessCommand(ctx, cmd)
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 1, cmd.prepareCalls)
	assert.Equal(t, 0, cmd.executeCalls)
	mockTransactions.AssertNotCalled(t, "BeginTx", mock.Anything)
}

func TestProcessCommand_ExecuteErrorRollsBackWithoutCommit(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.NewForTest()

	mockTransactions := db.NewMockTransactions(t)
	mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	// CommitTx is intentionally not expected.

	dbClient := &db.Client{Transactions: mockTransactions}

	// A non-OLE error so RetryOnOLE does not retry.
	wantErr := errors.New("execute failed")
	cmd := &fakeCommand{executeErr: wantErr}

	processor := NewCmdProcessor(log, dbClient, nil, nil, nil)

	err := processor.ProcessCommand(ctx, cmd)
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 1, cmd.executeCalls)

	mockTransactions.AssertCalled(t, "RollbackTx", mock.Anything)
	mockTransactions.AssertNotCalled(t, "CommitTx", mock.Anything)
}

// storeCommand is a test double whose Execute interacts with the run store.
type storeCommand struct {
	execute func(ctx context.Context, input *types.ExecuteInput) error
}

func (c *storeCommand) Execute(ctx context.Context, input *types.ExecuteInput) error {
	return c.execute(ctx, input)
}

// transformerFunc adapts a function to the types.Transformer interface.
type transformerFunc func(ctx context.Context, changes []types.RunChange, runStore types.RunStore) error

func (f transformerFunc) Transform(ctx context.Context, changes []types.RunChange, runStore types.RunStore) error {
	return f(ctx, changes, runStore)
}

// TestProcessCommand_TransformersSeeEachChangeExactlyOnce is the regression test for
// the double job-creation bug: a change recorded by one transformer mid-iteration
// must reach the other transformers exactly once (on the next fixpoint iteration) —
// not immediately via aliasing of the store's accumulated change list AND again on
// the next iteration. T1 mirrors the admission transformer (reacts to plan-pending
// by recording plan-queued); T2 mirrors the job-creation transformer (counts
// plan-queued deliveries — with double delivery it would create two jobs).
func TestProcessCommand_TransformersSeeEachChangeExactlyOnce(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.NewForTest()

	mockTransactions := db.NewMockTransactions(t)
	mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	dbClient := &db.Client{Transactions: mockTransactions}

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunQueuing,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanPending},
	}

	// The command registers the run and records the initial plan-pending change,
	// mirroring run creation. The model itself is not mutated, so the processor's
	// save loop has nothing to persist.
	cmd := &storeCommand{execute: func(_ context.Context, input *types.ExecuteInput) error {
		input.RunStore.AddRun(run)
		return input.RunStore.AddRunChanges(run, statemachine.PlanStatusChange{OldStatus: models.PlanCreated, NewStatus: models.PlanPending})
	}}

	var pendingReactions, queuedDeliveries int
	t1 := transformerFunc(func(_ context.Context, changes []types.RunChange, runStore types.RunStore) error {
		for _, change := range changes {
			for _, sc := range change.NodeStatusChanges {
				if c, ok := sc.(statemachine.PlanStatusChange); ok && c.NewStatus == models.PlanPending {
					pendingReactions++
					if err := runStore.AddRunChanges(change.Run, statemachine.PlanStatusChange{OldStatus: models.PlanPending, NewStatus: models.PlanQueued}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	t2 := transformerFunc(func(_ context.Context, changes []types.RunChange, _ types.RunStore) error {
		for _, change := range changes {
			for _, sc := range change.NodeStatusChanges {
				if c, ok := sc.(statemachine.PlanStatusChange); ok && c.NewStatus == models.PlanQueued {
					queuedDeliveries++
				}
			}
		}
		return nil
	})

	processor := NewCmdProcessor(log, dbClient, []types.Transformer{t1, t2}, nil, nil)

	require.NoError(t, processor.ProcessCommand(ctx, cmd))

	assert.Equal(t, 1, pendingReactions, "the pending change must be delivered to T1 exactly once")
	assert.Equal(t, 1, queuedDeliveries, "T1's queued change must be delivered to T2 exactly once")
}

// recordingHandler records the changes it receives so the test can assert when
// (relative to the commit) each handler kind runs.
type recordingHandler struct {
	called *bool
}

func (h *recordingHandler) HandleRunChanges(_ context.Context, _ []types.RunChange) error {
	*h.called = true
	return nil
}

func TestProcessCommand_RunsStatefulBeforeCommitAndStatelessAfter(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.NewForTest()

	var (
		committed     bool
		statefulRun   bool
		statelessRun  bool
		statefulOK    bool
		statelessAtOK bool
	)

	mockTransactions := db.NewMockTransactions(t)
	mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Run(func(_ mock.Arguments) {
		// At commit time the stateful handler must already have run; the stateless one not yet.
		statefulOK = statefulRun
		statelessAtOK = !statelessRun
		committed = true
	}).Return(nil)

	dbClient := &db.Client{Transactions: mockTransactions}

	stateful := &recordingHandler{called: &statefulRun}
	stateless := &recordingHandler{called: &statelessRun}

	processor := NewCmdProcessor(log, dbClient,
		nil,
		[]types.RunChangeHandler{stateful},
		[]types.RunChangeHandler{stateless},
	)

	err := processor.ProcessCommand(ctx, &fakeCommand{})
	assert.NoError(t, err)

	assert.True(t, committed)
	assert.True(t, statefulRun, "stateful handler should have run")
	assert.True(t, statelessRun, "stateless handler should have run")
	assert.True(t, statefulOK, "stateful handler must run before commit")
	assert.True(t, statelessAtOK, "stateless handler must run after commit")
}
