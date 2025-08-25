// Package eventhandlers provides handlers for run state change events.
package eventhandlers

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var workspaceDriftCount = metric.NewCounter("workspace_drift_count", "Total count of workspaces that have drifted.")

// AssessmentRunHandler manages errored runs.
type AssessmentRunHandler struct {
	logger          logger.Logger
	dbClient        *db.Client
	runStateManager state.RunStateManager
}

// NewAssessmentRunHandler returns an instance of AssessmentRunHandler.
func NewAssessmentRunHandler(
	logger logger.Logger,
	dbClient *db.Client,
	runStateManager state.RunStateManager,
) *AssessmentRunHandler {
	return &AssessmentRunHandler{
		logger:          logger,
		dbClient:        dbClient,
		runStateManager: runStateManager,
	}
}

// RegisterHandlers registers the handlers with the run state manager.
func (t *AssessmentRunHandler) RegisterHandlers() {
	t.runStateManager.RegisterHandler(state.RunEventType, t.handleRunEvent)
}

func (t *AssessmentRunHandler) handleRunEvent(ctx context.Context, _ state.EventType, _ interface{}, newModel interface{}) error {
	run := newModel.(*models.Run)

	// Skip this run if it's not complete
	if !run.IsComplete() {
		return nil
	}

	// We're only interested in assessment runs or non-speculative runs
	if !run.IsAssessmentRun && run.Speculative() {
		return nil
	}

	// Check for assessment record
	assessment, err := t.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "assessment run handler failed to query for assessment associated with workspace %q", run.WorkspaceID)
	}

	if !run.IsAssessmentRun {
		// Check if there is an existing assessment record
		if assessment == nil {
			return nil
		}

		ws, err := t.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
		if err != nil {
			return errors.Wrap(err, "assessment run handler failed to query for workspace %q", run.WorkspaceID)
		}

		if ws == nil {
			return nil
		}

		// There is nothing to do if workspace doesn't have a current state version
		if ws.CurrentStateVersionID == "" {
			return nil
		}

		stateVersion, err := t.dbClient.StateVersions.GetStateVersionByID(ctx, ws.CurrentStateVersionID)
		if err != nil {
			return errors.Wrap(err, "assessment run handler failed to query for state version %q", ws.CurrentStateVersionID)
		}

		if stateVersion == nil {
			return nil
		}

		if stateVersion.RunID != nil && *stateVersion.RunID == run.Metadata.ID {
			// This run updated the workspace's state version so we need to clear the assessment since it's not longer valid
			if err = t.dbClient.WorkspaceAssessments.DeleteWorkspaceAssessment(ctx, assessment); err != nil {
				return err
			}
		}

		return nil
	}

	if assessment == nil {
		// This should never happen unless the assessment record is manually deleted from the database after the run was started
		t.logger.WithContextFields(ctx).Errorf("assessment record not found for workspace %q", run.WorkspaceID)
		return nil
	}

	// Only update drift status if the run was completed successfully
	if run.Status == models.RunPlannedAndFinished {
		plan, err := t.dbClient.Plans.GetPlanByID(ctx, run.PlanID)
		if err != nil {
			return errors.Wrap(err, "assessment run handler failed to query for plan %q", run.PlanID)
		}

		if plan == nil {
			return nil
		}

		// Check if there are any drifted resources
		hasDrift := plan.Summary.ResourceDrift > 0

		if hasDrift && !assessment.HasDrift {
			workspaceDriftCount.Inc()
		}

		// Notification is only required if the workspace now has drift but did not previously have drift
		assessment.RequiresNotification = hasDrift && !assessment.HasDrift

		assessment.HasDrift = hasDrift
		assessment.RunID = &run.Metadata.ID
	}

	assessment.CompletedAtTimestamp = ptr.Time(time.Now().UTC())

	// Update assessment
	if _, err := t.dbClient.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, assessment); err != nil {
		return errors.Wrap(err, "assessment run handler failed to update assessment associated with workspace %q", run.WorkspaceID)
	}

	return nil
}
