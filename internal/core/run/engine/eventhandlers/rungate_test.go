package eventhandlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	gateTestRunID   = "run-1"
	gateTestCheckID = "check-1"
	gateTestUserID  = "user-1"
	gateTestTeamID  = "team-1"
)

// checkScopedGateQuery matches the gate lookup the handler makes for the check under test. Gate
// queries are scoped to the policy check, not the run, so a run that soft-fails at more than one
// stage gets a gate per check.
func checkScopedGateQuery(in *db.GetRunGatesInput) bool {
	return in.Filter != nil && len(in.Filter.PolicyCheckIDs) == 1 && in.Filter.PolicyCheckIDs[0] == gateTestCheckID
}

// awaitingOverrideRun builds a run whose post-plan check is parked at awaiting_override with the
// given pinned policies (which drive gate creation).
func awaitingOverrideRun(policies []*models.PolicyCheckPolicy) *models.Run {
	return &models.Run{
		Metadata:    models.ResourceMetadata{ID: gateTestRunID},
		WorkspaceID: "ws-1",
		TaskStages: []*models.RunTaskStage{{
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageAwaitingOverride,
			PolicyChecks: []*models.PolicyCheck{{
				ID:        gateTestCheckID,
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindOPA,
				Status:    models.PolicyCheckSoftFailed,
				Policies:  policies,
			}},
		}},
	}
}

// moduleAttestationAwaitingOverrideRun mirrors awaitingOverrideRun but for a module attestation
// check, so gate-creation/type-mapping tests can reuse the same policy fixtures.
func moduleAttestationAwaitingOverrideRun(policies []*models.PolicyCheckPolicy) *models.Run {
	return &models.Run{
		Metadata:    models.ResourceMetadata{ID: gateTestRunID},
		WorkspaceID: "ws-1",
		TaskStages: []*models.RunTaskStage{{
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageAwaitingOverride,
			PolicyChecks: []*models.PolicyCheck{{
				ID:        gateTestCheckID,
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindModuleAttestation,
				Status:    models.PolicyCheckSoftFailed,
				Policies:  policies,
			}},
		}},
	}
}

func awaitingOverrideChange(run *models.Run) []types.RunChange {
	return []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PolicyCheckStatusChange{
				OldStatus: models.PolicyCheckRunning,
				NewStatus: models.PolicyCheckSoftFailed,
				CheckID:   run.AllPolicyChecks()[0].ID,
			},
		},
	}}
}

// retriedCheckChange resets the run's check to pending from the given status, which is what retrying it
// does. Only errored, canceled and soft_failed can be the origin — pending is unreachable otherwise, so
// the handler treats the transition as a retry regardless of which one it came from.
func retriedCheckChange(run *models.Run, from models.PolicyCheckStatus) []types.RunChange {
	return []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PolicyCheckStatusChange{
				OldStatus: from,
				NewStatus: models.PolicyCheckPending,
				CheckID:   run.AllPolicyChecks()[0].ID,
			},
		},
	}}
}

// softFailedPolicy is a soft-mandatory policy that failed and declares approvers — the case that
// produces a gate.
func softFailedPolicy() *models.PolicyCheckPolicy {
	return &models.PolicyCheckPolicy{
		ID:                "policy-1",
		OPAData:           &models.PolicyCheckOPAData{PackageSource: "my-policy-set"},
		EnforcementLevel:  models.PolicyEnforcementSoftMandatory,
		Status:            models.PolicyCheckPolicyFailed,
		RequiredApprovals: 2,
		AllowedUserIDs:    []string{gateTestUserID},
		AllowedTeamIDs:    []string{gateTestTeamID},
	}
}

func TestRunGateManager_CreatesSingleGateWithRulePerPolicy(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	const secondUserID = "user-2"

	mockRunGates := db.NewMockRunGates(t)
	mockUsers := db.NewMockUsers(t)
	mockTeams := db.NewMockTeams(t)

	// No existing gates for the run.
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(func(in *db.GetRunGatesInput) bool {
		return checkScopedGateQuery(in)
	})).Return(&db.RunGatesResult{RunGates: []models.RunGate{}}, nil)

	mockUsers.On("GetUsers", mock.Anything, mock.MatchedBy(func(in *db.GetUsersInput) bool {
		return in.Filter != nil && len(in.Filter.UserIDs) == 1 && in.Filter.UserIDs[0] == gateTestUserID
	})).Return(&db.UsersResult{Users: []models.User{{
		Metadata: models.ResourceMetadata{ID: gateTestUserID, TRN: "trn:user:alice"},
		Username: "alice",
	}}}, nil)
	mockUsers.On("GetUsers", mock.Anything, mock.MatchedBy(func(in *db.GetUsersInput) bool {
		return in.Filter != nil && len(in.Filter.UserIDs) == 1 && in.Filter.UserIDs[0] == secondUserID
	})).Return(&db.UsersResult{Users: []models.User{{
		Metadata: models.ResourceMetadata{ID: secondUserID, TRN: "trn:user:bob"},
		Username: "bob",
	}}}, nil)
	mockTeams.On("GetTeams", mock.Anything, mock.MatchedBy(func(in *db.GetTeamsInput) bool {
		return in.Filter != nil && len(in.Filter.TeamIDs) == 1 && in.Filter.TeamIDs[0] == gateTestTeamID
	})).Return(&db.TeamsResult{Teams: []models.Team{{
		Metadata: models.ResourceMetadata{ID: gateTestTeamID, TRN: "trn:team:my-team"},
		Name:     "my-team",
	}}}, nil)

	// A run whose check soft-failed two policies produces exactly ONE gate with a rule per policy
	// and the deduped union of allowed subjects seeding the index tables.
	policies := []*models.PolicyCheckPolicy{
		softFailedPolicy(),
		{
			ID:                "policy-2",
			EnforcementLevel:  models.PolicyEnforcementSoftMandatory,
			Status:            models.PolicyCheckPolicyFailed,
			RequiredApprovals: 1,
			AllowedUserIDs:    []string{secondUserID},
		},
	}

	mockRunGates.On("CreateRunGate", mock.Anything, mock.MatchedBy(func(gate *models.RunGate) bool {
		if gate.RunID != gateTestRunID ||
			gate.Status != models.RunGatePending ||
			gate.Type != models.RunGateTypeOPAPolicy ||
			gate.WorkspaceID != "ws-1" {
			return false
		}
		if len(gate.ApprovalRules) != 2 {
			return false
		}
		byName := map[string]*models.RunGateApprovalRule{}
		for _, r := range gate.ApprovalRules {
			byName[r.Name] = r
		}
		r1, ok1 := byName["policy-1"]
		r2, ok2 := byName["policy-2"]
		if !ok1 || !ok2 {
			return false
		}
		// rule policy-1: required 2, allowed user alice + team my-team.
		if r1.RequiredApprovals != 2 || len(r1.AllowedSubjects) != 2 {
			return false
		}
		// rule policy-2: required 1, allowed user bob.
		if r2.RequiredApprovals != 1 || len(r2.AllowedSubjects) != 1 || r2.AllowedSubjects[0].TRN != "trn:user:bob" {
			return false
		}
		return true
	})).Return(&models.RunGate{}, nil).Once()

	dbClient := &db.Client{
		RunGates: mockRunGates,
		Users:    mockUsers,
		Teams:    mockTeams,
	}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx, awaitingOverrideChange(awaitingOverrideRun(policies))))

	mockRunGates.AssertExpectations(t)
	mockUsers.AssertExpectations(t)
	mockTeams.AssertExpectations(t)
}

func TestRunGateManager_IdempotentWhenGateExists(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	// A gate already exists for this check -> no new gate is created.
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{
			{
				Metadata:      models.ResourceMetadata{ID: "existing"},
				RunID:         gateTestRunID,
				PolicyCheckID: gateTestCheckID,
				Type:          models.RunGateTypeOPAPolicy,
				Status:        models.RunGatePending,
			},
		}}, nil)

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx, awaitingOverrideChange(awaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}))))

	mockRunGates.AssertNotCalled(t, "CreateRunGate", mock.Anything, mock.Anything)
}

// TestRunGateManager_CreatesGateWhenAnotherCheckHasOne covers a run that soft-fails at more than one
// stage: a gate belonging to a different check must not suppress this check's gate. The mock filters
// by PolicyCheckIDs the way the DB does, so a run-scoped lookup fails this test.
func TestRunGateManager_CreatesGateWhenAnotherCheckHasOne(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	mockUsers := db.NewMockUsers(t)
	mockTeams := db.NewMockTeams(t)

	// The run already carries an OPA gate, but it governs the other stage's check.
	otherCheckGate := models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-pre-plan"},
		RunID:         gateTestRunID,
		PolicyCheckID: "check-other",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
	}
	mockRunGates.On("GetRunGates", mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *db.GetRunGatesInput) (*db.RunGatesResult, error) {
			gates := []models.RunGate{}
			for _, checkID := range in.Filter.PolicyCheckIDs {
				if checkID == otherCheckGate.PolicyCheckID {
					gates = append(gates, otherCheckGate)
				}
			}
			return &db.RunGatesResult{RunGates: gates}, nil
		})

	mockUsers.On("GetUsers", mock.Anything, mock.MatchedBy(func(in *db.GetUsersInput) bool {
		return in.Filter != nil && len(in.Filter.UserIDs) == 1 && in.Filter.UserIDs[0] == gateTestUserID
	})).Return(&db.UsersResult{Users: []models.User{{
		Metadata: models.ResourceMetadata{ID: gateTestUserID, TRN: "trn:user:alice"},
		Username: "alice",
	}}}, nil)
	mockTeams.On("GetTeams", mock.Anything, mock.MatchedBy(func(in *db.GetTeamsInput) bool {
		return in.Filter != nil && len(in.Filter.TeamIDs) == 1 && in.Filter.TeamIDs[0] == gateTestTeamID
	})).Return(&db.TeamsResult{Teams: []models.Team{{
		Metadata: models.ResourceMetadata{ID: gateTestTeamID, TRN: "trn:team:my-team"},
		Name:     "my-team",
	}}}, nil)

	mockRunGates.On("CreateRunGate", mock.Anything, mock.MatchedBy(func(gate *models.RunGate) bool {
		return gate.PolicyCheckID == gateTestCheckID && gate.Status == models.RunGatePending
	})).Return(&models.RunGate{}, nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates, Users: mockUsers, Teams: mockTeams}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx, awaitingOverrideChange(awaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}))))

	mockRunGates.AssertExpectations(t)
}

// TestRunGateManager_DeletesGateWhenCheckRetried covers the retry path: a check returning to pending
// discards its gate so the re-evaluation can create a fresh one. Pending is only reachable from errored,
// canceled or soft_failed, so this transition is exactly a retry whichever it came from. The soft-failed
// origin is the one that actually has a gate to delete — and any approvals collected on it go too, since
// they approved a verdict that is being replaced.
func TestRunGateManager_DeletesGateWhenCheckRetried(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	for _, from := range []models.PolicyCheckStatus{
		models.PolicyCheckErrored,
		models.PolicyCheckCanceled,
		models.PolicyCheckSoftFailed,
	} {
		t.Run(string(from), func(t *testing.T) {
			existing := models.RunGate{
				Metadata:      models.ResourceMetadata{ID: "gate-stale", Version: 3},
				RunID:         gateTestRunID,
				PolicyCheckID: gateTestCheckID,
				Type:          models.RunGateTypeOPAPolicy,
				Status:        models.RunGatePending,
			}

			mockRunGates := db.NewMockRunGates(t)
			mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
				Return(&db.RunGatesResult{RunGates: []models.RunGate{existing}}, nil)
			mockRunGates.On("DeleteRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
				return g.Metadata.ID == existing.Metadata.ID && g.Metadata.Version == existing.Metadata.Version
			})).Return(nil).Once()

			dbClient := &db.Client{RunGates: mockRunGates}
			handler := NewRunGateManager(logr, dbClient)

			require.NoError(t, handler.HandleRunChanges(ctx,
				retriedCheckChange(awaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}), from)))

			mockRunGates.AssertExpectations(t)
			mockRunGates.AssertNotCalled(t, "CreateRunGate", mock.Anything, mock.Anything)
		})
	}
}

// TestRunGateManager_CreatesGateForModuleAttestationCheck verifies a soft-failed module attestation
// check produces a gate of type module_attestation, exercising runGateTypeForCheck's mapping through
// the full gate-creation path rather than in isolation.
func TestRunGateManager_CreatesGateForModuleAttestationCheck(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	mockUsers := db.NewMockUsers(t)
	mockTeams := db.NewMockTeams(t)

	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{}}, nil)
	mockUsers.On("GetUsers", mock.Anything, mock.MatchedBy(func(in *db.GetUsersInput) bool {
		return in.Filter != nil && len(in.Filter.UserIDs) == 1 && in.Filter.UserIDs[0] == gateTestUserID
	})).Return(&db.UsersResult{Users: []models.User{{
		Metadata: models.ResourceMetadata{ID: gateTestUserID, TRN: "trn:user:alice"},
		Username: "alice",
	}}}, nil)
	mockTeams.On("GetTeams", mock.Anything, mock.MatchedBy(func(in *db.GetTeamsInput) bool {
		return in.Filter != nil && len(in.Filter.TeamIDs) == 1 && in.Filter.TeamIDs[0] == gateTestTeamID
	})).Return(&db.TeamsResult{Teams: []models.Team{{
		Metadata: models.ResourceMetadata{ID: gateTestTeamID, TRN: "trn:team:my-team"},
		Name:     "my-team",
	}}}, nil)

	mockRunGates.On("CreateRunGate", mock.Anything, mock.MatchedBy(func(gate *models.RunGate) bool {
		return gate.RunID == gateTestRunID && gate.Type == models.RunGateTypeModuleAttestation
	})).Return(&models.RunGate{}, nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates, Users: mockUsers, Teams: mockTeams}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx,
		awaitingOverrideChange(moduleAttestationAwaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}))))

	mockRunGates.AssertExpectations(t)
}

// TestRunGateManager_DeletesGateWhenModuleAttestationCheckRetried mirrors
// TestRunGateManager_DeletesGateWhenCheckRetried for a module attestation check: the gate deletion on
// retry is keyed by policy check ID, not check kind, so it must behave identically.
func TestRunGateManager_DeletesGateWhenModuleAttestationCheckRetried(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	existing := models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-stale", Version: 3},
		RunID:         gateTestRunID,
		PolicyCheckID: gateTestCheckID,
		Type:          models.RunGateTypeModuleAttestation,
		Status:        models.RunGatePending,
	}

	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{existing}}, nil)
	mockRunGates.On("DeleteRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
		return g.Metadata.ID == existing.Metadata.ID && g.Metadata.Version == existing.Metadata.Version
	})).Return(nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx,
		retriedCheckChange(moduleAttestationAwaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}), models.PolicyCheckSoftFailed)))

	mockRunGates.AssertExpectations(t)
}

// TestRunGateManager_CancelsGateWhenCheckCanceled covers a check that is settled without its gate ever
// being decided: its stage errored, or the run was canceled/discarded under it. The gate must leave the
// approvers' queue, and canceling it (rather than deleting, which is the retry path) keeps it and its
// approvals as history.
func TestRunGateManager_CancelsGateWhenCheckCanceled(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	existing := models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-abandoned", Version: 2},
		RunID:         gateTestRunID,
		PolicyCheckID: gateTestCheckID,
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
	}

	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{existing}}, nil)
	mockRunGates.On("UpdateRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
		return g.Metadata.ID == existing.Metadata.ID && g.Status == models.RunGateCanceled
	})).Return(&models.RunGate{}, nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	run := awaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()})
	run.AllPolicyChecks()[0].Status = models.PolicyCheckCanceled
	require.NoError(t, handler.HandleRunChanges(ctx, []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PolicyCheckStatusChange{
				OldStatus: models.PolicyCheckSoftFailed,
				NewStatus: models.PolicyCheckCanceled,
				CheckID:   gateTestCheckID,
			},
		},
	}}))

	mockRunGates.AssertExpectations(t)
	mockRunGates.AssertNotCalled(t, "DeleteRunGate", mock.Anything, mock.Anything)
}

// TestRunGateManager_NoGateForCheckSoftFailedAndCanceledInSameBatch guards the leak a multi-check stage
// can produce: one check fails a hard gate while a sibling soft-fails, so the sibling's soft failure and
// its cancellation land in the SAME batch of changes as the run's terminal status. The changes arrive in
// tree order (run first), so the run-level sweep has already run by the time the soft failure is handled
// — a gate created then would sit pending in an approver's queue forever, for a canceled check on an
// errored run. The batch is produced by the real state machine rather than hand-written, so it stays
// honest about the order the handler actually sees.
func TestRunGateManager_NoGateForCheckSoftFailedAndCanceledInSameBatch(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: gateTestRunID},
		WorkspaceID: "ws-1",
		Status:      models.RunPostPlanRunning,
		Plan:        models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:       &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{{
			ID:        "stage-post",
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageRunning,
			PolicyChecks: []*models.PolicyCheck{
				{ID: "check-hard", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckRunning},
				{
					ID:        gateTestCheckID,
					StageName: models.RunTaskStageNamePostPlan,
					CheckType: models.PolicyKindModuleAttestation,
					Status:    models.PolicyCheckRunning,
					Policies:  []*models.PolicyCheckPolicy{softFailedPolicy()},
				},
			},
		}},
	}

	// The hard failure alone settles nothing: the stage waits for its sibling, so no gate and no
	// terminal run status yet.
	_, err := statemachine.SetPolicyCheckStatus(run, "post_plan.opa", models.PolicyCheckErrored)
	require.NoError(t, err)
	require.Equal(t, models.RunPostPlanRunning, run.Status)

	// The sibling soft-fails, which settles the stage: it errors on the hard failure and cancels the
	// soft-failed sibling, all in this one batch.
	changes, err := statemachine.SetPolicyCheckStatus(run, "post_plan.module_attestation", models.PolicyCheckSoftFailed)
	require.NoError(t, err)
	require.Equal(t, models.RunErrored, run.Status)

	// The batch carries the run's terminal status BEFORE the sibling's soft failure — the ordering that
	// makes this leak possible.
	var sawRunErrored bool
	var softFailedAfterRunErrored bool
	for _, c := range changes {
		switch sc := c.(type) {
		case statemachine.RunStatusChange:
			if sc.NewStatus == models.RunErrored {
				sawRunErrored = true
			}
		case statemachine.PolicyCheckStatusChange:
			if sc.NewStatus == models.PolicyCheckSoftFailed && sawRunErrored {
				softFailedAfterRunErrored = true
			}
		}
	}
	require.True(t, softFailedAfterRunErrored, "the batch must carry the soft failure after the run's terminal status")

	mockRunGates := db.NewMockRunGates(t)
	// Both cleanup sweeps find nothing: no gate was ever created for this run.
	mockRunGates.On("GetRunGates", mock.Anything, mock.Anything).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{}}, nil)

	// The approver lookups gate creation would perform are allowed but not required, so the only thing
	// that can keep CreateRunGate from being called is the handler refusing to open a gate for a check
	// whose block no longer stands. CreateRunGate is deliberately left unmocked: a call fails the test.
	mockUsers := db.NewMockUsers(t)
	mockUsers.On("GetUsers", mock.Anything, mock.Anything).
		Return(&db.UsersResult{Users: []models.User{{
			Metadata: models.ResourceMetadata{ID: gateTestUserID, TRN: "trn:user:alice"},
			Username: "alice",
		}}}, nil).Maybe()
	mockTeams := db.NewMockTeams(t)
	mockTeams.On("GetTeams", mock.Anything, mock.Anything).
		Return(&db.TeamsResult{Teams: []models.Team{{
			Metadata: models.ResourceMetadata{ID: gateTestTeamID, TRN: "trn:team:my-team"},
			Name:     "my-team",
		}}}, nil).Maybe()

	dbClient := &db.Client{RunGates: mockRunGates, Users: mockUsers, Teams: mockTeams}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx, []types.RunChange{{Run: run, NodeStatusChanges: changes}}))

	mockRunGates.AssertNotCalled(t, "CreateRunGate", mock.Anything, mock.Anything)
}

// TestRunGateTypeForCheck exhaustively covers the check-type-to-gate-type mapping: both supported
// policy kinds map to their own gate type, and anything else is rejected rather than defaulting.
func TestRunGateTypeForCheck(t *testing.T) {
	tests := []struct {
		checkType    models.PolicyKind
		wantGateType models.RunGateType
		expectErr    bool
	}{
		{checkType: models.PolicyKindOPA, wantGateType: models.RunGateTypeOPAPolicy},
		{checkType: models.PolicyKindModuleAttestation, wantGateType: models.RunGateTypeModuleAttestation},
		{checkType: models.PolicyKind("sentinel"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.checkType), func(t *testing.T) {
			gateType, err := runGateTypeForCheck(tt.checkType)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantGateType, gateType)
		})
	}
}

// TestRetryLoop_ClosesForModuleAttestationCheck proves the Task 4/Task 7 loop actually closes: a
// retry's pending transition both deletes the stale gate and (once the ungated stage restarts and
// re-queues the check in the same pass) causes PolicyCheckWorkItemEnqueuer to enqueue a fresh work
// item — the same NodeStatusChanges RetryRunNode would produce are consumed by both handlers.
func TestRetryLoop_ClosesForModuleAttestationCheck(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	run := moduleAttestationAwaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()})
	run.Status = models.RunPostPlanAwaitingDecision
	checkPath := run.AllPolicyChecks()[0].GetPath()

	// Mirrors what RetryRunNode does: set the check back to pending. Since the stage (post_plan) is
	// not workspace-gated, this cascades synchronously through the stage restarting and re-queuing the
	// check, so the returned changes include both the pending and the queued transition.
	changes, err := statemachine.SetPolicyCheckStatus(run, checkPath, models.PolicyCheckPending)
	require.NoError(t, err)

	var sawQueued bool
	for _, c := range changes {
		if pc, ok := c.(statemachine.PolicyCheckStatusChange); ok && pc.NewStatus == models.PolicyCheckQueued {
			sawQueued = true
		}
	}
	require.True(t, sawQueued, "an ungated stage must re-queue the check in the same pass as the retry")

	runChanges := []types.RunChange{{Run: run, NodeStatusChanges: changes}}

	existing := models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-stale", Version: 1},
		RunID:         gateTestRunID,
		PolicyCheckID: gateTestCheckID,
		Type:          models.RunGateTypeModuleAttestation,
		Status:        models.RunGatePending,
	}
	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{existing}}, nil)
	mockRunGates.On("DeleteRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
		return g.Metadata.ID == existing.Metadata.ID
	})).Return(nil).Once()

	gateHandler := NewRunGateManager(logr, &db.Client{RunGates: mockRunGates})
	require.NoError(t, gateHandler.HandleRunChanges(ctx, runChanges))
	mockRunGates.AssertExpectations(t)

	mockWIQ := db.NewMockWorkItemsQueue(t)
	mockWIQ.On("AddWorkItemToQueue", mock.Anything, mock.MatchedBy(func(in *db.AddWorkItemToQueueInput) bool {
		payload, ok := in.Payload.(*db.EvaluateRunPolicyCheckPayload)
		return in.Type == db.EvaluateRunPolicyCheckType &&
			ok && payload.RunID == gateTestRunID && payload.PolicyCheckID == gateTestCheckID
	})).Return(&db.WorkItem{}, nil).Once()

	enqueuer := NewPolicyCheckWorkItemEnqueuer(logr, &db.Client{WorkItemsQueue: mockWIQ})
	require.NoError(t, enqueuer.HandleRunChanges(ctx, runChanges))
	mockWIQ.AssertExpectations(t)
}

// TestRunGateManager_RetryWithoutGateIsNoOp covers the common retry: a check that never had a gate
// (nothing soft-failed, or its failures declared no approvers) has nothing to delete.
func TestRunGateManager_RetryWithoutGateIsNoOp(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(checkScopedGateQuery)).
		Return(&db.RunGatesResult{RunGates: []models.RunGate{}}, nil)

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx,
		retriedCheckChange(awaitingOverrideRun([]*models.PolicyCheckPolicy{softFailedPolicy()}), models.PolicyCheckCanceled)))

	mockRunGates.AssertNotCalled(t, "DeleteRunGate", mock.Anything, mock.Anything)
}

// TestRunGateManager_NoGateWhenNoApproversOrNotSoftFail verifies that policies which are not
// soft-mandatory failures with approvers do not produce a gate.
func TestRunGateManager_CreatesRuleLessGateWhenNoPolicyDeclaresApprovers(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(func(in *db.GetRunGatesInput) bool {
		return checkScopedGateQuery(in)
	})).Return(&db.RunGatesResult{RunGates: []models.RunGate{}}, nil)

	policies := []*models.PolicyCheckPolicy{
		// Soft-mandatory failure but no approvers -> no rule to approve, gate cleared by override.
		{ID: "p-noapprovers", EnforcementLevel: models.PolicyEnforcementSoftMandatory, Status: models.PolicyCheckPolicyFailed},
		// Soft-mandatory but passed -> no rule.
		{ID: "p-passed", EnforcementLevel: models.PolicyEnforcementSoftMandatory, Status: models.PolicyCheckPolicyPassed, RequiredApprovals: 1, AllowedUserIDs: []string{gateTestUserID}},
		// Hard-mandatory failure with approvers -> not overridable by approval, no rule.
		{ID: "p-hard", EnforcementLevel: models.PolicyEnforcementHardMandatory, Status: models.PolicyCheckPolicyFailed, RequiredApprovals: 1, AllowedUserIDs: []string{gateTestUserID}},
	}

	// The gate is still created: it is the block on the check, and overriding it is the only way to
	// clear one. With no rules it seeds no allowed-subject rows, so it reaches no approver's inbox.
	mockRunGates.On("CreateRunGate", mock.Anything, mock.MatchedBy(func(gate *models.RunGate) bool {
		return gate.RunID == gateTestRunID &&
			gate.PolicyCheckID == gateTestCheckID &&
			gate.Status == models.RunGatePending &&
			gate.Type == models.RunGateTypeOPAPolicy &&
			len(gate.ApprovalRules) == 0
	})).Return(&models.RunGate{}, nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	require.NoError(t, handler.HandleRunChanges(ctx, awaitingOverrideChange(awaitingOverrideRun(policies))))

	mockRunGates.AssertExpectations(t)
}

func TestRunGateManager_IgnoresNonAwaitingOverrideChange(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	// No mocks should be invoked for an unrelated status change.
	mockRunGates := db.NewMockRunGates(t)
	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: gateTestRunID},
		WorkspaceID: "ws-1",
		TaskStages: []*models.RunTaskStage{{
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageRunning,
			PolicyChecks: []*models.PolicyCheck{{
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindOPA,
				Status:    models.PolicyCheckRunning,
			}},
		}},
	}
	changes := []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PolicyCheckStatusChange{OldStatus: models.PolicyCheckQueued, NewStatus: models.PolicyCheckRunning},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(ctx, changes))
	mockRunGates.AssertNotCalled(t, "GetRunGates", mock.Anything, mock.Anything)
}

func TestRunGateManager_ClosesPendingGatesOnTerminalRunStatus(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	mockRunGates.On("GetRunGates", mock.Anything, mock.MatchedBy(func(in *db.GetRunGatesInput) bool {
		return in.Filter != nil && in.Filter.RunID != nil && *in.Filter.RunID == gateTestRunID
	})).Return(&db.RunGatesResult{RunGates: []models.RunGate{
		{Metadata: models.ResourceMetadata{ID: "gate-pending"}, RunID: gateTestRunID, Status: models.RunGatePending},
		{Metadata: models.ResourceMetadata{ID: "gate-approved"}, RunID: gateTestRunID, Status: models.RunGateApproved},
	}}, nil)

	// Only the pending gate is canceled; the approved gate is left untouched (no UpdateRunGate
	// expectation is registered for it, so a call would fail the mock).
	mockRunGates.On("UpdateRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
		return g.Metadata.ID == "gate-pending" && g.Status == models.RunGateCanceled
	})).Return(&models.RunGate{Metadata: models.ResourceMetadata{ID: "gate-pending"}, Status: models.RunGateCanceled}, nil).Once()

	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	run := &models.Run{Metadata: models.ResourceMetadata{ID: gateTestRunID}, Status: models.RunDiscarded}
	changes := []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.RunStatusChange{OldStatus: models.RunPostPlanAwaitingDecision, NewStatus: models.RunDiscarded},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(ctx, changes))
	mockRunGates.AssertExpectations(t)
}

func TestRunGateManager_IgnoresNonTerminalRunStatus(t *testing.T) {
	logr, _ := logger.NewForTest()
	ctx := context.Background()

	mockRunGates := db.NewMockRunGates(t)
	dbClient := &db.Client{RunGates: mockRunGates}
	handler := NewRunGateManager(logr, dbClient)

	run := &models.Run{Metadata: models.ResourceMetadata{ID: gateTestRunID}, Status: models.RunPostPlanRunning}
	changes := []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.RunStatusChange{OldStatus: models.RunPlanning, NewStatus: models.RunPostPlanRunning},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(ctx, changes))
	mockRunGates.AssertNotCalled(t, "GetRunGates", mock.Anything, mock.Anything)
}
