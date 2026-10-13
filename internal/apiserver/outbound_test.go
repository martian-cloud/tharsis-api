package apiserver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
)

func TestTrustedOutboundURLs(t *testing.T) {
	cfg := &config.Config{
		TharsisAPIURL:        "https://tharsis.example.com",
		ServiceDiscoveryHost: "tharsis.example.com",
		JWTIssuerURL:         "https://tharsis.example.com",
		OauthProviders: []config.IdpConfig{
			{IssuerURL: "https://login.internal.example.com/realms/tharsis"},
		},
		FederatedRegistryTrustPolicies: []config.FederatedRegistryTrustPolicy{
			{IssuerURL: "https://other-tharsis.internal.example.com"},
		},
	}

	// Admin-configured issuers may be on a private network, so they are trusted like the API's own URLs.
	assert.ElementsMatch(t, []string{
		"https://tharsis.example.com",
		"tharsis.example.com",
		"https://tharsis.example.com",
		"https://login.internal.example.com/realms/tharsis",
		"https://other-tharsis.internal.example.com",
	}, trustedOutboundURLs(cfg))
}
