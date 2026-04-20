package serviceaccount

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
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

type serviceAccountMocks struct {
	caller               *auth.MockCaller
	serviceAccounts      *db.MockServiceAccounts
	namespaceMemberships *db.MockNamespaceMemberships
	transactions         *db.MockTransactions
	activityEvents       *activityevent.MockService
	limitChecker         *limits.MockLimitChecker
	signingKeyMgr        *auth.MockSigningKeyManager
	configFetcher        *auth.MockOpenIDConfigFetcher
	tokenVerifier        *auth.MockOIDCTokenVerifier
}

func newServiceAccountMocks(t *testing.T) *serviceAccountMocks {
	return &serviceAccountMocks{
		caller:               auth.NewMockCaller(t),
		serviceAccounts:      db.NewMockServiceAccounts(t),
		namespaceMemberships: db.NewMockNamespaceMemberships(t),
		transactions:         db.NewMockTransactions(t),
		activityEvents:       activityevent.NewMockService(t),
		limitChecker:         limits.NewMockLimitChecker(t),
		signingKeyMgr:        auth.NewMockSigningKeyManager(t),
		configFetcher:        auth.NewMockOpenIDConfigFetcher(t),
		tokenVerifier:        auth.NewMockOIDCTokenVerifier(t),
	}
}

func (m *serviceAccountMocks) dbClient() *db.Client {
	return &db.Client{
		ServiceAccounts:      m.serviceAccounts,
		NamespaceMemberships: m.namespaceMemberships,
		Transactions:         m.transactions,
	}
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
		setupMocks      func(*serviceAccountMocks)
		expectErrorCode terrs.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get service account by ID",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(sampleServiceAccount, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "service account not found",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(nil, nil)
			},
			expectErrorCode: terrs.ENotFound,
		},
		{
			name: "subject is not authorized to view service account",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(sampleServiceAccount, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(terrs.New("Forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrorCode: terrs.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			service := &service{dbClient: m.dbClient()}

			actualServiceAccount, err := service.GetServiceAccountByID(auth.WithCaller(t.Context(), m.caller), sampleServiceAccount.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, sampleServiceAccount, actualServiceAccount)
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
		setupMocks      func(*serviceAccountMocks)
		expectErrorCode terrs.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get service account by trn",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByTRN", mock.Anything, sampleServiceAccount.Metadata.TRN).Return(sampleServiceAccount, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "service account not found",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByTRN", mock.Anything, sampleServiceAccount.Metadata.TRN).Return(nil, nil)
			},
			expectErrorCode: terrs.ENotFound,
		},
		{
			name: "subject is not authorized to view service account",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByTRN", mock.Anything, sampleServiceAccount.Metadata.TRN).Return(sampleServiceAccount, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything, mock.Anything).Return(terrs.New("Forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrorCode: terrs.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			service := &service{dbClient: m.dbClient()}

			actualServiceAccount, err := service.GetServiceAccountByTRN(auth.WithCaller(t.Context(), m.caller), sampleServiceAccount.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, sampleServiceAccount, actualServiceAccount)
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
		setupMocks    func(*serviceAccountMocks)
		expectErrCode terrs.CodeType
	}{
		{
			name: "delete service account",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("RequirePermission", mock.Anything, models.DeleteServiceAccountPermission, mock.Anything).Return(nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.serviceAccounts.On("DeleteServiceAccount", mock.Anything, existingSA).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			},
		},
		{
			name: "not found",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(nil, nil)
			},
			expectErrCode: terrs.ENotFound,
		},
		{
			name: "permission denied",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("RequirePermission", mock.Anything, models.DeleteServiceAccountPermission, mock.Anything).Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:                  testLogger,
				dbClient:                m.dbClient(),
				activityService:         m.activityEvents,
				secretMaxExpirationDays: 90,
			}

			err := service.DeleteServiceAccount(auth.WithCaller(t.Context(), m.caller), &DeleteServiceAccountInput{ID: serviceAccountID})

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
		setupMocks    func(*serviceAccountMocks)
		expectErrCode terrs.CodeType
	}{
		{
			name: "get service accounts",
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.ViewServiceAccountPermission, mock.Anything).Return(nil)
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.ServiceAccountsResult{}, nil)
			},
		},
		{
			name: "permission denied",
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.ViewServiceAccountPermission, mock.Anything).Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{logger: testLogger, dbClient: m.dbClient()}

			_, err := service.GetServiceAccounts(auth.WithCaller(t.Context(), m.caller), &GetServiceAccountsInput{NamespacePath: groupPath})

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
		setupMocks    func(*serviceAccountMocks)
		expectErrCode terrs.CodeType
	}{
		{
			name: "get by ids",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).
					Return(&db.ServiceAccountsResult{ServiceAccounts: []models.ServiceAccount{sa}}, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything).Return(nil)
			},
		},
		{
			name: "permission denied",
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).
					Return(&db.ServiceAccountsResult{ServiceAccounts: []models.ServiceAccount{sa}}, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything, types.ServiceAccountModelType, mock.Anything).Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{logger: testLogger, dbClient: m.dbClient()}

			result, err := service.GetServiceAccountsByIDs(auth.WithCaller(t.Context(), m.caller), []string{saID})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, 1)
		})
	}
}

func TestCreateServiceAccount(t *testing.T) {
	serviceAccountName := "test-service-account"
	serviceAccountDescription := "test service account description"
	groupName := "group-name"
	groupID := "group123"
	resourcePath := groupName + "/" + serviceAccountName
	issuer := "http://some/identity/issuer"
	claimKey := "bound-claim-key"
	claimVal := "bound-claim-value"

	clientSecretExpiry := time.Now().Add(48 * time.Hour)

	sampleOIDCTrustPolicy := []models.OIDCTrustPolicy{
		{
			Issuer:          issuer,
			BoundClaimsType: models.BoundClaimsTypeString,
			BoundClaims:     map[string]string{claimKey: claimVal},
		},
	}

	createdSA := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  groupID,
			TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
		},
		Name:              serviceAccountName,
		Description:       serviceAccountDescription,
		GroupID:           groupID,
		CreatedBy:         "mockSubject",
		OIDCTrustPolicies: sampleOIDCTrustPolicy,
	}

	tests := []struct {
		name                           string
		input                          CreateServiceAccountInput
		setupMocks                     func(*serviceAccountMocks)
		expectErrCode                  terrs.CodeType
		expectClientCredentialsEnabled bool
	}{
		{
			name: "create service account",
			input: CreateServiceAccountInput{
				Name:              serviceAccountName,
				Description:       serviceAccountDescription,
				GroupID:           groupID,
				OIDCTrustPolicies: sampleOIDCTrustPolicy,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.CreateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("GetSubject").Return("mockSubject")
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.serviceAccounts.On("CreateServiceAccount", mock.Anything, mock.Anything).Return(createdSA, nil)
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.ServiceAccountsResult{
					PageInfo: &pagination.PageInfo{TotalCount: 5},
				}, nil)
				m.limitChecker.On("CheckLimit", mock.Anything, mock.Anything, int32(5)).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			},
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
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.CreateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("GetSubject").Return("mockSubject")
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.serviceAccounts.On("CreateServiceAccount", mock.Anything, mock.Anything).Return(&models.ServiceAccount{
					Metadata:              createdSA.Metadata,
					Name:                  serviceAccountName,
					Description:           serviceAccountDescription,
					GroupID:               groupID,
					CreatedBy:             "mockSubject",
					ClientSecretHash:      ptr.String("hash"),
					ClientSecretExpiresAt: &clientSecretExpiry,
				}, nil)
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.ServiceAccountsResult{
					PageInfo: &pagination.PageInfo{TotalCount: 5},
				}, nil)
				m.limitChecker.On("CheckLimit", mock.Anything, mock.Anything, int32(5)).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			},
			expectClientCredentialsEnabled: true,
		},
		{
			name: "subject does not have permission",
			input: CreateServiceAccountInput{
				Name:              serviceAccountName,
				Description:       serviceAccountDescription,
				GroupID:           groupID,
				OIDCTrustPolicies: sampleOIDCTrustPolicy,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.CreateServiceAccountPermission, mock.Anything).Return(terrs.New("Unauthorized", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "exceeds limit",
			input: CreateServiceAccountInput{
				Name:              serviceAccountName,
				Description:       serviceAccountDescription,
				GroupID:           groupID,
				OIDCTrustPolicies: sampleOIDCTrustPolicy,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.caller.On("RequirePermission", mock.Anything, models.CreateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("GetSubject").Return("mockSubject")
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.serviceAccounts.On("CreateServiceAccount", mock.Anything, mock.Anything).Return(createdSA, nil)
				m.serviceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).Return(&db.ServiceAccountsResult{
					PageInfo: &pagination.PageInfo{TotalCount: 6},
				}, nil)
				m.limitChecker.On("CheckLimit", mock.Anything, mock.Anything, int32(6)).Return(terrs.New("limit exceeded", terrs.WithErrorCode(terrs.EInvalid)))
			},
			expectErrCode: terrs.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:                  testLogger,
				dbClient:                m.dbClient(),
				limitChecker:            m.limitChecker,
				activityService:         m.activityEvents,
				secretMaxExpirationDays: 90,
			}

			response, err := service.CreateServiceAccount(auth.WithCaller(t.Context(), m.caller), &test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, terrs.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, serviceAccountName, response.ServiceAccount.Name)
			assert.Equal(t, serviceAccountDescription, response.ServiceAccount.Description)
			assert.Equal(t, groupID, response.ServiceAccount.GroupID)

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
	groupPath := "group-name"
	resourcePath := groupPath + "/test-sa"
	updatedDescription := "updated"
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

	type testCase struct {
		name                           string
		input                          *UpdateServiceAccountInput
		setupMocks                     func(*serviceAccountMocks)
		expectErrCode                  terrs.CodeType
		expectClientCredentialsEnabled bool
	}

	tests := []testCase{
		{
			name: "update description with no memberships",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)

				updated := *existingSA
				updated.Description = updatedDescription
				m.serviceAccounts.On("UpdateServiceAccount", mock.Anything, &updated).Return(&updated, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					NamespacePath: &groupPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetServiceAccount,
					TargetID:      serviceAccountID,
				}).Return(&models.ActivityEvent{}, nil)
			},
		},
		{
			name: "update with memberships when caller is owner",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: []models.NamespaceMembership{
						{Namespace: models.MembershipNamespace{Path: "other-group"}},
					},
					PageInfo: &pagination.PageInfo{TotalCount: 1},
				}, nil)
				m.caller.On("RequireRole", mock.Anything, models.OwnerRoleID.String(), mock.Anything).Return(nil)

				updated := *existingSA
				updated.Description = updatedDescription
				m.serviceAccounts.On("UpdateServiceAccount", mock.Anything, &updated).Return(&updated, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					NamespacePath: &groupPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetServiceAccount,
					TargetID:      serviceAccountID,
				}).Return(&models.ActivityEvent{}, nil)
			},
		},
		{
			name: "admin bypasses ownership check on service account with memberships",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("IsAdmin").Return(true)

				updated := *existingSA
				updated.Description = updatedDescription
				m.serviceAccounts.On("UpdateServiceAccount", mock.Anything, &updated).Return(&updated, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					NamespacePath: &groupPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetServiceAccount,
					TargetID:      serviceAccountID,
				}).Return(&models.ActivityEvent{}, nil)
			},
		},
		{
			name: "non-owner cannot update service account with memberships",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: []models.NamespaceMembership{
						{Namespace: models.MembershipNamespace{Path: "other-group"}},
					},
					PageInfo: &pagination.PageInfo{TotalCount: 1},
				}, nil)
				m.caller.On("RequireRole", mock.Anything, models.OwnerRoleID.String(), mock.Anything).
					Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "enable client credentials",
			input: &UpdateServiceAccountInput{
				ID:                      serviceAccountID,
				EnableClientCredentials: ptr.Bool(true),
				ClientSecretExpiresAt:   &clientSecretExpiry,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)

				// Can't match exact model since client secret hash is generated
				m.serviceAccounts.On("UpdateServiceAccount", mock.Anything, mock.Anything).Return(&models.ServiceAccount{
					Metadata:              existingSA.Metadata,
					Name:                  existingSA.Name,
					GroupID:               existingSA.GroupID,
					Description:           existingSA.Description,
					OIDCTrustPolicies:     existingSA.OIDCTrustPolicies,
					ClientSecretHash:      ptr.String("hash"),
					ClientSecretExpiresAt: &clientSecretExpiry,
				}, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.activityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					NamespacePath: &groupPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetServiceAccount,
					TargetID:      serviceAccountID,
				}).Return(&models.ActivityEvent{}, nil)
			},
			expectClientCredentialsEnabled: true,
		},
		{
			name: "permission denied",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(existingSA, nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).
					Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "not found",
			input: &UpdateServiceAccountInput{
				ID:          serviceAccountID,
				Description: &updatedDescription,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(nil, nil)
			},
			expectErrCode: terrs.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:                  testLogger,
				dbClient:                m.dbClient(),
				activityService:         m.activityEvents,
				secretMaxExpirationDays: 90,
			}

			response, err := service.UpdateServiceAccount(auth.WithCaller(t.Context(), m.caller), test.input)

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

	genericErrPrefix := fmt.Sprintf(
		"failed to create service account token for service account %q due to one of the following reasons: "+
			"the service account does not exist; the JWT token used as input is invalid; "+
			"the issuer for the token is not a valid issuer; "+
			"the claims in the token do not satisfy the trust policy requirements",
		serviceAccountTRN,
	)

	// Test cases
	tests := []struct {
		expectErr      error
		verifyErr      error
		name           string
		serviceAccount string
		policy         []models.OIDCTrustPolicy
		token          []byte
		isTRN          bool
	}{
		{
			name:           "empty token",
			serviceAccount: serviceAccountID,
			token:          []byte(""),
			policy:         basicPolicy,
			expectErr:      errors.New("service account token is empty"),
		},
		{
			name:           "whitespace-only token",
			serviceAccount: serviceAccountID,
			token:          []byte("   \t\n  "),
			policy:         basicPolicy,
			expectErr:      errors.New("service account token is empty"),
		},
		{
			name:           "extremely long invalid token",
			serviceAccount: serviceAccountTRN,
			token:          []byte(string(make([]byte, 10000)) + "invalidtoken"),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to create token for service account trn:service_account:groupA/serviceAccount1 - token is not a valid JWT"),
			isTRN:          true,
		},
		{
			name:           "subject claim doesn't match",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New(genericErrPrefix),
			isTRN:          true,
		},
		{
			name:           "no matching trust policy",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://notavalidissuer",
					BoundClaims: map[string]string{},
				},
			},
			expectErr: errors.New(genericErrPrefix),
			isTRN:     true,
		},
		{
			name:           "empty trust policy",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         []models.OIDCTrustPolicy{},
			expectErr:      errors.New(genericErrPrefix),
			isTRN:          true,
		},
		{
			name:           "invalid token",
			serviceAccount: serviceAccountTRN,
			token:          []byte("invalidtoken"),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to create token for service account trn:service_account:groupA/serviceAccount1 - token is not a valid JWT"),
			isTRN:          true,
		},
		{
			name:           "missing issuer",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, "", sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to create token for service account trn:service_account:groupA/serviceAccount1 - issuer claim in token is empty"),
			isTRN:          true,
		},
		{
			name:           "empty service account ID",
			serviceAccount: "",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("service account ID is empty"),
		},
		{
			name:           "whitespace-only service account path",
			serviceAccount: "   ",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("service account ID is empty"),
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
			serviceAccount: serviceAccountTRN,
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
			expectErr: errors.New(genericErrPrefix),
			isTRN:     true,
		},
		{
			name:           "positive: multiple trust policies with same issuer: match first",
			serviceAccount: serviceAccountTRN,
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
			isTRN: true,
		},
		{
			name:           "positive: multiple trust policies with same issuer: match second",
			serviceAccount: serviceAccountTRN,
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
			isTRN: true,
		},
		{
			name:           "positive: trust policy issuer has forward slash, token issuer does not",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis/",
					BoundClaims: map[string]string{},
				},
			},
			isTRN: true,
		},
		{
			name:           "positive: token issuer has forward slash, trust policy issuer does not",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis/", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis",
					BoundClaims: map[string]string{},
				},
			},
			isTRN: true,
		},
		{
			name:           "negative: expired token",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(-time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to create token for service account trn:service_account:groupA/serviceAccount1 - token is expired"),
			isTRN:          true,
		},
		{
			name:           "negative: invalid signature",
			serviceAccount: serviceAccountTRN,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			verifyErr:      terrs.New(failedToVerifyJWSSignature, terrs.WithErrorCode(terrs.EUnauthorized)),
			expectErr:      errors.New(genericErrPrefix),
			isTRN:          true,
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
					mockServiceAccounts.On("GetServiceAccountByTRN", mock.Anything, test.serviceAccount).Return(&sa, nil).Maybe()
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
					if test.verifyErr != nil {
						return test.verifyErr
					}

					// Validate JWT claims (expiration, etc.)
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
				assert.Contains(t, err.Error(), test.expectErr.Error())
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
	groupID := "group-123"
	resourcePath := groupID + "/test-sa"

	saWithClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:      serviceAccountID,
			TRN:     types.ServiceAccountModelType.BuildTRN(resourcePath),
			Version: 1,
		},
		Name:             "test-sa",
		GroupID:          groupID,
		ClientSecretHash: ptr.String("existing-hash"),
	}

	saWithoutClientCreds := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:  serviceAccountID,
			TRN: types.ServiceAccountModelType.BuildTRN(resourcePath),
		},
		Name:    "test-sa-no-client-creds",
		GroupID: groupID,
	}

	tests := []struct {
		name          string
		input         *ResetClientCredentialsInput
		setupMocks    func(*serviceAccountMocks)
		expectErrCode terrs.CodeType
	}{
		{
			name: "successful reset",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(saWithClientCreds, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)
				m.serviceAccounts.On("UpdateServiceAccount", mock.Anything, mock.Anything).Return(saWithClientCreds, nil)
			},
		},
		{
			name: "service account not found",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(nil, nil)
			},
			expectErrCode: terrs.ENotFound,
		},
		{
			name: "permission denied",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(saWithClientCreds, nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).
					Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "non-owner cannot reset with memberships",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(saWithClientCreds, nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: []models.NamespaceMembership{
						{Namespace: models.MembershipNamespace{Path: "other-group"}},
					},
					PageInfo: &pagination.PageInfo{TotalCount: 1},
				}, nil)
				m.caller.On("RequireRole", mock.Anything, models.OwnerRoleID.String(), mock.Anything).
					Return(terrs.New("forbidden", terrs.WithErrorCode(terrs.EForbidden)))
			},
			expectErrCode: terrs.EForbidden,
		},
		{
			name: "client credentials not enabled",
			input: &ResetClientCredentialsInput{
				ID: serviceAccountID,
			},
			setupMocks: func(m *serviceAccountMocks) {
				m.serviceAccounts.On("GetServiceAccountByID", mock.Anything, serviceAccountID).Return(saWithoutClientCreds, nil)
				m.caller.On("RequirePermission", mock.Anything, models.UpdateServiceAccountPermission, mock.Anything).Return(nil)
				m.caller.On("IsAdmin").Return(false)
				m.namespaceMemberships.On("GetNamespaceMemberships", mock.Anything, &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						ServiceAccountID: &serviceAccountID,
					},
				}).Return(&db.NamespaceMembershipResult{
					PageInfo: &pagination.PageInfo{TotalCount: 0},
				}, nil)
			},
			expectErrCode: terrs.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newServiceAccountMocks(t)
			test.setupMocks(m)

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:                  testLogger,
				dbClient:                m.dbClient(),
				secretMaxExpirationDays: 90,
			}

			response, err := service.ResetClientCredentials(auth.WithCaller(t.Context(), m.caller), test.input)

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
