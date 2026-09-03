package eventhandlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestPolicyCheckWorkItemEnqueuer_HandleRunChanges(t *testing.T) {
	logr, _ := logger.NewForTest()

	// runWithCheck builds a run holding a single policy check of the given kind and status under a
	// pre_plan task stage, so run.PolicyCheckByPath(c.Path) resolves it. status matters here: the
	// handler re-reads the check's current status off the run rather than trusting the change event's
	// NewStatus, so a fixture whose check is not actually queued must not be enqueued even though the
	// event claims queued.
	runWithCheck := func(checkID string, checkType models.PolicyKind, status models.PolicyCheckStatus) *models.Run {
		check := &models.PolicyCheck{
			ID:        checkID,
			StageName: models.RunTaskStageNamePrePlan,
			CheckType: checkType,
			Status:    status,
		}
		return &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			TaskStages: []*models.RunTaskStage{{
				StageName:    models.RunTaskStageNamePrePlan,
				PolicyChecks: []*models.PolicyCheck{check},
			}},
		}
	}

	t.Run("enqueues on a module attestation check entering queued", func(t *testing.T) {
		run := runWithCheck("check-1", models.PolicyKindModuleAttestation, models.PolicyCheckQueued)
		check := run.PolicyCheckByPath(string(models.RunTaskStageNamePrePlan) + "." + string(models.PolicyKindModuleAttestation))
		require.NotNil(t, check)

		mockWIQ := db.NewMockWorkItemsQueue(t)
		mockWIQ.On("AddWorkItemToQueue", mock.Anything, mock.MatchedBy(func(in *db.AddWorkItemToQueueInput) bool {
			payload, ok := in.Payload.(*db.EvaluateRunPolicyCheckPayload)
			return in.Type == db.EvaluateRunPolicyCheckType &&
				ok && payload.RunID == "run-1" && payload.PolicyCheckID == "check-1"
		})).Return(&db.WorkItem{}, nil).Once()

		handler := NewPolicyCheckWorkItemEnqueuer(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{{
			Run: run,
			NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PolicyCheckStatusChange{
				NewStatus: models.PolicyCheckQueued,
				CheckID:   "check-1",
				Path:      check.GetPath(),
			}},
		}})
		require.NoError(t, err)
		mockWIQ.AssertExpectations(t)
	})

	t.Run("does not enqueue for an OPA check", func(t *testing.T) {
		run := runWithCheck("check-1", models.PolicyKindOPA, models.PolicyCheckQueued)
		check := run.PolicyCheckByPath(string(models.RunTaskStageNamePrePlan) + "." + string(models.PolicyKindOPA))
		require.NotNil(t, check)

		// NewMockWorkItemsQueue(t) fails the test on any unexpected call.
		mockWIQ := db.NewMockWorkItemsQueue(t)
		handler := NewPolicyCheckWorkItemEnqueuer(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{{
			Run: run,
			NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PolicyCheckStatusChange{
				NewStatus: models.PolicyCheckQueued,
				CheckID:   "check-1",
				Path:      check.GetPath(),
			}},
		}})
		require.NoError(t, err)
	})

	t.Run("does not enqueue for a non-queued status", func(t *testing.T) {
		run := runWithCheck("check-1", models.PolicyKindModuleAttestation, models.PolicyCheckRunning)
		check := run.PolicyCheckByPath(string(models.RunTaskStageNamePrePlan) + "." + string(models.PolicyKindModuleAttestation))
		require.NotNil(t, check)

		mockWIQ := db.NewMockWorkItemsQueue(t)
		handler := NewPolicyCheckWorkItemEnqueuer(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{{
			Run: run,
			NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PolicyCheckStatusChange{
				NewStatus: models.PolicyCheckRunning,
				CheckID:   "check-1",
				Path:      check.GetPath(),
			}},
		}})
		require.NoError(t, err)
	})

	t.Run("ignores non-policy-check status changes", func(t *testing.T) {
		run := runWithCheck("check-1", models.PolicyKindModuleAttestation, models.PolicyCheckQueued)

		mockWIQ := db.NewMockWorkItemsQueue(t)
		handler := NewPolicyCheckWorkItemEnqueuer(logr, &db.Client{WorkItemsQueue: mockWIQ})

		err := handler.HandleRunChanges(context.Background(), []types.RunChange{{
			Run: run,
			NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.RunStatusChange{
				NewStatus: models.RunPlanning,
			}},
		}})
		require.NoError(t, err)
	})
}
