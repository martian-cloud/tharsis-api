package file

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeToken(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestGetToken(t *testing.T) {
	tests := []struct {
		name              string
		contents          string
		expectToken       string
		expectErrContains string
	}{
		{
			name:        "a token as Kubernetes writes it, with no trailing newline",
			contents:    "a.b.c",
			expectToken: "a.b.c",
		},
		{
			name:        "a trailing newline is trimmed",
			contents:    "a.b.c\n",
			expectToken: "a.b.c",
		},
		{
			name:        "surrounding whitespace is trimmed",
			contents:    " \n\ta.b.c\n ",
			expectToken: "a.b.c",
		},
		{
			name:              "an empty file is an error rather than an empty token",
			contents:          "",
			expectErrContains: "is empty",
		},
		{
			name:              "a whitespace-only file is an error rather than an empty token",
			contents:          " \n\t ",
			expectErrContains: "is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := New(writeToken(t, test.contents)).GetToken(t.Context())

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

func TestGetTokenMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent")

	_, err := New(path).GetToken(t.Context())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read OIDC token file")
	require.Contains(t, err.Error(), path, "the message must name the path that was configured")
}

func TestGetTokenUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes do not deny access the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root, which can read a mode 000 file")
	}

	path := writeToken(t, "a.b.c")
	require.NoError(t, os.Chmod(path, 0o000))

	_, err := New(path).GetToken(t.Context())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read OIDC token file")
}

// TestGetTokenRereadsOnEveryCall is the property the Kubernetes service account token method
// depends on: the kubelet replaces the projected token in place, and the source must return what
// is on disk now rather than what it read first.
func TestGetTokenRereadsOnEveryCall(t *testing.T) {
	path := writeToken(t, "first")
	source := New(path)

	first, err := source.GetToken(t.Context())
	require.NoError(t, err)
	require.Equal(t, "first", first)

	// What the kubelet does: same path, different contents.
	require.NoError(t, os.WriteFile(path, []byte("second"), 0o600))

	second, err := source.GetToken(t.Context())
	require.NoError(t, err)
	require.Equal(t, "second", second, "the source must re-read the file, not cache the first token")
}
