// Package commands implements the run engine's commands (create, queue, start
// apply, cancel, update plan/apply, sync job status), each split into a pre-tx
// Prepare phase and an in-tx Execute phase run by the command processor.
package commands

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// forceCancelWait is how long a run must be soft-canceled before it is allowed to be forcefully canceled.
const forceCancelWait = 15 * time.Second

// CancelRunInput carries everything CancelRun needs across its Prepare and
// Execute phases.
type CancelRunInput struct {
	RunID      string
	CanceledBy string
	Force      bool
}

// CancelRun cancels a run (graceful or forced). It cancels the run's jobs,
// transitions the run state machine, and records the cancel activity event in a
// single transaction.
type CancelRun struct {
	dbClient *db.Client
	in       *CancelRunInput

	// Populated by Prepare.
	namespacePath string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare resolves the workspace namespace path used for the cancel activity
// event. It runs before the transaction is opened.
func (c *CancelRun) Prepare(ctx context.Context) error {
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

// Execute executes the cancel run command.
func (c *CancelRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	// Verify run is in a valid state to be canceled
	switch run.Status {
	case models.RunPlannedAndFinished:
		return errors.New("run has been planned and finished, so it cannot be canceled", errors.WithErrorCode(errors.EInvalid))
	case models.RunApplied:
		return errors.New("run has been applied, so it cannot be canceled", errors.WithErrorCode(errors.EInvalid))
	case models.RunErrored:
		return errors.New("run has errored, so it cannot be canceled", errors.WithErrorCode(errors.EInvalid))
	case models.RunCanceled:
		return errors.New("run has already been canceled", errors.WithErrorCode(errors.EInvalid))
	case models.RunDiscarded:
		return errors.New("run has been discarded, so it cannot be canceled", errors.WithErrorCode(errors.EInvalid))
	}

	// If this is a force cancel request, verify graceful cancel was already attempted
	if c.in.Force {
		if run.ForceCancelAvailableAt == nil {
			return errors.New("run has not already received a graceful request to cancel", errors.WithErrorCode(errors.EInvalid))
		}
		if time.Now().Before(*run.ForceCancelAvailableAt) {
			return errors.New("insufficient time has elapsed since graceful cancel request; force cancel will be available at %s", *run.ForceCancelAvailableAt, errors.WithErrorCode(errors.EInvalid))
		}
	}

	// Record the force-cancel bookkeeping: a graceful request makes force-cancel
	// available after a delay; a force request marks the run as force-canceled.
	if c.in.Force {
		run.ForceCanceled = true
		run.ForceCanceledBy = &c.in.CanceledBy
	} else if run.ForceCancelAvailableAt == nil {
		forceCancelAt := time.Now().UTC().Add(forceCancelWait)
		run.ForceCancelAvailableAt = &forceCancelAt
	}

	// Cancel the run's active phase: its job, and its node once the work has stopped.
	changes, err := cancelActivePhase(ctx, c.dbClient, run, c.in.Force)
	if err != nil {
		return err
	}

	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}

	if _, err := activity.CreateActivityEvent(ctx, c.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &c.namespacePath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetRun,
		TargetID:      run.Metadata.ID,
		Payload: &models.ActivityEventUpdateRunPayload{
			Type: string(models.RunUpdateTypeCancel),
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event")
	}

	c.Updated = run
	return nil
}

// cancelActivePhase cancels the run's earliest non-final phase (plan, then apply).
// It cancels that phase's job and, once the work has actually stopped — the job is
// gone or already final, or this is a force cancel — transitions the node to
// canceled, which cascades the run (and any not-yet-started downstream node) to
// canceled. For a graceful cancel of a still-running job it only requests
// cancellation and leaves the node in place, so the run keeps reporting its real
// progress; the node is canceled later when the runner confirms the job canceled via
// the normal job-status sync.
func cancelActivePhase(ctx context.Context, dbClient *db.Client, run *models.Run, force bool) ([]statemachine.NodeStatusChange, error) {
	switch {
	case !run.Plan.Status.IsFinalStatus():
		cancelNode, err := cancelLatestJob(ctx, dbClient, run.Plan.LatestJobID, force)
		if err != nil {
			return nil, err
		}
		if !cancelNode {
			return nil, nil
		}
		changes, err := statemachine.SetPlanStatus(run, models.PlanCanceled)
		if err != nil {
			return nil, errors.Wrap(err, "failed to cancel run plan")
		}
		return changes, nil
	case run.Apply != nil && !run.Apply.Status.IsFinalStatus():
		cancelNode, err := cancelLatestJob(ctx, dbClient, run.Apply.LatestJobID, force)
		if err != nil {
			return nil, err
		}
		if !cancelNode {
			return nil, nil
		}
		changes, err := statemachine.SetApplyStatus(run, models.ApplyCanceled)
		if err != nil {
			return nil, errors.Wrap(err, "failed to cancel run apply")
		}
		return changes, nil
	default:
		return nil, nil
	}
}

// cancelLatestJob cancels a phase's latest job (when present and not already final)
// for a run cancellation and reports whether the phase's node should be transitioned
// to canceled now. A force cancel stops the job immediately. A graceful cancel stops
// a queued/pending job immediately (no work has started), but only requests
// cancellation of a running job (job -> canceling) and leaves the node running. When
// there is no job, or it is already final, the node is canceled now.
func cancelLatestJob(ctx context.Context, dbClient *db.Client, jobID *string, force bool) (bool, error) {
	if jobID == nil {
		return true, nil
	}
	job, err := dbClient.Jobs.GetJobByID(ctx, *jobID)
	if err != nil {
		return false, err
	}
	if job == nil || job.GetStatus().IsFinal() {
		return true, nil
	}

	now := time.Now().UTC()
	cancelNode := true
	switch {
	case force:
		if err := job.SetStatus(models.JobCanceled); err != nil {
			return false, err
		}
		job.ForceCanceled = true
	case job.GetStatus() == models.JobRunning:
		// The job is executing; request cancellation and wait for the runner to
		// confirm before transitioning the node.
		if err := job.SetStatus(models.JobCanceling); err != nil {
			return false, err
		}
		job.CancelRequestedTimestamp = &now
		cancelNode = false
	default: // queued or pending: nothing is executing, so cancel immediately.
		if err := job.SetStatus(models.JobCanceled); err != nil {
			return false, err
		}
		job.CancelRequestedTimestamp = &now
	}

	if _, err := dbClient.Jobs.UpdateJob(ctx, job); err != nil {
		return false, err
	}

	// A job canceled here reaches its final state without a runner ever reporting on
	// it, so the job-status sync that normally completes the log stream will never
	// run. Complete the stream now so subscribers aren't left waiting on a stream
	// that will receive no more logs. (A graceful cancel of a running job is only
	// moved to canceling — not final — and its stream is completed by the job-status
	// sync once the runner confirms.)
	if job.GetStatus() == models.JobCanceled {
		if err := completeJobLogStream(ctx, dbClient, job.Metadata.ID); err != nil {
			return false, err
		}
	}
	return cancelNode, nil
}

// completeJobLogStream marks the job's log stream as completed, if one exists.
func completeJobLogStream(ctx context.Context, dbClient *db.Client, jobID string) error {
	logStream, err := dbClient.LogStreams.GetLogStreamByJobID(ctx, jobID)
	if err != nil {
		return err
	}
	if logStream == nil {
		return nil
	}
	logStream.Completed = true
	if _, err := dbClient.LogStreams.UpdateLogStream(ctx, logStream); err != nil {
		return err
	}
	return nil
}
