package resolver

import (
	"context"
)

/* AuthSettings Query Resolvers */

// AuthSettingsResolver resolves auth settings
type AuthSettingsResolver struct {
	OIDCIssuerURL     *string
	OIDCClientID      *string
	OIDCUsernameClaim *string
	OIDCScope         *string
	OIDCLogoutURL     *string
}

func authSettingsQuery(ctx context.Context) *AuthSettingsResolver {
	cfg := getConfig(ctx)

	if len(cfg.OauthProviders) > 0 {
		provider := cfg.OauthProviders[0]
		return &AuthSettingsResolver{
			OIDCIssuerURL:     &provider.IssuerURL,
			OIDCClientID:      &provider.ClientID,
			OIDCUsernameClaim: &provider.UsernameClaim,
			OIDCScope:         &provider.Scope,
			OIDCLogoutURL:     &provider.LogoutURL,
		}
	}

	return &AuthSettingsResolver{}
}
