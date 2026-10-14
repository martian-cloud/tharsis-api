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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestAuthenticator_Authenticate(t *testing.T) {
	// Test cases
	tests := []struct {
		name               string
		tokenString        string
		mockSetup          func(*mockTokenAuthenticator)
		expectCaller       Caller
		expectErrorMessage string
	}{
		{
			name:               "empty token string",
			tokenString:        "",
			expectErrorMessage: "authentication token is missing",
		},
		{
			name:               "invalid token format",
			tokenString:        "invalid-token",
			expectErrorMessage: "failed to decode token",
		},
		{
			name:        "successful authentication",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0LWlzc3VlciJ9.FVmNJJmSuLYbN5E-8qHFDRuBsJgbqbDF7zKhpUGdzA8",
			mockSetup: func(m *mockTokenAuthenticator) {
				m.On("Use", mock.Anything).Return(true)
				m.On("Authenticate", mock.Anything, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0LWlzc3VlciJ9.FVmNJJmSuLYbN5E-8qHFDRuBsJgbqbDF7zKhpUGdzA8", true).Return(&UserCaller{}, nil)
			},
			expectCaller: &UserCaller{},
		},
		{
			name:        "authenticator returns error",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0LWlzc3VlciJ9.FVmNJJmSuLYbN5E-8qHFDRuBsJgbqbDF7zKhpUGdzA8",
			mockSetup: func(m *mockTokenAuthenticator) {
				m.On("Use", mock.Anything).Return(true)
				m.On("Authenticate", mock.Anything, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0LWlzc3VlciJ9.FVmNJJmSuLYbN5E-8qHFDRuBsJgbqbDF7zKhpUGdzA8", true).Return(nil, errors.New("authentication failed"))
			},
			expectErrorMessage: "authentication failed",
		},
		{
			name:        "no authenticator found for token",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0LWlzc3VlciJ9.FVmNJJmSuLYbN5E-8qHFDRuBsJgbqbDF7zKhpUGdzA8",
			mockSetup: func(m *mockTokenAuthenticator) {
				m.On("Use", mock.Anything).Return(false)
			},
			expectErrorMessage: "token issuer test-issuer is not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockTokenAuth := newMockTokenAuthenticator(t)

			if test.mockSetup != nil {
				test.mockSetup(mockTokenAuth)
			}

			authenticator := newAuthenticator([]tokenAuthenticator{mockTokenAuth})

			caller, err := authenticator.Authenticate(context.Background(), test.tokenString, true)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, caller)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCaller, caller)
			}
		})
	}
}

func TestNewAuthenticator(t *testing.T) {
	t.Run("creates authenticator with correct token authenticators", func(t *testing.T) {
		mockUserAuth := &UserAuth{}
		mockFederatedRegistryAuth := &FederatedRegistryAuth{}
		mockIDP := NewMockSigningKeyManager(t)
		mockDBClient := &db.Client{}
		mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
		issuerURL := "https://test-issuer.com"

		authenticator := NewAuthenticator(
			mockUserAuth,
			mockFederatedRegistryAuth,
			mockIDP,
			mockDBClient,
			mockMaintenanceMonitor,
			nil, // inheritedSettingResolver - not needed for this test
			issuerURL,
		)

		require.NotNil(t, authenticator)
	})
}

func TestTharsisIDPTokenAuthenticator_Use(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		tokenIss  string
		expected  bool
	}{
		{
			name:      "token issuer matches",
			issuerURL: "https://test-issuer.com",
			tokenIss:  "https://test-issuer.com",
			expected:  true,
		},
		{
			name:      "token issuer does not match",
			issuerURL: "https://test-issuer.com",
			tokenIss:  "https://other-issuer.com",
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &tharsisIDPTokenAuthenticator{
				issuerURL: test.issuerURL,
			}

			token := jwt.New()
			err := token.Set(jwt.IssuerKey, test.tokenIss)
			require.NoError(t, err)

			result := authenticator.Use(token)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestTharsisIDPTokenAuthenticator_Authenticate(t *testing.T) {
	jobID := uuid.NewString()
	runID := uuid.NewString()
	linkID := uuid.NewString()
	workspaceID := uuid.NewString()

	activeJob := func(status models.JobStatus) *models.Job {
		job := &models.Job{
			Metadata:    models.ResourceMetadata{ID: jobID},
			Type:        models.JobPlanType,
			RunID:       runID,
			WorkspaceID: workspaceID,
		}
		require.NoError(t, job.SetStatus(models.JobPending))
		if status != models.JobPending {
			require.NoError(t, job.SetStatus(models.JobRunning))
		}
		if status != models.JobPending && status != models.JobRunning {
			require.NoError(t, job.SetStatus(status))
		}
		return job
	}

	// jobTokenMocks sets up a valid job token whose job lookup returns job.
	jobTokenMocks := func(job *models.Job) func(*MockSigningKeyManager, *db.Client) {
		return func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
			mockJobs := db.NewMockJobs(t)
			mockJobs.On("GetJobByID", mock.Anything, jobID).Return(job, nil)
			mockDBClient.Jobs = mockJobs

			mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
				Token: jwt.New(),
				PrivateClaims: map[string]string{
					"type":         JobTokenType,
					"job_id":       gid.ToGlobalID(types.JobModelType, jobID),
					"run_id":       gid.ToGlobalID(types.RunModelType, runID),
					"workspace_id": gid.ToGlobalID(types.WorkspaceModelType, workspaceID),
				},
			}, nil)
		}
	}

	// jobTokenCaller is the caller authenticating a jobTokenMocks token for job should produce: it
	// carries the job loaded during authentication (the token has no subject, so no TRN).
	jobTokenCaller := func(job *models.Job) *JobCaller {
		return &JobCaller{Job: job, RunID: runID, WorkspaceID: workspaceID}
	}

	userID := uuid.NewString()
	sessionID := uuid.NewString()

	// userSessionTokenMocks sets up a valid user session access token whose session and user lookups
	// return session and user.
	userSessionTokenMocks := func(session *models.UserSession, user *models.User) func(*MockSigningKeyManager, *db.Client) {
		return func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
			token := jwt.New()
			require.NoError(t, token.Set(jwt.SubjectKey, gid.ToGlobalID(types.UserModelType, userID)))

			mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
				Token: token,
				PrivateClaims: map[string]string{
					"type":         UserSessionAccessTokenType,
					SessionIDClaim: gid.ToGlobalID(types.UserSessionModelType, sessionID),
				},
			}, nil)

			mockSessions := db.NewMockUserSessions(t)
			mockSessions.On("GetUserSessionByID", mock.Anything, sessionID).Return(session, nil)
			mockDBClient.UserSessions = mockSessions

			if user != nil {
				mockUsers := db.NewMockUsers(t)
				mockUsers.On("GetUserByID", mock.Anything, userID).Return(user, nil)
				mockDBClient.Users = mockUsers
			}
		}
	}

	activeSession := &models.UserSession{
		Metadata:   models.ResourceMetadata{ID: sessionID},
		UserID:     userID,
		Expiration: time.Now().Add(time.Hour),
	}
	activeUser := &models.User{Metadata: models.ResourceMetadata{ID: userID}, Active: true}

	tests := []struct {
		name               string
		tokenString        string
		setupMocks         func(*MockSigningKeyManager, *db.Client)
		expectCaller       Caller
		expectErrorMessage string
	}{
		{
			name:        "token verification fails",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(nil, errors.New("exp not satisfied"))
			},
			expectErrorMessage: errExpired,
		},
		{
			name:        "token missing type claim",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{},
				}, nil)
			},
			expectErrorMessage: "failed to get token type",
		},
		{
			name:        "unsupported token type",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type": "unsupported",
					},
				}, nil)
			},
			expectErrorMessage: "unsupported token type received",
		},
		{
			name:         "user session token with an active session",
			tokenString:  "test-token",
			setupMocks:   userSessionTokenMocks(activeSession, activeUser),
			expectCaller: &UserCaller{},
		},
		{
			name:               "user session token whose session was deleted",
			tokenString:        "test-token",
			setupMocks:         userSessionTokenMocks(nil, nil),
			expectErrorMessage: "user session is no longer valid",
		},
		{
			name:        "user session token whose session has expired",
			tokenString: "test-token",
			setupMocks: userSessionTokenMocks(&models.UserSession{
				Metadata:   models.ResourceMetadata{ID: sessionID},
				UserID:     userID,
				Expiration: time.Now().Add(-time.Minute),
			}, nil),
			expectErrorMessage: "user session is no longer valid",
		},
		{
			name:        "user session token whose session belongs to another user",
			tokenString: "test-token",
			setupMocks: userSessionTokenMocks(&models.UserSession{
				Metadata:   models.ResourceMetadata{ID: sessionID},
				UserID:     uuid.NewString(),
				Expiration: time.Now().Add(time.Hour),
			}, nil),
			expectErrorMessage: "user session is no longer valid",
		},
		{
			name:        "user session token for a disabled user",
			tokenString: "test-token",
			setupMocks: userSessionTokenMocks(activeSession, &models.User{
				Metadata: models.ResourceMetadata{ID: userID},
				Active:   false,
			}),
			expectErrorMessage: "User has been disabled",
		},
		{
			name:        "service account token type",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, _ *db.Client) {
				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					PrivateClaims: map[string]string{
						"type":                 ServiceAccountTokenType,
						"service_account_id":   gid.ToGlobalID(types.ServiceAccountModelType, uuid.NewString()),
						"service_account_path": "test/path",
					},
				}, nil)
			},
			expectCaller: &ServiceAccountCaller{},
		},
		{
			name:        "job token type",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				mockJobs := db.NewMockJobs(t)
				mockJobs.On("GetJobByID", mock.Anything, jobID).Return(activeJob(models.JobRunning), nil)
				mockDBClient.Jobs = mockJobs

				token := jwt.New()
				jobTRN := "trn:job:group/workspace/" + gid.ToGlobalID(types.JobModelType, jobID)
				err := token.Set(jwt.SubjectKey, jobTRN)
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":         JobTokenType,
						"job_id":       gid.ToGlobalID(types.JobModelType, jobID),
						"job_trn":      jobTRN,
						"run_id":       gid.ToGlobalID(types.RunModelType, runID),
						"workspace_id": gid.ToGlobalID(types.WorkspaceModelType, workspaceID),
					},
				}, nil)
			},
			expectCaller: &JobCaller{
				Job:         activeJob(models.JobRunning),
				JobTRN:      "trn:job:group/workspace/" + gid.ToGlobalID(types.JobModelType, jobID),
				RunID:       runID,
				WorkspaceID: workspaceID,
			},
		},
		{
			// The caller is limited to reading the job itself; see TestJobCaller_FinishedJob.
			name:         "job token for a finished job is accepted",
			tokenString:  "test-token",
			setupMocks:   jobTokenMocks(activeJob(models.JobFinished)),
			expectCaller: jobTokenCaller(activeJob(models.JobFinished)),
		},
		{
			name:         "job token for a canceled job is accepted",
			tokenString:  "test-token",
			setupMocks:   jobTokenMocks(activeJob(models.JobCanceled)),
			expectCaller: jobTokenCaller(activeJob(models.JobCanceled)),
		},
		{
			name:               "job token for a missing job is rejected",
			tokenString:        "test-token",
			setupMocks:         jobTokenMocks(nil),
			expectErrorMessage: "job token is no longer valid",
		},
		{
			name:        "job token whose claims do not match the job is rejected",
			tokenString: "test-token",
			setupMocks: jobTokenMocks(func() *models.Job {
				job := activeJob(models.JobRunning)
				job.RunID = uuid.NewString()
				return job
			}()),
			expectErrorMessage: "job token is no longer valid",
		},
		{
			name:         "job token for a canceling job is accepted",
			tokenString:  "test-token",
			setupMocks:   jobTokenMocks(activeJob(models.JobCanceling)),
			expectCaller: jobTokenCaller(activeJob(models.JobCanceling)),
		},
		{
			name:        "scim token type with valid token",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":                SCIMTokenType,
						IDPIssuerURLClaimName: "https://idp.example.com",
					},
				}, nil)

				mockSCIMTokens := db.NewMockSCIMTokens(t)
				mockSCIMTokens.On("GetTokenByNonce", mock.Anything, "test-jti").Return(&models.SCIMToken{}, nil)
				mockDBClient.SCIMTokens = mockSCIMTokens
			},
			expectCaller: &SCIMCaller{},
		},
		{
			name:        "scim token type without an idp issuer url",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type": SCIMTokenType,
					},
				}, nil)

				mockSCIMTokens := db.NewMockSCIMTokens(t)
				mockSCIMTokens.On("GetTokenByNonce", mock.Anything, "test-jti").Return(&models.SCIMToken{}, nil)
				mockDBClient.SCIMTokens = mockSCIMTokens
			},
			expectErrorMessage: "scim token is missing the idp_issuer_url claim",
		},
		{
			name:        "scim token type with invalid token",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type": SCIMTokenType,
					},
				}, nil)

				mockSCIMTokens := db.NewMockSCIMTokens(t)
				mockSCIMTokens.On("GetTokenByNonce", mock.Anything, "test-jti").Return(nil, nil)
				mockDBClient.SCIMTokens = mockSCIMTokens
			},
			expectErrorMessage: "scim token has an invalid jti claim",
		},
		{
			name:        "vcs token type missing link_id claim",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, _ *db.Client) {
				token := jwt.New()
				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type": VCSWorkspaceLinkTokenType,
					},
				}, nil)
			},
			expectErrorMessage: "failed to get provider link id token claim",
		},
		{
			name:        "vcs token type with valid link",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":    VCSWorkspaceLinkTokenType,
						"link_id": gid.ToGlobalID(types.WorkspaceVCSProviderLinkModelType, linkID),
					},
				}, nil)

				mockLinks := db.NewMockWorkspaceVCSProviderLinks(t)
				mockLinks.On("GetLinkByID", mock.Anything, linkID).Return(&models.WorkspaceVCSProviderLink{
					ProviderID: "test-provider-id",
					TokenNonce: "test-jti",
					Metadata:   models.ResourceMetadata{ID: linkID},
				}, nil)
				mockDBClient.WorkspaceVCSProviderLinks = mockLinks

				mockProviders := db.NewMockVCSProviders(t)
				mockProviders.On("GetProviderByID", mock.Anything, "test-provider-id").Return(&models.VCSProvider{}, nil)
				mockDBClient.VCSProviders = mockProviders
			},
			expectCaller: &VCSWorkspaceLinkCaller{},
		},
		{
			name:        "vcs token type with invalid jti claim",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":    VCSWorkspaceLinkTokenType,
						"link_id": gid.ToGlobalID(types.WorkspaceVCSProviderLinkModelType, linkID),
					},
				}, nil)

				mockLinks := db.NewMockWorkspaceVCSProviderLinks(t)
				mockLinks.On("GetLinkByID", mock.Anything, linkID).Return(&models.WorkspaceVCSProviderLink{
					ProviderID: "test-provider-id",
					TokenNonce: "different-jti",
					Metadata:   models.ResourceMetadata{ID: linkID},
				}, nil)
				mockDBClient.WorkspaceVCSProviderLinks = mockLinks
			},
			expectErrorMessage: "vcs token has an invalid jti claim",
		},
		{
			name:        "vcs token type with missing provider",
			tokenString: "test-token",
			setupMocks: func(mockIDP *MockSigningKeyManager, mockDBClient *db.Client) {
				token := jwt.New()
				err := token.Set(jwt.JwtIDKey, "test-jti")
				require.NoError(t, err)

				mockIDP.On("VerifyToken", mock.Anything, "test-token").Return(&VerifyTokenOutput{
					Token: token,
					PrivateClaims: map[string]string{
						"type":    VCSWorkspaceLinkTokenType,
						"link_id": gid.ToGlobalID(types.WorkspaceVCSProviderLinkModelType, linkID),
					},
				}, nil)

				mockLinks := db.NewMockWorkspaceVCSProviderLinks(t)
				mockLinks.On("GetLinkByID", mock.Anything, linkID).Return(&models.WorkspaceVCSProviderLink{
					ProviderID: "test-provider-id",
					TokenNonce: "test-jti",
					Metadata:   models.ResourceMetadata{ID: linkID},
				}, nil)
				mockDBClient.WorkspaceVCSProviderLinks = mockLinks

				mockProviders := db.NewMockVCSProviders(t)
				mockProviders.On("GetProviderByID", mock.Anything, "test-provider-id").Return(nil, nil)
				mockDBClient.VCSProviders = mockProviders
			},
			expectErrorMessage: "failed to get vcs provider associated with link " + linkID,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockIDP := NewMockSigningKeyManager(t)
			mockDBClient := &db.Client{}
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			if test.setupMocks != nil {
				test.setupMocks(mockIDP, mockDBClient)
			}

			authenticator := &tharsisIDPTokenAuthenticator{
				issuerURL:          "https://test-issuer.com",
				signingKeyManager:  mockIDP,
				dbClient:           mockDBClient,
				maintenanceMonitor: mockMaintenanceMonitor,
			}

			caller, err := authenticator.Authenticate(context.Background(), test.tokenString, true)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, caller)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, caller)

				// A job caller carries the job loaded during authentication.
				if expected, ok := test.expectCaller.(*JobCaller); ok {
					require.NotNil(t, expected.Job, "job token cases must set the expected job")
					jobCaller, ok := caller.(*JobCaller)
					require.True(t, ok)
					assert.Equal(t, expected.Job, jobCaller.Job)
					assert.Equal(t, expected.JobTRN, jobCaller.JobTRN)
					assert.Equal(t, expected.RunID, jobCaller.RunID)
					assert.Equal(t, expected.WorkspaceID, jobCaller.WorkspaceID)
				}
			}
		})
	}
}

func TestErrorReason(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "expired token error",
			err:      errors.New("exp not satisfied"),
			expected: errExpired,
		},
		{
			name:     "iat not satisfied error",
			err:      errors.New("iat not satisfied"),
			expected: errIATInvalid,
		},
		{
			name:     "nbf not satisfied error",
			err:      errors.New("nbf not satisfied"),
			expected: errNBFInvalid,
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			expected: errUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := errorReason(test.err)
			assert.Equal(t, test.expected, result)
		})
	}
}
