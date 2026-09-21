// Package sanitize coerces untrusted strings into bounded, valid UTF-8 before they are stored or displayed.
package sanitize

import "strings"

// replacement is substituted for any invalid UTF-8 byte sequence.
const replacement = "�"

// ToValidUTF8 returns s with invalid UTF-8 byte sequences replaced by the Unicode replacement character.
func ToValidUTF8(s string) string {
	return strings.ToValidUTF8(s, replacement)
}

// TruncateToValidUTF8 slices s to at most maxBytes bytes then sanitizes to valid UTF-8;
// a rune or invalid sequence split by the cut becomes the replacement character
// (so output may slightly exceed maxBytes). maxBytes <= 0 only sanitizes.
func TruncateToValidUTF8(s string, maxBytes int) string {
	if maxBytes > 0 && len(s) > maxBytes {
		s = s[:maxBytes]
	}

	return ToValidUTF8(s)
}
