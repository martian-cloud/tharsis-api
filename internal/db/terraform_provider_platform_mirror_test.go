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

// getValue implements the sortableField interface for TerraformProviderPlatformMirrorSortableField
func (tppm TerraformProviderPlatformMirrorSortableField) getValue() string {
	return string(tppm)
}

func TestTerraformProviderPlatformMirrors_CreateTerraformProviderPlatformMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-mirror",
		Description: "test group for provider platform mirror",
		FullPath:    "test-group-provider-platform-mirror",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror first (required dependency)
	versionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		GroupID:           group.Metadata.ID,
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		SemanticVersion:   "5.0.0",
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		platformMirror  *models.TerraformProviderPlatformMirror
	}

	testCases := []testCase{
		{
			name: "successfully create platform mirror",
			platformMirror: &models.TerraformProviderPlatformMirror{
				OS:              "linux",
				Architecture:    "amd64",
				VersionMirrorID: versionMirror.Metadata.ID,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			platformMirror, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, test.platformMirror)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, platformMirror)
			assert.Equal(t, test.platformMirror.OS, platformMirror.OS)
			assert.Equal(t, test.platformMirror.Architecture, platformMirror.Architecture)
			assert.Equal(t, test.platformMirror.VersionMirrorID, platformMirror.VersionMirrorID)
		})
	}
}

func TestTerraformProviderPlatformMirrors_DeleteTerraformProviderPlatformMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-mirror-delete",
		Description: "test group for provider platform mirror delete",
		FullPath:    "test-group-provider-platform-mirror-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror first (required dependency)
	versionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		GroupID:           group.Metadata.ID,
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		SemanticVersion:   "5.0.0",
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a platform mirror to delete
	createdPlatformMirror, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
		OS:              "linux",
		Architecture:    "amd64",
		VersionMirrorID: versionMirror.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		platformMirror  *models.TerraformProviderPlatformMirror
	}

	testCases := []testCase{
		{
			name:           "successfully delete platform mirror",
			platformMirror: createdPlatformMirror,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviderPlatformMirrors.DeletePlatformMirror(ctx, test.platformMirror)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify the platform mirror was deleted
			deletedPlatformMirror, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrorByID(ctx, test.platformMirror.Metadata.ID)
			if err != nil {
				assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
			}
			assert.Nil(t, deletedPlatformMirror)
		})
	}
}

func TestTerraformProviderPlatformMirrors_GetPlatformMirrorByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-platform-mirror-get-by-id",
		Description: "test group for platform mirror get by id",
		FullPath:    "test-group-platform-mirror-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	terraformProviderVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		Type:              "registry",
		SemanticVersion:   "1.0.0",
		RegistryNamespace: "test-namespace",
		RegistryHostname:  "registry.terraform.io",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a platform mirror for testing
	createdPlatformMirror, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
		VersionMirrorID: terraformProviderVersionMirror.Metadata.ID,
		OS:              "linux",
		Architecture:    "amd64",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		id                   string
		expectPlatformMirror bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by id",
			id:                   createdPlatformMirror.Metadata.ID,
			expectPlatformMirror: true,
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
			platformMirror, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrorByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPlatformMirror {
				require.NotNil(t, platformMirror)
				assert.Equal(t, test.id, platformMirror.Metadata.ID)
			} else {
				assert.Nil(t, platformMirror)
			}
		})
	}
}

func TestTerraformProviderPlatformMirrors_GetPlatformMirrors(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-platform-mirrors-list",
		Description: "test group for platform mirrors list",
		FullPath:    "test-group-platform-mirrors-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	terraformProviderVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		Type:              "registry",
		SemanticVersion:   "1.0.0",
		RegistryNamespace: "test-namespace-list",
		RegistryHostname:  "registry.terraform.io",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test platform mirrors
	platformMirrors := []models.TerraformProviderPlatformMirror{
		{
			VersionMirrorID: terraformProviderVersionMirror.Metadata.ID,
			OS:              "linux",
			Architecture:    "amd64",
		},
		{
			VersionMirrorID: terraformProviderVersionMirror.Metadata.ID,
			OS:              "darwin",
			Architecture:    "amd64",
		},
	}

	createdPlatformMirrors := []models.TerraformProviderPlatformMirror{}
	for _, platformMirror := range platformMirrors {
		created, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &platformMirror)
		require.NoError(t, err)
		createdPlatformMirrors = append(createdPlatformMirrors, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetProviderPlatformMirrorsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all platform mirrors",
			input:       &GetProviderPlatformMirrorsInput{},
			expectCount: len(createdPlatformMirrors),
		},
		{
			name: "filter by version mirror ID",
			input: &GetProviderPlatformMirrorsInput{
				Filter: &TerraformProviderPlatformMirrorFilter{
					VersionMirrorID: &terraformProviderVersionMirror.Metadata.ID,
				},
			},
			expectCount: len(createdPlatformMirrors),
		},
		{
			name: "filter by OS",
			input: &GetProviderPlatformMirrorsInput{
				Filter: &TerraformProviderPlatformMirrorFilter{
					OS: ptr.String("linux"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by architecture",
			input: &GetProviderPlatformMirrorsInput{
				Filter: &TerraformProviderPlatformMirrorFilter{
					Architecture: ptr.String("amd64"),
				},
			},
			expectCount: len(createdPlatformMirrors),
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.PlatformMirrors, test.expectCount)
		})
	}
}

func TestTerraformProviderPlatformMirrors_GetPlatformMirrorsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-platform-mirrors-pagination",
		Description: "test group for platform mirrors pagination",
		FullPath:    "test-group-platform-mirrors-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	terraformProviderVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		Type:              "registry",
		SemanticVersion:   "1.0.0",
		RegistryNamespace: "test-namespace-pagination",
		RegistryHostname:  "registry.terraform.io",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	// Create unique OS/Architecture combinations
	osArchCombinations := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
		{"freebsd", "amd64"},
		{"freebsd", "arm64"},
		{"openbsd", "amd64"},
		{"netbsd", "amd64"},
	}

	resourceCount := len(osArchCombinations)
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
			VersionMirrorID: terraformProviderVersionMirror.Metadata.ID,
			OS:              osArchCombinations[i].os,
			Architecture:    osArchCombinations[i].arch,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc,
		TerraformProviderPlatformMirrorSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformProviderPlatformMirrorSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, &GetProviderPlatformMirrorsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.PlatformMirrors {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformProviderPlatformMirrors_GetPlatformMirrorByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-platform-mirror-get-by-trn",
		Description: "test group for platform mirror get by trn",
		FullPath:    "test-group-platform-mirror-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	terraformProviderVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		Type:              "registry",
		SemanticVersion:   "1.0.0",
		RegistryNamespace: "test-namespace-trn",
		RegistryHostname:  "registry.terraform.io",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a platform mirror for testing
	createdPlatformMirror, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
		VersionMirrorID: terraformProviderVersionMirror.Metadata.ID,
		OS:              "linux",
		Architecture:    "amd64",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		trn                  string
		expectPlatformMirror bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by TRN",
			trn:                  createdPlatformMirror.Metadata.TRN,
			expectPlatformMirror: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform_provider_platform_mirror:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			platformMirror, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrorByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPlatformMirror {
				require.NotNil(t, platformMirror)
				assert.Equal(t, createdPlatformMirror.Metadata.ID, platformMirror.Metadata.ID)
			} else {
				assert.Nil(t, platformMirror)
			}
		})
	}
}
