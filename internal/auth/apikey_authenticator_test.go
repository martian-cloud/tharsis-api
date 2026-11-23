package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/apikey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestAPIKeyAuthenticator_Authenticate(t *testing.T) {
	scimKey, scimHash, err := apikey.Generate(apikey.TypeSCIM)
	require.NoError(t, err)

	vcsKey, vcsHash, err := apikey.Generate(apikey.TypeVCS)
	require.NoError(t, err)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name            string
		key             string
		setupMocks      func(*db.MockSCIMTokens, *db.MockWorkspaceVCSProviderLinks, *db.MockVCSProviders)
		expectErrorCode errors.CodeType
		expectCaller    any
	}{
		{
			name:            "malformed key",
			key:             scimKey[:len(scimKey)-1] + "!",
			setupMocks:      func(*db.MockSCIMTokens, *db.MockWorkspaceVCSProviderLinks, *db.MockVCSProviders) {},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "scim token with no expiry",
			key:  scimKey,
			setupMocks: func(tokens *db.MockSCIMTokens, _ *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				tokens.On("GetTokenByHash", t.Context(), scimHash).Return(&models.SCIMToken{IDPIssuerURL: "https://idp.example.com"}, nil)
			},
			expectCaller: &SCIMCaller{},
		},
		{
			name: "scim token not expired",
			key:  scimKey,
			setupMocks: func(tokens *db.MockSCIMTokens, _ *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				tokens.On("GetTokenByHash", t.Context(), scimHash).Return(&models.SCIMToken{IDPIssuerURL: "https://idp.example.com", ExpiresAt: &future}, nil)
			},
			expectCaller: &SCIMCaller{},
		},
		{
			name: "scim token expired",
			key:  scimKey,
			setupMocks: func(tokens *db.MockSCIMTokens, _ *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				tokens.On("GetTokenByHash", t.Context(), scimHash).Return(&models.SCIMToken{IDPIssuerURL: "https://idp.example.com", ExpiresAt: &past}, nil)
			},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "scim token unknown",
			key:  scimKey,
			setupMocks: func(tokens *db.MockSCIMTokens, _ *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				tokens.On("GetTokenByHash", t.Context(), scimHash).Return(nil, nil)
			},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "scim token lookup fails",
			key:  scimKey,
			setupMocks: func(tokens *db.MockSCIMTokens, _ *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				tokens.On("GetTokenByHash", t.Context(), scimHash).Return(nil, errors.New("db error"))
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "vcs token valid",
			key:  vcsKey,
			setupMocks: func(_ *db.MockSCIMTokens, links *db.MockWorkspaceVCSProviderLinks, providers *db.MockVCSProviders) {
				links.On("GetLinkByTokenHash", t.Context(), vcsHash).Return(&models.WorkspaceVCSProviderLink{ProviderID: "provider-id"}, nil)
				providers.On("GetProviderByID", t.Context(), "provider-id").Return(&models.VCSProvider{}, nil)
			},
			expectCaller: &VCSWorkspaceLinkCaller{},
		},
		{
			name: "vcs token unknown",
			key:  vcsKey,
			setupMocks: func(_ *db.MockSCIMTokens, links *db.MockWorkspaceVCSProviderLinks, _ *db.MockVCSProviders) {
				links.On("GetLinkByTokenHash", t.Context(), vcsHash).Return(nil, nil)
			},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "vcs token whose provider is missing",
			key:  vcsKey,
			setupMocks: func(_ *db.MockSCIMTokens, links *db.MockWorkspaceVCSProviderLinks, providers *db.MockVCSProviders) {
				links.On("GetLinkByTokenHash", t.Context(), vcsHash).Return(&models.WorkspaceVCSProviderLink{ProviderID: "provider-id"}, nil)
				providers.On("GetProviderByID", t.Context(), "provider-id").Return(nil, nil)
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockTokens := db.NewMockSCIMTokens(t)
			mockLinks := db.NewMockWorkspaceVCSProviderLinks(t)
			mockProviders := db.NewMockVCSProviders(t)
			test.setupMocks(mockTokens, mockLinks, mockProviders)

			authenticator := newAuthenticator(&apiKeyAuthenticator{
				dbClient: &db.Client{
					SCIMTokens:                mockTokens,
					WorkspaceVCSProviderLinks: mockLinks,
					VCSProviders:              mockProviders,
				},
				maintenanceMonitor: maintenance.NewMockMonitor(t),
			}, nil)

			caller, err := authenticator.Authenticate(t.Context(), test.key, false)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, caller)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, test.expectCaller, caller)
		})
	}
}
