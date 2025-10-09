package resolver

import (
	"context"
)

// UserAuthType represents the type of user authentication
type UserAuthType string

// UserAuthType enum values
const (
	UserAuthTypeBasic UserAuthType = "BASIC"
	UserAuthTypeOIDC  UserAuthType = "OIDC"
)

/* AuthSettings Query Resolvers */

// OIDCAuthSettingsResolver resolves OIDC auth settings
type OIDCAuthSettingsResolver struct {
	IssuerURL string
	ClientID  string
	Scope     string
}

// AuthSettingsResolver resolves auth settings
type AuthSettingsResolver struct {
	AuthType UserAuthType
	OIDC     *OIDCAuthSettingsResolver
}

func authSettingsQuery(ctx context.Context) *AuthSettingsResolver {
	cfg := getConfig(ctx)

	if len(cfg.OauthProviders) > 0 {
		provider := cfg.OauthProviders[0]
		return &AuthSettingsResolver{
			AuthType: UserAuthTypeOIDC,
			OIDC: &OIDCAuthSettingsResolver{
				IssuerURL: provider.IssuerURL,
				ClientID:  provider.ClientID,
				Scope:     provider.Scope,
			},
		}
	}

	return &AuthSettingsResolver{
		AuthType: UserAuthTypeBasic,
	}
}
