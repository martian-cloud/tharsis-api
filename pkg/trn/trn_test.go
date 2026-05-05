package trn

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	type testCase struct {
		name              string
		input             string
		expectError       bool
		expectedType      Type
		expectedPath      string
		expectedParent    string
		expectedBaseName  string
		expectedParts     []string
		expectedHasParent bool
	}

	testCases := []testCase{
		{
			name:              "multi-segment path",
			input:             "trn:workspace:group/my-ws",
			expectedType:      TypeWorkspace,
			expectedPath:      "group/my-ws",
			expectedParent:    "group",
			expectedBaseName:  "my-ws",
			expectedParts:     []string{"group", "my-ws"},
			expectedHasParent: true,
		},
		{
			name:              "deep path",
			input:             "trn:run:top/sub/workspace/run-id",
			expectedType:      TypeRun,
			expectedPath:      "top/sub/workspace/run-id",
			expectedParent:    "top/sub/workspace",
			expectedBaseName:  "run-id",
			expectedParts:     []string{"top", "sub", "workspace", "run-id"},
			expectedHasParent: true,
		},
		{
			name:             "single-segment path",
			input:            "trn:team:my-team",
			expectedType:     TypeTeam,
			expectedPath:     "my-team",
			expectedBaseName: "my-team",
			expectedParts:    []string{"my-team"},
		},
		{
			name:             "colon in path",
			input:            "trn:group:parent:child",
			expectedType:     TypeGroup,
			expectedPath:     "parent:child",
			expectedBaseName: "parent:child",
			expectedParts:    []string{"parent:child"},
		},
		{
			name:        "not a TRN",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "empty path",
			input:       "trn:group:",
			expectError: true,
		},
		{
			name:        "path starts with slash",
			input:       "trn:group:/parent",
			expectError: true,
		},
		{
			name:        "path ends with slash",
			input:       "trn:group:parent/",
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := ParseAny(test.input)

			if test.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedType, parsed.Type())
			assert.Equal(t, test.expectedPath, parsed.Path())
			assert.Equal(t, test.expectedParent, parsed.ParentPath())
			assert.Equal(t, test.expectedBaseName, parsed.BaseName())
			assert.Equal(t, test.expectedParts, parsed.PathParts())
			assert.Equal(t, test.expectedHasParent, parsed.HasParent())
			assert.Equal(t, test.input, parsed.String())
		})
	}
}

func TestMustParse(t *testing.T) {
	t.Run("valid TRN", func(t *testing.T) {
		parsed := MustParseAny("trn:workspace:group/my-ws")
		assert.Equal(t, TypeWorkspace, parsed.Type())
		assert.Equal(t, "group/my-ws", parsed.Path())
	})

	t.Run("invalid TRN panics", func(t *testing.T) {
		assert.Panics(t, func() { MustParseAny("invalid") })
	})
}

func TestBuild(t *testing.T) {
	type testCase struct {
		name     string
		typeName Type
		parts    []string
		expected string
	}

	testCases := []testCase{
		{
			name:     "single path part",
			typeName: TypeGroup,
			parts:    []string{"mygroup"},
			expected: "trn:group:mygroup",
		},
		{
			name:     "multiple path parts",
			typeName: TypeWorkspace,
			parts:    []string{"parent", "child"},
			expected: "trn:workspace:parent/child",
		},
		{
			name:     "pre-joined path",
			typeName: TypeWorkspace,
			parts:    []string{"parent/child"},
			expected: "trn:workspace:parent/child",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.typeName.Build(test.parts...))
		})
	}
}

func TestNormalize(t *testing.T) {
	validGID := base64.RawURLEncoding.EncodeToString([]byte("SA_d3b07384-d113-4ec6-a62d-45e1027a6b9a"))

	type testCase struct {
		name     string
		typeName Type
		input    string
		expected string
	}

	testCases := []testCase{
		{
			name:     "already a TRN",
			typeName: TypeGroup,
			input:    "trn:group:mygroup",
			expected: "trn:group:mygroup",
		},
		{
			name:     "GID passthrough",
			typeName: TypeServiceAccount,
			input:    validGID,
			expected: validGID,
		},
		{
			name:     "path becomes TRN",
			typeName: TypeGroup,
			input:    "parent/child",
			expected: "trn:group:parent/child",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.typeName.Normalize(test.input))
		})
	}
}

func TestIsTRN(t *testing.T) {
	type testCase struct {
		name     string
		value    string
		expected bool
	}

	testCases := []testCase{
		{
			name:     "valid TRN",
			value:    "trn:group:parent/child",
			expected: true,
		},
		{
			name:  "wrong prefix",
			value: "invalid:group:parent/child",
		},
		{
			name: "empty string",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, IsTRN(test.value))
		})
	}
}

func TestTypeString(t *testing.T) {
	assert.Equal(t, "workspace", TypeWorkspace.String())
	assert.Equal(t, "group", TypeGroup.String())
}
