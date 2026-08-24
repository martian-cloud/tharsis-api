package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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

func TestRetryRunNode_Prepare(t *testing.T) {
	bg := context.Background()

	t.Run("resolves the workspace namespace path", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", bg, "ws-1").Return(&models.Workspace{FullPath: "groupA/ws"}, nil)

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &RetryRunNodeInput{RunID: "run-1", NodePath: "plan"},
		}

		require.NoError(t, cmd.Prepare(bg))
		assert.Equal(t, "groupA/ws", cmd.namespacePath)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").Return(nil, nil)

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &RetryRunNodeInput{RunID: "run-1"},
		}

		err := cmd.Prepare(bg)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("missing workspace yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", bg, "ws-1").Return(nil, nil)

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &RetryRunNodeInput{RunID: "run-1"},
		}

		err := cmd.Prepare(bg)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestRetryRunNode_Execute(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))
	errMsg := "boom"
	forceCanceledBy := "user-1"
	now := time.Now().UTC()

	// evaluatedPolicies is the per-policy state the evaluator leaves behind on a check that reported a
	// failure: one policy passed, one failed with messages in object storage. A retry throws that
	// verdict away, so every success case asserts these came back to pending with no messages. The
	// advisory failure is what makes the run's HasAdvisoryFailures flag true going in, so the assertion
	// that a retry clears it is not vacuous.
	messagesKey := "workspaces/ws-1/runs/run-1/policy_messages/obj-1.json"
	evaluatedPolicies := func() []*models.PolicyCheckPolicy {
		return []*models.PolicyCheckPolicy{
			{ID: "policy-ok", Status: models.PolicyCheckPolicyPassed},
			{
				ID:                     "policy-bad",
				Status:                 models.PolicyCheckPolicyFailed,
				MessagesObjectStoreKey: &messagesKey,
				EnforcementLevel:       models.PolicyEnforcementSoftMandatory,
				RequiredApprovals:      1,
			},
			{
				ID:               "policy-advisory",
				Status:           models.PolicyCheckPolicyFailed,
				EnforcementLevel: models.PolicyEnforcementAdvisory,
			},
		}
	}

	// postPlanRun builds a run whose plan finished with changes and whose post-plan policy check
	// settled in checkStatus, with the stage and run statuses that accompany it. The apply is
	// skipped, as it is whenever a post-plan check does not clear.
	postPlanRun := func(
		runStatus models.RunStatus,
		stageStatus models.RunTaskStageStatus,
		checkStatus models.PolicyCheckStatus,
		applyStatus models.ApplyStatus,
		policies []*models.PolicyCheckPolicy,
	) *models.Run {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   runStatus,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: applyStatus},
			TaskStages: []*models.RunTaskStage{{
				ID:        "stage-post",
				StageName: models.RunTaskStageNamePostPlan,
				Status:    stageStatus,
				PolicyChecks: []*models.PolicyCheck{{
					ID:        "check-1",
					StageName: models.RunTaskStageNamePostPlan,
					CheckType: models.PolicyKindOPA,
					Status:    checkStatus,
					Policies:  policies,
					// Set on every check so the "cleared on retry" assertion is not vacuous. A check
					// that never evaluated has none, but no case here starts from one.
					MessagesSummary: &models.PolicyCheckMessagesSummary{Messages: []string{errMsg}},
				}},
			}},
		}
		// Derived rather than hard-coded so the fixture stays consistent with whatever policies the case
		// passes in — the "retry a passed check" case has no failures and must not claim any.
		run.HasAdvisoryFailures = run.ComputeHasAdvisoryFailures()
		return run
	}

	tests := []struct {
		name      string
		run       *models.Run
		nodePath  string
		wantCode  errors.CodeType
		wantRun   models.RunStatus
		wantNode  string                    // status to assert on the retried node
		wantStage models.RunTaskStageStatus // asserted only when set
	}{
		{
			name: "retry failed plan resets plan to pending and run to plan_queuing",
			run: &models.Run{
				Metadata:               models.ResourceMetadata{ID: "run-1"},
				Status:                 models.RunErrored,
				Plan:                   models.Plan{ID: "plan-1", Status: models.PlanErrored, ErrorMessage: &errMsg},
				Apply:                  &models.Apply{ID: "apply-1", Status: models.ApplySkipped},
				ForceCanceled:          true,
				ForceCanceledBy:        &forceCanceledBy,
				ForceCancelAvailableAt: &now,
			},
			nodePath: "plan",
			wantRun:  models.RunPlanQueuing,
			wantNode: string(models.PlanPending),
		},
		{
			name: "retry canceled apply resets apply to pending and run to apply_queuing",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunCanceled,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCanceled, ErrorMessage: &errMsg},
			},
			nodePath: "apply",
			wantRun:  models.RunApplyQueuing,
			wantNode: string(models.ApplyPending),
		},
		{
			name: "retry a finished plan yields conflict",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanned,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
			},
			nodePath: "plan",
			wantCode: errors.EConflict,
		},
		{
			name: "invalid node path yields invalid error",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunErrored,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanErrored},
			},
			nodePath: "bogus",
			wantCode: errors.EInvalid,
		},
		{
			name: "retry apply on a speculative run yields invalid error",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunErrored,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanErrored},
			},
			nodePath: "apply",
			wantCode: errors.EInvalid,
		},
		{
			// The post-plan stage is not workspace-gated, so it restarts straight to running and
			// re-queues the check in the same transaction — the check lands on queued, not pending.
			name: "retry failed policy check re-queues it and returns the run to post_plan_running",
			run: postPlanRun(
				models.RunErrored,
				models.RunTaskStageErrored,
				models.PolicyCheckErrored,
				models.ApplySkipped,
				evaluatedPolicies(),
			),
			nodePath:  "post_plan.opa",
			wantRun:   models.RunPostPlanRunning,
			wantNode:  string(models.PolicyCheckQueued),
			wantStage: models.RunTaskStageRunning,
		},
		{
			name: "retry canceled policy check re-queues it",
			run: postPlanRun(
				models.RunCanceled,
				models.RunTaskStageCanceled,
				models.PolicyCheckCanceled,
				models.ApplySkipped,
				evaluatedPolicies(),
			),
			nodePath:  "post_plan.opa",
			wantRun:   models.RunPostPlanRunning,
			wantNode:  string(models.PolicyCheckQueued),
			wantStage: models.RunTaskStageRunning,
		},
		{
			// A workspace-gated stage returns to pending instead and must be re-admitted before it
			// runs, so no job is created yet and the check stays pending.
			name: "retry failed pre-plan check returns the run to pre_plan_queuing without queueing",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunErrored,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanSkipped},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplySkipped},
				TaskStages: []*models.RunTaskStage{{
					ID:        "stage-pre",
					StageName: models.RunTaskStageNamePrePlan,
					Status:    models.RunTaskStageErrored,
					PolicyChecks: []*models.PolicyCheck{{
						ID:        "check-1",
						StageName: models.RunTaskStageNamePrePlan,
						CheckType: models.PolicyKindOPA,
						Status:    models.PolicyCheckErrored,
						Policies:  evaluatedPolicies(),
					}},
				}},
				HasAdvisoryFailures: true,
			},
			nodePath:  "pre_plan.opa",
			wantRun:   models.RunPrePlanQueuing,
			wantNode:  string(models.PolicyCheckPending),
			wantStage: models.RunTaskStagePending,
		},
		{
			// A soft-failed check is retryable as well as overridable: re-evaluating is the resolution
			// once the policy itself has been fixed. The stage was blocked at awaiting_override rather
			// than errored, and it restarts from there the same way.
			name: "retry a soft-failed policy check re-queues it and returns the run to post_plan_running",
			run: postPlanRun(
				models.RunPostPlanAwaitingDecision,
				models.RunTaskStageAwaitingOverride,
				models.PolicyCheckSoftFailed,
				models.ApplyCreated,
				evaluatedPolicies(),
			),
			nodePath:  "post_plan.opa",
			wantRun:   models.RunPostPlanRunning,
			wantNode:  string(models.PolicyCheckQueued),
			wantStage: models.RunTaskStageRunning,
		},
		{
			// The gated counterpart: the run gave up its workspace slot when it parked at the gate, so
			// the retry leaves it queuing for admission with no job created yet.
			name: "retry a soft-failed pre-plan check returns the run to pre_plan_queuing without queueing",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPrePlanAwaitingDecision,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
				TaskStages: []*models.RunTaskStage{{
					ID:        "stage-pre",
					StageName: models.RunTaskStageNamePrePlan,
					Status:    models.RunTaskStageAwaitingOverride,
					PolicyChecks: []*models.PolicyCheck{{
						ID:        "check-1",
						StageName: models.RunTaskStageNamePrePlan,
						CheckType: models.PolicyKindOPA,
						Status:    models.PolicyCheckSoftFailed,
						Policies:  evaluatedPolicies(),
					}},
				}},
				HasAdvisoryFailures: true,
			},
			nodePath:  "pre_plan.opa",
			wantRun:   models.RunPrePlanQueuing,
			wantNode:  string(models.PolicyCheckPending),
			wantStage: models.RunTaskStagePending,
		},
		{
			// A discarded run keeps its soft-failed check and gate so undiscard can restore it; a retry
			// would delete the gate and leave nothing to restore, so it is rejected outright.
			name: "retry a soft-failed policy check on a discarded run yields conflict",
			run: postPlanRun(
				models.RunDiscarded,
				models.RunTaskStageAwaitingOverride,
				models.PolicyCheckSoftFailed,
				models.ApplyCreated,
				evaluatedPolicies(),
			),
			nodePath: "post_plan.opa",
			wantCode: errors.EConflict,
		},
		{
			name: "retry a passed policy check yields conflict",
			run: postPlanRun(
				models.RunPlanned,
				models.RunTaskStageCompleted,
				models.PolicyCheckPassed,
				models.ApplyCreated,
				[]*models.PolicyCheckPolicy{{ID: "policy-ok", Status: models.PolicyCheckPolicyPassed}},
			),
			nodePath: "post_plan.opa",
			wantCode: errors.EConflict,
		},
		{
			name: "retry a well-formed path the run has no check for yields invalid error",
			run: postPlanRun(
				models.RunErrored,
				models.RunTaskStageErrored,
				models.PolicyCheckErrored,
				models.ApplySkipped,
				evaluatedPolicies(),
			),
			nodePath: "pre_apply.opa",
			wantCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(tt.run)

			// On success the retry is recorded as an UPDATE activity event whose payload
			// identifies the retry sub-action and the node path that was retried. On the
			// error cases Execute returns before recording, so NewMockActivityEvents(t)
			// (which fails on any unexpected call) verifies no event is created.
			mockActivityEvents := db.NewMockActivityEvents(t)
			if tt.wantCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
					var payload models.ActivityEventUpdateRunPayload
					if err := json.Unmarshal(in.Payload, &payload); err != nil {
						return false
					}
					return in.Action == models.ActionUpdate &&
						in.TargetType == models.TargetRun &&
						payload.Type == string(models.RunUpdateTypeRetry) &&
						payload.NodePath != nil && *payload.NodePath == tt.nodePath
				})).Return(&models.ActivityEvent{}, nil)
			}

			cmd := &RetryRunNode{
				dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
				in:            &RetryRunNodeInput{RunID: "run-1", NodePath: tt.nodePath},
				namespacePath: "groupA/ws",
			}

			err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, errors.ErrorCode(err))
				assert.Nil(t, cmd.Updated)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd.Updated)
			assert.Equal(t, tt.wantRun, cmd.Updated.Status)
			assert.False(t, cmd.Updated.ForceCanceled, "force-cancellation state should be cleared on retry")
			assert.Nil(t, cmd.Updated.ForceCanceledBy)
			assert.Nil(t, cmd.Updated.ForceCancelAvailableAt)

			switch tt.nodePath {
			case "plan":
				assert.Equal(t, models.PlanStatus(tt.wantNode), cmd.Updated.Plan.Status)
				assert.Nil(t, cmd.Updated.Plan.ErrorMessage, "plan error message should be cleared on retry")
			case "apply":
				assert.Equal(t, models.ApplyStatus(tt.wantNode), cmd.Updated.Apply.Status)
				assert.Nil(t, cmd.Updated.Apply.ErrorMessage, "apply error message should be cleared on retry")
			default:
				check := cmd.Updated.PolicyCheckByPath(tt.nodePath)
				require.NotNil(t, check)
				assert.Equal(t, models.PolicyCheckStatus(tt.wantNode), check.Status)
				// The previous attempt's verdicts describe an evaluation that is being replaced, so none
				// of them may survive into the re-evaluation.
				require.NotEmpty(t, check.Policies)
				for _, policy := range check.Policies {
					assert.Equal(t, models.PolicyCheckPolicyPending, policy.Status,
						"policy %s should be reset to pending on retry", policy.ID)
					assert.Nil(t, policy.MessagesObjectStoreKey,
						"policy %s messages should be cleared on retry", policy.ID)
				}
				assert.Nil(t, check.MessagesSummary, "check message summary should be cleared on retry")
				// This run has one check, so returning its policies to pending leaves no advisory
				// failure anywhere on the run.
				assert.False(t, cmd.Updated.HasAdvisoryFailures,
					"advisory failures flag should be cleared when the only failing check is retried")
			}

			if tt.wantStage != "" {
				// A check's node path is "<stage name>.<check type>".
				stageName, _, ok := strings.Cut(tt.nodePath, ".")
				require.True(t, ok, "wantStage only applies to a policy check path")
				stage := cmd.Updated.TaskStageByStageName(models.RunTaskStageName(stageName))
				require.NotNil(t, stage)
				assert.Equal(t, tt.wantStage, stage.Status)
			}
		})
	}
}

// TestRetryRunNode_Execute_AdvisoryFailuresOnAnotherStage covers why the retry recomputes the run's
// advisory-failures flag from every check instead of clearing it: the retried check's advisory failure is
// discarded, but an earlier stage's check still holds one, so the run is still carrying findings.
func TestRetryRunNode_Execute_AdvisoryFailuresOnAnotherStage(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))

	advisoryFailed := func(id string) []*models.PolicyCheckPolicy {
		return []*models.PolicyCheckPolicy{{
			ID:               id,
			Status:           models.PolicyCheckPolicyFailed,
			EnforcementLevel: models.PolicyEnforcementAdvisory,
		}}
	}

	run := &models.Run{
		Metadata:            models.ResourceMetadata{ID: "run-1"},
		Status:              models.RunErrored,
		Plan:                models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:               &models.Apply{ID: "apply-1", Status: models.ApplySkipped},
		HasAdvisoryFailures: true,
		TaskStages: []*models.RunTaskStage{
			{
				// Completed: its advisory failure did not block the run, and a retry of the post-plan
				// check leaves this stage alone.
				ID:        "stage-pre",
				StageName: models.RunTaskStageNamePrePlan,
				Status:    models.RunTaskStageCompleted,
				PolicyChecks: []*models.PolicyCheck{{
					ID:        "check-pre",
					StageName: models.RunTaskStageNamePrePlan,
					CheckType: models.PolicyKindOPA,
					Status:    models.PolicyCheckPassed,
					Policies:  advisoryFailed("policy-pre-advisory"),
				}},
			},
			{
				ID:        "stage-post",
				StageName: models.RunTaskStageNamePostPlan,
				Status:    models.RunTaskStageErrored,
				PolicyChecks: []*models.PolicyCheck{{
					ID:        "check-post",
					StageName: models.RunTaskStageNamePostPlan,
					CheckType: models.PolicyKindOPA,
					Status:    models.PolicyCheckErrored,
					Policies:  advisoryFailed("policy-post-advisory"),
				}},
			},
		},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	mockActivityEvents := db.NewMockActivityEvents(t)
	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	cmd := &RetryRunNode{
		dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
		in:            &RetryRunNodeInput{RunID: "run-1", NodePath: "post_plan.opa"},
		namespacePath: "groupA/ws",
	}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	// The retried check's policy is back to pending...
	retried := cmd.Updated.PolicyCheckByPath("post_plan.opa")
	require.NotNil(t, retried)
	require.Len(t, retried.Policies, 1)
	assert.Equal(t, models.PolicyCheckPolicyPending, retried.Policies[0].Status)

	// ...but the pre-plan check's advisory failure survives, so the run flag stays set.
	assert.True(t, cmd.Updated.HasAdvisoryFailures,
		"another stage still holds a failed advisory policy, so the flag must not be cleared")
}
