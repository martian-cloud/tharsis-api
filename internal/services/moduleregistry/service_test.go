package moduleregistry

import (
	"context"
	"crypto/sha256"
	io "io"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

func TestGetModuleByID(t *testing.T) {
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModule  *models.TerraformModule
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get private module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name: "get public module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
		},
		{
			name: "subject does not have access to private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "module not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)

			mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(test.expectModule, nil)

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			module, err := service.GetModuleByID(auth.WithCaller(ctx, mockCaller), moduleID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectModule, module)
		})
	}
}

func TestGetModuleByPath(t *testing.T) {
	path := "group-1/module-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModule  *models.TerraformModule
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get private module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name: "get public module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
		},
		{
			name: "subject does not have access to private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "module not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)

			mockModules.On("GetModuleByPath", mock.Anything, path).Return(test.expectModule, nil)

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			module, err := service.GetModuleByPath(auth.WithCaller(ctx, mockCaller), path)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectModule, module)
		})
	}
}

func TestGetModuleByAddress(t *testing.T) {
	namespace := "group-1"
	moduleName := "module-1"
	system := "aws"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModule  *models.TerraformModule
		rootGroup     *models.Group
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get private module by ID",
			rootGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: groupID,
				},
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name: "get public module by ID",
			rootGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: groupID,
				},
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
		},
		{
			name: "subject does not have access to private module",
			rootGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: groupID,
				},
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module not found",
			rootGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: groupID,
				},
			},
			expectErrCode: errors.ENotFound,
		},
		{
			name:          "namespace not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockGroups := db.NewMockGroups(t)

			mockGroups.On("GetGroupByFullPath", mock.Anything, namespace).Return(test.rootGroup, nil)

			mockModules := db.NewMockTerraformModules(t)

			getModulesResponse := db.ModulesResult{
				Modules: []models.TerraformModule{},
			}

			if test.expectModule != nil {
				getModulesResponse.Modules = append(getModulesResponse.Modules, *test.expectModule)
			}

			if test.rootGroup != nil {
				mockModules.On("GetModules", mock.Anything, &db.GetModulesInput{
					PaginationOptions: &db.PaginationOptions{First: ptr.Int32(1)},
					Filter: &db.TerraformModuleFilter{
						RootGroupID: &groupID,
						Name:        &moduleName,
						System:      &system,
					},
				}).Return(&getModulesResponse, nil)
			}

			dbClient := db.Client{
				Groups:           mockGroups,
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			module, err := service.GetModuleByAddress(auth.WithCaller(ctx, mockCaller), namespace, moduleName, system)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectModule, module)
		})
	}
}

func TestGetModulesByIDs(t *testing.T) {
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModule  *models.TerraformModule
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get private module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name: "get public module by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
		},
		{
			name: "subject does not have access to private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)

			getModulesResponse := db.ModulesResult{
				Modules: []models.TerraformModule{},
			}

			if test.expectModule != nil {
				getModulesResponse.Modules = append(getModulesResponse.Modules, *test.expectModule)
			}

			mockModules.On("GetModules", mock.Anything, &db.GetModulesInput{
				Filter: &db.TerraformModuleFilter{
					TerraformModuleIDs: []string{moduleID},
				},
			}).Return(&getModulesResponse, nil)

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			modules, err := service.GetModulesByIDs(auth.WithCaller(ctx, mockCaller), []string{moduleID})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectModule != nil {
				assert.Equal(t, 1, len(modules))
				assert.Equal(t, test.expectModule, &modules[0])
			} else {
				assert.Equal(t, 0, len(modules))
			}
		})
	}
}

func TestGetModules(t *testing.T) {
	groupID := "group-1"
	// Test cases
	tests := []struct {
		input                 *GetModulesInput
		namespaceAccessPolicy *auth.NamespaceAccessPolicy
		expectModule          *models.TerraformModule
		userID                *string
		serviceAccountID      *string
		handleCaller          handleCallerFunc
		name                  string
		authError             error
		expectErrCode         string
	}{
		{
			name: "filter modules by group and allow access",
			input: &GetModulesInput{
				Group: &models.Group{
					Metadata: models.ResourceMetadata{ID: groupID},
					FullPath: "group-1",
				},
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: groupID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name: "subject does not have viewer role for group",
			input: &GetModulesInput{
				Group: &models.Group{
					Metadata: models.ResourceMetadata{ID: groupID},
					FullPath: "group-1",
				},
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "no modules matching filters",
			input: &GetModulesInput{
				Group: &models.Group{
					Metadata: models.ResourceMetadata{ID: groupID},
					FullPath: "group-1",
				},
			},
		},
		{
			name:  "subject has allow all namespace access policy",
			input: &GetModulesInput{},
			namespaceAccessPolicy: &auth.NamespaceAccessPolicy{
				AllowAll: true,
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: groupID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
		{
			name:  "user does not have allow all namespace access policy",
			input: &GetModulesInput{},
			namespaceAccessPolicy: &auth.NamespaceAccessPolicy{
				AllowAll: false,
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: groupID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			userID: ptr.String("user-1"),
			handleCaller: func(ctx context.Context, userHandler func(ctx context.Context, caller *auth.UserCaller) error, _ func(ctx context.Context, caller *auth.ServiceAccountCaller) error) error {
				return userHandler(ctx, &auth.UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}}})
			},
		},
		{
			name:  "service account does not have allow all namespace access policy",
			input: &GetModulesInput{},
			namespaceAccessPolicy: &auth.NamespaceAccessPolicy{
				AllowAll: false,
			},
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: groupID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			serviceAccountID: ptr.String("sa-1"),
			handleCaller: func(ctx context.Context, _ func(ctx context.Context, caller *auth.UserCaller) error, serviceAccountHandler func(ctx context.Context, caller *auth.ServiceAccountCaller) error) error {
				return serviceAccountHandler(ctx, &auth.ServiceAccountCaller{ServiceAccountID: "sa-1"})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.input.Group != nil {
				mockCaller.On("RequireAccessToNamespace", mock.Anything, test.input.Group.FullPath, models.ViewerRole).Return(test.authError)
			}

			if test.namespaceAccessPolicy != nil {
				mockCaller.On("GetNamespaceAccessPolicy", mock.Anything).Return(test.namespaceAccessPolicy, nil)
			}

			mockModules := db.NewMockTerraformModules(t)

			getModulesResponse := db.ModulesResult{
				Modules: []models.TerraformModule{},
			}

			if test.expectModule != nil {
				getModulesResponse.Modules = append(getModulesResponse.Modules, *test.expectModule)
			}

			if test.input.Group != nil && test.authError == nil {
				mockModules.On("GetModules", mock.Anything, &db.GetModulesInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
					Filter: &db.TerraformModuleFilter{
						Search:           test.input.Search,
						GroupID:          &test.input.Group.Metadata.ID,
					},
				}).Return(&getModulesResponse, nil)
			}

			if test.namespaceAccessPolicy != nil {
				mockModules.On("GetModules", mock.Anything, &db.GetModulesInput{
					Filter: &db.TerraformModuleFilter{
						UserID:           test.userID,
						ServiceAccountID: test.serviceAccountID,
					},
				}).Return(&getModulesResponse, nil)
			}

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := newService(testLogger, &dbClient, nil, nil, nil, test.handleCaller)

			resp, err := service.GetModules(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectModule != nil {
				assert.Equal(t, 1, len(resp.Modules))
				assert.Equal(t, test.expectModule, &resp.Modules[0])
			} else {
				assert.Equal(t, 0, len(resp.Modules))
			}
		})
	}
}

func TestCreateModule(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError           error
		group               *models.Group
		expectCreatedModule *models.TerraformModule
		name                string
		expectErrCode       string
		input               CreateModuleInput
	}{
		{
			name: "create module in root group",
			input: CreateModuleInput{
				Name:    "test-module",
				System:  "aws",
				GroupID: groupID,
				Private: true,
			},
			group: &models.Group{
				ParentID: "",
			},
			expectCreatedModule: &models.TerraformModule{
				Name:        "test-module",
				System:      "aws",
				GroupID:     groupID,
				RootGroupID: groupID,
				Private:     true,
				CreatedBy:   "mockSubject",
			},
		},
		{
			name: "create module in nested group",
			input: CreateModuleInput{
				Name:    "test-module",
				System:  "aws",
				GroupID: groupID,
				Private: true,
			},
			group: &models.Group{
				ParentID: "root-group",
				FullPath: "root-group/group-1",
			},
			expectCreatedModule: &models.TerraformModule{
				Name:        "test-module",
				System:      "aws",
				GroupID:     groupID,
				RootGroupID: "root-group",
				Private:     true,
				CreatedBy:   "mockSubject",
			},
		},
		{
			name: "subject does not have deployer role",
			input: CreateModuleInput{
				Name:    "test-module",
				System:  "aws",
				GroupID: groupID,
				Private: true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockModules := db.NewMockTerraformModules(t)
			mockGroups := db.NewMockGroups(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			if test.expectCreatedModule != nil {
				mockModules.On("CreateModule", mock.Anything, test.expectCreatedModule).
					Return(test.expectCreatedModule, nil)
			}

			if test.group != nil {
				mockGroups.On("GetGroupByID", mock.Anything, mock.Anything).Return(test.group, nil)
			}

			if test.group != nil && test.group.ParentID != "" {
				mockGroups.On("GetGroupByFullPath", mock.Anything, test.group.GetRootGroupPath()).Return(&models.Group{
					Metadata: models.ResourceMetadata{ID: "root-group"},
				}, nil)
			}

			dbClient := db.Client{
				Transactions:     mockTransactions,
				TerraformModules: mockModules,
				Groups:           mockGroups,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, asynctask.NewMockManager(t))

			module, err := service.CreateModule(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedModule, module)
		})
	}
}

func TestUpdateModule(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError     error
		input         *models.TerraformModule
		name          string
		expectErrCode string
	}{
		{
			name: "update module",
			input: &models.TerraformModule{
				Name:         "test-module",
				System:       "aws",
				GroupID:      groupID,
				ResourcePath: "group123/test-module/aws",
			},
		},
		{
			name: "subject does not have deployer role",
			input: &models.TerraformModule{
				Name:         "test-module",
				System:       "aws",
				GroupID:      groupID,
				ResourcePath: "group123/test-module/aws",
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockModules := db.NewMockTerraformModules(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockModules.On("UpdateModule", mock.Anything, test.input).
					Return(test.input, nil)
			}

			dbClient := db.Client{
				Transactions:     mockTransactions,
				TerraformModules: mockModules,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, asynctask.NewMockManager(t))

			module, err := service.UpdateModule(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.input, module)
		})
	}
}

func TestDeleteModule(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError     error
		input         *models.TerraformModule
		name          string
		expectErrCode string
	}{
		{
			name: "delete module",
			input: &models.TerraformModule{
				Name:         "test-module",
				System:       "aws",
				GroupID:      groupID,
				ResourcePath: "group123/test-module/aws",
			},
		},
		{
			name: "subject does not have deployer role",
			input: &models.TerraformModule{
				Name:         "test-module",
				System:       "aws",
				GroupID:      groupID,
				ResourcePath: "group123/test-module/aws",
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockModules := db.NewMockTerraformModules(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockModules.On("DeleteModule", mock.Anything, test.input).
					Return(nil)
			}

			dbClient := db.Client{
				Transactions:     mockTransactions,
				TerraformModules: mockModules,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, asynctask.NewMockManager(t))

			err := service.DeleteModule(auth.WithCaller(ctx, &mockCaller), test.input)
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

func TestGetModuleVersionByID(t *testing.T) {
	moduleVersionID := "module-version-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModuleVersion *models.TerraformModuleVersion
		expectModule        *models.TerraformModule
		name                string
		authError           error
		expectErrCode       string
	}{
		{
			name: "get private module version by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "get public module version by ID",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "subject does not have access to private module version",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "module version not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)
			mockModuleVersions := db.NewMockTerraformModuleVersions(t)

			if test.expectModule != nil {
				mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(test.expectModule, nil)
			}

			mockModuleVersions.On("GetModuleVersionByID", mock.Anything, moduleVersionID).Return(test.expectModuleVersion, nil)

			dbClient := db.Client{
				TerraformModules:        mockModules,
				TerraformModuleVersions: mockModuleVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			moduleVersion, err := service.GetModuleVersionByID(auth.WithCaller(ctx, mockCaller), moduleVersionID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectModuleVersion, moduleVersion)
		})
	}
}

func TestGetModuleVersions(t *testing.T) {
	moduleVersionID := "module-version-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModuleVersion *models.TerraformModuleVersion
		expectModule        *models.TerraformModule
		name                string
		authError           error
		expectErrCode       string
	}{
		{
			name: "get versions for private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "get versions for public module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "subject does not have access to private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module doesn't have any versions",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)
			mockModuleVersions := db.NewMockTerraformModuleVersions(t)

			if test.expectModule != nil {
				mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(test.expectModule, nil)
			}

			getModuleVersionsResponse := db.ModuleVersionsResult{
				ModuleVersions: []models.TerraformModuleVersion{},
			}

			if test.expectModuleVersion != nil {
				getModuleVersionsResponse.ModuleVersions = append(getModuleVersionsResponse.ModuleVersions, *test.expectModuleVersion)
			}

			if test.authError == nil {
				mockModuleVersions.On("GetModuleVersions", mock.Anything, &db.GetModuleVersionsInput{
					Filter: &db.TerraformModuleVersionFilter{
						ModuleID: &moduleID,
					},
				}).Return(&getModuleVersionsResponse, nil)
			}

			dbClient := db.Client{
				TerraformModules:        mockModules,
				TerraformModuleVersions: mockModuleVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			response, err := service.GetModuleVersions(auth.WithCaller(ctx, mockCaller), &GetModuleVersionsInput{
				ModuleID: moduleID,
			})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectModuleVersion != nil {
				assert.Equal(t, 1, len(response.ModuleVersions))
				assert.Equal(t, test.expectModuleVersion, &response.ModuleVersions[0])
			} else {
				assert.Equal(t, 0, len(response.ModuleVersions))
			}
		})
	}
}

func TestGetModuleVersionsByIDs(t *testing.T) {
	moduleVersionID := "module-version-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectModuleVersion *models.TerraformModuleVersion
		expectModule        *models.TerraformModule
		name                string
		authError           error
		expectErrCode       string
	}{
		{
			name: "get module version by ID for private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "get module version by ID for public module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
		},
		{
			name: "subject does not have access to private module",
			expectModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: moduleVersionID},
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module version not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectModule != nil && test.expectModule.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockModules := db.NewMockTerraformModules(t)
			mockModuleVersions := db.NewMockTerraformModuleVersions(t)

			if test.expectModuleVersion != nil {
				getModulesResponse := db.ModulesResult{
					Modules: []models.TerraformModule{},
				}

				if test.expectModule != nil {
					getModulesResponse.Modules = append(getModulesResponse.Modules, *test.expectModule)
				}

				mockModules.On("GetModules", mock.Anything, &db.GetModulesInput{
					Filter: &db.TerraformModuleFilter{
						TerraformModuleIDs: []string{moduleID},
					},
				}).Return(&getModulesResponse, nil)
			}

			getModuleVersionsResponse := db.ModuleVersionsResult{
				ModuleVersions: []models.TerraformModuleVersion{},
			}

			if test.expectModuleVersion != nil {
				getModuleVersionsResponse.ModuleVersions = append(getModuleVersionsResponse.ModuleVersions, *test.expectModuleVersion)
			}

			mockModuleVersions.On("GetModuleVersions", mock.Anything, &db.GetModuleVersionsInput{
				Filter: &db.TerraformModuleVersionFilter{
					ModuleVersionIDs: []string{moduleVersionID},
				},
			}).Return(&getModuleVersionsResponse, nil)

			dbClient := db.Client{
				TerraformModules:        mockModules,
				TerraformModuleVersions: mockModuleVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil)

			moduleVersions, err := service.GetModuleVersionsByIDs(auth.WithCaller(ctx, mockCaller), []string{moduleVersionID})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectModuleVersion != nil {
				assert.Equal(t, 1, len(moduleVersions))
				assert.Equal(t, test.expectModuleVersion, &moduleVersions[0])
			} else {
				assert.Equal(t, 0, len(moduleVersions))
			}
		})
	}
}

func TestCreateModuleVersion(t *testing.T) {
	moduleID := "module123"
	groupID := "group123"

	// Test cases
	tests := []struct {
		latestModuleVersion        *models.TerraformModuleVersion
		expectUpdatedModuleVersion *models.TerraformModuleVersion
		expectCreatedModuleVersion *models.TerraformModuleVersion
		name                       string
		input                      CreateModuleVersionInput
		authError                  error
		expectErrCode              string
	}{
		{
			name: "subject does not have deployer role",
			input: CreateModuleVersionInput{
				SemanticVersion: "0.1.0",
				ModuleID:        moduleID,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "existing latest is a pre-release and new version is not a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "0.1.0",
				ModuleID:        moduleID,
			},
			latestModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
			expectUpdatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          false,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.1.0",
				Latest:          true,
			},
		},
		{
			name: "existing latest is not a pre-release and new version is a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "1.0.0-pre",
				ModuleID:        moduleID,
			},
			latestModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.0.1",
				Latest:          true,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          false,
			},
		},
		{
			name: "existing latest is a pre-release and new version is a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "1.0.0-pre",
				ModuleID:        moduleID,
			},
			latestModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.0.1-pre",
				Latest:          true,
			},
			expectUpdatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.0.1-pre",
				Latest:          false,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
		},
		{
			name: "existing latest is not a pre-release and new version is not a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
			latestModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.0.1",
				Latest:          true,
			},
			expectUpdatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "0.0.1",
				Latest:          false,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
		},
		{
			name: "no current latest and new version is not a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "1.0.0",
				ModuleID:        moduleID,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0",
				Latest:          true,
				Status:          models.TerraformModuleVersionStatusPending,
			},
		},
		{
			name: "no current latest and new version is a pre-release",
			input: CreateModuleVersionInput{
				SemanticVersion: "1.0.0-pre",
				ModuleID:        moduleID,
			},
			expectCreatedModuleVersion: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockModules := db.MockTerraformModules{}
			mockModules.Test(t)

			mockModuleVersions := db.MockTerraformModuleVersions{}
			mockModuleVersions.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(&models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID: moduleID,
				},
				GroupID:      groupID,
				ResourcePath: "testgroup/testmodule",
			}, nil)

			mockActivityEvents := activityevent.NewMockService(t)

			moduleVersionsResult := db.ModuleVersionsResult{
				ModuleVersions: []models.TerraformModuleVersion{},
			}

			if test.latestModuleVersion != nil {
				moduleVersionsResult.ModuleVersions = append(moduleVersionsResult.ModuleVersions, *test.latestModuleVersion)
			}

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockModuleVersions.On("GetModuleVersions", mock.Anything, &db.GetModuleVersionsInput{
					PaginationOptions: &db.PaginationOptions{
						First: ptr.Int32(1),
					},
					Filter: &db.TerraformModuleVersionFilter{
						ModuleID: &moduleID,
						Latest:   ptr.Bool(true),
					},
				}).Return(&moduleVersionsResult, nil)

				if test.expectUpdatedModuleVersion != nil {
					mockModuleVersions.On("UpdateModuleVersion", mock.Anything, test.expectUpdatedModuleVersion).
						Return(test.expectUpdatedModuleVersion, nil)
				}

				mockModuleVersions.On("CreateModuleVersion", mock.Anything, &models.TerraformModuleVersion{
					ModuleID:        moduleID,
					SemanticVersion: test.expectCreatedModuleVersion.SemanticVersion,
					Latest:          test.expectCreatedModuleVersion.Latest,
					SHASum:          test.expectCreatedModuleVersion.SHASum,
					CreatedBy:       "mockSubject",
					Status:          models.TerraformModuleVersionStatusPending,
				}).
					Return(test.expectCreatedModuleVersion, nil)

				mockGroups.On("GetGroupByID", mock.Anything, mock.Anything).Return(&models.Group{
					FullPath: "testGroupFullPath",
				}, nil)
			}

			dbClient := db.Client{
				Transactions:            &mockTransactions,
				TerraformModules:        &mockModules,
				TerraformModuleVersions: &mockModuleVersions,
				Groups:                  &mockGroups,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, asynctask.NewMockManager(t))

			moduleVersion, err := service.CreateModuleVersion(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedModuleVersion.SemanticVersion, moduleVersion.SemanticVersion)
			assert.Equal(t, test.expectCreatedModuleVersion.Latest, moduleVersion.Latest)
		})
	}
}

func TestDeleteModuleVersion(t *testing.T) {
	moduleID := "module123"
	groupID := "group123"

	// Test cases
	tests := []struct {
		expectUpdatedModuleVersion *models.TerraformModuleVersion
		name                       string
		existingModuleVersions     []models.TerraformModuleVersion
		moduleVersionToDelete      models.TerraformModuleVersion
		authError                  error
		expectErrCode              string
	}{
		{
			name: "subject does not have deployer role",
			moduleVersionToDelete: models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			authError:     errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "version to delete is the latest version",
			moduleVersionToDelete: models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			existingModuleVersions: []models.TerraformModuleVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0-pre",
					Latest:          false,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ModuleID:        moduleID,
					SemanticVersion: "0.9.0",
					Latest:          false,
				},
			},
			expectUpdatedModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "2"},
				ModuleID:        moduleID,
				SemanticVersion: "0.9.0",
				Latest:          true,
			},
		},
		{
			name: "version to delete is not the latest version",
			moduleVersionToDelete: models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0",
				Latest:          false,
			},
			existingModuleVersions: []models.TerraformModuleVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0",
					Latest:          false,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.1",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ModuleID:        moduleID,
					SemanticVersion: "0.9.0",
					Latest:          false,
				},
			},
		},
		{
			name: "version to delete is the latest version and a pre-release",
			moduleVersionToDelete: models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0-pre.2",
				Latest:          true,
			},
			existingModuleVersions: []models.TerraformModuleVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0-pre.2",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0-pre.1",
					Latest:          false,
				},
			},
			expectUpdatedModuleVersion: &models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "2"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0-pre.1",
				Latest:          true,
			},
		},
		{
			name: "version to delete is the only version",
			moduleVersionToDelete: models.TerraformModuleVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ModuleID:        moduleID,
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			existingModuleVersions: []models.TerraformModuleVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ModuleID:        moduleID,
					SemanticVersion: "1.0.0",
					Latest:          true,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockModules := db.MockTerraformModules{}
			mockModules.Test(t)

			mockModuleVersions := db.MockTerraformModuleVersions{}
			mockModuleVersions.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(&models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID: moduleID,
				},
				GroupID: groupID,
			}, nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				moduleVersionsResult := db.ModuleVersionsResult{
					ModuleVersions: test.existingModuleVersions,
				}

				mockModuleVersions.On("GetModuleVersions", mock.Anything, &db.GetModuleVersionsInput{
					Filter: &db.TerraformModuleVersionFilter{
						ModuleID: &moduleID,
					},
				}).Return(&moduleVersionsResult, nil)

				if test.expectUpdatedModuleVersion != nil {
					mockModuleVersions.On("UpdateModuleVersion", mock.Anything, test.expectUpdatedModuleVersion).
						Return(test.expectUpdatedModuleVersion, nil)
				}

				mockModuleVersions.On("DeleteModuleVersion", mock.Anything, &test.moduleVersionToDelete).Return(nil)
			}

			dbClient := db.Client{
				Transactions:            &mockTransactions,
				TerraformModules:        &mockModules,
				TerraformModuleVersions: &mockModuleVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, &mockActivityEvents, nil)

			err := service.DeleteModuleVersion(auth.WithCaller(ctx, &mockCaller), &test.moduleVersionToDelete)
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

func TestGetModuleConfigurationDetails(t *testing.T) {
	moduleVersionID := "module-version-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		input         *models.TerraformModuleVersion
		module        *models.TerraformModule
		expectDetails *ModuleConfigurationDetails
		path          string
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get config details for private module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			path:          "root",
			expectDetails: &ModuleConfigurationDetails{},
		},
		{
			name: "get config details for public module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
			path:          "examples/example1",
			expectDetails: &ModuleConfigurationDetails{},
		},
		{
			name: "subject does not have access to private module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module not found",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.module != nil && test.module.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockRegistryStore := NewMockRegistryStore(t)

			mockModules := db.NewMockTerraformModules(t)

			mockModules.On("GetModuleByID", mock.Anything, test.input.ModuleID).Return(test.module, nil)

			if test.expectErrCode == "" {
				mockRegistryStore.
					On("GetModuleConfigurationDetails", mock.Anything, test.input, test.module, test.path).
					Return(io.NopCloser(strings.NewReader("{}")), nil)
			}

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, mockRegistryStore, nil, nil)

			details, err := service.GetModuleConfigurationDetails(auth.WithCaller(ctx, mockCaller), test.input, test.path)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectDetails, details)
		})
	}
}

func TestUploadModuleVersionPackage(t *testing.T) {
	moduleID := "module123"
	groupID := "group123"

	checksum := sha256.New()
	checksum.Write([]byte("test module"))
	shaSum := checksum.Sum(nil)

	// Test cases
	tests := []struct {
		input         *models.TerraformModuleVersion
		data          string
		name          string
		shaSum        []byte
		expectStatus  models.TerraformModuleVersionStatus
		authError     error
		expectErrCode string
	}{
		{
			name: "subject does not have deployer role",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusPending,
			},
			authError:     errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module version upload is already in progress",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusUploadInProgress,
			},
			expectErrCode: errors.EConflict,
		},
		{
			name: "module version upload is already complete",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusUploaded,
			},
			expectErrCode: errors.EConflict,
		},
		{
			name: "module version upload has failed",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusErrored,
			},
			expectErrCode: errors.EConflict,
		},
		{
			name: "successsful module version upload",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusPending,
			},
			data:         "test module",
			shaSum:       shaSum,
			expectStatus: models.TerraformModuleVersionStatusUploadInProgress,
		},
		{
			name: "checksum does not match expected",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: "1"},
				ModuleID: moduleID,
				Status:   models.TerraformModuleVersionStatusPending,
			},
			data:         "test module",
			shaSum:       []byte("invalid checksum"),
			expectStatus: models.TerraformModuleVersionStatusErrored,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequireAccessToGroup", mock.Anything, groupID, models.DeployerRole).Return(test.authError)
			//mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockModules := db.NewMockTerraformModules(t)
			mockModuleVersions := db.NewMockTerraformModuleVersions(t)

			mockModules.On("GetModuleByID", mock.Anything, moduleID).Return(&models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID: moduleID,
				},
				GroupID: groupID,
			}, nil)

			mockActivityEvents := activityevent.NewMockService(t)
			mockRegistryStore := NewMockRegistryStore(t)
			mockTaskManager := asynctask.NewMockManager(t)

			if test.expectErrCode == "" {
				mockRegistryStore.
					On(
						"UploadModulePackage",
						mock.Anything,
						mock.MatchedBy(func(mv *models.TerraformModuleVersion) bool {
							return mv.Metadata.ID == test.input.Metadata.ID
						}),
						mock.MatchedBy(func(m *models.TerraformModule) bool {
							return m.Metadata.ID == test.input.ModuleID
						}),
						mock.Anything,
					).
					Return(func(
						_ context.Context,
						_ *models.TerraformModuleVersion,
						_ *models.TerraformModule,
						body io.Reader,
					) error {
						// Read all input to calculate checksum
						_, err := io.ReadAll(body)
						return err
					})

				if test.expectStatus != models.TerraformModuleVersionStatusErrored {
					mockTaskManager.On("StartTask", mock.Anything)
				}

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				argMatcherFunc := mock.MatchedBy(func(mv *models.TerraformModuleVersion) bool {
					return mv.Status == models.TerraformModuleVersionStatusUploadInProgress
				})
				mockModuleVersions.
					On("UpdateModuleVersion", mock.Anything, argMatcherFunc).
					Return(&models.TerraformModuleVersion{
						Metadata: models.ResourceMetadata{ID: "1"},
						ModuleID: moduleID,
						Status:   test.expectStatus,
						SHASum:   test.shaSum,
					}, nil)

				if test.expectStatus == models.TerraformModuleVersionStatusErrored {
					mockModuleVersions.
						On("GetModuleVersionByID", mock.Anything, test.input.Metadata.ID).Return(test.input, nil)

					argMatcherFunc := mock.MatchedBy(func(mv *models.TerraformModuleVersion) bool {
						return mv.Status == models.TerraformModuleVersionStatusErrored
					})
					mockModuleVersions.
						On("UpdateModuleVersion", mock.Anything, argMatcherFunc).
						Return(&models.TerraformModuleVersion{
							Metadata: models.ResourceMetadata{ID: "1"},
							ModuleID: moduleID,
							Status:   test.expectStatus,
							SHASum:   test.shaSum,
						}, nil)
				}
			}

			dbClient := db.Client{
				Transactions:            mockTransactions,
				TerraformModules:        mockModules,
				TerraformModuleVersions: mockModuleVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, mockRegistryStore, mockActivityEvents, mockTaskManager)

			err := service.UploadModuleVersionPackage(auth.WithCaller(ctx, mockCaller), test.input, strings.NewReader(test.data))
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

func TestGetModuleVersionPackageDownloadURL(t *testing.T) {
	moduleVersionID := "module-version-1"
	moduleID := "module-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		input         *models.TerraformModuleVersion
		module        *models.TerraformModule
		expectURL     string
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get download url for private module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			expectURL: "http://testdownload",
		},
		{
			name: "get download url for public module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  false,
			},
			expectURL: "http://testdownload",
		},
		{
			name: "subject does not have access to private module",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			module: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: moduleID},
				GroupID:  groupID,
				Name:     "test-module",
				Private:  true,
			},
			authError:     errors.NewError(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "module not found",
			input: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{ID: moduleVersionID},
				ModuleID: moduleID,
			},
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.module != nil && test.module.Private {
				mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, groupID).Return(test.authError)
			}

			mockRegistryStore := NewMockRegistryStore(t)

			mockModules := db.NewMockTerraformModules(t)

			mockModules.On("GetModuleByID", mock.Anything, test.input.ModuleID).Return(test.module, nil)

			if test.expectErrCode == "" {
				mockRegistryStore.
					On("GetModulePackagePresignedURL", mock.Anything, test.input, test.module).
					Return(test.expectURL, nil)
			}

			dbClient := db.Client{
				TerraformModules: mockModules,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, mockRegistryStore, nil, nil)

			url, err := service.GetModuleVersionPackageDownloadURL(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectURL, url)
		})
	}
}
