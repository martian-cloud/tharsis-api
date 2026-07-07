package run

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeAndTruncateErrorMessage(t *testing.T) {
	// A string longer than the limit that is otherwise valid UTF-8.
	longInput := strings.Repeat("a", maxErrorMessageLength+100)

	tests := []struct {
		name string
		// assert runs against the returned (dereferenced) string.
		assert func(t *testing.T, out string)
		input  string
	}{
		{
			name:  "short valid string passes through unchanged",
			input: "a normal error message",
			assert: func(t *testing.T, out string) {
				assert.Equal(t, "a normal error message", out)
			},
		},
		{
			name:  "empty string passes through unchanged",
			input: "",
			assert: func(t *testing.T, out string) {
				assert.Equal(t, "", out)
			},
		},
		{
			name:  "string exactly at the limit is not truncated",
			input: strings.Repeat("b", maxErrorMessageLength),
			assert: func(t *testing.T, out string) {
				assert.Equal(t, strings.Repeat("b", maxErrorMessageLength), out)
				assert.NotContains(t, out, "truncated")
			},
		},
		{
			name:  "string over the limit is truncated with notice appended",
			input: longInput,
			assert: func(t *testing.T, out string) {
				// The retained prefix is exactly maxErrorMessageLength of the original content.
				assert.True(t, strings.HasPrefix(out, strings.Repeat("a", maxErrorMessageLength)+"..."))
				assert.Contains(t, out, "truncated")
				// Output is longer than the prefix because the notice is appended.
				assert.Greater(t, len(out), maxErrorMessageLength)
			},
		},
		{
			name:  "invalid UTF-8 is replaced with the replacement character",
			input: "bad\xc3\x28sequence",
			assert: func(t *testing.T, out string) {
				assert.True(t, strings.ContainsRune(out, '�'))
				assert.NotContains(t, out, "\xc3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeAndTruncateErrorMessage(tt.input)
			require.NotNil(t, got)
			tt.assert(t, *got)
		})
	}
}
