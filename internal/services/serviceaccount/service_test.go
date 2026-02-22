package serviceaccount

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	terrs "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type keyPair struct {
	priv jwk.Key
	pub  jwk.Key
}

func TestGetServiceAccountByID(t *testing.T) {
	sampleServiceAccount := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  "service-account-id-1",
			TRN: types.ServiceAccountModelType.BuildTRN("my-group/service-account-1"),
		},
		Name:        "service-account-1",
		GroupID:     "group-1",
		Description: "test service account",
	}

	type testCase struct {
		name            string
		authError       error
		serviceAccount  *models.ServiceAccount
		expectErrorCode terrs.CodeType
	}

	testCases := []testCase{
		{
			name:           "successfully get service account by ID",
			serviceAccount: sampleServiceAccount,
		},
		{
			name:            "service account not found",
			expectErrorCode: terrs.ENotFound,
		},
		{
			name:            "subject is not authorized to view service account",
			serviceAccount:  sampleServiceAccount,
			authError:       terrs.New("Forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrorCode: terrs.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(test.serviceAccount, nil)

			if test.serviceAccount != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				ServiceAccounts: mockServiceAccounts,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualServiceAccount, err := service.GetServiceAccountByID(auth.WithCaller(ctx, mockCaller), sampleServiceAccount.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.serviceAccount, actualServiceAccount)
		})
	}
}

func TestGetServiceAccountByTRN(t *testing.T) {
	sampleServiceAccount := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  "service-account-id-1",
			TRN: types.ServiceAccountModelType.BuildTRN("my-group/service-account-1"),
		},
		Name:        "service-account-1",
		GroupID:     "group-1",
		Description: "test service account",
	}

	type testCase struct {
		name            string
		authError       error
		serviceAccount  *models.ServiceAccount
		expectErrorCode terrs.CodeType
	}

	testCases := []testCase{
		{
			name:           "successfully get service account by trn",
			serviceAccount: sampleServiceAccount,
		},
		{
			name:            "service account not found",
			expectErrorCode: terrs.ENotFound,
		},
		{
			name:            "subject is not authorized to view service account",
			serviceAccount:  sampleServiceAccount,
			authError:       terrs.New("Forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrorCode: terrs.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockServiceAccounts.On("GetServiceAccountByTRN", mock.Anything, sampleServiceAccount.Metadata.TRN).Return(test.serviceAccount, nil)

			if test.serviceAccount != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				ServiceAccounts: mockServiceAccounts,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualServiceAccount, err := service.GetServiceAccountByTRN(auth.WithCaller(ctx, mockCaller), sampleServiceAccount.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.serviceAccount, actualServiceAccount)
		})
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	serviceAccountID := "12345678-1234-1234-1234-123456789012"
	groupID := "group-123"
	resourcePath := "group-name/test-sa"

	existingSA := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{ID: serviceAccountID, TRN: types.ServiceAccountModelType.BuildTRN(resourcePath), Version: 1},
		Name:     "test-sa",
		GroupID:  groupID,
	}

	tests := []struct {
		name          string
		existingSA    *models.ServiceAccount
		authError     error
		expectErrCode terrs.CodeType
	}{
		{
			name:       "delete service account",
			existingSA: existingSA,
		},
		{
			name:          "not found",
			existingSA:    nil,
			expectErrCode: terrs.ENotFound,
		},
		{
			name:          "permission denied",
			existingSA:    existingSA,
			authError:     terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(test.existingSA, nil)

			if test.existingSA != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.DeleteServiceAccountPermission, mock.Anything).Return(test.authError)

				if test.authError == nil {
					mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
					mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
					mockServiceAccounts.On("DeleteServiceAccount", mock.Anything, test.existingSA).Return(nil)
					mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
				}
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{ServiceAccounts: mockServiceAccounts, Transactions: mockTransactions}

			service := &service{logger: testLogger, dbClient: dbClient, activityService: mockActivityEvents, secretMaxExpirationDays: 90}

			err := service.DeleteServiceAccount(auth.WithCaller(ctx, &mockCaller), &DeleteServiceAccountInput{ID: serviceAccountID})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetServiceAccounts(t *testing.T) {
	groupPath := "group-name"

	tests := []struct {
		name          string
		authError     error
		expectErrCode terrs.CodeType
	}{
		{
			name: "get service accounts",
		},
		{
			name:          "permission denied",
			authError:     terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockCaller.On("RequirePermission", mock.Anything, models.ViewServiceAccountPermission, mock.Anything).Return(test.authError)

			if test.authError == nil {
				mockServiceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.ServiceAccountsResult{}, nil)
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{ServiceAccounts: mockServiceAccounts}

			service := &service{logger: testLogger, dbClient: dbClient}

			_, err := service.GetServiceAccounts(auth.WithCaller(ctx, &mockCaller), &GetServiceAccountsInput{NamespacePath: groupPath})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetServiceAccountsByIDs(t *testing.T) {
	saID := "12345678-1234-1234-1234-123456789012"
	groupID := "group-123"
	resourcePath := "group-name/test-sa"

	sa := models.ServiceAccount{
		Metadata: models.ResourceMetadata{ID: saID, TRN: types.ServiceAccountModelType.BuildTRN(resourcePath)},
		Name:     "test-sa",
		GroupID:  groupID,
	}

	tests := []struct {
		name          string
		ids           []string
		dbResult      []models.ServiceAccount
		authError     error
		expectErrCode terrs.CodeType
	}{
		{
			name:     "get by ids",
			ids:      []string{saID},
			dbResult: []models.ServiceAccount{sa},
		},
		{
			name:          "permission denied",
			ids:           []string{saID},
			dbResult:      []models.ServiceAccount{sa},
			authError:     terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockServiceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).
				Return(&db.ServiceAccountsResult{ServiceAccounts: test.dbResult}, nil)

			if len(test.dbResult) > 0 {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything).Return(test.authError)
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{ServiceAccounts: mockServiceAccounts}

			service := &service{logger: testLogger, dbClient: dbClient}

			result, err := service.GetServiceAccountsByIDs(auth.WithCaller(ctx, &mockCaller), test.ids)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(test.dbResult))
		})
	}
}

func TestCreateServiceAccount(t *testing.T) {
	serviceAccountName := "test-service-account"
	serviceAccountDescription := "test service account description"
	groupName := "group-name"
	groupID := "group123"
	createdBy := "service-account-created-by"
	resourcePath := groupName + "/" + serviceAccountName
	issuer := "http://some/identity/issuer"
	claimKey := "bound-claim-key"
	claimVal := "bound-claim-value"

	clientSecretExpiry := time.Now().Add(48 * time.Hour)

	// Test cases
	tests := []struct {
		authError                      error
		expectCreatedServiceAccount    *models.ServiceAccount
		name                           string
		expectErrCode                  terrs.CodeType
		input                          CreateServiceAccountInput
		limit                          int
		injectServiceAccountsPerGroup  int32
		exceedsLimit                   bool
		expectClientCredentialsEnabled bool
	}{
		{
			name: "create service account",
			input: CreateServiceAccountInput{
				Name:        serviceAccountName,
				Description: serviceAccountDescription,
				GroupID:     groupID,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:          issuer,
						BoundClaimsType: models.BoundClaimsTypeString,
						BoundClaims:     map[string]string{claimKey: claimVal},
					},
				},
			},
			expectCreatedServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:  groupID,
					TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
				},
				Name:        serviceAccountName,
				Description: serviceAccountDescription,
				GroupID:     groupID,
				CreatedBy:   createdBy,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:          issuer,
						BoundClaimsType: models.BoundClaimsTypeString,
						BoundClaims:     map[string]string{claimKey: claimVal},
					},
				},
			},
			limit:                         5,
			injectServiceAccountsPerGroup: 5,
		},
		{
			name: "create service account with client credentials",
			input: CreateServiceAccountInput{
				Name:                    serviceAccountName,
				Description:             serviceAccountDescription,
				GroupID:                 groupID,
				EnableClientCredentials: true,
				ClientSecretExpiresAt:   &clientSecretExpiry,
			},
			expectCreatedServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:  groupID,
					TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
				},
				Name:                  serviceAccountName,
				Description:           serviceAccountDescription,
				GroupID:               groupID,
				CreatedBy:             createdBy,
				ClientSecretHash:      ptr.String("hash"),
				ClientSecretExpiresAt: &clientSecretExpiry,
			},
			limit:                          5,
			injectServiceAccountsPerGroup:  5,
			expectClientCredentialsEnabled: true,
		},
		{
			name: "subject does not have permission",
			input: CreateServiceAccountInput{
				Name:        serviceAccountName,
				Description: serviceAccountDescription,
				GroupID:     groupID,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:          issuer,
						BoundClaimsType: models.BoundClaimsTypeString,
						BoundClaims:     map[string]string{claimKey: claimVal},
					},
				},
			},
			authError:     terrs.New("Unauthorized", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "exceeds limit",
			input: CreateServiceAccountInput{
				Name:        serviceAccountName,
				Description: serviceAccountDescription,
				GroupID:     groupID,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:          issuer,
						BoundClaimsType: models.BoundClaimsTypeString,
						BoundClaims:     map[string]string{claimKey: claimVal},
					},
				},
			},
			expectCreatedServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:  groupID,
					TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
				},
				Name:        serviceAccountName,
				Description: serviceAccountDescription,
				GroupID:     groupID,
				CreatedBy:   createdBy,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:          issuer,
						BoundClaimsType: models.BoundClaimsTypeString,
						BoundClaims:     map[string]string{claimKey: claimVal},
					},
				},
			},
			limit:                         5,
			injectServiceAccountsPerGroup: 6,
			exceedsLimit:                  true,
			expectErrCode:                 terrs.EInvalid,
		},
	}
	for _, t1 := range tests {
		t.Run(t1.name, func(t *testing.T) {
			test := t1

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateServiceAccountPermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			if (test.expectCreatedServiceAccount != nil) || test.exceedsLimit {
				mockServiceAccounts.On("CreateServiceAccount", mock.Anything, mock.Anything).
					Return(test.expectCreatedServiceAccount, nil)
			}

			dbClient := db.Client{
				Transactions:    mockTransactions,
				ServiceAccounts: mockServiceAccounts,
				ResourceLimits:  mockResourceLimits,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil && !test.exceedsLimit {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockServiceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.GetServiceAccountsInput{
					Filter: &db.ServiceAccountFilter{
						NamespacePaths: []string{groupName},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetServiceAccountsInput) *db.ServiceAccountsResult {
					_ = ctx
					_ = input

					return &db.ServiceAccountsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectServiceAccountsPerGroup,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := &service{logger: testLogger, dbClient: &dbClient, limitChecker: limits.NewLimitChecker(&dbClient), activityService: mockActivityEvents, secretMaxExpirationDays: 90}

			response, err := service.CreateServiceAccount(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedServiceAccount.Name, response.ServiceAccount.Name)
			assert.Equal(t, test.expectCreatedServiceAccount.Description, response.ServiceAccount.Description)
			assert.Equal(t, test.expectCreatedServiceAccount.GroupID, response.ServiceAccount.GroupID)

			if test.expectClientCredentialsEnabled {
				assert.True(t, response.ServiceAccount.ClientCredentialsEnabled())
				assert.NotNil(t, response.ClientSecret)
			}
		})
	}
}

func TestUpdateServiceAccount(t *testing.T) {
	serviceAccountID := "12345678-1234-1234-1234-123456789012"
	groupID := "group-123"
	resourcePath := "group-name/test-sa"
	clientSecretExpiry := time.Now().Add(48 * time.Hour)

	existingSA := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:      serviceAccountID,
			TRN:     types.ServiceAccountModelType.BuildTRN(resourcePath),
			Version: 1,
		},
		Name:        "test-sa",
		Description: "original",
		GroupID:     groupID,
		OIDCTrustPolicies: []models.OIDCTrustPolicy{
			{Issuer: "https://issuer.example.com", BoundClaimsType: models.BoundClaimsTypeString, BoundClaims: map[string]string{"sub": "test"}},
		},
	}

	tests := []struct {
		name                           string
		input                          *UpdateServiceAccountInput
		existingSA                     *models.ServiceAccount
		authError                      error
		expectErrCode                  terrs.CodeType
		expectClientCredentialsEnabled bool
	}{
		{
			name: "update description",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: ptr.String("updated"),
			},
			existingSA: existingSA,
		},
		{
			name: "enable client credentials",
			input: &UpdateServiceAccountInput{
				ID:                      serviceAccountID,
				EnableClientCredentials: ptr.Bool(true),
				ClientSecretExpiresAt:   &clientSecretExpiry,
			},
			existingSA:                     existingSA,
			expectClientCredentialsEnabled: true,
		},
		{
			name: "permission denied",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: ptr.String("updated"),
			},
			existingSA:    existingSA,
			authError:     terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "not found",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: ptr.String("updated"),
			},
			existingSA:    nil,
			expectErrCode: terrs.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).
				Return(test.existingSA, nil)

			if test.existingSA != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).
					Return(test.authError)

				if test.authError == nil {
					mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
					mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)

					updatedSA := *test.existingSA
					if test.expectClientCredentialsEnabled {
						updatedSA.ClientSecretHash = ptr.String("hash")
						updatedSA.ClientSecretExpiresAt = &clientSecretExpiry
					}
					mockServiceAccounts.On("UpdateServiceAccount", mock.Anything, mock.Anything).
						Return(&updatedSA, nil)
					mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
						Return(&models.ActivityEvent{}, nil)
				}
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				ServiceAccounts: mockServiceAccounts,
				Transactions:    mockTransactions,
			}

			service := &service{logger: testLogger, dbClient: dbClient, activityService: mockActivityEvents, secretMaxExpirationDays: 90}

			response, err := service.UpdateServiceAccount(auth.WithCaller(ctx, &mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, response.ServiceAccount)

			if test.expectClientCredentialsEnabled {
				assert.True(t, response.ServiceAccount.ClientCredentialsEnabled())
				assert.NotNil(t, response.ClientSecret)
			}
		})
	}
}

func TestCreateOIDCToken(t *testing.T) {
	validKeyPair := createKeyPair(t)

	keyID := validKeyPair.pub.KeyID()
	serviceAccountID := "d4a94ff5-154e-4758-8039-55e2147fa154"
	serviceAccountGID := gid.ToGlobalID(types.ServiceAccountModelType, serviceAccountID)
	serviceAccountTRN := types.ServiceAccountModelType.BuildTRN("groupA/serviceAccount1")
	issuer := "https://test.tharsis"
	sub := "testSubject1"

	basicPolicy := []models.OIDCTrustPolicy{
		{
			Issuer: issuer,
			BoundClaims: map[string]string{
				"sub": sub,
				"aud": "tharsis",
			},
		},
	}

	// Test cases
	tests := []struct {
		expectErr      error
		name           string
		serviceAccount string
		policy         []models.OIDCTrustPolicy
		token          []byte
		isTRN          bool
	}{
		{
			name:           "subject claim doesn't match",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("of the trust policies for issuer https://test.tharsis, none was satisfied"),
		},
		{
			name:           "no matching trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://notavalidissuer",
					BoundClaims: map[string]string{},
				},
			},
			expectErr: errFailedCreateOIDCToken,
		},
		{
			name:           "empty trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         []models.OIDCTrustPolicy{},
			expectErr:      errFailedCreateOIDCToken,
		},
		{
			name:           "invalid token",
			serviceAccount: "groupA/serviceAccount1",
			token:          []byte("invalidtoken"),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to decode token: invalid JWT"),
		},
		{
			name:           "missing issuer",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "", sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("JWT is missing issuer claim"),
		},
		{
			name:           "empty service account ID",
			serviceAccount: "",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errFailedCreateOIDCToken,
		},
		{
			name:           "using TRN as service account ID",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			isTRN:          true,
		},
		{
			name:           "using GID as service account ID",
			serviceAccount: serviceAccountGID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
		},
		{
			name:           "negative: multiple trust policies with same issuer: all mismatch",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "noMatchSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
			expectErr: errors.New("of the trust policies for issuer https://test.tharsis, none was satisfied"),
		},
		{
			name:           "positive: multiple trust policies with same issuer: match first",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "firstSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
		},
		{
			name:           "positive: multiple trust policies with same issuer: match second",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "secondSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
		},
		{
			name:           "positive: trust policy issuer has forward slash, token issuer does not",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis/",
					BoundClaims: map[string]string{},
				},
			},
		},
		{
			name:           "positive: token issuer has forward slash, trust policy issuer does not",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis/", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis",
					BoundClaims: map[string]string{},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sa := models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:  serviceAccountID,
					TRN: serviceAccountTRN,
				},
				Name:              "serviceAccount1",
				OIDCTrustPolicies: test.policy,
			}

			mockServiceAccounts := db.NewMockServiceAccounts(t)

			// Set up the appropriate mock based on whether we're testing with a TRN or GID
			if test.serviceAccount != "" {
				if test.isTRN {
					mockServiceAccounts.On("GetServiceAccountByTRN", mock.Anything, test.serviceAccount).Return(&sa, nil)
				} else {
					mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).Return(&sa, nil).Maybe()
				}
			}

			mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			mockSigningKeyManager.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *auth.TokenInput) bool {

				if input.Subject != sa.GetResourcePath() {
					return false
				}
				privClaims := input.Claims

				return privClaims["service_account_id"] == gid.ToGlobalID(types.ServiceAccountModelType, sa.Metadata.ID) &&
					privClaims["service_account_name"] == sa.Name &&
					privClaims["service_account_path"] == sa.GetResourcePath()
			})).Return([]byte("signedtoken"), nil).Maybe()

			mockResourceLimits := db.NewMockResourceLimits(t)

			dbClient := db.Client{
				ServiceAccounts: mockServiceAccounts,
				ResourceLimits:  mockResourceLimits,
			}

			mockConfigFetcher := auth.NewMockOpenIDConfigFetcher(t)

			mockActivityEvents := activityevent.NewMockService(t)
			mockTokenVerifier := auth.NewMockOIDCTokenVerifier(t)

			mockTokenVerifier.On("VerifyToken", mock.Anything, string(test.token), mock.Anything).Return(
				nil,
				func(_ context.Context, token string, validationOptions []jwt.ValidateOption) error {
					parseOptions := []jwt.ParseOption{
						jwt.WithVerify(false),
						jwt.WithValidate(true),
					}
					for _, o := range validationOptions {
						parseOptions = append(parseOptions, o)
					}
					_, err := jwt.Parse([]byte(token), parseOptions...)
					return err
				},
			).Maybe()

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:              testLogger,
				dbClient:            &dbClient,
				limitChecker:        limits.NewLimitChecker(&dbClient),
				signingKeyManager:   mockSigningKeyManager,
				openIDConfigFetcher: mockConfigFetcher,
				activityService:     mockActivityEvents,
				buildOIDCTokenVerifier: func(_ context.Context, _ []string, _ auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier {
					return mockTokenVerifier
				},
				secretMaxExpirationDays: 90,
			}

			resp, err := service.CreateOIDCToken(ctx, &CreateOIDCTokenInput{ServiceAccountPublicID: test.serviceAccount, Token: test.token})
			if err != nil && test.expectErr == nil {
				t.Fatal(err)
			}

			if test.expectErr == nil {
				expected := CreateTokenResponse{
					Token:     []byte("signedtoken"),
					ExpiresIn: int32(serviceAccountLoginDuration / time.Second),
				}

				assert.Equal(t, &expected, resp)
			} else {
				assert.EqualError(t, err, test.expectErr.Error())
			}
		})
	}
}

func TestCreateClientCredentialsToken(t *testing.T) {
	serviceAccountID := "12345678-1234-1234-1234-123456789012"
	serviceAccountGID := gid.ToGlobalID(types.ServiceAccountModelType, serviceAccountID)
	serviceAccountTRN := types.ServiceAccountModelType.BuildTRN("group-123/test-sa")
	clientSecret := "test-secret"
	resourcePath := "group-123/test-sa"

	// Create a service account with client credentials enabled
	saWithClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  serviceAccountID,
			TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
		},
		Name:    "test-sa",
		GroupID: "group-123",
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	saWithClientCreds.ClientSecretHash = ptr.String(string(hash))

	saWithoutClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  serviceAccountID,
			TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
		},
		Name:    "test-sa-no-client-creds",
		GroupID: "group-123",
	}

	tests := []struct {
		name           string
		input          *CreateClientCredentialsTokenInput
		serviceAccount *models.ServiceAccount
		useTRN         bool
		expectErrCode  terrs.CodeType
	}{
		{
			name: "successful token creation with GID",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountGID,
				ClientSecret: clientSecret,
			},
			serviceAccount: saWithClientCreds,
		},
		{
			name: "successful token creation with TRN",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountTRN,
				ClientSecret: clientSecret,
			},
			serviceAccount: saWithClientCreds,
			useTRN:         true,
		},
		{
			name: "empty client ID",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     "",
				ClientSecret: clientSecret,
			},
			expectErrCode: terrs.EUnauthorized,
		},
		{
			name: "empty client secret",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountGID,
				ClientSecret: "",
			},
			expectErrCode: terrs.EUnauthorized,
		},
		{
			name: "service account not found",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountGID,
				ClientSecret: clientSecret,
			},
			serviceAccount: nil,
			expectErrCode:  terrs.EUnauthorized,
		},
		{
			name: "client credentials not enabled",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountGID,
				ClientSecret: clientSecret,
			},
			serviceAccount: saWithoutClientCreds,
			expectErrCode:  terrs.EUnauthorized,
		},
		{
			name: "invalid client secret",
			input: &CreateClientCredentialsTokenInput{
				ClientID:     serviceAccountGID,
				ClientSecret: "wrong-secret",
			},
			serviceAccount: saWithClientCreds,
			expectErrCode:  terrs.EUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			if test.input.ClientID != "" && test.input.ClientSecret != "" {
				if test.useTRN {
					mockServiceAccounts.On("GetServiceAccountByTRN", mock.Anything, serviceAccountTRN).
						Return(test.serviceAccount, nil)
				} else {
					mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).
						Return(test.serviceAccount, nil)
				}
			}

			if test.expectErrCode == "" {
				mockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).
					Return([]byte("test-token"), nil)
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{ServiceAccounts: mockServiceAccounts}

			service := &service{
				logger:            testLogger,
				dbClient:          dbClient,
				signingKeyManager: mockSigningKeyManager,
			}

			response, err := service.CreateClientCredentialsToken(ctx, test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, response.Token)
			assert.Equal(t, int32(serviceAccountLoginDuration.Seconds()), response.ExpiresIn)
		})
	}
}

func TestResetClientCredentials(t *testing.T) {
	serviceAccountID := "12345678-1234-1234-1234-123456789012"
	resourcePath := "group-123/test-sa"

	saWithClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:      serviceAccountID,
			TRN:     types.ServiceAccountModelType.BuildTRN(resourcePath),
			Version: 1,
		},
		Name:             "test-sa",
		GroupID:          "group-123",
		ClientSecretHash: ptr.String("existing-hash"),
	}

	saWithoutClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  serviceAccountID,
			TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
		},
		Name:    "test-sa-no-client-creds",
		GroupID: "group-123",
	}

	tests := []struct {
		name           string
		input          *ResetClientCredentialsInput
		serviceAccount *models.ServiceAccount
		authError      error
		expectErrCode  terrs.CodeType
	}{
		{
			name: "successful reset",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			serviceAccount: saWithClientCreds,
		},
		{
			name: "service account not found",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			serviceAccount: nil,
			expectErrCode:  terrs.ENotFound,
		},
		{
			name: "permission denied",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			serviceAccount: saWithClientCreds,
			authError:      terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode:  terrs.EForbidden,
		},
		{
			name: "client credentials not enabled",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			serviceAccount: saWithoutClientCreds,
			expectErrCode:  terrs.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).
				Return(test.serviceAccount, nil)

			if test.serviceAccount != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).
					Return(test.authError)
			}

			if test.expectErrCode == "" {
				mockServiceAccounts.On("UpdateServiceAccount", mock.Anything, mock.Anything).
					Return(test.serviceAccount, nil)
			}

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{ServiceAccounts: mockServiceAccounts}

			service := &service{
				logger:                  testLogger,
				dbClient:                dbClient,
				secretMaxExpirationDays: 90,
			}

			response, err := service.ResetClientCredentials(auth.WithCaller(ctx, &mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, response.ServiceAccount)
			assert.NotNil(t, response.ClientSecret)
		})
	}
}

func createKeyPair(t *testing.T) keyPair {
	rsaPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	privKey, err := jwk.FromRaw(rsaPrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := jwk.FromRaw(rsaPrivKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := jwk.AssignKeyID(pubKey); err != nil {
		t.Fatal(err)
	}

	return keyPair{priv: privKey, pub: pubKey}
}

func createJWT(t *testing.T, key jwk.Key, keyID string, issuer string, sub string, exp time.Time) []byte {
	token := jwt.New()

	_ = token.Set(jwt.ExpirationKey, exp.Unix())
	_ = token.Set(jwt.SubjectKey, sub)
	_ = token.Set(jwt.AudienceKey, "tharsis")
	if issuer != "" {
		_ = token.Set(jwt.IssuerKey, issuer)
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.TypeKey, "JWT")
	_ = hdrs.Set(jws.KeyIDKey, keyID)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		t.Fatal(err)
	}

	return signed
}
