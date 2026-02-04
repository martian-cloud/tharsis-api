//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for TerraformProviderVersionMirrorSortableField
func (tpvm TerraformProviderVersionMirrorSortableField) getValue() string {
	return string(tpvm)
}

func TestTerraformProviderVersionMirrors_CreateVersionMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-mirror",
		Description: "test group for provider version mirror",
		FullPath:    "test-group-provider-version-mirror",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		versionMirror   *models.TerraformProviderVersionMirror
	}

	testCases := []testCase{
		{
			name: "successfully create version mirror",
			versionMirror: &models.TerraformProviderVersionMirror{
				GroupID:           group.Metadata.ID,
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "5.0.0",
				CreatedBy:         "db-integration-tests",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			versionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, test.versionMirror)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, versionMirror)
			assert.Equal(t, test.versionMirror.RegistryHostname, versionMirror.RegistryHostname)
			assert.Equal(t, test.versionMirror.Type, versionMirror.Type)
			assert.Equal(t, test.versionMirror.SemanticVersion, versionMirror.SemanticVersion)
		})
	}
}

func TestTerraformProviderVersionMirrors_DeleteVersionMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-mirror-delete",
		Description: "test group for provider version mirror delete",
		FullPath:    "test-group-provider-version-mirror-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror to delete
	createdVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
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
		versionMirror   *models.TerraformProviderVersionMirror
	}

	testCases := []testCase{
		{
			name:          "successfully delete version mirror",
			versionMirror: createdVersionMirror,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviderVersionMirrors.DeleteVersionMirror(ctx, test.versionMirror)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify the version mirror was deleted
			deletedVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrorByID(ctx, test.versionMirror.Metadata.ID)
			if err != nil {
				// Version mirror should not be found after deletion
				assert.Nil(t, deletedVersionMirror)
			}
		})
	}
}

func TestTerraformProviderVersionMirrors_GetVersionMirrorByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the provider version mirror
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-version-mirror",
		Description: "test group for version mirror",
		FullPath:    "test-group-version-mirror",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	createdMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		SemanticVersion:   "1.0.0",
		Type:              "aws",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectMirror    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdMirror.Metadata.ID,
			expectMirror: true,
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
			mirror, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrorByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectMirror {
				require.NotNil(t, mirror)
				assert.Equal(t, test.id, mirror.Metadata.ID)
			} else {
				assert.Nil(t, mirror)
			}
		})
	}
}

func TestTerraformProviderVersionMirrors_GetVersionMirrors(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the provider version mirrors
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-version-mirrors",
		Description: "test group for version mirrors",
		FullPath:    "test-group-version-mirrors",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test version mirrors
	mirrors := []models.TerraformProviderVersionMirror{
		{
			SemanticVersion:   "1.0.0",
			Type:              "aws",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			GroupID:           group.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		},
		{
			SemanticVersion:   "1.1.0",
			Type:              "aws",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			GroupID:           group.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		},
	}

	createdMirrors := []models.TerraformProviderVersionMirror{}
	for _, mirror := range mirrors {
		created, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &mirror)
		require.NoError(t, err)
		createdMirrors = append(createdMirrors, *created)
	}

	// Create a platform mirror for the first version mirror to test HasPackages filter
	_, err = testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
		VersionMirrorID: createdMirrors[0].Metadata.ID,
		OS:              "linux",
		Architecture:    "amd64",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetProviderVersionMirrorsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all mirrors",
			input:       &GetProviderVersionMirrorsInput{},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by group ID",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					GroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by registry hostname",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryHostname: &createdMirrors[0].RegistryHostname,
				},
			},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by registry namespace",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryNamespace: &createdMirrors[0].RegistryNamespace,
				},
			},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by type",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					Type: &createdMirrors[0].Type,
				},
			},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by semantic version",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					SemanticVersion: &createdMirrors[0].SemanticVersion,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by version mirror IDs",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					VersionMirrorIDs: []string{createdMirrors[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by namespace paths",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdMirrors),
		},
		{
			name: "filter by has packages true",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					HasPackages: ptr.Bool(true),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by has packages false",
			input: &GetProviderVersionMirrorsInput{
				Filter: &TerraformProviderVersionMirrorFilter{
					HasPackages: ptr.Bool(false),
				},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.VersionMirrors, test.expectCount)
		})
	}
}

func TestTerraformProviderVersionMirrors_GetVersionMirrorsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the provider version mirrors
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-version-pagination",
		Description: "test group for version pagination",
		FullPath:    "test-group-version-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
			SemanticVersion:   fmt.Sprintf("1.%d.0", i),
			Type:              "aws",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			GroupID:           group.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformProviderVersionMirrorSortableFieldCreatedAtAsc,
		TerraformProviderVersionMirrorSortableFieldCreatedAtDesc,
		TerraformProviderVersionMirrorSortableFieldTypeAsc,
		TerraformProviderVersionMirrorSortableFieldTypeDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformProviderVersionMirrorSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &GetProviderVersionMirrorsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.VersionMirrors {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformProviderVersionMirrors_GetVersionMirrorByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the provider version mirror
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-version-trn",
		Description: "test group for version trn",
		FullPath:    "test-group-version-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a version mirror for testing
	createdMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		SemanticVersion:   "1.0.0",
		Type:              "aws",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		GroupID:           group.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectMirror    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdMirror.Metadata.TRN,
			expectMirror: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform-provider-version-mirror:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mirror, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrorByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectMirror {
				require.NotNil(t, mirror)
				assert.Equal(t, test.trn, mirror.Metadata.TRN)
			} else {
				assert.Nil(t, mirror)
			}
		})
	}
}
