package auth

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewFederatedRegistryAuth(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockLogger, _ := logger.NewForTest()
	mockOIDCConfigFetcher := NewMockOpenIDConfigFetcher(t)
	mockDBClient := &db.Client{}

	trustPolicies := []config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
		{
			IssuerURL:         "https://issuer2.example.com",
			Subject:           ptr.String("subject2"),
			Audience:          ptr.String("audience2"),
			GroupGlobPatterns: []string{"group2/*"},
		},
	}

	// Execute
	federatedAuth := NewFederatedRegistryAuth(ctx, trustPolicies, mockLogger, mockOIDCConfigFetcher, mockDBClient)

	// Verify
	require.NotNil(t, federatedAuth)
	require.Equal(t, trustPolicies, federatedAuth.trustPolicies)
	require.Equal(t, mockDBClient, federatedAuth.dbClient)
	require.Equal(t, mockLogger, federatedAuth.logger)
	require.NotNil(t, federatedAuth.tokenVerifier)
}

func TestFederatedRegistryAuth_Use(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockLogger, _ := logger.NewForTest()
	mockOIDCConfigFetcher := NewMockOpenIDConfigFetcher(t)
	mockDBClient := &db.Client{}

	trustPolicies := []config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}

	federatedAuth := NewFederatedRegistryAuth(ctx, trustPolicies, mockLogger, mockOIDCConfigFetcher, mockDBClient)

	testCases := []struct {
		name           string
		tokenSetup     func() jwt.Token
		expectedResult bool
	}{
		{
			name: "valid federated registry token",
			tokenSetup: func() jwt.Token {
				mockToken := jwt.New()
				_ = mockToken.Set("typ", FederatedRegistryTokenType)
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				return mockToken
			},
			expectedResult: true,
		},
		{
			name: "token with incorrect type",
			tokenSetup: func() jwt.Token {
				mockToken := jwt.New()
				_ = mockToken.Set("typ", "incorrect-type")
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				return mockToken
			},
			expectedResult: false,
		},
		{
			name: "token with incorrect issuer",
			tokenSetup: func() jwt.Token {
				mockToken := jwt.New()
				_ = mockToken.Set("typ", FederatedRegistryTokenType)
				_ = mockToken.Set(jwt.IssuerKey, "https://incorrect-issuer.example.com")
				return mockToken
			},
			expectedResult: false,
		},
		{
			name: "token with missing type claim",
			tokenSetup: func() jwt.Token {
				mockToken := jwt.New()
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				return mockToken
			},
			expectedResult: false,
		},
		{
			name: "token with non-string type claim",
			tokenSetup: func() jwt.Token {
				mockToken := jwt.New()
				_ = mockToken.Set("typ", 123)
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				return mockToken
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token := tc.tokenSetup()
			result := federatedAuth.Use(token)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestFederatedRegistryAuth_Authenticate(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockLogger, _ := logger.NewForTest()
	mockDBClient := &db.Client{}
	mockTokenVerifier := NewMockOIDCTokenVerifier(t)

	trustPolicies := []config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
		{
			IssuerURL:         "https://issuer2.example.com",
			Subject:           nil,
			Audience:          nil,
			GroupGlobPatterns: []string{"group2/*"},
		},
	}

	federatedAuth := &FederatedRegistryAuth{
		trustPolicies: trustPolicies,
		tokenVerifier: mockTokenVerifier,
		dbClient:      mockDBClient,
		logger:        mockLogger,
	}

	testCases := []struct {
		name               string
		token              string
		setupMocks         func(mockTokenVerifier *MockOIDCTokenVerifier, token string)
		expectErrorMessage string
	}{
		{
			name:  "successful authentication with matching policy",
			token: "valid-token",
			setupMocks: func(mockTokenVerifier *MockOIDCTokenVerifier, token string) {
				mockToken := jwt.New()
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				_ = mockToken.Set(jwt.SubjectKey, "subject1")
				_ = mockToken.Set(jwt.AudienceKey, []string{"audience1"})

				mockTokenVerifier.On("VerifyToken", mock.Anything, token, mock.Anything).
					Return(mockToken, nil)
			},
		},
		{
			name:  "successful authentication with policy without subject and audience",
			token: "valid-token-no-constraints",
			setupMocks: func(mockTokenVerifier *MockOIDCTokenVerifier, token string) {
				mockToken := jwt.New()
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer2.example.com")
				_ = mockToken.Set(jwt.SubjectKey, "any-subject")
				_ = mockToken.Set(jwt.AudienceKey, []string{"any-audience"})

				mockTokenVerifier.On("VerifyToken", mock.Anything, token, mock.Anything).
					Return(mockToken, nil)
			},
		},
		{
			name:  "token verification fails",
			token: "invalid-token",
			setupMocks: func(mockTokenVerifier *MockOIDCTokenVerifier, token string) {
				mockTokenVerifier.On("VerifyToken", mock.Anything, token, mock.Anything).
					Return(nil, errors.New("token is unauthorized"))
			},
			expectErrorMessage: "token is unauthorized",
		},
		{
			name:  "no matching trust policy",
			token: "no-matching-policy-token",
			setupMocks: func(mockTokenVerifier *MockOIDCTokenVerifier, token string) {
				mockToken := jwt.New()
				_ = mockToken.Set(jwt.IssuerKey, "https://issuer1.example.com")
				_ = mockToken.Set(jwt.SubjectKey, "wrong-subject")
				_ = mockToken.Set(jwt.AudienceKey, []string{"audience1"})

				mockTokenVerifier.On("VerifyToken", mock.Anything, token, mock.Anything).
					Return(mockToken, nil)
			},
			expectErrorMessage: "no federated trust policies match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks(mockTokenVerifier, tc.token)

			caller, err := federatedAuth.Authenticate(ctx, tc.token, false)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErrorMessage)
				require.Nil(t, caller)
			} else {
				require.NoError(t, err)
				require.NotNil(t, caller)
			}
		})
	}
}

func TestFederatedRegistryCaller_GetSubject(t *testing.T) {
	// Test setup
	mockDBClient := &db.Client{}
	trustPolicies := []*config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

	// Execute
	result := caller.GetSubject()

	// Verify
	require.Equal(t, subject, result)
}

func TestFederatedRegistryCaller_IsAdmin(t *testing.T) {
	// Test setup
	mockDBClient := &db.Client{}
	trustPolicies := []*config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

	// Execute
	result := caller.IsAdmin()

	// Verify
	require.False(t, result)
}

func TestFederatedRegistryCaller_UnauthorizedError(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockDBClient := &db.Client{}
	trustPolicies := []*config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

	testCases := []struct {
		name            string
		hasViewerAccess bool
		expectedCode    errors.CodeType
	}{
		{
			name:            "with viewer access",
			hasViewerAccess: true,
			expectedCode:    errors.EForbidden,
		},
		{
			name:            "without viewer access",
			hasViewerAccess: false,
			expectedCode:    errors.ENotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute
			err := caller.UnauthorizedError(ctx, tc.hasViewerAccess)

			// Verify
			require.Error(t, err)
			require.Equal(t, tc.expectedCode, errors.ErrorCode(err))
		})
	}
}

func TestFederatedRegistryCaller_GetNamespaceAccessPolicy(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockDBClient := &db.Client{}
	trustPolicies := []*config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

	// Execute
	policy, err := caller.GetNamespaceAccessPolicy(ctx)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, policy)
	require.Empty(t, policy.RootNamespaceIDs)
}

func TestFederatedRegistryCaller_RequirePermission(t *testing.T) {
	// Test setup
	ctx := context.Background()
	mockDBClient := &db.Client{}
	trustPolicies := []*config.FederatedRegistryTrustPolicy{
		{
			IssuerURL:         "https://issuer1.example.com",
			Subject:           ptr.String("subject1"),
			Audience:          ptr.String("audience1"),
			GroupGlobPatterns: []string{"group1/*"},
		},
	}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

	// Execute
	err := caller.RequirePermission(ctx, permissions.ViewTerraformModulePermission)

	// Verify
	require.Error(t, err)
	require.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestFederatedRegistryCaller_RequireAccessToInheritableResource(t *testing.T) {
	// Test setup
	ctx := context.Background()

	testCases := []struct {
		name               string
		resourceType       permissions.ResourceType
		constraints        []func(*constraints)
		setupMocks         func(mockGroups *db.MockGroups)
		expectErrorMessage string
	}{
		{
			name:         "successful access with matching group pattern",
			resourceType: permissions.TerraformModuleResourceType,
			constraints: []func(*constraints){
				WithGroupID("group-id"),
			},
			setupMocks: func(mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-id").
					Return(&models.Group{FullPath: "group1/nested1"}, nil)
			},
		},
		{
			name:         "successful access with namespace paths",
			resourceType: permissions.TerraformProviderResourceType,
			constraints: []func(*constraints){
				WithNamespacePaths([]string{"group1"}),
			},
		},
		{
			name:               "unsupported resource type",
			resourceType:       permissions.WorkspaceResourceType,
			constraints:        []func(*constraints){},
			expectErrorMessage: "unsupported resource type",
		},
		{
			name:         "no matching group pattern",
			resourceType: permissions.TerraformModuleResourceType,
			constraints: []func(*constraints){
				WithGroupID("group-id"),
			},
			setupMocks: func(mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-id").
					Return(&models.Group{FullPath: "non-matching"}, nil)
			},
			expectErrorMessage: "not authorized",
		},
		{
			name:               "missing constraints",
			resourceType:       permissions.TerraformModuleResourceType,
			constraints:        []func(*constraints){},
			expectErrorMessage: "missing required permissions or constraints",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDBClient := &db.Client{}
			mockGroups := db.NewMockGroups(t)
			mockDBClient.Groups = mockGroups

			trustPolicies := []*config.FederatedRegistryTrustPolicy{
				{
					IssuerURL:         "https://issuer1.example.com",
					Subject:           ptr.String("subject1"),
					Audience:          ptr.String("audience1"),
					GroupGlobPatterns: []string{},
				},
				{
					IssuerURL:         "https://issuer1.example.com",
					Subject:           ptr.String("subject1"),
					Audience:          ptr.String("audience1"),
					GroupGlobPatterns: []string{"group1", "group1/*", "group2/*"},
				},
			}
			subject := "test-subject"

			if tc.setupMocks != nil {
				tc.setupMocks(mockGroups)
			}

			caller := NewFederatedRegistryCaller(mockDBClient, trustPolicies, subject)

			// Execute
			err := caller.RequireAccessToInheritableResource(ctx, tc.resourceType, tc.constraints...)

			// Verify
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFederatedRegistryCaller_trustPolicySatisfied(t *testing.T) {
	// Test setup
	mockDBClient := &db.Client{}
	subject := "test-subject"

	caller := NewFederatedRegistryCaller(mockDBClient, []*config.FederatedRegistryTrustPolicy{}, subject)

	testCases := []struct {
		name                    string
		requestedNamespacePaths []string
		groupGlobPatterns       []string
		expectedResult          bool
	}{
		{
			name:                    "all paths match patterns",
			requestedNamespacePaths: []string{"group1/module", "test-group/provider"},
			groupGlobPatterns:       []string{"group1/*", "test-*/*"},
			expectedResult:          true,
		},
		{
			name:                    "exact match without wildcard",
			requestedNamespacePaths: []string{"group1"},
			groupGlobPatterns:       []string{"group1", "group2"},
			expectedResult:          true,
		},
		{
			name:                    "don't allow access to nested group if pattern doesn't match",
			requestedNamespacePaths: []string{"group1/nested1/nested2"},
			groupGlobPatterns:       []string{"group1/nested1", "group1/nested1/nested2/*"},
			expectedResult:          false,
		},
		{
			name:                    "one path doesn't match any pattern",
			requestedNamespacePaths: []string{"group1/module", "other-group/provider"},
			groupGlobPatterns:       []string{"group1/*", "test-*/*"},
			expectedResult:          false,
		},
		{
			name:                    "allow all groups",
			requestedNamespacePaths: []string{"group1/module", "other-group"},
			groupGlobPatterns:       []string{"*"},
			expectedResult:          true,
		},
		{
			name:                    "don't allow any groups",
			requestedNamespacePaths: []string{"group1"},
			groupGlobPatterns:       []string{},
			expectedResult:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute
			result := caller.trustPolicySatisfied(tc.requestedNamespacePaths, tc.groupGlobPatterns)

			// Verify
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestFederatedRegistryCaller_getRequestedNamespacePaths(t *testing.T) {
	// Test setup
	ctx := context.Background()

	subject := "test-subject"

	testCases := []struct {
		name               string
		constraints        *constraints
		setupMocks         func(mockGroups *db.MockGroups)
		expectedPaths      []string
		expectErrorMessage string
	}{
		{
			name: "get paths from group ID",
			constraints: &constraints{
				groupID: ptr.String("group-id"),
			},
			setupMocks: func(mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-id").
					Return(&models.Group{FullPath: "test-group/module"}, nil)
			},
			expectedPaths: []string{"test-group/module"},
		},
		{
			name: "get paths from namespace paths",
			constraints: &constraints{
				namespacePaths: []string{"path1", "path2"},
			},
			setupMocks:    func(_ *db.MockGroups) {},
			expectedPaths: []string{"path1", "path2"},
		},
		{
			name: "get paths from both group ID and namespace paths",
			constraints: &constraints{
				groupID:        ptr.String("group-id"),
				namespacePaths: []string{"path1", "path2"},
			},
			setupMocks: func(mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-id").
					Return(&models.Group{FullPath: "test-group/module"}, nil)
			},
			expectedPaths: []string{"test-group/module", "path1", "path2"},
		},
		{
			name:               "missing constraints",
			constraints:        &constraints{},
			setupMocks:         func(_ *db.MockGroups) {},
			expectErrorMessage: "missing required permissions or constraints",
		},
		{
			name: "group not found",
			constraints: &constraints{
				groupID: ptr.String("group-id"),
			},
			setupMocks: func(mockGroups *db.MockGroups) {
				mockGroups.On("GetGroupByID", mock.Anything, "group-id").
					Return(nil, nil)
			},
			expectErrorMessage: "missing required permissions or constraints",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDBClient := &db.Client{}
			mockGroups := db.NewMockGroups(t)
			mockDBClient.Groups = mockGroups

			tc.setupMocks(mockGroups)

			caller := NewFederatedRegistryCaller(mockDBClient, []*config.FederatedRegistryTrustPolicy{}, subject)

			// Execute
			paths, err := caller.getRequestedNamespacePaths(ctx, tc.constraints)

			// Verify
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPaths, paths)
			}
		})
	}
}
