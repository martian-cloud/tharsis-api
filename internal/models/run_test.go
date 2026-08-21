package models

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStatus_IsFinalStatus(t *testing.T) {
	final := []RunStatus{RunApplied, RunPlannedAndFinished, RunErrored, RunCanceled, RunDiscarded}
	nonFinal := []RunStatus{
		RunPending,
		RunPrePlanQueuing, RunPlanQueuing, RunPlanQueued, RunPlanning, RunPlanned,
		RunPreApplyQueuing, RunApplyQueuing, RunApplyQueued, RunApplying,
	}

	for _, s := range final {
		assert.Truef(t, s.IsFinalStatus(), "%s should be final", s)
	}
	for _, s := range nonFinal {
		assert.Falsef(t, s.IsFinalStatus(), "%s should not be final", s)
	}
}

func TestPlanStatus_IsFinalStatus(t *testing.T) {
	final := []PlanStatus{PlanFinished, PlanErrored, PlanCanceled}
	nonFinal := []PlanStatus{PlanCreated, PlanPending, PlanQueued, PlanRunning}

	for _, s := range final {
		assert.Truef(t, s.IsFinalStatus(), "%s should be final", s)
	}
	for _, s := range nonFinal {
		assert.Falsef(t, s.IsFinalStatus(), "%s should not be final", s)
	}
}

func TestApplyStatus_IsFinalStatus(t *testing.T) {
	final := []ApplyStatus{ApplyFinished, ApplyErrored, ApplyCanceled, ApplySkipped}
	nonFinal := []ApplyStatus{ApplyCreated, ApplyPending, ApplyQueued, ApplyRunning}

	for _, s := range final {
		assert.Truef(t, s.IsFinalStatus(), "%s should be final", s)
	}
	for _, s := range nonFinal {
		assert.Falsef(t, s.IsFinalStatus(), "%s should not be final", s)
	}
}

func TestRun_Diff_ApplyNullability(t *testing.T) {
	// base produces two structurally identical runs so Diff isolates apply-node changes
	// (run-level and plan fields compare equal and are never flagged).
	base := func() *Run {
		return &Run{
			Metadata: ResourceMetadata{ID: "run-1"},
			Plan:     Plan{ID: "plan-1", Status: PlanCreated},
		}
	}
	withApply := func(status ApplyStatus) *Run {
		r := base()
		r.Apply = &Apply{ID: "apply-1", Status: status}
		return r
	}

	t.Run("apply present now, absent before is flagged", func(t *testing.T) {
		assert.Equal(t, []string{"apply-1"}, withApply(ApplyCreated).Diff(base()))
	})

	t.Run("apply absent in both is not flagged", func(t *testing.T) {
		assert.Empty(t, base().Diff(base()))
	})

	t.Run("apply present and equal in both is not flagged", func(t *testing.T) {
		assert.Empty(t, withApply(ApplyCreated).Diff(withApply(ApplyCreated)))
	})

	t.Run("apply present in both but differing is flagged", func(t *testing.T) {
		assert.Equal(t, []string{"apply-1"}, withApply(ApplyRunning).Diff(withApply(ApplyCreated)))
	})
}

func TestRun_ShallowCompare_AllFields(t *testing.T) {
	ptr := func(s string) *string { return &s }
	now := time.Now()

	// full builds a run with every run-level content field populated.
	full := func() *Run {
		return &Run{
			ConfigurationVersionID: ptr("cv-1"),
			ForceCancelAvailableAt: &now,
			ForceCanceledBy:        ptr("user-1"),
			ModuleVersion:          ptr("1.0.0"),
			ModuleSource:           ptr("registry/mod"),
			TargetAddresses:        []string{"a", "b"},
			ModuleDigest:           []byte{1, 2, 3},
			CreatedBy:              "creator",
			WorkspaceID:            "ws-1",
			Status:                 RunPending,
			Comment:                "hello",
			TerraformVersion:       "1.7.0",
			Metadata:               ResourceMetadata{ID: "run-1"},
			IsDestroy:              true,
			IsAssessmentRun:        true,
			AutoApply:              true,
			Refresh:                true,
			Plan:                   Plan{ID: "plan-1", Status: PlanCreated},
		}
	}

	t.Run("identical runs compare equal", func(t *testing.T) {
		assert.True(t, full().ShallowCompare(full()))
	})

	// Every run-level content field, mutated one at a time, must be detected.
	cases := map[string]func(*Run){
		"Status":                 func(r *Run) { r.Status = RunApplied },
		"CreatedBy":              func(r *Run) { r.CreatedBy = "other" },
		"WorkspaceID":            func(r *Run) { r.WorkspaceID = "ws-2" },
		"Comment":                func(r *Run) { r.Comment = "changed" },
		"TerraformVersion":       func(r *Run) { r.TerraformVersion = "1.8.0" },
		"IsDestroy":              func(r *Run) { r.IsDestroy = false },
		"IsAssessmentRun":        func(r *Run) { r.IsAssessmentRun = false },
		"ForceCanceled":          func(r *Run) { r.ForceCanceled = true },
		"AutoApply":              func(r *Run) { r.AutoApply = false },
		"Refresh":                func(r *Run) { r.Refresh = false },
		"RefreshOnly":            func(r *Run) { r.RefreshOnly = true },
		"ConfigurationVersionID": func(r *Run) { r.ConfigurationVersionID = ptr("cv-2") },
		"ModuleSource":           func(r *Run) { r.ModuleSource = ptr("registry/other") },
		"ModuleVersion":          func(r *Run) { r.ModuleVersion = ptr("2.0.0") },
		"ForceCanceledBy":        func(r *Run) { r.ForceCanceledBy = ptr("user-2") },
		"ForceCancelAvailableAt": func(r *Run) { later := now.Add(time.Hour); r.ForceCancelAvailableAt = &later },
		"TargetAddresses":        func(r *Run) { r.TargetAddresses = []string{"a", "c"} },
		"ModuleDigest":           func(r *Run) { r.ModuleDigest = []byte{1, 2, 4} },
	}
	for name, mutate := range cases {
		t.Run(name+" change is detected", func(t *testing.T) {
			r := full()
			mutate(r)
			assert.False(t, r.ShallowCompare(full()))
		})
	}

	t.Run("Metadata and nodes are excluded from run-level compare", func(t *testing.T) {
		r := full()
		r.Metadata = ResourceMetadata{ID: "run-1", Version: 99}
		r.Plan.Status = PlanFinished
		assert.True(t, r.ShallowCompare(full()))
	})
}

func TestRun_Copy_SlicesAreIndependent(t *testing.T) {
	orig := &Run{
		Metadata:        ResourceMetadata{ID: "run-1"},
		Plan:            Plan{ID: "plan-1"},
		TargetAddresses: []string{"a", "b"},
		ModuleDigest:    []byte{1, 2, 3},
	}
	cp := orig.Copy()

	// Mutating the copy's slices in place must not affect the original.
	cp.TargetAddresses[0] = "z"
	cp.ModuleDigest[0] = 9

	assert.Equal(t, []string{"a", "b"}, orig.TargetAddresses)
	assert.Equal(t, []byte{1, 2, 3}, orig.ModuleDigest)
	// And ShallowCompare detects the divergence between the mutated copy and the original.
	assert.False(t, orig.ShallowCompare(cp))
}

// TestPolicyCheck_MessagesSummary_CopyAndCompare covers the summary through the two paths the run
// engine relies on: a copy must not share the message slice with its original (the engine diffs a
// copy against the live run), and ShallowCompare must see a changed summary so the row is written.
func TestPolicyCheck_MessagesSummary_CopyAndCompare(t *testing.T) {
	orig := &PolicyCheck{
		ID:              "check-1",
		StageName:       RunTaskStageNamePostPlan,
		CheckType:       PolicyKindOPA,
		MessagesSummary: &PolicyCheckMessagesSummary{Messages: []string{"denied by rule X"}},
	}

	cp, ok := orig.Copy().(*PolicyCheck)
	require.True(t, ok)
	require.True(t, orig.ShallowCompare(cp), "a fresh copy should compare equal")

	cp.MessagesSummary.Messages[0] = "denied by rule Y"
	assert.Equal(t, []string{"denied by rule X"}, orig.MessagesSummary.Messages)
	assert.False(t, orig.ShallowCompare(cp), "a changed message should be detected")

	// Truncation is part of the comparison too: the flag alone changes what the UI renders.
	truncated, ok := orig.Copy().(*PolicyCheck)
	require.True(t, ok)
	truncated.MessagesSummary.Truncated = true
	assert.False(t, orig.ShallowCompare(truncated))

	// A check that has never reported is distinct from one that reported nothing, so that the
	// difference survives a round trip rather than both reading as "no messages".
	empty, ok := orig.Copy().(*PolicyCheck)
	require.True(t, ok)
	empty.MessagesSummary = nil
	assert.False(t, orig.ShallowCompare(empty))
}

func TestRun_HasChanges(t *testing.T) {
	tests := []struct {
		name           string
		planHasChanges bool
		want           bool
	}{
		{
			name:           "plan reports changes",
			planHasChanges: true,
			want:           true,
		},
		{
			name:           "plan reports no changes",
			planHasChanges: false,
			want:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HasChanges is derived from the plan node's flag, not stored on the run.
			r := &Run{Plan: Plan{HasChanges: tt.planHasChanges}}
			assert.Equal(t, tt.want, r.HasChanges())
		})
	}
}

// TestRun_ComputeHasAdvisoryFailures covers the rule behind the run's stored flag: only a *failed*
// *advisory* policy counts, wherever on the run it sits, and the recompute is what lets the flag go back
// to false once those failures are gone.
func TestRun_ComputeHasAdvisoryFailures(t *testing.T) {
	policy := func(level PolicyEnforcementLevel, status PolicyCheckPolicyStatus) *PolicyCheckPolicy {
		return &PolicyCheckPolicy{ID: "policy-1", EnforcementLevel: level, Status: status}
	}
	// runWithStages wraps each group of policies in its own task stage, so a case can put failures on a
	// stage other than the first.
	runWithStages := func(stages ...[]*PolicyCheckPolicy) *Run {
		r := &Run{Metadata: ResourceMetadata{ID: "run-1"}}
		for i, policies := range stages {
			r.TaskStages = append(r.TaskStages, &RunTaskStage{
				ID:           "stage-" + strconv.Itoa(i),
				PolicyChecks: []*PolicyCheck{{ID: "check-" + strconv.Itoa(i), Policies: policies}},
			})
		}
		return r
	}

	tests := []struct {
		name string
		run  *Run
		want bool
	}{
		{
			name: "no policy checks at all",
			run:  &Run{Metadata: ResourceMetadata{ID: "run-1"}},
		},
		{
			name: "advisory policy failed",
			run:  runWithStages([]*PolicyCheckPolicy{policy(PolicyEnforcementAdvisory, PolicyCheckPolicyFailed)}),
			want: true,
		},
		{
			name: "advisory policy passed",
			run:  runWithStages([]*PolicyCheckPolicy{policy(PolicyEnforcementAdvisory, PolicyCheckPolicyPassed)}),
		},
		{
			name: "advisory policy not evaluated yet",
			run:  runWithStages([]*PolicyCheckPolicy{policy(PolicyEnforcementAdvisory, PolicyCheckPolicyPending)}),
		},
		{
			// The distinction the flag exists for: a mandatory failure is already visible in the run's
			// status, so it must not set this.
			name: "soft-mandatory policy failed",
			run:  runWithStages([]*PolicyCheckPolicy{policy(PolicyEnforcementSoftMandatory, PolicyCheckPolicyFailed)}),
		},
		{
			name: "hard-mandatory policy failed",
			run:  runWithStages([]*PolicyCheckPolicy{policy(PolicyEnforcementHardMandatory, PolicyCheckPolicyFailed)}),
		},
		{
			// Every stage is walked, not just the first — a pre-plan check's findings still count once
			// the run has moved on to post-plan.
			name: "advisory failure on the second stage",
			run: runWithStages(
				[]*PolicyCheckPolicy{policy(PolicyEnforcementHardMandatory, PolicyCheckPolicyPassed)},
				[]*PolicyCheckPolicy{policy(PolicyEnforcementAdvisory, PolicyCheckPolicyFailed)},
			),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Seeded to the opposite of the expectation to prove the result comes from the checks and
			// not from the stored flag. That the caller's assignment then clears a stale true is covered
			// by the retry command's tests.
			tt.run.HasAdvisoryFailures = !tt.want
			assert.Equal(t, tt.want, tt.run.ComputeHasAdvisoryFailures())
		})
	}
}

// TestRun_HasAdvisoryFailures_CopyAndCompare verifies the flag is carried by Copy and noticed by
// ShallowCompare — the comparison Diff uses to decide the run row needs writing, which is the only thing
// that persists the field.
func TestRun_HasAdvisoryFailures_CopyAndCompare(t *testing.T) {
	orig := &Run{
		Metadata:            ResourceMetadata{ID: "run-1"},
		Plan:                Plan{ID: "plan-1"},
		HasAdvisoryFailures: true,
	}

	cp := orig.Copy()
	assert.True(t, cp.HasAdvisoryFailures)
	require.True(t, orig.ShallowCompare(cp), "a fresh copy should compare equal")

	cp.HasAdvisoryFailures = false
	assert.False(t, orig.ShallowCompare(cp), "a changed flag should be detected")
	assert.Equal(t, []string{orig.Metadata.ID}, orig.Diff(cp), "the change belongs to the run row")
}

// TestRun_NodeByPath verifies every kind of run node is reachable by the same run-relative path it
// reports from GetPath — the paths the runNode query and the retryRunNode mutation address nodes by.
// A path is checked against the node's own GetPath rather than a literal so the two cannot drift.
func TestRun_NodeByPath(t *testing.T) {
	check := &PolicyCheck{ID: "check-1", StageName: RunTaskStageNamePostPlan, CheckType: PolicyKindOPA}
	stage := &RunTaskStage{ID: "stage-1", StageName: RunTaskStageNamePostPlan, PolicyChecks: []*PolicyCheck{check}}
	run := &Run{
		Metadata:   ResourceMetadata{ID: "run-1"},
		Plan:       Plan{ID: "plan-1"},
		Apply:      &Apply{ID: "apply-1"},
		TaskStages: []*RunTaskStage{stage},
	}

	for _, node := range []RunNode{&run.Plan, run.Apply, stage, check} {
		assert.Samef(t, node, run.NodeByPath(node.GetPath()), "NodeByPath(%q)", node.GetPath())
	}

	// A stage path and a policy check path share a prefix, so neither may resolve to the other.
	assert.NotSame(t, stage, run.NodeByPath(check.GetPath()))

	assert.Nil(t, run.NodeByPath("pre_plan"), "a stage the run does not have")
	assert.Nil(t, run.NodeByPath("post_plan.sentinel"), "a check the stage does not have")
	assert.Nil(t, run.NodeByPath(""))

	// A speculative run has no apply node, and asking for it must not hand back a typed nil.
	speculative := &Run{Metadata: ResourceMetadata{ID: "run-2"}, Plan: Plan{ID: "plan-2"}}
	assert.Nil(t, speculative.NodeByPath(ApplyNodePath))
}

// TestRunStatus_IsQueuing verifies exactly the workspace-gated phases report as queuing — the condition
// the admitter acts on, the work item consumer and reconciler filter on, the admission indexes are scoped
// to, and the TFE controller collapses onto TFE's queuing / queuing_apply.
//
// It is driven off AllRunStatuses rather than a hand-written list of negatives, so a status added to the
// enum has to be classified deliberately: a new gated phase must be added to QueuingRunStatuses (and to
// both partial indexes in the add_policy_enforcement migration), and any other new status must not be.
func TestRunStatus_IsQueuing(t *testing.T) {
	queuing := map[RunStatus]struct{}{
		RunPrePlanQueuing:  {},
		RunPlanQueuing:     {},
		RunPreApplyQueuing: {},
		RunApplyQueuing:    {},
	}

	require.Len(t, QueuingRunStatuses, len(queuing), "QueuingRunStatuses and this test disagree on how many gated phases exist")
	for _, status := range QueuingRunStatuses {
		assert.Contains(t, queuing, status, "QueuingRunStatuses has unexpected status %q", status)
	}

	for _, status := range AllRunStatuses {
		_, want := queuing[status]
		assert.Equalf(t, want, status.IsQueuing(), "IsQueuing(%q)", status)
	}

	// The *_queued statuses are the other half of each gated node's wait: admitted, waiting for a runner.
	// Reporting them as queuing would put an already-admitted run back into the admission sweeps.
	for _, status := range []RunStatus{RunPlanQueued, RunApplyQueued} {
		assert.Falsef(t, status.IsQueuing(), "%q holds the workspace slot, so it is not queuing", status)
	}
}
