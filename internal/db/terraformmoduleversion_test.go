//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// terraformModuleVersionInfo aids convenience in accessing the information
// TestGetModuleVersions needs about the warmup objects.
type terraformModuleVersionInfo struct {
	updateTime time.Time
	id         string
}

// terraformModuleVersionInfoIDSlice makes a slice of terraformModuleVersionInfo sortable by ID string
type terraformModuleVersionInfoIDSlice []terraformModuleVersionInfo

// terraformModuleVersionInfoUpdateSlice makes a slice of terraformModuleVersionInfo sortable by last updated time
type terraformModuleVersionInfoUpdateSlice []terraformModuleVersionInfo

// warmupTerraformModuleVersions holds the inputs to and outputs from createWarmupTerraformModuleVersions.
type warmupTerraformModuleVersions struct {
	groups                  []models.Group
	terraformModules        []models.TerraformModule
	terraformModuleVersions []models.TerraformModuleVersion
}

func TestGetModuleVersionByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:                  standardWarmupGroupsForTerraformModuleVersions,
		terraformModules:        standardWarmupTerraformModulesForTerraformModulesVersions,
		terraformModuleVersions: standardWarmupTerraformModuleVersions,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg                    *string
		expectTerraformModuleVersion *models.TerraformModuleVersion
		name                         string
		searchID                     string
	}

	testCases := []testCase{
		{
			name:                         "get module version by ID",
			searchID:                     warmupItems.terraformModuleVersions[0].Metadata.ID,
			expectTerraformModuleVersion: &warmupItems.terraformModuleVersions[0],
		},

		{
			name:     "returns nil because module version does not exist",
			searchID: nonExistentID,
		},

		{
			name:      "returns an error because the module version ID is invalid",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformModuleVersion, err := testClient.client.TerraformModuleVersions.GetModuleVersionByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformModuleVersion != nil {
				require.NotNil(t, actualTerraformModuleVersion)
				assert.Equal(t, test.expectTerraformModuleVersion, actualTerraformModuleVersion)
			} else {
				assert.Nil(t, actualTerraformModuleVersion)
			}
		})
	}
}

func TestGetModuleVersionsWithPagination(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:                  standardWarmupGroupsForTerraformModuleVersions,
		terraformModules:        standardWarmupTerraformModulesForTerraformModulesVersions,
		terraformModuleVersions: standardWarmupTerraformModuleVersions,
	})
	require.Nil(t, err)

	// Query for first page
	middleIndex := len(warmupItems.terraformModuleVersions) / 2
	page1, err := testClient.client.TerraformModuleVersions.GetModuleVersions(ctx, &GetModuleVersionsInput{
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(int32(middleIndex)),
		},
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(page1.ModuleVersions))
	assert.True(t, page1.PageInfo.HasNextPage)
	assert.False(t, page1.PageInfo.HasPreviousPage)

	cursor, err := page1.PageInfo.Cursor(&page1.ModuleVersions[len(page1.ModuleVersions)-1])
	require.Nil(t, err)

	remaining := len(warmupItems.terraformModuleVersions) - middleIndex
	page2, err := testClient.client.TerraformModuleVersions.GetModuleVersions(ctx, &GetModuleVersionsInput{
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(int32(remaining)),
			After: cursor,
		},
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(page2.ModuleVersions))
	assert.True(t, page2.PageInfo.HasPreviousPage)
	assert.False(t, page2.PageInfo.HasNextPage)
}

func TestGetModuleVersions(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:                  standardWarmupGroupsForTerraformModuleVersions,
		terraformModules:        standardWarmupTerraformModulesForTerraformModulesVersions,
		terraformModuleVersions: standardWarmupTerraformModuleVersions,
	})
	require.Nil(t, err)

	allTerraformModuleVersionInfos := terraformModuleVersionInfoFromTerraformModuleVersions(warmupItems.terraformModuleVersions)

	// Sort by Terraform module version IDs.
	sort.Sort(terraformModuleVersionInfoIDSlice(allTerraformModuleVersionInfos))
	allTerraformModuleVersionIDs := terraformModuleVersionIDsFromTerraformModuleVersionInfos(allTerraformModuleVersionInfos)

	// Sort by last update times.
	sort.Sort(terraformModuleVersionInfoUpdateSlice(allTerraformModuleVersionInfos))
	allTerraformModuleVersionIDsByTime := terraformModuleVersionIDsFromTerraformModuleVersionInfos(allTerraformModuleVersionInfos)
	reverseTerraformModuleVersionIDsByTime := reverseStringSlice(allTerraformModuleVersionIDsByTime)

	type testCase struct {
		input                           *GetModuleVersionsInput
		expectMsg                       *string
		name                            string
		expectTerraformModuleVersionIDs []string
	}

	testCases := []testCase{
		{
			name: "non-nil but mostly empty input",
			input: &GetModuleVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformModuleVersionIDs: allTerraformModuleVersionIDs,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformModuleVersionIDs: allTerraformModuleVersionIDsByTime,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
			},
			expectTerraformModuleVersionIDs: allTerraformModuleVersionIDsByTime,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtDesc),
			},
			expectTerraformModuleVersionIDs: reverseTerraformModuleVersionIDsByTime,
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                       ptr.String("only first or last can be defined, not both"),
			expectTerraformModuleVersionIDs: allTerraformModuleVersionIDs[4:],
		},

		{
			name: "filter, terraform module version IDs, positive",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleVersionFilter{
					ModuleVersionIDs: []string{
						allTerraformModuleVersionIDsByTime[0], allTerraformModuleVersionIDsByTime[1], allTerraformModuleVersionIDsByTime[3]},
				},
			},
			expectTerraformModuleVersionIDs: []string{
				allTerraformModuleVersionIDsByTime[0], allTerraformModuleVersionIDsByTime[1], allTerraformModuleVersionIDsByTime[3],
			},
		},

		{
			name: "filter, terraform module version IDs, non-existent",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleVersionFilter{
					ModuleVersionIDs: []string{nonExistentID},
				},
			},
			expectTerraformModuleVersionIDs: []string{},
		},

		{
			name: "filter, terraform module version IDs, invalid ID",
			input: &GetModuleVersionsInput{
				Sort: ptrTerraformModuleVersionSortableField(TerraformModuleVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleVersionFilter{
					ModuleVersionIDs: []string{invalidID},
				},
			},
			expectMsg:                       invalidUUIDMsg2,
			expectTerraformModuleVersionIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			terraformModuleVersionsResult, err := testClient.client.TerraformModuleVersions.GetModuleVersions(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if err == nil {
				// Never returns nil if error is nil.
				require.NotNil(t, terraformModuleVersionsResult.PageInfo)

				terraformModuleVersions := terraformModuleVersionsResult.ModuleVersions

				// Check the terraform moduleVersions result by comparing a list of the terraform module version IDs.
				actualTerraformModuleVersionIDs := []string{}
				for _, terraformModuleVersion := range terraformModuleVersions {
					actualTerraformModuleVersionIDs = append(actualTerraformModuleVersionIDs, terraformModuleVersion.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformModuleVersionIDs)
				}

				assert.Equal(t, len(test.expectTerraformModuleVersionIDs), len(actualTerraformModuleVersionIDs))
				assert.Equal(t, test.expectTerraformModuleVersionIDs, actualTerraformModuleVersionIDs)
			}
		})
	}
}

func TestCreateModuleVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:           standardWarmupGroupsForTerraformModuleVersions,
		terraformModules: standardWarmupTerraformModulesForTerraformModulesVersions,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformModuleVersion
		expectCreated *models.TerraformModuleVersion
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformModuleVersion{
				ModuleID:        warmupItems.terraformModules[0].Metadata.ID,
				SemanticVersion: "1.0.0",
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{"submodule1"},
				Examples:        []string{"example1"},
				Latest:          true,
				CreatedBy:       "TestCreateModuleVersion",
			},
			expectCreated: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ModuleID:        warmupItems.terraformModules[0].Metadata.ID,
				SemanticVersion: "1.0.0",
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{"submodule1"},
				Examples:        []string{"example1"},
				Latest:          true,
				CreatedBy:       "TestCreateModuleVersion",
			},
		},

		{
			name: "duplicate version",
			toCreate: &models.TerraformModuleVersion{
				ModuleID:        warmupItems.terraformModules[0].Metadata.ID,
				SemanticVersion: "1.0.0",
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{"submodule1"},
				Examples:        []string{"example1"},
				Latest:          false,
				CreatedBy:       "TestCreateModuleVersion",
			},
			expectMsg: ptr.String("terraform module version 1.0.0 already exists"),
		},

		{
			name: "duplicate latest",
			toCreate: &models.TerraformModuleVersion{
				ModuleID:        warmupItems.terraformModules[0].Metadata.ID,
				SemanticVersion: "1.0.1",
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{},
				Examples:        []string{},
				Latest:          true,
				CreatedBy:       "TestCreateModuleVersion",
			},
			expectMsg: ptr.String("another terraform module version is already marked as the latest for the same module"),
		},

		{
			name: "module does not exist",
			toCreate: &models.TerraformModuleVersion{
				ModuleID:        nonExistentID,
				SemanticVersion: "0.0.1",
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{"submodule1"},
				Examples:        []string{"example1"},
				Latest:          false,
				CreatedBy:       "TestCreateModuleVersion",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_module_versions\" violates foreign key constraint \"fk_module_id\" (SQLSTATE 23503)"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformModuleVersions(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateModuleVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:                  standardWarmupGroupsForTerraformModuleVersions,
		terraformModules:        standardWarmupTerraformModulesForTerraformModulesVersions,
		terraformModuleVersions: standardWarmupTerraformModuleVersions,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformModuleVersion
		expectUpdated *models.TerraformModuleVersion
		name          string
	}

	now := time.Now()
	testCases := []testCase{

		{
			name: "positive",
			toUpdate: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.terraformModuleVersions[0].Metadata.ID,
					Version: initialResourceVersion,
				},
				Status:     models.TerraformModuleVersionStatusUploaded,
				SHASum:     []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules: []string{"submodule1"},
				Examples:   []string{"example1"},
				Latest:     false,
			},
			expectUpdated: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:                   warmupItems.terraformModuleVersions[0].Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    warmupItems.terraformModuleVersions[0].Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Status:          models.TerraformModuleVersionStatusUploaded,
				SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
				Submodules:      []string{"submodule1"},
				Examples:        []string{"example1"},
				Latest:          false,
				ModuleID:        warmupItems.terraformModuleVersions[0].ModuleID,
				SemanticVersion: warmupItems.terraformModuleVersions[0].SemanticVersion,
				CreatedBy:       warmupItems.terraformModuleVersions[0].CreatedBy,
			},
		},

		{
			name: "negative, non-existent Terraform module version ID",
			toUpdate: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformModuleVersion, err := testClient.client.TerraformModuleVersions.UpdateModuleVersion(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformModuleVersion)
				compareTerraformModuleVersions(t, test.expectUpdated, actualTerraformModuleVersion, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformModuleVersion)
			}
		})
	}
}

func TestDeleteModuleVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleVersions(ctx, testClient, warmupTerraformModuleVersions{
		groups:                  standardWarmupGroupsForTerraformModuleVersions,
		terraformModules:        standardWarmupTerraformModulesForTerraformModulesVersions,
		terraformModuleVersions: standardWarmupTerraformModuleVersions,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformModuleVersion
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.terraformModuleVersions[0].Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform module version ID",
			toDelete: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformModuleVersion{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.TerraformModuleVersions.DeleteModuleVersion(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this moduleVersion:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformModuleVersions = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform module version functions",
		FullPath:    "top-level-group-0-for-terraform-module-versions",
		CreatedBy:   "someone-g0",
	},
	{
		Description: "top level group 1 for testing terraform module version functions",
		FullPath:    "top-level-group-1-for-terraform-module-versions",
		CreatedBy:   "someone-g1",
	},
	{
		Description: "top level group 2 for testing terraform module version functions",
		FullPath:    "top-level-group-2-for-terraform-module-versions",
		CreatedBy:   "someone-g2",
	},
	{
		Description: "top level group 3 for testing terraform module version functions",
		FullPath:    "top-level-group-3-for-terraform-module-versions",
		CreatedBy:   "someone-g3",
	},
	{
		Description: "top level group 4 for testing terraform module version functions",
		FullPath:    "top-level-group-4-for-terraform-module-versions",
		CreatedBy:   "someone-g4",
	},
	// Nested groups:
	{
		Description: "nested group 5 for testing terraform module version functions",
		FullPath:    "top-level-group-4-for-terraform-module-versions/nested-group-5-for-terraform-module-versions",
		CreatedBy:   "someone-g5",
	},
	{
		Description: "nested group 6 for testing terraform module version functions",
		FullPath:    "top-level-group-3-for-terraform-module-versions/nested-group-6-for-terraform-module-versions",
		CreatedBy:   "someone-g6",
	},
	{
		Description: "nested group 7 for testing terraform module version functions",
		FullPath:    "top-level-group-2-for-terraform-module-versions/nested-group-7-for-terraform-module-versions",
		CreatedBy:   "someone-g7",
	},
	{
		Description: "nested group 8 for testing terraform module version functions",
		FullPath:    "top-level-group-1-for-terraform-module-versions/nested-group-8-for-terraform-module-versions",
		CreatedBy:   "someone-g8",
	},
	{
		Description: "nested group 9 for testing terraform module version functions",
		FullPath:    "top-level-group-0-for-terraform-module-versions/nested-group-9-for-terraform-module-versions",
		CreatedBy:   "someone-g9",
	},
}

// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformModulesForTerraformModulesVersions = []models.TerraformModule{
	{
		Name:        "module-a",
		System:      "aws",
		RootGroupID: "top-level-group-0-for-terraform-module-versions",
		GroupID:     "top-level-group-0-for-terraform-module-versions",
		Private:     false,
		CreatedBy:   "someone-tp0",
	},
	{
		Name:        "module-b",
		System:      "azure",
		RootGroupID: "top-level-group-0-for-terraform-module-versions",
		GroupID:     "top-level-group-0-for-terraform-module-versions",
		Private:     false,
		CreatedBy:   "someone-tp1",
	},
	{
		Name:        "module-c",
		System:      "azure",
		RootGroupID: "top-level-group-0-for-terraform-module-versions",
		GroupID:     "top-level-group-0-for-terraform-module-versions",
		Private:     false,
		CreatedBy:   "someone-tp1",
	},
}

// Standard warmup terraform moduleVersions for tests in this moduleVersion:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformModuleVersions = []models.TerraformModuleVersion{
	{
		ModuleID:        "top-level-group-0-for-terraform-module-versions/module-a/aws",
		SemanticVersion: "0.0.1",
		Status:          models.TerraformModuleVersionStatusUploaded,
		SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
		Submodules:      []string{"submodule1"},
		Examples:        []string{"example1"},
		Latest:          false,
		CreatedBy:       "someone-0",
	},
	{
		ModuleID:        "top-level-group-0-for-terraform-module-versions/module-a/aws",
		SemanticVersion: "0.0.2",
		Status:          models.TerraformModuleVersionStatusUploaded,
		SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
		Submodules:      []string{"submodule1"},
		Examples:        []string{"example1"},
		Latest:          true,
		CreatedBy:       "someone-0",
	},
	{
		ModuleID:        "top-level-group-0-for-terraform-module-versions/module-b/azure",
		SemanticVersion: "0.0.2",
		Status:          models.TerraformModuleVersionStatusUploaded,
		SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
		Submodules:      []string{"submodule1"},
		Examples:        []string{"example1"},
		Latest:          true,
		CreatedBy:       "someone-0",
	},
	{
		ModuleID:        "top-level-group-0-for-terraform-module-versions/module-c/azure",
		SemanticVersion: "1.0.0",
		Status:          models.TerraformModuleVersionStatusErrored,
		SHASum:          []byte("9ecb4d0fff7208208a11a432001e44eb6fb2dbb58cc4fdec87e3f29dbe35fa11"),
		Submodules:      []string{},
		Examples:        []string{},
		Error:           "failed validation",
		Diagnostics:     "syntax error on line 2",
		Latest:          true,
		CreatedBy:       "someone-0",
	},
}

// createWarmupTerraformModuleVersions creates some warmup terraform moduleVersions for a test
// The warmup terraform moduleVersions to create can be standard or otherwise.
func createWarmupTerraformModuleVersions(ctx context.Context, testClient *testClient,
	input warmupTerraformModuleVersions) (*warmupTerraformModuleVersions, error) {

	// It is necessary to create at least one group in order to
	// provide the necessary IDs for the terraform module versions.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultTerraformProviders, moduleResourcePath2ID, err := createInitialTerraformModules(ctx, testClient,
		input.terraformModules, parentPath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderVersions, err := createInitialTerraformModuleVersions(ctx, testClient,
		input.terraformModuleVersions, moduleResourcePath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformModuleVersions{
		groups:                  resultGroups,
		terraformModules:        resultTerraformProviders,
		terraformModuleVersions: resultTerraformProviderVersions,
	}, nil
}

func ptrTerraformModuleVersionSortableField(arg TerraformModuleVersionSortableField) *TerraformModuleVersionSortableField {
	return &arg
}

func (wis terraformModuleVersionInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleVersionInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleVersionInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformModuleVersionInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleVersionInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleVersionInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// terraformModuleVersionInfoFromTerraformModuleVersions returns a slice of terraformModuleVersionInfo, not necessarily sorted in any order.
func terraformModuleVersionInfoFromTerraformModuleVersions(terraformModuleVersions []models.TerraformModuleVersion) []terraformModuleVersionInfo {
	result := []terraformModuleVersionInfo{}

	for _, tp := range terraformModuleVersions {
		result = append(result, terraformModuleVersionInfo{
			id:         tp.Metadata.ID,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformModuleVersionIDsFromTerraformModuleVersionInfos preserves order
func terraformModuleVersionIDsFromTerraformModuleVersionInfos(terraformModuleVersionInfos []terraformModuleVersionInfo) []string {
	result := []string{}
	for _, terraformModuleVersionInfo := range terraformModuleVersionInfos {
		result = append(result, terraformModuleVersionInfo.id)
	}
	return result
}

// compareTerraformModuleVersions compares two terraform module version objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformModuleVersions(t *testing.T, expected, actual *models.TerraformModuleVersion,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.ModuleID, actual.ModuleID)
	assert.Equal(t, expected.SemanticVersion, actual.SemanticVersion)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Diagnostics, actual.Diagnostics)
	assert.Equal(t, expected.Submodules, actual.Submodules)
	assert.Equal(t, expected.Examples, actual.Examples)
	assert.Equal(t, expected.Error, actual.Error)
	assert.Equal(t, expected.UploadStartedTimestamp, actual.UploadStartedTimestamp)
	assert.Equal(t, expected.SHASum, actual.SHASum)
	assert.Equal(t, expected.Latest, actual.Latest)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}

// createInitialTerraformModuleVersions creates some warmup Terraform moduleVersions for a test.
func createInitialTerraformModuleVersions(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformModuleVersion, moduleResourcePath2ID map[string]string) ([]models.TerraformModuleVersion, error) {
	result := []models.TerraformModuleVersion{}

	for _, input := range toCreate {

		moduleResourcePath := input.ModuleID
		moduleID, ok := moduleResourcePath2ID[moduleResourcePath]
		if !ok {
			return nil,
				fmt.Errorf("createInitialTerraformModuleVersions failed to look up module resource path: %s",
					moduleResourcePath)
		}
		input.ModuleID = moduleID

		created, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}
