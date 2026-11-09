package run

import (
	"context"
	"fmt"
	"strings"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// taskStage follows one of a run's task stages (pre-plan, post-plan, pre-apply or post-apply). It
// reports each policy check's findings once per evaluation, and asks for a decision when a
// soft-failed check blocks the run on its gate.
//
// A stage can hold work this package doesn't know how to show: a check of a newer policy type, or
// other kinds of tasks that aren't visible at all. That work is reported rather than shown, and the
// stage waits for the run to move on past it.
type taskStage struct {
	f         *follower
	stageName pb.RunTaskStageName
	stage     StageRef

	announced bool
	// retried is set once the stage has re-run, so its start is reported as a retry.
	retried    bool
	lastStatus pb.RunTaskStageStatus
	// settledBeforeFollow is set when the stage had already completed when following began and
	// SkipFinished is set; it isn't shown unless it runs again.
	settledBeforeFollow bool
	// results holds the latest results reported for each policy check, by check ID. A check is
	// reported once per evaluation; the entry is dropped when the check is re-run.
	results map[string]*pb.PolicyCheckResults
	// reportedJobIDs holds, for each check whose results were reported, the job those results came
	// from. A check with a different job has been re-run since, however long ago this stage was last
	// synced.
	reportedJobIDs map[string]string
	// approved and denied hold the gates this caller approved, or wasn't allowed to act on.
	approved map[string]bool
	denied   map[string]bool
	// checkOrder is the stage's check IDs in run order, so gates are always listed the same way.
	checkOrder []string
	// unsupported holds the checks reported as being of a type this package can't show.
	unsupported map[string]bool
	// reportedNoTasks is set once the stage has been reported as having no tasks that can be shown.
	reportedNoTasks bool
	// reportedChecks is set once a check of the current evaluation has been reported.
	reportedChecks bool
	// deferred is set once the decision on the gates was left to be made elsewhere.
	deferred bool
}

func newTaskStage(f *follower, stageName pb.RunTaskStageName) *taskStage {
	return &taskStage{
		f:              f,
		stageName:      stageName,
		stage:          StageRef{Name: TaskStageTitle(stageName), Kind: StageTask, TaskStage: stageName},
		results:        map[string]*pb.PolicyCheckResults{},
		reportedJobIDs: map[string]string{},
		approved:       map[string]bool{},
		denied:         map[string]bool{},
		unsupported:    map[string]bool{},
	}
}

func (s *taskStage) ref() StageRef {
	return s.stage
}

func (s *taskStage) sync(ctx context.Context, run *pb.Run) (stageStatus, error) {
	stage := findTaskStage(run, s.stageName)
	if stage == nil {
		return stageDone, nil
	}

	if s.isRestarted(stage) {
		// Retried or undiscarded: the stage is evaluating again, so show it again.
		s.reset()
		s.retried = true
	}
	s.lastStatus = stage.Status
	s.checkOrder = s.checkOrder[:0]
	for _, check := range stage.PolicyChecks {
		s.checkOrder = append(s.checkOrder, check.Id)
	}

	switch stage.Status {
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_SKIPPED:
		return stageDone, nil
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_UNSPECIFIED,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED:
		s.f.waiting(s.stage, waitReason(run.Status), "")
		return stageWaiting, nil
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED:
		if s.f.shouldSkipFinished() && !s.announced {
			s.settledBeforeFollow = true
		}
		if s.settledBeforeFollow {
			return stageDone, nil
		}
	}

	// The stage has started.
	if !s.announced {
		s.announced = true
		s.f.emit(StageStarted{Stage: s.stage, Retry: s.retried})
	}
	if stage.Status == pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING {
		s.f.waiting(s.stage, WaitWorkspace, "")
		return stageWaiting, nil
	}

	if len(stage.PolicyChecks) == 0 && !s.reportedNoTasks {
		// Its tasks are of kinds that aren't visible.
		s.reportedNoTasks = true
		s.f.emit(NoVisibleTasks{Stage: s.stage})
	}
	for _, check := range stage.PolicyChecks {
		if err := s.syncCheck(ctx, check); err != nil {
			return stageWaiting, err
		}
	}

	switch stage.Status {
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED:
		return stageDone, nil
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_ERRORED,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CANCELED:
		return stageFailed, nil
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE:
		return s.syncBlocked(ctx)
	default:
		s.f.waiting(s.stage, waitReason(run.Status), "")
		return stageWaiting, nil
	}
}

// syncBlocked handles a stage blocked on its gates: it asks for a decision on any the caller may
// still act on, or waits for them to be decided elsewhere.
func (s *taskStage) syncBlocked(ctx context.Context) (stageStatus, error) {
	if s.deferred || s.f.decider == nil || len(s.approved) > 0 {
		// Keep the gates current, so the waiting line follows the approvals as they come in.
		// That includes the caller's own, once the stage is only waiting on other approvers.
		if err := s.refreshResults(ctx); err != nil {
			return stageWaiting, err
		}
	}

	pending := s.pendingGates()
	if len(pending) == 0 {
		// Blocked on something that can't be shown or acted on, such as an unsupported check. The
		// decider is still asked, so it can choose to leave the run waiting rather than wait here.
		if s.f.decider != nil && !s.deferred && s.blockedOnUnshowable() {
			return stageNeedsDecision, nil
		}
		s.f.waiting(s.stage, WaitUnsupportedTask, "")
		return stageWaitingOnDecision, nil
	}

	if s.f.decider == nil && !s.deferred {
		s.deferred = true
		s.f.emit(GatesPending{Stage: s.stage, Gates: s.gateInfos(pending)})
		s.f.emit(DecisionDeferred{Stage: s.stage})
	}
	if s.deferred {
		s.f.waiting(s.stage, WaitDecisionElsewhere, s.pendingSummary())
		return stageWaitingOnDecision, nil
	}
	if s.onlyWaitingOnOthers() {
		s.f.waiting(s.stage, WaitOtherApprovers, s.pendingSummary())
		return stageWaitingOnDecision, nil
	}
	return stageNeedsDecision, nil
}

// syncCheck reports a check's findings once it has a verdict. Its job's logs aren't streamed: the
// findings already say everything the logs would.
func (s *taskStage) syncCheck(ctx context.Context, check *pb.PolicyCheck) error {
	if !isSupportedCheckType(check.CheckType) {
		// A newer policy type: its results may not have a shape this package understands, so don't
		// fetch them. Report it once and wait for the run to move on.
		if !s.unsupported[check.Id] {
			s.unsupported[check.Id] = true
			s.f.emit(UnsupportedPolicyCheck{Stage: s.stage, CheckType: check.CheckType, First: s.firstCheck()})
		}
		return nil
	}

	if !isCheckFinal(check.Status) {
		// Not evaluated yet, or re-running: its findings (if any) are from an earlier evaluation.
		delete(s.results, check.Id)
		return nil
	}

	if s.results[check.Id] == nil {
		results, err := s.getResults(ctx, check.Id)
		if err != nil {
			return err
		}
		s.results[check.Id] = results
		s.reportedJobIDs[check.Id] = check.GetLatestJobId()
		s.f.recordAdvisories(check.Id, AdvisoryFailures(results))
		s.f.emit(PolicyCheckFinished{Stage: s.stage, Results: results, First: s.firstCheck()})
	}

	return nil
}

// firstCheck reports whether no check of this evaluation has been reported yet, and notes that one
// now has.
func (s *taskStage) firstCheck() bool {
	first := !s.reportedChecks
	s.reportedChecks = true
	return first
}

func (s *taskStage) decide(ctx context.Context, run *pb.Run) (decisionResult, error) {
	// Refresh the gates: approvals may have come in since the findings were reported.
	if err := s.refreshResults(ctx); err != nil {
		return decisionActed, err
	}

	pending := s.pendingGates()
	if len(pending) == 0 && !s.blockedOnUnshowable() {
		// The gates cleared since the findings were reported. Re-read the run; sync waits if the stage
		// is still blocked.
		return decisionActed, nil
	}
	if len(pending) > 0 {
		s.f.emit(GatesPending{Stage: s.stage, Gates: s.gateInfos(pending)})
	}

	stillBlocked := func(latest *pb.Run) bool {
		stage := findTaskStage(latest, s.stageName)
		return stage != nil && stage.Status == pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE
	}

	for {
		var approvable, overridable []*pb.RunGate
		for _, g := range s.undecidedGates() {
			if NeedsOverride(g) {
				overridable = append(overridable, g)
			} else {
				approvable = append(approvable, g)
			}
		}
		req := &GateRequest{
			Run:         run,
			Stage:       s.stage,
			Pending:     s.gateInfos(s.pendingGates()),
			CanApprove:  len(approvable) > 0,
			CanOverride: len(overridable) > 0,
		}

		decision, how, err := decide(ctx, s.f, run, stillBlocked,
			func(ctx context.Context) (GateDecision, error) { return s.f.decider.DecideGates(ctx, req) })
		if err != nil {
			return decisionActed, err
		}
		switch how {
		case abandoned:
			s.f.emit(DecisionAbandoned{Stage: s.stage})
			return decisionActed, nil
		case timedOut:
			return decisionTimedOut, nil
		}

		switch decision.Action {
		case GateApprove:
			if len(approvable) == 0 {
				return decisionActed, fmt.Errorf("the decider chose to approve, but no gate can be approved")
			}
			acted, err := s.approve(ctx, approvable)
			if err != nil || acted {
				return decisionActed, err
			}
			// Not allowed to approve any of them: ask again without them.
		case GateOverride:
			if len(overridable) == 0 {
				return decisionActed, fmt.Errorf("the decider chose to override, but no gate can be overridden")
			}
			acted, err := s.override(ctx, overridable)
			if err != nil || acted {
				return decisionActed, err
			}
			// Not allowed to override any of them: ask again without them.
		case GateDiscard:
			if _, err := s.f.client.RunsClient.DiscardRun(ctx, &pb.DiscardRunRequest{RunId: run.Metadata.Id}); err != nil {
				if isStateConflict(err) {
					s.f.emit(ActionConflict{Stage: s.stage, Action: "discard"})
					return decisionActed, nil
				}
				return decisionActed, fmt.Errorf("failed to discard run: %w", err)
			}
			return decisionDiscarded, nil
		case GateLeave:
			s.f.leaveChosen, s.f.leaveReason = decision.Chosen, decision.Reason
			return decisionDetach, nil
		case GateWaitElsewhere:
			s.deferred = true
			s.f.emit(DecisionDeferred{Stage: s.stage, Reason: decision.Reason})
			return decisionActed, nil
		default:
			return decisionActed, fmt.Errorf("unknown gate decision %d", decision.Action)
		}
	}
}

// blockedOnUnshowable reports whether the stage holds work that can't be shown, and so may be what
// it is blocked on: a check of an unsupported type, or no visible checks at all.
func (s *taskStage) blockedOnUnshowable() bool {
	if len(s.checkOrder) == 0 {
		return true
	}
	for _, checkID := range s.checkOrder {
		if s.unsupported[checkID] {
			return true
		}
	}
	return false
}

// gateInfos pairs each gate with its check's results.
func (s *taskStage) gateInfos(gates []*pb.RunGate) []GateInfo {
	infos := make([]GateInfo, 0, len(gates))
	for _, g := range gates {
		infos = append(infos, GateInfo{Gate: g, Results: s.results[g.GetPolicyCheckId()], NeedsOverride: NeedsOverride(g)})
	}
	return infos
}

// refreshResults re-reads the results of each gated check reported so far, so its gate is current.
func (s *taskStage) refreshResults(ctx context.Context) error {
	for checkID, cached := range s.results {
		if cached.GetRunGate() == nil {
			continue
		}
		results, err := s.getResults(ctx, checkID)
		if err != nil {
			return err
		}
		s.results[checkID] = results
	}
	return nil
}

// approve approves each gate, reporting whether any approval was recorded (or the gate had already
// moved on). Gates the caller may not approve are remembered so they aren't offered again.
func (s *taskStage) approve(ctx context.Context, gates []*pb.RunGate) (bool, error) {
	acted := false
	for _, g := range gates {
		updated, err := s.f.client.RunGatesClient.ApproveRunGate(ctx, &pb.ApproveRunGateRequest{
			GateId:   g.GetMetadata().GetId(),
			Decision: pb.RunGateDecision_RUN_GATE_DECISION_APPROVE,
		})
		switch {
		case err == nil:
			acted = true
			s.approved[g.GetMetadata().GetId()] = true
			if results := s.results[g.GetPolicyCheckId()]; results != nil {
				results.RunGate = updated
			}
			s.f.emit(GateApproved{
				Stage:          s.stage,
				Gate:           updated,
				AllApprovalsIn: updated.GetStatus() != pb.RunGateStatus_RUN_GATE_STATUS_PENDING,
			})
		case isPermissionDenied(err):
			s.denied[g.GetMetadata().GetId()] = true
			s.f.emit(GateDenied{Stage: s.stage, Gate: g, Action: GateApprove})
		case isStateConflict(err):
			// Cleared or canceled in the meantime; re-read the run.
			acted = true
		default:
			return false, fmt.Errorf("failed to approve run gate: %w", err)
		}
	}
	return acted, nil
}

// override overrides each gate, reporting whether any override was recorded (or the gate had already
// moved on). Gates the caller may not override are remembered so they aren't offered again.
func (s *taskStage) override(ctx context.Context, gates []*pb.RunGate) (bool, error) {
	acted := false
	for _, g := range gates {
		updated, err := s.f.client.RunGatesClient.OverrideRunGate(ctx, &pb.OverrideRunGateRequest{
			GateId: g.GetMetadata().GetId(),
		})
		switch {
		case err == nil:
			acted = true
			if results := s.results[g.GetPolicyCheckId()]; results != nil {
				results.RunGate = updated
			}
			s.f.emit(GateOverridden{Stage: s.stage, Gate: updated})
		case isPermissionDenied(err):
			s.denied[g.GetMetadata().GetId()] = true
			s.f.emit(GateDenied{Stage: s.stage, Gate: g, Action: GateOverride})
		case isStateConflict(err):
			// Cleared or canceled in the meantime; re-read the run.
			acted = true
		default:
			return false, fmt.Errorf("failed to override run gate: %w", err)
		}
	}
	return acted, nil
}

// pendingSummary describes the approvals the pending gates still need, for the waiting line.
func (s *taskStage) pendingSummary() string {
	parts := []string{}
	for _, g := range s.pendingGates() {
		if summary := GateSummary(g, s.results[g.GetPolicyCheckId()]); summary != "" {
			parts = append(parts, summary)
		}
	}
	return strings.Join(parts, "; ")
}

// pendingGates returns the gates, among the checks reported so far, that are still pending.
func (s *taskStage) pendingGates() []*pb.RunGate {
	gates := []*pb.RunGate{}
	for _, checkID := range s.checkOrder {
		results := s.results[checkID]
		if results == nil {
			continue
		}
		if g := results.RunGate; g != nil && g.Status == pb.RunGateStatus_RUN_GATE_STATUS_PENDING {
			gates = append(gates, g)
		}
	}
	return gates
}

// onlyWaitingOnOthers reports whether this caller has approved every pending gate, so the stage is
// only waiting on other approvers.
func (s *taskStage) onlyWaitingOnOthers() bool {
	if len(s.approved) == 0 {
		return false
	}
	for _, g := range s.pendingGates() {
		if !s.approved[g.GetMetadata().GetId()] {
			return false
		}
	}
	return true
}

// undecidedGates returns the pending gates this caller hasn't approved and may still act on.
func (s *taskStage) undecidedGates() []*pb.RunGate {
	gates := []*pb.RunGate{}
	for _, g := range s.pendingGates() {
		if id := g.GetMetadata().GetId(); !s.approved[id] && !s.denied[id] {
			gates = append(gates, g)
		}
	}
	return gates
}

func (s *taskStage) getResults(ctx context.Context, checkID string) (*pb.PolicyCheckResults, error) {
	results, err := retryRead(ctx, s.f, func(ctx context.Context) (*pb.PolicyCheckResults, error) {
		return s.f.client.RunsClient.GetPolicyCheckResults(ctx, &pb.GetPolicyCheckResultsRequest{PolicyCheckId: checkID})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get policy check results: %w", err)
	}
	return results, nil
}

func (s *taskStage) reset() {
	s.announced = false
	s.settledBeforeFollow = false
	s.results = map[string]*pb.PolicyCheckResults{}
	s.reportedJobIDs = map[string]string{}
	s.approved = map[string]bool{}
	s.denied = map[string]bool{}
	s.unsupported = map[string]bool{}
	s.reportedNoTasks = false
	s.reportedChecks = false
	s.deferred = false
}

// isRestarted reports whether the stage has evaluated again since it was last shown: it went back
// to evaluating after settling, or one of its checks has a different job than the one its reported
// results came from. The second holds even if the stage re-ran and settled again between two syncs,
// as it can while an earlier stage streams logs or a decision is being made.
func (s *taskStage) isRestarted(stage *pb.RunTaskStage) bool {
	if isStageRestarted(s.lastStatus, stage.Status) {
		return true
	}
	for _, check := range stage.PolicyChecks {
		if reported, ok := s.reportedJobIDs[check.Id]; ok && reported != check.GetLatestJobId() {
			return true
		}
	}
	return false
}

// isStageRestarted reports whether a stage went back to evaluating after it had settled.
func isStageRestarted(prev, cur pb.RunTaskStageStatus) bool {
	switch prev {
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_ERRORED,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CANCELED:
	default:
		return false
	}
	switch cur {
	case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING,
		pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING:
		return true
	default:
		return false
	}
}

// isSupportedCheckType reports whether this package knows how to show a check's results. Types
// added to the API after this package was built aren't, and are waited on without being shown.
func isSupportedCheckType(t pb.PolicyCheckType) bool {
	switch t {
	case pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA,
		pb.PolicyCheckType_POLICY_CHECK_TYPE_MODULE_ATTESTATION:
		return true
	default:
		return false
	}
}

// isCheckFinal reports whether a check has a verdict to show.
func isCheckFinal(s pb.PolicyCheckStatus) bool {
	switch s {
	case pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_OVERRIDDEN,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_ERRORED:
		return true
	default:
		return false
	}
}
