// Package tfe package
package tfe

import (
	"fmt"

	gotfe "github.com/hashicorp/go-tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// TharsisWorkspaceToWorkspace converts a tharsis workspace to a TFE workspace
func TharsisWorkspaceToWorkspace(workspace *models.Workspace) *Workspace {
	resp := &Workspace{
		ID:               workspace.GetGlobalID(),
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
		resp.CurrentStateVersion = &gotfe.StateVersion{ID: gid.ToGlobalID(types.StateVersionModelType, workspace.CurrentStateVersionID)}
	}

	return resp
}

// TharsisStateVersionToStateVersion converts a tharsis state version to a TFE state version
func TharsisStateVersionToStateVersion(sv *models.StateVersion, tharsisAPIURL, tfeStateVersionedPath string) *gotfe.StateVersion {
	resp := &gotfe.StateVersion{
		ID: sv.GetGlobalID(),
	}

	if sv.RunID != nil {
		resp.Run = &gotfe.Run{
			ID: gid.ToGlobalID(types.RunModelType, *sv.RunID),
			Workspace: &gotfe.Workspace{
				ID: gid.ToGlobalID(types.WorkspaceModelType, sv.WorkspaceID),
			},
		}
	}

	if tharsisAPIURL != "" {
		resp.DownloadURL = fmt.Sprintf("%s%s/state-versions/%s/content", tharsisAPIURL, tfeStateVersionedPath, sv.GetGlobalID())
	}

	return resp
}

// TharsisRunToRun converts a tharsis run to a TFE run
func TharsisRunToRun(run *models.Run) *Run {
	resp := &Run{
		ID:         run.GetGlobalID(),
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
		Workspace: &Workspace{ID: gid.ToGlobalID(types.WorkspaceModelType, run.WorkspaceID)},
	}

	if run.ConfigurationVersionID != nil {
		resp.ConfigurationVersion = &gotfe.ConfigurationVersion{ID: gid.ToGlobalID(types.ConfigurationVersionModelType, *run.ConfigurationVersionID)}
	}

	if run.PlanID != "" {
		resp.Plan = &gotfe.Plan{ID: gid.ToGlobalID(types.PlanModelType, run.PlanID)}
	}

	if run.ApplyID != "" {
		resp.Apply = &gotfe.Apply{ID: gid.ToGlobalID(types.ApplyModelType, run.ApplyID)}
	}

	return resp
}

// TharsisCVToCV converts a tharsis configuration version to a TFE configuration version
func TharsisCVToCV(cv *models.ConfigurationVersion, uploadURL string) *gotfe.ConfigurationVersion {
	return &gotfe.ConfigurationVersion{
		ID:            cv.GetGlobalID(),
		Status:        gotfe.ConfigurationStatus(cv.Status),
		Speculative:   cv.Speculative,
		AutoQueueRuns: false,
		UploadURL:     uploadURL,
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
		ID:        variable.GetGlobalID(),
		HCL:       variable.Hcl,
		Key:       variable.Key,
		Value:     value,
	}
}

// TharsisErrorToTfeError translates Tharsis error to TFE equivalent or returns original.
func TharsisErrorToTfeError(err error) error {
	var tfeError error

	switch err {
	case workspace.ErrWorkspaceLocked:
		tfeError = errors.New(gotfe.ErrWorkspaceLocked.Error(), errors.WithErrorCode(errors.EConflict))
	case workspace.ErrWorkspaceUnlocked:
		tfeError = errors.New(gotfe.ErrWorkspaceNotLocked.Error(), errors.WithErrorCode(errors.EConflict))
	case workspace.ErrWorkspaceLockedByRun:
		tfeError = errors.New(gotfe.ErrWorkspaceLockedByRun.Error(), errors.WithErrorCode(errors.EConflict))
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
