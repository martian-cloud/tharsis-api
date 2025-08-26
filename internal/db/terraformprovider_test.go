//go:build integration

package db

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// terraformProviderInfo aids convenience in accessing the information
// TestGetProviders needs about the warmup objects.
type terraformProviderInfo struct {
	updateTime time.Time
	id         string
	name       string
}

// terraformProviderInfoIDSlice makes a slice of terraformProviderInfo sortable by ID string
type terraformProviderInfoIDSlice []terraformProviderInfo

// terraformProviderInfoNameSlice makes a slice of terraformProviderInfo sortable by name string
type terraformProviderInfoNameSlice []terraformProviderInfo

// terraformProviderInfoUpdateSlice makes a slice of terraformProviderInfo sortable by last updated time
type terraformProviderInfoUpdateSlice []terraformProviderInfo

// warmupTerraformProviders holds the inputs to and outputs from createWarmupTerraformProviders.
type warmupTerraformProviders struct {
	groups                  []models.Group
	workspaces              []models.Workspace
	teams                   []models.Team
	users                   []models.User
	teamMembers             []models.TeamMember
	serviceAccounts         []models.ServiceAccount
	namespaceMembershipsIn  []CreateNamespaceMembershipInput
	namespaceMembershipsOut []models.NamespaceMembership
	terraformProviders      []models.TerraformProvider
}

func TestGetProviderByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupItems, err := createWarmupTerraformProviders(ctx, testClient, warmupTerraformProviders{
		groups:             standardWarmupGroupsForTerraformProviders,
		terraformProviders: standardWarmupTerraformProviders,
	})
	require.Nil(t, err)
	createdHigh := time.Now()

	type testCase struct {
		expectMsg               *string
		expectTerraformProvider *models.TerraformProvider
		name                    string
		searchID                string
	}

	positiveTerraformProvider := warmupItems.terraformProviders[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveTerraformProvider.Metadata.ID,
			expectTerraformProvider: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:                positiveTerraformProvider.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
					TRN:               positiveTerraformProvider.Metadata.TRN,
				},
				Name:        positiveTerraformProvider.Name,
				RootGroupID: positiveTerraformProvider.RootGroupID,
				GroupID:     positiveTerraformProvider.GroupID,
				Private:     positiveTerraformProvider.Private,
				CreatedBy:   positiveTerraformProvider.CreatedBy,
			},
		},

		{
			name:     "negative, non-existent Terraform provider ID",
			searchID: nonExistentID,
			// expect terraform provider and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTerraformProvider, err := testClient.client.TerraformProviders.GetProviderByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformProvider != nil {
				require.NotNil(t, actualTerraformProvider)
				compareTerraformProviders(t, test.expectTerraformProvider, actualTerraformProvider, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualTerraformProvider)
			}
		})
	}
}

func TestGetProviderByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider",
		RootGroupID: group.Metadata.ID,
		GroupID:     group.Metadata.ID,
		Private:     false,
		CreatedBy:   "TestGetProviderByTRN",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectProvider  bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:           "get provider by TRN",
			trn:            provider.Metadata.TRN,
			expectProvider: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TerraformProviderModelType.BuildTRN(group.FullPath, "non-existent-provider"),
		},
		{
			name:            "provider TRN has less than 2 parts",
			trn:             types.TerraformProviderModelType.BuildTRN("test-group"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualProvider, err := testClient.client.TerraformProviders.GetProviderByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectProvider {
				require.NotNil(t, actualProvider)
				assert.Equal(t,
					types.TerraformProviderModelType.BuildTRN(
						group.FullPath,
						provider.Name,
					),
					actualProvider.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualProvider)
			}
		})
	}
}

func TestGetProviders(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviders(ctx, testClient, warmupTerraformProviders{
		groups:                 standardWarmupGroupsForTerraformProviders,
		workspaces:             standardWarmupWorkspacesForTerraformProviders,
		teams:                  standardWarmupTeamsForTerraformProviders,
		users:                  standardWarmupUsersForTerraformProviders,
		teamMembers:            standardWarmupTeamMembersForTerraformProviders,
		serviceAccounts:        standardWarmupServiceAccountsForTerraformProviders,
		namespaceMembershipsIn: standardWarmupNamespaceMembershipsForTerraformProviders,
		terraformProviders:     standardWarmupTerraformProviders,
	})
	require.Nil(t, err)
	allTerraformProviderInfos := terraformProviderInfoFromTerraformProviders(warmupItems.terraformProviders)

	// Sort by Terraform provider IDs.
	sort.Sort(terraformProviderInfoIDSlice(allTerraformProviderInfos))
	allTerraformProviderIDs := terraformProviderIDsFromTerraformProviderInfos(allTerraformProviderInfos)

	// Sort by names.
	sort.Sort(terraformProviderInfoNameSlice(allTerraformProviderInfos))
	allTerraformProviderIDsByName := terraformProviderIDsFromTerraformProviderInfos(allTerraformProviderInfos)
	reverseTerraformProviderIDsByName := reverseStringSlice(allTerraformProviderIDsByName)

	// Sort by last update times.
	sort.Sort(terraformProviderInfoUpdateSlice(allTerraformProviderInfos))
	allTerraformProviderIDsByTime := terraformProviderIDsFromTerraformProviderInfos(allTerraformProviderInfos)
	reverseTerraformProviderIDsByTime := reverseStringSlice(allTerraformProviderIDsByTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetProvidersInput
		expectMsg                   *string
		name                        string
		expectPageInfo              pagination.PageInfo
		expectTerraformProviderIDs  []string
		getBeforeCursorFromPrevious bool
		sortedDescending            bool
		expectHasStartCursor        bool
		getAfterCursorFromPrevious  bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetProvidersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectTerraformProviderIDs:  []string{},
			expectPageInfo: pagination.PageInfo{
				Cursor:          nil,
				TotalCount:      0,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectStartCursorError: nil,
			expectHasStartCursor:   false,
			expectEndCursorError:   nil,
			expectHasEndCursor:     false,
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetProviders, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetProvidersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformProviderIDs: allTerraformProviderIDs,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "sort in ascending order of name",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldNameAsc),
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByName,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "sort in descending order of name",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldNameDesc),
			},
			sortedDescending:           true,
			expectTerraformProviderIDs: reverseTerraformProviderIDsByName,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtDesc),
			},
			sortedDescending:           true,
			expectTerraformProviderIDs: reverseTerraformProviderIDsByTime,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "pagination: everything at once",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "pagination: first two",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// When Last is supplied, the sort order is intended to be reversed.
		{
			name: "pagination: last three",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:           true,
			expectTerraformProviderIDs: reverseTerraformProviderIDsByTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		/*

			The input.PaginationOptions.After field is tested earlier via getAfterCursorFromPrevious.

			The input.PaginationOptions.Before field is not really supported and does not work.
			If it did work, it could be tested by adapting the test cases corresponding to the
			next few cases after a similar block of text from group_test.go

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetProvidersInput{
				Sort:              ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectTerraformProviderIDs:  []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                  ptr.String("only first or last can be defined, not both"),
			expectTerraformProviderIDs: allTerraformProviderIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &TerraformProviderFilter{
					Search:           ptr.String(""),
					Name:             ptr.String(""),
					RootGroupID:      ptr.String(""),
					GroupID:          ptr.String(""),
					UserID:           ptr.String(""),
					ServiceAccountID: ptr.String(""),
					// Passing an empty slice to TerraformProviderIDs likely causes
					// an SQL syntax error ("... IN ()"), so don't try it.
					// TerraformProvidersIDs: []string{},
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{},
		},

		{
			name: "filter, search field, empty string",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Search: ptr.String(""),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime,
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(len(allTerraformProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, search field, 1",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Search: ptr.String("1"),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[0:2],
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, search field, 2",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Search: ptr.String("2"),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[2:4],
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, search field, 5",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Search: ptr.String("5"),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[4:],
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, name, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Name: ptr.String(warmupItems.terraformProviders[0].Name),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[0:1],
			expectPageInfo:             pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, name, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					Name: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, root group ID, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					RootGroupID: ptr.String(warmupItems.terraformProviders[0].RootGroupID),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[0:1],
			expectPageInfo:             pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, root group ID, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					RootGroupID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, root group ID, invalid",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					RootGroupID: ptr.String(invalidID),
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{},
		},

		{
			name: "filter, group ID, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					GroupID: ptr.String(warmupItems.terraformProviders[0].GroupID),
				},
			},
			expectTerraformProviderIDs: allTerraformProviderIDsByTime[0:1],
			expectPageInfo:             pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{},
		},

		{
			name: "filter, user ID, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					UserID: ptr.String(warmupItems.users[0].Metadata.ID),
				},
			},
			// Gets 0 because it's public, 4 by user ID.
			expectTerraformProviderIDs: []string{allTerraformProviderIDsByName[0], allTerraformProviderIDsByName[4]},
			expectPageInfo:             pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, user ID, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					UserID: ptr.String(nonExistentID),
				},
			},
			// Gets 0 because it's public.
			expectTerraformProviderIDs: []string{allTerraformProviderIDsByName[0]},
			expectPageInfo:             pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, user, invalid",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					UserID: ptr.String(invalidID),
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{},
		},

		{
			name: "filter, service account ID, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					ServiceAccountID: ptr.String(warmupItems.serviceAccounts[0].Metadata.ID),
				},
			},
			// Gets 0 because it's public, 4 by service account ID.
			expectTerraformProviderIDs: []string{allTerraformProviderIDsByName[0], allTerraformProviderIDsByName[4]},
			expectPageInfo:             pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, service account ID, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					ServiceAccountID: ptr.String(nonExistentID),
				},
			},
			// Gets 0 because it's public.
			expectTerraformProviderIDs: []string{allTerraformProviderIDsByName[0]},
			expectPageInfo:             pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, service account ID, invalid",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					ServiceAccountID: ptr.String(invalidID),
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{},
		},

		{
			name: "filter, terraform provider IDs, positive",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					TerraformProviderIDs: []string{
						allTerraformProviderIDsByTime[0], allTerraformProviderIDsByTime[1], allTerraformProviderIDsByTime[3],
					},
				},
			},
			expectTerraformProviderIDs: []string{
				allTerraformProviderIDsByTime[0], allTerraformProviderIDsByTime[1], allTerraformProviderIDsByTime[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, terraform provider IDs, non-existent",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					TerraformProviderIDs: []string{nonExistentID},
				},
			},
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},

		{
			name: "filter, terraform provider IDs, invalid ID",
			input: &GetProvidersInput{
				Sort: ptrTerraformProviderSortableField(TerraformProviderSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderFilter{
					TerraformProviderIDs: []string{invalidID},
				},
			},
			expectMsg:                  invalidUUIDMsg,
			expectTerraformProviderIDs: []string{},
			expectPageInfo:             pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:       true,
			expectHasEndCursor:         true,
		},
	}

	var (
		previousEndCursorValue   *string
		previousStartCursorValue *string
	)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// For some pagination tests, a previous case's cursor value gets piped into the next case.
			if test.getAfterCursorFromPrevious || test.getBeforeCursorFromPrevious {

				// Make sure there's a place to put it.
				require.NotNil(t, test.input.PaginationOptions)

				if test.getAfterCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousEndCursorValue)
					test.input.PaginationOptions.After = previousEndCursorValue
				}

				if test.getBeforeCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousStartCursorValue)
					test.input.PaginationOptions.Before = previousStartCursorValue
				}

				// Clear the values so they won't be used twice.
				previousEndCursorValue = nil
				previousStartCursorValue = nil
			}

			terraformProvidersResult, err := testClient.client.TerraformProviders.GetProviders(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, terraformProvidersResult.PageInfo)
				assert.NotNil(t, terraformProvidersResult.Providers)
				pageInfo := terraformProvidersResult.PageInfo
				terraformProviders := terraformProvidersResult.Providers

				// Check the terraform providers result by comparing a list of the terraform provider IDs.
				actualTerraformProviderIDs := []string{}
				for _, terraformProvider := range terraformProviders {
					actualTerraformProviderIDs = append(actualTerraformProviderIDs, terraformProvider.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformProviderIDs)
				}

				assert.Equal(t, len(test.expectTerraformProviderIDs), len(actualTerraformProviderIDs))
				assert.Equal(t, test.expectTerraformProviderIDs, actualTerraformProviderIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one terraform provider returned.
				// If there are no terraform providers returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(terraformProviders) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&terraformProviders[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&terraformProviders[len(terraformProviders)-1])
					assert.Equal(t, test.expectStartCursorError, resultStartCursorError)
					assert.Equal(t, test.expectHasStartCursor, resultStartCursor != nil)
					assert.Equal(t, test.expectEndCursorError, resultEndCursorError)
					assert.Equal(t, test.expectHasEndCursor, resultEndCursor != nil)

					// Capture the ending cursor values for the next case.
					previousEndCursorValue = resultEndCursor
					previousStartCursorValue = resultStartCursor
				}
			}
		})
	}
}

func TestCreateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviders(ctx, testClient, warmupTerraformProviders{
		groups: standardWarmupGroupsForTerraformProviders,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformProvider
		expectCreated *models.TerraformProvider
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
				Private:     true,
				CreatedBy:   "TestCreateProvider",
			},
			expectCreated: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
					TRN:               types.TerraformProviderModelType.BuildTRN(warmupItems.groups[0].FullPath, "terraform-provider-create-test"),
				},
				Name:        "terraform-provider-create-test",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
				Private:     true,
				CreatedBy:   "TestCreateProvider",
			},
		},

		{
			name: "duplicate group ID and Terraform provider name",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("terraform provider with name terraform-provider-create-test already exists"),
		},

		{
			name: "negative, non-existent root group ID",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test-non-existent-root-group-id",
				RootGroupID: nonExistentID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_providers\" violates foreign key constraint \"fk_root_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, non-existent group ID",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test-non-existent-group-id",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_providers\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid root group ID",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test-invalid-root-group-id",
				RootGroupID: invalidID,
				GroupID:     warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: invalidUUIDMsg,
		},

		{
			name: "negative, invalid group ID",
			toCreate: &models.TerraformProvider{
				Name:        "terraform-provider-create-test-invalid-group-id",
				RootGroupID: warmupItems.groups[0].Metadata.ID,
				GroupID:     invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.TerraformProviders.CreateProvider(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformProviders(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviders(ctx, testClient, warmupTerraformProviders{
		groups:             standardWarmupGroupsForTerraformProviders,
		terraformProviders: standardWarmupTerraformProviders,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformProvider
		expectUpdated *models.TerraformProvider
		name          string
	}

	// Looks up by ID and version.  Also requires group ID.
	// Updates name and private.  Returns rebuilt resource path.
	// The NamespacePath field is not updated in the DB, but the value from the argument is returned.
	positiveTerraformProvider := warmupItems.terraformProviders[0]
	positiveGroup := warmupItems.groups[9]
	otherTerraformProvider := warmupItems.terraformProviders[1]
	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProvider.Metadata.ID,
					Version: initialResourceVersion,
				},
				Name:    positiveTerraformProvider.Name,
				Private: !positiveTerraformProvider.Private,
				GroupID: positiveGroup.Metadata.ID,
			},
			expectUpdated: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:                   positiveTerraformProvider.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveTerraformProvider.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
					TRN:                  positiveTerraformProvider.Metadata.TRN,
				},
				Name:        positiveTerraformProvider.Name,
				RootGroupID: positiveTerraformProvider.RootGroupID,
				GroupID:     positiveTerraformProvider.GroupID,
				Private:     !positiveTerraformProvider.Private,
				CreatedBy:   positiveTerraformProvider.CreatedBy,
			},
		},

		{
			name: "would-be-duplicate-group-id-and-provider-name",
			toUpdate: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProvider.Metadata.ID,
					Version: initialResourceVersion,
				},
				// Would duplicate a different Terraform provider.
				Name: otherTerraformProvider.Name,
			},
			expectMsg: ptr.String("resource version does not match specified version"),
		},

		{
			name: "negative, non-existent Terraform provider ID",
			toUpdate: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTerraformProvider, err := testClient.client.TerraformProviders.UpdateProvider(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformProvider)
				compareTerraformProviders(t, test.expectUpdated, actualTerraformProvider, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformProvider)
			}
		})
	}
}

func TestDeleteProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviders(ctx, testClient, warmupTerraformProviders{
		groups:             standardWarmupGroupsForTerraformProviders,
		terraformProviders: standardWarmupTerraformProviders,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformProvider
		name      string
	}

	// Looks up by ID and version.
	positiveTerraformProvider := warmupItems.terraformProviders[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProvider.Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform provider ID",
			toDelete: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformProvider{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviders.DeleteProvider(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformProviders = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform provider functions",
		FullPath:    "top-level-group-0-for-terraform-providers",
		CreatedBy:   "someone-g0",
	},
	{
		Description: "top level group 1 for testing terraform provider functions",
		FullPath:    "top-level-group-1-for-terraform-providers",
		CreatedBy:   "someone-g1",
	},
	{
		Description: "top level group 2 for testing terraform provider functions",
		FullPath:    "top-level-group-2-for-terraform-providers",
		CreatedBy:   "someone-g2",
	},
	{
		Description: "top level group 3 for testing terraform provider functions",
		FullPath:    "top-level-group-3-for-terraform-providers",
		CreatedBy:   "someone-g3",
	},
	{
		Description: "top level group 4 for testing terraform provider functions",
		FullPath:    "top-level-group-4-for-terraform-providers",
		CreatedBy:   "someone-g4",
	},
	// Nested groups:
	{
		Description: "nested group 5 for testing terraform provider functions",
		FullPath:    "top-level-group-4-for-terraform-providers/nested-group-5-for-terraform-providers",
		CreatedBy:   "someone-g5",
	},
	{
		Description: "nested group 6 for testing terraform provider functions",
		FullPath:    "top-level-group-3-for-terraform-providers/nested-group-6-for-terraform-providers",
		CreatedBy:   "someone-g6",
	},
	{
		Description: "nested group 7 for testing terraform provider functions",
		FullPath:    "top-level-group-2-for-terraform-providers/nested-group-7-for-terraform-providers",
		CreatedBy:   "someone-g7",
	},
	{
		Description: "nested group 8 for testing terraform provider functions",
		FullPath:    "top-level-group-1-for-terraform-providers/nested-group-8-for-terraform-providers",
		CreatedBy:   "someone-g8",
	},
	{
		Description: "nested group 9 for testing terraform provider functions",
		FullPath:    "top-level-group-0-for-terraform-providers/nested-group-9-for-terraform-providers",
		CreatedBy:   "someone-g9",
	},
}

// Standard warmup workspaces for tests in this module:
// The create function will derive the group ID and name from the namespace path.
var standardWarmupWorkspacesForTerraformProviders = []models.Workspace{
	{
		Description: "workspace 0 for testing terraform provider functions",
		FullPath:    "top-level-group-0-for-terraform-providers/workspace-0-in-group-0",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing terraform provider functions",
		FullPath:    "top-level-group-1-for-terraform-providers/workspace-1-in-group-1",
		CreatedBy:   "someone-w1",
	},
	{
		Description: "workspace 2 for testing terraform provider functions",
		FullPath:    "top-level-group-2-for-terraform-providers/workspace-2-in-group-2",
		CreatedBy:   "someone-w2",
	},
}

// Standard warmup teams for tests in this module:
var standardWarmupTeamsForTerraformProviders = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for terraform provider tests",
	},
	{
		Name:        "team-b",
		Description: "team b for terraform provider tests",
	},
}

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsersForTerraformProviders = []models.User{
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
var standardWarmupTeamMembersForTerraformProviders = []models.TeamMember{
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
var standardWarmupServiceAccountsForTerraformProviders = []models.ServiceAccount{
	{
		Name:              "service-account-0",
		Description:       "service account 0",
		GroupID:           "top-level-group-2-for-terraform-providers/nested-group-7-for-terraform-providers",
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "service-account-1",
		Description:       "service account 1",
		GroupID:           "top-level-group-1-for-terraform-providers/nested-group-8-for-terraform-providers",
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

// Standard warmup namespace memberships for tests in this module:
// In this variable, the ID field is the user, service account, and team _NAME_, NOT the ID.
var standardWarmupNamespaceMembershipsForTerraformProviders = []CreateNamespaceMembershipInput{
	// Team access to group:
	{
		NamespacePath: "top-level-group-3-for-terraform-providers",
		TeamID:        ptr.String("team-a"),
		RoleID:        "role-a",
	},

	// User access to group:
	{
		NamespacePath: "top-level-group-4-for-terraform-providers",
		UserID:        ptr.String("user-0"),
		RoleID:        "role-b",
	},

	// Service accounts access to group:
	{
		NamespacePath:    "top-level-group-4-for-terraform-providers/nested-group-5-for-terraform-providers",
		ServiceAccountID: ptr.String("service-account-0"),
		RoleID:           "role-c",
	},

	// Team access to workspace:
	{
		NamespacePath: "top-level-group-0-for-terraform-providers/workspace-0-in-group-0",
		TeamID:        ptr.String("team-b"),
		RoleID:        "role-a",
	},

	// User access to workspace:
	{
		NamespacePath: "top-level-group-1-for-terraform-providers/workspace-1-in-group-1",
		UserID:        ptr.String("user-1"),
		RoleID:        "role-b",
	},

	// Service account access to workspace:
	{
		NamespacePath:    "top-level-group-2-for-terraform-providers/workspace-2-in-group-2",
		ServiceAccountID: ptr.String("service-account-1"),
		RoleID:           "role-c",
	},
}

// Standard warmup terraform providers for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformProviders = []models.TerraformProvider{
	{
		// This one is public.
		Name:        "1-terraform-provider-0",
		RootGroupID: "top-level-group-0-for-terraform-providers",
		GroupID:     "top-level-group-0-for-terraform-providers/nested-group-9-for-terraform-providers",
		Private:     false,
		CreatedBy:   "someone-sv0",
	},
	{
		Name:        "1-terraform-provider-1",
		RootGroupID: "top-level-group-1-for-terraform-providers",
		GroupID:     "top-level-group-1-for-terraform-providers",
		Private:     true,
		CreatedBy:   "someone-sv1",
	},
	{
		Name:        "2-terraform-provider-2",
		RootGroupID: "top-level-group-2-for-terraform-providers",
		GroupID:     "top-level-group-2-for-terraform-providers/nested-group-7-for-terraform-providers",
		Private:     true,
		CreatedBy:   "someone-sv2",
	},
	{
		Name:        "2-terraform-provider-3",
		RootGroupID: "top-level-group-3-for-terraform-providers",
		GroupID:     "top-level-group-3-for-terraform-providers",
		Private:     true,
		CreatedBy:   "someone-sv3",
	},
	{
		Name:        "5-terraform-provider-4",
		RootGroupID: "top-level-group-4-for-terraform-providers",
		GroupID:     "top-level-group-4-for-terraform-providers/nested-group-5-for-terraform-providers",
		Private:     true,
		CreatedBy:   "someone-sv4",
	},
}

// Standard warmup roles for tests in this module:
var standardWarmupRolesForTerraformProviders = []models.Role{
	{
		Name:        "role-a",
		Description: "role a for namespace membership tests",
		CreatedBy:   "someone-a",
	},
	{
		Name:        "role-b",
		Description: "role b for namespace membership tests",
		CreatedBy:   "someone-b",
	},
	{
		Name:        "role-c",
		Description: "role c for namespace membership tests",
		CreatedBy:   "someone-c",
	},
}

// createWarmupTerraformProviders creates some warmup terraform providers for a test
// The warmup terraform providers to create can be standard or otherwise.
func createWarmupTerraformProviders(ctx context.Context, testClient *testClient,
	input warmupTerraformProviders,
) (*warmupTerraformProviders, error) {
	// It is necessary to create several groups in order to provide the necessary IDs for the terraform providers.

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

	_, roleName2ID, err := createInitialRoles(ctx, testClient, standardWarmupRolesForTerraformProviders)
	if err != nil {
		return nil, err
	}

	resultNamespaceMemberships, err := createInitialNamespaceMemberships(ctx, testClient,
		teamName2ID, username2ID, parentPath2ID, serviceAccountName2ID, roleName2ID, input.namespaceMembershipsIn)
	if err != nil {
		return nil, err
	}

	resultTerraformProviders, _, err := createInitialTerraformProviders(ctx, testClient,
		input.terraformProviders, parentPath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformProviders{
		groups:                  resultGroups,
		workspaces:              resultWorkspaces,
		teams:                   resultTeams,
		users:                   resultUsers,
		teamMembers:             resultTeamMembers,
		serviceAccounts:         resultServiceAccounts,
		namespaceMembershipsOut: resultNamespaceMemberships,
		terraformProviders:      resultTerraformProviders,
	}, nil
}

func ptrTerraformProviderSortableField(arg TerraformProviderSortableField) *TerraformProviderSortableField {
	return &arg
}

func (wis terraformProviderInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformProviderInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

func (wis terraformProviderInfoNameSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderInfoNameSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderInfoNameSlice) Less(i, j int) bool {
	return wis[i].name < wis[j].name
}

// terraformProviderInfoFromTerraformProviders returns a slice of terraformProviderInfo, not necessarily sorted in any order.
func terraformProviderInfoFromTerraformProviders(terraformProviders []models.TerraformProvider) []terraformProviderInfo {
	result := []terraformProviderInfo{}

	for _, tp := range terraformProviders {
		result = append(result, terraformProviderInfo{
			id:         tp.Metadata.ID,
			name:       tp.Name,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformProviderIDsFromTerraformProviderInfos preserves order
func terraformProviderIDsFromTerraformProviderInfos(terraformProviderInfos []terraformProviderInfo) []string {
	result := []string{}
	for _, terraformProviderInfo := range terraformProviderInfos {
		result = append(result, terraformProviderInfo.id)
	}
	return result
}

// compareTerraformProviders compares two terraform provider objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformProviders(t *testing.T, expected, actual *models.TerraformProvider,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.RootGroupID, actual.RootGroupID)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Private, actual.Private)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
