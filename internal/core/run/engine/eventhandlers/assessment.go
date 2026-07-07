package eventhandlers

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var workspaceDriftCount = metric.NewCounter("workspace_drift_count", "Total count of workspaces that have drifted.")

// AssessmentRunHandler updates a workspace's drift assessment when an assessment
// run completes, and invalidates a stale assessment when a regular run updates the
// workspace's state.
type AssessmentRunHandler struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewAssessmentRunHandler creates a new AssessmentRunHandler.
func NewAssessmentRunHandler(logger logger.Logger, dbClient *db.Client) *AssessmentRunHandler {
	return &AssessmentRunHandler{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler.
func (h *AssessmentRunHandler) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	for _, change := range changes {
		if err := h.handleRun(ctx, change.Run); err != nil {
			return err
		}
	}
	return nil
}

func (h *AssessmentRunHandler) handleRun(ctx context.Context, run *models.Run) error {
	// Only act once a run has reached a terminal state.
	if !run.IsComplete() {
		return nil
	}

	// We only care about assessment runs (which record drift) and non-speculative
	// runs (which may invalidate a previously recorded drift assessment).
	if !run.IsAssessmentRun && run.Speculative() {
		return nil
	}

	assessment, err := h.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, run.WorkspaceID)
	if err != nil {
		return err
	}

	if !run.IsAssessmentRun {
		// A regular run completed: if it updated the workspace's current state
		// version, any recorded drift no longer reflects reality.
		return h.invalidateStaleAssessment(ctx, run, assessment)
	}

	if assessment == nil {
		// This should never happen unless the assessment record was manually deleted
		// after the run was started.
		h.logger.WithContextFields(ctx).Errorf("assessment record not found for workspace %s", run.WorkspaceID)
		return nil
	}

	now := time.Now().UTC()
	assessment.CompletedAtTimestamp = &now
	// Link the run that completed this assessment, regardless of outcome, so the
	// recorded run reflects the latest attempt (and its status is visible).
	assessment.RunID = &run.Metadata.ID

	if run.Status == models.RunPlannedAndFinished {
		hasDrift := run.Plan.Summary.ResourceDrift > 0

		if hasDrift && !assessment.HasDrift {
			workspaceDriftCount.Inc()
		}

		// Notification is only required when the workspace newly drifted (it has
		// drift now but did not before).
		assessment.RequiresNotification = hasDrift && !assessment.HasDrift
		assessment.HasDrift = hasDrift
	} else {
		// The run failed/canceled/discarded, so it produced no fresh drift verdict.
		// Clear the previous verdict rather than presenting it as this run's result.
		assessment.RequiresNotification = false
		assessment.HasDrift = false
	}

	if _, err := h.dbClient.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, assessment); err != nil {
		return err
	}
	return nil
}

// invalidateStaleAssessment deletes the workspace's drift assessment when the given
// run produced the workspace's current state version, since the recorded drift is no
// longer valid.
func (h *AssessmentRunHandler) invalidateStaleAssessment(ctx context.Context, run *models.Run, assessment *models.WorkspaceAssessment) error {
	if assessment == nil {
		return nil
	}

	ws, err := h.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return err
	}
	if ws == nil || ws.CurrentStateVersionID == "" {
		return nil
	}

	stateVersion, err := h.dbClient.StateVersions.GetStateVersionByID(ctx, ws.CurrentStateVersionID)
	if err != nil {
		return err
	}
	if stateVersion == nil {
		return nil
	}

	if stateVersion.RunID != nil && *stateVersion.RunID == run.Metadata.ID {
		return h.dbClient.WorkspaceAssessments.DeleteWorkspaceAssessment(ctx, assessment)
	}
	return nil
}
