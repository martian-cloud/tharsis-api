package eventhandlers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	assessTestRunID       = "run-1"
	assessTestWorkspaceID = "ws-1"
)

func TestAssessmentRunHandler_HandleRunChanges_NoOpCases(t *testing.T) {
	type testCase struct {
		name string
		run  *models.Run
	}

	testCases := []testCase{
		{
			name: "incomplete run is ignored",
			run: &models.Run{
				Metadata:        models.ResourceMetadata{ID: assessTestRunID},
				WorkspaceID:     assessTestWorkspaceID,
				IsAssessmentRun: true,
				Status:          models.RunPlanning,
			},
		},
		{
			name: "completed speculative non-assessment run is ignored",
			run: &models.Run{
				Metadata:        models.ResourceMetadata{ID: assessTestRunID},
				WorkspaceID:     assessTestWorkspaceID,
				IsAssessmentRun: false,
				Apply:           nil, // speculative
				Status:          models.RunPlannedAndFinished,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logr, _ := logger.NewForTest()
			// No db sub-clients should be touched for these no-op cases.
			dbClient := &db.Client{WorkspaceAssessments: db.NewMockWorkspaceAssessments(t)}
			handler := NewAssessmentRunHandler(logr, dbClient)

			require.NoError(t, handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: test.run}}))
		})
	}
}

func TestAssessmentRunHandler_AssessmentRun_DriftUpdates(t *testing.T) {
	type testCase struct {
		name string
		// run plan summary drift count + final status.
		resourceDrift int32
		runStatus     models.RunStatus
		// existing assessment state.
		assessmentHasDrift bool
		// expectations.
		expectHasDrift             bool
		expectRequiresNotification bool
		expectRunIDSet             bool
	}

	testCases := []testCase{
		{
			name:                       "newly drifted increments and requires notification",
			resourceDrift:              3,
			runStatus:                  models.RunPlannedAndFinished,
			assessmentHasDrift:         false,
			expectHasDrift:             true,
			expectRequiresNotification: true,
			expectRunIDSet:             true,
		},
		{
			name:                       "still drifted does not require notification",
			resourceDrift:              2,
			runStatus:                  models.RunPlannedAndFinished,
			assessmentHasDrift:         true,
			expectHasDrift:             true,
			expectRequiresNotification: false,
			expectRunIDSet:             true,
		},
		{
			name:                       "drift cleared sets has-drift false",
			resourceDrift:              0,
			runStatus:                  models.RunPlannedAndFinished,
			assessmentHasDrift:         true,
			expectHasDrift:             false,
			expectRequiresNotification: false,
			expectRunIDSet:             true,
		},
		{
			name: "errored assessment run clears stale verdict and links the run",
			// Not RunPlannedAndFinished: the run produced no fresh verdict, so a previously
			// recorded drift is cleared rather than presented as this run's result, and the
			// errored run is still linked.
			resourceDrift:              5,
			runStatus:                  models.RunErrored,
			assessmentHasDrift:         true,
			expectHasDrift:             false,
			expectRequiresNotification: false,
			expectRunIDSet:             true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logr, _ := logger.NewForTest()

			assessment := &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{ID: "assess-1"},
				HasDrift: test.assessmentHasDrift,
			}

			mockAssess := db.NewMockWorkspaceAssessments(t)
			mockAssess.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, assessTestWorkspaceID).
				Return(assessment, nil)
			mockAssess.On("UpdateWorkspaceAssessment", mock.Anything, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
				if a.HasDrift != test.expectHasDrift {
					return false
				}
				if a.RequiresNotification != test.expectRequiresNotification {
					return false
				}
				if test.expectRunIDSet && (a.RunID == nil || *a.RunID != assessTestRunID) {
					return false
				}
				// CompletedAtTimestamp is always stamped.
				return a.CompletedAtTimestamp != nil
			})).Return(assessment, nil)

			dbClient := &db.Client{WorkspaceAssessments: mockAssess}
			handler := NewAssessmentRunHandler(logr, dbClient)

			run := &models.Run{
				Metadata:        models.ResourceMetadata{ID: assessTestRunID},
				WorkspaceID:     assessTestWorkspaceID,
				IsAssessmentRun: true,
				Status:          test.runStatus,
				Plan:            models.Plan{Summary: models.PlanSummary{ResourceDrift: test.resourceDrift}},
			}

			require.NoError(t, handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: run}}))
			mockAssess.AssertExpectations(t)
		})
	}
}

func TestAssessmentRunHandler_AssessmentRun_MissingAssessmentRecord(t *testing.T) {
	logr, observed := logger.NewForTest()

	mockAssess := db.NewMockWorkspaceAssessments(t)
	mockAssess.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, assessTestWorkspaceID).
		Return(nil, nil)
	// UpdateWorkspaceAssessment must not be called when the record is missing.

	dbClient := &db.Client{WorkspaceAssessments: mockAssess}
	handler := NewAssessmentRunHandler(logr, dbClient)

	run := &models.Run{
		Metadata:        models.ResourceMetadata{ID: assessTestRunID},
		WorkspaceID:     assessTestWorkspaceID,
		IsAssessmentRun: true,
		Status:          models.RunPlannedAndFinished,
	}

	require.NoError(t, handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: run}}))
	assert.NotZero(t, observed.Len(), "a missing assessment record should be logged")
}

func TestAssessmentRunHandler_RegularRun_InvalidateStaleAssessment(t *testing.T) {
	type testCase struct {
		name string
		// existing assessment; nil means none recorded.
		assessment *models.WorkspaceAssessment
		// workspace returned (nil = not found).
		workspace *models.Workspace
		// expectations
		expectGetWorkspace bool
		expectGetStateVer  bool
		stateVersion       *models.StateVersion
		expectDelete       bool
	}

	assessment := &models.WorkspaceAssessment{Metadata: models.ResourceMetadata{ID: "assess-1"}}

	testCases := []testCase{
		{
			name:               "no assessment recorded does nothing",
			assessment:         nil,
			expectGetWorkspace: false,
		},
		{
			name:               "workspace without current state version is left alone",
			assessment:         assessment,
			workspace:          &models.Workspace{CurrentStateVersionID: ""},
			expectGetWorkspace: true,
		},
		{
			name:               "state version produced by this run invalidates assessment",
			assessment:         assessment,
			workspace:          &models.Workspace{CurrentStateVersionID: "sv-1"},
			expectGetWorkspace: true,
			expectGetStateVer:  true,
			stateVersion:       &models.StateVersion{RunID: ptr.String(assessTestRunID)},
			expectDelete:       true,
		},
		{
			name:               "state version produced by a different run is kept",
			assessment:         assessment,
			workspace:          &models.Workspace{CurrentStateVersionID: "sv-1"},
			expectGetWorkspace: true,
			expectGetStateVer:  true,
			stateVersion:       &models.StateVersion{RunID: ptr.String("other-run")},
			expectDelete:       false,
		},
		{
			name:               "missing state version is a no-op",
			assessment:         assessment,
			workspace:          &models.Workspace{CurrentStateVersionID: "sv-1"},
			expectGetWorkspace: true,
			expectGetStateVer:  true,
			stateVersion:       nil,
			expectDelete:       false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logr, _ := logger.NewForTest()

			mockAssess := db.NewMockWorkspaceAssessments(t)
			mockAssess.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, assessTestWorkspaceID).
				Return(test.assessment, nil)

			mockWS := db.NewMockWorkspaces(t)
			mockSV := db.NewMockStateVersions(t)

			if test.expectGetWorkspace {
				mockWS.On("GetWorkspaceByID", mock.Anything, assessTestWorkspaceID).Return(test.workspace, nil)
			}
			if test.expectGetStateVer {
				mockSV.On("GetStateVersionByID", mock.Anything, "sv-1").Return(test.stateVersion, nil)
			}
			if test.expectDelete {
				mockAssess.On("DeleteWorkspaceAssessment", mock.Anything, test.assessment).Return(nil)
			}

			dbClient := &db.Client{
				WorkspaceAssessments: mockAssess,
				Workspaces:           mockWS,
				StateVersions:        mockSV,
			}
			handler := NewAssessmentRunHandler(logr, dbClient)

			// Non-assessment, non-speculative, completed run.
			run := &models.Run{
				Metadata:        models.ResourceMetadata{ID: assessTestRunID},
				WorkspaceID:     assessTestWorkspaceID,
				IsAssessmentRun: false,
				Apply:           &models.Apply{},
				Status:          models.RunApplied,
			}

			require.NoError(t, handler.HandleRunChanges(context.Background(), []types.RunChange{{Run: run}}))
			mockAssess.AssertExpectations(t)
			mockWS.AssertExpectations(t)
			mockSV.AssertExpectations(t)
		})
	}
}
