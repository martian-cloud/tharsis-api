package workspace

import (
	"context"
	goerrors "errors"
	"strings"
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
	"go.uber.org/zap/zapcore"
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
						AssessmentDue:   ptr.Bool(true),
						Locked:          ptr.Bool(false),
						HasStateVersion: ptr.Bool(true),
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
		{
			name: "a recorded failure's backoff is honored even though the min interval has elapsed",
			assessment: &models.WorkspaceAssessment{
				Metadata:             models.ResourceMetadata{ID: "wa1", Version: 1},
				StartedAtTimestamp:   stale, // old enough to satisfy the min interval on its own
				CompletedAtTimestamp: &stale,
				AvailableAtTimestamp: ptr.Time(now.Add(time.Hour)), // backoff still in effect
			},
			expectCreate: false,
		},
		{
			name: "a recorded failure whose backoff has elapsed is retried",
			assessment: &models.WorkspaceAssessment{
				Metadata:             models.ResourceMetadata{ID: "wa1", Version: 1},
				StartedAtTimestamp:   stale,
				CompletedAtTimestamp: &stale,
				AvailableAtTimestamp: ptr.Time(stale), // backoff window already passed
			},
			inProgressCount: 0,
			expectCreate:    true,
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

			limitExceeded, _, err := scheduler.startWorkspaceAssessment(ctx, workspace)
			require.NoError(t, err)
			assert.Equal(t, test.expectLimitExceeded, limitExceeded)
		})
	}
}

// TestAssessmentScheduler_startWorkspaceAssessmentErrorHandling asserts the contract that
// startWorkspaceAssessment propagates every error from the command processor unchanged.
// Deciding what each error means is the caller's job (handleAssessmentError), so that the
// policy lives in exactly one place; the expected disposition is asserted here too, to keep
// the two halves of that contract described together.
func TestAssessmentScheduler_startWorkspaceAssessmentErrorHandling(t *testing.T) {
	testCases := []struct {
		name              string
		processErr        error
		expectDisposition assessmentErrorDisposition
	}{
		{
			name:              "optimistic lock errors propagate and are classified as ignorable",
			processErr:        errors.New("conflict", errors.WithErrorCode(errors.EOptimisticLock)),
			expectDisposition: assessmentErrorIgnore,
		},
		{
			name:              "conflict errors propagate and are classified as ignorable",
			processErr:        errors.New("in progress", errors.WithErrorCode(errors.EConflict)),
			expectDisposition: assessmentErrorIgnore,
		},
		{
			name:              "module not found propagates and is classified as recordable",
			processErr:        errors.New("module not found", errors.WithErrorCode(errors.ENotFound)),
			expectDisposition: assessmentErrorRecord,
		},
		{
			name:              "registry auth failure propagates and is classified as recordable",
			processErr:        errors.New("token not authorized", errors.WithErrorCode(errors.EInvalid)),
			expectDisposition: assessmentErrorRecord,
		},
		{
			name:              "internal errors propagate and stay unexpected",
			processErr:        errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectDisposition: assessmentErrorUnexpected,
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

			limitExceeded, _, err := scheduler.startWorkspaceAssessment(ctx, workspace)
			assert.False(t, limitExceeded)
			require.Error(t, err, "every error must reach the caller so one place classifies it")
			assert.Equal(t, test.expectDisposition, classifyAssessmentError(err))
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

func TestClassifyAssessmentError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		expect assessmentErrorDisposition
	}{
		{
			name:   "optimistic lock is a benign race with another instance",
			err:    errors.New("stale version", errors.WithErrorCode(errors.EOptimisticLock)),
			expect: assessmentErrorIgnore,
		},
		{
			name:   "conflict is a benign race with another instance",
			err:    errors.New("already in progress", errors.WithErrorCode(errors.EConflict)),
			expect: assessmentErrorIgnore,
		},
		{
			name:   "not found covers a deleted registry module",
			err:    errors.New("module not found for source x", errors.WithErrorCode(errors.ENotFound)),
			expect: assessmentErrorRecord,
		},
		{
			name:   "invalid covers an external registry rejecting the stored credential",
			err:    errors.New("token is not authorized", errors.WithErrorCode(errors.EInvalid)),
			expect: assessmentErrorRecord,
		},
		{
			name:   "forbidden covers a managed identity rule the system cannot satisfy",
			err:    errors.New("rule not satisfied", errors.WithErrorCode(errors.EForbidden)),
			expect: assessmentErrorRecord,
		},
		{
			name:   "internal stays unexpected",
			err:    errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expect: assessmentErrorUnexpected,
		},
		{
			name:   "unauthorized stays unexpected",
			err:    errors.New("no caller", errors.WithErrorCode(errors.EUnauthorized)),
			expect: assessmentErrorUnexpected,
		},
		{
			name:   "service unavailable stays unexpected",
			err:    errors.New("down", errors.WithErrorCode(errors.EServiceUnavailable)),
			expect: assessmentErrorUnexpected,
		},
		{
			// ErrorCode reports an uncoded error as EInternal, so it must land in the default branch.
			name:   "an uncoded error stays unexpected",
			err:    goerrors.New("plain error"),
			expect: assessmentErrorUnexpected,
		},
		{
			name:   "a nil error is not mistaken for a benign condition",
			err:    nil,
			expect: assessmentErrorUnexpected,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, classifyAssessmentError(test.err))
		})
	}
}

// TestClassifyAssessmentError_SurvivesWrapping guards the property the scheduler depends on:
// these errors reach it through several errors.Wrap layers (Prepare -> ResolveModule -> the
// registry source), and the classification must still follow the innermost coded error.
func TestClassifyAssessmentError_SurvivesWrapping(t *testing.T) {
	inner := errors.New("module not found for source x", errors.WithErrorCode(errors.ENotFound))
	wrapped := errors.Wrap(errors.Wrap(inner, "failed to resolve module source"), "failed to prepare command")

	assert.Equal(t, assessmentErrorRecord, classifyAssessmentError(wrapped))

	// A wrap that restamps the code wins, which is how ResolveModule turns a registry 401 into
	// EInvalid.
	restamped := errors.Wrap(goerrors.New("401 Unauthorized"), "failed to resolve module source",
		errors.WithErrorCode(errors.EInvalid))

	assert.Equal(t, assessmentErrorRecord, classifyAssessmentError(restamped))
}

// TestAssessmentScheduler_handleAssessmentErrorLogLevel is the heart of the issue: the three
// unactionable conditions must stop producing ERROR lines, while genuine faults keep producing
// them.
func TestAssessmentScheduler_handleAssessmentErrorLogLevel(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		// expectRecord is set for the three conditions classified as assessmentErrorRecord,
		// which write to the assessment record and need a db client stubbed for it; the other
		// dispositions must leave it untouched. Kept separate from expectLevel because
		// zapcore.InfoLevel is zap's zero value, so inferring this from expectLevel would also
		// match every case that leaves it unset.
		expectRecord bool
		expectCount  int
		expectLevel  zapcore.Level
	}{
		{
			name:        "optimistic lock logs nothing",
			err:         errors.New("conflict", errors.WithErrorCode(errors.EOptimisticLock)),
			expectCount: 0,
		},
		{
			name:        "conflict logs nothing",
			err:         errors.New("in progress", errors.WithErrorCode(errors.EConflict)),
			expectCount: 0,
		},
		{
			name:         "a deleted module logs at info, not error",
			err:          errors.New("module not found for source x", errors.WithErrorCode(errors.ENotFound)),
			expectRecord: true,
			expectCount:  1,
			expectLevel:  zapcore.InfoLevel,
		},
		{
			name:         "an expired registry credential logs at info, not error",
			err:          errors.New("token is not authorized", errors.WithErrorCode(errors.EInvalid)),
			expectRecord: true,
			expectCount:  1,
			expectLevel:  zapcore.InfoLevel,
		},
		{
			name:         "an unsatisfiable managed identity rule logs at info, not error",
			err:          errors.New("rule not satisfied", errors.WithErrorCode(errors.EForbidden)),
			expectRecord: true,
			expectCount:  1,
			expectLevel:  zapcore.InfoLevel,
		},
		{
			name:        "a genuine fault still logs at error",
			err:         errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectCount: 1,
			expectLevel: zapcore.ErrorLevel,
		},
		{
			name:        "an uncoded error still logs at error",
			err:         goerrors.New("plain failure"),
			expectCount: 1,
			expectLevel: zapcore.ErrorLevel,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			testLogger, logs := logger.NewForTest()

			// The record disposition writes the reason to the assessment record, so those cases
			// need a db client. Build one unconditionally and let mockery assert that the other
			// dispositions leave it untouched.
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			if test.expectRecord {
				mockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, "ws1").
					Return(nil, nil).Once()
				mockWorkspaceAssessments.On("CreateWorkspaceAssessment", mock.Anything, mock.Anything).
					Return(&models.WorkspaceAssessment{}, nil).Once()
			}

			dbClient := &db.Client{WorkspaceAssessments: mockWorkspaceAssessments}
			scheduler := NewAssessmentScheduler(dbClient, testLogger, nil, nil, nil, nil, time.Hour, 10)
			workspace := &models.Workspace{
				Metadata: models.ResourceMetadata{ID: "ws1"},
				FullPath: "grp/ws-a",
			}

			scheduler.handleAssessmentError(context.Background(), workspace, nil, test.err)

			require.Equal(t, test.expectCount, logs.Len())
			if test.expectCount == 0 {
				return
			}

			entry := logs.All()[0]
			assert.Equal(t, test.expectLevel, entry.Level)
			assert.Contains(t, entry.Message, "grp/ws-a", "the log must name the affected workspace")
		})
	}
}

// TestAssessmentBackoffInterval covers the escalation Brandon.Breuil asked for on the MR: a
// workspace that keeps failing should be reconsidered less often over time, not retried at a
// fixed interval forever, but it must still be bounded so it isn't abandoned.
func TestAssessmentBackoffInterval(t *testing.T) {
	base := time.Hour

	testCases := []struct {
		name                string
		consecutiveFailures int
		expect              time.Duration
	}{
		{name: "zero failures is treated the same as the first", consecutiveFailures: 0, expect: base},
		{name: "first failure uses the base interval", consecutiveFailures: 1, expect: base},
		{name: "second consecutive failure doubles it", consecutiveFailures: 2, expect: 2 * base},
		{name: "third consecutive failure doubles again", consecutiveFailures: 3, expect: 4 * base},
		{name: "fourth consecutive failure doubles again", consecutiveFailures: 4, expect: 8 * base},
		{name: "fifth consecutive failure hits the cap", consecutiveFailures: 5, expect: 16 * base},
		{name: "well past the cap stays at the cap", consecutiveFailures: 100, expect: 16 * base},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, assessmentBackoffInterval(base, test.consecutiveFailures))
		})
	}
}

func TestAssessmentScheduler_recordAssessmentFailure(t *testing.T) {
	const (
		workspaceID = "ws1"
		message     = "module not found for source registry/example/aws"
	)

	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: workspaceID},
		FullPath: "grp/ws-a",
	}

	t.Run("creates a record when the workspace has never been assessed", func(t *testing.T) {
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(nil, nil).Once()
		mockAssessments.On("CreateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			// completed_at must be set: it is what drops the workspace out of the scheduler's
			// candidate set until the interval elapses. The first failure uses the base
			// interval (no backoff multiplier yet) and starts the consecutive-failure count at 1.
			return a.WorkspaceID == workspaceID &&
				a.ErrorMessage != nil && *a.ErrorMessage == message &&
				a.CompletedAtTimestamp != nil &&
				!a.StartedAtTimestamp.IsZero() &&
				a.ConsecutiveFailures == 1 &&
				a.AvailableAtTimestamp != nil &&
				a.AvailableAtTimestamp.Sub(*a.CompletedAtTimestamp) == time.Hour
		})).Return(&models.WorkspaceAssessment{}, nil).Once()

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, nil, message)

		assert.Equal(t, 0, logs.Len(), "a successful record logs nothing of its own")
	})

	t.Run("updates an existing record without disturbing the drift verdict", func(t *testing.T) {
		previouslyCompleted := time.Now().UTC().Add(-2 * time.Hour)
		existing := &models.WorkspaceAssessment{
			Metadata:             models.ResourceMetadata{ID: "assessment1", Version: 3},
			WorkspaceID:          workspaceID,
			HasDrift:             true,
			RequiresNotification: true,
			RunID:                ptr.String("run1"),
			CompletedAtTimestamp: &previouslyCompleted,
		}

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(existing, nil).Once()
		mockAssessments.On("UpdateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			// A drifted workspace must not be reported as clean just because this attempt could
			// not run, and a failed attempt is not news worth notifying about. The record had no
			// prior consecutive failures (its zero value), so this is treated as the first.
			return a.HasDrift &&
				!a.RequiresNotification &&
				a.ErrorMessage != nil && *a.ErrorMessage == message &&
				a.CompletedAtTimestamp != nil &&
				a.ConsecutiveFailures == 1 &&
				a.AvailableAtTimestamp != nil &&
				a.AvailableAtTimestamp.Sub(*a.CompletedAtTimestamp) == time.Hour
		})).Return(existing, nil).Once()

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, ptr.Int(3), message)

		assert.Equal(t, 0, logs.Len())
	})

	t.Run("escalates backoff on each further consecutive failure", func(t *testing.T) {
		// Three prior consecutive failures; this is the fourth, so backoff should be 8x the
		// base interval (2^(4-1)), per TestAssessmentBackoffInterval.
		previouslyCompleted := time.Now().UTC().Add(-2 * time.Hour)
		existing := &models.WorkspaceAssessment{
			Metadata:             models.ResourceMetadata{ID: "assessment1", Version: 3},
			WorkspaceID:          workspaceID,
			ConsecutiveFailures:  3,
			CompletedAtTimestamp: &previouslyCompleted,
		}

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(existing, nil).Once()
		mockAssessments.On("UpdateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			return a.ConsecutiveFailures == 4 &&
				a.AvailableAtTimestamp != nil &&
				a.AvailableAtTimestamp.Sub(*a.CompletedAtTimestamp) == 8*time.Hour
		})).Return(existing, nil).Once()

		testLogger, _ := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, ptr.Int(3), message)
	})

	t.Run("tolerates another instance creating the record first", func(t *testing.T) {
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(nil, nil).Once()
		mockAssessments.On("CreateWorkspaceAssessment", mock.Anything, mock.Anything).
			Return(nil, errors.New("already exists", errors.WithErrorCode(errors.EConflict))).Once()

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, nil, message)

		assert.Equal(t, 0, logs.Len(), "losing the create race is expected, not an error")
	})

	t.Run("skips the write when the row has moved on to a newer, in-progress attempt", func(t *testing.T) {
		// Another writer (another instance, or a user clicking "Run Drift Detection") started a
		// real, still-running attempt after the one that just failed read the row. Overwriting
		// it would misreport that running attempt as failed, and would let the scheduler
		// re-select the workspace once backoff elapses even though it has not finished.
		recentlyStarted := time.Now().UTC().Add(-time.Minute)
		newerInProgress := &models.WorkspaceAssessment{
			Metadata:           models.ResourceMetadata{ID: "assessment1", Version: 4},
			WorkspaceID:        workspaceID,
			StartedAtTimestamp: recentlyStarted,
			// CompletedAtTimestamp left nil: still in progress, and recent enough not to be
			// considered stale.
		}

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(newerInProgress, nil).Once()
		// No Create or Update call is stubbed: mockery fails the test if either is invoked.

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		// expectedVersion of 3 is what the failed attempt read before the other writer moved
		// the row to version 4.
		scheduler.recordAssessmentFailure(context.Background(), workspace, ptr.Int(3), message)

		assert.Equal(t, 0, logs.Len(), "skipping a stale write is not itself a failure")
	})

	t.Run("skips the write when no row was read but one exists now", func(t *testing.T) {
		// Symmetric to the create branch's EConflict handling: another writer created the row
		// after the failed attempt saw none, so there is nothing for this stale failure to do.
		recentlyStarted := time.Now().UTC().Add(-time.Minute)
		createdSinceRead := &models.WorkspaceAssessment{
			Metadata:           models.ResourceMetadata{ID: "assessment1", Version: 1},
			WorkspaceID:        workspaceID,
			StartedAtTimestamp: recentlyStarted,
		}

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(createdSinceRead, nil).Once()

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, nil, message)

		assert.Equal(t, 0, logs.Len())
	})

	t.Run("losing an optimistic-lock race on update logs at info, not error", func(t *testing.T) {
		previouslyCompleted := time.Now().UTC().Add(-2 * time.Hour)
		existing := &models.WorkspaceAssessment{
			Metadata:             models.ResourceMetadata{ID: "assessment1", Version: 3},
			WorkspaceID:          workspaceID,
			CompletedAtTimestamp: &previouslyCompleted,
		}

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(existing, nil).Once()
		mockAssessments.On("UpdateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			return a.Metadata.Version == 3
		})).Return(nil, db.ErrOptimisticLockError).Once()
		// There is no retry: a concurrent writer (another instance, or a user-triggered run)
		// already left something valid on the row, and the next scheduler pass will reconsider
		// the workspace if it's genuinely still failing. Same benign race as the create branch
		// tolerates, so it's logged at info rather than treated as a fault.

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, ptr.Int(3), message)

		require.Equal(t, 1, logs.Len())
		entry := logs.All()[0]
		assert.Equal(t, zapcore.InfoLevel, entry.Level)
		assert.Contains(t, entry.Message, "grp/ws-a")
	})

	t.Run("truncates an over-long message before writing", func(t *testing.T) {
		longMessage := strings.Repeat("x", models.MaxAssessmentErrorMessageLength*3)

		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(nil, nil).Once()
		mockAssessments.On("CreateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			if a.ErrorMessage == nil {
				return false
			}
			// What reaches the database must satisfy the model's own validation.
			return len(*a.ErrorMessage) <= models.MaxAssessmentErrorMessageLength && a.Validate() == nil
		})).Return(&models.WorkspaceAssessment{}, nil).Once()

		testLogger, _ := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, nil, longMessage)
	})

	t.Run("a database fault is logged and swallowed so the pass continues", func(t *testing.T) {
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, workspaceID).
			Return(nil, errors.New("connection refused", errors.WithErrorCode(errors.EInternal))).Once()

		testLogger, logs := logger.NewForTest()
		scheduler := NewAssessmentScheduler(
			&db.Client{WorkspaceAssessments: mockAssessments}, testLogger, nil, nil, nil, nil, time.Hour, 10)

		scheduler.recordAssessmentFailure(context.Background(), workspace, nil, message)

		require.Equal(t, 1, logs.Len())
		entry := logs.All()[0]
		assert.Equal(t, zapcore.ErrorLevel, entry.Level, "failing to write the record is a real fault")
		assert.Contains(t, entry.Message, "grp/ws-a")
	})
}
