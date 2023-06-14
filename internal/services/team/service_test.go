package team

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gotest.tools/v3/assert"
)

func TestAddUserToTeam(t *testing.T) {
	teamMemberID := "team-member-id"
	teamID := "team-id"
	userID := "user-id"
	isMaintainer := false

	// Test cases
	tests := []struct {
		authError                error
		expectCreatedTeamMember  *models.TeamMember
		name                     string
		expectErrCode            string
		input                    models.TeamMember
		injectTeamMembersPerTeam int
	}{
		{
			name: "add user to team",
			input: models.TeamMember{
				UserID:       userID,
				TeamID:       teamID,
				IsMaintainer: isMaintainer,
			},
			expectCreatedTeamMember: &models.TeamMember{
				Metadata:     models.ResourceMetadata{ID: teamMemberID},
				UserID:       userID,
				TeamID:       teamID,
				IsMaintainer: isMaintainer,
			},
		},
		{
			name: "subject does not have permission",
			input: models.TeamMember{
				UserID:       userID,
				TeamID:       teamID,
				IsMaintainer: isMaintainer,
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateTeamPermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockTeams := db.NewMockTeams(t)
			mockTeamMembers := db.NewMockTeamMembers(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			if test.expectCreatedTeamMember != nil {
				mockTeamMembers.On("AddUserToTeam", mock.Anything, mock.Anything).
					Return(test.expectCreatedTeamMember, nil)
			}

			mockTeams.On("GetTeamByID", mock.Anything, mock.Anything).Return(&models.Team{
				Metadata: models.ResourceMetadata{ID: teamID}}, nil)

			dbClient := db.Client{
				Transactions: mockTransactions,
				Teams:        mockTeams,
				TeamMembers:  mockTeamMembers,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, mockActivityEvents)

			teamMember, err := service.AddUserToTeam(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedTeamMember, teamMember)
		})
	}
}
