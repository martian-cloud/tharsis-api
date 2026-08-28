// Package tfe package
package tfe

import (
	"fmt"
	"strings"

	gotfe "github.com/hashicorp/go-tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
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

		// The rendering can be uploaded at any point after the state version exists — the job executor
		// sends it once the apply finishes — so the URL is always advertised. UploadURL has no
		// counterpart because raw state only arrives with the state version itself.
		resp.JSONUploadURL = fmt.Sprintf("%s%s/state-versions/%s/content.json",
			tharsisAPIURL, tfeStateVersionedPath, sv.GetGlobalID())

		// Only advertise the rendering for download when one is stored. A state version created by an
		// older job executor has none, and HCP Terraform likewise leaves the field empty until it has
		// one, so a TFE client already has to treat it as optional rather than assume the URL resolves.
		if sv.JSONObjectStoreKey != nil {
			resp.JSONDownloadURL = fmt.Sprintf("%s%s/state-versions/%s/content.json",
				tharsisAPIURL, tfeStateVersionedPath, sv.GetGlobalID())
		}
	}

	return resp
}

// TharsisRunToRun converts a tharsis run to a TFE run
func TharsisRunToRun(run *models.Run) *Run {
	resp := &Run{
		ID:         run.GetGlobalID(),
		Status:     tharsisRunStatusToTFE(run.Status),
		IsDestroy:  run.IsDestroy,
		HasChanges: run.HasChanges(),
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

	resp.Plan = &gotfe.Plan{ID: run.Plan.GetGlobalID()}

	if applyNode := run.Apply; applyNode != nil {
		resp.Apply = &gotfe.Apply{ID: applyNode.GetGlobalID()}
	}

	// Surface post-plan policy checks through the legacy Sentinel-style policy-checks relation (kept
	// for override + logs). Only the post-plan stage is included: the classic TFE policy-check model
	// is post-plan only and has no notion of pre-plan policies, so pre-plan checks are surfaced solely
	// through the task-stages relation below.
	if postStage := run.TaskStageByStageName(models.RunTaskStageNamePostPlan); postStage != nil {
		for _, check := range postStage.PolicyChecks {
			resp.PolicyChecks = append(resp.PolicyChecks, TharsisPolicyCheckToPolicyCheck(check))
		}
	}
	// Surface each persisted task stage through the task-stages relation (one per run stage, in
	// canonical order), each carrying an OPA policy evaluation per check it owns.
	for _, stage := range run.TaskStages {
		resp.TaskStages = append(resp.TaskStages, tharsisTaskStageToTaskStage(stage))
	}

	return resp
}

// tharsisRunStatusToTFE maps a Tharsis run status onto the TFE run status the API surfaces. Most
// statuses share a spelling and map 1:1, but a raw string cast is never safe: Tharsis has statuses that
// are not part of the TFE run-status vocabulary, and the Terraform CLI misbehaves on a value it does not
// recognise. Every branch here returns a constant from allRunStatuses, and a test asserts both that
// every Tharsis status is mapped and that every result is in that set.
//
// Two things make this more than a rename table.
//
// Waiting for the workspace slot. Tharsis names one *_queuing status per workspace-gated node, where TFE
// has just queuing (before the plan) and queuing_apply (before the apply). So the four collapse onto
// those two by which side of the plan they sit on: pre_plan_queuing and plan_queuing are the run waiting
// for capacity before it plans, pre_apply_queuing and apply_queuing before it applies. The *_queued
// statuses, meaning admitted and waiting for a runner, keep their 1:1 TFE counterparts.
//
// Override gates. The CLI's override confirmation (confirm() in the cloud backend) only treats a run as
// still awaiting an override when its status is policy_override or post_plan_awaiting_decision —
// otherwise it concludes the run already moved past the decision and refuses to wait for the override
// input ("The run needs to be manually overridden or discarded."). go-tfe has no
// pre_plan_awaiting_decision or pre_apply_awaiting_decision, so reporting the nearest literal match
// (pre_plan_running / pre_apply_running) would break the interactive override even though the task stage
// itself correctly reports awaiting_override. Both are therefore reported as policy_override — the
// generic "run paused for a policy override" status the CLI accepts — which makes them behave like the
// post-plan override. A post-plan block maps straight through to its matching TFE status.
func tharsisRunStatusToTFE(status models.RunStatus) RunStatus {
	switch status {
	case models.RunPending:
		return RunPending

	// The pre-plan policy stage.
	case models.RunPrePlanQueuing:
		return RunQueuing
	case models.RunPrePlanRunning:
		return RunPrePlanRunning
	case models.RunPrePlanAwaitingDecision:
		return RunPolicyOverride
	case models.RunPrePlanCompleted:
		return RunPrePlanCompleted

	// The plan: waiting for the workspace, then waiting for a runner.
	case models.RunPlanQueuing:
		return RunQueuing
	case models.RunPlanQueued:
		return RunPlanQueued
	case models.RunPlanning:
		return RunPlanning

	// The post-plan policy stage.
	case models.RunPostPlanRunning:
		return RunPostPlanRunning
	case models.RunPostPlanAwaitingDecision:
		return RunPostPlanAwaitingDecision
	case models.RunPostPlanCompleted:
		return RunPostPlanCompleted
	case models.RunPlanned:
		return RunPlanned
	case models.RunPlannedAndFinished:
		return RunPlannedAndFinished

	// The pre-apply policy stage.
	case models.RunPreApplyQueuing:
		return RunQueuingApply
	case models.RunPreApplyRunning:
		return RunPreApplyRunning
	case models.RunPreApplyAwaitingDecision:
		return RunPolicyOverride
	case models.RunPreApplyCompleted:
		return RunPreApplyCompleted

	// The apply, mirroring the plan.
	case models.RunApplyQueuing:
		return RunQueuingApply
	case models.RunApplyQueued:
		return RunApplyQueued
	case models.RunApplying:
		return RunApplying
	case models.RunPostApplyRunning:
		return RunPostApplyRunning
	case models.RunPostApplyCompleted:
		return RunPostApplyCompleted
	case models.RunApplied:
		return RunApplied
	case models.RunCanceled:
		return RunCanceled
	case models.RunDiscarded:
		return RunDiscarded
	case models.RunErrored:
		return RunErrored
	default:
		// Unreachable: the exhaustiveness test fails if a Tharsis status has no case above. Report
		// pending rather than casting the raw string, so an unmapped status can never reach the CLI as a
		// value outside its vocabulary.
		return RunPending
	}
}

// tharsisTaskStageToTaskStage converts a Tharsis task stage into a go-tfe TaskStage carrying one OPA
// PolicyEvaluation per policy check the stage owns. The task stage's ID is the persisted stage node
// GID, giving it stable identity independent of its checks. See TaskStage and PolicyEvaluation for why
// these are local mirror types rather than the go-tfe ones.
func tharsisTaskStageToTaskStage(stage *models.RunTaskStage) *TaskStage {
	ts := &TaskStage{
		ID:     stage.GetGlobalID(),
		Stage:  string(tharsisRunStageToTFEStage(stage.StageName)),
		Status: string(tharsisTaskStageStatusToTFE(stage.Status)),
	}
	for _, check := range stage.PolicyChecks {
		count := tharsisPolicyResultCount(check)
		ts.PolicyEvaluations = append(ts.PolicyEvaluations, &PolicyEvaluation{
			ID:         check.GetGlobalID(),
			Status:     string(tharsisPolicyCheckStatusToPolicyEvaluationStatus(check.Status)),
			PolicyKind: string(gotfe.OPA),
			ResultCount: &PolicyResultCount{
				AdvisoryFailed:  count.AdvisoryFailed,
				MandatoryFailed: count.MandatoryFailed,
				Passed:          count.Passed,
				Errored:         count.Errored,
			},
		})
	}
	return ts
}

// tharsisPolicyResultCount tallies a check's per-policy results into a go-tfe PolicyResultCount.
func tharsisPolicyResultCount(check *models.PolicyCheck) *gotfe.PolicyResultCount {
	count := &gotfe.PolicyResultCount{}
	for _, p := range check.Policies {
		if p.Status == models.PolicyCheckPolicyPassed {
			count.Passed++
			continue
		}
		if p.Status == models.PolicyCheckPolicyFailed {
			if p.EnforcementLevel == models.PolicyEnforcementAdvisory {
				count.AdvisoryFailed++
			} else {
				count.MandatoryFailed++
			}
		}
	}
	return count
}

// tharsisEnforcementToTFE maps a Tharsis policy enforcement level onto the go-tfe enforcement level.
func tharsisEnforcementToTFE(level models.PolicyEnforcementLevel) gotfe.EnforcementLevel {
	switch level {
	case models.PolicyEnforcementSoftMandatory:
		return gotfe.EnforcementSoft
	case models.PolicyEnforcementHardMandatory:
		return gotfe.EnforcementHard
	default: // advisory
		return gotfe.EnforcementAdvisory
	}
}

// tharsisPolicyCheckToPolicySetOutcomes maps a Tharsis policy check onto the policy-set outcomes
// surfaced at /policy-evaluations/{id}/policy-set-outcomes, which the "cloud" backend reads to render
// OPA policy results. Tharsis models a check as a flat list of evaluated policies rather than nested
// policy sets, so the whole check is surfaced as a single policy-set outcome whose per-policy results
// become individual outcomes.
// messages holds each policy's violation messages keyed by policy id; the caller loads them, because
// they live in object storage and this mapping has no service access. A tfe outcome describes its
// failure with a single string, so the messages are joined for display.
func tharsisPolicyCheckToPolicySetOutcomes(check *models.PolicyCheck, messages map[string][]string) []*PolicySetOutcome {
	outcomes := make([]Outcome, len(check.Policies))
	for i, p := range check.Policies {
		policyTRN := trn.MustParseAny(p.Provenance.PolicyTRN)
		description := strings.Join(messages[p.ID], "\n")
		outcomes[i] = Outcome{
			EnforcementLevel: string(tharsisEnforcementToTFE(p.EnforcementLevel)),
			Status:           string(p.Status),
			PolicyName:       policyTRN.Path(),
			Description:      description,
		}
	}

	count := tharsisPolicyResultCount(check)
	// A soft-failed check can be cleared by a human override; a hard failure or a passing check cannot.
	overridable := check.Status == models.PolicyCheckSoftFailed
	return []*PolicySetOutcome{
		{
			ID:          check.GetGlobalID(),
			Outcomes:    outcomes,
			Overridable: &overridable,
			ResultCount: PolicyResultCount{
				AdvisoryFailed:  count.AdvisoryFailed,
				MandatoryFailed: count.MandatoryFailed,
				Passed:          count.Passed,
				Errored:         count.Errored,
			},
		},
	}
}

// tharsisRunStageToTFEStage maps a Tharsis run stage onto the go-tfe task-stage Stage.
func tharsisRunStageToTFEStage(stage models.RunTaskStageName) gotfe.Stage {
	switch stage {
	case models.RunTaskStageNamePrePlan:
		return gotfe.PrePlan
	case models.RunTaskStageNamePostPlan:
		return gotfe.PostPlan
	case models.RunTaskStageNamePreApply:
		return gotfe.PreApply
	case models.RunTaskStageNamePostApply:
		return gotfe.PostApply
	default:
		return gotfe.Stage(stage)
	}
}

// tharsisTaskStageStatusToTFE maps a Tharsis task stage status onto the go-tfe task-stage status.
// awaiting_override is a human override gate; completed maps to passed and skipped to unreachable,
// since go-tfe task stages have no dedicated values for those. A stage that is created, or pending
// (readied but still waiting for the workspace slot), is reported as go-tfe's pending — from a client's
// point of view both mean the stage has not started.
func tharsisTaskStageStatusToTFE(status models.RunTaskStageStatus) gotfe.TaskStageStatus {
	switch status {
	case models.RunTaskStagePending:
		return gotfe.TaskStagePending
	case models.RunTaskStageRunning:
		return gotfe.TaskStageRunning
	case models.RunTaskStageAwaitingOverride:
		return gotfe.TaskStageAwaitingOverride
	case models.RunTaskStageCompleted:
		return gotfe.TaskStagePassed
	case models.RunTaskStageErrored:
		return gotfe.TaskStageErrored
	case models.RunTaskStageCanceled:
		return gotfe.TaskStageCanceled
	case models.RunTaskStageSkipped:
		return gotfe.TaskStageUnreachable
	default: // created
		return gotfe.TaskStagePending
	}
}

// tharsisPolicyCheckStatusToPolicyEvaluationStatus maps a policy check status onto the go-tfe
// policy-evaluation status.
func tharsisPolicyCheckStatusToPolicyEvaluationStatus(status models.PolicyCheckStatus) gotfe.PolicyEvaluationStatus {
	switch status {
	case models.PolicyCheckRunning:
		return gotfe.PolicyEvaluationRunning
	case models.PolicyCheckPassed:
		return gotfe.PolicyEvaluationPassed
	case models.PolicyCheckSoftFailed:
		return gotfe.PolicyEvaluationFailed
	case models.PolicyCheckOverridden:
		return gotfe.PolicyEvaluationOverridden
	case models.PolicyCheckErrored:
		return gotfe.PolicyEvaluationErrored
	case models.PolicyCheckCanceled:
		return gotfe.PolicyEvaluationCanceled
	case models.PolicyCheckSkipped:
		return gotfe.PolicyEvaluationUnreachable
	default: // created, queued
		return gotfe.PolicyEvaluationPending
	}
}

// TharsisPolicyCheckToPolicyCheck converts a Tharsis policy check to a go-tfe PolicyCheck.
func TharsisPolicyCheckToPolicyCheck(check *models.PolicyCheck) *gotfe.PolicyCheck {
	result := &gotfe.PolicyResult{}
	for _, p := range check.Policies {
		switch p.Status {
		case models.PolicyCheckPolicyPassed:
			result.Passed++
		case models.PolicyCheckPolicyFailed:
			switch p.EnforcementLevel {
			case models.PolicyEnforcementHardMandatory:
				result.HardFailed++
			case models.PolicyEnforcementSoftMandatory:
				result.SoftFailed++
			case models.PolicyEnforcementAdvisory:
				result.AdvisoryFailed++
			}
		}
	}
	result.TotalFailed = result.HardFailed + result.SoftFailed + result.AdvisoryFailed
	result.Result = result.TotalFailed == 0

	status := tharsisPolicyCheckStatusToTFE(check.Status)
	if result.HardFailed > 0 {
		status = gotfe.PolicyHardFailed
	}

	overridable := check.Status == models.PolicyCheckSoftFailed
	return &gotfe.PolicyCheck{
		ID:     check.GetGlobalID(),
		Status: status,
		Result: result,
		Actions: &gotfe.PolicyActions{
			IsOverridable: overridable,
		},
		Permissions: &gotfe.PolicyPermissions{
			CanOverride: overridable,
		},
		Scope: gotfe.PolicyScopeOrganization,
	}
}

func tharsisPolicyCheckStatusToTFE(status models.PolicyCheckStatus) gotfe.PolicyStatus {
	switch status {
	case models.PolicyCheckCreated, models.PolicyCheckPending:
		// Both are pre-dispatch: no policy-eval job exists for the check yet. pending is a retried
		// check whose stage has not restarted, which the CLI cannot distinguish from a fresh one.
		return gotfe.PolicyPending
	case models.PolicyCheckQueued:
		return gotfe.PolicyQueued
	case models.PolicyCheckRunning:
		return gotfe.PolicyQueued
	case models.PolicyCheckPassed:
		return gotfe.PolicyPasses
	case models.PolicyCheckSoftFailed:
		return gotfe.PolicySoftFailed
	case models.PolicyCheckOverridden:
		return gotfe.PolicyOverridden
	case models.PolicyCheckErrored:
		return gotfe.PolicyErrored
	case models.PolicyCheckCanceled:
		return gotfe.PolicyCanceled
	case models.PolicyCheckSkipped:
		return gotfe.PolicyCanceled
	default:
		return gotfe.PolicyPending
	}
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
