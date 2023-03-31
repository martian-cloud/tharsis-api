// Package tfe package
package tfe

import (
	"fmt"

	gotfe "github.com/hashicorp/go-tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

// TharsisWorkspaceToWorkspace converts a tharsis workspace to a TFE workspace
func TharsisWorkspaceToWorkspace(workspace *models.Workspace) *Workspace {
	resp := &Workspace{
		ID:               gid.ToGlobalID(gid.WorkspaceType, workspace.Metadata.ID),
		Name:             workspace.Name,
		Operations:       true,
		AutoApply:        false,
		TerraformVersion: workspace.TerraformVersion,
		Locked:           workspace.Locked,
		Permissions: &WorkspacePermissions{
			CanQueueRun:     true,
			CanQueueApply:   true,
			CanLock:         true,
			CanUnlock:       true,
			CanQueueDestroy: true,
			CanDestroy:      true,
			CanUpdate:       true,
			CanReadSettings: true,
		},
		AllowDestroyPlan: !workspace.PreventDestroyPlan,
	}

	if workspace.CurrentStateVersionID != "" {
		resp.CurrentStateVersion = &gotfe.StateVersion{ID: gid.ToGlobalID(gid.StateVersionType, workspace.CurrentStateVersionID)}
	}

	return resp
}

// TharsisStateVersionToStateVersion converts a tharsis state version to a TFE state version
func TharsisStateVersionToStateVersion(sv *models.StateVersion, tharsisAPIURL, tfeStateVersionedPath string) *gotfe.StateVersion {
	resp := &gotfe.StateVersion{
		ID: gid.ToGlobalID(gid.StateVersionType, sv.Metadata.ID),
	}

	if sv.RunID != nil {
		resp.Run = &gotfe.Run{
			ID: gid.ToGlobalID(gid.RunType, *sv.RunID),
			Workspace: &gotfe.Workspace{
				ID: gid.ToGlobalID(gid.WorkspaceType, sv.WorkspaceID),
			},
		}
	}

	if tharsisAPIURL != "" {
		resp.DownloadURL = fmt.Sprintf("%s%s/state-versions/%s/content", tharsisAPIURL, tfeStateVersionedPath, gid.ToGlobalID(gid.StateVersionType, sv.Metadata.ID))
	}

	return resp
}

// TharsisRunToRun converts a tharsis run to a TFE run
func TharsisRunToRun(run *models.Run) *Run {
	resp := &Run{
		ID:         gid.ToGlobalID(gid.RunType, run.Metadata.ID),
		Status:     RunStatus(run.Status),
		IsDestroy:  run.IsDestroy,
		HasChanges: run.HasChanges,
		Actions: &RunActions{
			IsCancelable:      true,
			IsConfirmable:     true,
			IsForceCancelable: true,
			IsDiscardable:     true,
		},
		Permissions: &RunPermissions{
			CanApply:        true,
			CanCancel:       true,
			CanDiscard:      true,
			CanForceCancel:  true,
			CanForceExecute: true,
		},
		Workspace: &Workspace{ID: gid.ToGlobalID(gid.WorkspaceType, run.WorkspaceID)},
	}

	if run.ConfigurationVersionID != nil {
		resp.ConfigurationVersion = &gotfe.ConfigurationVersion{ID: gid.ToGlobalID(gid.ConfigurationVersionType, *run.ConfigurationVersionID)}
	}

	if run.PlanID != "" {
		resp.Plan = &gotfe.Plan{ID: gid.ToGlobalID(gid.PlanType, run.PlanID)}
	}

	if run.ApplyID != "" {
		resp.Apply = &gotfe.Apply{ID: gid.ToGlobalID(gid.ApplyType, run.ApplyID)}
	}

	return resp
}

// TharsisCVToCV converts a tharsis configuration version to a TFE configuration version
func TharsisCVToCV(cv *models.ConfigurationVersion, tharsisAPIURL, tfeWorkspacesVersionedPath string) *gotfe.ConfigurationVersion {
	cvGID := gid.ToGlobalID(gid.ConfigurationVersionType, cv.Metadata.ID)
	return &gotfe.ConfigurationVersion{
		ID:            cvGID,
		Status:        gotfe.ConfigurationStatus(cv.Status),
		Speculative:   cv.Speculative,
		AutoQueueRuns: false,
		UploadURL: fmt.Sprintf(
			"%s%s/workspaces/%s/configuration-versions/%s/upload",
			tharsisAPIURL,
			tfeWorkspacesVersionedPath,
			gid.ToGlobalID(gid.WorkspaceType, cv.WorkspaceID),
			cvGID,
		),
	}
}

// TharsisVariableToVariable converts a Tharsis variable to TFE variable.
func TharsisVariableToVariable(variable *models.Variable, workspace *models.Workspace) *Variable {
	var value string
	if val := variable.Value; val != nil {
		value = *val
	}

	return &Variable{
		Workspace: TharsisWorkspaceToWorkspace(workspace),
		Category:  getVariableCategory(variable.Category),
		ID:        gid.ToGlobalID(gid.VariableType, variable.Metadata.ID),
		HCL:       variable.Hcl,
		Key:       variable.Key,
		Value:     value,
	}
}

// TharsisErrorToTfeError translates Tharsis error to TFE equivalent or returns original.
func TharsisErrorToTfeError(err error) error {
	var tfeError error

	switch err {
	case workspace.WorkspaceLockedError:
		tfeError = errors.NewError(errors.EConflict, gotfe.ErrWorkspaceLocked.Error())
	case workspace.WorkspaceUnlockedError:
		tfeError = errors.NewError(errors.EConflict, gotfe.ErrWorkspaceNotLocked.Error())
	case workspace.WorkspaceLockedByRunError:
		tfeError = errors.NewError(errors.EConflict, gotfe.ErrWorkspaceLockedByRun.Error())
	default:
		tfeError = err
	}

	return tfeError
}

// getVariableCategory is a helper method to determine equivalent TFE variable category.
func getVariableCategory(category models.VariableCategory) CategoryType {
	if category == models.TerraformVariableCategory {
		return CategoryTerraform
	}

	return CategoryEnv
}
