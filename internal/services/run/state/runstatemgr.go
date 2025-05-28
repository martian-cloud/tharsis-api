// Package state package
package state

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// EventType identifies the type of an event.
type EventType string

const (
	// RunEventType is for run events.
	RunEventType   EventType = "run"
	planEventType  EventType = "plan"
	applyEventType EventType = "apply"
	jobEventType   EventType = "job"
)

var (
	planExecutionTime  = metric.NewHistogram("plan_execution_time", "Amount of time a plan took to execute.", 1, 2, 10)
	applyExecutionTime = metric.NewHistogram("apply_execution_time", "Amount of time a plan took to apply.", 1, 2, 10)

	planFinished  = metric.NewCounter("plan_completed_count", "Amount of times a plan is completed.")
	applyFinished = metric.NewCounter("apply_completed_count", "Amount of times an apply is completed.")
	runFinished   = metric.NewCounter("run_completed_count", "Amount of times a run is completed.")
)

type eventHandlerFunc func(ctx context.Context, eventType EventType, oldModel interface{}, newModel interface{}) error

// RunStateManager is used to manage state changes for run resources
type RunStateManager struct {
	dbClient   *db.Client
	logger     logger.Logger
	handlerMap map[EventType][]eventHandlerFunc
}

// NewRunStateManager creates a new RunStateManager instance
func NewRunStateManager(dbClient *db.Client, logger logger.Logger) *RunStateManager {
	manager := &RunStateManager{
		dbClient:   dbClient,
		logger:     logger,
		handlerMap: map[EventType][]eventHandlerFunc{},
	}

	registerRunHandlers(manager)
	registerPlanHandlers(manager)
	registerApplyHandlers(manager)
	registerJobHandlers(manager)
	registerWorkspaceHandlers(manager)

	return manager
}

// RegisterHandler registers an event handler for a particular event type.
func (r *RunStateManager) RegisterHandler(eventType EventType, handler eventHandlerFunc) {
	if _, ok := r.handlerMap[eventType]; !ok {
		r.handlerMap[eventType] = []eventHandlerFunc{}
	}
	r.handlerMap[eventType] = append(r.handlerMap[eventType], handler)
}

func (r *RunStateManager) fireEvent(ctx context.Context, eventType EventType, oldModel interface{}, newModel interface{}) error {
	for _, h := range r.handlerMap[eventType] {
		// Use retry handler here for optimistic lock errors since these are internal updates and
		// we don't want to return until the handler completes successfully.
		if err := retry.Do(
			func() error {
				return h(ctx, eventType, oldModel, newModel)
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

// UpdateJob handles the state transitions for updating a job resource
func (r *RunStateManager) UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
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

// UpdateRun handles the state transitions for updating a run resource
func (r *RunStateManager) UpdateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
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

	oldRun, err := r.dbClient.Runs.GetRunByID(txContext, run.Metadata.ID)
	if err != nil {
		return nil, err
	}

	if oldRun == nil {
		return nil, errors.New("run with ID %s not found", run.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	if rErr := checkRunStatusChange(oldRun.Status, run.Status); rErr != nil {
		return nil, rErr
	}

	updatedRun, err := r.dbClient.Runs.UpdateRun(txContext, run)
	if err != nil {
		return nil, err
	}

	if err := r.fireEvent(txContext, RunEventType, oldRun, updatedRun); err != nil {
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

// UpdatePlan handles the state transitions for updating a plan resource
func (r *RunStateManager) UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
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
	oldPlan, err := r.dbClient.Plans.GetPlanByID(txContext, plan.Metadata.ID)
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

// UpdateApply handles the state transitions for updating an apply resource
func (r *RunStateManager) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
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
	oldApply, err := r.dbClient.Applies.GetApplyByID(txContext, apply.Metadata.ID)
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
	manager *RunStateManager
}

func registerRunHandlers(manager *RunStateManager) {
	handlers := &runHandlers{manager: manager}
	manager.RegisterHandler(RunEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleRunStateChangeEvent(ctx, oldModel.(*models.Run), newModel.(*models.Run))
	})
	manager.RegisterHandler(planEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handlePlanStateChangeEvent(ctx, oldModel.(*models.Plan), newModel.(*models.Plan))
	})
	manager.RegisterHandler(applyEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleApplyStateChangeEvent(ctx, oldModel.(*models.Apply), newModel.(*models.Apply))
	})
}

func (r *runHandlers) handleRunStateChangeEvent(_ context.Context, oldRun *models.Run, newRun *models.Run) error {
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
		if _, err := r.manager.UpdateRun(ctx, run); err != nil {
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

		if _, err := r.manager.UpdateRun(ctx, run); err != nil {
			return err
		}
	}

	return nil
}

/* Plan Handlers */

type planHandlers struct {
	manager *RunStateManager
}

func registerPlanHandlers(manager *RunStateManager) {
	handlers := &planHandlers{manager: manager}
	manager.RegisterHandler(jobEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleJobStateChangeEvent(ctx, oldModel.(*models.Job), newModel.(*models.Job))
	})
}

func (p *planHandlers) handleJobStateChangeEvent(ctx context.Context, oldJob *models.Job, newJob *models.Job) error {
	if newJob.Type == models.JobPlanType && oldJob.Status != newJob.Status && newJob.Status == models.JobPending {
		// Get run associated with job
		run, err := p.manager.dbClient.Runs.GetRunByID(ctx, newJob.RunID)
		if err != nil {
			return err
		}

		if run == nil {
			return errors.New("run with ID %s not found", newJob.RunID, errors.WithErrorCode(errors.ENotFound))
		}

		plan, err := p.manager.dbClient.Plans.GetPlanByID(ctx, run.PlanID)
		if err != nil {
			return err
		}

		plan.Status = models.PlanPending

		if _, err := p.manager.UpdatePlan(ctx, plan); err != nil {
			return err
		}
	}

	return nil
}

/* Apply Handlers */

type applyHandlers struct {
	manager *RunStateManager
}

func registerApplyHandlers(manager *RunStateManager) {
	handlers := &applyHandlers{manager: manager}
	manager.RegisterHandler(jobEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleJobStateChangeEvent(ctx, oldModel.(*models.Job), newModel.(*models.Job))
	})
}

func (a *applyHandlers) handleJobStateChangeEvent(ctx context.Context, oldJob *models.Job, newJob *models.Job) error {
	if newJob.Type == models.JobApplyType && oldJob.Status != newJob.Status && newJob.Status == models.JobPending {
		// Get run associated with job
		run, err := a.manager.dbClient.Runs.GetRunByID(ctx, newJob.RunID)
		if err != nil {
			return err
		}

		if run == nil {
			return errors.New("run with ID %s not found", newJob.RunID, errors.WithErrorCode(errors.ENotFound))
		}

		apply, err := a.manager.dbClient.Applies.GetApplyByID(ctx, run.ApplyID)
		if err != nil {
			return err
		}

		apply.Status = models.ApplyPending

		if _, err := a.manager.UpdateApply(ctx, apply); err != nil {
			return err
		}
	}

	return nil
}

/* Job Handlers */

type jobHandlers struct {
	manager *RunStateManager
}

func registerJobHandlers(manager *RunStateManager) {
	handlers := &jobHandlers{manager: manager}
	manager.RegisterHandler(planEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handlePlanStateChangeEvent(ctx, oldModel.(*models.Plan), newModel.(*models.Plan))
	})
	manager.RegisterHandler(applyEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleApplyStateChangeEvent(ctx, oldModel.(*models.Apply), newModel.(*models.Apply))
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

			if _, err := j.manager.UpdateJob(ctx, job); err != nil {
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

			if _, err := j.manager.UpdateJob(ctx, job); err != nil {
				return err
			}
		}
	}

	return nil
}

/* Workspace Handlers */

type workspaceHandlers struct {
	manager *RunStateManager
}

func registerWorkspaceHandlers(manager *RunStateManager) {
	handlers := &workspaceHandlers{manager: manager}
	manager.RegisterHandler(RunEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleRunStateChangeEvent(ctx, oldModel.(*models.Run), newModel.(*models.Run))
	})
	manager.RegisterHandler(jobEventType, func(ctx context.Context, _ EventType, oldModel interface{}, newModel interface{}) error {
		return handlers.handleJobStateChangeEvent(ctx, oldModel.(*models.Job), newModel.(*models.Job))
	})
}

func (w *workspaceHandlers) handleRunStateChangeEvent(ctx context.Context, oldRun *models.Run, newRun *models.Run) error {
	if !oldRun.ForceCanceled && !newRun.Speculative() && newRun.ForceCanceled {
		// Check if this run was force cancelled during the apply stage
		apply, err := w.manager.dbClient.Applies.GetApplyByID(ctx, newRun.ApplyID)
		if err != nil {
			return err
		}

		if apply != nil && apply.Status == models.ApplyCanceled {
			workspace, err := w.manager.dbClient.Workspaces.GetWorkspaceByID(ctx, newRun.WorkspaceID)
			if err != nil {
				return err
			}
			// Set workspace state to dirty since this apply was force canceled while in progress and did not exit gracefully
			workspace.DirtyState = true
			_, err = w.manager.dbClient.Workspaces.UpdateWorkspace(ctx, workspace)
			if err != nil {
				return err
			}
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
				return errors.New("runner cannot claim job %s because workspace is locked", newJob.Metadata.ID, errors.WithErrorCode(errors.EConflict))
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
func checkPlanStatusChange(oldStatus, newStatus models.PlanStatus) error {
	if oldStatus == newStatus {
		return nil
	}

	// Assume invalid until proven valid.
	transitionValid := false

	switch oldStatus {
	case models.PlanQueued:
		transitionValid = (newStatus == models.PlanPending) || (newStatus == models.PlanCanceled)
	case models.PlanPending:
		transitionValid = (newStatus == models.PlanRunning) || (newStatus == models.PlanCanceled)
	case models.PlanRunning:
		transitionValid = (newStatus == models.PlanCanceled) || (newStatus == models.PlanErrored) || (newStatus == models.PlanFinished)
	}

	// If an error was found, turn it into an error.
	if !transitionValid {
		return errors.New(
			"plan status is not allowed to transition from %s to %s", oldStatus, newStatus,
			errors.WithErrorCode(errors.EInvalid))
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
func checkApplyStatusChange(oldStatus, newStatus models.ApplyStatus) error {
	if oldStatus == newStatus {
		return nil
	}

	// Assume invalid until proven valid.
	transitionValid := false

	switch oldStatus {
	case models.ApplyCreated:
		transitionValid = (newStatus == models.ApplyQueued) || (newStatus == models.ApplyCanceled)
	case models.ApplyQueued:
		transitionValid = (newStatus == models.ApplyPending) || (newStatus == models.ApplyCanceled)
	case models.ApplyPending:
		transitionValid = (newStatus == models.ApplyRunning) || (newStatus == models.ApplyCanceled)
	case models.ApplyRunning:
		transitionValid = (newStatus == models.ApplyCanceled) || (newStatus == models.ApplyErrored) || (newStatus == models.ApplyFinished)
	}

	// If an error was found, turn it into an error.
	if !transitionValid {
		return errors.New(
			"apply status is not allowed to transition from %s to %s", oldStatus, newStatus,
			errors.WithErrorCode(errors.EInvalid))
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
func checkRunStatusChange(oldStatus, newStatus models.RunStatus) error {
	if oldStatus == newStatus {
		return nil
	}

	transitionValid := false

	switch oldStatus {
	case models.RunPlanQueued:
		transitionValid = (newStatus == models.RunCanceled) || (newStatus == models.RunErrored) || (newStatus == models.RunPlanning)
	case models.RunApplyQueued:
		transitionValid = (newStatus == models.RunCanceled) || (newStatus == models.RunErrored) || (newStatus == models.RunApplying)
	case models.RunPlanning:
		transitionValid = (newStatus == models.RunCanceled) || (newStatus == models.RunErrored) || (newStatus == models.RunPlanned) || (newStatus == models.RunPlannedAndFinished)
	case models.RunApplying:
		transitionValid = (newStatus == models.RunCanceled) || (newStatus == models.RunErrored) || (newStatus == models.RunApplied)
	case models.RunPlanned:
		transitionValid = (newStatus == models.RunCanceled) || (newStatus == models.RunApplyQueued)
	}

	if !transitionValid {
		return errors.New(
			"run status is not allowed to transition from %s to %s", oldStatus, newStatus,
			errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
