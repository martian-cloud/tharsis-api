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

// getValue implements the sortableField interface for PackageVersionSortableField
func (ps PackageVersionSortableField) getValue() string {
	return string(ps)
}

// createTestPackageForVersion creates a group and package for use as a package version's parent.
func createTestPackageForVersion(ctx context.Context, t *testing.T, testClient *testClient, groupName string) *models.Package {
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      groupName,
		FullPath:  groupName,
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	pkg, err := testClient.client.Packages.CreatePackage(ctx, &models.Package{
		Name:        groupName + "-pkg",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Kind:        models.PackageKindOPAPolicy,
		Visibility:  models.PackageVisibilityPrivate,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	return pkg
}

func TestPackageVersions_CreatePackageVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion")

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		packageID       string
		semanticVersion string
	}

	testCases := []testCase{
		{
			name:            "create package version",
			packageID:       pkg.Metadata.ID,
			semanticVersion: "1.0.0",
		},
		{
			name:            "negative, package does not exist",
			packageID:       nonExistentID,
			semanticVersion: "1.0.0",
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			packageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
				PackageID:       test.packageID,
				SemanticVersion: test.semanticVersion,
				Status:          models.PackageVersionStatusPending,
				CreatedBy:       "db-integration-tests",
				SHASum:          []byte("test-sha-sum"),
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, packageVersion)

			assert.Equal(t, test.packageID, packageVersion.PackageID)
			assert.Equal(t, test.semanticVersion, packageVersion.SemanticVersion)
			assert.Equal(t, models.PackageVersionStatusPending, packageVersion.Status)
			assert.NotEmpty(t, packageVersion.Metadata.ID)
		})
	}
}

func TestPackageVersions_CreatePackageVersion_DuplicateSemanticVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-dup")

	_, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.NoError(t, err)

	_, err = testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
}

func TestPackageVersions_UpdatePackageVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-update")

	createdPackageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.PackageVersionStatus
	}

	testCases := []testCase{
		{
			name:    "update package version",
			version: createdPackageVersion.Metadata.Version,
			status:  models.PackageVersionStatusUploaded,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.PackageVersionStatusErrored,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			packageVersionToUpdate := *createdPackageVersion
			packageVersionToUpdate.Metadata.Version = test.version
			packageVersionToUpdate.Status = test.status

			updatedPackageVersion, err := testClient.client.PackageVersions.UpdatePackageVersion(ctx, &packageVersionToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedPackageVersion)

			assert.Equal(t, test.status, updatedPackageVersion.Status)
			assert.Equal(t, createdPackageVersion.Metadata.Version+1, updatedPackageVersion.Metadata.Version)
		})
	}
}

func TestPackageVersions_DeletePackageVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-delete")

	createdPackageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete package version",
			id:      createdPackageVersion.Metadata.ID,
			version: createdPackageVersion.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdPackageVersion.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.PackageVersions.DeletePackageVersion(ctx, &models.PackageVersion{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			packageVersion, err := testClient.client.PackageVersions.GetPackageVersionByID(ctx, test.id)
			assert.Nil(t, packageVersion)
			assert.Nil(t, err)
		})
	}
}

func TestPackageVersions_GetPackageVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-get-by-id")

	createdPackageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		id                   string
		expectPackageVersion bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by id",
			id:                   createdPackageVersion.Metadata.ID,
			expectPackageVersion: true,
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
			packageVersion, err := testClient.client.PackageVersions.GetPackageVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPackageVersion {
				require.NotNil(t, packageVersion)
				assert.Equal(t, test.id, packageVersion.Metadata.ID)
			} else {
				assert.Nil(t, packageVersion)
			}
		})
	}
}

func TestPackageVersions_GetPackageVersionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-trn")

	createdPackageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		trn                  string
		expectPackageVersion bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by TRN",
			trn:                  createdPackageVersion.Metadata.TRN,
			expectPackageVersion: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:package_version:test-group-pkgversion-trn/test-group-pkgversion-trn-pkg/9.9.9",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			packageVersion, err := testClient.client.PackageVersions.GetPackageVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPackageVersion {
				require.NotNil(t, packageVersion)
				assert.Equal(t, test.trn, packageVersion.Metadata.TRN)
			} else {
				assert.Nil(t, packageVersion)
			}
		})
	}
}

func TestPackageVersions_GetPackageVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-list")
	otherPkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-list-other")

	createdPackageVersion, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       pkg.Metadata.ID,
		SemanticVersion: "1.0.0",
		Status:          models.PackageVersionStatusUploaded,
		Latest:          true,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.NoError(t, err)

	_, err = testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
		PackageID:       otherPkg.Metadata.ID,
		SemanticVersion: "2.0.0",
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       "db-integration-tests",
		SHASum:          []byte("test-sha-sum"),
	})
	require.NoError(t, err)

	pendingStatus := models.PackageVersionStatusPending

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetPackageVersionsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all package versions",
			input:       &GetPackageVersionsInput{},
			expectCount: 2,
		},
		{
			name: "get package versions filtered by package id",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{PackageID: &pkg.Metadata.ID},
			},
			expectCount: 1,
		},
		{
			name: "get package versions filtered by status",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{Status: &pendingStatus},
			},
			expectCount: 1,
		},
		{
			name: "get package versions filtered by semantic version",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{SemanticVersion: &createdPackageVersion.SemanticVersion},
			},
			expectCount: 1,
		},
		{
			name: "get package versions filtered by latest",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{Latest: ptr.Bool(true)},
			},
			expectCount: 1,
		},
		{
			name: "get package versions filtered by search",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{Search: ptr.String("1.0")},
			},
			expectCount: 1,
		},
		{
			name: "get package versions filtered by package version ids",
			input: &GetPackageVersionsInput{
				Filter: &PackageVersionFilter{PackageVersionIDs: []string{createdPackageVersion.Metadata.ID}},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.PackageVersions.GetPackageVersions(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.PackageVersions, test.expectCount)
		})
	}
}

func TestPackageVersions_GetPackageVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	pkg := createTestPackageForVersion(ctx, t, testClient, "test-group-pkgversion-pagination")

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.PackageVersions.CreatePackageVersion(ctx, &models.PackageVersion{
			PackageID:       pkg.Metadata.ID,
			SemanticVersion: fmt.Sprintf("1.0.%d", i),
			Status:          models.PackageVersionStatusPending,
			CreatedBy:       "db-integration-tests",
			SHASum:          []byte("test-sha-sum"),
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		PackageVersionSortableFieldCreatedAtAsc,
		PackageVersionSortableFieldCreatedAtDesc,
		PackageVersionSortableFieldUpdatedAtAsc,
		PackageVersionSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := PackageVersionSortableField(sortByField.getValue())

		result, err := testClient.client.PackageVersions.GetPackageVersions(ctx, &GetPackageVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.PackageVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
