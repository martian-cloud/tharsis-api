package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestModelType_ResourcePathFromTRN(t *testing.T) {
	type testCase struct {
		name            string
		trn             string
		expectErrorCode errors.CodeType
		expectedPath    string
	}

	testCases := []testCase{
		{
			name:         "valid TRN",
			trn:          "trn:group:parent/child",
			expectedPath: "parent/child",
		},
		{
			name:            "invalid TRN prefix",
			trn:             "invalid:group:parent/child",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid TRN format",
			trn:             "trn:group:parent:child",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid model type",
			trn:             "trn:user:username",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid resource path - empty",
			trn:             "trn:group:",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid resource path - starts with slash",
			trn:             "trn:group:/parent/child",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid resource path - ends with slash",
			trn:             "trn:group:parent/child/",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			resourcePath, err := GroupModelType.ResourcePathFromTRN(test.trn)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedPath, resourcePath)
		})
	}
}

func TestModelType_BuildTRN(t *testing.T) {
	type testCase struct {
		name        string
		pathParts   []string
		expectedTRN string
	}

	testCases := []testCase{
		{
			name:        "build TRN with single path part",
			pathParts:   []string{"mygroup"},
			expectedTRN: "trn:group:mygroup",
		},
		{
			name:        "build TRN with multiple path parts",
			pathParts:   []string{"parent", "child", "grandchild"},
			expectedTRN: "trn:group:parent/child/grandchild",
		},
		{
			name:        "build TRN with no path parts",
			pathParts:   []string{},
			expectedTRN: "trn:group:",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := GroupModelType.BuildTRN(test.pathParts...)
			assert.Equal(t, test.expectedTRN, result)
		})
	}
}

func TestModelType_Name(t *testing.T) {
	type testCase struct {
		name         string
		modelType    ModelType
		expectedName string
	}

	testCases := []testCase{
		{
			name:         "group model type",
			modelType:    GroupModelType,
			expectedName: "group",
		},
		{
			name:         "user model type",
			modelType:    UserModelType,
			expectedName: "user",
		},
		{
			name:         "workspace model type",
			modelType:    WorkspaceModelType,
			expectedName: "workspace",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedName, test.modelType.Name())
		})
	}
}

func TestModelType_GIDCode(t *testing.T) {
	type testCase struct {
		name            string
		modelType       ModelType
		expectedGIDCode string
	}

	testCases := []testCase{
		{
			name:            "group model type",
			modelType:       GroupModelType,
			expectedGIDCode: "G",
		},
		{
			name:            "user model type",
			modelType:       UserModelType,
			expectedGIDCode: "U",
		},
		{
			name:            "workspace model type",
			modelType:       WorkspaceModelType,
			expectedGIDCode: "W",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedGIDCode, test.modelType.GIDCode())
		})
	}
}

func TestModelType_Equals(t *testing.T) {
	type testCase struct {
		name           string
		modelType1     ModelType
		modelType2     ModelType
		expectedResult bool
	}

	customType1 := ModelType{name: "custom", gidCode: "C1"}
	customType2 := ModelType{name: "custom", gidCode: "C1"}
	customType3 := ModelType{name: "custom", gidCode: "C2"}

	testCases := []testCase{
		{
			name:           "equal model types - group",
			modelType1:     GroupModelType,
			modelType2:     GroupModelType,
			expectedResult: true,
		},
		{
			name:           "equal model types - user",
			modelType1:     UserModelType,
			modelType2:     UserModelType,
			expectedResult: true,
		},
		{
			name:       "different model types - group vs user",
			modelType1: GroupModelType,
			modelType2: UserModelType,
		},
		{
			name:       "different model types - workspace vs team",
			modelType1: WorkspaceModelType,
			modelType2: TeamModelType,
		},
		{
			name:           "custom equal model types",
			modelType1:     customType1,
			modelType2:     customType2,
			expectedResult: true,
		},
		{
			name:       "custom different model types",
			modelType1: customType1,
			modelType2: customType3,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := test.modelType1.Equals(test.modelType2)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestIsTRN(t *testing.T) {
	type testCase struct {
		name           string
		value          string
		expectedResult bool
	}

	testCases := []testCase{
		{
			name:           "valid TRN - group",
			value:          "trn:group:parent/child",
			expectedResult: true,
		},
		{
			name:           "valid TRN - user",
			value:          "trn:user:username",
			expectedResult: true,
		},
		{
			name:  "invalid TRN - wrong prefix",
			value: "invalid:group:parent/child",
		},
		{
			name:  "invalid TRN - missing prefix",
			value: "group:parent/child",
		},
		{
			name: "invalid TRN - empty string",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := IsTRN(test.value)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestGetModelNameFromTRN(t *testing.T) {
	type testCase struct {
		name           string
		trn            string
		expectedResult string
	}

	testCases := []testCase{
		{
			name:           "valid TRN - group",
			trn:            "trn:group:parent/child",
			expectedResult: "group",
		},
		{
			name:           "valid TRN - user",
			trn:            "trn:user:username",
			expectedResult: "user",
		},
		{
			name:           "valid TRN - workspace",
			trn:            "trn:workspace:path",
			expectedResult: "workspace",
		},
		{
			name: "invalid TRN - wrong prefix",
			trn:  "invalid:group:parent/child",
		},
		{
			name: "invalid TRN - empty string",
		},
		{
			name: "TRN with empty model name",
			trn:  "trn::path",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := GetModelNameFromTRN(test.trn)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}
