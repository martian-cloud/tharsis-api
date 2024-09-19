package providerregistry

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
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
)

func TestCreateProvider(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError               error
		group                   *models.Group
		expectCreatedProvider   *models.TerraformProvider
		name                    string
		expectErrCode           errors.CodeType
		input                   CreateProviderInput
		limit                   int
		injectProvidersPerGroup int32
		exceedsLimit            bool
	}{
		{
			name: "create provider in root group",
			input: CreateProviderInput{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				Private:       true,
			},
			group: &models.Group{
				ParentID: "",
			},
			expectCreatedProvider: &models.TerraformProvider{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				RootGroupID:   groupID,
				Private:       true,
				CreatedBy:     "mockSubject",
			},
			limit:                   5,
			injectProvidersPerGroup: 5,
		},
		{
			name: "create provider in nested group",
			input: CreateProviderInput{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				Private:       true,
			},
			group: &models.Group{
				ParentID: "root-group",
				FullPath: "root-group/group-1",
			},
			expectCreatedProvider: &models.TerraformProvider{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				RootGroupID:   "root-group",
				Private:       true,
				CreatedBy:     "mockSubject",
			},
			limit:                   5,
			injectProvidersPerGroup: 5,
		},
		{
			name: "subject does not have deployer role",
			input: CreateProviderInput{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				Private:       true,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "exceeds limit",
			input: CreateProviderInput{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				Private:       true,
			},
			group: &models.Group{
				ParentID: "",
			},
			expectCreatedProvider: &models.TerraformProvider{
				Name:          "test-provider",
				RepositoryURL: "https://github.com/owner/repository",
				GroupID:       groupID,
				RootGroupID:   groupID,
				Private:       true,
				CreatedBy:     "mockSubject",
			},
			limit:                   5,
			injectProvidersPerGroup: 6,
			exceedsLimit:            true,
			expectErrCode:           errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateTerraformProviderPermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockProviders := db.NewMockTerraformProviders(t)
			mockGroups := db.NewMockGroups(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			if test.expectCreatedProvider != nil {
				mockProviders.On("CreateProvider", mock.Anything, test.expectCreatedProvider).
					Return(test.expectCreatedProvider, nil)
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
				Transactions:       mockTransactions,
				TerraformProviders: mockProviders,
				Groups:             mockGroups,
				ResourceLimits:     mockResourceLimits,
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockProviders.On("GetProviders", mock.Anything, mock.Anything).Return(&db.GetProvidersInput{
					Filter: &db.TerraformProviderFilter{
						GroupID: &groupID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetProvidersInput) *db.ProvidersResult {
					_ = ctx
					_ = input

					return &db.ProvidersResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectProvidersPerGroup,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if (test.authError == nil) && !test.exceedsLimit {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, mockActivityEvents)

			provider, err := service.CreateProvider(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedProvider, provider)
		})
	}
}

func TestCreateProviderVersion(t *testing.T) {
	providerID := "provider123"
	groupID := "group123"
	currentTime := time.Now().UTC()

	// Test cases
	tests := []struct {
		latestProviderVersion        *models.TerraformProviderVersion
		expectUpdatedProviderVersion *models.TerraformProviderVersion
		expectCreatedProviderVersion *models.TerraformProviderVersion
		name                         string
		expectErrorCode              errors.CodeType
		input                        CreateProviderVersionInput
		limit                        int
		injectVersionsPerProvider    int32
		exceedsLimit                 bool
	}{
		{
			name: "existing latest is a pre-release and new version is not a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "0.1.0",
				ProviderID:      providerID,
			},
			latestProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
			expectUpdatedProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0-pre",
				Latest:          false,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "0.1.0",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "existing latest is not a pre-release and new version is a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0-pre",
				ProviderID:      providerID,
			},
			latestProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "0.0.1",
				Latest:          true,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0-pre",
				Latest:          false,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "existing latest is a pre-release and new version is a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0-pre",
				ProviderID:      providerID,
			},
			latestProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "0.0.1-pre",
				Latest:          true,
			},
			expectUpdatedProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "0.0.1-pre",
				Latest:          false,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "existing latest is not a pre-release and new version is not a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0",
				ProviderID:      providerID,
			},
			latestProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "0.0.1",
				Latest:          true,
			},
			expectUpdatedProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "0.0.1",
				Latest:          false,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "no current latest and new version is not a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0",
				ProviderID:      providerID,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "no current latest and new version is a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0-pre",
				ProviderID:      providerID,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 5,
		},
		{
			name: "exceeds limit",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0-pre",
				ProviderID:      providerID,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
			limit:                     5,
			injectVersionsPerProvider: 6,
			exceedsLimit:              true,
			expectErrorCode:           errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateTerraformProviderPermission, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockProviders := db.MockTerraformProviders{}
			mockProviders.Test(t)

			mockProviderVersions := db.MockTerraformProviderVersions{}
			mockProviderVersions.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockResourceLimits := db.NewMockResourceLimits(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockProviders.On("GetProviderByID", mock.Anything, providerID).Return(&models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID: providerID,
				},
				GroupID:      groupID,
				ResourcePath: "testgroup/testprovider",
			}, nil)

			providerVersionsResult := db.ProviderVersionsResult{
				ProviderVersions: []models.TerraformProviderVersion{},
			}

			if test.latestProviderVersion != nil {
				providerVersionsResult.ProviderVersions = append(providerVersionsResult.ProviderVersions, *test.latestProviderVersion)
			}

			mockProviderVersions.On("GetProviderVersions", mock.Anything, &db.GetProviderVersionsInput{
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
				},
				Filter: &db.TerraformProviderVersionFilter{
					ProviderID: &providerID,
					Latest:     ptr.Bool(true),
				},
			}).Return(&providerVersionsResult, nil)

			if test.expectUpdatedProviderVersion != nil {
				mockProviderVersions.On("UpdateProviderVersion", mock.Anything, test.expectUpdatedProviderVersion).
					Return(test.expectUpdatedProviderVersion, nil)
			}

			mockProviderVersions.On("CreateProviderVersion", mock.Anything, mock.Anything).
				Return(test.expectCreatedProviderVersion, nil)

			mockGroups.On("GetGroupByID", mock.Anything, mock.Anything).Return(&models.Group{
				FullPath: "testGroupFullPath",
			}, nil)

			dbClient := db.Client{
				Transactions:              &mockTransactions,
				TerraformProviders:        &mockProviders,
				TerraformProviderVersions: &mockProviderVersions,
				Groups:                    &mockGroups,
				ResourceLimits:            mockResourceLimits,
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockProviderVersions.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.GetProviderVersionsInput{
					Filter: &db.TerraformProviderVersionFilter{
						ProviderID: &test.input.ProviderID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetProviderVersionsInput) *db.ProviderVersionsResult {
					_ = ctx
					_ = input

					return &db.ProviderVersionsResult{PageInfo: &pagination.PageInfo{
						TotalCount: test.injectVersionsPerProvider,
					},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, &mockActivityEvents)

			providerVersion, err := service.CreateProviderVersion(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedProviderVersion.SemanticVersion, providerVersion.SemanticVersion)
			assert.Equal(t, test.expectCreatedProviderVersion.Latest, providerVersion.Latest)
		})
	}
}

func TestDeleteProviderVersion(t *testing.T) {
	providerID := "provider123"
	groupID := "group123"

	// Test cases
	tests := []struct {
		expectUpdatedProviderVersion *models.TerraformProviderVersion
		name                         string
		existingProviderVersions     []models.TerraformProviderVersion
		providerVersionToDelete      models.TerraformProviderVersion
	}{
		{
			name: "version to delete is the latest version",
			providerVersionToDelete: models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ProviderID:      providerID,
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			existingProviderVersions: []models.TerraformProviderVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.0",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.0-pre",
					Latest:          false,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ProviderID:      providerID,
					SemanticVersion: "0.9.0",
					Latest:          false,
				},
			},
			expectUpdatedProviderVersion: &models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "2"},
				ProviderID:      providerID,
				SemanticVersion: "0.9.0",
				Latest:          true,
			},
		},
		{
			name: "version to delete is not the latest version",
			providerVersionToDelete: models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ProviderID:      providerID,
				SemanticVersion: "1.0.0",
				Latest:          false,
			},
			existingProviderVersions: []models.TerraformProviderVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.0",
					Latest:          false,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.1",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ProviderID:      providerID,
					SemanticVersion: "0.9.0",
					Latest:          false,
				},
			},
		},
		{
			name: "version to delete is the latest version and a pre-release",
			providerVersionToDelete: models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ProviderID:      providerID,
				SemanticVersion: "1.0.0-pre.2",
				Latest:          true,
			},
			existingProviderVersions: []models.TerraformProviderVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.0-pre.2",
					Latest:          true,
				},
				{
					Metadata:        models.ResourceMetadata{ID: "2"},
					ProviderID:      providerID,
					SemanticVersion: "1.0.0-pre.1",
					Latest:          false,
				},
			},
			expectUpdatedProviderVersion: &models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "2"},
				ProviderID:      providerID,
				SemanticVersion: "1.0.0-pre.1",
				Latest:          true,
			},
		},
		{
			name: "version to delete is the only version",
			providerVersionToDelete: models.TerraformProviderVersion{
				Metadata:        models.ResourceMetadata{ID: "1"},
				ProviderID:      providerID,
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
			existingProviderVersions: []models.TerraformProviderVersion{
				{
					Metadata:        models.ResourceMetadata{ID: "1"},
					ProviderID:      providerID,
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

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateTerraformProviderPermission, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockProviders := db.MockTerraformProviders{}
			mockProviders.Test(t)

			mockProviderVersions := db.MockTerraformProviderVersions{}
			mockProviderVersions.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockProviders.On("GetProviderByID", mock.Anything, providerID).Return(&models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID: providerID,
				},
				GroupID: groupID,
			}, nil)

			providerVersionsResult := db.ProviderVersionsResult{
				ProviderVersions: test.existingProviderVersions,
			}

			mockProviderVersions.On("GetProviderVersions", mock.Anything, &db.GetProviderVersionsInput{
				Filter: &db.TerraformProviderVersionFilter{
					ProviderID: &providerID,
				},
			}).Return(&providerVersionsResult, nil)

			if test.expectUpdatedProviderVersion != nil {
				mockProviderVersions.On("UpdateProviderVersion", mock.Anything, test.expectUpdatedProviderVersion).
					Return(test.expectUpdatedProviderVersion, nil)
			}

			mockProviderVersions.On("DeleteProviderVersion", mock.Anything, &test.providerVersionToDelete).Return(nil)

			dbClient := db.Client{
				Transactions:              &mockTransactions,
				TerraformProviders:        &mockProviders,
				TerraformProviderVersions: &mockProviderVersions,
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, &mockActivityEvents)

			err := service.DeleteProviderVersion(auth.WithCaller(ctx, &mockCaller), &test.providerVersionToDelete)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateProviderPlatform(t *testing.T) {
	providerID := "provider123"
	providerVersionID := "provider-version-123"
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError                     error
		expectCreatedProviderPlatform *models.TerraformProviderPlatform
		name                          string
		expectErrCode                 errors.CodeType
		input                         CreateProviderPlatformInput
		limit                         int
		injectPlatformsPerProvider    int32
		shouldDoTx                    bool
		exceedsLimit                  bool
	}{
		{
			name: "subject does not have deployer role",
			input: CreateProviderPlatformInput{
				ProviderVersionID: providerID,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "should create provider platform",
			input: CreateProviderPlatformInput{
				ProviderVersionID: providerID,
				OperatingSystem:   "some-os",
				Architecture:      "some-arch",
				SHASum:            "some-sum",
				Filename:          "some-filename",
			},
			expectCreatedProviderPlatform: &models.TerraformProviderPlatform{
				ProviderVersionID: providerID,
				OperatingSystem:   "some-os",
				Architecture:      "some-arch",
				SHASum:            "some-sum",
				Filename:          "some-filename",
				CreatedBy:         "mockSubject",
				BinaryUploaded:    false,
			},
			limit:                      5,
			injectPlatformsPerProvider: 5,
			shouldDoTx:                 true,
		},
		{
			name: "exceeds limit",
			input: CreateProviderPlatformInput{
				ProviderVersionID: providerID,
				OperatingSystem:   "some-os",
				Architecture:      "some-arch",
				SHASum:            "some-sum",
				Filename:          "some-filename",
			},
			expectCreatedProviderPlatform: &models.TerraformProviderPlatform{
				ProviderVersionID: providerID,
				OperatingSystem:   "some-os",
				Architecture:      "some-arch",
				SHASum:            "some-sum",
				Filename:          "some-filename",
				CreatedBy:         "mockSubject",
				BinaryUploaded:    false,
			},
			limit:                      5,
			injectPlatformsPerProvider: 6,
			shouldDoTx:                 true,
			exceedsLimit:               true,
			expectErrCode:              errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateTerraformProviderPermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject")

			mockProviders := db.NewMockTerraformProviders(t)
			mockProviderVersions := db.NewMockTerraformProviderVersions(t)
			mockProviderPlatforms := db.NewMockTerraformProviderPlatforms(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockProviders.On("GetProviderByID", mock.Anything, providerID).Return(&models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID: providerID,
				},
				GroupID:      groupID,
				ResourcePath: "testgroup/testprovider",
			}, nil)

			mockProviderVersions.On("GetProviderVersionByID", mock.Anything, mock.Anything).Return(&models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID: providerVersionID,
				},
				ProviderID: providerID,
			}, nil)

			mockActivityEvents := activityevent.NewMockService(t)

			if test.expectErrCode == "" || test.exceedsLimit {
				mockProviderPlatforms.On("CreateProviderPlatform", mock.Anything, test.expectCreatedProviderPlatform).
					Return(test.expectCreatedProviderPlatform, nil)
			}

			if test.shouldDoTx {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockProviderPlatforms.On("GetProviderPlatforms", mock.Anything, mock.Anything).Return(&db.GetProviderPlatformsInput{
					Filter: &db.TerraformProviderPlatformFilter{
						ProviderVersionID: &test.input.ProviderVersionID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetProviderPlatformsInput) *db.ProviderPlatformsResult {
					_ = ctx
					_ = input

					return &db.ProviderPlatformsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectPlatformsPerProvider,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			dbClient := db.Client{
				TerraformProviders:         mockProviders,
				TerraformProviderVersions:  mockProviderVersions,
				TerraformProviderPlatforms: mockProviderPlatforms,
				Transactions:               mockTransactions,
				ResourceLimits:             mockResourceLimits,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, mockActivityEvents)

			providerPlatform, err := service.CreateProviderPlatform(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedProviderPlatform.ProviderVersionID, providerPlatform.ProviderVersionID)
			assert.Equal(t, test.expectCreatedProviderPlatform.OperatingSystem, providerPlatform.OperatingSystem)
			assert.Equal(t, test.expectCreatedProviderPlatform.Architecture, providerPlatform.Architecture)
			assert.Equal(t, test.expectCreatedProviderPlatform.SHASum, providerPlatform.SHASum)
			assert.Equal(t, test.expectCreatedProviderPlatform.Filename, providerPlatform.Filename)
			assert.Equal(t, test.expectCreatedProviderPlatform.CreatedBy, providerPlatform.CreatedBy)
			assert.Equal(t, test.expectCreatedProviderPlatform.BinaryUploaded, providerPlatform.BinaryUploaded)
		})
	}
}
