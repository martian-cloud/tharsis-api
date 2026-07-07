package workspace

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestAssessmentScheduler_execute(t *testing.T) {
	// Avoid real sleeps between paginated batches during tests.
	originalPageSleep := pageSleepInterval
	pageSleepInterval = 0
	defer func() { pageSleepInterval = originalPageSleep }()

	runLimit := 10
	minInterval := time.Hour

	// page describes a single batch returned by GetWorkspaces during one pass.
	type page struct {
		workspaces  []models.Workspace
		cursor      string
		hasNextPage bool
	}

	type testCase struct {
		name                      string
		inMaintenanceMode         bool
		assessmentsInProgress     int
		pages                     []page
		expectProcessCommandCalls int
	}

	enabled := func(id string) models.Workspace {
		return models.Workspace{Metadata: models.ResourceMetadata{ID: id}, EnableDriftDetection: ptr.Bool(true)}
	}
	disabled := func(id string) models.Workspace {
		return models.Workspace{Metadata: models.ResourceMetadata{ID: id}, EnableDriftDetection: ptr.Bool(false)}
	}

	testCases := []testCase{
		{
			name:              "skip checking workspaces since maintenance mode is enabled",
			inMaintenanceMode: true,
		},
		{
			name:  "single page with drift detection disabled creates no assessments",
			pages: []page{{workspaces: []models.Workspace{disabled("ws1"), disabled("ws2")}, cursor: "c1", hasNextPage: false}},
		},
		{
			name: "paginates through all pages in a single pass",
			pages: []page{
				{workspaces: []models.Workspace{disabled("ws1"), enabled("ws2")}, cursor: "c1", hasNextPage: true},
				{workspaces: []models.Workspace{enabled("ws3")}, cursor: "c2", hasNextPage: false},
			},
			expectProcessCommandCalls: 2,
		},
		{
			name:                  "stops the pass when the in-progress limit is reached",
			assessmentsInProgress: runLimit,
			pages: []page{
				// The first due workspace sees the live count at the limit and halts the pass
				// before the next page (hasNextPage=true) is fetched.
				{workspaces: []models.Workspace{enabled("ws1")}, cursor: "c1", hasNextPage: true},
			},
			expectProcessCommandCalls: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			mockCmdProcessor := engine.NewMockCmdProcessor(t)
			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			// The in-progress count is read once per pass to seed the create budget (skipped
			// entirely in maintenance mode, hence Maybe).
			mockWorkspaceAssessments.On("GetWorkspaceAssessments", mock.Anything, &db.GetWorkspaceAssessmentsInput{
				PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
				Filter:            &db.WorkspaceAssessmentFilter{InProgress: ptr.Bool(true)},
			}).Return(&db.WorkspaceAssessmentsResult{
				PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(int32(test.assessmentsInProgress))},
			}, nil).Maybe()

			workspaceSort := db.WorkspaceSortableFieldFullPathAsc
			for i, pg := range test.pages {
				var after *string
				if i > 0 {
					after = ptr.String(test.pages[i-1].cursor)
				}

				cursor := pg.cursor
				mockWorkspaces.On("GetWorkspaces", mock.Anything, &db.GetWorkspacesInput{
					Sort: &workspaceSort,
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(workspaceBatchSize),
						After: after,
					},
					Filter: &db.WorkspaceFilter{
						MinDurationSinceLastAssessment: &minInterval,
						Locked:                         ptr.Bool(false),
						HasStateVersion:                ptr.Bool(true),
					},
				}).Return(&db.WorkspacesResult{
					Workspaces: pg.workspaces,
					PageInfo: &pagination.PageInfo{
						HasNextPage: pg.hasNextPage,
						Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
							return ptr.String(cursor), nil
						},
					},
				}, nil).Once()

				for _, workspace := range pg.workspaces {
					if *workspace.EnableDriftDetection {
						mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspace.Metadata.ID).
							Return(nil, nil).Once()
					}
				}
			}

			if test.expectProcessCommandCalls > 0 {
				// The command's input fields are unexported, so we assert the processor is
				// invoked with a CreateAssessmentRun command rather than its contents.
				mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.AnythingOfType("*commands.CreateAssessmentRun")).
					Return(nil).Times(test.expectProcessCommandCalls)
			}

			dbClient := &db.Client{
				Transactions:         mockTransactions,
				Workspaces:           mockWorkspaces,
				WorkspaceAssessments: mockWorkspaceAssessments,
			}

			mockLogger, _ := logger.NewForTest()

			// ProcessCommand is mocked, so the command is never prepared/executed; the
			// factory's collaborators are unused and can be nil.
			cmdFactory := commands.NewFactory(mockLogger, dbClient, nil, nil, nil, "", nil, nil)

			scheduler := NewAssessmentScheduler(
				dbClient,
				mockLogger,
				mockCmdProcessor,
				cmdFactory,
				mockInheritedSettingsResolver,
				mockMaintenanceMonitor,
				minInterval,
				runLimit,
			)

			err := scheduler.execute(ctx)
			require.NoError(t, err)
		})
	}
}

func TestAssessmentScheduler_startWorkspaceAssessment(t *testing.T) {
	minInterval := time.Hour
	runLimit := 10
	now := time.Now()
	recent := now.Add(-time.Minute)
	stale := now.Add(-2 * models.AssessmentStaleTimeout)

	type testCase struct {
		name                string
		assessment          *models.WorkspaceAssessment
		inProgressCount     int
		expectCreate        bool
		expectLimitExceeded bool
	}

	testCases := []testCase{
		{
			name: "fresh in-progress assessment within the min interval is skipped",
			assessment: &models.WorkspaceAssessment{
				Metadata:           models.ResourceMetadata{ID: "wa1", Version: 1, LastUpdatedTimestamp: &recent},
				StartedAtTimestamp: recent,
			},
			expectCreate: false,
		},
		{
			name: "stale in-progress assessment is restarted when under the limit",
			assessment: &models.WorkspaceAssessment{
				Metadata:           models.ResourceMetadata{ID: "wa1", Version: 1, LastUpdatedTimestamp: &stale},
				StartedAtTimestamp: recent,
			},
			inProgressCount: 0,
			expectCreate:    true,
		},
		{
			name:                "due workspace is not created when the live count is at the limit",
			assessment:          nil, // never assessed, so due
			inProgressCount:     runLimit,
			expectCreate:        false,
			expectLimitExceeded: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			workspace := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}}

			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			mockCmdProcessor := engine.NewMockCmdProcessor(t)

			mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, "ws1").
				Return(test.assessment, nil).Once()

			// The live in-progress count is read for any workspace that isn't skipped before
			// the create decision.
			mockWorkspaceAssessments.On("GetWorkspaceAssessments", mock.Anything, &db.GetWorkspaceAssessmentsInput{
				PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
				Filter:            &db.WorkspaceAssessmentFilter{InProgress: ptr.Bool(true)},
			}).Return(&db.WorkspaceAssessmentsResult{
				PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(int32(test.inProgressCount))},
			}, nil).Maybe()

			if test.expectCreate {
				mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.AnythingOfType("*commands.CreateAssessmentRun")).
					Return(nil).Once()
			}

			dbClient := &db.Client{
				WorkspaceAssessments: mockWorkspaceAssessments,
			}
			mockLogger, _ := logger.NewForTest()
			cmdFactory := commands.NewFactory(mockLogger, dbClient, nil, nil, nil, "", nil, nil)

			scheduler := NewAssessmentScheduler(
				dbClient,
				mockLogger,
				mockCmdProcessor,
				cmdFactory,
				namespace.NewMockInheritedSettingResolver(t),
				maintenance.NewMockMonitor(t),
				minInterval,
				runLimit,
			)

			limitExceeded, err := scheduler.startWorkspaceAssessment(ctx, workspace)
			require.NoError(t, err)
			assert.Equal(t, test.expectLimitExceeded, limitExceeded)
		})
	}
}

func TestAssessmentScheduler_startWorkspaceAssessmentErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		processErr  error
		expectError bool
	}{
		{
			name:        "optimistic lock errors are swallowed",
			processErr:  errors.New("conflict", errors.WithErrorCode(errors.EOptimisticLock)),
			expectError: false,
		},
		{
			name:        "conflict errors are swallowed",
			processErr:  errors.New("in progress", errors.WithErrorCode(errors.EConflict)),
			expectError: false,
		},
		{
			name:        "internal errors are surfaced",
			processErr:  errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			workspace := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}}

			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			mockCmdProcessor := engine.NewMockCmdProcessor(t)

			// No existing assessment, so the scheduler goes straight to creating one.
			mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, "ws1").
				Return(nil, nil).Once()
			// Under the limit, so the create proceeds.
			mockWorkspaceAssessments.On("GetWorkspaceAssessments", mock.Anything, mock.Anything).
				Return(&db.WorkspaceAssessmentsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil).Once()
			mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.AnythingOfType("*commands.CreateAssessmentRun")).
				Return(test.processErr).Once()

			dbClient := &db.Client{
				WorkspaceAssessments: mockWorkspaceAssessments,
			}
			mockLogger, _ := logger.NewForTest()
			cmdFactory := commands.NewFactory(mockLogger, dbClient, nil, nil, nil, "", nil, nil)

			scheduler := NewAssessmentScheduler(
				dbClient,
				mockLogger,
				mockCmdProcessor,
				cmdFactory,
				namespace.NewMockInheritedSettingResolver(t),
				maintenance.NewMockMonitor(t),
				time.Hour,
				10,
			)

			limitExceeded, err := scheduler.startWorkspaceAssessment(ctx, workspace)
			assert.False(t, limitExceeded)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAssessmentScheduler_isDriftDetectionEnabledCaching(t *testing.T) {
	ctx := context.Background()

	mockResolver := namespace.NewMockInheritedSettingResolver(t)
	// The inherited setting is resolved once for the shared group path; the second
	// workspace in the same group hits the per-pass cache.
	mockResolver.On("GetDriftDetectionEnabled", mock.Anything, mock.Anything).
		Return(&namespace.DriftDetectionEnabledSetting{NamespacePath: "grp", Value: true}, nil).Once()

	scheduler := NewAssessmentScheduler(nil, nil, nil, nil, mockResolver, nil, time.Hour, 10)

	cache := map[string]bool{}
	// EnableDriftDetection is nil, so the resolver is consulted. Both workspaces share the
	// group path "grp".
	wsA := &models.Workspace{Metadata: models.ResourceMetadata{ID: "a"}, FullPath: "grp/ws-a"}
	wsB := &models.Workspace{Metadata: models.ResourceMetadata{ID: "b"}, FullPath: "grp/ws-b"}

	enabledA, err := scheduler.isDriftDetectionEnabled(ctx, wsA, cache)
	require.NoError(t, err)
	assert.True(t, enabledA)

	enabledB, err := scheduler.isDriftDetectionEnabled(ctx, wsB, cache)
	require.NoError(t, err)
	assert.True(t, enabledB)
}
