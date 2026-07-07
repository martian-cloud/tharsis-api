// Package transformers contains run-store transformers that react to node status
// changes within a command's transaction (e.g. admitting queued nodes, tagging
// auto-apply runs, and creating jobs for queued nodes).
package transformers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// AdmissionTransformer queues a plan or apply node as soon as it enters the
// pending state. Pending means "ready to run, waiting on the workspace", so this
// is the single place that turns that readiness into a queue attempt via the
// admitter. Commands and the state machine only need to move a node to pending;
// they no longer call the admitter directly. (Re-evaluating an already-pending
// node, e.g. when the work item consumer advances a parked run, has no pending transition
// to react to and is still driven explicitly by the QueueRun command.)
type AdmissionTransformer struct {
	admitter *admission.Admitter
}

// NewAdmissionTransformer creates a new AdmissionTransformer.
func NewAdmissionTransformer(admitter *admission.Admitter) *AdmissionTransformer {
	return &AdmissionTransformer{admitter: admitter}
}

// Transform attempts to queue any plan or apply node that just transitioned to
// pending.
//
// An optimistic-lock error from the admitter means another instance changed the
// workspace concurrently while we were acquiring it. That is swallowed rather than
// propagated: the node stays pending, the WorkspaceLockManager enqueues a work
// item for it, and the work item consumer retries admission later. Propagating it would
// instead force a retry of the entire surrounding command (e.g. run creation),
// which is wasteful and unnecessary.
func (t *AdmissionTransformer) Transform(ctx context.Context, changeList []types.RunChange, runStore types.RunStore) error {
	for _, change := range changeList {
		run := change.Run
		for _, sc := range change.NodeStatusChanges {
			switch c := sc.(type) {
			case statemachine.PlanStatusChange:
				if c.NewStatus != models.PlanPending {
					continue
				}
				queued, changes, err := t.admitter.TryQueuePlan(ctx, run)
				if err != nil {
					// Ignore OLE here since a work item will be queued in the ws lock manager event handler
					if isOptimisticLock(err) {
						continue
					}
					return err
				}
				if queued {
					if err := runStore.AddRunChanges(run, changes...); err != nil {
						return err
					}
				}
			case statemachine.ApplyStatusChange:
				if c.NewStatus != models.ApplyPending {
					continue
				}
				queued, changes, err := t.admitter.TryQueueApply(ctx, run)
				if err != nil {
					// Ignore OLE here since a work item will be queued in the ws lock manager event handler
					if isOptimisticLock(err) {
						continue
					}
					return err
				}
				if queued {
					if err := runStore.AddRunChanges(run, changes...); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func isOptimisticLock(err error) bool {
	return errors.ErrorCode(err) == errors.EOptimisticLock
}
