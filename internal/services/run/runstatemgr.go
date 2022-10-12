package run

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

type eventType string

const (
	runEventType   eventType = "run"
	planEventType  eventType = "plan"
	applyEventType eventType = "apply"
	jobEventType   eventType = "job"
)

type eventHandlerFunc func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error

// runStateManager is used to manage state changes for run resources
type runStateManager struct {
	dbClient   *db.Client
	logger     logger.Logger
	handlerMap map[eventType][]eventHandlerFunc
}

func newRunStateManager(dbClient *db.Client, logger logger.Logger) *runStateManager {
	manager := &runStateManager{
		dbClient:   dbClient,
		logger:     logger,
		handlerMap: map[eventType][]eventHandlerFunc{},
	}

	registerRunHandlers(manager)
	registerJobHandlers(manager)
	registerWorkspaceHandlers(manager)

	return manager
}

func (r *runStateManager) registerHandler(eventType eventType, handler eventHandlerFunc) {
	if _, ok := r.handlerMap[eventType]; !ok {
		r.handlerMap[eventType] = []eventHandlerFunc{}
	}
	r.handlerMap[eventType] = append(r.handlerMap[eventType], handler)
}

func (r *runStateManager) fireEvent(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
	for _, h := range r.handlerMap[eventType] {
		// Use retry handler here for optimistic lock errors since these are internal updates and
		// we don't want to return until the handler completes successfully.
		if err := retry.Do(
			func() error {
				return h(ctx, eventType, old, new)
			},
			retry.Attempts(100),
			retry.DelayType(retry.FixedDelay),
			retry.Delay(10*time.Millisecond),
			retry.RetryIf(func(err error) bool {
				// Only retry on optimistic lock errors
				return errors.ErrorCode(err) == errors.EOptimisticLock
			}),
			retry.OnRetry(func(n uint, err error) {
				r.logger.Infof("Retrying event handler for event type %s: attempt= %d error=%s", eventType, n+1, err)
			}),
			retry.LastErrorOnly(true),
			retry.Context(ctx),
		); err != nil {
			return err
		}
	}
	return nil
}

// updateJob handles the state transitions for updating a job resource
func (r *runStateManager) updateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap all DB work in a transaction.
	txContext, err := r.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := r.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			r.logger.Errorf("failed to rollback tx for updateJob: %v", txErr)
		}
	}()

	oldJob, err := r.dbClient.Jobs.GetJobByID(txContext, job.Metadata.ID)
	if err != nil {
		return nil, err
	}

	updatedJob, err := r.dbClient.Jobs.UpdateJob(txContext, job)
	if err != nil {
		return nil, err
	}

	if err := r.fireEvent(txContext, jobEventType, oldJob, updatedJob); err != nil {
		return nil, err
	}

	if err := r.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	r.logger.Infow("Updated a job.",
		"caller", caller.GetSubject(),
		"workspaceID", job.WorkspaceID,
		"oldJobStatus", oldJob.Status,
		"newJobStatus", updatedJob.Status,
		"jobID", updatedJob.Metadata.ID,
	)

	return updatedJob, nil
}

// updateRun handles the state transitions for updating a run resource
func (r *runStateManager) updateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap all DB work in a transaction.
	txContext, err := r.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := r.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			r.logger.Errorf("failed to rollback tx for updateRun: %v", txErr)
		}
	}()

	oldRun, err := r.dbClient.Runs.GetRun(txContext, run.Metadata.ID)
	if err != nil {
		return nil, err
	}

	if oldRun == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Run with ID %s not found", run.Metadata.ID))
	}

	if rErr := checkRunStatusChange(oldRun.Status, run.Status); rErr != nil {
		return nil, rErr
	}

	updatedRun, err := r.dbClient.Runs.UpdateRun(txContext, run)
	if err != nil {
		return nil, err
	}

	if err := r.fireEvent(txContext, runEventType, oldRun, updatedRun); err != nil {
		return nil, err
	}

	if err := r.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	r.logger.Infow("Updated a run.",
		"caller", caller.GetSubject(),
		"workspaceID", run.WorkspaceID,
		"oldRunStatus", oldRun.Status,
		"newRunStatus", updatedRun.Status,
		"runID", updatedRun.Metadata.ID,
	)

	return updatedRun, nil
}

// updatePlan handles the state transitions for updating a plan resource
func (r *runStateManager) updatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap all DB work in a transaction.
	txContext, err := r.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := r.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			r.logger.Errorf("failed to rollback tx for updatePlan: %v", txErr)
		}
	}()

	// Get the old plan object from the DB.
	oldPlan, err := r.dbClient.Plans.GetPlan(txContext, plan.Metadata.ID)
	if err != nil {
		return nil, err
	}

	// Check whether the state transition is allowed.
	err = checkPlanStatusChange(oldPlan.Status, plan.Status)
	if err != nil {
		return nil, err
	}

	updatedPlan, err := r.dbClient.Plans.UpdatePlan(txContext, plan)
	if err != nil {
		return nil, err
	}

	if err := r.fireEvent(txContext, planEventType, oldPlan, updatedPlan); err != nil {
		return nil, err
	}

	if err := r.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	r.logger.Infow("Updated a plan.",
		"caller", caller.GetSubject(),
		"workspaceID", plan.WorkspaceID,
		"oldPlanStatus", oldPlan.Status,
		"newPlanStatus", updatedPlan.Status,
		"planID", updatedPlan.Metadata.ID,
	)

	return updatedPlan, nil
}

// updateApply handles the state transitions for updating an apply resource
func (r *runStateManager) updateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap all DB work in a transaction.
	txContext, err := r.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := r.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			r.logger.Errorf("failed to rollback tx for updateApply: %v", txErr)
		}
	}()

	// Get the old apply object from the DB.
	oldApply, err := r.dbClient.Applies.GetApply(txContext, apply.Metadata.ID)
	if err != nil {
		return nil, err
	}

	// Check whether the state transition is allowed.
	err = checkApplyStatusChange(oldApply.Status, apply.Status)
	if err != nil {
		return nil, err
	}

	updatedApply, err := r.dbClient.Applies.UpdateApply(txContext, apply)
	if err != nil {
		return nil, err
	}

	if err := r.fireEvent(txContext, applyEventType, oldApply, updatedApply); err != nil {
		return nil, err
	}

	if err := r.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	r.logger.Infow("Updated an apply.",
		"caller", caller.GetSubject(),
		"workspaceID", apply.WorkspaceID,
		"oldApplyStatus", oldApply.Status,
		"newApplyStatus", updatedApply.Status,
		"applyID", updatedApply.Metadata.ID,
	)

	return updatedApply, nil
}

type runHandlers struct {
	manager *runStateManager
}

func registerRunHandlers(manager *runStateManager) {
	handlers := &runHandlers{manager: manager}
	manager.registerHandler(runEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handleRunStateChangeEvent(ctx, old.(*models.Run), new.(*models.Run))
	})
	manager.registerHandler(planEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handlePlanStateChangeEvent(ctx, old.(*models.Plan), new.(*models.Plan))
	})
	manager.registerHandler(applyEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handleApplyStateChangeEvent(ctx, old.(*models.Apply), new.(*models.Apply))
	})
}

func (r *runHandlers) handleRunStateChangeEvent(ctx context.Context, oldRun *models.Run, newRun *models.Run) error {
	if oldRun.Status != newRun.Status {
		switch newRun.Status {
		case models.RunPlannedAndFinished, models.RunApplied, models.RunCanceled, models.RunErrored:
			runFinished.Inc()
		}
	}
	return nil
}

func (r *runHandlers) handlePlanStateChangeEvent(ctx context.Context, oldPlan *models.Plan, newPlan *models.Plan) error {
	if oldPlan.Status != newPlan.Status {
		run, err := r.manager.dbClient.Runs.GetRunByPlanID(ctx, newPlan.Metadata.ID)
		if err != nil {
			return err
		}

		switch newPlan.Status {
		case models.PlanCanceled:
			run.Status = models.RunCanceled
			planFinished.Inc()
		case models.PlanErrored:
			run.Status = models.RunErrored
			planFinished.Inc()
		case models.PlanRunning:
			run.Status = models.RunPlanning
		case models.PlanFinished:
			// This is a speculative plan if the apply resource is not set
			if run.ApplyID != "" && newPlan.HasChanges {
				run.Status = models.RunPlanned
			} else {
				run.Status = models.RunPlannedAndFinished
			}

			planFinished.Inc()
		}

		run.HasChanges = newPlan.HasChanges
		if _, err := r.manager.updateRun(ctx, run); err != nil {
			return err
		}
	}

	return nil
}

func (r *runHandlers) handleApplyStateChangeEvent(ctx context.Context, oldApply *models.Apply, newApply *models.Apply) error {
	if oldApply.Status != newApply.Status {
		run, err := r.manager.dbClient.Runs.GetRunByApplyID(ctx, newApply.Metadata.ID)
		if err != nil {
			return err
		}

		switch newApply.Status {
		case models.ApplyQueued:
			run.Status = models.RunApplyQueued
		case models.ApplyCanceled:
			run.Status = models.RunCanceled
			applyFinished.Inc()
		case models.ApplyErrored:
			run.Status = models.RunErrored
			applyFinished.Inc()
		case models.ApplyRunning:
			run.Status = models.RunApplying
		case models.ApplyFinished:
			run.Status = models.RunApplied
			applyFinished.Inc()
		}

		if _, err := r.manager.updateRun(ctx, run); err != nil {
			return err
		}
	}

	return nil
}

/* Job Handlers */

type jobHandlers struct {
	manager *runStateManager
}

func registerJobHandlers(manager *runStateManager) {
	handlers := &jobHandlers{manager: manager}
	manager.registerHandler(planEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handlePlanStateChangeEvent(ctx, old.(*models.Plan), new.(*models.Plan))
	})
	manager.registerHandler(applyEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handleApplyStateChangeEvent(ctx, old.(*models.Apply), new.(*models.Apply))
	})
}

func (j *jobHandlers) handlePlanStateChangeEvent(ctx context.Context, oldPlan *models.Plan, newPlan *models.Plan) error {
	if oldPlan.Status != newPlan.Status {
		run, err := j.manager.dbClient.Runs.GetRunByPlanID(ctx, newPlan.Metadata.ID)
		if err != nil {
			return err
		}

		job, err := j.manager.dbClient.Jobs.GetLatestJobByType(ctx, run.Metadata.ID, models.JobPlanType)
		if err != nil {
			return err
		}

		if job != nil {
			now := time.Now()

			switch newPlan.Status {
			case models.PlanCanceled:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			case models.PlanErrored:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			case models.PlanRunning:
				job.Timestamps.RunningTimestamp = &now
				job.Status = models.JobRunning
			case models.PlanFinished:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			}

			// Difference between running and finished timestamp equates to execution time.
			if job.Timestamps.RunningTimestamp != nil && job.Timestamps.FinishedTimestamp != nil {
				difference := job.Timestamps.FinishedTimestamp.Sub(*job.Timestamps.RunningTimestamp)
				planExecutionTime.Observe(float64(difference.Minutes()))
			}

			if _, err := j.manager.updateJob(ctx, job); err != nil {
				return err
			}
		}
	}

	return nil
}

func (j *jobHandlers) handleApplyStateChangeEvent(ctx context.Context, oldApply *models.Apply, newApply *models.Apply) error {
	if oldApply.Status != newApply.Status {
		run, err := j.manager.dbClient.Runs.GetRunByApplyID(ctx, newApply.Metadata.ID)
		if err != nil {
			return err
		}

		job, err := j.manager.dbClient.Jobs.GetLatestJobByType(ctx, run.Metadata.ID, models.JobApplyType)
		if err != nil {
			return err
		}

		if job != nil {
			now := time.Now()

			switch newApply.Status {
			case models.ApplyCanceled:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			case models.ApplyErrored:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			case models.ApplyRunning:
				job.Timestamps.RunningTimestamp = &now
				job.Status = models.JobRunning
			case models.ApplyFinished:
				job.Timestamps.FinishedTimestamp = &now
				job.Status = models.JobFinished
			}

			// Difference between running and finished timestamp equates to execution time.
			if job.Timestamps.RunningTimestamp != nil && job.Timestamps.FinishedTimestamp != nil {
				difference := job.Timestamps.FinishedTimestamp.Sub(*job.Timestamps.RunningTimestamp)
				applyExecutionTime.Observe(float64(difference.Minutes()))
			}

			if _, err := j.manager.updateJob(ctx, job); err != nil {
				return err
			}
		}
	}

	return nil
}

/* Workspace Handlers */

type workspaceHandlers struct {
	manager *runStateManager
}

func registerWorkspaceHandlers(manager *runStateManager) {
	handlers := &workspaceHandlers{manager: manager}
	manager.registerHandler(runEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handleRunStateChangeEvent(ctx, old.(*models.Run), new.(*models.Run))
	})
	manager.registerHandler(jobEventType, func(ctx context.Context, eventType eventType, old interface{}, new interface{}) error {
		return handlers.handleJobStateChangeEvent(ctx, old.(*models.Job), new.(*models.Job))
	})
}

func (w *workspaceHandlers) handleRunStateChangeEvent(ctx context.Context, oldRun *models.Run, newRun *models.Run) error {
	if !oldRun.ForceCanceled && newRun.ForceCanceled {
		workspace, err := w.manager.dbClient.Workspaces.GetWorkspaceByID(ctx, newRun.WorkspaceID)
		if err != nil {
			return err
		}
		workspace.DirtyState = true
		_, err = w.manager.dbClient.Workspaces.UpdateWorkspace(ctx, workspace)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *workspaceHandlers) handleJobStateChangeEvent(ctx context.Context, oldJob *models.Job, newJob *models.Job) error {
	if oldJob.Status != newJob.Status {
		// For tracking current running job in workspace.
		ws, err := w.manager.dbClient.Workspaces.GetWorkspaceByID(ctx, newJob.WorkspaceID)
		if err != nil {
			return err
		}

		if newJob.Status == models.JobPending {
			if ws.Locked {
				return errors.NewError(errors.EConflict, fmt.Sprintf("Runner cannot claim job %s because workspace is locked", newJob.Metadata.ID))
			}
			ws.Locked = true
			ws.CurrentJobID = newJob.Metadata.ID
			if _, err = w.manager.dbClient.Workspaces.UpdateWorkspace(ctx, ws); err != nil {
				return err
			}
		}

		if newJob.Status == models.JobFinished && ws.CurrentJobID == newJob.Metadata.ID {
			ws.Locked = false
			ws.CurrentJobID = ""
			if _, err = w.manager.dbClient.Workspaces.UpdateWorkspace(ctx, ws); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkPlanStatusChange returns an error if the specified plan status change is invalid.
// This function is similar to checkApplyStatusChange below.
func checkPlanStatusChange(old, new models.PlanStatus) error {
	if old == new {
		return nil
	}

	// Assume invalid until proven valid.
	transitionValid := false

	switch old {
	case models.PlanQueued:
		transitionValid = (new == models.PlanPending) || (new == models.PlanCanceled)
	case models.PlanPending:
		transitionValid = (new == models.PlanRunning) || (new == models.PlanCanceled)
	case models.PlanRunning:
		transitionValid = (new == models.PlanCanceled) || (new == models.PlanErrored) || (new == models.PlanFinished)
	}

	// If an error was found, turn it into an error.
	if !transitionValid {
		return errors.NewError(
			errors.EInvalid,
			fmt.Sprintf("plan status is not allowed to transition from %s to %s", old, new),
		)
	}

	return nil
}

// For both checkPlanStatusChange and checkApplyStatusChange, the (only) valid state transitions are:
//
// created -> queued, canceled (only for apply)
// queued -> pending, canceled
// pending -> running, canceled
// running -> canceled, errored, finished
// canceled, errored, finished -> no valid states

// checkApplyStatusChange returns an error if the specified Apply status change is invalid.
// This function is similar to checkPlanStatusChange above.
func checkApplyStatusChange(old, new models.ApplyStatus) error {
	if old == new {
		return nil
	}

	// Assume invalid until proven valid.
	transitionValid := false

	switch old {
	case models.ApplyCreated:
		transitionValid = (new == models.ApplyQueued) || (new == models.ApplyCanceled)
	case models.ApplyQueued:
		transitionValid = (new == models.ApplyPending) || (new == models.ApplyCanceled)
	case models.ApplyPending:
		transitionValid = (new == models.ApplyRunning) || (new == models.ApplyCanceled)
	case models.ApplyRunning:
		transitionValid = (new == models.ApplyCanceled) || (new == models.ApplyErrored) || (new == models.ApplyFinished)
	}

	// If an error was found, turn it into an error.
	if !transitionValid {
		return errors.NewError(
			errors.EInvalid,
			fmt.Sprintf("apply status is not allowed to transition from %s to %s", old, new),
		)
	}

	return nil
}

// Valid transitions for a run:
// planQueued -> canceled, errored, planning
// applyQueued -> canceled, errored, applying
// planning -> canceled, errored, planned, plannedAndFinished
// applying -> canceled, errored, applied
// planned -> canceled, applyQueued

// checkRunStatusChange returns an error for an invalid run transition.
func checkRunStatusChange(old, new models.RunStatus) error {
	if old == new {
		return nil
	}

	transitionValid := false

	switch old {
	case models.RunPlanQueued:
		transitionValid = (new == models.RunCanceled) || (new == models.RunErrored) || (new == models.RunPlanning)
	case models.RunApplyQueued:
		transitionValid = (new == models.RunCanceled) || (new == models.RunErrored) || (new == models.RunApplying)
	case models.RunPlanning:
		transitionValid = (new == models.RunCanceled) || (new == models.RunErrored) || (new == models.RunPlanned) || (new == models.RunPlannedAndFinished)
	case models.RunApplying:
		transitionValid = (new == models.RunCanceled) || (new == models.RunErrored) || (new == models.RunApplied)
	case models.RunPlanned:
		transitionValid = (new == models.RunCanceled) || (new == models.RunApplyQueued)
	}

	if !transitionValid {
		return errors.NewError(
			errors.EInvalid,
			fmt.Sprintf("run status is not allowed to transition from %s to %s", old, new),
		)
	}

	return nil
}
