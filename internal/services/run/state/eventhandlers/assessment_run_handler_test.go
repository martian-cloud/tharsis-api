package eventhandlers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestAssessmentRunHandler_handleRunEvent(t *testing.T) {
	workspaceID := "workspace-1"
	planID := "plan-1"
	applyID := "apply-1"
	stateVersionID := "state-version-1"

	type testCase struct {
		name             string
		run              *models.Run
		assessment       *models.WorkspaceAssessment
		workspace        *models.Workspace
		stateVersion     *models.StateVersion
		plan             *models.Plan
		expectAssessment *models.WorkspaceAssessment
	}

	testCases := []testCase{
		{
			name: "run should be ignored since it's not complete",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:      models.RunPlanned,
				WorkspaceID: workspaceID,
				PlanID:      planID,
				ApplyID:     applyID,
			},
		},
		{
			name: "run should be ignored since it's a speculative non-assessment run",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:      models.RunPlannedAndFinished,
				WorkspaceID: workspaceID,
				PlanID:      planID,
			},
		},
		{
			name: "completed run should not clear the assessment since the workspace doesn't have a current state version",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:      models.RunApplied,
				WorkspaceID: workspaceID,
				PlanID:      planID,
				ApplyID:     applyID,
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "workspace-1",
				},
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
			},
		},
		{
			name: "completed run should clear assessment record since the state version has changed due to the run",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:      models.RunApplied,
				WorkspaceID: workspaceID,
				PlanID:      planID,
				ApplyID:     applyID,
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "workspace-1",
				},
				CurrentStateVersionID: stateVersionID,
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				RunID: ptr.String("run-1"),
			},
		},
		{
			name: "completed assessment should set drift to true",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:          models.RunPlannedAndFinished,
				WorkspaceID:     workspaceID,
				PlanID:          planID,
				ApplyID:         applyID,
				IsAssessmentRun: true,
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
			},
			plan: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
				Summary: models.PlanSummary{
					ResourceDrift: 1,
				},
			},
			expectAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID:          workspaceID,
				HasDrift:             true,
				RequiresNotification: true,
			},
		},
		{
			name: "completed assessment should set drift to true and requires notifications to false",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:          models.RunPlannedAndFinished,
				WorkspaceID:     workspaceID,
				PlanID:          planID,
				ApplyID:         applyID,
				IsAssessmentRun: true,
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
				HasDrift:    true,
			},
			plan: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
				Summary: models.PlanSummary{
					ResourceDrift: 1,
				},
			},
			expectAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID:          workspaceID,
				HasDrift:             true,
				RequiresNotification: false,
			},
		},
		{
			name: "completed assessment should set drift to false",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:          models.RunPlannedAndFinished,
				WorkspaceID:     workspaceID,
				PlanID:          planID,
				ApplyID:         applyID,
				IsAssessmentRun: true,
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
			},
			plan: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
				Summary: models.PlanSummary{
					ResourceDrift: 0,
				},
			},
			expectAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID:          workspaceID,
				HasDrift:             false,
				RequiresNotification: false,
			},
		},
		{
			name: "failed assessment should set completed timestamp",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run-1",
				},
				Status:          models.RunErrored,
				WorkspaceID:     workspaceID,
				PlanID:          planID,
				ApplyID:         applyID,
				IsAssessmentRun: true,
			},
			assessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID: workspaceID,
			},
			expectAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID: "assessment-1",
				},
				WorkspaceID:          workspaceID,
				HasDrift:             false,
				RequiresNotification: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockPlans := db.NewMockPlans(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			mockStateVersions := db.NewMockStateVersions(t)

			if test.assessment != nil {
				mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, test.run.WorkspaceID).Return(test.assessment, nil).Maybe()
			}

			if test.workspace != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.run.WorkspaceID).
					Return(test.workspace, nil).Maybe()
			}

			if test.stateVersion != nil && test.workspace != nil {
				mockStateVersions.On("GetStateVersionByID", mock.Anything, test.workspace.CurrentStateVersionID).Return(test.stateVersion, nil).Maybe()
			}

			if test.stateVersion != nil && test.stateVersion.RunID != nil && *test.stateVersion.RunID == test.run.Metadata.ID {
				mockWorkspaceAssessments.On("DeleteWorkspaceAssessment", mock.Anything, test.assessment).Return(nil)
			}

			if test.plan != nil {
				mockPlans.On("GetPlanByID", mock.Anything, test.run.PlanID).
					Return(test.plan, nil).Maybe()
			}

			if test.expectAssessment != nil {
				matcher := mock.MatchedBy(func(assessment *models.WorkspaceAssessment) bool {
					return assessment.HasDrift == test.expectAssessment.HasDrift &&
						assessment.RequiresNotification == test.expectAssessment.RequiresNotification &&
						assessment.CompletedAtTimestamp != nil
				})
				mockWorkspaceAssessments.On("UpdateWorkspaceAssessment", mock.Anything, matcher).
					Return(test.expectAssessment, nil).Maybe()
			}

			dbClient := &db.Client{
				Plans:                mockPlans,
				Workspaces:           mockWorkspaces,
				WorkspaceAssessments: mockWorkspaceAssessments,
				StateVersions:        mockStateVersions,
			}

			mockLogger, _ := logger.NewForTest()

			runStateManager := state.NewRunStateManager(dbClient, mockLogger)
			handler := NewAssessmentRunHandler(mockLogger, dbClient, runStateManager)
			require.NotNil(t, handler)

			err := handler.handleRunEvent(ctx, state.RunEventType, nil, test.run)
			require.NoError(t, err)
		})
	}
}
