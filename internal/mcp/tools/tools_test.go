package tools

import (
	stdErrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestNewToolContext(t *testing.T) {
	catalog := &services.Catalog{}
	tc := &ToolContext{servicesCatalog: catalog}
	assert.NotNil(t, tc)
	assert.Equal(t, catalog, tc.servicesCatalog)
}

func TestMCPToolError(t *testing.T) {
	type testCase struct {
		name     string
		err      error
		expected string
	}

	tests := []testCase{
		{
			name:     "simple error",
			err:      NewMCPToolError("test error"),
			expected: "test error",
		},
		{
			name:     "formatted error",
			err:      NewMCPToolError("error: %s %d", "test", 123),
			expected: "error: test 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
			_, ok := tt.err.(*MCPToolError)
			assert.True(t, ok)
		})
	}
}

func TestWrapMCPToolError(t *testing.T) {
	type testCase struct {
		name     string
		err      error
		expected string
	}

	tests := []testCase{
		{
			name:     "EInvalid error not sanitized",
			err:      errors.New("invalid input: field is required", errors.WithErrorCode(errors.EInvalid)),
			expected: "prefix: test: invalid input: field is required",
		},
		{
			name:     "EInternal error sanitized",
			err:      errors.New("database connection failed", errors.WithErrorCode(errors.EInternal)),
			expected: "prefix: test: " + errors.InternalErrorMessage,
		},
		{
			name:     "non-Tharsis error sanitized",
			err:      stdErrors.New("some error"),
			expected: "prefix: test: " + errors.InternalErrorMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapMCPToolError(tt.err, "prefix: %s", "test")
			assert.Equal(t, tt.expected, wrapped.Error())
			_, ok := wrapped.(*MCPToolError)
			assert.True(t, ok)
		})
	}
}
