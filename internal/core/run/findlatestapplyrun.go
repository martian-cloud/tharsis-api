package run

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// FindLatestApplyRunForWorkspace finds the run that produced the workspace's current
// state version. It is used by runs that re-run the configuration behind the current
// state (destroy, reconcile, assessment); callers read the returned run's variables
// (e.g. via the run variables builder) and apply any run-kind-specific validation.
func FindLatestApplyRunForWorkspace(ctx context.Context, dbClient *db.Client, workspaceID string) (*models.Run, error) {
	ws, err := dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace with ID %s", workspaceID)
	}
	if ws == nil {
		return nil, errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound))
	}
	if ws.CurrentStateVersionID == "" {
		return nil, errors.New("workspace has no current state version", errors.WithErrorCode(errors.EConflict))
	}

	stateVersion, err := dbClient.StateVersions.GetStateVersionByID(ctx, ws.CurrentStateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version with ID %s", ws.CurrentStateVersionID)
	}
	if stateVersion == nil {
		return nil, errors.New("workspace's current state version was not found", errors.WithErrorCode(errors.ENotFound))
	}
	if stateVersion.RunID == nil {
		return nil, errors.New("current state version was created manually and has no module or configuration version associated", errors.WithErrorCode(errors.EConflict))
	}

	source, err := dbClient.Runs.GetRunByID(ctx, *stateVersion.RunID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run with ID %s", *stateVersion.RunID)
	}
	if source == nil {
		return nil, errors.New("run associated with the workspace's current state version was not found", errors.WithErrorCode(errors.ENotFound))
	}

	return source, nil
}
