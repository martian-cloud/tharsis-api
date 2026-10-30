package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth/credentialhelper"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth/file"
)

// envMap turns a map into the getenv function newOIDCTokenSource expects, so tests never touch
// the real process environment.
func envMap(vars map[string]string) func(string) string {
	return func(key string) string {
		return vars[key]
	}
}

// writeHelper writes an executable stand-in for the credential helper, reporting its own
// arguments through the token field so what it was given is assertable.
func writeHelper(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "helper.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700)) // nosemgrep: gosec.G306-1

	return path
}

func TestNewOIDCTokenSource(t *testing.T) {
	// A real file to point the path at, so GetToken does not depend on the test environment.
	kubeSATokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(kubeSATokenPath, []byte("kube-sa-token"), 0o600))

	// Reports its arguments, so the credential helper cases can assert what reached the command.
	helperPath := writeHelper(t, `printf '{"token":"helper:%s"}' "$*"`)

	tests := []struct {
		name string
		env  map[string]string

		expectErrContains string

		// Which source is expected. Exactly one of these should be true.
		expectFile   bool
		expectHelper bool

		// What the source must reveal when asked for a token. The sources keep their
		// configuration private, matching the Kubernetes configurer implementations, so what
		// they were configured with is asserted through behaviour rather than field access.
		expectToken       string
		expectGetTokenErr string
	}{
		{
			// Also the regression this whole change is most likely to break: with no credential
			// helper set, the legacy requirement must not fire. It only does not because that
			// check lives inside the `case "":` branch of newOIDCTokenSource.
			name: "kubernetes_service_account_token method reads the configured token path",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                           methodKubernetesServiceAccountToken,
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": kubeSATokenPath,
			},
			expectFile:  true,
			expectToken: "kube-sa-token",
		},
		{
			// Phobos settled on ignoring the other methods' settings rather than rejecting them,
			// so a deployment can migrate between methods without a flag day.
			name: "kubernetes_service_account_token method ignores credential helper settings if also set",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                           methodKubernetesServiceAccountToken,
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": kubeSATokenPath,
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH":            helperPath,
				"THARSIS_CREDENTIAL_HELPER_CMD_ARGS":            "-region=us-east-2",
			},
			expectFile:  true,
			expectToken: "kube-sa-token",
		},
		{
			name: "kubernetes_service_account_token method without a token path is rejected",
			env: map[string]string{
				"THARSIS_AUTH_METHOD": methodKubernetesServiceAccountToken,
			},
			expectErrContains: "THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH is required when THARSIS_AUTH_METHOD is \"kubernetes_service_account_token\"",
		},
		{
			name: "kubernetes_service_account_token method names the file it could not read",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                           methodKubernetesServiceAccountToken,
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": filepath.Join(t.TempDir(), "absent"),
			},
			expectFile:        true,
			expectGetTokenErr: "failed to read OIDC token file",
		},
		{
			name: "credential_helper method uses the helper path and args",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                methodCredentialHelper,
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH": helperPath,
				"THARSIS_CREDENTIAL_HELPER_CMD_ARGS": "-region=us-east-2 -aud=tharsis",
			},
			expectHelper: true,
			expectToken:  "helper:-region=us-east-2 -aud=tharsis",
		},
		{
			// The unconditional split yields one empty argument rather than none. Preserved
			// behaviour, pinned here so a future tidy-up does not change every command line.
			name: "credential_helper with no args is accepted",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                methodCredentialHelper,
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH": helperPath,
			},
			expectHelper: true,
			expectToken:  "helper:",
		},
		{
			name: "credential_helper method ignores a token path if also set",
			env: map[string]string{
				"THARSIS_AUTH_METHOD":                           methodCredentialHelper,
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH":            helperPath,
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": kubeSATokenPath,
			},
			expectHelper: true,
			expectToken:  "helper:",
		},
		{
			name: "credential_helper method without a path is rejected",
			env: map[string]string{
				"THARSIS_AUTH_METHOD": methodCredentialHelper,
			},
			expectErrContains: "THARSIS_CREDENTIAL_HELPER_CMD_PATH is required when THARSIS_AUTH_METHOD is \"credential_helper\"",
		},
		{
			// Also covers an explicitly empty THARSIS_AUTH_METHOD, which a chart that templates the
			// variable unconditionally produces: getenv cannot distinguish an absent key from an
			// empty value, so a separate case for it would add no coverage. The behaviour comes from
			// the `case "":` branch of newOIDCTokenSource; if that became `default:` this case would
			// fail with "unsupported THARSIS_AUTH_METHOD".
			name: "legacy mode with a credential helper, or an explicitly empty method",
			env: map[string]string{
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH": helperPath,
			},
			expectHelper: true,
			expectToken:  "helper:",
		},
		{
			// The wording is the one an existing deployment already logs, so it is asserted in
			// full rather than by substring.
			name:              "legacy mode without a credential helper is rejected in the pre-existing wording",
			env:               map[string]string{},
			expectErrContains: "THARSIS_CREDENTIAL_HELPER_CMD_PATH environment variable is required",
		},
		{
			name: "legacy mode ignores a token path if set without a method",
			env: map[string]string{
				"THARSIS_CREDENTIAL_HELPER_CMD_PATH":            helperPath,
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": kubeSATokenPath,
			},
			expectHelper: true,
			expectToken:  "helper:",
		},
		{
			// A plausible mistake: the token path set but THARSIS_AUTH_METHOD forgotten. The
			// runner agent fails at startup, but the message names the credential helper
			// because the legacy branch is what an absent method selects.
			name: "legacy mode with only a token path is rejected, naming the helper",
			env: map[string]string{
				"THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH": kubeSATokenPath,
			},
			expectErrContains: "THARSIS_CREDENTIAL_HELPER_CMD_PATH environment variable is required",
		},
		{
			name: "unknown method is rejected and lists the valid values",
			env: map[string]string{
				"THARSIS_AUTH_METHOD": "kubernetes",
			},
			expectErrContains: "unsupported THARSIS_AUTH_METHOD \"kubernetes\", valid values are \"kubernetes_service_account_token\" and \"credential_helper\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, err := newOIDCTokenSource(envMap(test.env))

			if test.expectErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErrContains)
				require.Nil(t, source)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, source)

			// The concrete type is the source selection. A single interface value makes two
			// simultaneous sources unrepresentable.
			switch {
			case test.expectFile:
				require.IsType(t, &file.OIDCTokenSource{}, source)
			case test.expectHelper:
				require.IsType(t, &credentialhelper.OIDCTokenSource{}, source)
			default:
				t.Fatal("test case declares no expected source")
			}

			// What the source was configured with, observed through what it does.
			token, getErr := source.GetToken(t.Context())
			if test.expectGetTokenErr != "" {
				require.Error(t, getErr, "expected the source to fail and name what it tried to use")
				require.Contains(t, getErr.Error(), test.expectGetTokenErr)
				return
			}

			require.NoError(t, getErr)
			require.Equal(t, test.expectToken, token)
		})
	}
}

// The name is a configuration contract: renaming it would stop every deployment that sets it.
func TestEnvKubeSATokenPath(t *testing.T) {
	require.Equal(t, "THARSIS_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH", envKubeSATokenPath)
}

// The values are a configuration contract shared with the Helm chart and with Phobos, which uses
// the same vocabulary for the same two methods.
func TestMethodValues(t *testing.T) {
	require.Equal(t, "kubernetes_service_account_token", methodKubernetesServiceAccountToken)
	require.Equal(t, "credential_helper", methodCredentialHelper)
}
