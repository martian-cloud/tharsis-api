package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatic(t *testing.T) {
	type testCase struct {
		name        string
		tokenFunc   func() (string, error)
		expectError bool
	}

	testCases := []testCase{
		{
			name:      "valid token",
			tokenFunc: func() (string, error) { return "my-token", nil },
		},
		{
			name:        "empty token",
			tokenFunc:   func() (string, error) { return "", nil },
			expectError: true,
		},
		{
			name:        "token func error",
			tokenFunc:   func() (string, error) { return "", assert.AnError },
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := NewStatic(test.tokenFunc)

			if test.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			token, err := resolver.Token(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "my-token", token)
		})
	}
}

func TestStaticTokenResolver_ReReadsOnEachCall(t *testing.T) {
	callCount := 0
	resolver, err := NewStatic(func() (string, error) {
		callCount++
		return "token", nil
	})
	require.NoError(t, err)

	_, _ = resolver.Token(t.Context())
	_, _ = resolver.Token(t.Context())

	// Once at construction + twice from Token calls.
	assert.Equal(t, 3, callCount)
}
