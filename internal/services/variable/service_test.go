package variable

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gotest.tools/v3/assert"
)

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
		expectErrCode               string
		input                       models.Variable
		limit                       int
		injectVariablesPerNamespace int32
		exceedsLimit                bool
	}{
		{
			name: "create namespace variable",
			input: models.Variable{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         &variableValue,
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
			name: "subject does not have permission",
			input: models.Variable{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         &variableValue,
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "exceeds limit",
			input: models.Variable{
				NamespacePath: namespacePath,
				Category:      variableCategory,
				Hcl:           variableHcl,
				Key:           variableKey,
				Value:         &variableValue,
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateVariablePermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockVariables := db.NewMockVariables(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			if (test.expectCreatedVariable != nil) || test.exceedsLimit {
				mockVariables.On("CreateVariable", mock.Anything, mock.Anything).
					Return(test.expectCreatedVariable, nil)
			}

			dbClient := db.Client{
				Transactions:   mockTransactions,
				Variables:      mockVariables,
				ResourceLimits: mockResourceLimits,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil && !test.exceedsLimit {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.GetVariablesInput{
					Filter: &db.VariableFilter{
						NamespacePaths: []string{namespacePath},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetVariablesInput) *db.VariableResult {
					_ = ctx
					_ = input

					return &db.VariableResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectVariablesPerNamespace,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents)

			variable, err := service.CreateVariable(auth.WithCaller(ctx, &mockCaller), &test.input)
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
