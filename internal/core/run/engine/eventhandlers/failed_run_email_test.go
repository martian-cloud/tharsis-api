package eventhandlers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetFailedRuns(t *testing.T) {
	runA := &models.Run{Metadata: models.ResourceMetadata{ID: "run-a"}}
	runB := &models.Run{Metadata: models.ResourceMetadata{ID: "run-b"}}

	type testCase struct {
		name    string
		changes []types.RunChange
		// expected pairs of run-id + stage in order
		expectStages []builder.RunStage
		expectRunIDs []string
	}

	testCases := []testCase{
		{
			name:         "no node status changes yields nothing",
			changes:      []types.RunChange{{Run: runA}},
			expectStages: nil,
			expectRunIDs: nil,
		},
		{
			name: "non-errored transitions yield nothing",
			changes: []types.RunChange{{
				Run: runA,
				NodeStatusChanges: []statemachine.NodeStatusChange{
					statemachine.PlanStatusChange{NewStatus: models.PlanFinished},
					statemachine.ApplyStatusChange{NewStatus: models.ApplyFinished},
					statemachine.RunStatusChange{NewStatus: models.RunErrored},
				},
			}},
			expectStages: nil,
			expectRunIDs: nil,
		},
		{
			name: "plan errored yields a plan-stage entry",
			changes: []types.RunChange{{
				Run: runA,
				NodeStatusChanges: []statemachine.NodeStatusChange{
					statemachine.PlanStatusChange{OldStatus: models.PlanRunning, NewStatus: models.PlanErrored},
				},
			}},
			expectStages: []builder.RunStage{builder.PlanStage},
			expectRunIDs: []string{"run-a"},
		},
		{
			name: "apply errored yields an apply-stage entry",
			changes: []types.RunChange{{
				Run: runA,
				NodeStatusChanges: []statemachine.NodeStatusChange{
					statemachine.ApplyStatusChange{OldStatus: models.ApplyRunning, NewStatus: models.ApplyErrored},
				},
			}},
			expectStages: []builder.RunStage{builder.ApplyStage},
			expectRunIDs: []string{"run-a"},
		},
		{
			name: "one entry per errored node across multiple changes",
			changes: []types.RunChange{
				{
					Run: runA,
					NodeStatusChanges: []statemachine.NodeStatusChange{
						statemachine.PlanStatusChange{NewStatus: models.PlanErrored},
					},
				},
				{
					Run: runB,
					NodeStatusChanges: []statemachine.NodeStatusChange{
						statemachine.PlanStatusChange{NewStatus: models.PlanFinished},
						statemachine.ApplyStatusChange{NewStatus: models.ApplyErrored},
					},
				},
			},
			expectStages: []builder.RunStage{builder.PlanStage, builder.ApplyStage},
			expectRunIDs: []string{"run-a", "run-b"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got := getFailedRuns(test.changes)
			require.Len(t, got, len(test.expectStages))
			for i := range test.expectStages {
				assert.Equal(t, test.expectStages[i], got[i].stage)
				assert.Equal(t, test.expectRunIDs[i], got[i].run.Metadata.ID)
			}
		})
	}
}

func TestFailureSubject(t *testing.T) {
	type testCase struct {
		name     string
		run      *models.Run
		stage    builder.RunStage
		expected string
	}

	testCases := []testCase{
		{
			name:     "speculative plan",
			run:      &models.Run{Apply: nil},
			stage:    builder.PlanStage,
			expected: "speculative plan failed",
		},
		{
			name:     "speculative destroy plan",
			run:      &models.Run{Apply: nil, IsDestroy: true},
			stage:    builder.PlanStage,
			expected: "speculative destroy plan failed",
		},
		{
			name:     "destroy plan stage",
			run:      &models.Run{Apply: &models.Apply{}, IsDestroy: true},
			stage:    builder.PlanStage,
			expected: "destroy plan failed",
		},
		{
			name:     "destroy apply stage",
			run:      &models.Run{Apply: &models.Apply{}, IsDestroy: true},
			stage:    builder.ApplyStage,
			expected: "destroy failed",
		},
		{
			name:     "regular plan",
			run:      &models.Run{Apply: &models.Apply{}},
			stage:    builder.PlanStage,
			expected: "plan failed",
		},
		{
			name:     "regular apply",
			run:      &models.Run{Apply: &models.Apply{}},
			stage:    builder.ApplyStage,
			expected: "apply failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, failureSubject(test.run, test.stage))
		})
	}
}

func TestFailedRunEmailHandler_HandleRunChanges_SkipsAssessmentRuns(t *testing.T) {
	logr, _ := logger.NewForTest()

	// All mocks are constructed but should never be called for an assessment run.
	mockWS := db.NewMockWorkspaces(t)
	dbClient := &db.Client{Workspaces: mockWS}
	emailClient := email.NewMockEnqueuer(t)
	notifMgr := namespace.NewMockNotificationManager(t)

	handler := NewFailedRunEmailHandler(logr, dbClient, emailClient, notifMgr)

	changes := []types.RunChange{{
		Run: &models.Run{
			Metadata:        models.ResourceMetadata{ID: "run-1"},
			IsAssessmentRun: true,
		},
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PlanStatusChange{NewStatus: models.PlanErrored},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(context.Background(), changes))
	emailClient.AssertNotCalled(t, "EnqueueEmail", mock.Anything, mock.Anything)
}

func TestFailedRunEmailHandler_HandleRunChanges_SendsEmailPerFailedRun(t *testing.T) {
	logr, _ := logger.NewForTest()

	ws := &models.Workspace{FullPath: "group/ws"}

	mockWS := db.NewMockWorkspaces(t)
	mockWS.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(ws, nil)

	emailClient := email.NewMockEnqueuer(t)
	emailClient.On("EnqueueEmail", mock.Anything, mock.MatchedBy(func(in *email.EnqueueEmailInput) bool {
		return in.Subject == "Tharsis plan failed" && len(in.UserIDs) == 1 && in.UserIDs[0] == "user-1"
	})).Return(nil)

	notifMgr := namespace.NewMockNotificationManager(t)
	notifMgr.On("GetUsersToNotify", mock.Anything, mock.Anything).Return([]string{"user-1"}, nil)

	dbClient := &db.Client{Workspaces: mockWS}

	handler := NewFailedRunEmailHandler(logr, dbClient, emailClient, notifMgr)

	changes := []types.RunChange{{
		Run: &models.Run{
			Metadata:    models.ResourceMetadata{ID: "run-1"},
			WorkspaceID: "ws-1",
			CreatedBy:   "system",
			Apply:       &models.Apply{},
			Plan:        models.Plan{ErrorMessage: ptr.String("boom")},
		},
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.PlanStatusChange{NewStatus: models.PlanErrored},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(context.Background(), changes))

	emailClient.AssertExpectations(t)
}

func TestFailedRunEmailHandler_HandleRunChanges_NoUsersToNotifySkipsSend(t *testing.T) {
	logr, _ := logger.NewForTest()

	mockWS := db.NewMockWorkspaces(t)
	mockWS.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{FullPath: "group/ws"}, nil)

	emailClient := email.NewMockEnqueuer(t) // SendMail must never be called.

	notifMgr := namespace.NewMockNotificationManager(t)
	notifMgr.On("GetUsersToNotify", mock.Anything, mock.Anything).Return([]string{}, nil)

	dbClient := &db.Client{Workspaces: mockWS}

	handler := NewFailedRunEmailHandler(logr, dbClient, emailClient, notifMgr)

	changes := []types.RunChange{{
		Run: &models.Run{
			Metadata:    models.ResourceMetadata{ID: "run-1"},
			WorkspaceID: "ws-1",
			Apply:       &models.Apply{ErrorMessage: ptr.String("nope")},
		},
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.ApplyStatusChange{NewStatus: models.ApplyErrored},
		},
	}}

	require.NoError(t, handler.HandleRunChanges(context.Background(), changes))

}
