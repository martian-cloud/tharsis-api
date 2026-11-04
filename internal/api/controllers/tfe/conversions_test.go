package tfe

import (
	"bytes"
	"reflect"
	"testing"

	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// TestTharsisRunStatusToTFE verifies each Tharsis run status maps onto the TFE run status the API
// surfaces. Tharsis statuses must never be passed through as raw strings: the Terraform CLI misbehaves
// on a value outside its own vocabulary, which is why TestTharsisRunStatusToTFE_Exhaustive below asserts
// every result is a known TFE status.
//
// Two groups are not 1:1. The *_awaiting_decision statuses for the pre-plan and pre-apply stages have no
// go-tfe counterpart and report policy_override, one of only two statuses the CLI keeps waiting on for an
// override. And Tharsis names one *_queuing status per workspace-gated node where TFE has only queuing
// and queuing_apply, so the four collapse onto those two by which side of the plan they sit on.
func TestTharsisRunStatusToTFE(t *testing.T) {
	cases := map[models.RunStatus]RunStatus{
		models.RunPending: RunPending,

		models.RunPrePlanQueuing:          RunQueuing,
		models.RunPrePlanRunning:          RunPrePlanRunning,
		models.RunPrePlanAwaitingDecision: RunPolicyOverride,
		models.RunPrePlanCompleted:        RunPrePlanCompleted,

		// Waiting for the workspace, then waiting for a runner. The distinction Tharsis draws here is the
		// one TFE draws between queuing and plan_queued, so both sides map straight across.
		models.RunPlanQueuing: RunQueuing,
		models.RunPlanQueued:  RunPlanQueued,
		models.RunPlanning:    RunPlanning,

		models.RunPostPlanRunning:          RunPostPlanRunning,
		models.RunPostPlanAwaitingDecision: RunPostPlanAwaitingDecision,
		models.RunPostPlanCompleted:        RunPostPlanCompleted,
		models.RunPlanned:                  RunPlanned,
		models.RunPlannedAndFinished:       RunPlannedAndFinished,

		models.RunPreApplyQueuing:          RunQueuingApply,
		models.RunPreApplyRunning:          RunPreApplyRunning,
		models.RunPreApplyAwaitingDecision: RunPolicyOverride,
		models.RunPreApplyCompleted:        RunPreApplyCompleted,

		models.RunApplyQueuing: RunQueuingApply,
		models.RunApplyQueued:  RunApplyQueued,
		models.RunApplying:     RunApplying,

		models.RunPostApplyRunning:   RunPostApplyRunning,
		models.RunPostApplyCompleted: RunPostApplyCompleted,

		models.RunApplied:   RunApplied,
		models.RunCanceled:  RunCanceled,
		models.RunDiscarded: RunDiscarded,
		models.RunErrored:   RunErrored,
	}
	for in, want := range cases {
		assert.Equal(t, want, tharsisRunStatusToTFE(in), "status %q", in)
	}
}

// TestTharsisRunStatusToTFE_QueuingCollapses is the CLI-compatibility assertion behind the mapping
// above: every status where Tharsis is waiting for the workspace must reach the CLI as queuing or
// queuing_apply, the two statuses it reads as "waiting for capacity". A gated node added later that
// forgot its mapping would land on the default arm and be reported as pending, which the CLI would read
// as a run that has not started.
func TestTharsisRunStatusToTFE_QueuingCollapses(t *testing.T) {
	beforeApply := map[models.RunStatus]struct{}{
		models.RunPreApplyQueuing: {},
		models.RunApplyQueuing:    {},
	}
	for _, status := range models.QueuingRunStatuses {
		want := RunQueuing
		if _, ok := beforeApply[status]; ok {
			want = RunQueuingApply
		}
		assert.Equal(t, want, tharsisRunStatusToTFE(status), "queuing status %q", status)
	}
}

// TestTharsisRunStatusToTFE_Exhaustive asserts every Tharsis run status has an explicit mapping, and
// that every mapping lands on a status the Terraform CLI understands. Without this a status added to
// models would silently fall through to the default arm, and a wrong or unknown status reaching the CLI
// breaks it in ways that are awkward to diagnose (see the tharsisRunStatusToTFE doc comment).
func TestTharsisRunStatusToTFE_Exhaustive(t *testing.T) {
	known := make(map[RunStatus]struct{}, len(allRunStatuses))
	for _, status := range allRunStatuses {
		known[status] = struct{}{}
	}

	for _, status := range models.AllRunStatuses {
		got := tharsisRunStatusToTFE(status)
		assert.Contains(t, known, got, "status %q maps to %q, which is not a TFE run status", status, got)
		// The default arm returns pending, so only the genuine pending status may map to it.
		if status != models.RunPending {
			assert.NotEqual(t, RunPending, got, "status %q has no explicit mapping", status)
		}
	}
}

// TestTharsisWorkspaceToWorkspace_CurrentRun verifies the workspace's current-run relation is the run
// holding the workspace slot, and that go-tfe decodes it: the Terraform CLI uses it to stop counting a
// waiting run's queue position at the run that is currently active.
func TestTharsisWorkspaceToWorkspace_CurrentRun(t *testing.T) {
	newWorkspace := func(currentApplyRunID *string) *models.Workspace {
		return &models.Workspace{
			Metadata:          models.ResourceMetadata{ID: "11111111-1111-1111-1111-111111111111"},
			Name:              "ws",
			CurrentApplyRunID: currentApplyRunID,
		}
	}

	t.Run("no run holds the workspace", func(t *testing.T) {
		assert.Nil(t, TharsisWorkspaceToWorkspace(newWorkspace(nil)).CurrentRun)
		assert.Nil(t, TharsisWorkspaceToWorkspace(newWorkspace(new(string))).CurrentRun)
	})

	t.Run("a run holds the workspace", func(t *testing.T) {
		runID := "22222222-2222-2222-2222-222222222222"
		ws := TharsisWorkspaceToWorkspace(newWorkspace(&runID))
		require.NotNil(t, ws.CurrentRun)
		assert.Equal(t, gid.ToGlobalID(types.RunModelType, runID), ws.CurrentRun.ID)

		var buf bytes.Buffer
		require.NoError(t, jsonapi.MarshalPayload(&buf, ws))
		decoded := new(gotfe.Workspace)
		require.NoError(t, jsonapi.UnmarshalPayload(&buf, decoded))
		require.NotNil(t, decoded.CurrentRun)
		assert.Equal(t, ws.CurrentRun.ID, decoded.CurrentRun.ID)
	})
}

// TestTharsisRunToRun_TaskStages verifies a run's pre-plan and post-plan policy checks are surfaced
// as distinct task stages, each carrying one policy evaluation per check with the mapped status and
// TestTharsisRunToRun_TaskStages verifies a run's pre-plan and post-plan policy checks are surfaced
// as distinct task stages. An OPA check becomes a policy evaluation with the mapped status; a
// module attestation check becomes a task result instead -- go-tfe has no policy-kind word for it,
// so it must not appear among the policy evaluations at all.
func TestTharsisRunToRun_TaskStages(t *testing.T) {
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunPrePlanAwaitingDecision,
		Plan:        models.Plan{ID: "plan-1"},
		TaskStages: []*models.RunTaskStage{
			{ID: "pre-stage", StageName: models.RunTaskStageNamePrePlan, Status: models.RunTaskStageAwaitingOverride, PolicyChecks: []*models.PolicyCheck{
				{ID: "pre-1", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckSoftFailed},
				{
					ID: "pre-2", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindModuleAttestation, Status: models.PolicyCheckPassed,
					MessagesSummary: &models.PolicyCheckMessagesSummary{Messages: []string{"attestation verified"}},
					Policies:        []*models.PolicyCheckPolicy{{EnforcementLevel: models.PolicyEnforcementHardMandatory}},
				},
			}},
			{ID: "post-stage", StageName: models.RunTaskStageNamePostPlan, Status: models.RunTaskStageCompleted, PolicyChecks: []*models.PolicyCheck{
				{ID: "post-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckPassed},
			}},
		},
	}

	resp := TharsisRunToRun(run)

	require.Len(t, resp.TaskStages, 2)
	// The legacy policy-checks relation carries only the post-plan check: classic TFE has no pre-plan
	// policies, so pre-plan checks are surfaced solely through the task-stages relation.
	require.Len(t, resp.PolicyChecks, 1)
	assert.Equal(t, run.TaskStages[1].PolicyChecks[0].GetGlobalID(), resp.PolicyChecks[0].ID, "only the post-plan check is surfaced")

	byStage := map[string]*TaskStage{}
	for _, ts := range resp.TaskStages {
		byStage[ts.Stage] = ts
	}

	pre := byStage[string(gotfe.PrePlan)]
	require.NotNil(t, pre)
	assert.Equal(t, string(gotfe.TaskStageAwaitingOverride), pre.Status)
	// Only the OPA check is a policy evaluation.
	require.Len(t, pre.PolicyEvaluations, 1)
	assert.Equal(t, string(gotfe.OPA), pre.PolicyEvaluations[0].PolicyKind)
	assert.Equal(t, string(gotfe.PolicyEvaluationFailed), pre.PolicyEvaluations[0].Status)
	// The module attestation check is a task result, not a policy evaluation.
	require.Len(t, pre.TaskResults, 1)
	assert.Equal(t, string(gotfe.TaskPassed), pre.TaskResults[0].Status)
	assert.Equal(t, "Module Attestation", pre.TaskResults[0].TaskName)
	assert.Equal(t, "attestation verified", pre.TaskResults[0].Message)
	assert.Equal(t, string(gotfe.Mandatory), pre.TaskResults[0].WorkspaceTaskEnforcementLevel)

	post := byStage[string(gotfe.PostPlan)]
	require.NotNil(t, post)
	assert.Equal(t, string(gotfe.TaskStagePassed), post.Status)
	require.Len(t, post.PolicyEvaluations, 1)
	assert.Equal(t, string(gotfe.OPA), post.PolicyEvaluations[0].PolicyKind)
	assert.Equal(t, string(gotfe.PolicyEvaluationPassed), post.PolicyEvaluations[0].Status)
	assert.Empty(t, post.TaskResults, "an all-OPA stage has no task results")
}

// TestTharsisPolicyCheckStatusToTaskResultStatus verifies every Tharsis policy check status maps to
// a go-tfe task-result status the CLI understands, including the two collapses this mapping makes:
// a soft-failed and an overridden check both report failed (go-tfe task results have no
// soft-failed/overridden concept -- see tharsisPolicyCheckStatusToTaskResultStatus), and a canceled
// or skipped check reports unreachable.
func TestTharsisPolicyCheckStatusToTaskResultStatus(t *testing.T) {
	cases := map[models.PolicyCheckStatus]gotfe.TaskResultStatus{
		models.PolicyCheckCreated:    gotfe.TaskPending,
		models.PolicyCheckPending:    gotfe.TaskPending,
		models.PolicyCheckQueued:     gotfe.TaskPending,
		models.PolicyCheckRunning:    gotfe.TaskRunning,
		models.PolicyCheckPassed:     gotfe.TaskPassed,
		models.PolicyCheckSoftFailed: gotfe.TaskFailed,
		models.PolicyCheckOverridden: gotfe.TaskFailed,
		models.PolicyCheckErrored:    gotfe.TaskErrored,
		models.PolicyCheckCanceled:   gotfe.TaskUnreachable,
		models.PolicyCheckSkipped:    gotfe.TaskUnreachable,
	}

	for status, want := range cases {
		assert.Equal(t, want, tharsisPolicyCheckStatusToTaskResultStatus(status), "status %q", status)
	}
}

// TestTharsisPolicyCheckEnforcementToTFETask verifies enforcement collapses onto go-tfe's two-level
// TaskEnforcementLevel: soft_mandatory reports mandatory, same as hard_mandatory, since go-tfe run
// tasks have no in-between and Tharsis's override already happens at the task-stage level regardless
// of which mandatory level a check declares. advisory only wins when every policy on the check is
// advisory.
func TestTharsisPolicyCheckEnforcementToTFETask(t *testing.T) {
	tests := []struct {
		name   string
		levels []models.PolicyEnforcementLevel
		want   gotfe.TaskEnforcementLevel
	}{
		{name: "all advisory", levels: []models.PolicyEnforcementLevel{models.PolicyEnforcementAdvisory, models.PolicyEnforcementAdvisory}, want: gotfe.Advisory},
		{name: "soft mandatory present", levels: []models.PolicyEnforcementLevel{models.PolicyEnforcementAdvisory, models.PolicyEnforcementSoftMandatory}, want: gotfe.Mandatory},
		{name: "hard mandatory present", levels: []models.PolicyEnforcementLevel{models.PolicyEnforcementHardMandatory}, want: gotfe.Mandatory},
		{name: "no policies defaults to advisory", levels: nil, want: gotfe.Advisory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := &models.PolicyCheck{}
			for _, level := range tt.levels {
				check.Policies = append(check.Policies, &models.PolicyCheckPolicy{EnforcementLevel: level})
			}
			assert.Equal(t, tt.want, tharsisPolicyCheckEnforcementToTFETask(check))
		})
	}
}

// TestTharsisPolicyCheckToPolicySetOutcomes verifies a policy check is surfaced as a single
// policy-set outcome whose per-policy results become individual outcomes with the mapped status and
// enforcement level, and that a soft-failed check is reported as overridable.
func TestTharsisPolicyCheckToPolicySetOutcomes(t *testing.T) {
	check := &models.PolicyCheck{
		ID:     "check-1",
		Status: models.PolicyCheckSoftFailed,
		Policies: []*models.PolicyCheckPolicy{
			{ID: "pol-net", OPAData: &models.PolicyCheckOPAData{PackageSource: "acme/net"}, Status: models.PolicyCheckPolicyPassed, EnforcementLevel: models.PolicyEnforcementAdvisory, Provenance: models.PolicyCheckPolicyProvenance{PolicyTRN: "trn:policy:acme/net"}},
			{ID: "pol-sec", OPAData: &models.PolicyCheckOPAData{PackageSource: "acme/sec"}, Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementSoftMandatory, Provenance: models.PolicyCheckPolicyProvenance{PolicyTRN: "trn:policy:acme/sec"}},
		},
	}

	// A tfe outcome describes its failure with one string, so a policy's messages are joined.
	messages := map[string][]string{"pol-sec": {"denied by rule X", "denied by rule Y"}}

	out := tharsisPolicyCheckToPolicySetOutcomes(check, messages)
	require.Len(t, out, 1)
	require.NotNil(t, out[0].Overridable)
	assert.True(t, *out[0].Overridable, "a soft-failed check is overridable")
	require.Len(t, out[0].Outcomes, 2)

	assert.Equal(t, "acme/net", out[0].Outcomes[0].PolicyName)
	assert.Equal(t, "passed", out[0].Outcomes[0].Status)
	assert.Equal(t, string(gotfe.EnforcementAdvisory), out[0].Outcomes[0].EnforcementLevel)

	assert.Equal(t, "failed", out[0].Outcomes[1].Status)
	assert.Equal(t, string(gotfe.EnforcementSoft), out[0].Outcomes[1].EnforcementLevel)
	assert.Equal(t, "denied by rule X\ndenied by rule Y", out[0].Outcomes[1].Description)
	assert.Empty(t, out[0].Outcomes[0].Description, "a passing policy has no messages")
}

// TestPolicySetOutcomes_RoundTripToGoTFE guards the nested-attribute serialization: it marshals the
// response with hashicorp/jsonapi (as the API does) and decodes it into the go-tfe PolicySetOutcome
// type (as the terraform CLI does). This fails if the nested outcome/result-count keys don't match
// the jsonapi tags the client reads — the bug that produced "0 policies evaluated" with blank names.
func TestPolicySetOutcomes_RoundTripToGoTFE(t *testing.T) {
	check := &models.PolicyCheck{
		ID:     "check-1",
		Status: models.PolicyCheckSoftFailed,
		Policies: []*models.PolicyCheckPolicy{
			{ID: "pol-sec", OPAData: &models.PolicyCheckOPAData{PackageSource: "acme/sec"}, Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementSoftMandatory, Provenance: models.PolicyCheckPolicyProvenance{PolicyTRN: "trn:policy:acme/sec"}},
		},
	}
	messages := map[string][]string{"pol-sec": {"denied by rule X"}}

	var buf bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayload(&buf, tharsisPolicyCheckToPolicySetOutcomes(check, messages)))

	decoded, err := jsonapi.UnmarshalManyPayload(&buf, reflect.TypeOf(new(gotfe.PolicySetOutcome)))
	require.NoError(t, err)
	require.Len(t, decoded, 1)

	out := decoded[0].(*gotfe.PolicySetOutcome)
	require.NotNil(t, out.Overridable)
	assert.True(t, *out.Overridable)
	assert.Equal(t, 1, out.ResultCount.MandatoryFailed, "result count must decode (not 0)")
	require.Len(t, out.Outcomes, 1)
	assert.Equal(t, "acme/sec", out.Outcomes[0].PolicyName, "policy name must decode (not blank)")
	assert.Equal(t, "failed", out.Outcomes[0].Status)
	assert.Equal(t, gotfe.EnforcementSoft, out.Outcomes[0].EnforcementLevel)
	assert.Equal(t, "denied by rule X", out.Outcomes[0].Description)
}

// TestTaskStage_ResultCountRoundTripToGoTFE guards the task-stage response's nested
// PolicyEvaluation.ResultCount: it must decode into the go-tfe type (as the CLI does) with a
// non-zero count, so the CLI does not print "0 policies evaluated" for a failed policy.
func TestTaskStage_ResultCountRoundTripToGoTFE(t *testing.T) {
	stage := &models.RunTaskStage{
		ID:        "stage-1",
		StageName: models.RunTaskStageNamePrePlan,
		Status:    models.RunTaskStageAwaitingOverride,
		PolicyChecks: []*models.PolicyCheck{
			{ID: "check-1", CheckType: models.PolicyKindOPA, Status: models.PolicyCheckSoftFailed, Policies: []*models.PolicyCheckPolicy{
				{Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementSoftMandatory},
			}},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayload(&buf, tharsisTaskStageToTaskStage(stage)))

	out := new(gotfe.TaskStage)
	require.NoError(t, jsonapi.UnmarshalPayload(&buf, out))
	require.Len(t, out.PolicyEvaluations, 1)
	assert.Equal(t, gotfe.OPA, out.PolicyEvaluations[0].PolicyKind)
	require.NotNil(t, out.PolicyEvaluations[0].ResultCount)
	assert.Equal(t, 1, out.PolicyEvaluations[0].ResultCount.MandatoryFailed, "result count must decode (not 0)")
}

// TestTaskStage_ModuleAttestationTaskResultRoundTripToGoTFE guards the jsonapi round trip for a
// module attestation check: it must decode as a TaskResult on the go-tfe TaskStage, with its
// status/message/task-name/enforcement-level attributes intact, and must not appear among
// PolicyEvaluations at all.
func TestTaskStage_ModuleAttestationTaskResultRoundTripToGoTFE(t *testing.T) {
	stage := &models.RunTaskStage{
		ID:        "stage-1",
		StageName: models.RunTaskStageNamePrePlan,
		Status:    models.RunTaskStageCompleted,
		PolicyChecks: []*models.PolicyCheck{
			{
				ID: "check-1", CheckType: models.PolicyKindModuleAttestation, Status: models.PolicyCheckPassed,
				MessagesSummary: &models.PolicyCheckMessagesSummary{Messages: []string{"attestation verified"}},
				Policies:        []*models.PolicyCheckPolicy{{EnforcementLevel: models.PolicyEnforcementHardMandatory}},
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayload(&buf, tharsisTaskStageToTaskStage(stage)))

	out := new(gotfe.TaskStage)
	require.NoError(t, jsonapi.UnmarshalPayload(&buf, out))
	assert.Empty(t, out.PolicyEvaluations, "a module attestation check is not a policy evaluation")
	require.Len(t, out.TaskResults, 1)
	assert.Equal(t, gotfe.TaskPassed, out.TaskResults[0].Status)
	assert.Equal(t, "attestation verified", out.TaskResults[0].Message)
	assert.Equal(t, "Module Attestation", out.TaskResults[0].TaskName)
	assert.Equal(t, gotfe.Mandatory, out.TaskResults[0].WorkspaceTaskEnforcementLevel)
}

// TestTharsisPolicyCheckToPolicyCheck verifies the legacy policy check tallies failures by enforcement
// level, reports hard-failed whenever a hard-mandatory policy failed, and is overridable only when
// soft-failed.
func TestTharsisPolicyCheckToPolicyCheck(t *testing.T) {
	policies := func(levels ...models.PolicyEnforcementLevel) []*models.PolicyCheckPolicy {
		result := []*models.PolicyCheckPolicy{{Status: models.PolicyCheckPolicyPassed}, {Status: models.PolicyCheckPolicyPending}}
		for _, level := range levels {
			result = append(result, &models.PolicyCheckPolicy{Status: models.PolicyCheckPolicyFailed, EnforcementLevel: level})
		}
		return result
	}

	tests := []struct {
		name              string
		status            models.PolicyCheckStatus
		policies          []*models.PolicyCheckPolicy
		expectStatus      gotfe.PolicyStatus
		expectResult      gotfe.PolicyResult
		expectOverridable bool
	}{
		{
			name:         "passed",
			status:       models.PolicyCheckPassed,
			policies:     policies(),
			expectStatus: gotfe.PolicyPasses,
			expectResult: gotfe.PolicyResult{Passed: 1, Result: true},
		},
		{
			name:         "advisory failures still pass",
			status:       models.PolicyCheckPassed,
			policies:     policies(models.PolicyEnforcementAdvisory),
			expectStatus: gotfe.PolicyPasses,
			expectResult: gotfe.PolicyResult{Passed: 1, AdvisoryFailed: 1, TotalFailed: 1},
		},
		{
			name:              "soft-failed is overridable",
			status:            models.PolicyCheckSoftFailed,
			policies:          policies(models.PolicyEnforcementSoftMandatory, models.PolicyEnforcementAdvisory),
			expectStatus:      gotfe.PolicySoftFailed,
			expectResult:      gotfe.PolicyResult{Passed: 1, SoftFailed: 1, AdvisoryFailed: 1, TotalFailed: 2},
			expectOverridable: true,
		},
		{
			name:              "a hard-mandatory failure reports hard-failed whatever the check status",
			status:            models.PolicyCheckSoftFailed,
			policies:          policies(models.PolicyEnforcementHardMandatory, models.PolicyEnforcementSoftMandatory),
			expectStatus:      gotfe.PolicyHardFailed,
			expectResult:      gotfe.PolicyResult{Passed: 1, HardFailed: 1, SoftFailed: 1, TotalFailed: 2},
			expectOverridable: true,
		},
		{
			name:         "a failure at an unknown enforcement level is not counted",
			status:       models.PolicyCheckErrored,
			policies:     policies("unknown"),
			expectStatus: gotfe.PolicyErrored,
			expectResult: gotfe.PolicyResult{Passed: 1, Result: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check := &models.PolicyCheck{ID: "check-1", Status: test.status, Policies: test.policies}

			pc := TharsisPolicyCheckToPolicyCheck(check)

			assert.Equal(t, check.GetGlobalID(), pc.ID)
			assert.Equal(t, test.expectStatus, pc.Status)
			assert.Equal(t, test.expectResult, *pc.Result)
			assert.Equal(t, test.expectOverridable, pc.Actions.IsOverridable)
			assert.Equal(t, test.expectOverridable, pc.Permissions.CanOverride)
		})
	}
}

// TestTharsisStateVersionToStateVersion_noAPIURL verifies no content URLs are advertised without a
// configured API URL, and that the run relation is set only for a state version created by a run.
func TestTharsisStateVersionToStateVersion_noAPIURL(t *testing.T) {
	runID := "33333333-3333-3333-3333-333333333333"
	sv := &models.StateVersion{
		Metadata:    models.ResourceMetadata{ID: "11111111-1111-1111-1111-111111111111"},
		WorkspaceID: "22222222-2222-2222-2222-222222222222",
	}

	resp := TharsisStateVersionToStateVersion(sv, "", "/tfe/v2")
	assert.Empty(t, resp.DownloadURL)
	assert.Empty(t, resp.JSONUploadURL)
	assert.Empty(t, resp.JSONDownloadURL)
	assert.Nil(t, resp.Run)

	sv.RunID = &runID
	resp = TharsisStateVersionToStateVersion(sv, "", "/tfe/v2")
	require.NotNil(t, resp.Run)
	assert.Equal(t, gid.ToGlobalID(types.RunModelType, runID), resp.Run.ID)
	assert.Equal(t, gid.ToGlobalID(types.WorkspaceModelType, sv.WorkspaceID), resp.Run.Workspace.ID)
}

// TestTharsisVariableToVariable verifies a variable without a value reads as empty and that only the
// terraform category maps to terraform: every other category, known or not, is an environment variable.
func TestTharsisVariableToVariable(t *testing.T) {
	value := "v"
	tests := []struct {
		name           string
		value          *string
		category       models.VariableCategory
		expectValue    string
		expectCategory CategoryType
	}{
		{name: "terraform variable", value: &value, category: models.TerraformVariableCategory, expectValue: "v", expectCategory: CategoryTerraform},
		{name: "environment variable", value: &value, category: models.EnvironmentVariableCategory, expectValue: "v", expectCategory: CategoryEnv},
		{name: "unknown category", value: &value, category: "unknown", expectValue: "v", expectCategory: CategoryEnv},
		{name: "no value", category: models.TerraformVariableCategory, expectCategory: CategoryTerraform},
	}

	ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			variable := &models.Variable{Metadata: models.ResourceMetadata{ID: "var-1"}, Key: "k", Value: test.value, Category: test.category}

			got := TharsisVariableToVariable(variable, ws)

			assert.Equal(t, variable.GetGlobalID(), got.ID)
			assert.Equal(t, "k", got.Key)
			assert.Equal(t, test.expectValue, got.Value)
			assert.Equal(t, test.expectCategory, got.Category)
			assert.Equal(t, ws.GetGlobalID(), got.Workspace.ID)
		})
	}
}

// TestTharsisPolicyResultCount tallies per-policy results into the go-tfe result count.
func TestTharsisPolicyResultCount(t *testing.T) {
	check := &models.PolicyCheck{
		Policies: []*models.PolicyCheckPolicy{
			{Status: models.PolicyCheckPolicyPassed},
			{Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementAdvisory},
			{Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementHardMandatory},
			{Status: models.PolicyCheckPolicyFailed, EnforcementLevel: models.PolicyEnforcementSoftMandatory},
		},
	}

	count := tharsisPolicyResultCount(check)
	assert.Equal(t, 1, count.Passed)
	assert.Equal(t, 1, count.AdvisoryFailed)
	assert.Equal(t, 2, count.MandatoryFailed)
}
