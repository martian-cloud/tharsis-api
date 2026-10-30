package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth/credentialhelper"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth/file"
)

// Supported values for the THARSIS_AUTH_METHOD environment variable.
//
// Each value names where the OIDC token comes from. That token is what the runner agent exchanges
// for a Tharsis service account token; it is not the service account token itself.
//
// When THARSIS_AUTH_METHOD is unset the runner agent stays in legacy mode, which requires
// THARSIS_CREDENTIAL_HELPER_CMD_PATH exactly as it did before this variable existed. Existing
// deployments therefore keep working untouched.
const (
	// methodKubernetesServiceAccountToken reads a projected Kubernetes service account token from
	// THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH. The kubelet rotates the file in place, so it
	// is re-read on every renewal.
	methodKubernetesServiceAccountToken = "kubernetes_service_account_token"

	// methodCredentialHelper runs an external command that prints an OIDC token as JSON. The
	// command runs on every renewal, so it can mint a fresh token each time.
	methodCredentialHelper = "credential_helper"
)

// envKubeSATokenPath names the file holding the token methodKubernetesServiceAccountToken reads.
// Required, with no default: the automounted token is issued for the cluster's API server, so
// defaulting to it would silently present the wrong audience to Tharsis. Deployments project a
// token with their own audience and name it here.
const envKubeSATokenPath = "THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH"

// newOIDCTokenSource selects a token source from the environment.
//
// Selection lives with the runner agent command rather than in the auth package because it is
// driven entirely by this binary's THARSIS_* environment contract, matching how the job dispatcher
// is selected by its consumer.
//
// getenv is injected rather than calling os.Getenv directly so that every branch can be unit
// tested without mutating the process environment.
//
// Settings that belong to a method other than the selected one are ignored rather than rejected.
// A deployment migrating between methods can leave the old variables in place, and the method
// named in THARSIS_AUTH_METHOD is the only thing that decides which credential is presented.
func newOIDCTokenSource(getenv func(string) string) (auth.OIDCTokenSource, error) {
	credHelperPath := getenv("THARSIS_CREDENTIAL_HELPER_CMD_PATH")
	kubeSATokenPath := getenv(envKubeSATokenPath)

	// Split unconditionally, matching the behaviour that predates THARSIS_AUTH_METHOD exactly.
	// Note that splitting an empty string yields a single empty argument rather than no arguments;
	// that quirk only reaches a credential helper that is actually configured.
	credHelperArgs := strings.Split(getenv("THARSIS_CREDENTIAL_HELPER_CMD_ARGS"), " ")

	switch method := getenv("THARSIS_AUTH_METHOD"); method {
	case methodKubernetesServiceAccountToken:
		if kubeSATokenPath == "" {
			return nil, fmt.Errorf("%s is required when THARSIS_AUTH_METHOD is %q", envKubeSATokenPath, methodKubernetesServiceAccountToken)
		}
		// Not at the getenv: Clean("") is ".".
		return file.New(filepath.Clean(kubeSATokenPath)), nil

	case methodCredentialHelper:
		if credHelperPath == "" {
			return nil, fmt.Errorf("THARSIS_CREDENTIAL_HELPER_CMD_PATH is required when THARSIS_AUTH_METHOD is %q", methodCredentialHelper)
		}
		return credentialhelper.New(credHelperPath, credHelperArgs), nil

	case "":
		// Legacy mode. This check used to run unconditionally, before THARSIS_AUTH_METHOD existed,
		// and its wording is preserved so that an operator's existing log search still matches. It
		// must stay confined to this branch: the kubernetes_service_account_token method sets no
		// credential helper, so hoisting it would make the runner agent exit before it ever read
		// the projected token.
		if credHelperPath == "" {
			return nil, fmt.Errorf("THARSIS_CREDENTIAL_HELPER_CMD_PATH environment variable is required")
		}
		return credentialhelper.New(credHelperPath, credHelperArgs), nil

	default:
		return nil, fmt.Errorf("unsupported THARSIS_AUTH_METHOD %q, valid values are %q and %q",
			method, methodKubernetesServiceAccountToken, methodCredentialHelper)
	}
}
