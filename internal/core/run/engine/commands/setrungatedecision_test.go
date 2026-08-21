package commands

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	approverID = "11111111-1111-1111-1111-111111111111"
	outsiderID = "22222222-2222-2222-2222-222222222222"
)

// eligibleUserCtx returns a context whose caller is a user with the given ID.
func eligibleUserCtx(dbClient *db.Client, userID string) context.Context {
	caller := auth.NewUserCaller(
		&models.User{Metadata: models.ResourceMetadata{ID: userID}, Email: "user@example.com"},
		nil,
		dbClient,
		nil,
		nil,
	)
	return auth.WithCaller(context.Background(), caller)
}

// awaitingOverrideRun returns a run parked at the post-plan decision point with one soft-failed
// policy check, the state every gate decision acts on.
func awaitingOverrideRun() *models.Run {
	return &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPostPlanAwaitingDecision,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{{
			ID:           "ts-post",
			StageName:    models.RunTaskStageNamePostPlan,
			Status:       models.RunTaskStageAwaitingOverride,
			PolicyChecks: []*models.PolicyCheck{{ID: "stage-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckSoftFailed}},
		}},
	}
}

// pendingGateWith returns a pending gate with a single approval rule allowing the given user.
func pendingGateWith(ruleName, userID string, requiredApprovals int) *models.RunGate {
	return &models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-1"},
		RunID:         "run-1",
		PolicyCheckID: "stage-1",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{
				Name:              ruleName,
				RequiredApprovals: requiredApprovals,
				AllowedSubjects: []*models.RunGateAllowedSubject{
					{ID: userID, Type: models.RunGateSubjectUser},
				},
			},
		},
	}
}

// --- Approve tests ---

func TestSetRunGateDecision_Approve_GateSatisfied(t *testing.T) {
	ctx := eligibleUserCtx(&db.Client{}, approverID)
	gate := pendingGateWith("p1", approverID, 1)
	run := awaitingOverrideRun()

	approvedGate := &models.RunGate{
		Metadata: models.ResourceMetadata{ID: "gate-1"},
		RunID:    "run-1",
		Status:   models.RunGateApproved,
	}

	mockGates := db.NewMockRunGates(t)
	mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)
	mockGates.On("UpdateRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
		// A gate cleared by collecting its approvals was overridden by nobody — that distinction is
		// what the run screen keys on to tell an approved gate from a bypassed one.
		return g.Status == models.RunGateApproved && g.OverriddenBy == nil && g.OverrideComment == nil
	})).Return(approvedGate, nil)

	mockApprovals := db.NewMockRunGateApprovals(t)
	mockApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return([]models.RunGateApproval{}, nil).Once()
	mockApprovals.On("CreateRunGateApproval", mock.Anything, mock.MatchedBy(func(a *models.RunGateApproval) bool {
		return a.Decision == models.RunGateDecisionApprove && contains(a.CoveredRules, "p1")
	})).Return(&models.RunGateApproval{}, nil)
	mockApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return([]models.RunGateApproval{
		{Decision: models.RunGateDecisionApprove, CoveredRules: []string{"p1"}},
	}, nil)

	dbClient := &db.Client{RunGates: mockGates, RunGateApprovals: mockApprovals}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
		GateID:   "gate-1",
		Decision: GateDecisionApprove,
		Comment:  ptr.String("lgtm"),
		Caller:   callerFromCtx(ctx),
	}}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	require.NotNil(t, cmd.UpdatedGate)
	assert.Equal(t, models.RunGateApproved, cmd.UpdatedGate.Status)
	assert.Equal(t, models.PolicyCheckOverridden, run.AllPolicyChecks()[0].Status)
	assert.Equal(t, models.RunPlanned, run.Status)
}

func TestSetRunGateDecision_Approve_GateNotSatisfied(t *testing.T) {
	ctx := eligibleUserCtx(&db.Client{}, approverID)
	gate := pendingGateWith("p1", approverID, 2) // needs 2 approvals
	run := awaitingOverrideRun()

	mockGates := db.NewMockRunGates(t)
	mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)

	mockApprovals := db.NewMockRunGateApprovals(t)
	mockApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return([]models.RunGateApproval{}, nil).Once()
	mockApprovals.On("CreateRunGateApproval", mock.Anything, mock.Anything).Return(&models.RunGateApproval{}, nil)
	mockApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return([]models.RunGateApproval{
		{Decision: models.RunGateDecisionApprove, CoveredRules: []string{"p1"}},
	}, nil)

	dbClient := &db.Client{RunGates: mockGates, RunGateApprovals: mockApprovals}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
		GateID: "gate-1", Decision: GateDecisionApprove, Caller: callerFromCtx(ctx),
	}}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	require.NotNil(t, cmd.UpdatedGate)
	assert.Equal(t, models.RunGatePending, cmd.UpdatedGate.Status)
	assert.Equal(t, models.PolicyCheckSoftFailed, run.AllPolicyChecks()[0].Status)
}

func TestSetRunGateDecision_Approve_IneligibleCaller(t *testing.T) {
	ctx := eligibleUserCtx(&db.Client{}, outsiderID) // outsider not in allowed subjects
	gate := pendingGateWith("p1", approverID, 1)
	run := awaitingOverrideRun()

	mockGates := db.NewMockRunGates(t)
	mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)

	dbClient := &db.Client{RunGates: mockGates}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
		GateID: "gate-1", Decision: GateDecisionApprove, Caller: callerFromCtx(ctx),
	}}
	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
}

// --- Reject tests ---

func TestSetRunGateDecision_Reject_RecordedGateStaysPending(t *testing.T) {
	ctx := eligibleUserCtx(&db.Client{}, approverID)
	gate := pendingGateWith("p1", approverID, 1)
	run := awaitingOverrideRun()

	mockGates := db.NewMockRunGates(t)
	mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)

	mockApprovals := db.NewMockRunGateApprovals(t)
	mockApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return([]models.RunGateApproval{}, nil)
	mockApprovals.On("CreateRunGateApproval", mock.Anything, mock.MatchedBy(func(a *models.RunGateApproval) bool {
		return a.Decision == models.RunGateDecisionReject
	})).Return(&models.RunGateApproval{}, nil)

	dbClient := &db.Client{RunGates: mockGates, RunGateApprovals: mockApprovals}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
		GateID: "gate-1", Decision: GateDecisionReject, Comment: ptr.String("nope"), Caller: callerFromCtx(ctx),
	}}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	require.NotNil(t, cmd.UpdatedGate)
	assert.Equal(t, models.RunGatePending, cmd.UpdatedGate.Status)
	assert.Equal(t, models.PolicyCheckSoftFailed, run.AllPolicyChecks()[0].Status)
}

func TestSetRunGateDecision_Reject_IneligibleCaller(t *testing.T) {
	ctx := eligibleUserCtx(&db.Client{}, outsiderID)
	gate := pendingGateWith("p1", approverID, 1)
	run := awaitingOverrideRun()

	mockGates := db.NewMockRunGates(t)
	mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)

	dbClient := &db.Client{RunGates: mockGates}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
		GateID: "gate-1", Decision: GateDecisionReject, Caller: callerFromCtx(ctx),
	}}
	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
}

// --- Override tests ---

// TestSetRunGateDecision_Override verifies the command sets the gate to overridden, records who
// overrode it and why, and advances the blocked policy check. An override writes no approval row, so
// the gate is the only durable trace of the decision. Authorization is the caller's responsibility
// and happens before the command is dispatched; the command does not check permissions.
func TestSetRunGateDecision_Override(t *testing.T) {
	tests := []struct {
		name        string
		comment     *string
		wantComment *string
	}{
		{
			name:        "with a reason",
			comment:     ptr.String("hotfix window"),
			wantComment: ptr.String("hotfix window"),
		},
		{
			name:        "without a reason",
			comment:     nil,
			wantComment: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := &models.RunGate{
				Metadata:      models.ResourceMetadata{ID: "gate-1"},
				RunID:         "run-1",
				Type:          models.RunGateTypeOPAPolicy,
				PolicyCheckID: "stage-1",
				Status:        models.RunGatePending,
			}
			run := awaitingOverrideRun()

			overriddenGate := &models.RunGate{
				Metadata:        models.ResourceMetadata{ID: "gate-1"},
				Status:          models.RunGateOverridden,
				OverriddenBy:    ptr.String("admin@example.com"),
				OverrideComment: tt.wantComment,
			}

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return("admin@example.com")

			mockGates := db.NewMockRunGates(t)
			mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)
			mockGates.On("UpdateRunGate", mock.Anything, mock.MatchedBy(func(g *models.RunGate) bool {
				return g.Status == models.RunGateOverridden &&
					g.OverriddenBy != nil && *g.OverriddenBy == "admin@example.com" &&
					assert.ObjectsAreEqual(tt.wantComment, g.OverrideComment)
			})).Return(overriddenGate, nil)

			dbClient := &db.Client{RunGates: mockGates}
			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(run)

			cmd := &SetRunGateDecision{dbClient: dbClient, in: &SetRunGateDecisionInput{
				GateID: "gate-1", Decision: GateDecisionOverride, Comment: tt.comment, Caller: mockCaller,
			}}
			require.NoError(t, cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: runStore}))

			require.NotNil(t, cmd.UpdatedGate)
			assert.Equal(t, models.RunGateOverridden, cmd.UpdatedGate.Status)
			require.NotNil(t, cmd.UpdatedGate.OverriddenBy)
			assert.Equal(t, "admin@example.com", *cmd.UpdatedGate.OverriddenBy)
			assert.Equal(t, tt.wantComment, cmd.UpdatedGate.OverrideComment)
			assert.Equal(t, models.PolicyCheckOverridden, run.AllPolicyChecks()[0].Status)
		})
	}
}

// --- Gate not pending ---

func TestSetRunGateDecision_NotPending(t *testing.T) {
	for _, decision := range []GateDecision{GateDecisionApprove, GateDecisionReject, GateDecisionOverride} {
		t.Run(string(decision), func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)

			gate := &models.RunGate{
				Metadata: models.ResourceMetadata{ID: "gate-1"},
				Status:   models.RunGateApproved,
			}

			mockGates := db.NewMockRunGates(t)
			mockGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(gate, nil)

			cmd := &SetRunGateDecision{
				dbClient: &db.Client{RunGates: mockGates},
				in: &SetRunGateDecisionInput{
					GateID: "gate-1", Decision: decision, Caller: mockCaller,
				},
			}
			err := cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: store.NewRunStore(&db.Client{})})
			require.Error(t, err)
			assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
		})
	}
}

func callerFromCtx(ctx context.Context) auth.Caller {
	caller, _ := auth.AuthorizeCaller(ctx)
	return caller
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
