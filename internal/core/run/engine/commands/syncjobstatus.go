package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// SyncJobStatus maps a job status change to the corresponding run node.
type SyncJobStatus struct {
	logger    logger.Logger
	RunID     string
	JobType   models.JobType
	JobID     string
	NewStatus models.JobStatus

	// PersistJob is responsible for persisting the job status change (and any log-stream completion) within the command transaction.
	// This is supplied by the job service
	PersistJob func(ctx context.Context) error
}

// isStaleJob reports whether this command's job is not the node's current job — for
// example a superseded attempt after the node was retried with a fresh job. Such a
// job's status must not be projected onto the node (the job row itself is still
// persisted by PersistJob; only the node projection is skipped).
//
// In normal operation a job-status sync always targets the node's current job, so a
// stale job is logged as a warning: it signals a retry/supersession bug or an
// unexpected out-of-order delivery rather than expected flow.
func (c *SyncJobStatus) isStaleJob(nodeLatestJobID *string) bool {
	if nodeLatestJobID != nil && *nodeLatestJobID == c.JobID {
		return false
	}

	latest := "<nil>"
	if nodeLatestJobID != nil {
		latest = *nodeLatestJobID
	}
	c.logger.Warnf("ignoring stale %s job status sync for run %s: reporting job %s is not the node's current job %s",
		c.JobType, c.RunID, c.JobID, latest)
	return true
}

// Execute executes the sync job status command.
func (c *SyncJobStatus) Execute(ctx context.Context, input *types.ExecuteInput) error {
	// Persist the job (and any log-stream completion) within the command tx
	// before syncing the node, so the job write and node change are atomic.
	if err := c.PersistJob(ctx); err != nil {
		return err
	}

	run, err := input.RunStore.GetRunByID(ctx, c.RunID)
	if err != nil {
		return err
	}

	var changes []statemachine.NodeStatusChange

	switch c.JobType {
	case models.JobPlanType:
		if c.isStaleJob(run.Plan.LatestJobID) {
			return nil
		}
		// A node that has already reached a final status (e.g. after a force cancel)
		// must not be transitioned by a late-arriving job status. The job is still the
		// node's current job, so isStaleJob does not catch this; treat it as a no-op
		// rather than attempting an illegal transition (e.g. canceled -> finished).
		if run.Plan.Status.IsFinalStatus() {
			return nil
		}
		planStatus, ok := mapJobStatusToPlanStatus(c.NewStatus)
		if !ok {
			return nil
		}
		changes, err = statemachine.SetPlanStatus(run, planStatus)
	case models.JobApplyType:
		if run.Apply != nil && c.isStaleJob(run.Apply.LatestJobID) {
			return nil
		}
		if run.Apply != nil && run.Apply.Status.IsFinalStatus() {
			return nil
		}
		applyStatus, ok := mapJobStatusToApplyStatus(c.NewStatus)
		if !ok {
			return nil
		}
		changes, err = statemachine.SetApplyStatus(run, applyStatus)
	default:
		return errors.New("unknown job type %s for run %s", c.JobType, c.RunID)
	}

	if err != nil {
		return errors.Wrap(err, "failed to set node status")
	}

	return input.RunStore.AddRunChanges(run, changes...)
}

// mapJobStatusToPlanStatus maps a job status to a plan node status. The job's
// queued and pending states both correspond to the node's queued state (the job
// exists and is waiting for a runner), so JobPending produces no node change.
func mapJobStatusToPlanStatus(jobStatus models.JobStatus) (models.PlanStatus, bool) {
	switch jobStatus {
	case models.JobQueued:
		return models.PlanQueued, true
	case models.JobRunning:
		return models.PlanRunning, true
	case models.JobFinished:
		return models.PlanFinished, true
	case models.JobFailed:
		return models.PlanErrored, true
	case models.JobCanceled:
		return models.PlanCanceled, true
	}
	return "", false
}

// mapJobStatusToApplyStatus maps a job status to an apply node status. As with
// the plan, JobPending produces no node change.
func mapJobStatusToApplyStatus(jobStatus models.JobStatus) (models.ApplyStatus, bool) {
	switch jobStatus {
	case models.JobQueued:
		return models.ApplyQueued, true
	case models.JobRunning:
		return models.ApplyRunning, true
	case models.JobFinished:
		return models.ApplyFinished, true
	case models.JobFailed:
		return models.ApplyErrored, true
	case models.JobCanceled:
		return models.ApplyCanceled, true
	}
	return "", false
}
