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

	// Test cases
	tests := []struct {
		authError                     error
		expectCreatedServiceAccount   *models.ServiceAccount
		name                          string
		expectErrCode                 terrs.CodeType
		input                         models.ServiceAccount
		limit                         int
		injectServiceAccountsPerGroup int32
		exceedsLimit                  bool
	}{
		{
			name: "create service account",
			input: models.ServiceAccount{
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
			name: "subject does not have permission",
			input: models.ServiceAccount{
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
			authError:     terrs.New("Unauthorized", terrs.WithErrorCode(terrs.EForbidden)),
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "exceeds limit",
			input: models.ServiceAccount{
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

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, nil, mockActivityEvents)

			serviceAccount, err := service.CreateServiceAccount(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedServiceAccount, serviceAccount)
		})
	}
}

func TestCreateToken(t *testing.T) {
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
			expectErr: errFailedCreateToken,
		},
		{
			name:           "empty trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         []models.OIDCTrustPolicy{},
			expectErr:      errFailedCreateToken,
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
			expectErr:      errFailedCreateToken,
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

			MockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			MockSigningKeyManager.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *auth.TokenInput) bool {

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

			service := newService(
				testLogger,
				&dbClient,
				limits.NewLimitChecker(&dbClient),
				MockSigningKeyManager,
				mockConfigFetcher,
				mockActivityEvents,
				func(_ context.Context, _ []string, _ auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier {
					return mockTokenVerifier
				})

			resp, err := service.CreateToken(ctx, &CreateTokenInput{ServiceAccountPublicID: test.serviceAccount, Token: test.token})
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
