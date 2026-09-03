package eventhandlers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// PolicyCheckWorkItemEnqueuer enqueues an EvaluateRunPolicyCheck work item when a policy check that
// evaluates in-API (module attestation today) enters queued. Unlike OPA, which JobCreationTransformer
// hands to a runner, this check type has no job: it is evaluated directly by the work item consumer.
// It is a stateful change handler so the work item commits atomically with the run change, mirroring
// how RunGateManager and JobCreationTransformer key off the same status change.
type PolicyCheckWorkItemEnqueuer struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewPolicyCheckWorkItemEnqueuer creates a new PolicyCheckWorkItemEnqueuer.
func NewPolicyCheckWorkItemEnqueuer(logger logger.Logger, dbClient *db.Client) *PolicyCheckWorkItemEnqueuer {
	return &PolicyCheckWorkItemEnqueuer{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler.
func (h *PolicyCheckWorkItemEnqueuer) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	for _, change := range changes {
		run := change.Run
		for _, sc := range change.NodeStatusChanges {
			c, ok := sc.(statemachine.PolicyCheckStatusChange)
			if !ok || c.NewStatus != models.PolicyCheckQueued {
				continue
			}

			check := run.PolicyCheckByPath(c.Path)
			if check == nil || check.CheckType == models.PolicyKindOPA {
				// OPA is handed to a runner via JobCreationTransformer; nothing else to do here.
				continue
			}

			if check.Status != models.PolicyCheckQueued {
				// The check has already transitioned to a new state so there is nothing to do here
				continue
			}

			if _, err := h.dbClient.WorkItemsQueue.AddWorkItemToQueue(ctx, &db.AddWorkItemToQueueInput{
				Type: db.EvaluateRunPolicyCheckType,
				Payload: &db.EvaluateRunPolicyCheckPayload{
					RunID:         run.Metadata.ID,
					PolicyCheckID: check.ID,
				},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
