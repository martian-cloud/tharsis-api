package apiserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
)

func TestOAuthProviderScopeWarnings(t *testing.T) {
	testCases := []struct {
		name           string
		providers      []config.IdpConfig
		expectContains []string
	}{
		{
			name:      "no oauth providers",
			providers: nil,
		},
		{
			name: "scope requests email and profile",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp.example.com", Scope: "openid profile email", UsernameClaim: "preferred_username", ExternalIDClaim: "oid"},
			},
		},
		{
			name: "missing email scope",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp.example.com", Scope: "openid profile", UsernameClaim: "sub"},
			},
			expectContains: []string{"does not request the email scope"},
		},
		{
			name: "empty scope",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp.example.com", Scope: "", UsernameClaim: "sub"},
			},
			expectContains: []string{"does not request the email scope"},
		},
		{
			name: "missing profile scope when the username and external id claims come from it",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp.example.com", Scope: "openid email", UsernameClaim: "preferred_username", ExternalIDClaim: "oid"},
			},
			expectContains: []string{
				`the "preferred_username" claim is used for usernames: new users can't be created`,
				`the "oid" claim is used to identify users: sign-in fails`,
			},
		},
		{
			name: "missing profile scope is fine when no profile claim is used",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp.example.com", Scope: "openid email", UsernameClaim: "email", ExternalIDClaim: "sub"},
			},
		},
		{
			name: "each provider is checked",
			providers: []config.IdpConfig{
				{IssuerURL: "https://idp-1.example.com", Scope: "openid profile email", UsernameClaim: "sub"},
				{IssuerURL: "https://idp-2.example.com", Scope: "openid", UsernameClaim: "sub"},
			},
			expectContains: []string{`"https://idp-2.example.com" does not request the email scope`},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			warnings := oauthProviderScopeWarnings(&config.Config{OauthProviders: test.providers})

			assert.Len(t, warnings, len(test.expectContains))
			for i, expected := range test.expectContains {
				if i < len(warnings) {
					assert.Contains(t, warnings[i], expected)
				}
			}
		})
	}
}
