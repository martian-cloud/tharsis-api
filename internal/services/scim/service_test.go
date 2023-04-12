package scim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

const (
	// Used for ResourceMetadata.ID by several test resources.
	resourceUUID = "0e8408de-a2ff-4194-8481-d0bbb0874037"

	// Used as SCIMExternalID.
	externalID = "a1ef8922-aa06-4445-8d1e-9957e6c90ace"
)

func TestCreateSCIMToken(t *testing.T) {
	existingToken := models.SCIMToken{
		Nonce: "dc05adc5-4535-4251-9cc3-c01d77cbc9e9",
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}

	testCases := []struct {
		caller          *auth.UserCaller
		inputToken      *models.SCIMToken
		name            string
		expectErrorCode string
		existingTokens  []models.SCIMToken
	}{
		{
			name: "positive: caller is admin; expect signed token.",
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{
						ID: "user-1",
					},
					Email: "user-1@example.com",
					Admin: true,
				},
			},
			existingTokens: []models.SCIMToken{existingToken},
		},
		{
			name: "negative: caller is not admin; expect error EForbidden.",
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{
						ID: "user-2",
					},
					Email: "user-2@example.com",
					Admin: false,
				},
			},
			existingTokens:  []models.SCIMToken{existingToken},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockScimTokens := db.MockSCIMTokens{}
			mockScimTokens.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockScimTokens.On("GetTokens", ctx).Return(test.existingTokens, nil)
			mockScimTokens.On("DeleteToken", ctx, &test.existingTokens[0]).Return(nil)
			mockScimTokens.On("CreateToken", ctx, mock.Anything).Return(nil, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient := &db.Client{
				SCIMTokens:   &mockScimTokens,
				Transactions: &mockTransactions,
			}

			mockJWSProvider := jws.MockProvider{}
			mockJWSProvider.Test(t)

			mockJWSProvider.On("Sign", mock.Anything, mock.Anything).Return([]byte("signed-token"), nil)

			logger, _ := logger.NewForTest()
			identityProvider := auth.NewIdentityProvider(&mockJWSProvider, "https://tharsis.domain")
			service := NewService(logger, dbClient, identityProvider)

			token, err := service.CreateSCIMToken(auth.WithCaller(ctx, test.caller))
			if test.expectErrorCode != "" {
				// Negative case.
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Equal(t, []byte("signed-token"), token)
			}
		})
	}
}

func TestGetSCIMUsers(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	expectedUser := models.User{
		Username:       "expected-user-name",
		Email:          "expected-user-email",
		SCIMExternalID: externalID,
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		Admin:  false,
		Active: false,
	}
	expectedUsersList := []models.User{expectedUser}

	testCases := []struct {
		name          string
		input         *GetSCIMResourceInput
		expectedError string
		expectedUsers []models.User
	}{
		{
			name: "positive: SCIMExternalID is valid; expect a slice of users.",
			input: &GetSCIMResourceInput{
				SCIMExternalID: externalID,
			},
			expectedUsers: expectedUsersList,
			// expect nil errors
		},
		{
			name:          "positive: input is empty; expect a slice of all users.",
			input:         &GetSCIMResourceInput{},
			expectedUsers: expectedUsersList,
		},
		{
			name: "negative: input has an invalid scimExternalID; expect a nil slice of users.",
			input: &GetSCIMResourceInput{
				SCIMExternalID: "invalid-id",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			getUsersInput := &db.GetUsersInput{
				Filter: &db.UserFilter{
					SCIMExternalID: true,
				}}

			externalIDUser := &expectedUser
			if len(test.expectedUsers) == 0 {
				externalIDUser = nil
			}

			mockUsers.On("GetUserBySCIMExternalID", mock.Anything, test.input.SCIMExternalID).Return(externalIDUser, nil)
			mockUsers.On("GetUsers", mock.Anything, getUsersInput).Return(&db.UsersResult{Users: expectedUsersList}, nil)

			dbClient := &db.Client{
				Users: &mockUsers,
			}

			service := NewService(nil, dbClient, nil)

			users, err := service.GetSCIMUsers(ctx, test.input)
			if test.expectedError != "" {
				// Negative invalid filter.
				assert.Equal(t, test.expectedError, errors.ErrorMessage(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedUsers, users)
			}
		})
	}
}

func TestCreateSCIMUser(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	sampleUser := &models.User{
		Username:       "input-user",
		Email:          "input-user@example.com",
		SCIMExternalID: externalID,
		Admin:          false,
		Active:         true,
	}

	// testCases should never return an error since CreateSCIMUser is idempotent.
	testCases := []struct {
		input            *CreateSCIMUserInput
		existingSCIMUser *models.User
		returnedSCIMUser *models.User
		name             string
	}{
		{
			name: "positive: user already exists in the system without a scimExternalID; expect updated user.",
			input: &CreateSCIMUserInput{
				Email:          "input-user@example.com",
				SCIMExternalID: externalID,
				Active:         true,
			},
			existingSCIMUser: &models.User{
				Username: "input-user",
				Email:    "input-user@example.com",
				Admin:    false,
				Active:   true,
			},
			returnedSCIMUser: sampleUser,
		},
		{
			name: "positive: user does not already exist in the system; expect user to be created.",
			input: &CreateSCIMUserInput{
				Email:          "input-user@example.com",
				SCIMExternalID: externalID,
				Active:         true,
			},
			returnedSCIMUser: sampleUser,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockUsers.On("GetUserByEmail", mock.Anything, test.input.Email).Return(test.existingSCIMUser, nil)
			mockUsers.On("UpdateUser", mock.Anything, test.existingSCIMUser).Return(test.returnedSCIMUser, nil)
			mockUsers.On("CreateUser", mock.Anything, sampleUser).Return(test.returnedSCIMUser, nil)

			dbClient := &db.Client{
				Users: &mockUsers,
			}

			service := NewService(nil, dbClient, nil)

			user, err := service.CreateSCIMUser(ctx, test.input)
			if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.returnedSCIMUser, user)
			}
		})
	}
}

func TestUpdateSCIMUser(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	// sampleUser is the model for several user inputs.
	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		Username:       "input-user",
		SCIMExternalID: externalID,
	}

	testCases := []struct {
		input             *UpdateResourceInput
		existingUser      *models.User // User already in the system.
		expectedSCIMUser  *models.User // Updates user.
		name              string
		expectedErrorCode string
	}{
		{
			name: "positive: valid 'replace' operation; expect SCIMExternalID to be updated.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP:    replaceOPType, // This must be lowercase for tests (handled in controller, otherwise).
						Path:  "externalId",  // This, however, is case-sensitive.
						Value: "new-external-id",
					},
				},
			},
			existingUser: sampleUser,
			expectedSCIMUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				Username:       "input-user",
				SCIMExternalID: "new-external-id",
			},
			// We expect no error from this.
		},
		{
			name: "negative: invalid Metadata ID; expect error EInternal",
			input: &UpdateResourceInput{
				ID:         "bogus-id",
				Operations: []Operation{}, // Operation won't matter for this.
			},
			expectedErrorCode: errors.EInternal,
		},
		{
			name: "negative: invalid operation OP; expect error EInvalid.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP: "invalid-op",
						// Rest of the fields should not matter.
					},
				},
			},
			existingUser:      sampleUser,
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: invalid operation path; expect error EInvalid.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP:   replaceOPType,
						Path: "invalid-path",
						// Value does not matter here.
					},
				},
			},
			existingUser:      sampleUser,
			expectedErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			// This function just updates the user in the db layer.
			// Actual fields are updated via a private func.
			mockUsers.On("UpdateUser", mock.Anything, test.expectedSCIMUser).Return(test.expectedSCIMUser, nil)
			mockUsers.On("GetUserByID", mock.Anything, test.input.ID).Return(test.existingUser, nil)

			dbClient := &db.Client{
				Users:        &mockUsers,
				Transactions: &mockTransactions,
			}

			service := NewService(nil, dbClient, nil)

			user, err := service.UpdateSCIMUser(ctx, test.input)
			if test.expectedErrorCode != "" {
				// Negative cases.
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Equal(t, test.expectedSCIMUser, user)
			}
		})
	}
}

func TestDeleteSCIMUser(t *testing.T) {
	testCases := []struct {
		name              string
		input             *DeleteSCIMResourceInput
		existingUser      *models.User
		authError         error
		expectedErrorCode string
	}{
		{
			name: "positive: user was created via SCIM; expect no error.",
			input: &DeleteSCIMResourceInput{
				ID: resourceUUID,
			},
			existingUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				SCIMExternalID: externalID,
				// None of the other fields matter here.
			},
		},
		{
			name: "negative: user being deleted was not created via SCIM; expect error ENotFound.",
			input: &DeleteSCIMResourceInput{
				ID: resourceUUID,
			},
			existingUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				// None of the other fields matter here.
			},
			authError:         errors.New(errors.ENotFound, "Resource not found"),
			expectedErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			// Caller function mocks.
			mockCaller.On("RequirePermission", mock.Anything, permissions.DeleteUserPermission, mock.Anything).Return(test.authError)

			ctx := auth.WithCaller(context.Background(), &mockCaller)

			mockUsers.On("GetUserByID", mock.Anything, test.input.ID).Return(test.existingUser, nil)
			mockUsers.On("DeleteUser", mock.Anything, test.existingUser).Return(nil)

			dbClient := &db.Client{
				Users: &mockUsers,
			}

			service := NewService(nil, dbClient, nil)

			err := service.DeleteSCIMUser(ctx, test.input)
			if test.expectedErrorCode != "" {
				// Negative case.
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetSCIMGroups(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	expectedSCIMGroup := models.Team{
		Name:           "expected-scim-group",
		SCIMExternalID: externalID,
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}
	expectedSCIMGroupsList := []models.Team{expectedSCIMGroup}

	testCases := []struct {
		name               string
		input              *GetSCIMResourceInput
		expectedError      string
		expectedSCIMGroups []models.Team
	}{
		{
			name: "positive: SCIMExternalID is valid; expect a slice of SCIM groups (teams).",
			input: &GetSCIMResourceInput{
				SCIMExternalID: getFilter("externalId", expectedSCIMGroup.SCIMExternalID),
			},
			expectedSCIMGroups: expectedSCIMGroupsList,
			// expect nil errors
		},
		{
			name:               "positive: SCIMExternalID is empty; expect a slice of all SCIM groups (teams).",
			input:              &GetSCIMResourceInput{},
			expectedSCIMGroups: expectedSCIMGroupsList,
		},
		{
			name: "negative: input has an invalid scimExternalID; expect a nil slice of users.",
			input: &GetSCIMResourceInput{
				SCIMExternalID: "invalid-id",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			getTeamsInput := &db.GetTeamsInput{
				Filter: &db.TeamFilter{
					SCIMExternalID: true,
				},
			}

			returnedSCIMGroup := &expectedSCIMGroup
			if len(test.expectedSCIMGroups) == 0 {
				returnedSCIMGroup = nil
			}

			mockTeams.On("GetTeamBySCIMExternalID", mock.Anything, test.input.SCIMExternalID).Return(returnedSCIMGroup, nil)
			mockTeams.On("GetTeams", mock.Anything, getTeamsInput).Return(&db.TeamsResult{Teams: expectedSCIMGroupsList}, nil)

			dbClient := &db.Client{
				Teams: &mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			groups, err := service.GetSCIMGroups(ctx, test.input)
			if test.expectedError != "" {
				// Negative invalid filter.
				assert.Equal(t, test.expectedError, errors.ErrorMessage(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedSCIMGroups, groups)
			}
		})
	}
}

func TestCreateSCIMGroup(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	sampleSCIMGroup := &models.Team{
		Name:           "input-team",
		SCIMExternalID: externalID,
	}

	// testCases should never return an error since CreateSCIMGroup is idempotent.
	testCases := []struct {
		input             *CreateSCIMGroupInput
		existingSCIMGroup *models.Team
		returnedSCIMGroup *models.Team
		name              string
	}{
		{
			name: "positive: SCIM group already exists in the system without a scimExternalID; expect updated SCIM group.",
			input: &CreateSCIMGroupInput{
				Name: sampleSCIMGroup.Name,
			},
			existingSCIMGroup: &models.Team{
				Name: sampleSCIMGroup.Name,
			},
			returnedSCIMGroup: sampleSCIMGroup,
		},
		{
			name: "positive: SCIM group does not already exist in the system; expect SCIM group to be created.",
			input: &CreateSCIMGroupInput{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: sampleSCIMGroup.SCIMExternalID,
			},
			returnedSCIMGroup: sampleSCIMGroup,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			mockTeams.On("GetTeamByName", mock.Anything, test.input.Name).Return(test.existingSCIMGroup, nil)
			mockTeams.On("UpdateTeam", mock.Anything, test.existingSCIMGroup).Return(test.returnedSCIMGroup, nil)
			mockTeams.On("CreateTeam", mock.Anything, sampleSCIMGroup).Return(test.returnedSCIMGroup, nil)

			dbClient := &db.Client{
				Teams: &mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			group, err := service.CreateSCIMGroup(ctx, test.input)
			if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.returnedSCIMGroup, group)
			}
		})
	}
}

func TestUpdateSCIMGroup(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), &auth.SCIMCaller{})

	scimGroupID := "663a2ffa-5e95-41fa-9119-c7bcad3d775e"

	// sampleSCIMGroup is the model for several user inputs.
	sampleSCIMGroup := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: scimGroupID,
		},
		Name:           "input-team",
		SCIMExternalID: externalID,
	}

	// sampleUser is used when adding or removing SCIM group (team) member.
	sampleUser := &models.User{
		Username: "sample-users-name",
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}

	// sampleSCIMGroupMember is used for adding, removing team members.
	sampleSCIMGroupMember := &models.TeamMember{
		UserID: resourceUUID,
		TeamID: scimGroupID,
	}

	testCases := []struct {
		inputID                 string
		existingSCIMGroup       *models.Team       // SCIM Group (team) already in system.
		existingSCIMGroupMember *models.TeamMember // SCIM Group (team) member already in system.
		expectedSCIMGroup       *models.Team       // Updated SCIM Group (team).
		name                    string
		expectedErrorCode       string
		operations              []Operation
	}{
		{
			name:              "positive: valid 'replace' operation; expect SCIM group name to be updated.",
			inputID:           scimGroupID,
			existingSCIMGroup: sampleSCIMGroup,
			operations: []Operation{
				{
					OP:    replaceOPType,
					Path:  "displayName", // This, however, is case-sensitive.
					Value: "new-display-name",
				},
			},
			expectedSCIMGroup: &models.Team{
				Name: "new-display-name",
				Metadata: models.ResourceMetadata{
					ID: scimGroupID,
				},
				SCIMExternalID: externalID,
			},
			// We expect no error from this.
		},
		{
			name:                    "positive: valid 'add' operation; expect SCIM group member to be added to SCIM group.",
			inputID:                 scimGroupID,
			existingSCIMGroup:       sampleSCIMGroup,
			existingSCIMGroupMember: sampleSCIMGroupMember,
			operations: []Operation{
				{
					OP:   addOPType,
					Path: "members",
					Value: []interface{}{
						map[string]interface{}{
							"value": gid.ToGlobalID(gid.UserType, resourceUUID),
						},
					},
				},
			},
			expectedSCIMGroup: sampleSCIMGroup, // Team model does not update for modifying team members.
		},
		{
			name:                    "positive: SCIM group member is already added to the SCIM group; expect no updates.",
			inputID:                 scimGroupID,
			existingSCIMGroup:       sampleSCIMGroup,
			existingSCIMGroupMember: sampleSCIMGroupMember,
			operations: []Operation{
				{
					OP:   addOPType,
					Path: "members",
					Value: []interface{}{
						map[string]interface{}{
							"value": gid.ToGlobalID(gid.UserType, resourceUUID),
						},
					},
				},
			},
			expectedSCIMGroup: sampleSCIMGroup,
			// Normally, this would produce an EConflict since the SCIM group member
			// is already part of the SCIM group, but with the new update logic,
			// nothing should happen and the team will get returned back.
		},
		{
			name:                    "positive: valid 'remove' operation; expect SCIM group member to be removed from SCIM group.",
			inputID:                 scimGroupID,
			existingSCIMGroup:       sampleSCIMGroup,
			existingSCIMGroupMember: sampleSCIMGroupMember,
			operations: []Operation{
				{
					OP:   removeOPType,
					Path: "members",
					Value: []interface{}{
						map[string]interface{}{
							"value": gid.ToGlobalID(gid.UserType, resourceUUID),
						},
					},
				},
			},
			expectedSCIMGroup: sampleSCIMGroup, // Team model does not update for modifying team members.
		},
		{
			name:                    "positive: valid 'remove' operation with no value; expect ALL SCIM group member to be removed from SCIM group.",
			inputID:                 scimGroupID,
			existingSCIMGroup:       sampleSCIMGroup,
			existingSCIMGroupMember: sampleSCIMGroupMember,
			operations: []Operation{
				{
					OP:   removeOPType,
					Path: "members",
					// No value means remove ALL members.
				},
			},
			expectedSCIMGroup: sampleSCIMGroup, // Team model does not update for modifying team members.
		},
		{
			name:              "negative: invalid Metadata ID; expect error EInternal",
			inputID:           "bogus-id",
			operations:        []Operation{}, // Operation won't matter for this.
			expectedErrorCode: errors.EInternal,
		},
		{
			name:              "negative: invalid operation OP; expect error EInvalid.",
			inputID:           scimGroupID,
			existingSCIMGroup: sampleSCIMGroup,
			operations: []Operation{
				{
					OP: "invalid-op",
					// Rest of the fields should not matter.
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name:              "negative: invalid operation path; expect error EInvalid.",
			inputID:           scimGroupID,
			existingSCIMGroup: sampleSCIMGroup,
			operations: []Operation{
				{
					OP:   replaceOPType,
					Path: "invalid-path",
					// Value does not matter here.
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			mockTeamMembers := db.MockTeamMembers{}
			mockTeamMembers.Test(t)

			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			// Team function mocks.
			mockTeams.On("UpdateTeam", mock.Anything, test.expectedSCIMGroup).Return(test.expectedSCIMGroup, nil)
			mockTeams.On("GetTeamByID", mock.Anything, test.inputID).Return(test.existingSCIMGroup, nil)

			getTeamMembersInput := &db.GetTeamMembersInput{
				Filter: &db.TeamMemberFilter{
					TeamIDs: []string{test.inputID},
				},
			}

			addTeamMemberInput := &models.TeamMember{
				UserID: resourceUUID,
				TeamID: test.inputID,
			}

			sampleSCIMGroupMembers := &db.TeamMembersResult{}
			if test.existingSCIMGroupMember != nil {
				sampleSCIMGroupMembers = &db.TeamMembersResult{
					TeamMembers: []models.TeamMember{
						*test.existingSCIMGroupMember,
					},
				}
			}

			// TeamMember function mocks.
			mockTeamMembers.On("GetTeamMember", mock.Anything, sampleUser.Metadata.ID, test.inputID).Return(sampleSCIMGroupMember, nil)
			mockTeamMembers.On("GetTeamMembers", mock.Anything, getTeamMembersInput).Return(sampleSCIMGroupMembers, nil)
			mockTeamMembers.On("AddUserToTeam", mock.Anything, addTeamMemberInput).Return(nil, nil)
			mockTeamMembers.On("RemoveUserFromTeam", mock.Anything, test.existingSCIMGroupMember).Return(nil)

			// User function mocks.
			mockUsers.On("GetUserByID", mock.Anything, sampleUser.Metadata.ID).Return(sampleUser, nil)

			dbClient := &db.Client{
				Teams:        &mockTeams,
				TeamMembers:  &mockTeamMembers,
				Users:        &mockUsers,
				Transactions: &mockTransactions,
			}

			service := NewService(nil, dbClient, nil)

			input := &UpdateResourceInput{
				ID:         test.inputID,
				Operations: test.operations,
			}

			group, err := service.UpdateSCIMGroup(ctx, input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedSCIMGroup, group)
			}
		})
	}
}

func TestDeleteSCIMGroup(t *testing.T) {
	testCases := []struct {
		name              string
		input             *DeleteSCIMResourceInput
		existingSCIMGroup *models.Team
		authError         error
		expectedErrorCode string
	}{
		{
			name: "positive: team was created via SCIM; expect no error.",
			input: &DeleteSCIMResourceInput{
				ID: resourceUUID,
			},
			existingSCIMGroup: &models.Team{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				SCIMExternalID: externalID,
				// Rest of the fields won't matter here.
			},
		},
		{
			name: "negative: team being deleted was not created via SCIM; expect error ENotFound.",
			input: &DeleteSCIMResourceInput{
				ID: resourceUUID,
			},
			existingSCIMGroup: &models.Team{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
			},
			authError:         errors.New(errors.ENotFound, "Resource not found"),
			expectedErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			// Caller function mocks.
			mockCaller.On("RequirePermission", mock.Anything, permissions.DeleteTeamPermission, mock.Anything).Return(test.authError)

			ctx := auth.WithCaller(context.Background(), &mockCaller)

			// Team mocks.
			mockTeams.On("GetTeamByID", mock.Anything, test.input.ID).Return(test.existingSCIMGroup, nil)
			mockTeams.On("DeleteTeam", mock.Anything, test.existingSCIMGroup).Return(nil)

			dbClient := &db.Client{
				Teams: &mockTeams,
			}

			service := NewService(nil, dbClient, nil)

			err := service.DeleteSCIMGroup(ctx, test.input)
			if test.expectedErrorCode != "" {
				// Negative case.
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Nil(t, err)
			}
		})
	}
}

// getFilter is a helper function to prepare a SCIM resource filter.
func getFilter(attribute, value string) string {
	return attribute + " Eq \"" + value + "\""
}
