package transformers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// AutoApplyTransformer makes sure the apply node of an auto-apply run records who
// triggered it when the apply enters the pending state. For an auto-apply run the
// state machine moves the apply to pending itself (when the plan finishes with
// changes), so reacting to that transition keeps the state machine as the single
// source of truth; the AdmissionTransformer then queues it.
//
// The principal who authorized the apply is normally recorded already: the run's
// creator when it was created with auto-apply, or whoever enabled auto-apply
// afterwards. That principal is kept. The system is recorded only when nobody is,
// such as for a run created before TriggeredBy was set at authorization time.
type AutoApplyTransformer struct{}

// NewAutoApplyTransformer creates a new AutoApplyTransformer.
func NewAutoApplyTransformer() *AutoApplyTransformer {
	return &AutoApplyTransformer{}
}

// Transform records the system as the trigger on the apply node of an auto-apply run
// whose apply just transitioned to pending, unless a trigger is already recorded.
func (t *AutoApplyTransformer) Transform(_ context.Context, changeList []types.RunChange, _ types.RunStore) error {
	for _, change := range changeList {
		run := change.Run

		if !run.AutoApply || run.Speculative() || run.Apply == nil {
			continue
		}

		for _, sc := range change.NodeStatusChanges {
			if sc.GetNodeType() != statemachine.ApplyNodeType {
				continue
			}
			applyChange, ok := sc.(statemachine.ApplyStatusChange)
			if !ok {
				continue
			}

			if applyChange.NewStatus == models.ApplyPending && run.Apply.TriggeredBy == nil {
				run.Apply.TriggeredBy = new(models.SystemActor())
			}
		}
	}
	return nil
}
