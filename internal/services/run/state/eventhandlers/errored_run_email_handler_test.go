package eventhandlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

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
	globalRunID := gid.NewGlobalID(types.RunModelType, internalRunID).String() // "Ul9ydW4taWQ"

	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
	}

	user := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-1",
		},
		Email: userEmail,
	}

	type testCase struct {
		name                string
		run                 *models.Run
		usersToNotify       []string
		expectSendMailInput *email.SendMailInput
	}

	testCases := []testCase{
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
			usersToNotify: []string{},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
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
			usersToNotify: []string{user.Metadata.ID},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			//mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockPlans := db.NewMockPlans(t)
			mockApplies := db.NewMockApplies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockUsers := db.NewMockUsers(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockEmailClient := email.NewMockClient(t)
			mockNotificationManager := namespace.NewMockNotificationManager(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			mockNotificationManager.On("GetUsersToNotify", mock.Anything, mock.Anything).Return(test.usersToNotify, nil).Maybe()

			mockUsers.On("GetUserByEmail", mock.Anything, userEmail).
				Return(user, nil).Maybe()

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspace.Metadata.ID).
				Return(workspace, nil).Maybe()

			applyStatus := models.ApplyFinished
			var thisApplyMsg *string
			if test.run.ApplyID == applyBadID {
				applyStatus = models.ApplyErrored
				thisApplyMsg = &applyErrorMessage
			}
			mockApplies.On("GetApplyByID", mock.Anything, test.run.ApplyID).
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
			mockPlans.On("GetPlanByID", mock.Anything, test.run.PlanID).
				Return(&models.Plan{
					Metadata: models.ResourceMetadata{
						ID: test.run.PlanID,
					},
					Status:       planStatus,
					ErrorMessage: thisPlanMsg,
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

			mockLogger, _ := logger.NewForTest()

			handler := &ErroredRunEmailHandler{
				logger:              mockLogger,
				dbClient:            dbClient,
				emailClient:         mockEmailClient,
				notificationManager: mockNotificationManager,
			}

			// Update a run to see whether registration actually worked and that the handler was called.
			err := handler.sendFailedRunEmail(ctx, test.run)
			assert.Nil(t, err)
		})
	}
}
