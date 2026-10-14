package apiserver

import (
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
)

// defaultAdminUserPassword is the well-known admin password shipped in env.example and docker-compose.
const defaultAdminUserPassword = "martian"

// insecureConfigWarnings returns a warning for each config setting that is fine for local development
// but unsafe in production.
func insecureConfigWarnings(cfg *config.Config) []string {
	warnings := []string{}

	if cfg.SecretManagerPluginType == "" && !cfg.DisableSensitiveVariableFeature {
		warnings = append(warnings, "No secret manager plugin is configured: sensitive variables will be stored "+
			"in plain text in the database. Set THARSIS_SECRET_MANAGER_PLUGIN_TYPE for production deployments.")
	}

	if cfg.JWSProviderPluginType == "memory" {
		warnings = append(warnings, "The \"memory\" JWS provider plugin is configured: token signing private keys "+
			"are stored in the database, so a database leak allows forging any token. Use a KMS-backed JWS "+
			"provider plugin for production deployments.")
	}

	if cfg.AdminUserEmail != "" && cfg.AdminUserPassword != "" {
		if cfg.AdminUserPassword == defaultAdminUserPassword {
			warnings = append(warnings, fmt.Sprintf("The admin user %q is configured with the well-known default "+
				"password. Change THARSIS_ADMIN_USER_PASSWORD for production deployments.", cfg.AdminUserEmail))
		}
		warnings = append(warnings, fmt.Sprintf("THARSIS_ADMIN_USER_PASSWORD is set: the password for admin user %q "+
			"will be reset to this value on every restart, reverting any change made through the UI. Unset it "+
			"once the admin user has been created.", cfg.AdminUserEmail))
	}

	if len(cfg.OauthProviders) > 0 && !strings.HasPrefix(cfg.TharsisAPIURL, "https://") {
		warnings = append(warnings, "THARSIS_API_URL does not use https: session and CLI login cookies are not "+
			"marked Secure or host-only, so a network attacker or another subdomain can plant a login cookie and "+
			"sign a browser in to the attacker's account. Serve the API over https for production deployments.")
	}

	for _, idp := range cfg.OauthProviders {
		if idp.EmailLinking {
			warnings = append(warnings, fmt.Sprintf("Email linking is enabled for identity provider %q: a first sign-in "+
				"can be linked to an existing user with no external identity based only on the email the provider "+
				"asserts. Disable it once existing users have been migrated to SSO.", idp.IssuerURL))
		}
	}

	for _, r := range cfg.InternalRunners {
		if r.JobDispatcherType == "local" {
			warnings = append(warnings, fmt.Sprintf("Internal runner %q uses the \"local\" job dispatcher: jobs run "+
				"inside the API process with access to its host and credentials. Use an isolated job dispatcher "+
				"(e.g. docker, kubernetes, ecs) for production deployments.", r.Name))
		}
	}

	return warnings
}
