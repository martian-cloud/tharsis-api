package variable

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetVariableByID(t *testing.T) {
	sampleVariable := &models.Variable{
		Metadata: models.ResourceMetadata{
			ID: "var-1",
		},
		Key:   "key1",
		Value: ptr.String("test-value"),
	}

	type testCase struct {
		name            string
		authError       error
		variable        *models.Variable
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "successfully get variable by id",
			variable: sampleVariable,
		},
		{
			name:            "variable not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view variable",
			variable: &models.Variable{
				Metadata:   sampleVariable.Metadata,
				Key:        sampleVariable.Key,
				SecretData: []byte("secret-data"),
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVariables := db.NewMockVariables(t)
			mockSecretsManager := secret.NewMockManager(t)

			mockVariables.On("GetVariableByID", mock.Anything, sampleVariable.Metadata.ID).Return(test.variable, nil)

			if test.variable != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Variables: mockVariables,
			}

			service := &service{
				dbClient:      dbClient,
				secretManager: mockSecretsManager,
			}

			actualVariable, err := service.GetVariableByID(auth.WithCaller(ctx, mockCaller), sampleVariable.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.variable, actualVariable)
		})
	}
}

func TestGetVariableByTRN(t *testing.T) {
	variableID := "var-1"
	variableTRN := types.VariableModelType.BuildTRN("variable-gid-1")

	sampleVariable := &models.Variable{
		Metadata: models.ResourceMetadata{
			ID:  variableID,
			TRN: variableTRN,
		},
		Key:   "key1",
		Value: ptr.String("test-value"),
	}

	type testCase struct {
		name            string
		authError       error
		variable        *models.Variable
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "successfully get variable by trn",
			variable: sampleVariable,
		},
		{
			name:            "variable not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view variable",
			variable: &models.Variable{
				Metadata:   sampleVariable.Metadata,
				Key:        sampleVariable.Key,
				SecretData: []byte("secret-data"),
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVariables := db.NewMockVariables(t)
			mockSecretsManager := secret.NewMockManager(t)

			mockVariables.On("GetVariableByTRN", mock.Anything, variableTRN).Return(test.variable, nil)

			if test.variable != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Variables: mockVariables,
			}

			service := &service{
				dbClient:      dbClient,
				secretManager: mockSecretsManager,
			}

			actualVariable, err := service.GetVariableByTRN(auth.WithCaller(ctx, mockCaller), variableTRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.variable, actualVariable)
		})
	}
}

func TestGetVariableVersionByID(t *testing.T) {
	variableID := "var-1"
	variableVersionID := "var-version-1"
	// Test cases
	tests := []struct {
		authError             error
		variableVersion       *models.VariableVersion
		expectVariableVersion *models.VariableVersion
		sensitive             bool
		includeSensitiveValue bool
		name                  string
		expectErrCode         errors.CodeType
	}{
		{
			name: "get variable version by id",
			variableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				Value:      ptr.String("test-value"),
			},
			expectVariableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				Value:      ptr.String("test-value"),
			},
		},
		{
			name:      "get variable version by id for sensitive variable but don't include value",
			sensitive: true,
			variableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("test-value-cipher"),
			},
			expectVariableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("test-value-cipher"),
			},
		},
		{
			name:                  "get variable version by id for sensitive variable and include value",
			sensitive:             true,
			includeSensitiveValue: true,
			variableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("test-value-cipher"),
			},
			expectVariableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("test-value-cipher"),
				Value:      ptr.String("test-value"),
			},
		},
		{
			name:          "authorization error",
			authError:     errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
			variableVersion: &models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: variableVersionID},
				VariableID: variableID,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockSecretManager := secret.NewMockManager(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockVariables := db.NewMockVariables(t)
			mockVariableVersions := db.NewMockVariableVersions(t)

			mockVariableVersions.On("GetVariableVersionByID", mock.Anything, variableVersionID).Return(test.variableVersion, nil)

			mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(test.authError)
			mockVariables.On("GetVariableByID", mock.Anything, variableID).Return(&models.Variable{Metadata: models.ResourceMetadata{ID: variableID}, Sensitive: test.sensitive}, nil)

			if test.sensitive && test.includeSensitiveValue {
				mockSecretManager.On("Get", mock.Anything, test.variableVersion.Key, test.variableVersion.SecretData).Return(*test.expectVariableVersion.Value, nil)
			}

			dbClient := db.Client{
				Variables:        mockVariables,
				VariableVersions: mockVariableVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			variableVersion, err := service.GetVariableVersionByID(auth.WithCaller(ctx, mockCaller), variableVersionID, test.includeSensitiveValue)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectVariableVersion, variableVersion)
		})
	}
}

func TestGetVariableVersionByTRN(t *testing.T) {
	variableID := "var-1"
	variableVersionID := "var-version-1"
	variableVersionTRN := types.VariableVersionModelType.BuildTRN("variable-version-gid-1")
	secretValue := "test-secret-value"

	type testCase struct {
		name                  string
		isSensitive           bool
		includeSensitiveValue bool
		authError             error
		variableVersion       *models.VariableVersion
		expectErrorCode       errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get variable version by trn",
			variableVersion: &models.VariableVersion{
				Metadata: models.ResourceMetadata{
					ID:  variableVersionID,
					TRN: variableVersionTRN,
				},
				VariableID: variableID,
				Key:        "key1",
				Value:      ptr.String("test-value"),
			},
		},
		{
			name:            "variable version not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:                  "variable is sensitive",
			isSensitive:           true,
			includeSensitiveValue: true,
			variableVersion: &models.VariableVersion{
				Metadata: models.ResourceMetadata{
					ID:  variableVersionID,
					TRN: variableVersionTRN,
				},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("secret-data"),
			},
		},
		{
			name:        "variable is sensitive but don't include sensitive value",
			isSensitive: true,
			variableVersion: &models.VariableVersion{
				Metadata: models.ResourceMetadata{
					ID:  variableVersionID,
					TRN: variableVersionTRN,
				},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("secret-data"),
			},
		},
		{
			name: "subject is not authorized to view variable version",
			variableVersion: &models.VariableVersion{
				Metadata: models.ResourceMetadata{
					ID:  variableVersionID,
					TRN: variableVersionTRN,
				},
				VariableID: variableID,
				Key:        "key1",
				SecretData: []byte("secret-data"),
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVariables := db.NewMockVariables(t)
			mockSecretsManager := secret.NewMockManager(t)
			mockVariableVersions := db.NewMockVariableVersions(t)

			mockVariableVersions.On("GetVariableVersionByTRN", mock.Anything, variableVersionTRN).Return(test.variableVersion, nil)

			if test.variableVersion != nil {
				mockVariables.On("GetVariableByID", mock.Anything, variableID).Return(&models.Variable{
					Metadata: models.ResourceMetadata{
						ID: variableVersionID,
					},
					NamespacePath: "my-group",
					Sensitive:     test.isSensitive,
				}, nil)

				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(test.authError)

				if test.isSensitive && test.includeSensitiveValue {
					mockSecretsManager.On("Get", mock.Anything, test.variableVersion.Key, test.variableVersion.SecretData).Return(secretValue, nil)
				}
			}

			dbClient := &db.Client{
				Variables:        mockVariables,
				VariableVersions: mockVariableVersions,
			}

			service := &service{
				dbClient:      dbClient,
				secretManager: mockSecretsManager,
			}

			actualVariableVersion, err := service.GetVariableVersionByTRN(auth.WithCaller(ctx, mockCaller), variableVersionTRN, test.includeSensitiveValue)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.variableVersion, actualVariableVersion)

			if test.isSensitive && test.includeSensitiveValue {
				assert.Equal(t, secretValue, *actualVariableVersion.Value)
			}
		})
	}
}

func TestGetVariableVersions(t *testing.T) {
	variableID := "var-1"

	// Test cases
	tests := []struct {
		authError        error
		input            *GetVariableVersionsInput
		variableVersions []models.VariableVersion
		name             string
		expectErrCode    errors.CodeType
	}{
		{
			name: "get variable version by id",
			input: &GetVariableVersionsInput{
				VariableID: variableID,
			},
			variableVersions: []models.VariableVersion{
				{Metadata: models.ResourceMetadata{ID: "1"}},
				{Metadata: models.ResourceMetadata{ID: "2"}},
			},
		},
		{
			name: "authorization error",
			input: &GetVariableVersionsInput{
				VariableID: variableID,
			},
			authError:     errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockSecretManager := secret.NewMockManager(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockVariables := db.NewMockVariables(t)
			mockVariableVersions := db.NewMockVariableVersions(t)

			mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(test.authError)
			mockVariables.On("GetVariableByID", mock.Anything, variableID).Return(&models.Variable{Metadata: models.ResourceMetadata{ID: variableID}}, nil)

			if test.authError == nil {
				mockVariableVersions.On("GetVariableVersions", mock.Anything, &db.GetVariableVersionsInput{
					Filter: &db.VariableVersionFilter{
						VariableID: &test.input.VariableID,
					},
				}).Return(&db.VariableVersionResult{
					VariableVersions: test.variableVersions,
				}, nil)
			}

			dbClient := db.Client{
				Variables:        mockVariables,
				VariableVersions: mockVariableVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			result, err := service.GetVariableVersions(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.variableVersions, result.VariableVersions)
		})
	}
}

func TestSetVariables(t *testing.T) {
	namespacePath := "group1"
	// Test cases
	tests := []struct {
		authError              error
		input                  *SetVariablesInput
		existingVariables      []models.Variable
		expectCreatedVariables []*models.Variable
		expectUpdatedVariables []*models.Variable
		expectDeletedVariables []*models.Variable
		name                   string
		expectErrCode          errors.CodeType
	}{
		{
			name: "set variables with no existing variables",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.TerraformVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "key1-val"},
					{Key: "key2", Value: "key2-val"},
				},
			},
			expectCreatedVariables: []*models.Variable{
				{Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
				{Key: "key2", Value: ptr.String("key2-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
			existingVariables: []models.Variable{},
		},
		{
			name: "set variables with existing variables",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.TerraformVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "key1-val-updated"},
					{Key: "key2", Value: "key2-val", Sensitive: true},
					{Key: "key4", Value: "key4-val-updated", Sensitive: true},
				},
			},
			expectCreatedVariables: []*models.Variable{
				{Key: "key2", SecretData: []byte("key2-val-encrypted"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath, Sensitive: true},
			},
			expectUpdatedVariables: []*models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val-updated"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
				{Metadata: models.ResourceMetadata{ID: "4"}, Key: "key4", SecretData: []byte("key4-val-updated-encrypted"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath, Sensitive: true},
			},
			expectDeletedVariables: []*models.Variable{
				{Metadata: models.ResourceMetadata{ID: "3"}, Key: "key3", Value: ptr.String("key3-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
			existingVariables: []models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
				{Metadata: models.ResourceMetadata{ID: "3"}, Key: "key3", Value: ptr.String("key3-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
				{Metadata: models.ResourceMetadata{ID: "4"}, Key: "key4", SecretData: []byte("key4-val-encrypted"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath, Sensitive: true},
			},
		},
		{
			name: "set variables should delete existing variable if sensitive field changes",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.TerraformVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "key1-val", Sensitive: true},
				},
			},
			expectCreatedVariables: []*models.Variable{
				{Key: "key1", SecretData: []byte("key1-val-encrypted"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath, Sensitive: true},
			},
			expectDeletedVariables: []*models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
			existingVariables: []models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
		},
		{
			name: "remove all existing variables",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.TerraformVariableCategory,
				Variables:     []*SetVariablesInputVariable{},
			},
			expectDeletedVariables: []*models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
			existingVariables: []models.Variable{
				{Metadata: models.ResourceMetadata{ID: "1"}, Key: "key1", Value: ptr.String("key1-val"), Category: models.TerraformVariableCategory, NamespacePath: namespacePath},
			},
		},
		{
			name: "duplicate variable keys",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.TerraformVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "key1-val1"},
					{Key: "key1", Value: "key1-val2", Sensitive: true},
				},
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "hcl variable cannot have environment type",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.EnvironmentVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "key1-val", Hcl: true},
				},
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "sensitive value cannot be empty",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
				Category:      models.EnvironmentVariableCategory,
				Variables: []*SetVariablesInputVariable{
					{Key: "key1", Value: "", Sensitive: true},
				},
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "authorization error",
			input: &SetVariablesInput{
				NamespacePath: namespacePath,
			},
			authError:     errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockSecretManager := secret.NewMockManager(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockVariables := db.NewMockVariables(t)
			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateVariablePermission, mock.Anything).Return(test.authError)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			if test.expectErrCode == "" {
				mockVariables.On("GetVariables", mock.Anything, &db.GetVariablesInput{
					Filter: &db.VariableFilter{
						NamespacePaths: []string{test.input.NamespacePath},
						Category:       &test.input.Category,
					},
				}).Return(&db.VariableResult{
					Variables: test.existingVariables,
				}, nil)

				if len(test.expectCreatedVariables) > 0 {
					mockVariables.On("CreateVariables", mock.Anything, test.input.NamespacePath, test.expectCreatedVariables).Return(nil)

					for _, v := range test.expectCreatedVariables {
						if v.Sensitive {
							newValue := fmt.Sprintf("%s-val", v.Key)
							mockSecretManager.On("Create", mock.Anything, v.Key, newValue).Return([]byte(fmt.Sprintf("%s-encrypted", newValue)), nil)
						}
					}
				}

				for _, v := range test.expectUpdatedVariables {
					mockVariables.On("UpdateVariable", mock.Anything, v).Return(v, nil)

					if v.Sensitive {
						oldValue := fmt.Sprintf("%s-val", v.Key)
						newValue := fmt.Sprintf("%s-updated", oldValue)

						existingSecretData := []byte(fmt.Sprintf("%s-encrypted", oldValue))

						mockSecretManager.On("Get", mock.Anything, v.Key, existingSecretData).Return(oldValue, nil)
						mockSecretManager.On("Update", mock.Anything, v.Key, existingSecretData, newValue).Return([]byte(fmt.Sprintf("%s-encrypted", newValue)), nil)
					}
				}

				for _, v := range test.expectDeletedVariables {
					mockVariables.On("DeleteVariable", mock.Anything, v).Return(nil)
				}

				mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(test.input.NamespacePath)).Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}}, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			dbClient := db.Client{
				Variables:    mockVariables,
				Groups:       mockGroups,
				Transactions: mockTransactions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			err := service.SetVariables(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateVariable(t *testing.T) {
	namespacePath := "namespace-path"
	variableID := "variable-id"
	variableCategory := models.TerraformVariableCategory
	variableHcl := true
	variableKey := "variable-key"
	variableValue := "variable-value"

	// Test cases
	tests := []struct {
		authError                   error
		expectCreatedVariable       *models.Variable
		name                        string
		expectErrCode               errors.CodeType
		input                       *CreateVariableInput
		limit                       int
		injectVariablesPerNamespace int32
		exceedsLimit                bool
	}{
		{
			name: "create namespace variable",
			input: &CreateVariableInput{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         variableValue,
			},
			expectCreatedVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         &variableValue,
			},
			limit:                       5,
			injectVariablesPerNamespace: 5,
		},
		{
			name: "create sensitive namespace variable",
			input: &CreateVariableInput{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         variableValue,
				Sensitive:     true,
			},
			expectCreatedVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				SecretData:    []byte(fmt.Sprintf("%s-encrypted", variableValue)),
			},
			limit:                       5,
			injectVariablesPerNamespace: 5,
		},
		{
			name: "sensitive namespace variable value cannot be empty",
			input: &CreateVariableInput{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         "",
				Sensitive:     true,
			},
			limit:                       5,
			injectVariablesPerNamespace: 1,
			expectErrCode:               errors.EInvalid,
		},
		{
			name: "subject does not have permission",
			input: &CreateVariableInput{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         variableValue,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "exceeds limit",
			input: &CreateVariableInput{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         variableValue,
			},
			expectCreatedVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         &variableValue,
			},
			limit:                       5,
			injectVariablesPerNamespace: 6,
			exceedsLimit:                true,
			expectErrCode:               errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateVariablePermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockVariables := db.NewMockVariables(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockSecretManager := secret.NewMockManager(t)

			if test.authError == nil {
				mockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.GetVariablesInput{
					Filter: &db.VariableFilter{
						NamespacePaths: []string{namespacePath},
						Key:            &test.input.Key,
						Category:       &test.input.Category,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(_ context.Context, _ *db.GetVariablesInput) *db.VariableResult {
					return &db.VariableResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: 0,
						},
					}
				}, nil).Once()
			}

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			if (test.expectCreatedVariable != nil) || test.exceedsLimit {
				if test.input.Sensitive {
					mockSecretManager.On("Create", mock.Anything, test.input.Key, test.input.Value).Return([]byte(fmt.Sprintf("%s-encrypted", test.input.Value)), nil)
				}

				mockVariables.On("CreateVariable", mock.Anything, mock.Anything).
					Return(test.expectCreatedVariable, nil)

					// Called inside transaction to check resource limits.
				mockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.GetVariablesInput{
					Filter: &db.VariableFilter{
						NamespacePaths: []string{namespacePath},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(_ context.Context, _ *db.GetVariablesInput) *db.VariableResult {
					return &db.VariableResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectVariablesPerNamespace,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			dbClient := db.Client{
				Transactions:   mockTransactions,
				Variables:      mockVariables,
				ResourceLimits: mockResourceLimits,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.expectErrCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			variable, err := service.CreateVariable(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedVariable, variable)
		})
	}
}

func TestUpdateVariable(t *testing.T) {
	variableID := "var1"
	namespacePath := "namespace-path"
	variableCategory := models.TerraformVariableCategory

	// Test cases
	tests := []struct {
		authError             error
		expectUpdatedVariable *models.Variable
		existingVariable      *models.Variable
		name                  string
		expectErrCode         errors.CodeType
		input                 *UpdateVariableInput
	}{
		{
			name: "update namespace variable",
			input: &UpdateVariableInput{
				ID:    variableID,
				Key:   "key1-updated",
				Value: "key1-val-updated",
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				Value:         ptr.String("key1-val"),
			},
			expectUpdatedVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1-updated",
				Value:         ptr.String("key1-val-updated"),
			},
		},
		{
			name: "update sensitive namespace variable",
			input: &UpdateVariableInput{
				ID:    variableID,
				Key:   "key1-updated",
				Value: "key1-val-updated",
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				SecretData:    []byte("key1-val-encrypted"),
				Sensitive:     true,
			},
			expectUpdatedVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1-updated",
				SecretData:    []byte("key1-val-updated-encrypted"),
				Sensitive:     true,
			},
		},
		{
			name: "empty value is not allowed for sensitive variables",
			input: &UpdateVariableInput{
				ID:    variableID,
				Key:   "key1-updated",
				Value: "",
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				SecretData:    []byte("key1-val-encrypted"),
				Sensitive:     true,
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "subject does not have permission",
			input: &UpdateVariableInput{
				ID:    variableID,
				Key:   "key1",
				Value: "key1-val",
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				Value:         ptr.String("key1-val"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateVariablePermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockTransactions := db.NewMockTransactions(t)
			mockVariables := db.NewMockVariables(t)
			mockSecretManager := secret.NewMockManager(t)

			mockVariables.On("GetVariableByID", mock.Anything, variableID).Return(test.existingVariable, nil)

			if test.existingVariable != nil {
				mockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.GetVariablesInput{
					Filter: &db.VariableFilter{
						NamespacePaths: []string{namespacePath},
						Key:            &test.input.Key,
						Category:       &test.existingVariable.Category,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(_ context.Context, _ *db.GetVariablesInput) *db.VariableResult {
					return &db.VariableResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: 0,
						},
					}
				}, nil).Maybe()
			}

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			if test.expectErrCode == "" {
				if test.existingVariable != nil && test.existingVariable.Sensitive {
					mockSecretManager.On("Update", mock.Anything, test.input.Key, test.existingVariable.SecretData, test.input.Value).Return([]byte(fmt.Sprintf("%s-encrypted", test.input.Value)), nil)
				}

				if test.expectUpdatedVariable != nil {
					mockVariables.On("UpdateVariable", mock.Anything, test.expectUpdatedVariable).
						Return(test.expectUpdatedVariable, nil)
				}
			}

			dbClient := db.Client{
				Transactions: mockTransactions,
				Variables:    mockVariables,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.expectErrCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			variable, err := service.UpdateVariable(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectUpdatedVariable, variable)
		})
	}
}

func TestDeleteVariable(t *testing.T) {
	variableID := "var1"
	namespacePath := "namespace-path"
	variableCategory := models.TerraformVariableCategory

	// Test cases
	tests := []struct {
		authError        error
		existingVariable *models.Variable
		name             string
		expectErrCode    errors.CodeType
		input            *DeleteVariableInput
	}{
		{
			name: "update namespace variable",
			input: &DeleteVariableInput{
				ID: variableID,
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				Value:         ptr.String("key1-val"),
			},
		},
		{
			name: "subject does not have permission",
			input: &DeleteVariableInput{
				ID: variableID,
			},
			existingVariable: &models.Variable{
				Metadata:      models.ResourceMetadata{ID: variableID},
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Key:           "key1",
				Value:         ptr.String("key1-val"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, models.DeleteVariablePermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockTransactions := db.NewMockTransactions(t)
			mockVariables := db.NewMockVariables(t)
			mockGroups := db.NewMockGroups(t)
			mockSecretManager := secret.NewMockManager(t)

			mockVariables.On("GetVariableByID", mock.Anything, variableID).Return(test.existingVariable, nil)
			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			if test.expectErrCode == "" {
				mockVariables.On("DeleteVariable", mock.Anything, test.existingVariable).Return(nil)
				mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(test.existingVariable.NamespacePath)).Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}}, nil)
			}

			dbClient := db.Client{
				Transactions: mockTransactions,
				Variables:    mockVariables,
				Groups:       mockGroups,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.expectErrCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, mockSecretManager, false)

			err := service.DeleteVariable(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
