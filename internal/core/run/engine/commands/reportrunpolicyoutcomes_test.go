package commands

import (
	"context"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// policyCheckPolicy is a small helper to build a per-policy snapshot for the verdict tests,
// keyed by its stable id. EnforcementLevel drives the verdict.
func policyCheckPolicy(id string, level models.PolicyEnforcementLevel) *models.PolicyCheckPolicy {
	return &models.PolicyCheckPolicy{
		ID:               id,
		OPAData:          &models.PolicyCheckOPAData{PackageVersionConstraint: "~> 1.0"},
		EnforcementLevel: level,
		Status:           models.PolicyCheckPolicyPending,
	}
}

func TestReportRunPolicyOutcomes_Execute(t *testing.T) {
	type testCase struct {
		name         string
		policies     []*models.PolicyCheckPolicy
		outcomes     []RunPolicyOutcome
		wantVerdict  models.PolicyCheckStatus
		wantRunState models.RunStatus
		// wantAdvisoryFailures is the run-level flag after the report. It tracks failed *advisory*
		// policies specifically, so a mandatory failure must leave it false.
		wantAdvisoryFailures bool
	}

	tests := []testCase{
		{
			name:         "all pass -> passed",
			policies:     []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementHardMandatory)},
			outcomes:     []RunPolicyOutcome{{PolicyID: "a", Passed: true}},
			wantVerdict:  models.PolicyCheckPassed,
			wantRunState: models.RunPlanned,
		},
		{
			// The verdict is passed and the run advances, so the flag is the only record that anything
			// was reported at all.
			name:                 "advisory failure only -> passed",
			policies:             []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementAdvisory)},
			outcomes:             []RunPolicyOutcome{{PolicyID: "a", Passed: false}},
			wantVerdict:          models.PolicyCheckPassed,
			wantRunState:         models.RunPlanned,
			wantAdvisoryFailures: true,
		},
		{
			name:         "advisory policy that passed -> no advisory failures",
			policies:     []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementAdvisory)},
			outcomes:     []RunPolicyOutcome{{PolicyID: "a", Passed: true}},
			wantVerdict:  models.PolicyCheckPassed,
			wantRunState: models.RunPlanned,
		},
		{
			name:         "soft-mandatory failure -> awaiting_override",
			policies:     []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementSoftMandatory)},
			outcomes:     []RunPolicyOutcome{{PolicyID: "a", Passed: false}},
			wantVerdict:  models.PolicyCheckSoftFailed,
			wantRunState: models.RunPostPlanAwaitingDecision,
		},
		{
			name:         "hard-mandatory failure -> errored",
			policies:     []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementHardMandatory)},
			outcomes:     []RunPolicyOutcome{{PolicyID: "a", Passed: false}},
			wantVerdict:  models.PolicyCheckErrored,
			wantRunState: models.RunErrored,
		},
		{
			name: "soft fail with an advisory fail -> awaiting_override",
			policies: []*models.PolicyCheckPolicy{
				policyCheckPolicy("a", models.PolicyEnforcementSoftMandatory),
				policyCheckPolicy("b", models.PolicyEnforcementAdvisory),
			},
			outcomes: []RunPolicyOutcome{
				{PolicyID: "a", Passed: false},
				{PolicyID: "b", Passed: false},
			},
			wantVerdict:          models.PolicyCheckSoftFailed,
			wantRunState:         models.RunPostPlanAwaitingDecision,
			wantAdvisoryFailures: true,
		},
		{
			// Two policies of the same policy set + version resolve to distinct entries; only the
			// one whose id the outcome names is stamped (match-by-id, not by set+version).
			name: "same set twice, match by id",
			policies: []*models.PolicyCheckPolicy{
				policyCheckPolicy("a", models.PolicyEnforcementHardMandatory),
				policyCheckPolicy("b", models.PolicyEnforcementHardMandatory),
			},
			outcomes:     []RunPolicyOutcome{{PolicyID: "a", Passed: true}, {PolicyID: "b", Passed: true}},
			wantVerdict:  models.PolicyCheckPassed,
			wantRunState: models.RunPlanned,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			run := &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPostPlanRunning,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
				TaskStages: []*models.RunTaskStage{{
					ID:        "ts-post",
					StageName: models.RunTaskStageNamePostPlan,
					Status:    models.RunTaskStageRunning,
					PolicyChecks: []*models.PolicyCheck{{
						ID:        "stage-1",
						StageName: models.RunTaskStageNamePostPlan,
						CheckType: models.PolicyKindOPA,
						Status:    models.PolicyCheckRunning,
						// The policy snapshot the command reads for enforcement lives on the check.
						Policies: tc.policies,
					}},
				}},
			}

			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(run)

			cmd := &ReportRunPolicyOutcomes{dbClient: &db.Client{}, PolicyCheckID: "stage-1", Outcomes: tc.outcomes, runID: "run-1"}

			require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
			assert.Equal(t, tc.wantVerdict, run.AllPolicyChecks()[0].Status)
			assert.Equal(t, tc.wantRunState, run.Status)
			assert.Equal(t, tc.wantAdvisoryFailures, run.HasAdvisoryFailures)
			// Each reported outcome stamps a pass/fail status onto its matching policy entry (by id).
			for _, outcome := range tc.outcomes {
				var found bool
				for _, p := range run.AllPolicyChecks()[0].Policies {
					if p.ID == outcome.PolicyID {
						found = true
						if outcome.Passed {
							assert.Equal(t, models.PolicyCheckPolicyPassed, p.Status)
						} else {
							assert.Equal(t, models.PolicyCheckPolicyFailed, p.Status)
						}
					}
				}
				assert.True(t, found, "reported outcome should match a pinned policy")
			}
		})
	}
}

// TestReportRunPolicyOutcomes_Execute_RunningStage verifies that outcomes arriving while the
// check is running produce the correct verdict and advance the run to planned.
func TestReportRunPolicyOutcomes_Execute_RunningStage(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPostPlanRunning,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{{
			ID:        "ts-post",
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageRunning,
			PolicyChecks: []*models.PolicyCheck{{
				ID:        "stage-1",
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindOPA,
				Status:    models.PolicyCheckRunning,
				Policies:  []*models.PolicyCheckPolicy{policyCheckPolicy("a", models.PolicyEnforcementHardMandatory)},
			}},
		}},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &ReportRunPolicyOutcomes{
		dbClient:      &db.Client{},
		PolicyCheckID: "stage-1",
		Outcomes:      []RunPolicyOutcome{{PolicyID: "a", Passed: true}},
		runID:         "run-1",
	}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.PolicyCheckPassed, run.AllPolicyChecks()[0].Status)
	assert.Equal(t, models.RunPlanned, run.Status)
}

// TestReportRunPolicyOutcomes_Execute_Messages verifies where the reported messages end up: the object
// key Prepare uploaded is stamped on the policy that reported them and its ref is retained against the
// run, a policy that reported none is left with no key, and the check carries the summary. The upload
// itself belongs to Prepare, so the keys are seeded here the way runID is in the tests above.
func TestReportRunPolicyOutcomes_Execute_Messages(t *testing.T) {
	ctx := context.Background()

	// A stale key on the passing policy, to prove a report with no messages for it clears the key
	// rather than leaving the previous attempt's messages attached.
	staleKey := "workspaces/ws-1/runs/run-1/policy_messages/stale.json"
	failed := policyCheckPolicy("failed", models.PolicyEnforcementAdvisory)
	passed := policyCheckPolicy("passed", models.PolicyEnforcementAdvisory)
	passed.MessagesObjectStoreKey = &staleKey

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPostPlanRunning,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{{
			ID:        "ts-post",
			StageName: models.RunTaskStageNamePostPlan,
			Status:    models.RunTaskStageRunning,
			PolicyChecks: []*models.PolicyCheck{{
				ID:        "stage-1",
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindOPA,
				Status:    models.PolicyCheckRunning,
				Policies:  []*models.PolicyCheckPolicy{failed, passed},
			}},
		}},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	var retainedOwner string
	messagesKey := "workspaces/ws-1/runs/run-1/policy_messages/obj-1.json"
	cmd := &ReportRunPolicyOutcomes{
		dbClient:      &db.Client{},
		PolicyCheckID: "stage-1",
		Outcomes: []RunPolicyOutcome{
			{PolicyID: "failed", Passed: false, Messages: []string{"denied by rule X"}},
			{PolicyID: "passed", Passed: true},
		},
		runID:       "run-1",
		messageKeys: map[string]string{"failed": messagesKey},
		messageRetainFns: map[string]db.RetainObjectRefFunc{
			"failed": func(_ context.Context, ownerID string) error {
				retainedOwner = ownerID
				return nil
			},
		},
		messagesSummary: &models.PolicyCheckMessagesSummary{Messages: []string{"denied by rule X"}},
	}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	check := run.AllPolicyChecks()[0]
	require.NotNil(t, failed.MessagesObjectStoreKey)
	assert.Equal(t, messagesKey, *failed.MessagesObjectStoreKey)
	assert.Equal(t, "run-1", retainedOwner, "the messages object should be retained against the run")
	assert.Nil(t, passed.MessagesObjectStoreKey, "a policy reporting no messages keeps no key")
	require.NotNil(t, check.MessagesSummary)
	assert.Equal(t, []string{"denied by rule X"}, check.MessagesSummary.Messages)
}

// TestReportRunPolicyOutcomes_Prepare covers what Prepare does with the reported messages: uploads them
// per policy and builds the check summary, or rejects the whole report when one policy reported more
// than the limit. A rejection has to happen here, before anything is written — the policy-eval job
// turns the error into a failed job, and a partially stored result would contradict that.
func TestReportRunPolicyOutcomes_Prepare(t *testing.T) {
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		TaskStages: []*models.RunTaskStage{{
			ID:        "ts-post",
			StageName: models.RunTaskStageNamePostPlan,
			PolicyChecks: []*models.PolicyCheck{{
				ID:        "stage-1",
				StageName: models.RunTaskStageNamePostPlan,
				CheckType: models.PolicyKindOPA,
			}},
		}},
	}

	type testCase struct {
		name            string
		outcomes        []RunPolicyOutcome
		wantUploads     int
		wantSummary     []string
		wantTruncated   bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "messages are uploaded per policy and summarized across them",
			outcomes: []RunPolicyOutcome{
				{PolicyID: "pol-1", Messages: []string{"denied by rule X"}},
				{PolicyID: "pol-2", Messages: []string{"denied by rule Y"}},
			},
			wantUploads: 2,
			wantSummary: []string{"denied by rule X", "denied by rule Y"},
		},
		{
			name: "a policy reporting nothing is not uploaded",
			outcomes: []RunPolicyOutcome{
				{PolicyID: "pol-1", Passed: true},
			},
			wantUploads: 0,
		},
		{
			name: "messages beyond the summary budget are truncated rather than rejected",
			outcomes: []RunPolicyOutcome{
				{PolicyID: "pol-1", Messages: []string{strings.Repeat("a", maxPolicyCheckMessagesSummarySize+10)}},
			},
			wantUploads:   1,
			wantSummary:   []string{strings.Repeat("a", maxPolicyCheckMessagesSummarySize)},
			wantTruncated: true,
		},
		{
			name: "a policy over the message limit is rejected",
			outcomes: []RunPolicyOutcome{
				{PolicyID: "pol-1", Messages: []string{strings.Repeat("a", maxPolicyMessagesSize+1)}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			// The limit is the total across a policy's messages, not the size of any one of them.
			name: "many small messages over the limit in total are rejected",
			outcomes: []RunPolicyOutcome{
				{PolicyID: "pol-1", Messages: slices.Repeat([]string{strings.Repeat("a", 1024)}, 1025)},
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockRuns := db.NewMockRuns(t)
			mockRuns.On("GetRunByNodeID", mock.Anything, "stage-1").Return(run, nil)

			mockArtifactStore := workspace.NewMockArtifactStore(t)
			retainFn := func(_ context.Context, _ string) error { return nil }
			if test.wantUploads > 0 {
				mockArtifactStore.On("UploadPolicyCheckPolicyMessages", mock.Anything, run, mock.Anything).
					Return(db.RetainObjectRefFunc(retainFn), "policy_messages/obj.json", nil).
					Times(test.wantUploads)
			}

			cmd := &ReportRunPolicyOutcomes{
				dbClient:      &db.Client{Runs: mockRuns},
				artifactStore: mockArtifactStore,
				PolicyCheckID: "stage-1",
				Outcomes:      test.outcomes,
			}

			err := cmd.Prepare(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				// Nothing may be staged for Execute: the report is rejected whole.
				assert.Empty(t, cmd.messageKeys)
				assert.Nil(t, cmd.messagesSummary)
				return
			}

			require.NoError(t, err)
			assert.Len(t, cmd.messageKeys, test.wantUploads)
			if test.wantSummary == nil {
				assert.Nil(t, cmd.messagesSummary)
				return
			}
			require.NotNil(t, cmd.messagesSummary)
			assert.Equal(t, test.wantSummary, cmd.messagesSummary.Messages)
			assert.Equal(t, test.wantTruncated, cmd.messagesSummary.Truncated)
		})
	}
}

func TestSummarizePolicyMessages(t *testing.T) {
	// Just over half the budget, so two of these cannot both be kept.
	halfBudget := strings.Repeat("a", maxPolicyCheckMessagesSummarySize/2+1)

	tests := []struct {
		name          string
		input         []string
		want          []string
		wantTruncated bool
	}{
		{
			name:  "no messages summarize to nothing",
			input: nil,
			want:  []string{},
		},
		{
			name:  "messages within the budget are kept whole",
			input: []string{"first violation", "second violation"},
			want:  []string{"first violation", "second violation"},
		},
		{
			// The second message overruns the budget, so it is cut to what is left rather than
			// dropped whole — and the third has no room at all.
			name:  "the message that overruns the budget is cut to the space left",
			input: []string{halfBudget, halfBudget, "third violation"},
			want: []string{
				halfBudget,
				strings.Repeat("a", maxPolicyCheckMessagesSummarySize-len(halfBudget)),
			},
			wantTruncated: true,
		},
		{
			name:          "a lone message larger than the budget is cut and flagged",
			input:         []string{strings.Repeat("b", maxPolicyCheckMessagesSummarySize+10)},
			want:          []string{strings.Repeat("b", maxPolicyCheckMessagesSummarySize)},
			wantTruncated: true,
		},
		{
			// Exactly at the budget with more to come: there is no room for even one byte of the
			// next message, so nothing is appended but the summary is still partial.
			name: "a message with no room left adds nothing",
			input: []string{
				strings.Repeat("c", maxPolicyCheckMessagesSummarySize),
				"second violation",
			},
			want:          []string{strings.Repeat("c", maxPolicyCheckMessagesSummarySize)},
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := summarizePolicyMessages(tt.input)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantTruncated, truncated)
		})
	}
}

// TestSummarizePolicyMessages_MultiByteBoundary verifies a cut never splits a rune: the space left
// lands mid-character here, so a naive slice would leave an invalid trailing byte.
func TestSummarizePolicyMessages_MultiByteBoundary(t *testing.T) {
	// "é" is two bytes, so an odd amount of room puts a rune across the limit. One byte short of the
	// budget leaves the cut with an odd number of bytes to fill.
	got, truncated := summarizePolicyMessages([]string{
		strings.Repeat("a", maxPolicyCheckMessagesSummarySize-1),
		strings.Repeat("é", 10),
	})

	// The odd byte cannot hold a two-byte rune, so the second message contributes nothing.
	require.Len(t, got, 1)
	assert.True(t, truncated)

	// And a message cut mid-rune keeps only whole ones.
	got, truncated = summarizePolicyMessages([]string{strings.Repeat("é", maxPolicyCheckMessagesSummarySize)})
	require.Len(t, got, 1)
	assert.True(t, truncated)
	assert.True(t, utf8.ValidString(got[0]))
	assert.LessOrEqual(t, len(got[0]), maxPolicyCheckMessagesSummarySize)
}
