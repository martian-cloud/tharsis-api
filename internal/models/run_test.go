package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunStatus_IsFinalStatus(t *testing.T) {
	final := []RunStatus{RunApplied, RunPlannedAndFinished, RunErrored, RunCanceled, RunDiscarded}
	nonFinal := []RunStatus{RunPending, RunQueuing, RunPlanQueued, RunPlanning, RunPlanned, RunQueuingApply, RunApplyQueued, RunApplying}

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
