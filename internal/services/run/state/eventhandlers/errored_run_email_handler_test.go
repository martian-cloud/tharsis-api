package eventhandlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	pluginemail "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewErroredRunEmailHandler(t *testing.T) {
	logger := logger.New()
	dbClient := &db.Client{}
	runStateManager := &state.RunStateManager{}
	taskManager := asynctask.NewManager(time.Duration(1) * time.Second)
	emailClient := email.NewClient(&pluginemail.NoopProvider{}, taskManager, dbClient, logger, "test-url", "test-footer")
	expectHandler := &ErroredRunEmailHandler{
		logger:          logger,
		dbClient:        dbClient,
		runStateManager: runStateManager,
		emailClient:     emailClient,
	}
	require.NotNil(t, expectHandler)

	actualHandler := NewErroredRunEmailHandler(logger, dbClient, runStateManager, emailClient)

	assert.NotNil(t, actualHandler)
	assert.Equal(t, expectHandler, actualHandler)
}

// TestHandleErroredRunEvent also tests RegisterHandlers, because we can't get to the handler without that.
func TestHandleErroredRunEvent(t *testing.T) {
	userEmail := "user-email@example.invalid"

	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
	}

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run-id",
		},
		CreatedBy:   userEmail,
		Status:      models.RunErrored,
		WorkspaceID: workspace.Metadata.ID,
	}

	user := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-id",
		},
		Email: userEmail,
	}

	expectError := fmt.Errorf("test error")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockCaller := auth.NewMockCaller(t)
	mockRuns := db.NewMockRuns(t)
	mockTransactions := db.NewMockTransactions(t)
	mockUsers := db.NewMockUsers(t)
	mockWorkspaces := db.NewMockWorkspaces(t)

	mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
	mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

	mockCaller.On("GetSubject").Return("testSubject")

	mockRuns.On("GetRun", mock.Anything, run.Metadata.ID).
		Return(run, nil)

	mockRuns.On("UpdateRun", mock.Anything, run).
		Return(run, nil)

	mockUsers.On("GetUserByEmail", mock.Anything, userEmail).
		Return(user, expectError)

	dbClient := &db.Client{
		Runs:         mockRuns,
		Users:        mockUsers,
		Transactions: mockTransactions,
		Workspaces:   mockWorkspaces,
	}

	logger := logger.New()
	runStateManager := state.NewRunStateManager(dbClient, logger)
	taskManager := asynctask.NewManager(time.Duration(1) * time.Second)
	emailClient := email.NewClient(&pluginemail.NoopProvider{}, taskManager, dbClient, logger, "test-url", "test-footer")
	handler := NewErroredRunEmailHandler(logger, dbClient, runStateManager, emailClient)
	require.NotNil(t, handler)

	// Do the registration.
	handler.RegisterHandlers()

	// Update a run to see whether registration actually worked and that the handler was called.
	updatedRun, err := runStateManager.UpdateRun(auth.WithCaller(ctx, mockCaller), run)
	assert.Nil(t, err)
	assert.Equal(t, run, updatedRun)

	// Verification of the handler is accomplished by the call to GetUserByEmail
	// by the error from GetUserByEmail having been consumed,
	// and by the lack of a call to GetWorkspaceByID.
}

func TestInternalHandleErroredRunEvent(t *testing.T) {
	userEmail := "user-email@example.invalid"
	moduleSource := "module-source"
	moduleVersion := "1.2.3"
	planGoodID := "plan-good-id"
	planBadID := "plan-bad-id"
	applyGoodID := "apply-good-id"
	applyBadID := "apply-bad-id"
	planErrorMessage := "synthetic plan error message"
	applyErrorMessage := "synthetic apply error message"
	internalRunID := "run-id"
	globalRunID := gid.NewGlobalID(gid.RunType, internalRunID).String() // "Ul9ydW4taWQ"

	ignoreLogs := map[string]interface{}{
		"Updated a run.":  nil,
		"Updated a job.":  nil,
		"Updated a plan.": nil,
	}

	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
	}

	user := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-id",
		},
		Email: userEmail,
	}

	type testCase struct {
		name                    string
		injectGetUserError      error
		injectGetWorkspaceError error
		run                     *models.Run
		expectSendMailInput     *email.SendMailInput
		expectLog               bool
	}

	testCases := []testCase{
		{
			name: "run status is not errored; expect no logged error, no sendmail",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunApplied,
				WorkspaceID: workspace.Metadata.ID,
			},
		},
		{
			name: "created-by is not a user (no '@'); expect no logged error, no sendmail",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   "service-account",
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
			},
		},
		{
			name:               "GetUserByEmail error, expect logged error, no sendmail",
			injectGetUserError: fmt.Errorf("error in GetUserByEmail"),
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
			},
			expectLog: true,
		},
		{
			name:                    "GetWorkspaceByID error, expect logged error, no sendmail",
			injectGetWorkspaceError: fmt.Errorf("error in GetWorkspaceByID"),
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
			},
			expectLog: true,
		},
		{
			name: "speculative, error in plan",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				PlanID:      planBadID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis speculative plan failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Speculative Plan Failed",
					CreatedBy:    userEmail,
					ErrorMessage: planErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.PlanStage,
				},
			},
		},
		{
			name: "speculative destroy, error in plan",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				IsDestroy:   true,
				PlanID:      planBadID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis speculative destroy plan failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Speculative Destroy Plan Failed",
					CreatedBy:    userEmail,
					ErrorMessage: planErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.PlanStage,
				},
			},
		},
		{
			name: "error in plan",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				PlanID:      planBadID,
				ApplyID:     applyGoodID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis plan failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Plan Failed",
					CreatedBy:    userEmail,
					ErrorMessage: planErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.PlanStage,
				},
			},
		},
		{
			name: "error in apply",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				PlanID:      planGoodID,
				ApplyID:     applyBadID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis apply failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Apply Failed",
					CreatedBy:    userEmail,
					ErrorMessage: applyErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.ApplyStage,
				},
			},
		},
		{
			name: "error in apply with module source and version",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:     userEmail,
				Status:        models.RunErrored,
				WorkspaceID:   workspace.Metadata.ID,
				PlanID:        planGoodID,
				ApplyID:       applyBadID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis apply failed",
				Builder: &builder.FailedRunEmail{
					Title:         "Apply Failed",
					ModuleSource:  &moduleSource,
					ModuleVersion: &moduleVersion,
					CreatedBy:     userEmail,
					ErrorMessage:  applyErrorMessage,
					RunID:         globalRunID,
					RunStage:      builder.ApplyStage,
				},
			},
		},
		{
			name: "destroy, error in plan",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				IsDestroy:   true,
				PlanID:      planBadID,
				ApplyID:     applyGoodID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis destroy plan failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Destroy Plan Failed",
					CreatedBy:    userEmail,
					ErrorMessage: planErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.PlanStage,
				},
			},
		},
		{
			name: "destroy, error in apply",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: internalRunID,
				},
				CreatedBy:   userEmail,
				Status:      models.RunErrored,
				WorkspaceID: workspace.Metadata.ID,
				IsDestroy:   true,
				PlanID:      planGoodID,
				ApplyID:     applyBadID,
			},
			expectSendMailInput: &email.SendMailInput{
				UsersIDs: []string{user.Metadata.ID},
				Subject:  "Tharsis destroy failed",
				Builder: &builder.FailedRunEmail{
					Title:        "Destroy Failed",
					CreatedBy:    userEmail,
					ErrorMessage: applyErrorMessage,
					RunID:        globalRunID,
					RunStage:     builder.ApplyStage,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockPlans := db.NewMockPlans(t)
			mockApplies := db.NewMockApplies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockUsers := db.NewMockUsers(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockEmailClient := email.NewMockClient(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			mockCaller.On("GetSubject").Return("testSubject")

			mockRuns.On("GetRun", mock.Anything, test.run.Metadata.ID).
				Return(test.run, nil)

			mockRuns.On("UpdateRun", mock.Anything, test.run).
				Return(test.run, nil)

			mockUsers.On("GetUserByEmail", mock.Anything, userEmail).
				Return(user, test.injectGetUserError).Maybe()

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspace.Metadata.ID).
				Return(workspace, test.injectGetWorkspaceError).Maybe()

			applyStatus := models.ApplyFinished
			var thisApplyMsg *string
			if test.run.ApplyID == applyBadID {
				applyStatus = models.ApplyErrored
				thisApplyMsg = &applyErrorMessage
			}
			mockApplies.On("GetApply", mock.Anything, test.run.ApplyID).
				Return(&models.Apply{
					Metadata: models.ResourceMetadata{
						ID: test.run.ApplyID,
					},
					Status:       applyStatus,
					ErrorMessage: thisApplyMsg,
				}, nil).Maybe()

			planStatus := models.PlanFinished
			var thisPlanMsg *string
			if test.run.PlanID == planBadID {
				planStatus = models.PlanErrored
				thisPlanMsg = &planErrorMessage
			}
			mockPlans.On("GetPlan", mock.Anything, test.run.PlanID).
				Return(&models.Plan{
					Metadata: models.ResourceMetadata{
						ID: test.run.PlanID,
					},
					Status:       planStatus,
					ErrorMessage: thisPlanMsg,
				}, nil).Maybe()

			mockUsers.On("GetUsers", mock.Anything, mock.Anything).
				Return(&db.UsersResult{
					Users: []models.User{
						*user,
					},
				}, nil).Maybe()

			if test.expectSendMailInput != nil {
				mockEmailClient.On("SendMail", mock.Anything, test.expectSendMailInput).
					Return(nil)
			}

			dbClient := &db.Client{
				Runs:         mockRuns,
				Plans:        mockPlans,
				Applies:      mockApplies,
				Users:        mockUsers,
				Transactions: mockTransactions,
				Workspaces:   mockWorkspaces,
			}

			mockLogger, observedLogs := logger.NewForTest()

			runStateManager := state.NewRunStateManager(dbClient, mockLogger)
			handler := NewErroredRunEmailHandler(mockLogger, dbClient, runStateManager, mockEmailClient)
			require.NotNil(t, handler)

			handler.RegisterHandlers()

			// Update a run to see whether registration actually worked and that the handler was called.
			updatedRun, err := runStateManager.UpdateRun(auth.WithCaller(ctx, mockCaller), test.run)
			assert.Nil(t, err)
			assert.Equal(t, test.run, updatedRun)

			// Check the logs.  Must filter out logs generated by methods outside of the MUT.
			actualLogs := observedLogs.TakeAll()
			filteredLogs := []string{}
			for _, entry := range actualLogs {
				if _, ok := ignoreLogs[entry.Message]; !ok {
					filteredLogs = append(filteredLogs, entry.Message)
				}
			}
			gotAnyLogs := len(filteredLogs) > 0
			assert.Equal(t, test.expectLog, gotAnyLogs)
		})
	}
}
