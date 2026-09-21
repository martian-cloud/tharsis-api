package sanitize

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestToValidUTF8(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "valid string is unchanged", in: "hello world", want: "hello world"},
		{name: "empty string", in: "", want: ""},
		{name: "invalid bytes replaced", in: "bad \xff\xfe end", want: "bad � end"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ToValidUTF8(test.in)
			assert.Equal(t, test.want, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Run("shorter than max is unchanged", func(t *testing.T) {
		got := TruncateToValidUTF8("hello", 100)
		assert.Equal(t, "hello", got)
	})

	t.Run("truncates to max bytes", func(t *testing.T) {
		got := TruncateToValidUTF8("abcdefghij", 4)
		assert.Equal(t, "abcd", got)
		assert.LessOrEqual(t, len(got), 4)
	})

	t.Run("a rune split by truncation becomes the replacement char", func(t *testing.T) {
		// "é" is 2 bytes; cutting to 2 bytes splits it, so sanitizing replaces the dangling byte.
		got := TruncateToValidUTF8("aé", 2)
		assert.Equal(t, "a�", got)
		assert.True(t, utf8.ValidString(got))
	})

	t.Run("non-positive max only sanitizes", func(t *testing.T) {
		got := TruncateToValidUTF8("bad \xff end", 0)
		assert.Equal(t, "bad � end", got)
		assert.True(t, utf8.ValidString(got))
	})
}
