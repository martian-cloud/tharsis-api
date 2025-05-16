package gid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGlobalID_String(t *testing.T) {
	testCases := []struct {
		name           string
		globalID       *GlobalID
		expectedResult string
	}{
		{
			name: "global ID with workspace code and UUID",
			globalID: &GlobalID{
				Code: "W",
				ID:   "12345678-1234-1234-1234-123456789012",
			},
			expectedResult: "V18xMjM0NTY3OC0xMjM0LTEyMzQtMTIzNC0xMjM0NTY3ODkwMTI",
		},
		{
			name: "global ID with group code and UUID",
			globalID: &GlobalID{
				Code: "G",
				ID:   "98765432-4321-4321-4321-210987654321",
			},
			expectedResult: "R185ODc2NTQzMi00MzIxLTQzMjEtNDMyMS0yMTA5ODc2NTQzMjE",
		},
		{
			name: "global ID with empty code",
			globalID: &GlobalID{
				Code: "",
				ID:   "12345678-1234-1234-1234-123456789012",
			},
			expectedResult: "XzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg",
		},
		{
			name: "global ID with empty ID",
			globalID: &GlobalID{
				Code: "W",
				ID:   "",
			},
			expectedResult: "V18",
		},
		{
			name: "global ID with special characters in code",
			globalID: &GlobalID{
				Code: "W#$",
				ID:   "12345678-1234-1234-1234-123456789012",
			},
			expectedResult: "VyMkXzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := test.globalID.String()
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestNewGlobalID(t *testing.T) {
	testCases := []struct {
		name           string
		modelType      types.ModelType
		modelID        string
		expectedResult *GlobalID
	}{
		{
			name:      "workspace model type",
			modelType: types.WorkspaceModelType,
			modelID:   "12345678-1234-1234-1234-123456789012",
			expectedResult: &GlobalID{
				Code: "W",
				ID:   "12345678-1234-1234-1234-123456789012",
			},
		},
		{
			name:      "group model type",
			modelType: types.GroupModelType,
			modelID:   "98765432-4321-4321-4321-210987654321",
			expectedResult: &GlobalID{
				Code: "G",
				ID:   "98765432-4321-4321-4321-210987654321",
			},
		},
		{
			name:      "with empty ID",
			modelType: types.WorkspaceModelType,
			modelID:   "",
			expectedResult: &GlobalID{
				Code: "W",
				ID:   "",
			},
		},
		{
			name:      "with zero UUID",
			modelType: types.WorkspaceModelType,
			modelID:   "00000000-0000-0000-0000-000000000000",
			expectedResult: &GlobalID{
				Code: "W",
				ID:   "00000000-0000-0000-0000-000000000000",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := NewGlobalID(test.modelType, test.modelID)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestParseGlobalID(t *testing.T) {
	testCases := []struct {
		name            string
		globalIDString  string
		expectedResult  *GlobalID
		expectErrorCode errors.CodeType
	}{
		{
			name:           "valid global ID",
			globalIDString: "V18xMjM0NTY3OC0xMjM0LTEyMzQtMTIzNC0xMjM0NTY3ODkwMTI",
			expectedResult: &GlobalID{
				Code: "W",
				ID:   "12345678-1234-1234-1234-123456789012",
			},
		},
		{
			name:           "another valid global ID",
			globalIDString: "R185ODc2NTQzMi00MzIxLTQzMjEtNDMyMS0yMTA5ODc2NTQzMjE",
			expectedResult: &GlobalID{
				Code: "G",
				ID:   "98765432-4321-4321-4321-210987654321",
			},
		},
		{
			name:            "invalid base64 encoding",
			globalIDString:  "invalid-base64",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "missing underscore separator",
			globalIDString:  "V2FiY2RlZg", // Base64 of "Wabcdef"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid UUID format",
			globalIDString:  "V19pbnZhbGlkLXV1aWQ", // Base64 of "W_invalid-uuid"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "empty string",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "valid base64 but invalid internal format",
			globalIDString:  "YWJjZGVmZ2hpamtsbW5vcA", // Base64 of "abcdefghijklmnop"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:           "with zero UUID",
			globalIDString: "V18wMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDA", // Base64 of "W_00000000-0000-0000-0000-000000000000"
			expectedResult: &GlobalID{
				Code: "W",
				ID:   "00000000-0000-0000-0000-000000000000",
			},
		},
		{
			name:            "very long code",
			globalIDString:  "VkVSWUxPTkdDT0RFXzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg", // Base64 of "VERYLONGCODE_12345678-1234-1234-1234-123456789012"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "lowercase letters in code",
			globalIDString:  "YWJjXzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg", // Base64 of "abc_12345678-1234-1234-1234-123456789012"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "digits in code",
			globalIDString:  "QUJDMTIzXzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg", // Base64 of "ABC123_12345678-1234-1234-1234-123456789012"
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid characters in code",
			globalIDString:  "VyMkXzEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTAxMg", // Base64 of "W#$_12345678-1234-1234-1234-123456789012"
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseGlobalID(test.globalIDString)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func TestToGlobalID(t *testing.T) {
	testCases := []struct {
		name           string
		modelType      types.ModelType
		id             string
		expectedResult string
	}{
		{
			name:           "workspace model type",
			modelType:      types.WorkspaceModelType,
			id:             "12345678-1234-1234-1234-123456789012",
			expectedResult: "V18xMjM0NTY3OC0xMjM0LTEyMzQtMTIzNC0xMjM0NTY3ODkwMTI",
		},
		{
			name:           "group model type",
			modelType:      types.GroupModelType,
			id:             "98765432-4321-4321-4321-210987654321",
			expectedResult: "R185ODc2NTQzMi00MzIxLTQzMjEtNDMyMS0yMTA5ODc2NTQzMjE",
		},
		{
			name:           "with empty ID",
			modelType:      types.WorkspaceModelType,
			expectedResult: "V18",
		},
		{
			name:           "with zero UUID",
			modelType:      types.WorkspaceModelType,
			id:             "00000000-0000-0000-0000-000000000000",
			expectedResult: "V18wMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDA",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := ToGlobalID(test.modelType, test.id)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestFromGlobalID(t *testing.T) {
	testCases := []struct {
		name           string
		globalIDString string
		expectedResult string
	}{
		{
			name:           "valid global ID",
			globalIDString: "V18xMjM0NTY3OC0xMjM0LTEyMzQtMTIzNC0xMjM0NTY3ODkwMTI",
			expectedResult: "12345678-1234-1234-1234-123456789012",
		},
		{
			name:           "another valid global ID",
			globalIDString: "R185ODc2NTQzMi00MzIxLTQzMjEtNDMyMS0yMTA5ODc2NTQzMjE",
			expectedResult: "98765432-4321-4321-4321-210987654321",
		},
		{
			name:           "invalid global ID",
			globalIDString: "invalid-id",
			expectedResult: "invalid[invalid-id]",
		},
		{
			name:           "empty string",
			expectedResult: "invalid[]",
		},
		{
			name:           "with zero UUID",
			globalIDString: "V18wMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDA",
			expectedResult: "00000000-0000-0000-0000-000000000000",
		},
		{
			name:           "valid base64 but invalid internal format",
			globalIDString: "YWJjZGVmZ2hpamtsbW5vcA",
			expectedResult: "invalid[YWJjZGVmZ2hpamtsbW5vcA]",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := FromGlobalID(test.globalIDString)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}
