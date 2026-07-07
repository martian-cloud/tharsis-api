package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestCLIVersions_Latest(t *testing.T) {
	tests := []struct {
		name        string
		versions    CLIVersions
		expected    string
		expectPanic bool
	}{
		{
			name:        "empty slice panics",
			versions:    CLIVersions{},
			expectPanic: true,
		},
		{
			name:     "single element",
			versions: CLIVersions{"1.5.0"},
			expected: "1.5.0",
		},
		{
			name:     "multiple elements returns last",
			versions: CLIVersions{"1.3.0", "1.4.0", "1.5.7"},
			expected: "1.5.7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() {
					_ = test.versions.Latest()
				})
				return
			}

			assert.Equal(t, test.expected, test.versions.Latest())
		})
	}
}

func TestCLIVersions_Supported(t *testing.T) {
	tests := []struct {
		name            string
		versions        CLIVersions
		wantVersion     string
		expectErrorCode errors.CodeType
		expectError     bool
	}{
		{
			name:        "version found",
			versions:    CLIVersions{"1.3.0", "1.4.0", "1.5.7"},
			wantVersion: "1.4.0",
		},
		{
			name:            "version not found",
			versions:        CLIVersions{"1.3.0", "1.4.0", "1.5.7"},
			wantVersion:     "9.9.9",
			expectError:     true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "empty slice",
			versions:        CLIVersions{},
			wantVersion:     "1.4.0",
			expectError:     true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.versions.Supported(test.wantVersion)
			if test.expectError {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
		})
	}
}
