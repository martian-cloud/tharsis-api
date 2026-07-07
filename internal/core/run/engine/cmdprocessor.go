// Package engine drives run state transitions: it processes run commands within a
// transaction (applying transformers and change handlers) and consumes queued work
// items to advance pending runs.
package engine

//go:generate go tool mockery --name CmdProcessor --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// maxTransformIterations bounds the transform fixpoint loop. Each iteration exists
// only because a transformer produced new changes for the next one to react to; the
// deepest legitimate cascade today is a handful of steps (e.g. plan finished →
// auto-apply readies the apply → admission queues it → job creation), so hitting
// this cap means transformers are feeding each other in a cycle.
const maxTransformIterations = 10

// CmdProcessor manages the execution of run commands
type CmdProcessor interface {
	ProcessCommand(ctx context.Context, cmd types.Command) error
}

type cmdProcessor struct {
	logger                  logger.Logger
	dbClient                *db.Client
	statefulChangeHandlers  []types.RunChangeHandler
	statelessChangeHandlers []types.RunChangeHandler
	transformers            []types.Transformer
}

// NewCmdProcessor creates a run command processor
func NewCmdProcessor(
	logger logger.Logger,
	dbClient *db.Client,
	transformers []types.Transformer,
	statefulChangeHandlers []types.RunChangeHandler,
	statelessChangeHandlers []types.RunChangeHandler,
) CmdProcessor {
	return &cmdProcessor{
		logger:                  logger,
		dbClient:                dbClient,
		transformers:            transformers,
		statefulChangeHandlers:  statefulChangeHandlers,
		statelessChangeHandlers: statelessChangeHandlers,
	}
}

func (c *cmdProcessor) ProcessCommand(ctx context.Context, cmd types.Command) error {
	var changes []types.RunChange

	// Run any read-only preparation before opening the transaction so slow work
	// (reads, registry/network resolution) doesn't hold the transaction open and
	// isn't repeated on each OLE retry. Results are stashed on the command for
	// Execute to consume.
	if p, ok := cmd.(types.Preparer); ok {
		if err := p.Prepare(ctx); err != nil {
			return err
		}
	}

	if rErr := c.dbClient.RetryOnOLE(ctx, func() error {
		txCtx, err := c.dbClient.Transactions.BeginTx(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to begin command tx")
		}

		defer func() {
			if txErr := c.dbClient.Transactions.RollbackTx(txCtx); txErr != nil {
				c.logger.Errorf("failed to rollback command tx in run CmdProcessor: %v", txErr)
			}
		}()

		runStore := store.NewRunStore(c.dbClient)

		if err := cmd.Execute(txCtx, &types.ExecuteInput{
			RunStore: runStore,
		}); err != nil {
			return err
		}

		// Apply transforms
		if err := c.applyTransforms(txCtx, runStore.GetChanges(), runStore); err != nil {
			return err
		}

		// Save all updated runs
		for _, r := range runStore.GetRuns() {
			modifiedNodeIDs, err := runStore.GetChangedNodeIDsForRun(r.Metadata.ID)
			if err != nil {
				return err
			}

			if len(modifiedNodeIDs) > 0 {
				if _, err := c.dbClient.Runs.UpdateRun(txCtx, r, modifiedNodeIDs...); err != nil {
					return err
				}
			}
		}

		changes = runStore.GetChanges()

		for _, h := range c.statefulChangeHandlers {
			if err := h.HandleRunChanges(txCtx, changes); err != nil {
				return err
			}
		}

		if err := c.dbClient.Transactions.CommitTx(txCtx); err != nil {
			return errors.Wrap(err, "failed to commit command tx")
		}

		return nil
	}); rErr != nil {
		return rErr
	}

	// Stateless change handlers run after the tx has been committed. The command has already
	// succeeded, so a handler failure is logged rather than returned (and does not stop the
	// remaining handlers).
	for _, h := range c.statelessChangeHandlers {
		if err := h.HandleRunChanges(ctx, changes); err != nil {
			runIDs := make([]string, 0, len(changes))
			for _, change := range changes {
				runIDs = append(runIDs, change.Run.Metadata.ID)
			}
			c.logger.WithContextFields(ctx).Errorf(
				"stateless run change handler %T failed for runs %v: %v", h, runIDs, err)
		}
	}

	return nil
}

// applyTransforms runs the transformers over the changes repeatedly until they
// produce no new changes (a fixpoint), so a change made by one transformer is seen
// by all of them on the next iteration regardless of registration order.
func (c *cmdProcessor) applyTransforms(ctx context.Context, changes []types.RunChange, runStore types.RunStore) error {
	for iteration := 0; len(changes) > 0; iteration++ {
		if iteration >= maxTransformIterations {
			runIDs := make([]string, 0, len(changes))
			for _, change := range changes {
				runIDs = append(runIDs, change.Run.Metadata.ID)
			}
			return errors.New("run transformers did not converge after %d iterations; runs still changing: %v", maxTransformIterations, runIDs)
		}

		transformerChanges := []types.RunChange{}
		removeListener := runStore.RegisterListener(func(change types.RunChange) {
			transformerChanges = append(transformerChanges, change)
		})

		for _, transformer := range c.transformers {
			if err := transformer.Transform(ctx, changes, runStore); err != nil {
				removeListener()
				return err
			}
		}

		removeListener()
		changes = transformerChanges
	}

	return nil
}
