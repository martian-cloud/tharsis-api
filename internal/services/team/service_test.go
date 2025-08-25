package team

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetTeamByID(t *testing.T) {
	teamID := "team-1"

	type testCase struct {
		expectTeam      *models.Team
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return a team by id",
			expectTeam: &models.Team{
				Metadata: models.ResourceMetadata{
					ID: teamID,
				},
			},
		},
		{
			name:            "team does not exist",
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)

			mockTeams.On("GetTeamByID", mock.Anything, teamID).Return(test.expectTeam, nil)

			dbClient := &db.Client{
				Teams: mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			actualTeam, err := service.GetTeamByID(auth.WithCaller(ctx, mockCaller), teamID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectTeam, actualTeam)
		})
	}
}

func TestGetTeamByTRN(t *testing.T) {
	sampleTeam := &models.Team{
		Metadata: models.ResourceMetadata{
			ID:  "team-id-1",
			TRN: types.TeamModelType.BuildTRN("my-team/team-1"),
		},
		Name:        "team-1",
		Description: "Test team",
	}

	type testCase struct {
		caller          auth.Caller
		name            string
		team            *models.Team
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "successfully get team by trn",
			caller: &auth.SystemCaller{},
			team:   sampleTeam,
		},
		{
			name:            "team not found",
			caller:          &auth.SystemCaller{},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockTeams := db.NewMockTeams(t)

			if test.caller != nil {
				ctx = auth.WithCaller(ctx, test.caller)
				mockTeams.On("GetTeamByTRN", mock.Anything, sampleTeam.Metadata.TRN).Return(test.team, nil)
			}

			dbClient := &db.Client{
				Teams: mockTeams,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualTeam, err := service.GetTeamByTRN(ctx, sampleTeam.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.team, actualTeam)
		})
	}
}

func TestGetTeamsByIDs(t *testing.T) {
	teamID := "team-1"

	type testCase struct {
		name       string
		expectTeam models.Team
	}

	testCases := []testCase{
		{
			name: "successfully return a list of teams",
			expectTeam: models.Team{
				Metadata: models.ResourceMetadata{
					ID: teamID,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)

			input := &db.GetTeamsInput{Filter: &db.TeamFilter{TeamIDs: []string{teamID}}}

			mockTeams.On("GetTeams", mock.Anything, input).
				Return(&db.TeamsResult{
					Teams: []models.Team{test.expectTeam},
				}, nil)

			dbClient := &db.Client{
				Teams: mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			result, err := service.GetTeamsByIDs(auth.WithCaller(ctx, mockCaller), []string{teamID})

			if err != nil {
				t.Fatal(err)
			}

			assert.Len(t, result, 1)
			assert.Equal(t, test.expectTeam, result[0])
		})
	}
}

func TestGetTeams(t *testing.T) {
	teamName := "team-1"

	type testCase struct {
		name       string
		expectTeam models.Team
	}

	testCases := []testCase{
		{
			name: "successfully return a list of teams",
			expectTeam: models.Team{
				Metadata: models.ResourceMetadata{
					ID: teamName,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)

			dbInput := &db.GetTeamsInput{Filter: &db.TeamFilter{TeamNamePrefix: &teamName}}

			mockTeams.On("GetTeams", mock.Anything, dbInput).
				Return(&db.TeamsResult{
					Teams: []models.Team{test.expectTeam},
				}, nil)

			dbClient := &db.Client{
				Teams: mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			result, err := service.GetTeams(auth.WithCaller(ctx, mockCaller), &GetTeamsInput{TeamNamePrefix: &teamName})

			if err != nil {
				t.Fatal(err)
			}

			assert.Len(t, result.Teams, 1)
			assert.Equal(t, test.expectTeam, result.Teams[0])
		})
	}
}

func TestCreateTeam(t *testing.T) {
	type testCase struct {
		input           *CreateTeamInput
		expectTeam      *models.Team
		name            string
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully create a team",
			input: &CreateTeamInput{
				Name:        "team-1",
				Description: "team description",
			},
			expectTeam: &models.Team{
				Name:        "team-1",
				Description: "team description",
			},
		},
		{
			name: "team model is not valid",
			input: &CreateTeamInput{
				Name:        "team-1",
				Description: strings.Repeat("long description", 50),
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "subject does not have permission to create team",
			input:           &CreateTeamInput{},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateTeamPermission).Return(test.authError)

			if test.expectTeam != nil {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockTeams.On("CreateTeam", mock.Anything, test.expectTeam).Return(test.expectTeam, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					Action:     models.ActionCreate,
					TargetType: models.TargetTeam,
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				Teams:        mockTeams,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			created, err := service.CreateTeam(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectTeam, created)
		})
	}
}

func TestUpdateTeam(t *testing.T) {
	sampleTeam := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: "team-1",
		},
		Name:        "team",
		Description: "old description",
	}

	type testCase struct {
		input           *UpdateTeamInput
		existingTeam    *models.Team
		expectTeam      *models.Team
		name            string
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully update a team",
			input: &UpdateTeamInput{
				ID:          "team",
				Description: ptr.String("new description"),
			},
			existingTeam: sampleTeam,
			expectTeam: &models.Team{
				Metadata: models.ResourceMetadata{
					ID: sampleTeam.Metadata.ID,
				},
				Name:        "team",
				Description: "new description",
			},
		},
		{
			name: "team does not exist",
			input: &UpdateTeamInput{
				ID: "team",
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have permission to update team",
			input: &UpdateTeamInput{
				ID: "team",
			},
			existingTeam:    sampleTeam,
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "team model is not valid",
			input: &UpdateTeamInput{
				ID:          "team-1",
				Description: ptr.String(strings.Repeat("long description", 50)),
			},
			existingTeam:    sampleTeam,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockTeams.On("GetTeamByID", mock.Anything, test.input.ID).Return(test.existingTeam, nil)

			if test.existingTeam != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateTeamPermission, mock.Anything).Return(test.authError)
			}

			if test.expectTeam != nil {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockTeams.On("UpdateTeam", mock.Anything, test.expectTeam).Return(test.expectTeam, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					Action:     models.ActionUpdate,
					TargetType: models.TargetTeam,
					TargetID:   sampleTeam.Metadata.ID,
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				Teams:        mockTeams,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			created, err := service.UpdateTeam(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectTeam, created)
		})
	}
}

func TestDeleteTeam(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully delete a team",
		},
		{
			name:            "subject does not have permission to delete team",
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)

			mockCaller.On("RequirePermission", mock.Anything, models.DeleteTeamPermission, mock.Anything).Return(test.authError)

			if test.expectErrorCode == "" {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTeams.On("DeleteTeam", mock.Anything, &models.Team{}).Return(nil)
			}

			dbClient := &db.Client{
				Teams: mockTeams,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil)

			err := service.DeleteTeam(auth.WithCaller(ctx, mockCaller), &DeleteTeamInput{Team: &models.Team{}})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetTeamMember(t *testing.T) {
	sampleTeam := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: "team-1",
		},
		Name: "some-team",
	}

	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-1",
		},
		Username: "some-user",
	}

	type testCase struct {
		existingUser    *models.User
		existingTeam    *models.Team
		expectMember    *models.TeamMember
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "successfully return a team member",
			existingUser: sampleUser,
			existingTeam: sampleTeam,
			expectMember: &models.TeamMember{
				UserID:       sampleUser.Metadata.ID,
				TeamID:       sampleTeam.Metadata.ID,
				IsMaintainer: true,
			},
		},
		{
			name:            "user does not exist",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "team does not exist",
			existingUser:    sampleUser,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "team member not found",
			existingUser:    sampleUser,
			existingTeam:    sampleTeam,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)
			mockUsers := db.NewMockUsers(t)
			mockTeamMembers := db.NewMockTeamMembers(t)

			mockUsers.On("GetUserByTRN", mock.Anything, types.UserModelType.BuildTRN(sampleUser.Username)).Return(test.existingUser, nil)

			if test.existingUser != nil {
				mockTeams.On("GetTeamByTRN", mock.Anything, types.TeamModelType.BuildTRN(sampleTeam.Name)).Return(test.existingTeam, nil)
			}

			if test.existingTeam != nil && test.existingUser != nil {
				mockTeamMembers.On("GetTeamMember", mock.Anything, sampleUser.Metadata.ID, sampleTeam.Metadata.ID).Return(test.expectMember, nil)
			}

			dbClient := &db.Client{
				Users:       mockUsers,
				Teams:       mockTeams,
				TeamMembers: mockTeamMembers,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil)

			actualMember, err := service.GetTeamMember(auth.WithCaller(ctx, mockCaller), sampleUser.Username, sampleTeam.Name)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectMember, actualMember)
		})
	}
}

func TestGetTeamMembers(t *testing.T) {
	teamID := "team-1"

	type testCase struct {
		name         string
		expectMember models.TeamMember
	}

	testCases := []testCase{
		{
			name: "successfully return a list of teams",
			expectMember: models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID: "team",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeamMembers := db.NewMockTeamMembers(t)

			dbInput := &db.GetTeamMembersInput{
				Filter: &db.TeamMemberFilter{
					TeamIDs: []string{teamID},
				},
			}

			mockTeamMembers.On("GetTeamMembers", mock.Anything, dbInput).
				Return(&db.TeamMembersResult{
					TeamMembers: []models.TeamMember{test.expectMember},
				}, nil)

			dbClient := &db.Client{
				TeamMembers: mockTeamMembers,
			}

			service := NewService(nil, dbClient, nil)

			result, err := service.GetTeamMembers(auth.WithCaller(ctx, mockCaller), dbInput)

			if err != nil {
				t.Fatal(err)
			}

			assert.Len(t, result.TeamMembers, 1)
			assert.Equal(t, test.expectMember, result.TeamMembers[0])
		})
	}
}

func TestAddUserToTeam(t *testing.T) {
	sampleTeam := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: "team-1",
		},
		Name: "some-team",
	}

	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-1",
		},
		Username: "some-user",
	}

	type testCase struct {
		existingUser    *models.User
		existingTeam    *models.Team
		expectAdded     *models.TeamMember
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "successfully add a user to a team",
			existingUser: sampleUser,
			existingTeam: sampleTeam,
			expectAdded: &models.TeamMember{
				UserID:       sampleUser.Metadata.ID,
				TeamID:       sampleTeam.Metadata.ID,
				IsMaintainer: true,
			},
		},
		{
			name:            "team does not exist",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "user does not exist",
			existingTeam:    sampleTeam,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject does not have permission to add team members",
			existingUser:    sampleUser,
			existingTeam:    sampleTeam,
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)
			mockUsers := db.NewMockUsers(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockTeams.On("GetTeamByTRN", mock.Anything, types.TeamModelType.BuildTRN(sampleTeam.Name)).Return(test.existingTeam, nil)

			if test.existingTeam != nil {
				mockUsers.On("GetUserByTRN", mock.Anything, types.UserModelType.BuildTRN(sampleUser.Username)).Return(test.existingUser, nil)
			}

			if test.existingTeam != nil && test.existingUser != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateTeamPermission, mock.Anything).Return(test.authError)
			}

			if test.expectAdded != nil {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockTeamMembers.On("AddUserToTeam", mock.Anything, test.expectAdded).Return(test.expectAdded, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					Action:     models.ActionAddMember,
					TargetType: models.TargetTeam,
					TargetID:   sampleTeam.Metadata.ID,
					Payload: &models.ActivityEventAddTeamMemberPayload{
						UserID:     &sampleUser.Metadata.ID,
						Maintainer: true,
					},
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				Users:        mockUsers,
				Teams:        mockTeams,
				TeamMembers:  mockTeamMembers,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			input := &AddUserToTeamInput{
				TeamName:     sampleTeam.Name,
				Username:     sampleUser.Username,
				IsMaintainer: true,
			}

			added, err := service.AddUserToTeam(auth.WithCaller(ctx, mockCaller), input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAdded, added)
		})
	}
}

func TestUpdateTeamMember(t *testing.T) {
	sampleTeam := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: "team-1",
		},
		Name: "some-team",
	}

	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-1",
		},
		Username: "some-user",
	}

	sampleMember := &models.TeamMember{
		UserID: sampleUser.Metadata.ID,
		TeamID: sampleTeam.Metadata.ID,
	}

	type testCase struct {
		existingUser    *models.User
		existingTeam    *models.Team
		existingMember  *models.TeamMember
		expectUpdated   *models.TeamMember
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:           "successfully update a team member",
			existingUser:   sampleUser,
			existingTeam:   sampleTeam,
			existingMember: sampleMember,
			expectUpdated: &models.TeamMember{
				UserID:       sampleUser.Metadata.ID,
				TeamID:       sampleTeam.Metadata.ID,
				IsMaintainer: true,
			},
		},
		{
			name:            "team does not exist",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "user does not exist",
			existingTeam:    sampleTeam,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject does not have permission to update team members",
			existingUser:    sampleUser,
			existingTeam:    sampleTeam,
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "team member does not exist",
			existingUser:    sampleUser,
			existingTeam:    sampleTeam,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeams := db.NewMockTeams(t)
			mockUsers := db.NewMockUsers(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockTeams.On("GetTeamByTRN", mock.Anything, types.TeamModelType.BuildTRN(sampleTeam.Name)).Return(test.existingTeam, nil)

			if test.existingTeam != nil {
				mockUsers.On("GetUserByTRN", mock.Anything, types.UserModelType.BuildTRN(sampleUser.Username)).Return(test.existingUser, nil)
			}

			if test.existingTeam != nil && test.existingUser != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateTeamPermission, mock.Anything).Return(test.authError)

				if test.authError == nil {
					mockTeamMembers.On("GetTeamMember", mock.Anything, sampleUser.Metadata.ID, sampleTeam.Metadata.ID).Return(test.existingMember, nil)
				}
			}

			if test.expectUpdated != nil {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockTeamMembers.On("UpdateTeamMember", mock.Anything, test.expectUpdated).Return(test.expectUpdated, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					Action:     models.ActionUpdateMember,
					TargetType: models.TargetTeam,
					TargetID:   sampleTeam.Metadata.ID,
					Payload: &models.ActivityEventUpdateTeamMemberPayload{
						UserID:     &sampleUser.Metadata.ID,
						Maintainer: true,
					},
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				Users:        mockUsers,
				Teams:        mockTeams,
				TeamMembers:  mockTeamMembers,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			input := &UpdateTeamMemberInput{
				TeamName:     sampleTeam.Name,
				Username:     sampleUser.Username,
				IsMaintainer: true,
			}

			added, err := service.UpdateTeamMember(auth.WithCaller(ctx, mockCaller), input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectUpdated, added)
		})
	}
}

func TestDeleteTeamMember(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully delete a team member",
		},
		{
			name:            "subject does not have permission to delete team member",
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateTeamPermission, mock.Anything).Return(test.authError)

			if test.expectErrorCode == "" {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockTeamMembers.On("RemoveUserFromTeam", mock.Anything, &models.TeamMember{
					UserID: "user-1",
					TeamID: "team-1",
				}).Return(nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					Action:     models.ActionRemoveMember,
					TargetType: models.TargetTeam,
					TargetID:   "team-1",
					Payload: &models.ActivityEventRemoveTeamMemberPayload{
						UserID: ptr.String("user-1"),
					},
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				TeamMembers:  mockTeamMembers,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			input := &RemoveUserFromTeamInput{TeamMember: &models.TeamMember{
				UserID: "user-1",
				TeamID: "team-1",
			}}

			err := service.RemoveUserFromTeam(auth.WithCaller(ctx, mockCaller), input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
