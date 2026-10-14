package apiserver

import (
	"fmt"
	"slices"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
)

// profileScopeClaims are the claims an identity provider includes in the ID token for the profile scope
// (OIDC Core section 5.4), plus Entra ID's oid, which it only includes for the profile scope.
var profileScopeClaims = []string{
	"name", "family_name", "given_name", "middle_name", "nickname", "preferred_username", "profile",
	"picture", "website", "gender", "birthdate", "zoneinfo", "locale", "updated_at", "oid",
}

// oauthProviderScopeWarnings returns a warning for each oauth provider whose scope may not request the claims
// Tharsis needs. Some identity providers add these claims without the scope (e.g. through a claim mapper), so
// this only warns rather than failing startup or adding the scopes itself.
func oauthProviderScopeWarnings(cfg *config.Config) []string {
	warnings := []string{}

	for _, idp := range cfg.OauthProviders {
		scopes := strings.Fields(idp.Scope)

		if !slices.Contains(scopes, "email") {
			warnings = append(warnings, fmt.Sprintf("The scope %q for identity provider %q does not request the "+
				"email scope: new users can't be created, since their first sign-in needs the email claim, unless "+
				"the identity provider includes it without the scope.", idp.Scope, idp.IssuerURL))
		}

		if slices.Contains(scopes, "profile") {
			continue
		}
		if slices.Contains(profileScopeClaims, idp.UsernameClaim) {
			warnings = append(warnings, fmt.Sprintf("The scope %q for identity provider %q does not request "+
				"the profile scope, but the %q claim is used for usernames: new users can't be created unless the "+
				"identity provider includes the claim without the scope.", idp.Scope, idp.IssuerURL, idp.UsernameClaim))
		}
		if slices.Contains(profileScopeClaims, idp.ExternalIDClaim) {
			warnings = append(warnings, fmt.Sprintf("The scope %q for identity provider %q does not request "+
				"the profile scope, but the %q claim is used to identify users: sign-in fails unless the "+
				"identity provider includes the claim without the scope.", idp.Scope, idp.IssuerURL, idp.ExternalIDClaim))
		}
	}

	return warnings
}
