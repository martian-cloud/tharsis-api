package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestUserSessionManager_CreateSession(t *testing.T) {
	userID := uuid.NewString()
	sessionID := uuid.NewString()
	refreshTokenID := uuid.NewString()
	userAgent := "test-user-agent"
	token := "test-token"
	maxUserSessions := 5

	tests := []struct {
		name               string
		setupMocks         func(*MockAuthenticator, *MockIdentityProvider, *db.Client)
		expectErrorMessage string
	}{
		{
			name: "successful session creation",
			setupMocks: func(mockAuth *MockAuthenticator, mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(&UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{ID: userID},
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockTransactions := db.NewMockTransactions(t)

				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockUserSessions.On("CreateUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.UserID == userID && session.UserAgent == userAgent
				})).Return(&models.UserSession{
					UserID:         userID,
					RefreshTokenID: refreshTokenID,
					UserAgent:      userAgent,
					Expiration:     time.Now().Add(time.Hour).UTC(),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}, nil)

				mockUserSessions.On("GetUserSessions", mock.Anything, mock.MatchedBy(func(input *db.GetUserSessionsInput) bool {
					return input.Filter != nil && input.Filter.UserID != nil && *input.Filter.UserID == userID
				})).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
					PageInfo:     &pagination.PageInfo{TotalCount: 0},
				}, nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionAccessTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("access-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == refreshTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID) &&
						input.JwtID == refreshTokenID
				})).Return([]byte("refresh-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionCSRFTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("csrf-token"), nil)

				mockDBClient.UserSessions = mockUserSessions
				mockDBClient.Transactions = mockTransactions
			},
		},
		{
			name: "authentication fails",
			setupMocks: func(mockAuth *MockAuthenticator, _ *MockIdentityProvider, _ *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(nil, errors.New("authentication failed"))
			},
			expectErrorMessage: "oidc token is invalid",
		},
		{
			name: "invalid caller type",
			setupMocks: func(mockAuth *MockAuthenticator, _ *MockIdentityProvider, _ *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(&ServiceAccountCaller{}, nil)
			},
			expectErrorMessage: "invalid caller type",
		},
		{
			name: "removes oldest session when limit exceeded",
			setupMocks: func(mockAuth *MockAuthenticator, mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(&UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{ID: userID},
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockTransactions := db.NewMockTransactions(t)

				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockUserSessions.On("CreateUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.UserID == userID && session.UserAgent == userAgent
				})).Return(&models.UserSession{
					UserID:         userID,
					RefreshTokenID: refreshTokenID,
					UserAgent:      userAgent,
					Expiration:     time.Now().Add(time.Hour).UTC(),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}, nil)

				// Mock the cleanup call - simulate having 6 sessions (exceeding limit of 5)
				oldestSessionID := uuid.NewString()
				sortBy := db.UserSessionSortableFieldExpirationAsc
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.MatchedBy(func(input *db.GetUserSessionsInput) bool {
					return input.Sort != nil && *input.Sort == sortBy &&
						input.Filter != nil && input.Filter.UserID != nil && *input.Filter.UserID == userID &&
						input.PaginationOptions != nil && input.PaginationOptions.First != nil && *input.PaginationOptions.First == 1
				})).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{
						{
							UserID:     userID,
							Expiration: time.Now().Add(time.Hour).UTC(),
							Metadata:   models.ResourceMetadata{ID: oldestSessionID},
						},
					},
					PageInfo: &pagination.PageInfo{TotalCount: int32(maxUserSessions) + 1},
				}, nil)

				// Mock deletion of oldest session
				mockUserSessions.On("DeleteUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == oldestSessionID
				})).Return(nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionAccessTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("access-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == refreshTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID) &&
						input.JwtID == refreshTokenID
				})).Return([]byte("refresh-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionCSRFTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("csrf-token"), nil)

				mockDBClient.UserSessions = mockUserSessions
				mockDBClient.Transactions = mockTransactions
			},
		},
		{
			name: "removes expired session during cleanup",
			setupMocks: func(mockAuth *MockAuthenticator, mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(&UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{ID: userID},
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockTransactions := db.NewMockTransactions(t)

				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockUserSessions.On("CreateUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.UserID == userID && session.UserAgent == userAgent
				})).Return(&models.UserSession{
					UserID:         userID,
					RefreshTokenID: refreshTokenID,
					UserAgent:      userAgent,
					Expiration:     time.Now().Add(time.Hour).UTC(),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}, nil)

				// Mock the cleanup call - simulate having an expired session
				expiredSessionID := uuid.NewString()
				sortBy := db.UserSessionSortableFieldExpirationAsc
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.MatchedBy(func(input *db.GetUserSessionsInput) bool {
					return input.Sort != nil && *input.Sort == sortBy &&
						input.Filter != nil && input.Filter.UserID != nil && *input.Filter.UserID == userID &&
						input.PaginationOptions != nil && input.PaginationOptions.First != nil && *input.PaginationOptions.First == 1
				})).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{
						{
							UserID:     userID,
							Expiration: time.Now().Add(-time.Hour).UTC(), // Expired session
							Metadata:   models.ResourceMetadata{ID: expiredSessionID},
						},
					},
					PageInfo: &pagination.PageInfo{TotalCount: int32(maxUserSessions - 2)}, // Under limit but has expired session
				}, nil)

				// Mock deletion of expired session
				mockUserSessions.On("DeleteUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == expiredSessionID
				})).Return(nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionAccessTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("access-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == refreshTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID) &&
						input.JwtID == refreshTokenID
				})).Return([]byte("refresh-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionCSRFTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("csrf-token"), nil)

				mockDBClient.UserSessions = mockUserSessions
				mockDBClient.Transactions = mockTransactions
			},
		},
		{
			name: "database transaction fails",
			setupMocks: func(mockAuth *MockAuthenticator, _ *MockIdentityProvider, mockDBClient *db.Client) {
				mockAuth.On("Authenticate", mock.Anything, token, false).Return(&UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{ID: userID},
					},
				}, nil)

				mockTransactions := db.NewMockTransactions(t)
				mockTransactions.On("BeginTx", mock.Anything).Return(nil, errors.New("transaction failed"))

				mockDBClient.Transactions = mockTransactions
			},
			expectErrorMessage: "failed to begin transaction",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockAuth := NewMockAuthenticator(t)
			mockIDP := NewMockIdentityProvider(t)
			logger, _ := logger.NewForTest()

			mockDBClient := &db.Client{}

			if test.setupMocks != nil {
				test.setupMocks(mockAuth, mockIDP, mockDBClient)
			}

			manager := NewUserSessionManager(
				mockDBClient,
				mockIDP,
				mockAuth,
				logger,
				60,   // access token expiration
				1440, // refresh token expiration
				maxUserSessions,
			)

			result, err := manager.CreateSession(context.Background(), token, userAgent)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.NotEmpty(t, result.CSRFToken)
				assert.NotNil(t, result.SessionExpiration)
			}
		})
	}
}

func TestUserSessionManager_RefreshSession(t *testing.T) {
	userID := uuid.NewString()
	sessionID := uuid.NewString()
	refreshTokenID := uuid.NewString()
	newRefreshTokenID := uuid.NewString()
	refreshToken := "refresh-token"

	tests := []struct {
		name               string
		setupMocks         func(*MockIdentityProvider, *db.Client)
		expectErrorMessage string
	}{
		{
			name: "successful session refresh",
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, refreshTokenID)
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)

				mockUserSessions.On("GetUserSessions", mock.Anything, mock.MatchedBy(func(input *db.GetUserSessionsInput) bool {
					return input.Filter != nil &&
						len(input.Filter.UserSessionIDs) == 1 &&
						input.Filter.UserSessionIDs[0] == sessionID &&
						input.Filter.RefreshTokenID != nil &&
						*input.Filter.RefreshTokenID == refreshTokenID
				})).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{
						{
							UserID:         userID,
							RefreshTokenID: refreshTokenID,
							Expiration:     time.Now().Add(time.Hour).UTC(),
							Metadata:       models.ResourceMetadata{ID: sessionID},
						},
					},
				}, nil)

				mockUserSessions.On("UpdateUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == sessionID && session.RefreshTokenID != refreshTokenID
				})).Return(&models.UserSession{
					UserID:         userID,
					RefreshTokenID: newRefreshTokenID,
					Expiration:     time.Now().Add(time.Hour).UTC(),
					Metadata:       models.ResourceMetadata{ID: sessionID},
				}, nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == UserSessionAccessTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID)
				})).Return([]byte("new-access-token"), nil)

				mockIDP.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *TokenInput) bool {
					return input.Subject == gid.ToGlobalID(types.UserModelType, userID) &&
						input.Claims["type"] == refreshTokenType &&
						input.Claims[SessionIDClaim] == gid.ToGlobalID(types.UserSessionModelType, sessionID) &&
						input.JwtID == newRefreshTokenID
				})).Return([]byte("new-refresh-token"), nil)

				mockDBClient.UserSessions = mockUserSessions
			},
		},
		{
			name: "token verification fails",
			setupMocks: func(mockIDP *MockIdentityProvider, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(nil, errors.New("token verification failed"))
			},
			expectErrorMessage: "refresh token is invalid",
		},
		{
			name: "token is not a refresh token",
			setupMocks: func(mockIDP *MockIdentityProvider, _ *db.Client) {
				token := jwt.New()
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type": "access_token",
					},
				}, nil)
			},
			expectErrorMessage: "token is not a refresh token",
		},
		{
			name: "no user session found",
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, refreshTokenID)
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.Anything).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{},
				}, nil)

				mockDBClient.UserSessions = mockUserSessions
			},
			expectErrorMessage: "no user session found for refresh token",
		},
		{
			name: "session is expired",
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, refreshTokenID)
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.Anything).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{
						{
							UserID:         userID,
							RefreshTokenID: refreshTokenID,
							Expiration:     time.Now().Add(-time.Hour).UTC(), // Expired
							Metadata:       models.ResourceMetadata{ID: sessionID},
						},
					},
				}, nil)

				mockDBClient.UserSessions = mockUserSessions
			},
			expectErrorMessage: "no valid user session found for refresh token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockIDP := NewMockIdentityProvider(t)
			logger, _ := logger.NewForTest()

			mockDBClient := &db.Client{}

			if test.setupMocks != nil {
				test.setupMocks(mockIDP, mockDBClient)
			}

			manager := NewUserSessionManager(
				mockDBClient,
				mockIDP,
				nil, // authenticator not needed for refresh
				logger,
				60,   // access token expiration
				1440, // refresh token expiration
				5,    // max sessions per user
			)

			result, err := manager.RefreshSession(context.Background(), refreshToken)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.NotNil(t, result.SessionExpiration)
			}
		})
	}
}
func TestUserSessionManager_InvalidateSession(t *testing.T) {
	sessionID := uuid.NewString()
	accessToken := "access-token"
	refreshToken := "refresh-token"

	tests := []struct {
		name               string
		accessToken        string
		refreshToken       string
		setupMocks         func(*MockIdentityProvider, *db.Client)
		expectErrorMessage string
	}{
		{
			name:         "successful invalidation with refresh token",
			refreshToken: refreshToken,
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(&models.UserSession{
					Metadata: models.ResourceMetadata{ID: sessionID},
				}, nil)
				mockUserSessions.On("DeleteUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == sessionID
				})).Return(nil)

				mockDBClient.UserSessions = mockUserSessions
			},
		},
		{
			name:        "successful invalidation with access token",
			accessToken: accessToken,
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				mockIDP.On("VerifyToken", mock.Anything, accessToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         UserSessionAccessTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(&models.UserSession{
					Metadata: models.ResourceMetadata{ID: sessionID},
				}, nil)
				mockUserSessions.On("DeleteUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == sessionID
				})).Return(nil)

				mockDBClient.UserSessions = mockUserSessions
			},
		},
		{
			name:         "expired refresh token is handled gracefully",
			refreshToken: refreshToken,
			setupMocks: func(mockIDP *MockIdentityProvider, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(nil, jwt.ErrTokenExpired())
			},
		},
		{
			name: "no tokens provided",
		},
		{
			name:         "session not found",
			refreshToken: refreshToken,
			setupMocks: func(mockIDP *MockIdentityProvider, mockDBClient *db.Client) {
				token := jwt.New()
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         refreshTokenType,
						SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
					},
				}, nil)

				mockUserSessions := db.NewMockUserSessions(t)
				mockUserSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(nil, nil)

				mockDBClient.UserSessions = mockUserSessions
			},
		},
		{
			name:         "refresh token verification fails",
			refreshToken: refreshToken,
			setupMocks: func(mockIDP *MockIdentityProvider, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, refreshToken).Return(nil, errors.New("verification failed"))
			},
			expectErrorMessage: "failed to verify refresh token",
		},
		{
			name:        "access token verification fails",
			accessToken: accessToken,
			setupMocks: func(mockIDP *MockIdentityProvider, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, accessToken).Return(nil, errors.New("verification failed"))
			},
			expectErrorMessage: "failed to verify access token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockIDP := NewMockIdentityProvider(t)
			logger, _ := logger.NewForTest()

			mockDBClient := &db.Client{}

			if test.setupMocks != nil {
				test.setupMocks(mockIDP, mockDBClient)
			}

			manager := NewUserSessionManager(
				mockDBClient,
				mockIDP,
				nil, // authenticator not needed for invalidation
				logger,
				60,   // access token expiration
				1440, // refresh token expiration
				5,    // max sessions per user
			)

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
func TestGetUserSessionAccessTokenCookieName(t *testing.T) {
	tests := []struct {
		name     string
		secure   bool
		expected string
	}{
		{
			name:     "secure cookie",
			secure:   true,
			expected: "__Host-tharsis_access_token",
		},
		{
			name:     "non-secure cookie",
			secure:   false,
			expected: "tharsis_access_token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetUserSessionAccessTokenCookieName(test.secure)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestGetUserSessionRefreshTokenCookieName(t *testing.T) {
	tests := []struct {
		name     string
		secure   bool
		expected string
	}{
		{
			name:     "secure cookie",
			secure:   true,
			expected: "__Host-tharsis_refresh_token",
		},
		{
			name:     "non-secure cookie",
			secure:   false,
			expected: "tharsis_refresh_token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetUserSessionRefreshTokenCookieName(test.secure)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestUserSessionManager_VerifyCSRFToken(t *testing.T) {
	requestSessionID := uuid.NewString()
	sessionGID := gid.ToGlobalID(types.UserSessionModelType, requestSessionID)
	csrfToken := "csrf-token"

	tests := []struct {
		name               string
		requestSessionID   string
		csrfToken          string
		setupMocks         func(*MockIdentityProvider)
		expectErrorMessage string
	}{
		{
			name:             "successful csrf token verification",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         UserSessionCSRFTokenType,
						SessionIDClaim: sessionGID,
					},
				}, nil)
			},
		},
		{
			name:             "token verification fails",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(nil, errors.New("token verification failed"))
			},
			expectErrorMessage: "csrf token is invalid",
		},
		{
			name:             "token type is missing",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						SessionIDClaim: sessionGID,
					},
				}, nil)
			},
			expectErrorMessage: "csrf token has an invalid type",
		},
		{
			name:             "token type is invalid",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         "invalid_type",
						SessionIDClaim: sessionGID,
					},
				}, nil)
			},
			expectErrorMessage: "csrf token has an invalid type",
		},
		{
			name:             "session id claim is missing",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type": UserSessionCSRFTokenType,
					},
				}, nil)
			},
			expectErrorMessage: "csrf token is missing session id claim",
		},
		{
			name:             "session id does not match",
			requestSessionID: requestSessionID,
			csrfToken:        csrfToken,
			setupMocks: func(mockIDP *MockIdentityProvider) {
				differentSessionID := gid.ToGlobalID(types.UserSessionModelType, uuid.NewString())
				mockIDP.On("VerifyToken", mock.Anything, csrfToken).Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":         UserSessionCSRFTokenType,
						SessionIDClaim: differentSessionID,
					},
				}, nil)
			},
			expectErrorMessage: "csrf token session id does not match current session",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockIDP := NewMockIdentityProvider(t)
			logger, _ := logger.NewForTest()

			if test.setupMocks != nil {
				test.setupMocks(mockIDP)
			}

			manager := NewUserSessionManager(
				&db.Client{},
				mockIDP,
				nil, // authenticator not needed for CSRF verification
				logger,
				60,   // access token expiration
				1440, // refresh token expiration
				5,    // max sessions per user
			)

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

func TestNewUserSessionManager(t *testing.T) {
	t.Run("creates user session manager with correct configuration", func(t *testing.T) {
		mockDBClient := &db.Client{}
		mockIDP := NewMockIdentityProvider(t)
		mockAuth := NewMockAuthenticator(t)
		logger, _ := logger.NewForTest()

		manager := NewUserSessionManager(
			mockDBClient,
			mockIDP,
			mockAuth,
			logger,
			60,   // access token expiration
			1440, // refresh token expiration
			5,    // max sessions per user
		)

		assert.NotNil(t, manager)
	})
}
