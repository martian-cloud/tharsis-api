package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// RetryRunNodeInput carries everything RetryRunNode needs.
type RetryRunNodeInput struct {
	RunID    string
	NodePath string // "plan" or "apply"
}

// RetryRunNode resets a failed or canceled plan/apply node back to pending. Once the
// node is pending the existing pipeline takes over: the admission transformer re-queues
// it and the job-creation transformer creates a new job. Prepare resolves the workspace
// namespace path used for the retry activity event; the reset itself is an in-transaction
// state change recorded alongside the event in Execute.
type RetryRunNode struct {
	dbClient *db.Client
	in       *RetryRunNodeInput

	// Populated by Prepare.
	namespacePath string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare resolves the workspace namespace path used for the retry activity event. It
// runs before the transaction is opened (Execute validates the run is in a retryable state).
func (c *RetryRunNode) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run")
	}
	if run == nil {
		return errors.New("run with ID %s not found", c.in.RunID, errors.WithErrorCode(errors.ENotFound))
	}

	ws, err := c.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace by ID")
	}
	if ws == nil {
		return errors.New("failed to get workspace ID %s associated with run ID %s", run.WorkspaceID, run.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	c.namespacePath = ws.FullPath
	return nil
}

// Execute resets the node to pending, validating it is in a retryable (failed/canceled)
// state. The run status is updated automatically by the state machine's pending
// listeners (a plan retry returns the run to queuing; an apply retry to queuing_apply),
// the retried node's error message is cleared, and any prior force-cancellation state
// on the run is reset.
func (c *RetryRunNode) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	var changes []statemachine.NodeStatusChange

	switch c.in.NodePath {
	case models.PlanNodePath:
		if run.Plan.Status != models.PlanErrored && run.Plan.Status != models.PlanCanceled {
			return errors.New("plan node can only be retried when it is failed or canceled", errors.WithErrorCode(errors.EConflict))
		}
		changes, err = statemachine.SetPlanStatus(run, models.PlanPending)
		if err != nil {
			return err
		}
		run.Plan.ErrorMessage = nil
	case models.ApplyNodePath:
		if run.Apply == nil {
			return errors.New("run does not have an apply node to retry", errors.WithErrorCode(errors.EInvalid))
		}
		if run.Apply.Status != models.ApplyErrored && run.Apply.Status != models.ApplyCanceled {
			return errors.New("apply node can only be retried when it is failed or canceled", errors.WithErrorCode(errors.EConflict))
		}
		changes, err = statemachine.SetApplyStatus(run, models.ApplyPending)
		if err != nil {
			return err
		}
		run.Apply.ErrorMessage = nil
	default:
		return errors.New("invalid node path %q, must be \"plan\" or \"apply\"", c.in.NodePath, errors.WithErrorCode(errors.EInvalid))
	}

	// Clear any prior force-cancellation so the retried run starts from a clean state.
	run.ForceCanceled = false
	run.ForceCanceledBy = nil
	run.ForceCancelAvailableAt = nil

	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}

	nodePath := c.in.NodePath
	if _, err := activity.CreateActivityEvent(ctx, c.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &c.namespacePath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetRun,
		TargetID:      run.Metadata.ID,
		Payload: &models.ActivityEventUpdateRunPayload{
			Type:     string(models.RunUpdateTypeRetry),
			NodePath: &nodePath,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event")
	}

	c.Updated = run
	return nil
}
