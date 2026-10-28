package scim

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

const (
	// Used for ResourceMetadata.ID by several test resources.
	resourceUUID = "0e8408de-a2ff-4194-8481-d0bbb0874037"

	// Used as SCIMExternalID.
	externalID = "a1ef8922-aa06-4445-8d1e-9957e6c90ace"
)

func TestGetSCIMToken(t *testing.T) {
	existingToken := models.SCIMToken{CreatedBy: models.Actor{Subject: "test-user"}, Metadata: models.ResourceMetadata{ID: resourceUUID}}
	testCases := []struct {
		name            string
		admin           bool
		existingTokens  []models.SCIMToken
		expected        *models.SCIMToken
		expectErrorCode errors.CodeType
	}{
		{
			name:           "token record exists",
			admin:          true,
			existingTokens: []models.SCIMToken{existingToken},
			expected:       &existingToken,
		},
		{
			name:            "token record does not exist",
			admin:           true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "non-admin cannot view token details",
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSCIMTokens := db.NewMockSCIMTokens(t)
			if test.admin {
				mockSCIMTokens.On("GetTokens", mock.Anything).Return(test.existingTokens, nil)
			}
			user := &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}, Admin: test.admin}
			if test.admin {
				expiration := time.Now().Add(time.Hour)
				user.AdminModeExpiration = &expiration
			}
			mockUsers := db.NewMockUsers(t)
			mockUsers.On("GetUserByID", mock.Anything, user.Metadata.ID).Return(user, nil).Maybe()
			dbClient := &db.Client{SCIMTokens: mockSCIMTokens, Users: mockUsers}
			caller := auth.NewUserCaller(user, nil, dbClient, nil, nil)
			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, auth.NewMockSigningKeyManager(t), nil)

			actual, err := service.GetSCIMToken(auth.WithCaller(context.Background(), caller))
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, actual)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

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
		expectErrorCode errors.CodeType
		existingTokens  []models.SCIMToken
		idpIssuerURL    string
	}{
		{
			name: "positive: caller is admin; expect signed token.",
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{
						ID: "user-1",
					},
					Email:               "user-1@example.com",
					Admin:               true,
					AdminModeExpiration: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
				},
			},
			existingTokens: []models.SCIMToken{existingToken},
			idpIssuerURL:   "https://example.com/scim",
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
			idpIssuerURL:    "https://example.com/scim",
		},
		{
			name: "negative: invalid IDP issuer URL",
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					Email:               "admin@example.com",
					Admin:               true,
					AdminModeExpiration: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
				},
			},
			existingTokens:  []models.SCIMToken{},
			expectErrorCode: errors.EInvalid,
			idpIssuerURL:    "https://invalid-idp.com",
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
			if len(test.existingTokens) > 0 {
				mockScimTokens.On("DeleteToken", ctx, &test.existingTokens[0]).Return(nil)
			}
			// Echo the input so the assertion below checks the creator the service recorded.
			mockScimTokens.On("CreateToken", ctx, mock.Anything).Return(func(_ context.Context, token *models.SCIMToken) *models.SCIMToken { return token }, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockUsers := db.NewMockUsers(t)
			mockUsers.On("GetUserByID", mock.Anything, test.caller.User.Metadata.ID).Return(test.caller.User, nil).Maybe()

			dbClient := &db.Client{
				SCIMTokens:   &mockScimTokens,
				Transactions: &mockTransactions,
				Users:        mockUsers,
			}

			caller := auth.NewUserCaller(test.caller.User, nil, dbClient, nil, nil)

			mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			mockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("signed-token"), nil).Maybe()

			logger, _ := logger.NewForTest()

			cfg := &config.Config{OauthProviders: []config.IdpConfig{{IssuerURL: "https://example.com/scim"}}}
			service := NewService(logger, dbClient, mockSigningKeyManager, cfg.OauthProviders)

			output, err := service.CreateSCIMToken(auth.WithCaller(ctx, caller), test.idpIssuerURL)
			if test.expectErrorCode != "" {
				// Negative case.
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Equal(t, []byte("signed-token"), output.PlaintextToken)
				assert.Equal(t, models.NewUserActor(test.caller.User.Metadata.ID).WithSubject(test.caller.User.Email), output.SCIMToken.CreatedBy)
			}
		})
	}
}

func TestGetSCIMUsers(t *testing.T) {
	issuerURL := "https://test-idp.example.com"
	scimCaller := auth.NewSCIMCaller(nil, issuerURL)
	ctx := auth.WithCaller(context.Background(), scimCaller)

	expectedUser := models.User{
		Username:         "expected-user-name",
		Email:            "expected-user-email",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuerURL,
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		Admin:  false,
		Active: false,
	}

	expectedUsersList := []models.User{expectedUser}

	// The same user, but managed by another IDP's SCIM client.
	otherIDPUser := expectedUser
	otherIDPUser.SCIMIDPIssuerURL = "https://other-idp.example.com"

	userWithoutSCIMID := models.User{
		Username:       "user-no-scim",
		Email:          "user-no-scim@example.com",
		SCIMExternalID: "",
		Metadata: models.ResourceMetadata{
			ID: "no-scim-uuid",
		},
		Admin:  false,
		Active: true,
	}

	testCases := []struct {
		name          string
		input         *GetSCIMUsersInput
		expectedError string
		expectedUsers []models.User
		mockUser      *models.User
	}{
		{
			name: "positive: SCIMExternalID is valid; expect a slice of users.",
			input: &GetSCIMUsersInput{
				SCIMExternalID: ptr.String(externalID),
			},
			expectedUsers: expectedUsersList,
			// expect nil errors
		},
		{
			// The external ID only exists for another IDP; the lookup is scoped to the caller's IDP.
			name: "negative: SCIMExternalID of a user managed by another IDP; expect empty slice",
			input: &GetSCIMUsersInput{
				SCIMExternalID: ptr.String(externalID),
			},
			expectedUsers: []models.User{}, // not nil
		},
		{
			name:          "positive: input is empty; expect a slice of all users.",
			input:         &GetSCIMUsersInput{},
			expectedUsers: expectedUsersList,
		},
		{
			name: "negative: input has an invalid scimExternalID; expect a empty slice of users.",
			input: &GetSCIMUsersInput{
				SCIMExternalID: ptr.String("invalid-id"),
			},
			expectedUsers: []models.User{},
		},
		{
			name: "positive: username with matching issuer URL",
			input: &GetSCIMUsersInput{
				Username: ptr.String("expected-user-name"),
			},
			mockUser:      &expectedUser,
			expectedUsers: []models.User{expectedUser},
		},
		{
			name: "negative: username not found; expect empty slice",
			input: &GetSCIMUsersInput{
				Username: ptr.String("invalid-username"),
			},
			mockUser:      nil,
			expectedUsers: []models.User{}, // not nil
		},
		{
			name: "negative: user has no SCIM external ID; expect empty slice",
			input: &GetSCIMUsersInput{
				Username: ptr.String("user-no-scim"),
			},
			mockUser:      &userWithoutSCIMID,
			expectedUsers: []models.User{}, // not nil
		},
		{
			name: "negative: user managed by another IDP; expect empty slice",
			input: &GetSCIMUsersInput{
				Username: ptr.String("expected-user-name"),
			},
			mockUser:      &otherIDPUser,
			expectedUsers: []models.User{}, // not nil
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			// The list is scoped to the caller's IDP.
			getUsersInput := &db.GetUsersInput{
				Filter: &db.UserFilter{
					SCIMIDPIssuerURL: &issuerURL,
				}}

			if test.input.SCIMExternalID != nil && *test.input.SCIMExternalID != "" {
				externalIDUser := test.mockUser
				if externalIDUser == nil && len(test.expectedUsers) > 0 {
					externalIDUser = &expectedUser
				}
				mockUsers.On("GetUserBySCIMExternalID", mock.Anything, issuerURL, *test.input.SCIMExternalID).Return(externalIDUser, nil)
			} else if test.input.Username != nil && *test.input.Username != "" {
				mockUsers.On("GetUserByTRN", mock.Anything, trn.TypeUser.Build(*test.input.Username)).Return(test.mockUser, nil)
			} else {
				mockUsers.On("GetUsers", mock.Anything, getUsersInput).Return(&db.UsersResult{Users: expectedUsersList}, nil)
			}

			dbClient := &db.Client{
				Users: &mockUsers,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

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
	issuer := "https://example.com/scim"

	sampleUser := &models.User{
		Username:         "input-user",
		Email:            "input-user@example.com",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuer,
		Admin:            false,
		Active:           true,
	}

	input := &CreateSCIMUserInput{
		Email:          "input-user@example.com",
		SCIMExternalID: externalID,
		Active:         true,
	}

	testCases := []struct {
		input                  *CreateSCIMUserInput
		existingEmailUser      *models.User
		existingExternalIDUser *models.User
		returnedSCIMUser       *models.User
		name                   string
		userLinked             bool
		expectLink             bool
		expectedErrorCode      errors.CodeType
	}{
		{
			name:  "positive: user signed in through this IDP exists without a scimExternalID; expect adopted user without a new login link.",
			input: input,
			existingEmailUser: &models.User{
				Metadata: models.ResourceMetadata{ID: resourceUUID},
				Username: "input-user",
				Email:    "input-user@example.com",
				Admin:    false,
				Active:   true,
			},
			userLinked:       true,
			returnedSCIMUser: sampleUser,
		},
		{
			name:  "negative: user exists but has not signed in through this IDP; expect EConflict.",
			input: input,
			existingEmailUser: &models.User{
				Metadata: models.ResourceMetadata{ID: resourceUUID},
				Username: "input-user",
				Email:    "input-user@example.com",
				Active:   true,
			},
			expectedErrorCode: errors.EConflict,
		},
		{
			name:  "negative: user is already managed by SCIM under a different external id; expect EConflict.",
			input: input,
			existingEmailUser: &models.User{
				Metadata:       models.ResourceMetadata{ID: resourceUUID},
				Username:       "input-user",
				Email:          "input-user@example.com",
				SCIMExternalID: "other-external-id",
				Active:         true,
			},
			userLinked:        true,
			expectedErrorCode: errors.EConflict,
		},
		{
			name:  "positive: user exists with external ID but no SCIMExternalID; expect updated user.",
			input: input,
			existingExternalIDUser: &models.User{
				Metadata: models.ResourceMetadata{ID: resourceUUID},
				Username: "input-user",
				Email:    "input-user@example.com",
				Admin:    false,
				Active:   true,
			},
			userLinked:       true,
			returnedSCIMUser: sampleUser,
		},
		{
			name:  "negative: user linked by external ID is managed by another IDP's SCIM client; expect EConflict.",
			input: input,
			existingExternalIDUser: &models.User{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				Username:         "input-user",
				Email:            "input-user@example.com",
				SCIMExternalID:   "other-external-id",
				SCIMIDPIssuerURL: "https://other-idp.example.com",
				Active:           true,
			},
			userLinked:        true,
			expectedErrorCode: errors.EConflict,
		},
		{
			name:  "negative: user linked by external ID is managed by this IDP under another external id; expect EConflict.",
			input: input,
			existingExternalIDUser: &models.User{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				Username:         "input-user",
				Email:            "input-user@example.com",
				SCIMExternalID:   "other-external-id",
				SCIMIDPIssuerURL: issuer,
				Active:           true,
			},
			userLinked:        true,
			expectedErrorCode: errors.EConflict,
		},
		{
			name: "negative: empty external id; expect EInvalid.",
			input: &CreateSCIMUserInput{
				Email:  "input-user@example.com",
				Active: true,
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name:             "positive: user does not already exist in the system; expect user to be created and linked.",
			input:            input,
			returnedSCIMUser: sampleUser,
			expectLink:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			mockUsers.On("GetUserBySCIMExternalID", mock.Anything, issuer, externalID).Return(nil, nil).Maybe()
			mockUsers.On("GetUserByExternalID", mock.Anything, issuer, externalID).Return(test.existingExternalIDUser, nil).Maybe()
			mockUsers.On("GetUserByEmail", mock.Anything, test.input.Email).Return(test.existingEmailUser, nil).Maybe()
			mockUsers.On("HasExternalIdentity", mock.Anything, resourceUUID, &issuer).Return(test.userLinked, nil).Maybe()
			mockUsers.On("UpdateUser", mock.Anything, mock.Anything).Return(test.returnedSCIMUser, nil).Maybe()
			mockUsers.On("CreateUser", mock.Anything, sampleUser).Return(test.returnedSCIMUser, nil).Maybe()
			mockUsers.On("LinkUserWithExternalID", mock.Anything, issuer, externalID, mock.AnythingOfType("string")).Return(nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			dbClient := &db.Client{
				Users:        &mockUsers,
				Transactions: &mockTransactions,
			}

			scimCaller := auth.NewSCIMCaller(mockMaintenanceMonitor, issuer)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

			user, err := service.CreateSCIMUser(auth.WithCaller(ctx, scimCaller), test.input)

			if test.expectLink {
				mockUsers.AssertCalled(t, "LinkUserWithExternalID", mock.Anything, issuer, externalID, mock.AnythingOfType("string"))
			} else {
				mockUsers.AssertNotCalled(t, "LinkUserWithExternalID", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}

			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
				mockUsers.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, test.returnedSCIMUser, user)

			// Whichever path the user took (linked, adopted or created), it is now managed by this IDP.
			if test.existingEmailUser != nil || test.existingExternalIDUser != nil {
				mockUsers.AssertCalled(t, "UpdateUser", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
					return u.SCIMIDPIssuerURL == issuer && u.SCIMExternalID == externalID
				}))
			}
		})
	}
}

func TestUpdateSCIMUser(t *testing.T) {
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
		notLinked         bool // User is managed by another IDP's SCIM client.
		expectedErrorCode errors.CodeType
	}{
		{
			// Only the SCIM external ID changes; the login identity stays linked (asserted below).
			name: "positive: 'replace' operation changing the externalId; expect it to be updated.",
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
				Metadata:       models.ResourceMetadata{ID: resourceUUID},
				Username:       "input-user",
				SCIMExternalID: "new-external-id",
			},
		},
		{
			name: "positive: 'replace' operation with the current externalId; expect no change.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP:    replaceOPType,
						Path:  "externalId",
						Value: externalID,
					},
				},
			},
			existingUser:     sampleUser,
			expectedSCIMUser: sampleUser,
		},
		{
			name: "positive: valid 'replace' operation on email; expect email and username to be updated.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP:    replaceOPType,
						Path:  "emails[type eq \"work\"].value",
						Value: "new.user@example.com",
					},
				},
			},
			existingUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				Username:       "input-user",
				Email:          "input-user@example.com",
				SCIMExternalID: externalID,
			},
			expectedSCIMUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				Username:       "input-user", // Username should NOT change
				Email:          "new.user@example.com",
				SCIMExternalID: externalID,
			},
		},
		{
			name: "negative: user not managed by SCIM; expect error ENotFound",
			input: &UpdateResourceInput{
				ID:         resourceUUID,
				Operations: []Operation{},
			},
			existingUser: &models.User{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				Username: "input-user",
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name: "negative: SCIM user not linked to the caller's IDP; expect error ENotFound",
			input: &UpdateResourceInput{
				ID:         resourceUUID,
				Operations: []Operation{},
			},
			existingUser:      sampleUser,
			notLinked:         true,
			expectedErrorCode: errors.ENotFound,
		},
		{
			name: "negative: 'replace' operation with an empty externalId; expect error EInvalid.",
			input: &UpdateResourceInput{
				ID: resourceUUID,
				Operations: []Operation{
					{
						OP:    replaceOPType,
						Path:  "externalId",
						Value: "",
					},
				},
			},
			existingUser:      sampleUser,
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: invalid Metadata ID; expect error ENotFound",
			input: &UpdateResourceInput{
				ID:         "bogus-id",
				Operations: []Operation{}, // Operation won't matter for this.
			},
			expectedErrorCode: errors.ENotFound,
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			issuer := "https://example.com/scim"

			// SCIM users are managed by the caller's IDP, or by another one when notLinked; a user without a
			// SCIM external ID has no issuer. Copies, since the cases share user models.
			existingUser, expectedSCIMUser := test.existingUser, test.expectedSCIMUser
			if existingUser != nil && existingUser.SCIMExternalID != "" {
				u := *existingUser
				u.SCIMIDPIssuerURL = issuer
				if test.notLinked {
					u.SCIMIDPIssuerURL = "https://other-idp.example.com"
				}
				existingUser = &u
			}
			if expectedSCIMUser != nil {
				u := *expectedSCIMUser
				u.SCIMIDPIssuerURL = issuer
				expectedSCIMUser = &u
			}

			// This function just updates the user in the db layer.
			// Actual fields are updated via a private func.
			mockUsers.On("UpdateUser", mock.Anything, expectedSCIMUser).Return(expectedSCIMUser, nil).Maybe()
			mockUsers.On("GetUserByID", mock.Anything, test.input.ID).Return(existingUser, nil).Maybe()

			dbClient := &db.Client{
				Users:        &mockUsers,
				Transactions: &mockTransactions,
			}

			scimCaller := auth.NewSCIMCaller(mockMaintenanceMonitor, issuer)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

			user, err := service.UpdateSCIMUser(auth.WithCaller(ctx, scimCaller), test.input)

			// Login identities must never change through a SCIM update.
			mockUsers.AssertNotCalled(t, "LinkUserWithExternalID", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

			if test.expectedErrorCode != "" {
				// Negative cases.
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				// Positive case.
				assert.Equal(t, expectedSCIMUser, user)
			}
		})
	}
}

func TestDeleteSCIMUser(t *testing.T) {
	issuer := "https://test-idp.example.com"

	testCases := []struct {
		name              string
		existingUser      *models.User
		expectedErrorCode errors.CodeType
	}{
		{
			name: "positive: user is managed by the caller's IDP; expect no error.",
			existingUser: &models.User{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: issuer,
			},
		},
		{
			name: "negative: user is managed by another IDP; expect error ENotFound.",
			existingUser: &models.User{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: "https://other-idp.example.com",
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name: "negative: user being deleted was not created via SCIM; expect error ENotFound.",
			existingUser: &models.User{
				Metadata: models.ResourceMetadata{ID: resourceUUID},
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: user does not exist; expect error ENotFound.",
			expectedErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockUsers := db.NewMockUsers(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			ctx := auth.WithCaller(context.Background(), auth.NewSCIMCaller(mockMaintenanceMonitor, issuer))

			mockUsers.On("GetUserByID", mock.Anything, resourceUUID).Return(test.existingUser, nil)
			if test.expectedErrorCode == "" {
				mockUsers.On("DeleteUser", mock.Anything, test.existingUser).Return(nil)
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &db.Client{Users: mockUsers}, nil, []config.IdpConfig{})

			err := service.DeleteSCIMUser(ctx, &DeleteSCIMResourceInput{ID: resourceUUID})
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestGetSCIMGroups(t *testing.T) {
	issuerURL := "https://test-idp.example.com"
	ctx := auth.WithCaller(context.Background(), auth.NewSCIMCaller(nil, issuerURL))

	expectedSCIMGroup := models.Team{
		Name:             "expected-scim-group",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuerURL,
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}
	expectedSCIMGroupsList := []models.Team{expectedSCIMGroup}

	testCases := []struct {
		name               string
		input              *GetSCIMGroupsInput
		expectedError      string
		expectedSCIMGroups []models.Team
	}{
		{
			// The external ID only exists for another IDP; the lookup is scoped to the caller's IDP.
			name: "negative: SCIMExternalID of another IDP's team; expect no groups.",
			input: &GetSCIMGroupsInput{
				SCIMExternalID: getFilter("externalId", expectedSCIMGroup.SCIMExternalID),
			},
		},
		{
			name: "positive: SCIMExternalID is valid; expect a slice of SCIM groups (teams).",
			input: &GetSCIMGroupsInput{
				SCIMExternalID: getFilter("externalId", expectedSCIMGroup.SCIMExternalID),
			},
			expectedSCIMGroups: expectedSCIMGroupsList,
			// expect nil errors
		},
		{
			name:               "positive: SCIMExternalID is empty; expect a slice of all SCIM groups (teams).",
			input:              &GetSCIMGroupsInput{},
			expectedSCIMGroups: expectedSCIMGroupsList,
		},
		{
			name: "negative: input has an invalid scimExternalID; expect a nil slice of users.",
			input: &GetSCIMGroupsInput{
				SCIMExternalID: "invalid-id",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			// The list is scoped to the caller's IDP.
			getTeamsInput := &db.GetTeamsInput{
				Filter: &db.TeamFilter{
					SCIMIDPIssuerURL: &issuerURL,
				},
			}

			var returnedSCIMGroup *models.Team
			if len(test.expectedSCIMGroups) > 0 {
				returnedSCIMGroup = &expectedSCIMGroup
			}

			mockTeams.On("GetTeamBySCIMExternalID", mock.Anything, issuerURL, test.input.SCIMExternalID).Return(returnedSCIMGroup, nil).Maybe()
			mockTeams.On("GetTeams", mock.Anything, getTeamsInput).Return(&db.TeamsResult{Teams: expectedSCIMGroupsList}, nil).Maybe()

			dbClient := &db.Client{
				Teams: &mockTeams,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

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
	issuerURL := "https://test-idp.example.com"
	sampleSCIMGroup := &models.Team{
		Name:             "input-team",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuerURL,
	}

	testCases := []struct {
		input             *CreateSCIMGroupInput
		existingSCIMGroup *models.Team
		returnedSCIMGroup *models.Team
		name              string
		expectedErrorCode errors.CodeType
	}{
		{
			name: "negative: empty external id; expect EInvalid.",
			input: &CreateSCIMGroupInput{
				Name: sampleSCIMGroup.Name,
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: manually managed team with the same name exists; expect EConflict.",
			input: &CreateSCIMGroupInput{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: sampleSCIMGroup.SCIMExternalID,
			},
			existingSCIMGroup: &models.Team{
				Name: sampleSCIMGroup.Name,
			},
			expectedErrorCode: errors.EConflict,
		},
		{
			name: "positive: SCIM-managed team with the same name exists; expect updated SCIM group.",
			input: &CreateSCIMGroupInput{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: sampleSCIMGroup.SCIMExternalID,
			},
			existingSCIMGroup: &models.Team{
				Name:             sampleSCIMGroup.Name,
				SCIMExternalID:   "previous-external-id",
				SCIMIDPIssuerURL: issuerURL,
			},
			returnedSCIMGroup: sampleSCIMGroup,
		},
		{
			name: "negative: another IDP's team with the same name exists; expect EConflict.",
			input: &CreateSCIMGroupInput{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: sampleSCIMGroup.SCIMExternalID,
			},
			existingSCIMGroup: &models.Team{
				Name:             sampleSCIMGroup.Name,
				SCIMExternalID:   "previous-external-id",
				SCIMIDPIssuerURL: "https://other-idp.example.com",
			},
			expectedErrorCode: errors.EConflict,
		},
		{
			// Existing SCIM teams have no issuer until they're backfilled, so none is adopted.
			name: "negative: SCIM team with no IDP recorded exists; expect EConflict.",
			input: &CreateSCIMGroupInput{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: sampleSCIMGroup.SCIMExternalID,
			},
			existingSCIMGroup: &models.Team{
				Name:           sampleSCIMGroup.Name,
				SCIMExternalID: "previous-external-id",
			},
			expectedErrorCode: errors.EConflict,
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			maintenanceMonitor := maintenance.NewMockMonitor(t)
			maintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
			scimCaller := auth.NewSCIMCaller(maintenanceMonitor, issuerURL)

			mockTeams.On("GetTeamByTRN", mock.Anything, trn.TypeTeam.Build(test.input.Name)).Return(test.existingSCIMGroup, nil).Maybe()
			mockTeams.On("UpdateTeam", mock.Anything, test.existingSCIMGroup).Return(test.returnedSCIMGroup, nil).Maybe()
			mockTeams.On("CreateTeam", mock.Anything, sampleSCIMGroup).Return(test.returnedSCIMGroup, nil).Maybe()

			dbClient := &db.Client{
				Teams: &mockTeams,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

			group, err := service.CreateSCIMGroup(auth.WithCaller(ctx, scimCaller), test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
				mockTeams.AssertNotCalled(t, "UpdateTeam", mock.Anything, mock.Anything)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, test.returnedSCIMGroup, group)
		})
	}
}

func TestUpdateSCIMGroup(t *testing.T) {
	scimGroupID := "663a2ffa-5e95-41fa-9119-c7bcad3d775e"
	issuerURL := "https://test-idp.example.com"

	// sampleSCIMGroup is the model for several user inputs.
	sampleSCIMGroup := &models.Team{
		Metadata: models.ResourceMetadata{
			ID: scimGroupID,
		},
		Name:             "input-team",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuerURL,
	}

	// sampleUser is used when adding or removing SCIM group (team) member.
	sampleUser := &models.User{
		Username: "sample-users-name",
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		SCIMExternalID:   "user-external-id",
		SCIMIDPIssuerURL: issuerURL,
	}

	// unmanagedUser is not managed by the group's IDP.
	unmanagedUser := &models.User{
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
		memberUser              *models.User       // User the member operation refers to; sampleUser if nil.
		expectedSCIMGroup       *models.Team       // Updated SCIM Group (team).
		name                    string
		expectedErrorCode       errors.CodeType
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
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: issuerURL,
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
							"value": gid.ToGlobalID(types.UserModelType, resourceUUID),
						},
					},
				},
			},
			expectedSCIMGroup: sampleSCIMGroup, // Team model does not update for modifying team members.
		},
		{
			name:              "negative: 'add' operation for a user not managed by the group's IDP; expect ENotFound.",
			inputID:           scimGroupID,
			existingSCIMGroup: sampleSCIMGroup,
			memberUser:        unmanagedUser,
			operations: []Operation{
				{
					OP:   addOPType,
					Path: "members",
					Value: []interface{}{
						map[string]interface{}{
							"value": gid.ToGlobalID(types.UserModelType, resourceUUID),
						},
					},
				},
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: 'replace' operation with an empty externalId; expect EInvalid.",
			inputID:           scimGroupID,
			existingSCIMGroup: sampleSCIMGroup,
			operations: []Operation{
				{
					OP:    replaceOPType,
					Path:  "externalId",
					Value: "",
				},
			},
			expectedErrorCode: errors.EInvalid,
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
							"value": gid.ToGlobalID(types.UserModelType, resourceUUID),
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
							"value": gid.ToGlobalID(types.UserModelType, resourceUUID),
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
			name:              "negative: invalid Metadata ID; expect error ENotFound",
			inputID:           "bogus-id",
			operations:        []Operation{}, // Operation won't matter for this.
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:    "negative: team is managed by another IDP; expect error ENotFound",
			inputID: scimGroupID,
			existingSCIMGroup: &models.Team{
				Metadata:         models.ResourceMetadata{ID: scimGroupID},
				Name:             "input-team",
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: "https://other-idp.example.com",
			},
			operations: []Operation{
				{
					OP:    replaceOPType,
					Path:  "externalId",
					Value: "new-external-id",
				},
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:    "negative: team is not managed by SCIM; expect error ENotFound",
			inputID: scimGroupID,
			existingSCIMGroup: &models.Team{
				Metadata: models.ResourceMetadata{ID: scimGroupID},
				Name:     "input-team",
			},
			operations: []Operation{
				{
					OP:    replaceOPType,
					Path:  "externalId",
					Value: "new-external-id",
				},
			},
			expectedErrorCode: errors.ENotFound,
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockTeams := db.MockTeams{}
			mockTeams.Test(t)

			mockTeamMembers := db.MockTeamMembers{}
			mockTeamMembers.Test(t)

			mockUsers := db.MockUsers{}
			mockUsers.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			scimCaller := auth.NewSCIMCaller(mockMaintenanceMonitor, issuerURL)

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
			memberUser := test.memberUser
			if memberUser == nil {
				memberUser = sampleUser
			}
			mockUsers.On("GetUserByID", mock.Anything, sampleUser.Metadata.ID).Return(memberUser, nil)

			dbClient := &db.Client{
				Teams:        &mockTeams,
				TeamMembers:  &mockTeamMembers,
				Users:        &mockUsers,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, []config.IdpConfig{})

			input := &UpdateResourceInput{
				ID:         test.inputID,
				Operations: test.operations,
			}

			group, err := service.UpdateSCIMGroup(auth.WithCaller(ctx, scimCaller), input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
				mockTeamMembers.AssertNotCalled(t, "AddUserToTeam", mock.Anything, mock.Anything)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedSCIMGroup, group)
			}
		})
	}
}

func TestDeleteSCIMGroup(t *testing.T) {
	issuer := "https://test-idp.example.com"

	testCases := []struct {
		name              string
		existingSCIMGroup *models.Team
		expectedErrorCode errors.CodeType
	}{
		{
			name: "positive: team is managed by the caller's IDP; expect no error.",
			existingSCIMGroup: &models.Team{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: issuer,
			},
		},
		{
			name: "negative: team is managed by another IDP; expect error ENotFound.",
			existingSCIMGroup: &models.Team{
				Metadata:         models.ResourceMetadata{ID: resourceUUID},
				SCIMExternalID:   externalID,
				SCIMIDPIssuerURL: "https://other-idp.example.com",
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name: "negative: team being deleted was not created via SCIM; expect error ENotFound.",
			existingSCIMGroup: &models.Team{
				Metadata: models.ResourceMetadata{ID: resourceUUID},
			},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: team does not exist; expect error ENotFound.",
			expectedErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockTeams := db.NewMockTeams(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			ctx := auth.WithCaller(context.Background(), auth.NewSCIMCaller(mockMaintenanceMonitor, issuer))

			mockTeams.On("GetTeamByID", mock.Anything, resourceUUID).Return(test.existingSCIMGroup, nil)
			if test.expectedErrorCode == "" {
				mockTeams.On("DeleteTeam", mock.Anything, test.existingSCIMGroup).Return(nil)
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &db.Client{Teams: mockTeams}, nil, []config.IdpConfig{})

			err := service.DeleteSCIMGroup(ctx, &DeleteSCIMResourceInput{ID: resourceUUID})
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
				return
			}
			assert.Nil(t, err)
		})
	}
}

// getFilter is a helper function to prepare a SCIM resource filter.
func getFilter(attribute, value string) string {
	return attribute + " Eq \"" + value + "\""
}

// TestSCIMListsRequireSCIMCaller checks that only the IDP's SCIM client can list SCIM users and
// groups: the results include every provisioned user and team and their SCIM external IDs, while the
// SCIM routes only require an authenticated caller. The empty DB client fails the test on any query,
// so a refused caller is refused before anything is read.
func TestSCIMListsRequireSCIMCaller(t *testing.T) {
	callers := map[string]auth.Caller{
		"user":            &auth.UserCaller{User: &models.User{Username: "someone"}},
		"service account": &auth.ServiceAccountCaller{ServiceAccountPath: "group/sa"},
		"job":             &auth.JobCaller{Job: &models.Job{Metadata: models.ResourceMetadata{ID: "job-1"}}},
	}

	for name, caller := range callers {
		t.Run(name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), caller)
			service := &service{dbClient: &db.Client{}}

			for _, input := range []*GetSCIMUsersInput{{}, {SCIMExternalID: ptr.String(externalID)}, {Username: ptr.String("someone")}} {
				_, err := service.GetSCIMUsers(ctx, input)
				assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
			}

			for _, input := range []*GetSCIMGroupsInput{{}, {SCIMExternalID: externalID}} {
				_, err := service.GetSCIMGroups(ctx, input)
				assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
			}
		})
	}
}

// TestSCIMMutationsRequireSCIMCaller checks that a caller other than the IDP's SCIM client, such as an admin
// user who has the permission, can't update or delete users or teams through SCIM: only a SCIM caller has an
// IDP to check ownership against. The empty DB client fails the test on any query, so a refused caller is
// refused before anything is read.
func TestSCIMMutationsRequireSCIMCaller(t *testing.T) {
	mockCaller := auth.NewMockCaller(t)
	mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx := auth.WithCaller(context.Background(), mockCaller)
	service := &service{dbClient: &db.Client{}}

	_, err := service.UpdateSCIMUser(ctx, &UpdateResourceInput{ID: resourceUUID})
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err), "UpdateSCIMUser")

	err = service.DeleteSCIMUser(ctx, &DeleteSCIMResourceInput{ID: resourceUUID})
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err), "DeleteSCIMUser")

	_, err = service.UpdateSCIMGroup(ctx, &UpdateResourceInput{ID: resourceUUID})
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err), "UpdateSCIMGroup")

	err = service.DeleteSCIMGroup(ctx, &DeleteSCIMResourceInput{ID: resourceUUID})
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err), "DeleteSCIMGroup")
}
