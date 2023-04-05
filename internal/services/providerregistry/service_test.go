package providerregistry

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

func TestCreateProviderVersion(t *testing.T) {
	providerID := "provider123"
	groupID := "group123"

	// Test cases
	tests := []struct {
		latestProviderVersion        *models.TerraformProviderVersion
		expectUpdatedProviderVersion *models.TerraformProviderVersion
		expectCreatedProviderVersion *models.TerraformProviderVersion
		name                         string
		input                        CreateProviderVersionInput
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
				SemanticVersion: "0.1.0",
				Latest:          true,
			},
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
				SemanticVersion: "1.0.0-pre",
				Latest:          false,
			},
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
				SemanticVersion: "1.0.0-pre",
				Latest:          true,
			},
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
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
		},
		{
			name: "no current latest and new version is not a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0",
				ProviderID:      providerID,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0",
				Latest:          true,
			},
		},
		{
			name: "no current latest and new version is a pre-release",
			input: CreateProviderVersionInput{
				SemanticVersion: "1.0.0-pre",
				ProviderID:      providerID,
			},
			expectCreatedProviderVersion: &models.TerraformProviderVersion{
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
				PaginationOptions: &db.PaginationOptions{
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
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, &mockActivityEvents)

			providerVersion, err := service.CreateProviderVersion(auth.WithCaller(ctx, &mockCaller), &test.input)
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

			service := NewService(testLogger, &dbClient, nil, &mockActivityEvents)

			err := service.DeleteProviderVersion(auth.WithCaller(ctx, &mockCaller), &test.providerVersionToDelete)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
