package credentialhelper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeHelper(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "helper.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700)) // nosemgrep: gosec.G306-1

	return path
}

func TestGetToken(t *testing.T) {
	tests := []struct {
		name              string
		body              string
		args              []string
		expectToken       string
		expectErrContains string
	}{
		{
			name:        "a token on stdout",
			body:        `printf '{"token":"a.b.c"}'`,
			expectToken: "a.b.c",
		},
		{
			name:        "surrounding JSON whitespace is tolerated",
			body:        `printf '  {"token":"a.b.c"}  \n'`,
			expectToken: "a.b.c",
		},
		{
			name:        "arguments reach the command in order",
			body:        `printf '{"token":"%s"}' "$*"`,
			args:        []string{"--one", "--two"},
			expectToken: "--one --two",
		},
		{
			// The unconditional split in the caller produces this, so the command must tolerate
			// a single empty argument.
			name:        "a single empty argument is passed through",
			body:        `printf '{"token":"count=%s"}' "$#"`,
			args:        []string{""},
			expectToken: "count=1",
		},
		{
			// The prefix is the one the caller applied before this moved into a package, asserted
			// in full so an existing deployment matching on the message still matches.
			name:              "a non-zero exit surfaces stderr, with the pre-existing prefix",
			body:              "echo 'no credentials found' >&2\nexit 1",
			expectErrContains: "failed to invoke credential helper: credential helper returned an error no credentials found",
		},
		{
			name:              "output that is not JSON is an error",
			body:              `printf 'not json'`,
			expectErrContains: "invalid character",
		},
		{
			// Pre-existing behaviour, shared with Phobos: a well-formed document without the
			// field yields an empty token rather than an error. Pinned so that changing it is a
			// decision rather than an accident.
			name:        "JSON without a token field yields an empty token",
			body:        `printf '{"access_token":"a.b.c"}'`,
			expectToken: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := New(writeHelper(t, test.body), test.args).GetToken(t.Context())

			if test.expectErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErrContains)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectToken, token)
		})
	}
}

func TestGetTokenMissingBinary(t *testing.T) {
	_, err := New(filepath.Join(t.TempDir(), "absent"), nil).GetToken(t.Context())

	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

// TestGetTokenRunsOnEveryCall matters because the helper is what mints a short-lived token: if
// the result were cached, renewal would present an expired one.
func TestGetTokenRunsOnEveryCall(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "count")
	source := New(writeHelper(t, `printf x >> `+counter+`
printf '{"token":"%s"}' "$(wc -c < `+counter+` | tr -d ' ')"`), nil)

	first, err := source.GetToken(t.Context())
	require.NoError(t, err)
	require.Equal(t, "1", first)

	second, err := source.GetToken(t.Context())
	require.NoError(t, err)
	require.Equal(t, "2", second, "the helper must run again rather than return a cached token")
}
