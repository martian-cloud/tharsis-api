//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

var (
	// Used as the state value.
	warmupOAuthState = uuid.New()
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// vcsProviderInfo aids convenience in accessing the information TestVCSProviders
// needs about the warmup vcs providers.
type vcsProviderInfo struct {
	createTime    time.Time
	updateTime    time.Time
	vcsProviderID string
	name          string
}

// vcsProviderInfoIDSlice makes a slice of vcsProviderInfo sortable by ID string
type vcsProviderInfoIDSlice []vcsProviderInfo

// vcsProviderInfoCreateSlice makes a slice of vcsProviderInfo sortable by creation time
type vcsProviderInfoCreateSlice []vcsProviderInfo

// vcsProviderInfoUpdateSlice makes a slice of vcsProviderInfo sortable by last updated time
type vcsProviderInfoUpdateSlice []vcsProviderInfo

// vcsProviderInfoNameSlice makes a slice of vcsProviderInfo sortable by name
type vcsProviderInfoNameSlice []vcsProviderInfo

// warmupVCSProviders holds the inputs to and outputs from createWarmupVCSProviders.
type warmupVCSProviders struct {
	groups    []models.Group
	providers []models.VCSProvider
}

func TestVCSProviders_GetProviderByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			standardWarmupVCSProviders,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()

	type testCase struct {
		expectVCSProvider *models.VCSProvider
		expectMsg         *string
		name              string
		searchID          string
	}

	positiveVCSProvider := warmupItems.providers[0]
	testCases := []testCase{
		{
			name:              "positive",
			searchID:          positiveVCSProvider.Metadata.ID,
			expectVCSProvider: &positiveVCSProvider,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect vcs provider and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualVCSProvider, err :=
				testClient.client.VCSProviders.GetProviderByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectVCSProvider != nil {
				require.NotNil(t, actualVCSProvider)
				compareVCSProviders(t, test.expectVCSProvider, actualVCSProvider, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualVCSProvider)
			}

		})
	}
}

func TestGetProviderByOAuthState(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			standardWarmupVCSProviders,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()

	type testCase struct {
		expectVCSProvider *models.VCSProvider
		expectMsg         *string
		name              string
		searchID          string
	}

	positiveVCSProvider := warmupItems.providers[0]
	testCases := []testCase{
		{
			name:              "positive",
			searchID:          *positiveVCSProvider.OAuthState,
			expectVCSProvider: &positiveVCSProvider,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect vcs provider and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualVCSProvider, err :=
				testClient.client.VCSProviders.GetProviderByOAuthState(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectVCSProvider != nil {
				require.NotNil(t, actualVCSProvider)
				compareVCSProviders(t, test.expectVCSProvider, actualVCSProvider, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualVCSProvider)
			}

		})
	}
}

func TestVCSProviders_GetProviders(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			standardWarmupVCSProviders,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	allVCSProviderInfos := vcsProviderInfoFromVCSProviders(warmupItems.providers)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(vcsProviderInfoIDSlice(allVCSProviderInfos))
	allVCSProviderIDs := vcsProviderIDsFromVCSProviderInfos(allVCSProviderInfos)

	// Sort by creation times.
	sort.Sort(vcsProviderInfoCreateSlice(allVCSProviderInfos))
	allVCSProviderIDsByCreateTime := vcsProviderIDsFromVCSProviderInfos(allVCSProviderInfos)
	reverseVCSProviderIDsByCreateTime := reverseStringSlice(allVCSProviderIDsByCreateTime)

	// Sort by last update times.
	sort.Sort(vcsProviderInfoUpdateSlice(allVCSProviderInfos))
	allVCSProviderIDsByUpdateTime := vcsProviderIDsFromVCSProviderInfos(allVCSProviderInfos)
	reverseVCSProviderIDsByUpdateTime := reverseStringSlice(allVCSProviderIDsByUpdateTime)

	// Sort by names.
	sort.Sort(vcsProviderInfoNameSlice(allVCSProviderInfos))
	allVCSProviderIDsByName := vcsProviderIDsFromVCSProviderInfos(allVCSProviderInfos)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetVCSProvidersInput
		name                        string
		expectPageInfo              PageInfo
		expectVCSProviderIDs        []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetVCSProvidersInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectVCSProviderIDs    	[]string
		expectPageInfo              PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{

		// nil input likely causes a nil pointer dereference in GetVCSProvidersInput, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetVCSProvidersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectVCSProviderIDs: allVCSProviderIDs,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectVCSProviderIDs: allVCSProviderIDsByCreateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtDesc),
			},
			expectVCSProviderIDs: reverseVCSProviderIDsByCreateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectVCSProviderIDs: allVCSProviderIDsByUpdateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtDesc),
			},
			expectVCSProviderIDs: reverseVCSProviderIDsByUpdateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByUpdateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByUpdateTime[:2],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVCSProviderIDs:       allVCSProviderIDsByUpdateTime[2:4],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVCSProviderIDs:       allVCSProviderIDsByUpdateTime[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
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
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectVCSProviderIDs: reverseVCSProviderIDsByUpdateTime[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
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
			input: &GetVCSProvidersInput{
				Sort:              ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectVCSProviderIDs:        []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			// If there were more filter fields, this would allow nothing through the filters.
			name: "fully-populated types, everything allowed through filters",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &VCSProviderFilter{
					Search: ptr.String(""),
					// Passing an empty slice to NamespacePaths likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// NamespacePaths: []string{},
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByCreateTime,
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allVCSProviderIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, empty string",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					Search: ptr.String(""),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByName,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 1",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					Search: ptr.String("1"),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByName[0:2],
			expectPageInfo:       PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 2",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					Search: ptr.String("2"),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByName[2:4],
			expectPageInfo:       PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 5",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					Search: ptr.String("5"),
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByName[4:],
			expectPageInfo:       PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectVCSProviderIDs: []string{},
			expectPageInfo:       PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs provider ids, positive",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					VCSProviderIDs: allVCSProviderIDs,
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByCreateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(5), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs providers ids, non existent, negative",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					VCSProviderIDs: []string{nonExistentID},
				},
			},
			expectVCSProviderIDs: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs provider ids, invalid id, negative",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					VCSProviderIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					NamespacePaths: []string{"top-level-group-0-for-vcs-providers"},
				},
			},
			expectVCSProviderIDs: allVCSProviderIDsByName,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allVCSProviderIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, negative",
			input: &GetVCSProvidersInput{
				Sort: ptrVCSProviderSortableField(VCSProviderSortableFieldCreatedAtAsc),
				Filter: &VCSProviderFilter{
					NamespacePaths: []string{"top-level-group-9-for-vcs-providers"},
				},
			},
			expectVCSProviderIDs: []string{},
			expectPageInfo:       PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},
	}

	// Combinations of filter conditions are not (yet) tested.

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

			vcsProvidersActual, err := testClient.client.VCSProviders.GetProviders(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, vcsProvidersActual.PageInfo)
				assert.NotNil(t, vcsProvidersActual.VCSProviders)
				pageInfo := vcsProvidersActual.PageInfo
				vcsProviders := vcsProvidersActual.VCSProviders

				// Check the vcs providers result by comparing a list of the vcs provider IDs.
				actualVCSProviderIDs := []string{}
				for _, vcsProvider := range vcsProviders {
					actualVCSProviderIDs = append(actualVCSProviderIDs, vcsProvider.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualVCSProviderIDs)
				}

				assert.Equal(t, len(test.expectVCSProviderIDs), len(actualVCSProviderIDs))
				assert.Equal(t, test.expectVCSProviderIDs, actualVCSProviderIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one vcs provider returned.
				// If there are no vcs providers returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(vcsProviders) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&vcsProviders[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&vcsProviders[len(vcsProviders)-1])
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

func TestVCSProviders_CreateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			[]models.VCSProvider{},
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	warmupGroup := warmupItems.groups[0]
	warmupGroupID := warmupGroup.Metadata.ID

	type testCase struct {
		toCreate      *models.VCSProvider
		expectCreated *models.VCSProvider
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{

		{
			name: "positive, nearly empty",
			toCreate: &models.VCSProvider{
				Name:              "positive-create-vcs-provider-nearly-empty",
				GroupID:           warmupGroupID,
				Hostname:          "github.com",
				OAuthClientID:     "a-client-id",
				OAuthClientSecret: "a-client-secret",
				OAuthState:        ptr.String(warmupOAuthState.String()),
				Type:              models.GitHubProviderType,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:              "positive-create-vcs-provider-nearly-empty",
				GroupID:           warmupGroupID,
				ResourcePath:      warmupGroup.FullPath + "/positive-create-vcs-provider-nearly-empty",
				Hostname:          "github.com",
				OAuthClientID:     "a-client-id",
				OAuthClientSecret: "a-client-secret",
				OAuthState:        ptr.String(warmupOAuthState.String()),
				Type:              models.GitHubProviderType,
			},
		},

		{
			name: "positive full",
			toCreate: &models.VCSProvider{
				Name:               "positive-create-vcs-provider-full",
				Description:        "positive create vcs provider",
				GroupID:            warmupGroupID,
				Hostname:           "github.com",
				OAuthClientID:      "a-client-id",
				OAuthClientSecret:  "a-client-secret",
				OAuthState:         ptr.String(warmupOAuthState.String()),
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: false,
				CreatedBy:          "creator-of-vcs-providers",
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ResourcePath:       warmupGroup.FullPath + "/positive-create-vcs-provider-full",
				Name:               "positive-create-vcs-provider-full",
				Description:        "positive create vcs provider",
				GroupID:            warmupGroupID,
				Hostname:           "github.com",
				OAuthClientID:      "a-client-id",
				OAuthClientSecret:  "a-client-secret",
				OAuthState:         ptr.String(warmupOAuthState.String()),
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: false,
				CreatedBy:          "creator-of-vcs-providers",
			},
		},

		{
			name: "duplicate name in same group",
			toCreate: &models.VCSProvider{
				Name:              "positive-create-vcs-provider-nearly-empty",
				GroupID:           warmupGroupID,
				OAuthClientID:     "a-client-id",
				OAuthClientSecret: "a-client-secret",
				OAuthState:        ptr.String(warmupOAuthState.String()),
				// Resource path is not used when creating the object, but it is returned.
			},
			expectMsg: ptr.String("vcs provider already exists in the specified group"),
		},

		{
			name: "non-existent group ID",
			toCreate: &models.VCSProvider{
				Name:              "non-existent-group-id",
				GroupID:           nonExistentID,
				OAuthClientID:     "a-client-id",
				OAuthClientSecret: "a-client-secret",
				OAuthState:        ptr.String(warmupOAuthState.String()),
			},
			expectMsg: ptr.String("invalid group: the specified group does not exist"),
		},

		{
			name: "defective group ID",
			toCreate: &models.VCSProvider{
				Name:    "non-existent-group-id",
				GroupID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.VCSProviders.CreateProvider(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareVCSProviders(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestVCSProviders_UpdateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			standardWarmupVCSProviders,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()
	warmupGroup := warmupItems.groups[0]

	type testCase struct {
		toUpdate          *models.VCSProvider
		expectVCSProvider *models.VCSProvider
		expectMsg         *string
		name              string
	}

	// Do only one positive test case, because the logic is theoretically the same for all vcs providers.
	now := currentTime()
	positiveVCSProvider := warmupItems.providers[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:      positiveVCSProvider.Metadata.ID,
					Version: positiveVCSProvider.Metadata.Version,
				},
				Description:       "updated description",
				OAuthState:        ptr.String(warmupOAuthState.String()),
				OAuthAccessToken:  ptr.String("an-oauth-token"),
				OAuthClientID:     "new-client-id",
				OAuthClientSecret: "new-client-secret",
			},
			expectVCSProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:                   positiveVCSProvider.Metadata.ID,
					Version:              positiveVCSProvider.Metadata.Version + 1,
					CreationTimestamp:    positiveVCSProvider.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ResourcePath:      warmupGroup.FullPath + "/" + positiveVCSProvider.Name,
				Name:              "1-vcs-provider-0",
				Description:       "updated description",
				GroupID:           warmupGroup.Metadata.ID,
				Hostname:          positiveVCSProvider.Hostname,
				Type:              models.GitHubProviderType,
				OAuthClientID:     "new-client-id",
				OAuthClientSecret: "new-client-secret",
				OAuthState:        ptr.String(warmupOAuthState.String()),
				OAuthAccessToken:  ptr.String("an-oauth-token"),
				CreatedBy:         positiveVCSProvider.CreatedBy,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveVCSProvider.Metadata.Version,
				},
				OAuthState:       ptr.String(uuid.New().String()),
				OAuthAccessToken: ptr.String("an-oauth-token"),
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveVCSProvider.Metadata.Version,
				},
				OAuthState:       ptr.String(uuid.New().String()),
				OAuthAccessToken: ptr.String("an-oauth-token"),
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualVCSProvider, err :=
				testClient.client.VCSProviders.UpdateProvider(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := currentTime()
			if test.expectVCSProvider != nil {
				require.NotNil(t, actualVCSProvider)
				compareVCSProviders(t, test.expectVCSProvider, actualVCSProvider, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualVCSProvider)
			}
		})
	}
}

func TestVCSProviders_DeleteProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupVCSProviders(ctx, testClient,
		warmupVCSProviders{
			standardWarmupGroupsForVCSProviders,
			standardWarmupVCSProviders,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toDelete  *models.VCSProvider
		expectMsg *string
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.providers[0].Metadata.ID,
					Version: warmupItems.providers[0].Metadata.Version,
				},
			},
		},

		{
			name: "negative, non-existent ID",
			toDelete: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toDelete: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
				Description: "looking for a defective ID",
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.VCSProviders.DeleteProvider(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)

		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForVCSProviders = []models.Group{
	{
		Description: "top level group 0 for testing vcs provider functions",
		FullPath:    "top-level-group-0-for-vcs-providers",
		CreatedBy:   "someone-g0",
	},
}

var standardWarmupVCSProviders = []models.VCSProvider{
	{
		Name:              "1-vcs-provider-0",
		Description:       "vcs provider 0 for testing vcs providers",
		GroupID:           "top-level-group-0-for-vcs-providers",
		CreatedBy:         "someone-vp0",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "1-vcs-provider-1",
		Description:       "vcs provider 1 for testing vcs providers",
		GroupID:           "top-level-group-0-for-vcs-providers",
		CreatedBy:         "someone-vp1",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "2-vcs-provider-2",
		Description:       "vcs provider 2 for testing vcs providers",
		GroupID:           "top-level-group-0-for-vcs-providers",
		CreatedBy:         "someone-vp2",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "2-vcs-provider-3",
		Description:       "vcs provider 3 for testing vcs providers",
		GroupID:           "top-level-group-0-for-vcs-providers",
		CreatedBy:         "someone-vp3",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "5-vcs-provider-4",
		Description:       "vcs provider 4 for testing vcs providers",
		GroupID:           "top-level-group-0-for-vcs-providers",
		CreatedBy:         "someone-vp4",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
}

// createWarmupVCSProviders creates vcs providers for testing.
func createWarmupVCSProviders(ctx context.Context, testClient *testClient,
	input warmupVCSProviders) (*warmupVCSProviders, error) {

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultVCSProviders, err := createInitialVCSProviders(ctx, testClient,
		groupPath2ID, input.providers)
	if err != nil {
		return nil, err
	}

	return &warmupVCSProviders{
		groups:    resultGroups,
		providers: resultVCSProviders,
	}, nil
}

// createInitialVCSProviders creates some warmup vcs providers for a test.
func createInitialVCSProviders(ctx context.Context, testClient *testClient,
	groupMap map[string]string, toCreate []models.VCSProvider) (
	[]models.VCSProvider, error) {
	result := []models.VCSProvider{}

	for _, input := range toCreate {
		input.GroupID = groupMap[input.GroupID]
		created, err := testClient.client.VCSProviders.CreateProvider(ctx, &input)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial vcs provider: %s", err)
		}

		result = append(result, *created)
	}

	// In order to make the created-at and last-updated-at orders differ,
	// update every third object without changing any values.
	for ix, toUpdate := range result {
		if ix%3 == 0 {
			updated, err := testClient.client.VCSProviders.UpdateProvider(ctx, &toUpdate)
			if err != nil {
				return nil, fmt.Errorf("failed to update initial vcs provider: %s", err)
			}
			result[ix] = *updated
		}
	}

	return result, nil
}

func ptrVCSProviderSortableField(arg VCSProviderSortableField) *VCSProviderSortableField {
	return &arg
}

func (vp vcsProviderInfoIDSlice) Len() int {
	return len(vp)
}

func (vp vcsProviderInfoIDSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsProviderInfoIDSlice) Less(i, j int) bool {
	return vp[i].vcsProviderID < vp[j].vcsProviderID
}

func (vp vcsProviderInfoCreateSlice) Len() int {
	return len(vp)
}

func (vp vcsProviderInfoCreateSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsProviderInfoCreateSlice) Less(i, j int) bool {
	return vp[i].createTime.Before(vp[j].createTime)
}

func (vp vcsProviderInfoUpdateSlice) Len() int {
	return len(vp)
}

func (vp vcsProviderInfoUpdateSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsProviderInfoUpdateSlice) Less(i, j int) bool {
	return vp[i].updateTime.Before(vp[j].updateTime)
}

func (vp vcsProviderInfoNameSlice) Len() int {
	return len(vp)
}

func (vp vcsProviderInfoNameSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsProviderInfoNameSlice) Less(i, j int) bool {
	return vp[i].name < vp[j].name
}

// vcsProviderInfoFromVCSProviders returns a slice of vcsProviderInfo, not necessarily sorted in any order.
func vcsProviderInfoFromVCSProviders(vcsProviders []models.VCSProvider) []vcsProviderInfo {
	result := []vcsProviderInfo{}

	for _, vcsProvider := range vcsProviders {
		result = append(result, vcsProviderInfo{
			createTime:    *vcsProvider.Metadata.CreationTimestamp,
			updateTime:    *vcsProvider.Metadata.LastUpdatedTimestamp,
			vcsProviderID: vcsProvider.Metadata.ID,
			name:          vcsProvider.Name,
		})
	}

	return result
}

// vcsProviderIDsFromVCSProviderInfos preserves order
func vcsProviderIDsFromVCSProviderInfos(vcsProviderInfos []vcsProviderInfo) []string {
	result := []string{}
	for _, vcsProviderInfos := range vcsProviderInfos {
		result = append(result, vcsProviderInfos.vcsProviderID)
	}

	return result
}

// compareVCSProviders compares two vcs provider objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareVCSProviders(t *testing.T, expected, actual *models.VCSProvider,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.ResourcePath, actual.ResourcePath)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.Hostname, actual.Hostname)
	assert.Equal(t, expected.AutoCreateWebhooks, actual.AutoCreateWebhooks)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.OAuthClientID, actual.OAuthClientID)
	assert.Equal(t, expected.OAuthClientSecret, actual.OAuthClientSecret)
	assert.Equal(t, expected.OAuthState, actual.OAuthState)
	assert.Equal(t, expected.OAuthAccessToken, actual.OAuthAccessToken)
	assert.Equal(t, expected.OAuthRefreshToken, actual.OAuthRefreshToken)
	assert.Equal(t, expected.OAuthAccessTokenExpiresAt, actual.OAuthAccessTokenExpiresAt)

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
