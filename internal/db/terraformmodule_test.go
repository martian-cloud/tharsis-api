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

// terraformModuleInfo aids convenience in accessing the information
// TestGetModules needs about the warmup objects.
type terraformModuleInfo struct {
	updateTime time.Time
	id         string
	name       string
}

// terraformModuleInfoIDSlice makes a slice of terraformModuleInfo sortable by ID string
type terraformModuleInfoIDSlice []terraformModuleInfo

// terraformModuleInfoNameSlice makes a slice of terraformModuleInfo sortable by name string
type terraformModuleInfoNameSlice []terraformModuleInfo

// terraformModuleInfoUpdateSlice makes a slice of terraformModuleInfo sortable by last updated time
type terraformModuleInfoUpdateSlice []terraformModuleInfo

// warmupTerraformModules holds the inputs to and outputs from createWarmupTerraformModules.
type warmupTerraformModules struct {
	groups                  []models.Group
	workspaces              []models.Workspace
	teams                   []models.Team
	users                   []models.User
	teamMembers             []models.TeamMember
	serviceAccounts         []models.ServiceAccount
	namespaceMembershipsIn  []CreateNamespaceMembershipInput
	namespaceMembershipsOut []models.NamespaceMembership
	terraformModules        []models.TerraformModule
}

func TestGetModuleByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:           standardWarmupGroupsForTerraformModules,
		terraformModules: standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg             *string
		expectTerraformModule *models.TerraformModule
		name                  string
		searchID              string
	}

	testCases := []testCase{
		{
			name:                  "get module by ID",
			searchID:              warmupItems.terraformModules[0].Metadata.ID,
			expectTerraformModule: &warmupItems.terraformModules[0],
		},

		{
			name:     "returns nil because module does not exist",
			searchID: nonExistentID,
		},

		{
			name:      "returns an error because the module ID is invalid",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformModule, err := testClient.client.TerraformModules.GetModuleByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformModule != nil {
				require.NotNil(t, actualTerraformModule)
				assert.Equal(t, test.expectTerraformModule, actualTerraformModule)
			} else {
				assert.Nil(t, actualTerraformModule)
			}
		})
	}
}

func TestGetModuleByPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:           standardWarmupGroupsForTerraformModules,
		terraformModules: standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg             *string
		expectTerraformModule *models.TerraformModule
		name                  string
		searchPath            string
	}

	testCases := []testCase{
		{
			name:                  "positive",
			searchPath:            warmupItems.terraformModules[0].ResourcePath,
			expectTerraformModule: &warmupItems.terraformModules[0],
		},

		{
			name:       "negative, non-existent Terraform module ID",
			searchPath: "this/path/does/not/exist",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTerraformModule, err := testClient.client.TerraformModules.GetModuleByPath(ctx, test.searchPath)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformModule != nil {
				require.NotNil(t, actualTerraformModule)
				assert.Equal(t, test.expectTerraformModule, actualTerraformModule)
			} else {
				assert.Nil(t, actualTerraformModule)
			}
		})
	}
}

func TestGetModulesWithPagination(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:           standardWarmupGroupsForTerraformModules,
		terraformModules: standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	// Query for first page
	middleIndex := len(warmupItems.terraformModules) / 2
	page1, err := testClient.client.TerraformModules.GetModules(ctx, &GetModulesInput{
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(int32(middleIndex)),
		},
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(page1.Modules))
	assert.True(t, page1.PageInfo.HasNextPage)
	assert.False(t, page1.PageInfo.HasPreviousPage)

	cursor, err := page1.PageInfo.Cursor(&page1.Modules[len(page1.Modules)-1])
	require.Nil(t, err)

	remaining := len(warmupItems.terraformModules) - middleIndex
	page2, err := testClient.client.TerraformModules.GetModules(ctx, &GetModulesInput{
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(int32(remaining)),
			After: cursor,
		},
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(page2.Modules))
	assert.True(t, page2.PageInfo.HasPreviousPage)
	assert.False(t, page2.PageInfo.HasNextPage)
}

func TestGetModules(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:                 standardWarmupGroupsForTerraformModules,
		workspaces:             standardWarmupWorkspacesForTerraformModules,
		teams:                  standardWarmupTeamsForTerraformModules,
		users:                  standardWarmupUsersForTerraformModules,
		teamMembers:            standardWarmupTeamMembersForTerraformModules,
		serviceAccounts:        standardWarmupServiceAccountsForTerraformModules,
		namespaceMembershipsIn: standardWarmupNamespaceMembershipsForTerraformModules,
		terraformModules:       standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	allTerraformModuleInfos := terraformModuleInfoFromTerraformModules(warmupItems.terraformModules)

	// Sort by Terraform module IDs.
	sort.Sort(terraformModuleInfoIDSlice(allTerraformModuleInfos))
	allTerraformModuleIDs := terraformModuleIDsFromTerraformModuleInfos(allTerraformModuleInfos)

	// Sort by names.
	sort.Sort(terraformModuleInfoNameSlice(allTerraformModuleInfos))
	allTerraformModuleIDsByName := terraformModuleIDsFromTerraformModuleInfos(allTerraformModuleInfos)
	reverseTerraformModuleIDsByName := reverseStringSlice(allTerraformModuleIDsByName)

	// Sort by last update times.
	sort.Sort(terraformModuleInfoUpdateSlice(allTerraformModuleInfos))
	allTerraformModuleIDsByTime := terraformModuleIDsFromTerraformModuleInfos(allTerraformModuleInfos)
	reverseTerraformModuleIDsByTime := reverseStringSlice(allTerraformModuleIDsByTime)

	type testCase struct {
		input                    *GetModulesInput
		expectMsg                *string
		name                     string
		expectTerraformModuleIDs []string
	}

	testCases := []testCase{
		{
			name: "non-nil but mostly empty input",
			input: &GetModulesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformModuleIDs: allTerraformModuleIDs,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime,
		},

		{
			name: "sort in ascending order of name",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldNameAsc),
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByName,
		},

		{
			name: "sort in descending order of name",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldNameDesc),
			},
			expectTerraformModuleIDs: reverseTerraformModuleIDsByName,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtDesc),
			},
			expectTerraformModuleIDs: reverseTerraformModuleIDsByTime,
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                ptr.String("only first or last can be defined, not both"),
			expectTerraformModuleIDs: allTerraformModuleIDs[4:],
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &TerraformModuleFilter{
					Search:           ptr.String(""),
					Name:             ptr.String(""),
					RootGroupID:      ptr.String(""),
					GroupID:          ptr.String(""),
					UserID:           ptr.String(""),
					ServiceAccountID: ptr.String(""),
				},
			},
			expectMsg:                emptyUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, search field, empty string",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Search: ptr.String(""),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime,
		},

		{
			name: "filter, search field, 1",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Search: ptr.String("1"),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[0:2],
		},

		{
			name: "filter, search field, 2",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Search: ptr.String("2"),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[2:4],
		},

		{
			name: "filter, search field, 5",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Search: ptr.String("5"),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[4:],
		},

		{
			name: "filter, search field, bogus",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, name, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Name: ptr.String(warmupItems.terraformModules[0].Name),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[0:1],
		},

		{
			name: "filter, name, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					Name: ptr.String(nonExistentID),
				},
			},
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, root group ID, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					RootGroupID: ptr.String(warmupItems.terraformModules[0].RootGroupID),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[0:1],
		},

		{
			name: "filter, root group ID, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					RootGroupID: ptr.String(nonExistentID),
				},
			},
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, root group ID, invalid",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					RootGroupID: ptr.String(invalidID),
				},
			},
			expectMsg:                invalidUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, group ID, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					GroupID: ptr.String(warmupItems.terraformModules[0].GroupID),
				},
			},
			expectTerraformModuleIDs: allTerraformModuleIDsByTime[0:1],
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:                invalidUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, user ID, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					UserID: ptr.String(warmupItems.users[0].Metadata.ID),
				},
			},
			// Gets 0 because it's public, 4 by user ID.
			expectTerraformModuleIDs: []string{allTerraformModuleIDsByName[0], allTerraformModuleIDsByName[4]},
		},

		{
			name: "filter, user ID, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					UserID: ptr.String(nonExistentID),
				},
			},
			// Gets 0 because it's public.
			expectTerraformModuleIDs: []string{allTerraformModuleIDsByName[0]},
		},

		{
			name: "filter, user, invalid",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					UserID: ptr.String(invalidID),
				},
			},
			expectMsg:                invalidUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, service account ID, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					ServiceAccountID: ptr.String(warmupItems.serviceAccounts[0].Metadata.ID),
				},
			},
			// Gets 0 because it's public, 4 by service account ID.
			expectTerraformModuleIDs: []string{allTerraformModuleIDsByName[0], allTerraformModuleIDsByName[4]},
		},

		{
			name: "filter, service account ID, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					ServiceAccountID: ptr.String(nonExistentID),
				},
			},
			// Gets 0 because it's public.
			expectTerraformModuleIDs: []string{allTerraformModuleIDsByName[0]},
		},

		{
			name: "filter, service account ID, invalid",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					ServiceAccountID: ptr.String(invalidID),
				},
			},
			expectMsg:                invalidUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, terraform module IDs, positive",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					TerraformModuleIDs: []string{
						allTerraformModuleIDsByTime[0], allTerraformModuleIDsByTime[1], allTerraformModuleIDsByTime[3]},
				},
			},
			expectTerraformModuleIDs: []string{
				allTerraformModuleIDsByTime[0], allTerraformModuleIDsByTime[1], allTerraformModuleIDsByTime[3],
			},
		},

		{
			name: "filter, terraform module IDs, non-existent",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					TerraformModuleIDs: []string{nonExistentID},
				},
			},
			expectTerraformModuleIDs: []string{},
		},

		{
			name: "filter, terraform module IDs, invalid ID",
			input: &GetModulesInput{
				Sort: ptrTerraformModuleSortableField(TerraformModuleSortableFieldUpdatedAtAsc),
				Filter: &TerraformModuleFilter{
					TerraformModuleIDs: []string{invalidID},
				},
			},
			expectMsg:                invalidUUIDMsg2,
			expectTerraformModuleIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			terraformModulesResult, err := testClient.client.TerraformModules.GetModules(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if err == nil {
				// Never returns nil if error is nil.
				require.NotNil(t, terraformModulesResult.PageInfo)

				terraformModules := terraformModulesResult.Modules

				// Check the terraform modules result by comparing a list of the terraform module IDs.
				actualTerraformModuleIDs := []string{}
				for _, terraformModule := range terraformModules {
					actualTerraformModuleIDs = append(actualTerraformModuleIDs, terraformModule.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformModuleIDs)
				}

				assert.Equal(t, len(test.expectTerraformModuleIDs), len(actualTerraformModuleIDs))
				assert.Equal(t, test.expectTerraformModuleIDs, actualTerraformModuleIDs)
			}
		})
	}
}

func TestCreateModule(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups: standardWarmupGroupsForTerraformModules,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformModule
		expectCreated *models.TerraformModule
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformModule{
				Name:        "terraform-module-create-test",
				System:      "aws",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
				Private:     true,
				CreatedBy:   "TestCreateModule",
			},
			expectCreated: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:         "terraform-module-create-test",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test/aws",
				RootGroupID:  warmupItems.groups[0].Metadata.ID,
				GroupID:      warmupItems.groups[0].Metadata.ID,
				Private:      true,
				CreatedBy:    "TestCreateModule",
			},
		},

		{
			name: "allow duplicate name but different system",
			toCreate: &models.TerraformModule{
				Name:        "terraform-module-create-test",
				System:      "azure",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
				Private:     true,
				CreatedBy:   "TestCreateModule",
			},
			expectCreated: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:         "terraform-module-create-test",
				System:       "azure",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test/azure",
				RootGroupID:  warmupItems.groups[0].Metadata.ID,
				GroupID:      warmupItems.groups[0].Metadata.ID,
				Private:      true,
				CreatedBy:    "TestCreateModule",
			},
		},

		{
			name: "duplicate group ID and Terraform module name",
			toCreate: &models.TerraformModule{
				Name:         "terraform-module-create-test",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test",
				RootGroupID:  warmupItems.groups[0].Metadata.ID,
				GroupID:      warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("terraform module with name terraform-module-create-test and system aws already exists"),
		},

		{
			name: "negative, non-existent root group ID",
			toCreate: &models.TerraformModule{
				Name:         "terraform-module-create-test-non-existent-root-group-id",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test-non-existent-root-group-id",
				RootGroupID:  nonExistentID,
				GroupID:      warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_modules\" violates foreign key constraint \"fk_root_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, non-existent group ID",
			toCreate: &models.TerraformModule{
				Name:         "terraform-module-create-test-non-existent-group-id",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test-non-existent-group-id",
				RootGroupID:  warmupItems.groups[0].Metadata.ID,
				GroupID:      nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_modules\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid root group ID",
			toCreate: &models.TerraformModule{
				Name:         "terraform-module-create-test-invalid-root-group-id",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test-invalid-root-group-id",
				RootGroupID:  invalidID,
				GroupID:      warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: invalidUUIDMsg1,
		},

		{
			name: "negative, invalid group ID",
			toCreate: &models.TerraformModule{
				Name:         "terraform-module-create-test-invalid-group-id",
				System:       "aws",
				ResourcePath: warmupItems.groups[0].FullPath + "/terraform-module-create-test-invalid-group-id",
				RootGroupID:  warmupItems.groups[0].Metadata.ID,
				GroupID:      invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.TerraformModules.CreateModule(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformModules(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateModule(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:           standardWarmupGroupsForTerraformModules,
		terraformModules: standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformModule
		expectUpdated *models.TerraformModule
		name          string
	}

	// Looks up by ID and version.  Also requires group ID.
	// Updates name and private.  Returns rebuilt resource path.
	// The NamespacePath field is not updated in the DB, but the value from the argument is returned.
	positiveTerraformModule := warmupItems.terraformModules[0]
	positiveGroup := warmupItems.groups[9]
	otherTerraformModule := warmupItems.terraformModules[1]
	now := time.Now()
	testCases := []testCase{

		{
			name: "positive",
			toUpdate: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformModule.Metadata.ID,
					Version: initialResourceVersion,
				},
				Name:    "updated-terraform-module-name",
				System:  "aws",
				Private: !positiveTerraformModule.Private,
				GroupID: positiveGroup.Metadata.ID,
			},
			expectUpdated: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:                   positiveTerraformModule.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveTerraformModule.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Name:         "updated-terraform-module-name",
				System:       "aws",
				ResourcePath: positiveGroup.FullPath + "/updated-terraform-module-name/aws",
				RootGroupID:  positiveTerraformModule.RootGroupID,
				GroupID:      positiveTerraformModule.GroupID,
				Private:      !positiveTerraformModule.Private,
				CreatedBy:    positiveTerraformModule.CreatedBy,
			},
		},

		{
			name: "would-be-duplicate-group-id-and-module-name",
			toUpdate: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformModule.Metadata.ID,
					Version: initialResourceVersion,
				},
				// Would duplicate a different Terraform module.
				Name: otherTerraformModule.Name,
			},
			expectMsg: ptr.String("resource version does not match specified version"),
		},

		{
			name: "negative, non-existent Terraform module ID",
			toUpdate: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformModule{
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

			actualTerraformModule, err := testClient.client.TerraformModules.UpdateModule(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformModule)
				compareTerraformModules(t, test.expectUpdated, actualTerraformModule, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformModule)
			}
		})
	}
}

func TestDeleteModule(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModules(ctx, testClient, warmupTerraformModules{
		groups:           standardWarmupGroupsForTerraformModules,
		terraformModules: standardWarmupTerraformModules,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformModule
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.terraformModules[0].Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform module ID",
			toDelete: &models.TerraformModule{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformModule{
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

			err := testClient.client.TerraformModules.DeleteModule(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformModules = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform module functions",
		FullPath:    "top-level-group-0-for-terraform-modules",
		CreatedBy:   "someone-g0",
	},
	{
		Description: "top level group 1 for testing terraform module functions",
		FullPath:    "top-level-group-1-for-terraform-modules",
		CreatedBy:   "someone-g1",
	},
	{
		Description: "top level group 2 for testing terraform module functions",
		FullPath:    "top-level-group-2-for-terraform-modules",
		CreatedBy:   "someone-g2",
	},
	{
		Description: "top level group 3 for testing terraform module functions",
		FullPath:    "top-level-group-3-for-terraform-modules",
		CreatedBy:   "someone-g3",
	},
	{
		Description: "top level group 4 for testing terraform module functions",
		FullPath:    "top-level-group-4-for-terraform-modules",
		CreatedBy:   "someone-g4",
	},
	// Nested groups:
	{
		Description: "nested group 5 for testing terraform module functions",
		FullPath:    "top-level-group-4-for-terraform-modules/nested-group-5-for-terraform-modules",
		CreatedBy:   "someone-g5",
	},
	{
		Description: "nested group 6 for testing terraform module functions",
		FullPath:    "top-level-group-3-for-terraform-modules/nested-group-6-for-terraform-modules",
		CreatedBy:   "someone-g6",
	},
	{
		Description: "nested group 7 for testing terraform module functions",
		FullPath:    "top-level-group-2-for-terraform-modules/nested-group-7-for-terraform-modules",
		CreatedBy:   "someone-g7",
	},
	{
		Description: "nested group 8 for testing terraform module functions",
		FullPath:    "top-level-group-1-for-terraform-modules/nested-group-8-for-terraform-modules",
		CreatedBy:   "someone-g8",
	},
	{
		Description: "nested group 9 for testing terraform module functions",
		FullPath:    "top-level-group-0-for-terraform-modules/nested-group-9-for-terraform-modules",
		CreatedBy:   "someone-g9",
	},
}

// Standard warmup workspaces for tests in this module:
// The create function will derive the group ID and name from the namespace path.
var standardWarmupWorkspacesForTerraformModules = []models.Workspace{
	{
		Description: "workspace 0 for testing terraform module functions",
		FullPath:    "top-level-group-0-for-terraform-modules/workspace-0-in-group-0",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing terraform module functions",
		FullPath:    "top-level-group-1-for-terraform-modules/workspace-1-in-group-1",
		CreatedBy:   "someone-w1",
	},
	{
		Description: "workspace 2 for testing terraform module functions",
		FullPath:    "top-level-group-2-for-terraform-modules/workspace-2-in-group-2",
		CreatedBy:   "someone-w2",
	},
}

// Standard warmup teams for tests in this module:
var standardWarmupTeamsForTerraformModules = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for terraform module tests",
	},
	{
		Name:        "team-b",
		Description: "team b for terraform module tests",
	},
}

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsersForTerraformModules = []models.User{
	{
		Username: "user-0",
		Email:    "user-0@example.com",
	},
	{
		Username: "user-1",
		Email:    "user-1@example.com",
	},
	{
		Username: "user-team-a",
		Email:    "user-2@example.com",
	},
	{
		Username: "user-team-b",
		Email:    "user-3@example.com",
	},
}

// Standard warmup team member relationships for tests in this module:
// Please note that the ID fields contain names, not IDs.
var standardWarmupTeamMembersForTerraformModules = []models.TeamMember{
	{
		UserID: "user-team-a",
		TeamID: "team-a",
	},
	{
		UserID: "user-team-b",
		TeamID: "team-b",
	},
}

// Standard service account(s) for tests in this module:
// The create function will convert the group name to group ID.
var standardWarmupServiceAccountsForTerraformModules = []models.ServiceAccount{
	{
		ResourcePath:      "sa-resource-path-0",
		Name:              "service-account-0",
		Description:       "service account 0",
		GroupID:           "top-level-group-2-for-terraform-modules/nested-group-7-for-terraform-modules",
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-1",
		Name:              "service-account-1",
		Description:       "service account 1",
		GroupID:           "top-level-group-1-for-terraform-modules/nested-group-8-for-terraform-modules",
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

// standardWarmupRolesForTerraformModules for tests in this module.
var standardWarmupRolesForTerraformModules = []models.Role{
	{
		Name:        "role-a",
		Description: "Warmup role-a for terraform modules",
	},
	{
		Name:        "role-b",
		Description: "Warmup role-b for terraform modules",
	},
}

// Standard warmup namespace memberships for tests in this module:
// In this variable, the ID field is the user, service account, and team _NAME_, NOT the ID.
var standardWarmupNamespaceMembershipsForTerraformModules = []CreateNamespaceMembershipInput{

	// Team access to group:
	{
		NamespacePath: "top-level-group-3-for-terraform-modules",
		TeamID:        ptr.String("team-a"),
		RoleID:        "role-a",
	},

	// User access to group:
	{
		NamespacePath: "top-level-group-4-for-terraform-modules",
		UserID:        ptr.String("user-0"),
		RoleID:        "role-b",
	},

	// Service accounts access to group:
	{
		NamespacePath:    "top-level-group-4-for-terraform-modules/nested-group-5-for-terraform-modules",
		ServiceAccountID: ptr.String("service-account-0"),
		RoleID:           "role-a",
	},

	// Team access to workspace:
	{
		NamespacePath: "top-level-group-0-for-terraform-modules/workspace-0-in-group-0",
		TeamID:        ptr.String("team-b"),
		RoleID:        "role-a",
	},

	// User access to workspace:
	{
		NamespacePath: "top-level-group-1-for-terraform-modules/workspace-1-in-group-1",
		UserID:        ptr.String("user-1"),
		RoleID:        "role-b",
	},

	// Service account access to workspace:
	{
		NamespacePath:    "top-level-group-2-for-terraform-modules/workspace-2-in-group-2",
		ServiceAccountID: ptr.String("service-account-1"),
		RoleID:           "role-a",
	},
}

// Standard warmup terraform modules for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformModules = []models.TerraformModule{
	{
		// This one is public.
		Name:         "1-terraform-module-0",
		System:       "aws",
		ResourcePath: "top-level-group-0-for-terraform-modules/1-terraform-module-0",
		RootGroupID:  "top-level-group-0-for-terraform-modules",
		GroupID:      "top-level-group-0-for-terraform-modules/nested-group-9-for-terraform-modules",
		Private:      false,
		CreatedBy:    "someone-sv0",
	},
	{
		Name:         "1-terraform-module-1",
		System:       "aws",
		ResourcePath: "top-level-group-1-for-terraform-modules/1-terraform-module-1",
		RootGroupID:  "top-level-group-1-for-terraform-modules",
		GroupID:      "top-level-group-1-for-terraform-modules",
		Private:      true,
		CreatedBy:    "someone-sv1",
	},
	{
		Name:         "2-terraform-module-2",
		System:       "aws",
		ResourcePath: "top-level-group-2-for-terraform-modules/2-terraform-module-2",
		RootGroupID:  "top-level-group-2-for-terraform-modules",
		GroupID:      "top-level-group-2-for-terraform-modules/nested-group-7-for-terraform-modules",
		Private:      true,
		CreatedBy:    "someone-sv2",
	},
	{
		Name:         "2-terraform-module-3",
		System:       "aws",
		ResourcePath: "top-level-group-3-for-terraform-modules/2-terraform-module-3",
		RootGroupID:  "top-level-group-3-for-terraform-modules",
		GroupID:      "top-level-group-3-for-terraform-modules",
		Private:      true,
		CreatedBy:    "someone-sv3",
	},
	{
		Name:         "5-terraform-module-4",
		System:       "aws",
		ResourcePath: "top-level-group-4-for-terraform-modules/5-terraform-module-4",
		RootGroupID:  "top-level-group-4-for-terraform-modules",
		GroupID:      "top-level-group-4-for-terraform-modules/nested-group-5-for-terraform-modules",
		Private:      true,
		CreatedBy:    "someone-sv4",
	},
}

// createWarmupTerraformModules creates some warmup terraform modules for a test
// The warmup terraform modules to create can be standard or otherwise.
func createWarmupTerraformModules(ctx context.Context, testClient *testClient,
	input warmupTerraformModules) (*warmupTerraformModules, error) {

	// It is necessary to create several groups in order to provide the necessary IDs for the terraform modules.

	// If doing get operations based on user ID or service account ID, it is necessary to create a bunch of other things.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	resultTeams, teamName2ID, err := createInitialTeams(ctx, testClient, input.teams)
	if err != nil {
		return nil, err
	}

	resultUsers, username2ID, err := createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, err
	}

	resultTeamMembers, err := createInitialTeamMembers(ctx, testClient, teamName2ID, username2ID, input.teamMembers)
	if err != nil {
		return nil, err
	}

	resultServiceAccounts, serviceAccountName2ID, err := createInitialServiceAccounts(ctx, testClient,
		parentPath2ID, input.serviceAccounts)
	if err != nil {
		return nil, err
	}

	_, roleName2ID, err := createInitialRoles(ctx, testClient, standardWarmupRolesForTerraformModules)
	if err != nil {
		return nil, err
	}

	resultNamespaceMemberships, err := createInitialNamespaceMemberships(ctx, testClient,
		teamName2ID, username2ID, parentPath2ID, serviceAccountName2ID, roleName2ID, input.namespaceMembershipsIn)
	if err != nil {
		return nil, err
	}

	resultTerraformModules, _, err := createInitialTerraformModules(ctx, testClient,
		input.terraformModules, parentPath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformModules{
		groups:                  resultGroups,
		workspaces:              resultWorkspaces,
		teams:                   resultTeams,
		users:                   resultUsers,
		teamMembers:             resultTeamMembers,
		serviceAccounts:         resultServiceAccounts,
		namespaceMembershipsOut: resultNamespaceMemberships,
		terraformModules:        resultTerraformModules,
	}, nil
}

func ptrTerraformModuleSortableField(arg TerraformModuleSortableField) *TerraformModuleSortableField {
	return &arg
}

func (wis terraformModuleInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformModuleInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

func (wis terraformModuleInfoNameSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleInfoNameSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleInfoNameSlice) Less(i, j int) bool {
	return wis[i].name < wis[j].name
}

// terraformModuleInfoFromTerraformModules returns a slice of terraformModuleInfo, not necessarily sorted in any order.
func terraformModuleInfoFromTerraformModules(terraformModules []models.TerraformModule) []terraformModuleInfo {
	result := []terraformModuleInfo{}

	for _, tp := range terraformModules {
		result = append(result, terraformModuleInfo{
			id:         tp.Metadata.ID,
			name:       tp.Name,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformModuleIDsFromTerraformModuleInfos preserves order
func terraformModuleIDsFromTerraformModuleInfos(terraformModuleInfos []terraformModuleInfo) []string {
	result := []string{}
	for _, terraformModuleInfo := range terraformModuleInfos {
		result = append(result, terraformModuleInfo.id)
	}
	return result
}

// compareTerraformModules compares two terraform module objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformModules(t *testing.T, expected, actual *models.TerraformModule,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.ResourcePath, actual.ResourcePath)
	assert.Equal(t, expected.RootGroupID, actual.RootGroupID)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Private, actual.Private)
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

// createInitialTerraformModules creates some warmup Terraform modules for a test.
func createInitialTerraformModules(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformModule, groupPath2ID map[string]string) (
	[]models.TerraformModule, map[string]string, error) {
	result := []models.TerraformModule{}
	resourcePath2ID := make(map[string]string)

	for _, input := range toCreate {

		rootGroupPath := input.RootGroupID
		rootGroupID, ok := groupPath2ID[rootGroupPath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformModules failed to look up root group path: %s", rootGroupPath)
		}
		input.RootGroupID = rootGroupID

		groupPath := input.GroupID
		groupID, ok := groupPath2ID[groupPath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformModules failed to look up group path: %s", groupPath)
		}
		input.GroupID = groupID

		created, err := testClient.client.TerraformModules.CreateModule(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		resourcePath2ID[created.ResourcePath] = created.Metadata.ID
	}

	return result, resourcePath2ID, nil
}

// The End.
