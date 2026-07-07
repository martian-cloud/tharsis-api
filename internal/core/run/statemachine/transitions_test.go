package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestRunTransitions_Discard(t *testing.T) {
	// A planned run may be discarded; no other status may transition to discarded.
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunDiscarded), "planned -> discarded should be allowed")

	for _, s := range []models.RunStatus{models.RunPending, models.RunQueuing, models.RunPlanQueued, models.RunPlanning, models.RunQueuingApply, models.RunApplyQueued, models.RunApplying, models.RunApplied, models.RunErrored, models.RunCanceled} {
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunDiscarded), "%s -> discarded should not be allowed", s)
	}

	// discarded is reversible only back to planned (undiscard); no other outgoing edge.
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPlanned), "discarded -> planned (undiscard) should be allowed")
	for _, s := range []models.RunStatus{models.RunPending, models.RunQueuing, models.RunPlanQueued, models.RunPlanning, models.RunQueuingApply, models.RunApplyQueued, models.RunApplying, models.RunApplied, models.RunErrored, models.RunCanceled} {
		assert.Falsef(t, canTransitionTo(runTransitions, models.RunDiscarded, s), "discarded -> %s should not be allowed", s)
	}
}

func TestRunTransitions_Queuing(t *testing.T) {
	// The plan/apply nodes always pass through pending (waiting to be queued), so the
	// run passes through queuing / queuing_apply — the direct edges are gone.
	assert.True(t, canTransitionTo(runTransitions, models.RunPending, models.RunQueuing), "pending -> queuing should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunQueuing, models.RunPlanQueued), "queuing -> plan_queued should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunQueuing, models.RunCanceled), "queuing -> canceled should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPending, models.RunPlanQueued), "pending -> plan_queued should not skip queuing")

	assert.True(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunQueuingApply), "planned -> queuing_apply should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunQueuingApply, models.RunApplyQueued), "queuing_apply -> apply_queued should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunQueuingApply, models.RunCanceled), "queuing_apply -> canceled should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunApplyQueued), "planned -> apply_queued should not skip queuing_apply")

	// Retry resets land on the queuing states.
	for _, s := range []models.RunStatus{models.RunErrored, models.RunCanceled} {
		assert.Truef(t, canTransitionTo(runTransitions, s, models.RunQueuing), "%s -> queuing (plan retry) should be allowed", s)
		assert.Truef(t, canTransitionTo(runTransitions, s, models.RunQueuingApply), "%s -> queuing_apply (apply retry) should be allowed", s)
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunPending), "%s -> pending should not be allowed", s)
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunPlanned), "%s -> planned should not be allowed", s)
	}
}

func TestApplyTransitions_Skipped(t *testing.T) {
	// Only a never-started (created) apply may be skipped.
	assert.True(t, canTransitionTo(applyTransitions, models.ApplyCreated, models.ApplySkipped), "created -> skipped should be allowed")
	for _, s := range []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled} {
		assert.Falsef(t, canTransitionTo(applyTransitions, s, models.ApplySkipped), "%s -> skipped should not be allowed", s)
	}

	// A skipped apply may only be reset back to created (the plan-retry reset);
	// it can never advance directly into the apply lifecycle.
	assert.True(t, canTransitionTo(applyTransitions, models.ApplySkipped, models.ApplyCreated), "skipped -> created should be allowed")
	for _, s := range []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled} {
		assert.Falsef(t, canTransitionTo(applyTransitions, models.ApplySkipped, s), "skipped -> %s should not be allowed", s)
	}
}
