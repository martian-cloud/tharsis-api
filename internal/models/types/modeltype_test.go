package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	customType1 := ModelType{trnType: "custom", gidCode: "C1"}
	customType2 := ModelType{trnType: "custom", gidCode: "C1"}
	customType3 := ModelType{trnType: "custom", gidCode: "C2"}

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
