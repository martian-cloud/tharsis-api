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
	NodePath string // "plan", "apply", or a policy check path
}

// RetryRunNode resets a failed or canceled plan/apply node — or a failed, canceled, or soft-failed
// policy check node — back to pending. Once the node is pending the existing pipeline takes over: the
// admission transformer re-queues it and the job-creation transformer creates a new job. Prepare
// resolves the workspace namespace path used for the retry activity event; the reset itself is an
// in-transaction state change recorded alongside the event in Execute.
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

// Execute resets the node so it is retried, validating it is in a retryable state: failed or canceled
// for the plan and apply, and additionally soft-failed for a policy check. The run status is updated
// automatically by the state machine's pending listeners (a retry returns the run to the retried node's
// *_queuing status, so it is re-admitted to the workspace before running again — except the ungated
// post-plan stage, which restarts directly on post_plan_running), the retried node's error message is
// cleared, and any prior force-cancellation state on the run is reset. Retrying a policy check also
// clears the previous attempt's per-policy verdicts and, via RunGateManager, deletes the gate the check
// was blocked on.
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
		// A policy check node path.
		check := run.PolicyCheckByPath(c.in.NodePath)
		if check == nil {
			return errors.New("invalid node path %q, must be \"plan\", \"apply\", or a policy check path", c.in.NodePath, errors.WithErrorCode(errors.EInvalid))
		}
		if check.Status != models.PolicyCheckErrored &&
			check.Status != models.PolicyCheckCanceled &&
			check.Status != models.PolicyCheckSoftFailed {
			return errors.New("policy check node can only be retried when it is failed, canceled, or soft-failed", errors.WithErrorCode(errors.EConflict))
		}
		// A discarded run is revived by undiscarding it, not by retrying a node under it: discard cancels
		// the check the run was blocked on (along with its gate), and undiscarding is what re-runs it
		// (UndiscardRun). A canceled check is otherwise retryable, and the run status has an edge out of
		// discarded for the restarted stage to project onto, so without this guard a retry would quietly
		// revive a discarded run — leaving it active with no undiscard activity event recorded. The plan
		// and apply branches above need no such guard: a discarded run's plan is finished or skipped and
		// its apply is skipped, so neither is in a retryable state to begin with.
		if run.Status == models.RunDiscarded {
			return errors.New("policy check node cannot be retried while the run is discarded, undiscard the run first", errors.WithErrorCode(errors.EConflict))
		}
		// Setting the check to pending is the whole retry: its stage restarts itself from there and
		// re-queues the check, so the job-creation transformer makes a fresh policy-eval job. The check
		// is not set straight to queued because that would create the job immediately, and the retried
		// run has already released its workspace slot — a gated stage returns to pending and must be
		// re-admitted first.
		changes, err = statemachine.SetPolicyCheckStatus(run, check.GetPath(), models.PolicyCheckPending)
		if err != nil {
			return err
		}
		// The previous attempt's verdicts describe an evaluation that is being thrown away, so return the
		// policies to the state run creation pins them in. ReportRunPolicyOutcomes only stamps the
		// policies the evaluator reports back, so a leftover "failed" would otherwise survive a
		// re-evaluation that passed it — and go on to build an approval rule for it in RunGateManager the
		// next time the check soft-fails.
		for _, policy := range check.Policies {
			policy.Status = models.PolicyCheckPolicyPending
			// The messages objects themselves stay linked to the run and are collected with it; a
			// re-evaluation uploads new ones rather than overwriting these.
			policy.MessagesObjectStoreKey = nil
		}
		check.MessagesSummary = nil
		// The advisory failures this check reported are part of the evaluation being thrown away, but
		// another check may still hold its own, so the run-level flag is recomputed rather than cleared.
		run.HasAdvisoryFailures = run.ComputeHasAdvisoryFailures()
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
