package transformers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// AutoApplyTransformer marks the apply node of an auto-apply run as triggered by
// "auto-apply" when the apply enters the pending state. For an auto-apply run the
// state machine moves the apply to pending itself (when the plan finishes with
// changes), so reacting to that transition keeps the state machine as the single
// source of truth; the AdmissionTransformer then queues it. This transformer only
// records who triggered it.
type AutoApplyTransformer struct{}

// NewAutoApplyTransformer creates a new AutoApplyTransformer.
func NewAutoApplyTransformer() *AutoApplyTransformer {
	return &AutoApplyTransformer{}
}

// Transform sets the auto-apply trigger on the apply node of an auto-apply run
// whose apply just transitioned to pending.
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

			if applyChange.NewStatus == models.ApplyPending {
				run.Apply.TriggeredBy = auth.System
			}
		}
	}
	return nil
}
