package pagination

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	// Test cases
	tests := []struct {
		name            string
		encodedCursor   string
		expectPrimary   *primaryCursorField
		expectSecondary *secondaryCursorField
		expectErrMsg    string
	}{
		{
			name:          "primary only",
			encodedCursor: *buildTestCursor("1", ""),
			expectPrimary: &primaryCursorField{name: "id", value: "1"},
		},
		{
			name:            "primary and secondary",
			encodedCursor:   *buildTestCursor("1", "name1"),
			expectPrimary:   &primaryCursorField{name: "id", value: "1"},
			expectSecondary: &secondaryCursorField{name: "name", value: ptr.String("name1")},
		},
		{
			name:          "build cursor with invalid character",
			encodedCursor: "dGVzdA==",
			expectErrMsg:  "invalid cursor: invalid character 'e' in literal true (expecting 'r')",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cursor, err := newCursor(test.encodedCursor)
			if err != nil {
				assert.Equal(t, test.expectErrMsg, err.Error())
				return
			}

			assert.Equal(t, test.expectPrimary, cursor.primary)
			assert.Equal(t, test.expectSecondary, cursor.secondary)
		})
	}
}

func buildTestCursor(id string, name string) *string {
	c := &cursor{primary: &primaryCursorField{name: "id", value: id}}
	if name != "" {
		c.secondary = &secondaryCursorField{name: "name", value: ptr.String(name)}
	}
	encodedCursor, _ := c.encode()
	return &encodedCursor
}
