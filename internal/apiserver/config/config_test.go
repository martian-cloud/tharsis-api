package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOauthConfigFromEnvironment(t *testing.T) {
	t.Setenv(envOidcProviderConfigPrefix+"0_ISSUER_URL", "https://login.microsoftonline.com/tenant-id/v2.0")
	t.Setenv(envOidcProviderConfigPrefix+"0_CLIENT_ID", "tharsis")
	t.Setenv(envOidcProviderConfigPrefix+"0_SCOPE", "openid profile email")
	t.Setenv(envOidcProviderConfigPrefix+"0_EXTERNAL_ID_CLAIM", "oid")
	t.Setenv(envOidcProviderConfigPrefix+"0_EMAIL_LINKING", "true")
	t.Setenv(envOidcProviderConfigPrefix+"0_ALLOW_ADMIN_BOOTSTRAP", "true")

	idps, err := loadOauthConfigFromEnvironment()
	require.NoError(t, err)
	require.Len(t, idps, 1)

	assert.Equal(t, IdpConfig{
		IssuerURL:           "https://login.microsoftonline.com/tenant-id/v2.0",
		ClientID:            "tharsis",
		UsernameClaim:       "sub",
		Scope:               "openid profile email",
		ExternalIDClaim:     "oid",
		EmailLinking:        true,
		AllowAdminBootstrap: true,
	}, idps[0])
}

func TestLoadOauthConfigFromEnvironment_InvalidEmailLinking(t *testing.T) {
	t.Setenv(envOidcProviderConfigPrefix+"0_ISSUER_URL", "https://idp.example.com")
	t.Setenv(envOidcProviderConfigPrefix+"0_CLIENT_ID", "tharsis")
	t.Setenv(envOidcProviderConfigPrefix+"0_EMAIL_LINKING", "sometimes")

	_, err := loadOauthConfigFromEnvironment()
	assert.ErrorContains(t, err, envOidcProviderConfigPrefix+"0_EMAIL_LINKING")
}

func TestLoadOauthConfigFromEnvironment_InvalidAllowAdminBootstrap(t *testing.T) {
	t.Setenv(envOidcProviderConfigPrefix+"0_ISSUER_URL", "https://idp.example.com")
	t.Setenv(envOidcProviderConfigPrefix+"0_CLIENT_ID", "tharsis")
	t.Setenv(envOidcProviderConfigPrefix+"0_ALLOW_ADMIN_BOOTSTRAP", "sometimes")

	_, err := loadOauthConfigFromEnvironment()
	assert.ErrorContains(t, err, envOidcProviderConfigPrefix+"0_ALLOW_ADMIN_BOOTSTRAP")
}
