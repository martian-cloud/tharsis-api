package statemachine

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// runTransitions lists the statuses each run status may legally transition to. The
// run status is a projection of its plan/apply nodes, so these are the transitions
// the node listeners can drive. A plan or apply node always passes through pending
// (waiting to be queued) before being queued, so the run passes through queuing /
// queuing_apply on the way to plan_queued / apply_queued. The terminal
// applied/planned_and_finished statuses are absorbing; errored/canceled
// additionally permit the retry resets (a node being retried moves back to pending,
// so the run lands on queuing for a plan retry or queuing_apply for an apply retry).
// discarded is a run-level outcome (not driven by a child node) reachable only from
// planned, and reversible: undiscard returns it to planned (restoring the skipped apply).
var runTransitions = map[models.RunStatus][]models.RunStatus{
	models.RunPending:      {models.RunQueuing, models.RunCanceled},
	models.RunQueuing:      {models.RunPlanQueued, models.RunCanceled},
	models.RunPlanQueued:   {models.RunPlanning, models.RunErrored, models.RunCanceled},
	models.RunPlanning:     {models.RunPlanned, models.RunPlannedAndFinished, models.RunErrored, models.RunCanceled},
	models.RunPlanned:      {models.RunQueuingApply, models.RunCanceled, models.RunDiscarded},
	models.RunQueuingApply: {models.RunApplyQueued, models.RunCanceled},
	models.RunApplyQueued:  {models.RunApplying, models.RunErrored, models.RunCanceled},
	models.RunApplying:     {models.RunApplied, models.RunErrored, models.RunCanceled},
	models.RunErrored:      {models.RunQueuing, models.RunQueuingApply},
	models.RunCanceled:     {models.RunQueuing, models.RunQueuingApply},
	// discarded is otherwise terminal, but a discard may be reversed (undiscard),
	// returning the run to planned so it can be applied or discarded again.
	models.RunDiscarded: {models.RunPlanned},
}

// planTransitions lists the statuses each plan status may legally transition to.
// finished is absorbing; errored/canceled permit the retry reset back to pending.
var planTransitions = map[models.PlanStatus][]models.PlanStatus{
	models.PlanCreated:  {models.PlanPending, models.PlanCanceled},
	models.PlanPending:  {models.PlanQueued, models.PlanCanceled},
	models.PlanQueued:   {models.PlanRunning, models.PlanErrored, models.PlanCanceled},
	models.PlanRunning:  {models.PlanFinished, models.PlanErrored, models.PlanCanceled},
	models.PlanErrored:  {models.PlanPending},
	models.PlanCanceled: {models.PlanPending},
}

// applyTransitions lists the statuses each apply status may legally transition to.
// finished is absorbing; errored/canceled permit the retry reset back to pending.
// skipped permits the reset back to created: retrying the plan of an errored/canceled
// run returns the run to pending, and the apply must again await the plan's outcome.
var applyTransitions = map[models.ApplyStatus][]models.ApplyStatus{
	models.ApplyCreated:  {models.ApplyPending, models.ApplyCanceled, models.ApplySkipped},
	models.ApplyPending:  {models.ApplyQueued, models.ApplyCanceled},
	models.ApplyQueued:   {models.ApplyRunning, models.ApplyErrored, models.ApplyCanceled},
	models.ApplyRunning:  {models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled},
	models.ApplyErrored:  {models.ApplyPending},
	models.ApplyCanceled: {models.ApplyPending},
	models.ApplySkipped:  {models.ApplyCreated},
}

// canTransitionTo reports whether a node may move from its current status to next
// according to the given transition map.
func canTransitionTo[S comparable](transitions map[S][]S, from, next S) bool {
	for _, allowed := range transitions[from] {
		if allowed == next {
			return true
		}
	}
	return false
}
