package run

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateAndSanitizeErrorMessage(t *testing.T) {
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
			got := TruncateAndSanitizeErrorMessage(tt.input)
			require.NotNil(t, got)
			tt.assert(t, *got)
		})
	}
}

// TestTruncateAndSanitizeErrorMessage_TruncationOnRuneBoundary covers the case where the byte-based
// truncation limit falls inside a multi-byte character. Terraform diagnostics contain box-drawing
// characters (╷ │ ╵ ─, three bytes each), so a byte slice at maxErrorMessageLength can leave an
// incomplete sequence that Postgres rejects with SQLSTATE 22021, failing the plan/apply update.
// Every cut alignment must still produce valid UTF-8. Also covers a 4-byte character (an emoji,
// outside the Basic Multilingual Plane) to confirm the worst-case byte loss bound below, not just the
// 3-byte box-drawing case that motivated the fix.
func TestTruncateAndSanitizeErrorMessage_TruncationOnRuneBoundary(t *testing.T) {
	tests := []struct {
		name    string
		char    string // the multi-byte character to straddle the truncation boundary
		numPads []int  // ASCII padding lengths, chosen to land the cut at each byte offset within char
	}{
		{
			// "─" (U+2500) encodes as 0xe2 0x94 0x80 (3 bytes). Padding with 1, 2 or 3 fewer ASCII
			// bytes than the limit makes the cut land before, one byte into, or two bytes into it.
			name:    "3-byte character (box-drawing)",
			char:    "─",
			numPads: []int{maxErrorMessageLength - 3, maxErrorMessageLength - 2, maxErrorMessageLength - 1, maxErrorMessageLength},
		},
		{
			// "😀" (U+1F600) encodes as 0xf0 0x9f 0x98 0x80 (4 bytes) -- the worst case for byte loss,
			// since up to 3 orphaned bytes can be dropped from the retained prefix (see the bound
			// asserted below), one more than the 3-byte case can lose.
			name:    "4-byte character (emoji, outside the Basic Multilingual Plane)",
			char:    "😀",
			numPads: []int{maxErrorMessageLength - 4, maxErrorMessageLength - 3, maxErrorMessageLength - 2, maxErrorMessageLength - 1, maxErrorMessageLength},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, pad := range tc.numPads {
				t.Run(fmt.Sprintf("cut with %d bytes of ASCII padding", pad), func(t *testing.T) {
					input := strings.Repeat("a", pad) + strings.Repeat(tc.char, 100)

					got := TruncateAndSanitizeErrorMessage(input)
					require.NotNil(t, got)

					assert.True(t, utf8.ValidString(*got), "truncated message must be valid UTF-8, got bytes % x", []byte(*got))
					assert.Contains(t, *got, "truncated")
					// Dropping a partial character costs at most (character byte length - 1) bytes of
					// retained content -- e.g. up to 2 bytes for a 3-byte character, up to 3 bytes for a
					// 4-byte character. There is no fixed 2-byte bound across all of Unicode. Measure
					// only the retained prefix (up to the "..." marker), not the full output, since the
					// appended notice text is much longer than maxErrorMessageLength on its own and would
					// mask any regression in how much of the prefix was dropped.
					prefixEnd := strings.Index(*got, "...")
					require.GreaterOrEqual(t, prefixEnd, 0, "expected truncated output to contain the \"...\" marker")
					maxBytesLost := len([]byte(tc.char)) - 1
					assert.GreaterOrEqual(t, prefixEnd, maxErrorMessageLength-maxBytesLost)
				})
			}
		})
	}
}

// TestTruncateAndSanitizeErrorMessage_TruncationWithInvalidBytesNearBoundary covers truncating raw
// input that contains invalid UTF-8 bytes at or near the cut point. Truncation slices the raw
// (unsanitized) input first, so an invalid byte sequence straddling the cutoff must still resolve to
// valid UTF-8 once the slice is sanitized -- there is no assumption that invalid bytes only appear
// well before or after maxErrorMessageLength.
//
// Invalid bytes are replaced with "�", the same as the untruncated path in the main test above --
// they are never silently dropped, regardless of where in the kept slice they fall.
func TestTruncateAndSanitizeErrorMessage_TruncationWithInvalidBytesNearBoundary(t *testing.T) {
	// "\xc3\x28" is not valid UTF-8 on its own (an incomplete/invalid two-byte sequence) and is 2
	// bytes long. Padding with 1 or 2 fewer ASCII bytes than the limit places at least one of those
	// bytes inside the kept [0, maxErrorMessageLength) slice; padding with 0 fewer bytes would place
	// the invalid sequence entirely past the cutoff, outside the kept slice, and is not a "near
	// boundary" case.
	for _, pad := range []int{maxErrorMessageLength - 1, maxErrorMessageLength - 2} {
		t.Run(fmt.Sprintf("cut with %d bytes of ASCII padding before invalid input bytes", pad), func(t *testing.T) {
			input := strings.Repeat("a", pad) + "\xc3\x28" + strings.Repeat("b", 100)

			got := TruncateAndSanitizeErrorMessage(input)
			require.NotNil(t, got)

			assert.True(t, utf8.ValidString(*got), "truncated message must be valid UTF-8, got bytes % x", []byte(*got))
			assert.Contains(t, *got, "truncated")
			// Replaced, not dropped: the invalid bytes are visibly flagged with "�".
			assert.Contains(t, *got, "�")
		})
	}
}

// TestTruncateAndSanitizeErrorMessage_TruncationWithInvalidBytesFarFromBoundary covers invalid bytes
// well BEFORE the truncation cutoff rather than colliding with it. This confirms invalid bytes are
// replaced with "�" no matter where they fall in the kept slice, not just ones near the cut.
func TestTruncateAndSanitizeErrorMessage_TruncationWithInvalidBytesFarFromBoundary(t *testing.T) {
	input := "bad\xc3\x28sequence" + strings.Repeat("a", maxErrorMessageLength+100)

	got := TruncateAndSanitizeErrorMessage(input)
	require.NotNil(t, got)

	assert.True(t, utf8.ValidString(*got), "truncated message must be valid UTF-8, got bytes % x", []byte(*got))
	assert.Contains(t, *got, "truncated")
	// Replaced, not dropped, even though the invalid bytes are nowhere near the cut point.
	assert.Contains(t, *got, "�")
	assert.NotContains(t, *got, "\xc3\x28")
}
