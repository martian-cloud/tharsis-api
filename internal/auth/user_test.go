package auth

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestUserCaller_GetSubject(t *testing.T) {
	caller := UserCaller{User: &models.User{Email: "user@email"}}
	assert.Equal(t, "user@email", caller.GetSubject())
}

func TestUserCaller_IsAdminModeActivated(t *testing.T) {
	// IsAdminModeActivated re-queries the latest user, so the result tracks the DB, not the
	// in-memory caller.
	t.Run("admin mode inactive", func(t *testing.T) {
		// A non-admin user short-circuits to false without querying the DB.
		user := &models.User{Metadata: models.ResourceMetadata{ID: "u1"}}
		mockUsers := db.NewMockUsers(t)
		caller := UserCaller{User: user, dbClient: &db.Client{Users: mockUsers}}
		assert.False(t, caller.IsAdminModeActivated(t.Context()))
	})

	t.Run("admin mode active", func(t *testing.T) {
		expiration := time.Now().Add(time.Hour)
		user := &models.User{Metadata: models.ResourceMetadata{ID: "u1"}, Admin: true, AdminModeExpiration: &expiration}
		mockUsers := db.NewMockUsers(t)
		mockUsers.On("GetUserByID", mock.Anything, "u1").Return(user, nil)
		caller := UserCaller{User: user, dbClient: &db.Client{Users: mockUsers}}
		assert.True(t, caller.IsAdminModeActivated(t.Context()))
	})
}

func TestUserCaller_GetRootNamespaceMemberships(t *testing.T) {
	caller := UserCaller{User: &models.User{}}
	ctx := WithCaller(context.Background(), &caller)

	expectedNamespaces := []models.MembershipNamespace{{ID: "nm-1"}}

	mockAuthorizer := NewMockAuthorizer(t)
	mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return(expectedNamespaces, nil)
	caller.authorizer = mockAuthorizer

	namespaces, err := caller.GetRootNamespaceMemberships(ctx)
	assert.Nil(t, err)
	assert.Equal(t, expectedNamespaces, namespaces)
}

func TestUserCaller_GetNamespacePermissions(t *testing.T) {
	caller := UserCaller{User: &models.User{}}

	effectivePerms := map[string]models.Permission{
		models.ViewWorkspacePermission.String(): models.ViewWorkspacePermission,
	}

	mockAuthorizer := NewMockAuthorizer(t)
	mockAuthorizer.On("GetEffectivePermissions", mock.Anything, "some/namespace").Return(effectivePerms, nil)
	caller.authorizer = mockAuthorizer

	perms, err := caller.GetNamespacePermissions(context.Background(), "some/namespace")
	assert.NoError(t, err)
	assert.Equal(t, []*models.Permission{&models.ViewWorkspacePermission}, perms)
}

func TestUserCaller_RequirePermissions(t *testing.T) {
	teamID := "team1"
	caller := UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Email: "user@email"}}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name              string
		expectErrorCode   errors.CodeType
		teamMember        *models.TeamMember
		perm              models.Permission
		constraints       []func(*constraints)
		isAdmin           bool
		isAdminNoMode     bool
		withAuthorizer    bool
		inMaintenanceMode bool
	}{
		{
			name:           "access is granted by the authorizer",
			perm:           models.ViewGroupPermission,
			constraints:    []func(*constraints){WithNamespacePath("namespace")},
			withAuthorizer: true,
		},
		{
			name:            "access denied by the authorizer because a permission is not satisfied",
			perm:            models.DeleteGroupPermission,
			constraints:     []func(*constraints){WithNamespacePath("namespace")},
			expectErrorCode: errors.ENotFound,
			withAuthorizer:  true,
		},
		{
			name:    "permissions are only granted since user is admin",
			perm:    models.CreateTeamPermission,
			isAdmin: true,
		},
		{
			name:            "admin without active admin mode is denied assignable permission",
			perm:            models.CreateTeamPermission,
			isAdminNoMode:   true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access forbidden because user must be an admin",
			perm:            models.CreateTeamPermission,
			constraints:     []func(*constraints){WithGroupID("team-1")},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "team update allowed since user is an admin",
			perm:        models.UpdateTeamPermission,
			constraints: []func(*constraints){WithTeamID(teamID)},
			isAdmin:     true,
		},
		{
			name:            "access denied because user is not an admin or a team maintainer",
			teamMember:      &models.TeamMember{IsMaintainer: false},
			perm:            models.UpdateTeamPermission,
			constraints:     []func(*constraints){WithTeamID(teamID)},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "access denied because team member not found",
			perm:            models.UpdateTeamPermission,
			constraints:     []func(*constraints){WithTeamID(teamID)},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "access denied because user is not a maintainer",
			teamMember:      &models.TeamMember{IsMaintainer: false},
			perm:            models.UpdateTeamPermission,
			constraints:     []func(*constraints){WithTeamID(teamID)},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "access denied because required constraints are not specified",
			perm:            models.ViewWorkspacePermission,
			expectErrorCode: errors.EInternal,
			withAuthorizer:  true,
		},
		{
			name:              "cannot have write permission when system is in maintenance mode",
			perm:              models.CreateWorkspacePermission,
			expectErrorCode:   errors.EServiceUnavailable,
			inMaintenanceMode: true,
		},
		{
			name:              "can have read permission when system is in maintenance mode",
			perm:              models.ViewWorkspacePermission,
			constraints:       []func(*constraints){WithWorkspaceID("ws-1")},
			withAuthorizer:    true,
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			if test.perm == models.UpdateTeamPermission && !test.isAdmin {
				mockTeamMembers.On("GetTeamMember", mock.Anything, caller.User.Metadata.ID, teamID).Return(test.teamMember, nil)
			}

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{test.perm}, mock.Anything).Return(requireAccessAuthorizerFunc)
			}

			caller.User.Admin = test.isAdmin || test.isAdminNoMode
			caller.authorizer = mockAuthorizer
			if test.isAdmin {
				t := time.Now().Add(time.Hour)
				caller.User.AdminModeExpiration = &t
			} else {
				caller.User.AdminModeExpiration = nil
			}
			caller.maintenanceMonitor = mockMaintenanceMonitor
			caller.dbClient = &db.Client{TeamMembers: mockTeamMembers}

			err := caller.RequirePermission(ctx, test.perm, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestUserCaller_RequireInheritedPermissions(t *testing.T) {
	caller := UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Email: "user@email"}}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name            string
		expectErrorCode errors.CodeType
		modelType       types.ModelType
		constraints     []func(*constraints)
		isAdmin         bool
		isAdminNoMode   bool
		withAuthorizer  bool
	}{
		{
			name:           "multiple permissions granted by the authorizer",
			modelType:      types.ManagedIdentityModelType,
			constraints:    []func(*constraints){WithGroupID("group1")},
			withAuthorizer: true,
		},
		{
			name:            "access denied by the authorizer because a permission is not satisfied",
			modelType:       types.RunModelType, // Just using an arbitrary non-inheritable resource here to deny access.
			constraints:     []func(*constraints){WithWorkspaceID("ws2")},
			expectErrorCode: errors.ENotFound,
			withAuthorizer:  true,
		},
		{
			name:      "permissions granted since user is admin",
			modelType: types.GPGKeyModelType,
			isAdmin:   true,
		},
		{
			name:           "admin without admin mode falls through to authorizer",
			modelType:      types.GPGKeyModelType,
			isAdminNoMode:  true,
			constraints:    []func(*constraints){WithGroupID("group1")},
			withAuthorizer: true,
		},
		{
			name:            "access denied because required constraints are not specified",
			modelType:       types.TerraformModuleModelType,
			expectErrorCode: errors.EInternal,
			withAuthorizer:  true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccessToInheritableResource", mock.Anything, []types.ModelType{test.modelType}, mock.Anything).Return(requireInheritedAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer
			caller.User.Admin = test.isAdmin || test.isAdminNoMode

			if test.isAdmin {
				t := time.Now().Add(time.Hour)
				caller.User.AdminModeExpiration = &t
			} else {
				caller.User.AdminModeExpiration = nil
			}
			err := caller.RequireAccessToInheritableResource(ctx, test.modelType, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestUserAuth_createUserWithExternalID(t *testing.T) {
	testCases := []struct {
		name                string
		adminUserEmail      string
		email               string
		allowAdminBootstrap bool
		expectAdmin         bool
	}{
		{name: "email matches the configured admin email", adminUserEmail: "Admin@Example.com", email: "admin@example.com", allowAdminBootstrap: true, expectAdmin: true},
		{name: "email matches but the IDP is not allowed to bootstrap the admin", adminUserEmail: "admin@example.com", email: "admin@example.com"},
		{name: "email does not match the configured admin email", adminUserEmail: "admin@example.com", email: "user@example.com", allowAdminBootstrap: true},
		{name: "no admin email configured", email: "admin@example.com", allowAdminBootstrap: true},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockUsers := db.NewMockUsers(t)
			mockTransactions := db.NewMockTransactions(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockUsers.On("CreateUser", mock.Anything, mock.Anything).Return(func(_ context.Context, user *models.User) (*models.User, error) {
				user.Metadata.ID = "user-1"
				return user, nil
			})
			mockUsers.On("LinkUserWithExternalID", mock.Anything, "https://idp.example.com", "sub-1", "user-1").Return(nil)

			testLogger, _ := logger.NewForTest()
			idps := []config.IdpConfig{{IssuerURL: "https://idp.example.com", AllowAdminBootstrap: test.allowAdminBootstrap}}
			userAuth := NewUserAuth(ctx, idps, testLogger, &db.Client{Users: mockUsers, Transactions: mockTransactions}, nil, newTestOIDCConfigFetcher(t), test.adminUserEmail)

			user, err := userAuth.createUserWithExternalID(ctx, &externalIdentity{
				Issuer:   "https://idp.example.com",
				ID:       "sub-1",
				Username: ParseUsername(test.email),
				Email:    test.email,
			})
			require.NoError(t, err)
			assert.Equal(t, test.expectAdmin, user.Admin)
			assert.True(t, user.Active)
		})
	}
}

func TestUserAuth_Authenticate_ExternalIDClaim(t *testing.T) {
	issuer := "https://login.example.com/tenant/v2.0"
	clientID := "tharsis"
	existingUser := &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}, Username: "jdoe", Email: "jdoe@example.com", Active: true}

	testCases := []struct {
		name            string
		externalIDClaim string
		omitOID         bool
		email           any // Defaults to a valid email.
		userByOID       *models.User
		userBySub       *models.User
		linkErr         error
		expectLinkOID   bool
		expectCreate    bool
		expectErrorCode errors.CodeType
	}{
		{
			name:      "default claim matches the user linked by sub",
			userBySub: existingUser,
		},
		{
			name:            "oid claim matches the user linked by oid",
			externalIDClaim: "oid",
			userByOID:       existingUser,
		},
		{
			name:            "oid claim falls back to the user linked by sub and links the oid",
			externalIDClaim: "oid",
			userBySub:       existingUser,
			expectLinkOID:   true,
		},
		{
			name:            "oid link conflict from a concurrent sign-in is ignored",
			externalIDClaim: "oid",
			userBySub:       existingUser,
			linkErr:         errors.New("already linked", errors.WithErrorCode(errors.EConflict)),
			expectLinkOID:   true,
		},
		{
			name:            "oid claim creates a new user linked by oid",
			externalIDClaim: "oid",
			expectCreate:    true,
		},
		{
			name:            "missing oid claim is rejected rather than falling back to sub",
			externalIDClaim: "oid",
			omitOID:         true,
			userBySub:       existingUser,
			expectErrorCode: errors.EUnauthorized,
		},
		{
			// The claim is only read for a new user, and must be rejected rather than panic.
			name:            "a new user with a non-string email claim is rejected",
			email:           []string{"jdoe@example.com"},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			email := test.email
			if email == nil {
				email = "jdoe@example.com"
			}

			token := jwt.New()
			require.NoError(t, token.Set(jwt.IssuerKey, issuer))
			require.NoError(t, token.Set(jwt.AudienceKey, []string{clientID}))
			require.NoError(t, token.Set(jwt.SubjectKey, "pairwise-sub"))
			require.NoError(t, token.Set("preferred_username", "jdoe@example.com"))
			require.NoError(t, token.Set("email", email))
			if !test.omitOID {
				require.NoError(t, token.Set("oid", "object-id"))
			}

			mockVerifier := NewMockOIDCTokenVerifier(t)
			mockVerifier.On("VerifyToken", mock.Anything, "token", mock.Anything).Return(token, nil)

			mockUsers := db.NewMockUsers(t)
			mockTransactions := db.NewMockTransactions(t)

			if test.expectErrorCode == "" || test.email != nil {
				if test.externalIDClaim == "oid" {
					mockUsers.On("GetUserByExternalID", mock.Anything, issuer, "object-id").Return(test.userByOID, nil)
				}
				if test.userByOID == nil {
					mockUsers.On("GetUserByExternalID", mock.Anything, issuer, "pairwise-sub").Return(test.userBySub, nil)
				}
			}

			if test.expectLinkOID {
				mockUsers.On("LinkUserWithExternalID", mock.Anything, issuer, "object-id", "user-1").Return(test.linkErr)
			}

			if test.expectCreate {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockUsers.On("CreateUser", mock.Anything, mock.Anything).Return(&models.User{Metadata: models.ResourceMetadata{ID: "user-2"}, Active: true}, nil)
				mockUsers.On("LinkUserWithExternalID", mock.Anything, issuer, "object-id", "user-2").Return(nil)
			}

			testLogger, _ := logger.NewForTest()
			userAuth := NewUserAuth(ctx, []config.IdpConfig{{
				IssuerURL:       issuer,
				ClientID:        clientID,
				UsernameClaim:   "preferred_username",
				ExternalIDClaim: test.externalIDClaim,
			}}, testLogger, &db.Client{Users: mockUsers, Transactions: mockTransactions}, nil, newTestOIDCConfigFetcher(t), "")
			userAuth.oidcTokenVerifier = mockVerifier

			caller, err := userAuth.Authenticate(ctx, "token", false)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			userCaller, ok := caller.(*UserCaller)
			require.True(t, ok)

			if test.expectCreate {
				assert.Equal(t, "user-2", userCaller.User.Metadata.ID)
			} else {
				assert.Equal(t, existingUser.Metadata.ID, userCaller.User.Metadata.ID)
			}
		})
	}
}

func TestUserAuth_createUser_EmailLinking(t *testing.T) {
	identity := &externalIdentity{
		Issuer:   "https://idp.example.com",
		ID:       "object-id",
		Username: "jdoe",
		Email:    "jdoe@example.com",
	}
	emailUser := &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}, Username: "jdoe", Email: "jdoe@example.com", Active: true}

	testCases := []struct {
		name               string
		emailLinking       bool
		concurrentUser     *models.User
		emailUserLinked    bool
		expectLink         bool
		expectUser         *models.User
		expectErrorCode    errors.CodeType
		skipEmailUserCheck bool
	}{
		{
			name:            "email linking disabled; existing user with the same email is not linked",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "email linking enabled; links an existing user with no external identity",
			emailLinking:    true,
			expectLink:      true,
			expectUser:      emailUser,
			emailUserLinked: false,
		},
		{
			name:            "email linking enabled; refuses a user who already has an external identity",
			emailLinking:    true,
			emailUserLinked: true,
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:               "user for this identity was created concurrently; returns it without email matching",
			concurrentUser:     emailUser,
			expectUser:         emailUser,
			skipEmailUserCheck: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockUsers := db.NewMockUsers(t)
			mockTransactions := db.NewMockTransactions(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

			mockUsers.On("CreateUser", mock.Anything, mock.Anything).Return(nil, errors.New("username taken", errors.WithErrorCode(errors.EConflict)))
			mockUsers.On("GetUserByExternalID", mock.Anything, identity.Issuer, identity.ID).Return(test.concurrentUser, nil)

			if !test.skipEmailUserCheck {
				mockUsers.On("GetUserByEmail", mock.Anything, identity.Email).Return(emailUser, nil)
			}

			if test.emailLinking {
				mockUsers.On("HasExternalIdentity", mock.Anything, emailUser.Metadata.ID, (*string)(nil)).Return(test.emailUserLinked, nil)
			}

			if test.expectLink {
				mockUsers.On("LinkUserWithExternalID", mock.Anything, identity.Issuer, identity.ID, emailUser.Metadata.ID).Return(nil)
			}

			testLogger, _ := logger.NewForTest()
			userAuth := NewUserAuth(ctx, nil, testLogger, &db.Client{Users: mockUsers, Transactions: mockTransactions}, nil, newTestOIDCConfigFetcher(t), "")

			user, err := userAuth.createUser(ctx, identity, test.emailLinking)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectUser, user)
		})
	}
}

func TestUserAuth_GetUsernameClaim(t *testing.T) {
	issuer := "https://idp.example.com"

	tests := []struct {
		name              string
		issuer            string
		username          any
		expectUsername    string
		expectErrorCode   errors.CodeType
		expectAnyError    bool
		omitUsernameClaim bool
	}{
		{name: "returns the username claim", issuer: issuer, username: "jdoe", expectUsername: "jdoe"},
		{name: "a missing username claim is unauthorized", issuer: issuer, omitUsernameClaim: true, expectErrorCode: errors.EUnauthorized},
		{name: "a non-string username claim is unauthorized", issuer: issuer, username: 42, expectErrorCode: errors.EUnauthorized},
		{name: "an object username claim is unauthorized", issuer: issuer, username: map[string]any{"a": "b"}, expectErrorCode: errors.EUnauthorized},
		{name: "an unknown issuer is an error", issuer: "https://other.example.com", username: "jdoe", expectAnyError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			userAuth := NewUserAuth(context.Background(), []config.IdpConfig{{
				IssuerURL:     issuer,
				ClientID:      "client",
				UsernameClaim: "preferred_username",
			}}, testLogger, &db.Client{}, nil, newTestOIDCConfigFetcher(t), "")

			token := jwt.New()
			require.NoError(t, token.Set(jwt.IssuerKey, test.issuer))
			if !test.omitUsernameClaim {
				require.NoError(t, token.Set("preferred_username", test.username))
			}

			username, err := userAuth.GetUsernameClaim(token)
			switch {
			case test.expectErrorCode != "":
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			case test.expectAnyError:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, test.expectUsername, username)
			}
		})
	}
}

func TestGetExternalIDClaim(t *testing.T) {
	testCases := []struct {
		name               string
		claim              string
		value              any
		expectValue        string
		expectErrorMessage string
	}{
		{
			name:        "sub claim",
			claim:       subjectClaim,
			expectValue: "pairwise-sub",
		},
		{
			name:        "string custom claim",
			claim:       "oid",
			value:       "object-id",
			expectValue: "object-id",
		},
		{
			name:               "missing custom claim",
			claim:              "oid",
			expectErrorMessage: "token with issuer https://idp.example.com is missing the oid claim used to identify users",
		},
		{
			name:               "empty custom claim",
			claim:              "oid",
			value:              "",
			expectErrorMessage: "token with issuer https://idp.example.com is missing the oid claim used to identify users",
		},
		{
			name:               "non-string custom claim",
			claim:              "oid",
			value:              123456,
			expectErrorMessage: "token with issuer https://idp.example.com has a non-string oid claim used to identify users",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			token := jwt.New()
			require.NoError(t, token.Set(jwt.IssuerKey, "https://idp.example.com"))
			require.NoError(t, token.Set(jwt.SubjectKey, "pairwise-sub"))
			if test.value != nil {
				require.NoError(t, token.Set(test.claim, test.value))
			}

			value, err := getExternalIDClaim(token, test.claim)
			if test.expectErrorMessage != "" {
				assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
				assert.Equal(t, test.expectErrorMessage, errors.ErrorMessage(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectValue, value)
		})
	}
}
