package workspace

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestAssessmentScheduler_execute(t *testing.T) {
	runLimit := 10
	minInterval := time.Hour

	type testCase struct {
		name                  string
		inMaintenanceMode     bool
		cursor                *string
		expectCursor          *string
		assessmentsInProgress int
		workspaces            []models.Workspace
		existingAssessments   []models.WorkspaceAssessment
	}

	testCases := []testCase{
		{
			name:              "skip checking workspaces since maintenance mode is enabled",
			inMaintenanceMode: true,
			cursor:            ptr.String("c1"),
			expectCursor:      ptr.String("c1"),
		},
		{
			name:                  "skip checking workspaces since run limit has been reached",
			assessmentsInProgress: runLimit,
			cursor:                ptr.String("c1"),
			expectCursor:          ptr.String("c1"),
			workspaces: []models.Workspace{
				{
					Metadata: models.ResourceMetadata{
						ID: "ws1",
					},
					EnableDriftDetection: ptr.Bool(false),
				},
				{
					Metadata: models.ResourceMetadata{
						ID: "ws2",
					},
					EnableDriftDetection: ptr.Bool(false),
				},
			},
		},
		{
			name:                  "skip checking workspaces since auto drift detection is disabled for them",
			assessmentsInProgress: 0,
			cursor:                ptr.String("c1"),
			expectCursor:          ptr.String("c2"),
			workspaces: []models.Workspace{
				{
					Metadata: models.ResourceMetadata{
						ID: "ws1",
					},
					EnableDriftDetection: ptr.Bool(false),
				},
				{
					Metadata: models.ResourceMetadata{
						ID: "ws2",
					},
					EnableDriftDetection: ptr.Bool(false),
				},
			},
		},
		{
			name:                  "check workspace that has auto drift detection enabled",
			assessmentsInProgress: 0,
			cursor:                ptr.String("c1"),
			expectCursor:          ptr.String("c2"),
			workspaces: []models.Workspace{
				{
					Metadata: models.ResourceMetadata{
						ID: "ws1",
					},
					EnableDriftDetection: ptr.Bool(false),
				},
				{
					Metadata: models.ResourceMetadata{
						ID: "ws2",
					},
					EnableDriftDetection: ptr.Bool(true),
				},
				{
					Metadata: models.ResourceMetadata{
						ID: "ws3",
					},
					EnableDriftDetection: ptr.Bool(true),
				},
			},
			existingAssessments: []models.WorkspaceAssessment{
				{
					Metadata: models.ResourceMetadata{
						ID:      "wa1",
						Version: 1,
					},
					WorkspaceID: "ws3",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)

			mockRunService := run.NewMockService(t)

			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			if test.workspaces != nil {
				workspaceSort := db.WorkspaceSortableFieldFullPathAsc
				mockWorkspaces.On("GetWorkspaces", mock.Anything, &db.GetWorkspacesInput{
					Sort: &workspaceSort,
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(workspaceBatchSize), // Get the next batch of workspaces
						After: test.cursor,
					},
					Filter: &db.WorkspaceFilter{
						MinDurationSinceLastAssessment: &minInterval,
						Locked:                         ptr.Bool(false),
						HasStateVersion:                ptr.Bool(true),
					},
				}).Return(&db.WorkspacesResult{
					Workspaces: test.workspaces,
					PageInfo: &pagination.PageInfo{
						HasNextPage: true,
						Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
							return test.expectCursor, nil
						},
					},
				}, nil).Once()

				for _, workspace := range test.workspaces {
					mockInheritedSettingsResolver.On("GetDriftDetectionEnabled", mock.Anything, &workspace).Return(&namespace.DriftDetectionEnabledSetting{
						Value: *workspace.EnableDriftDetection,
					}, nil).Maybe()

					var assessment *models.WorkspaceAssessment
					for _, a := range test.existingAssessments {
						if a.WorkspaceID == workspace.Metadata.ID {
							assessment = &a
							break
						}
					}

					if *workspace.EnableDriftDetection {
						mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspace.Metadata.ID).
							Return(assessment, nil).Once()

						mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
						mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
						mockTransactions.On("CommitTx", mock.Anything).Return(nil)

						var latestAssessmentVersion *int
						if assessment != nil {
							latestAssessmentVersion = &assessment.Metadata.Version
						}

						mockRunService.On("CreateAssessmentRunForWorkspace", mock.Anything, &run.CreateAssessmentRunForWorkspaceInput{
							WorkspaceID:             workspace.Metadata.ID,
							LatestAssessmentVersion: latestAssessmentVersion,
						}).Return(nil, nil).Once()

						mockWorkspaceAssessments.On("GetWorkspaceAssessments", mock.Anything, &db.GetWorkspaceAssessmentsInput{
							PaginationOptions: &pagination.Options{
								First: ptr.Int32(0),
							},
							Filter: &db.WorkspaceAssessmentFilter{
								InProgress: ptr.Bool(true),
							},
						}).Return(&db.WorkspaceAssessmentsResult{
							PageInfo: &pagination.PageInfo{
								TotalCount: 0,
							},
						}, nil)
					}
				}
			}

			dbClient := &db.Client{
				Transactions:         mockTransactions,
				Workspaces:           mockWorkspaces,
				WorkspaceAssessments: mockWorkspaceAssessments,
			}

			mockLogger, _ := logger.NewForTest()

			scheduler := NewAssessmentScheduler(
				dbClient,
				mockLogger,
				mockRunService,
				mockInheritedSettingsResolver,
				mockMaintenanceMonitor,
				minInterval,
				runLimit,
			)

			nextCursor, err := scheduler.execute(ctx, test.cursor)
			require.NoError(t, err)

			assert.Equal(t, test.expectCursor, nextCursor)
		})
	}
}
