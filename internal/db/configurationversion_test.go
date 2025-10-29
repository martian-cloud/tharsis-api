//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for ConfigurationVersionSortableField
func (cv ConfigurationVersionSortableField) getValue() string {
	return string(cv)
}

func TestConfigurationVersions_CreateConfigurationVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-version",
		Description: "test group for config version",
		FullPath:    "test-group-config-version",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-version",
		Description:    "test workspace for config version",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		configVersion   models.ConfigurationVersion
	}

	testCases := []testCase{
		{
			name: "successfully create configuration version",
			configVersion: models.ConfigurationVersion{
				WorkspaceID: workspace.Metadata.ID,
				Status:      models.ConfigurationPending,
				CreatedBy:   "db-integration-tests",
				Speculative: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configVersion, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, test.configVersion)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, configVersion)
			assert.Equal(t, test.configVersion.WorkspaceID, configVersion.WorkspaceID)
			assert.Equal(t, test.configVersion.Status, configVersion.Status)
			assert.Equal(t, test.configVersion.Speculative, configVersion.Speculative)
		})
	}
}

func TestConfigurationVersions_UpdateConfigurationVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-version-update",
		Description: "test group for config version update",
		FullPath:    "test-group-config-version-update",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-version-update",
		Description:    "test workspace for config version update",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a configuration version to update
	createdConfigVersion, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.ConfigurationPending,
		CreatedBy:   "db-integration-tests",
		Speculative: false,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		updateConfig    models.ConfigurationVersion
	}

	testCases := []testCase{
		{
			name: "successfully update configuration version",
			updateConfig: models.ConfigurationVersion{
				Metadata:    createdConfigVersion.Metadata,
				WorkspaceID: createdConfigVersion.WorkspaceID,
				Status:      models.ConfigurationUploaded,
				CreatedBy:   createdConfigVersion.CreatedBy,
				Speculative: createdConfigVersion.Speculative,
			},
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			updateConfig: models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:      createdConfigVersion.Metadata.ID,
					Version: -1,
				},
				WorkspaceID: createdConfigVersion.WorkspaceID,
				Status:      models.ConfigurationUploaded,
				CreatedBy:   createdConfigVersion.CreatedBy,
				Speculative: createdConfigVersion.Speculative,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configVersion, err := testClient.client.ConfigurationVersions.UpdateConfigurationVersion(ctx, test.updateConfig)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, configVersion)
			assert.Equal(t, test.updateConfig.Status, configVersion.Status)
		})
	}
}

func TestConfigurationVersions_GetConfigurationVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-version-get-by-id",
		Description: "test group for config version get by id",
		FullPath:    "test-group-config-version-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-version-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a configuration version for testing
	createdConfigVersion, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Status:      models.ConfigurationUploaded,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode     errors.CodeType
		name                string
		id                  string
		expectConfigVersion bool
	}

	testCases := []testCase{
		{
			name:                "get resource by id",
			id:                  createdConfigVersion.Metadata.ID,
			expectConfigVersion: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configVersion, err := testClient.client.ConfigurationVersions.GetConfigurationVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectConfigVersion {
				require.NotNil(t, configVersion)
				assert.Equal(t, test.id, configVersion.Metadata.ID)
			} else {
				assert.Nil(t, configVersion)
			}
		})
	}
}

func TestConfigurationVersions_GetConfigurationVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-versions-list",
		Description: "test group for config versions list",
		FullPath:    "test-group-config-versions-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-versions-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test configuration versions
	configVersions := []models.ConfigurationVersion{
		{
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
			Status:      models.ConfigurationUploaded,
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
			Status:      models.ConfigurationPending,
		},
	}

	createdConfigVersions := []models.ConfigurationVersion{}
	for _, configVersion := range configVersions {
		created, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, configVersion)
		require.NoError(t, err)
		createdConfigVersions = append(createdConfigVersions, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetConfigurationVersionsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all configuration versions",
			input:       &GetConfigurationVersionsInput{},
			expectCount: len(createdConfigVersions),
		},
		{
			name: "filter by workspace ID",
			input: &GetConfigurationVersionsInput{
				Filter: &ConfigurationVersionFilter{
					WorkspaceID: &workspace.Metadata.ID,
				},
			},
			expectCount: len(createdConfigVersions),
		},
		{
			name: "filter by configuration version IDs",
			input: &GetConfigurationVersionsInput{
				Filter: &ConfigurationVersionFilter{
					ConfigurationVersionIDs: []string{createdConfigVersions[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.ConfigurationVersions.GetConfigurationVersions(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ConfigurationVersions, test.expectCount)
		})
	}
}

func TestConfigurationVersions_GetConfigurationVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-versions-pagination",
		Description: "test group for config versions pagination",
		FullPath:    "test-group-config-versions-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-versions-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
			Status:      models.ConfigurationUploaded,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		ConfigurationVersionSortableFieldUpdatedAtAsc,
		ConfigurationVersionSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := ConfigurationVersionSortableField(sortByField.getValue())

		result, err := testClient.client.ConfigurationVersions.GetConfigurationVersions(ctx, &GetConfigurationVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ConfigurationVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestConfigurationVersions_GetConfigurationVersionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-config-version-get-by-trn",
		Description: "test group for config version get by trn",
		FullPath:    "test-group-config-version-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-config-version-get-by-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a configuration version for testing
	createdConfigVersion, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Status:      models.ConfigurationUploaded,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode     errors.CodeType
		name                string
		trn                 string
		expectConfigVersion bool
	}

	testCases := []testCase{
		{
			name:                "get resource by TRN",
			trn:                 createdConfigVersion.Metadata.TRN,
			expectConfigVersion: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:configuration_version:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configVersion, err := testClient.client.ConfigurationVersions.GetConfigurationVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectConfigVersion {
				require.NotNil(t, configVersion)
				assert.Equal(t, createdConfigVersion.Metadata.ID, configVersion.Metadata.ID)
			} else {
				assert.Nil(t, configVersion)
			}
		})
	}
}
