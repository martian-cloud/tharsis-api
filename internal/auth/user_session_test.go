package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestNewUserSessionManager(t *testing.T) {
	tests := []struct {
		name                string
		tharsisAPIURL       string
		tharsisUIURL        string
		expectErrorMessage  string
		expectSecureCookies bool
		expectUIDomain      string
	}{
		{
			name:                "successful creation with https",
			tharsisAPIURL:       "https://api.example.com",
			tharsisUIURL:        "https://ui.example.com",
			expectSecureCookies: true,
			expectUIDomain:      "ui.example.com",
		},
		{
			name:                "successful creation with http",
			tharsisAPIURL:       "http://api.example.com",
			tharsisUIURL:        "http://ui.example.com",
			expectSecureCookies: false,
			expectUIDomain:      "ui.example.com",
		},
		{
			name:               "invalid ui url",
			tharsisAPIURL:      "https://api.example.com",
			tharsisUIURL:       "://invalid-url",
			expectErrorMessage: "failed to parse tharsis ui url",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockDBClient := &db.Client{}
			mockSigningKeyManager := &MockSigningKeyManager{}
			mockAuthenticator := &MockAuthenticator{}
			logger, _ := logger.NewForTest()

			manager, err := NewUserSessionManager(
				mockDBClient,
				mockSigningKeyManager,
				mockAuthenticator,
				logger,
				15,
				60,
				5,
				test.tharsisAPIURL,
				test.tharsisUIURL,
				true,
			)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, manager)
			} else {
				require.NoError(t, err)
				require.NotNil(t, manager)

				impl := manager.(*userSessionManager)
				assert.Equal(t, test.expectSecureCookies, impl.enableSecureCookies)
				assert.Equal(t, test.expectUIDomain, impl.tharsisUIDomain)
			}
		})
	}
}

func TestUserSessionManager_GetCurrentSession(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	userID := "user-123"

	tests := []struct {
		name               string
		setupContext       func() context.Context
		mockSetup          func(*db.MockUserSessions)
		expectSession      *models.UserSession
		expectErrorMessage string
	}{
		{
			name: "no session id in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectSession: nil,
		},
		{
			name: "session found and not expired",
			setupContext: func() context.Context {
				caller := &UserCaller{UserSessionID: &sessionID}
				return WithCaller(context.Background(), caller)
			},
			mockSetup: func(m *db.MockUserSessions) {
				session := &models.UserSession{
					UserID:     userID,
					Expiration: time.Now().Add(time.Hour),
					Metadata:   models.ResourceMetadata{ID: sessionID},
				}
				m.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
			},
			expectSession: &models.UserSession{
				UserID:     userID,
				Expiration: time.Now().Add(time.Hour),
				Metadata:   models.ResourceMetadata{ID: sessionID},
			},
		},
		{
			name: "session expired",
			setupContext: func() context.Context {
				caller := &UserCaller{UserSessionID: &sessionID}
				return WithCaller(context.Background(), caller)
			},
			mockSetup: func(m *db.MockUserSessions) {
				session := &models.UserSession{
					UserID:     userID,
					Expiration: time.Now().Add(-time.Hour),
					Metadata:   models.ResourceMetadata{ID: sessionID},
				}
				m.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
			},
			expectSession: nil,
		},
		{
			name: "session not found",
			setupContext: func() context.Context {
				caller := &UserCaller{UserSessionID: &sessionID}
				return WithCaller(context.Background(), caller)
			},
			mockSetup: func(m *db.MockUserSessions) {
				m.On("GetUserSessionByID", mock.Anything, sessionID).Return(nil, nil)
			},
			expectSession: nil,
		},
		{
			name: "database error",
			setupContext: func() context.Context {
				caller := &UserCaller{UserSessionID: &sessionID}
				return WithCaller(context.Background(), caller)
			},
			mockSetup: func(m *db.MockUserSessions) {
				m.On("GetUserSessionByID", mock.Anything, sessionID).Return(nil, errors.New("db error"))
			},
			expectErrorMessage: "failed to get user session by id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
			}

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions)
			}

			manager := &userSessionManager{
				dbClient: mockDBClient,
			}

			ctx := test.setupContext()
			session, err := manager.GetCurrentSession(ctx)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
			} else {
				require.NoError(t, err)
				if test.expectSession != nil {
					require.NotNil(t, session)
					assert.Equal(t, test.expectSession.UserID, session.UserID)
					assert.Equal(t, test.expectSession.Metadata.ID, session.Metadata.ID)
				} else {
					assert.Nil(t, session)
				}
			}
		})
	}
}
func TestUserSessionManager_CreateSession(t *testing.T) {
	userID := "user-123"
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	token := "valid-token"
	username := "testuser"
	password := "testpass"
	email := "testuser@example.com"

	tests := []struct {
		name               string
		input              *CreateSessionInput
		setupContext       func() context.Context
		mockSetup          func(*db.MockUserSessions, *db.MockUsers, *db.MockTransactions, *MockAuthenticator, *MockSigningKeyManager)
		expectErrorMessage string
	}{
		{
			name: "successful creation with token",
			input: &CreateSessionInput{
				Token:     &token,
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions, _ *db.MockUsers, mockTx *db.MockTransactions, mockAuth *MockAuthenticator, mockSigning *MockSigningKeyManager) {
				user := &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Email:    email,
				}
				caller := &UserCaller{User: user}
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(caller, nil)

				mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTx.On("CommitTx", mock.Anything).Return(nil)
				mockTx.On("RollbackTx", mock.Anything).Return(nil)

				session := &models.UserSession{
					UserID:    userID,
					UserAgent: "test-agent",
					Metadata:  models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("CreateUserSession", mock.Anything, mock.AnythingOfType("*models.UserSession")).Return(session, nil)
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
					PageInfo:     &pagination.PageInfo{TotalCount: 0},
				}, nil)

				mockSigning.On("GenerateToken", mock.Anything, mock.AnythingOfType("*auth.TokenInput")).Return([]byte("access-token"), nil).Times(3)
			},
		},
		{
			name: "successful creation with username and password",
			input: &CreateSessionInput{
				Username:  &username,
				Password:  &password,
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockUsers *db.MockUsers, mockTx *db.MockTransactions, _ *MockAuthenticator, mockSigning *MockSigningKeyManager) {
				user := &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Email:    email,
				}
				err := user.SetPassword(password)
				require.NoError(t, err)
				mockUsers.On("GetUserByTRN", mock.Anything, types.UserModelType.BuildTRN(username)).Return(user, nil)

				mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTx.On("CommitTx", mock.Anything).Return(nil)
				mockTx.On("RollbackTx", mock.Anything).Return(nil)

				session := &models.UserSession{
					UserID:    userID,
					UserAgent: "test-agent",
					Metadata:  models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("CreateUserSession", mock.Anything, mock.AnythingOfType("*models.UserSession")).Return(session, nil)
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
					PageInfo:     &pagination.PageInfo{TotalCount: 0},
				}, nil)

				mockSigning.On("GenerateToken", mock.Anything, mock.AnythingOfType("*auth.TokenInput")).Return([]byte("access-token"), nil).Times(3)
			},
		},
		{
			name: "invalid token",
			input: &CreateSessionInput{
				Token:     &token,
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			mockSetup: func(_ *db.MockUserSessions, _ *db.MockUsers, _ *db.MockTransactions, mockAuth *MockAuthenticator, _ *MockSigningKeyManager) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(nil, errors.New("invalid token"))
			},
			expectErrorMessage: "oidc token is invalid",
		},
		{
			name: "session already exists",
			input: &CreateSessionInput{
				Token:     &token,
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				user := &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
				}
				caller := &UserCaller{
					User:          user,
					UserSessionID: &sessionID,
				}
				return WithCaller(context.Background(), caller)
			},
			mockSetup: func(_ *db.MockUserSessions, _ *db.MockUsers, _ *db.MockTransactions, mockAuth *MockAuthenticator, _ *MockSigningKeyManager) {
				user := &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
				}
				caller := &UserCaller{User: user}
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(caller, nil)
			},
			expectErrorMessage: "an active session already exists for this user",
		},
		{
			name: "missing required fields",
			input: &CreateSessionInput{
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectErrorMessage: "either token or username and password must be provided",
		},
		{
			name: "user credential login disabled",
			input: &CreateSessionInput{
				Username:  &username,
				Password:  &password,
				UserAgent: "test-agent",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectErrorMessage: "user credential login is disabled",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockUsers := db.NewMockUsers(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
				Users:        mockUsers,
				Transactions: mockTransactions,
			}
			mockAuthenticator := &MockAuthenticator{}
			mockSigningKeyManager := &MockSigningKeyManager{}
			logger, _ := logger.NewForTest()

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions, mockUsers, mockTransactions, mockAuthenticator, mockSigningKeyManager)
			}

			manager := &userSessionManager{
				dbClient:                      mockDBClient,
				signingKeyManager:             mockSigningKeyManager,
				authenticator:                 mockAuthenticator,
				logger:                        logger,
				accessTokenExpirationMinutes:  15 * time.Minute,
				refreshTokenExpirationMinutes: 60 * time.Minute,
				maxSessionsPerUser:            5,
				userCredentialLoginEnabled:    test.name != "user credential login disabled",
			}

			ctx := test.setupContext()
			response, err := manager.CreateSession(ctx, test.input)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.NotEmpty(t, response.CSRFToken)
				assert.NotNil(t, response.Session)
			}
		})
	}
}
func TestUserSessionManager_RefreshSession(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	userID := "user-123"
	refreshTokenID := "refresh-123"
	refreshToken := "valid-refresh-token"

	tests := []struct {
		name               string
		refreshToken       string
		mockSetup          func(*db.MockUserSessions, *MockSigningKeyManager)
		expectErrorMessage string
	}{
		{
			name:         "successful refresh",
			refreshToken: refreshToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
					Token: &mockJWT{jwtID: refreshTokenID},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)

				session := &models.UserSession{
					UserID:         userID,
					RefreshTokenID: refreshTokenID,
					Expiration:     time.Now().Add(time.Hour),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{*session},
				}, nil)

				updatedSession := *session
				updatedSession.RefreshTokenID = "new-refresh-123"
				mockUserSessions.On("UpdateUserSession", mock.Anything, mock.AnythingOfType("*models.UserSession")).Return(&updatedSession, nil)

				mockSigning.On("GenerateToken", mock.Anything, mock.AnythingOfType("*auth.TokenInput")).Return([]byte("new-token"), nil).Times(2)
			},
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid-token",
			mockSetup: func(_ *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				mockSigning.On("VerifyToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid token"))
			},
			expectErrorMessage: "refresh token is invalid",
		},
		{
			name:         "wrong token type",
			refreshToken: refreshToken,
			mockSetup: func(_ *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         "wrong-type",
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
					Token: &mockJWT{jwtID: refreshTokenID},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "token is not a refresh token",
		},
		{
			name:         "no session found",
			refreshToken: refreshToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
					Token: &mockJWT{jwtID: refreshTokenID},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)

				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
				}, nil)
			},
			expectErrorMessage: "no user session found for refresh token",
		},
		{
			name:         "expired session",
			refreshToken: refreshToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
					Token: &mockJWT{jwtID: refreshTokenID},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)

				session := &models.UserSession{
					UserID:         userID,
					RefreshTokenID: refreshTokenID,
					Expiration:     time.Now().Add(-time.Hour),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{*session},
				}, nil)
			},
			expectErrorMessage: "no valid user session found for refresh token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
			}
			mockSigningKeyManager := &MockSigningKeyManager{}

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions, mockSigningKeyManager)
			}

			manager := &userSessionManager{
				dbClient:          mockDBClient,
				signingKeyManager: mockSigningKeyManager,
			}

			response, err := manager.RefreshSession(context.Background(), test.refreshToken)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.NotNil(t, response.Session)
			}
		})
	}
}
func TestUserSessionManager_InvalidateSession(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	userID := "user-123"
	accessToken := "valid-access-token"
	refreshToken := "valid-refresh-token"

	tests := []struct {
		name               string
		accessToken        string
		refreshToken       string
		mockSetup          func(*db.MockUserSessions, *MockSigningKeyManager)
		expectErrorMessage string
	}{
		{
			name:         "successful invalidation with refresh token",
			refreshToken: refreshToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)

				session := &models.UserSession{
					UserID:   userID,
					Metadata: models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
				mockUserSessions.On("DeleteUserSession", mock.Anything, session).Return(nil)
			},
		},
		{
			name:        "successful invalidation with access token",
			accessToken: accessToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         UserSessionAccessTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, accessToken).Return(verifyOutput, nil)

				session := &models.UserSession{
					UserID:   userID,
					Metadata: models.ResourceMetadata{ID: sessionID},
				}
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
				mockUserSessions.On("DeleteUserSession", mock.Anything, session).Return(nil)
			},
		},
		{
			name: "no tokens provided",
		},
		{
			name:         "session not found",
			refreshToken: refreshToken,
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(nil, nil)
			},
		},
		{
			name:         "invalid refresh token type",
			refreshToken: refreshToken,
			mockSetup: func(_ *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         "wrong-type",
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, refreshToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "token is not a refresh token",
		},
		{
			name:        "invalid access token type",
			accessToken: accessToken,
			mockSetup: func(_ *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         "wrong-type",
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, accessToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "token is not an access token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
			}
			mockSigningKeyManager := &MockSigningKeyManager{}

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions, mockSigningKeyManager)
			}

			manager := &userSessionManager{
				dbClient:          mockDBClient,
				signingKeyManager: mockSigningKeyManager,
			}

			err := manager.InvalidateSession(context.Background(), test.accessToken, test.refreshToken)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
func TestUserSessionManager_VerifyCSRFToken(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	csrfToken := "valid-csrf-token"

	tests := []struct {
		name               string
		requestSessionID   string
		csrfToken          string
		mockSetup          func(*MockSigningKeyManager)
		expectErrorMessage string
	}{
		{
			name:             "successful verification",
			requestSessionID: sessionID,
			csrfToken:        csrfToken,
			mockSetup: func(mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         UserSessionCSRFTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, csrfToken).Return(verifyOutput, nil)
			},
		},
		{
			name:             "invalid csrf token",
			requestSessionID: sessionID,
			csrfToken:        "invalid-token",
			mockSetup: func(mockSigning *MockSigningKeyManager) {
				mockSigning.On("VerifyToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid token"))
			},
			expectErrorMessage: "csrf token is invalid",
		},
		{
			name:             "wrong token type",
			requestSessionID: sessionID,
			csrfToken:        csrfToken,
			mockSetup: func(mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         "wrong-type",
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, csrfToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "csrf token has an invalid type",
		},
		{
			name:             "missing session id claim",
			requestSessionID: sessionID,
			csrfToken:        csrfToken,
			mockSetup: func(mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type": UserSessionCSRFTokenType,
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, csrfToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "csrf token is missing session id claim",
		},
		{
			name:             "session id mismatch",
			requestSessionID: sessionID,
			csrfToken:        csrfToken,
			mockSetup: func(mockSigning *MockSigningKeyManager) {
				verifyOutput := &VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         UserSessionCSRFTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, "different-session"),
					},
				}
				mockSigning.On("VerifyToken", mock.Anything, csrfToken).Return(verifyOutput, nil)
			},
			expectErrorMessage: "csrf token session id does not match current session",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSigningKeyManager := &MockSigningKeyManager{}

			if test.mockSetup != nil {
				test.mockSetup(mockSigningKeyManager)
			}

			manager := &userSessionManager{
				signingKeyManager: mockSigningKeyManager,
			}

			err := manager.VerifyCSRFToken(context.Background(), test.requestSessionID, test.csrfToken)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
func TestUserSessionManager_ExchangeOAuthCodeForSessionToken(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	userID := "user-123"
	oauthCode := "valid-oauth-code"
	codeVerifier := "valid-code-verifier"
	redirectURI := "http://localhost:8080"

	tests := []struct {
		name               string
		input              *ExchangeOAuthCodeForSessionTokenInput
		mockSetup          func(*db.MockUserSessions, *MockSigningKeyManager)
		expectErrorMessage string
	}{
		{
			name: "successful exchange",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCode:         oauthCode,
				OAuthCodeVerifier: codeVerifier,
				RedirectURI:       redirectURI,
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions, mockSigning *MockSigningKeyManager) {
				// Create sha256 hash of code verifier for comparison
				hash := sha256.Sum256([]byte(codeVerifier))
				hashedVerifier := base64.RawURLEncoding.EncodeToString(hash[:])
				session := &models.UserSession{
					UserID:                   userID,
					Expiration:               time.Now().Add(time.Hour),
					OAuthCode:                &oauthCode,
					OAuthCodeChallenge:       &hashedVerifier,
					OAuthCodeChallengeMethod: ptr.String("S256"),
					OAuthCodeExpiration:      ptr.Time(time.Now().Add(time.Minute)),
					OAuthRedirectURI:         &redirectURI,
					Metadata:                 models.ResourceMetadata{ID: sessionID},
				}

				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{*session},
				}, nil)

				updatedSession := *session
				updatedSession.OAuthCode = nil
				updatedSession.OAuthCodeChallenge = nil
				updatedSession.OAuthCodeChallengeMethod = nil
				updatedSession.OAuthCodeExpiration = nil
				updatedSession.OAuthRedirectURI = nil
				mockUserSessions.On("UpdateUserSession", mock.Anything, mock.AnythingOfType("*models.UserSession")).Return(&updatedSession, nil)

				mockSigning.On("GenerateToken", mock.Anything, mock.AnythingOfType("*auth.TokenInput")).Return([]byte("access-token"), nil)
			},
		},
		{
			name: "missing oauth code",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCodeVerifier: codeVerifier,
				RedirectURI:       redirectURI,
			},
			expectErrorMessage: "missing required fields in input",
		},
		{
			name: "missing code verifier",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCode:   oauthCode,
				RedirectURI: redirectURI,
			},
			expectErrorMessage: "missing required fields in input",
		},
		{
			name: "missing redirect uri",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCode:         oauthCode,
				OAuthCodeVerifier: codeVerifier,
			},
			expectErrorMessage: "missing required fields in input",
		},
		{
			name: "no session found for auth code",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCode:         oauthCode,
				OAuthCodeVerifier: codeVerifier,
				RedirectURI:       redirectURI,
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions, _ *MockSigningKeyManager) {
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
				}, nil)
			},
			expectErrorMessage: "no user session found for auth code",
		},
		{
			name: "invalid redirect uri",
			input: &ExchangeOAuthCodeForSessionTokenInput{
				OAuthCode:         oauthCode,
				OAuthCodeVerifier: codeVerifier,
				RedirectURI:       "http://different.com",
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions, _ *MockSigningKeyManager) {
				session := &models.UserSession{
					UserID:           userID,
					Expiration:       time.Now().Add(time.Hour),
					OAuthCode:        &oauthCode,
					OAuthRedirectURI: &redirectURI,
					Metadata:         models.ResourceMetadata{ID: sessionID},
				}

				mockUserSessions.On("GetUserSessions", mock.Anything, mock.AnythingOfType("*db.GetUserSessionsInput")).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{*session},
				}, nil)
			},
			expectErrorMessage: "invalid redirect uri",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
				Transactions: mockTransactions,
			}
			mockSigningKeyManager := &MockSigningKeyManager{}

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions, mockSigningKeyManager)
			}

			manager := &userSessionManager{
				dbClient:                     mockDBClient,
				signingKeyManager:            mockSigningKeyManager,
				accessTokenExpirationMinutes: 15 * time.Minute,
			}

			response, err := manager.ExchangeOAuthCodeForSessionToken(context.Background(), test.input)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.Equal(t, int((15 * time.Minute).Seconds()), response.ExpiresIn)
			}
		})
	}
}
func TestUserSessionManager_InitiateSessionOauthCodeFlow(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	userID := "user-123"

	tests := []struct {
		name               string
		input              *InitiateSessionOauthCodeFlowInput
		mockSetup          func(*db.MockUserSessions)
		expectErrorMessage string
	}{
		{
			name: "successful initiation",
			input: &InitiateSessionOauthCodeFlowInput{
				CodeChallenge:       "test-challenge",
				CodeChallengeMethod: "S256",
				RedirectURI:         "http://localhost:8080",
				UserSessionID:       sessionID,
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions) {
				session := &models.UserSession{
					UserID:     userID,
					Expiration: time.Now().Add(time.Hour),
					Metadata:   models.ResourceMetadata{ID: sessionID},
				}

				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
				mockUserSessions.On("UpdateUserSession", mock.Anything, mock.AnythingOfType("*models.UserSession")).Return(session, nil)
			},
		},
		{
			name: "invalid redirect uri - not localhost",
			input: &InitiateSessionOauthCodeFlowInput{
				CodeChallenge:       "test-challenge",
				CodeChallengeMethod: "S256",
				RedirectURI:         "http://example.com",
				UserSessionID:       sessionID,
			},
			expectErrorMessage: "invalid redirect uri",
		},
		{
			name: "invalid code challenge method",
			input: &InitiateSessionOauthCodeFlowInput{
				CodeChallenge:       "test-challenge",
				CodeChallengeMethod: "plain",
				RedirectURI:         "http://localhost:8080",
				UserSessionID:       sessionID,
			},
			expectErrorMessage: "invalid code challenge method",
		},
		{
			name: "session not found",
			input: &InitiateSessionOauthCodeFlowInput{
				CodeChallenge:       "test-challenge",
				CodeChallengeMethod: "S256",
				RedirectURI:         "http://localhost:8080",
				UserSessionID:       sessionID,
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions) {
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(nil, nil)
			},
			expectErrorMessage: "user session not found or has expired",
		},
		{
			name: "expired session",
			input: &InitiateSessionOauthCodeFlowInput{
				CodeChallenge:       "test-challenge",
				CodeChallengeMethod: "S256",
				RedirectURI:         "http://localhost:8080",
				UserSessionID:       sessionID,
			},
			mockSetup: func(mockUserSessions *db.MockUserSessions) {
				session := &models.UserSession{
					UserID:     userID,
					Expiration: time.Now().Add(-time.Hour),
					Metadata:   models.ResourceMetadata{ID: sessionID},
				}

				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
			},
			expectErrorMessage: "user session not found or has expired",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockUserSessions := db.NewMockUserSessions(t)
			mockDBClient := &db.Client{
				UserSessions: mockUserSessions,
			}

			if test.mockSetup != nil {
				test.mockSetup(mockUserSessions)
			}

			manager := &userSessionManager{
				dbClient: mockDBClient,
			}

			authCode, err := manager.InitiateSessionOauthCodeFlow(context.Background(), test.input)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Empty(t, authCode)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, authCode)
			}
		})
	}
}
func TestUserSessionManager_SetUserSessionCookies(t *testing.T) {
	tests := []struct {
		name                string
		enableSecureCookies bool
		input               *SetUserSessionCookiesInput
		expectCookieCount   int
	}{
		{
			name:                "set cookies with csrf token and secure cookies enabled",
			enableSecureCookies: true,
			input: &SetUserSessionCookiesInput{
				AccessToken:       "access-token",
				RefreshToken:      "refresh-token",
				CsrfToken:         ptr.String("csrf-token"),
				SessionExpiration: time.Now().Add(time.Hour),
			},
			expectCookieCount: 3,
		},
		{
			name:                "set cookies without csrf token",
			enableSecureCookies: false,
			input: &SetUserSessionCookiesInput{
				AccessToken:       "access-token",
				RefreshToken:      "refresh-token",
				SessionExpiration: time.Now().Add(time.Hour),
			},
			expectCookieCount: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &userSessionManager{
				enableSecureCookies:           test.enableSecureCookies,
				tharsisUIDomain:               "ui.example.com",
				accessTokenExpirationMinutes:  15 * time.Minute,
				refreshTokenExpirationMinutes: 60 * time.Minute,
			}

			recorder := httptest.NewRecorder()
			manager.SetUserSessionCookies(recorder, test.input)

			cookies := recorder.Result().Cookies()
			assert.Len(t, cookies, test.expectCookieCount)

			// Verify access token cookie
			var accessCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == manager.GetUserSessionAccessTokenCookieName() {
					accessCookie = cookie
					break
				}
			}
			require.NotNil(t, accessCookie)
			assert.Equal(t, test.input.AccessToken, accessCookie.Value)
			assert.True(t, accessCookie.HttpOnly)
			assert.Equal(t, test.enableSecureCookies, accessCookie.Secure)
		})
	}
}

func TestUserSessionManager_ClearUserSessionCookies(t *testing.T) {
	tests := []struct {
		name                string
		enableSecureCookies bool
	}{
		{
			name:                "clear cookies with secure cookies enabled",
			enableSecureCookies: true,
		},
		{
			name:                "clear cookies with secure cookies disabled",
			enableSecureCookies: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &userSessionManager{
				enableSecureCookies: test.enableSecureCookies,
				tharsisUIDomain:     "ui.example.com",
			}

			recorder := httptest.NewRecorder()
			manager.ClearUserSessionCookies(recorder)

			cookies := recorder.Result().Cookies()
			assert.Len(t, cookies, 3)

			for _, cookie := range cookies {
				assert.Equal(t, emptyCookieValue, cookie.Value)
				assert.True(t, cookie.Expires.Before(time.Now()))
			}
		})
	}
}

func TestUserSessionManager_GetUserSessionAccessTokenCookieName(t *testing.T) {
	tests := []struct {
		name                string
		enableSecureCookies bool
		expectedName        string
	}{
		{
			name:                "secure cookies enabled",
			enableSecureCookies: true,
			expectedName:        cookieHostPrefix + accessTokenCookieName,
		},
		{
			name:                "secure cookies disabled",
			enableSecureCookies: false,
			expectedName:        accessTokenCookieName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &userSessionManager{
				enableSecureCookies: test.enableSecureCookies,
			}

			name := manager.GetUserSessionAccessTokenCookieName()
			assert.Equal(t, test.expectedName, name)
		})
	}
}

func TestUserSessionManager_GetUserSessionRefreshTokenCookieName(t *testing.T) {
	tests := []struct {
		name                string
		enableSecureCookies bool
		expectedName        string
	}{
		{
			name:                "secure cookies enabled",
			enableSecureCookies: true,
			expectedName:        cookieHostPrefix + refreshTokenCookieName,
		},
		{
			name:                "secure cookies disabled",
			enableSecureCookies: false,
			expectedName:        refreshTokenCookieName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &userSessionManager{
				enableSecureCookies: test.enableSecureCookies,
			}

			name := manager.GetUserSessionRefreshTokenCookieName()
			assert.Equal(t, test.expectedName, name)
		})
	}
}

func TestUserSessionManager_GetUserSessionCSRFTokenCookieName(t *testing.T) {
	manager := &userSessionManager{}
	name := manager.GetUserSessionCSRFTokenCookieName()
	assert.Equal(t, csrfTokenCookieName, name)
}

func TestGetRequestUserSessionID(t *testing.T) {
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"

	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectedID    string
		expectedFound bool
	}{
		{
			name: "session id found",
			setupContext: func() context.Context {
				caller := &UserCaller{UserSessionID: &sessionID}
				return WithCaller(context.Background(), caller)
			},
			expectedID:    sessionID,
			expectedFound: true,
		},
		{
			name: "no caller in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedFound: false,
		},
		{
			name: "caller without session id",
			setupContext: func() context.Context {
				caller := &UserCaller{}
				return WithCaller(context.Background(), caller)
			},
			expectedFound: false,
		},
		{
			name: "non-user caller",
			setupContext: func() context.Context {
				caller := &ServiceAccountCaller{}
				return WithCaller(context.Background(), caller)
			},
			expectedFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.setupContext()
			id, found := GetRequestUserSessionID(ctx)

			assert.Equal(t, test.expectedFound, found)
			if test.expectedFound {
				assert.Equal(t, test.expectedID, id)
			} else {
				assert.Empty(t, id)
			}
		})
	}
}

type mockJWT struct {
	jwtID string
}

func (m *mockJWT) JwtID() string                                         { return m.jwtID }
func (m *mockJWT) Subject() string                                       { return "" }
func (m *mockJWT) Audience() []string                                    { return nil }
func (m *mockJWT) Expiration() time.Time                                 { return time.Time{} }
func (m *mockJWT) IssuedAt() time.Time                                   { return time.Time{} }
func (m *mockJWT) Issuer() string                                        { return "" }
func (m *mockJWT) NotBefore() time.Time                                  { return time.Time{} }
func (m *mockJWT) PrivateClaims() map[string]interface{}                 { return nil }
func (m *mockJWT) Get(string) (interface{}, bool)                        { return nil, false }
func (m *mockJWT) Set(string, interface{}) error                         { return nil }
func (m *mockJWT) Remove(string) error                                   { return nil }
func (m *mockJWT) Options() *jwt.TokenOptionSet                          { return nil }
func (m *mockJWT) Clone() (jwt.Token, error)                             { return nil, nil }
func (m *mockJWT) Iterate(context.Context) jwt.Iterator                  { return nil }
func (m *mockJWT) Walk(context.Context, jwt.Visitor) error               { return nil }
func (m *mockJWT) AsMap(context.Context) (map[string]interface{}, error) { return nil, nil }
