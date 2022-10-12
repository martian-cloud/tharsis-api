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
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// terraformProviderVersionInfo aids convenience in accessing the information
// TestGetTerraformProviderVersions needs about the warmup objects.
type terraformProviderVersionInfo struct {
	updateTime time.Time
	id         string
}

// terraformProviderVersionInfoIDSlice makes a slice of terraformProviderVersionInfo sortable by ID string
type terraformProviderVersionInfoIDSlice []terraformProviderVersionInfo

// terraformProviderVersionInfoUpdateSlice makes a slice of terraformProviderVersionInfo sortable by last updated time
type terraformProviderVersionInfoUpdateSlice []terraformProviderVersionInfo

// warmupTerraformProviderVersions holds the inputs to and outputs from createWarmupTerraformProviderVersions.
type warmupTerraformProviderVersions struct {
	groups                    []models.Group
	workspaces                []models.Workspace
	teams                     []models.Team
	users                     []models.User
	teamMembers               []models.TeamMember
	serviceAccounts           []models.ServiceAccount
	namespaceMembershipsIn    []CreateNamespaceMembershipInput
	namespaceMembershipsOut   []models.NamespaceMembership
	terraformProviders        []models.TerraformProvider
	terraformProviderVersions []models.TerraformProviderVersion
}

func TestGetProviderVersionByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupItems, err := createWarmupTerraformProviderVersions(ctx, testClient, warmupTerraformProviderVersions{
		groups:                    standardWarmupGroupsForTerraformProviderVersions,
		terraformProviders:        standardWarmupTerraformProvidersForTerraformProviderVersions,
		terraformProviderVersions: standardWarmupTerraformProviderVersions,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := time.Now()

	type testCase struct {
		expectMsg                      *string
		expectTerraformProviderVersion *models.TerraformProviderVersion
		name                           string
		searchID                       string
	}

	positiveTerraformProviderVersion := warmupItems.terraformProviderVersions[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveTerraformProviderVersion.Metadata.ID,
			expectTerraformProviderVersion: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:                positiveTerraformProviderVersion.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				SemanticVersion:          positiveTerraformProviderVersion.SemanticVersion,
				GPGASCIIArmor:            positiveTerraformProviderVersion.GPGASCIIArmor,
				GPGKeyID:                 positiveTerraformProviderVersion.GPGKeyID,
				ProviderID:               positiveTerraformProviderVersion.ProviderID,
				Protocols:                positiveTerraformProviderVersion.Protocols,
				SHASumsUploaded:          positiveTerraformProviderVersion.SHASumsUploaded,
				SHASumsSignatureUploaded: positiveTerraformProviderVersion.SHASumsSignatureUploaded,
				CreatedBy:                positiveTerraformProviderVersion.CreatedBy,
			},
		},

		{
			name:     "negative, non-existent Terraform provider version ID",
			searchID: nonExistentID,
			// expect terraform provider version and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformProviderVersion, err := testClient.client.TerraformProviderVersions.GetProviderVersionByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformProviderVersion != nil {
				require.NotNil(t, actualTerraformProviderVersion)
				compareTerraformProviderVersions(t, test.expectTerraformProviderVersion, actualTerraformProviderVersion, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualTerraformProviderVersion)
			}
		})
	}
}

func TestGetProviderVersions(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersions(ctx, testClient, warmupTerraformProviderVersions{
		groups:                    standardWarmupGroupsForTerraformProviderVersions,
		terraformProviders:        standardWarmupTerraformProvidersForTerraformProviderVersions,
		terraformProviderVersions: standardWarmupTerraformProviderVersions,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	allTerraformProviderVersionInfos := terraformProviderVersionInfoFromTerraformProviderVersions(warmupItems.terraformProviderVersions)

	// Sort by Terraform provider version IDs.
	sort.Sort(terraformProviderVersionInfoIDSlice(allTerraformProviderVersionInfos))
	allTerraformProviderVersionIDs := terraformProviderVersionIDsFromTerraformProviderVersionInfos(allTerraformProviderVersionInfos)

	// Sort by last update times.
	sort.Sort(terraformProviderVersionInfoUpdateSlice(allTerraformProviderVersionInfos))
	allTerraformProviderVersionIDsByTime := terraformProviderVersionIDsFromTerraformProviderVersionInfos(allTerraformProviderVersionInfos)
	reverseTerraformProviderVersionIDsByTime := reverseStringSlice(allTerraformProviderVersionIDsByTime)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError            error
		expectEndCursorError              error
		input                             *GetProviderVersionsInput
		expectMsg                         *string
		name                              string
		expectPageInfo                    PageInfo
		expectTerraformProviderVersionIDs []string
		getBeforeCursorFromPrevious       bool
		sortedDescending                  bool
		expectHasStartCursor              bool
		getAfterCursorFromPrevious        bool
		expectHasEndCursor                bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetProviderVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectTerraformProviderVersionIDs:  []string{},
			expectPageInfo: PageInfo{
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

		// nil input likely causes a nil pointer dereference in GetProviderVersions, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetProviderVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDs,
			expectPageInfo:                    PageInfo{TotalCount: int32(len(allTerraformProviderVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime,
			expectPageInfo:                    PageInfo{TotalCount: int32(len(allTerraformProviderVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime,
			expectPageInfo:                    PageInfo{TotalCount: int32(len(allTerraformProviderVersionIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtDesc),
			},
			sortedDescending:                  true,
			expectTerraformProviderVersionIDs: reverseTerraformProviderVersionIDsByTime,
			expectPageInfo:                    PageInfo{TotalCount: int32(len(allTerraformProviderVersionIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "pagination: everything at once",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime,
			expectPageInfo:                    PageInfo{TotalCount: int32(len(allTerraformProviderVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "pagination: first two",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[:2],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTerraformProviderVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:        true,
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[2:4],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTerraformProviderVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:        true,
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTerraformProviderVersionIDs)),
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
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:                  true,
			expectTerraformProviderVersionIDs: reverseTerraformProviderVersionIDsByTime[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTerraformProviderVersionIDs)),
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
			input: &GetProviderVersionsInput{
				Sort:              ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:        true,
			getBeforeCursorFromPrevious:       true,
			expectMsg:                         ptr.String("only before or after can be defined, not both"),
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                         ptr.String("only first or last can be defined, not both"),
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDs[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTerraformProviderVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &TerraformProviderVersionFilter{
					ProviderID:               ptr.String(""),
					SHASumsUploaded:          ptr.Bool(true),
					SHASumsSignatureUploaded: ptr.Bool(true),
					SemanticVersion:          ptr.String(""),
					// Passing an empty slice to ProviderVersionIDs likely causes
					// an SQL syntax error ("... IN ()"), so don't try it.
					// ProviderVersionsIDs: []string{},
				},
			},
			expectMsg:                         emptyUUIDMsg2,
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{},
		},

		{
			name: "filter, provider ID, positive",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderID: ptr.String(warmupItems.terraformProviderVersions[0].ProviderID),
				},
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[0:2],
			expectPageInfo:                    PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "filter, provider ID, non-existent",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider ID, invalid",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderID: ptr.String(invalidID),
				},
			},
			expectMsg:                         invalidUUIDMsg2,
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{},
		},

		{
			name: "filter, SHA sums uploaded, true",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SHASumsUploaded: ptr.Bool(true),
				},
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[2:4],
			expectPageInfo:                    PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "filter, SHA sums uploaded, false",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SHASumsUploaded: ptr.Bool(false),
				},
			},
			expectTerraformProviderVersionIDs: []string{
				allTerraformProviderVersionIDsByTime[0],
				allTerraformProviderVersionIDsByTime[1],
				allTerraformProviderVersionIDsByTime[4],
			},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, SHA sums signature uploaded, true",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SHASumsSignatureUploaded: ptr.Bool(true),
				},
			},
			expectTerraformProviderVersionIDs: []string{
				allTerraformProviderVersionIDsByTime[1],
				allTerraformProviderVersionIDsByTime[3],
			},
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, SHA sums signature uploaded, false",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SHASumsSignatureUploaded: ptr.Bool(false),
				},
			},
			expectTerraformProviderVersionIDs: []string{
				allTerraformProviderVersionIDsByTime[0],
				allTerraformProviderVersionIDsByTime[2],
				allTerraformProviderVersionIDsByTime[4],
			},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, semantic version, positive",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SemanticVersion: ptr.String("3.4.5"),
				},
			},
			expectTerraformProviderVersionIDs: allTerraformProviderVersionIDsByTime[2:3],
			expectPageInfo:                    PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:              true,
			expectHasEndCursor:                true,
		},

		{
			name: "filter, semantic version, non-existent",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SemanticVersion: ptr.String("9.8.7"),
				},
			},
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, semantic version, invalid",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					SemanticVersion: ptr.String("this-is-not-a-valid-semantic-version"),
				},
			},
			// expect no error, just an empty return slice
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider version IDs, positive",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderVersionIDs: []string{
						allTerraformProviderVersionIDsByTime[0],
						allTerraformProviderVersionIDsByTime[3],
					},
				},
			},
			expectTerraformProviderVersionIDs: []string{
				allTerraformProviderVersionIDsByTime[0],
				allTerraformProviderVersionIDsByTime[3],
			},
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, provider version IDs, non-existent",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderVersionIDs: []string{nonExistentID},
				},
			},
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider version IDs, invalid",
			input: &GetProviderVersionsInput{
				Sort: ptrTerraformProviderVersionSortableField(TerraformProviderVersionSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderVersionFilter{
					ProviderVersionIDs: []string{invalidID},
				},
			},
			expectMsg:                         invalidUUIDMsg2,
			expectTerraformProviderVersionIDs: []string{},
			expectPageInfo:                    PageInfo{},
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

			terraformProviderVersionsResult, err := testClient.client.TerraformProviderVersions.GetProviderVersions(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, terraformProviderVersionsResult.PageInfo)
				assert.NotNil(t, terraformProviderVersionsResult.ProviderVersions)
				pageInfo := terraformProviderVersionsResult.PageInfo
				terraformProviderVersions := terraformProviderVersionsResult.ProviderVersions

				// Check the terraform provider versions result by comparing a list of the terraform provider version IDs.
				actualTerraformProviderVersionIDs := []string{}
				for _, terraformProviderVersion := range terraformProviderVersions {
					actualTerraformProviderVersionIDs = append(actualTerraformProviderVersionIDs, terraformProviderVersion.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformProviderVersionIDs)
				}

				assert.Equal(t, len(test.expectTerraformProviderVersionIDs), len(actualTerraformProviderVersionIDs))
				assert.Equal(t, test.expectTerraformProviderVersionIDs, actualTerraformProviderVersionIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one terraform provider version returned.
				// If there are no terraform provider versions returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(terraformProviderVersions) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&terraformProviderVersions[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&terraformProviderVersions[len(terraformProviderVersions)-1])
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

func TestCreateProviderVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersions(ctx, testClient, warmupTerraformProviderVersions{
		groups:             standardWarmupGroupsForTerraformProviderVersions,
		terraformProviders: standardWarmupTerraformProvidersForTerraformProviderVersions,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toCreate      *models.TerraformProviderVersion
		expectCreated *models.TerraformProviderVersion
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformProviderVersion{
				ProviderID:               warmupItems.terraformProviders[0].Metadata.ID,
				SemanticVersion:          "2.4.6",
				GPGASCIIArmor:            ptr.String("chain-mail-test-create"),
				GPGKeyID:                 ptr.Uint64(888222333444555666),
				Protocols:                []string{"protocol-42", "protocol-43"},
				SHASumsUploaded:          false,
				SHASumsSignatureUploaded: true,
				CreatedBy:                "TestCreateProviderVersion",
			},
			expectCreated: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ProviderID:               warmupItems.terraformProviders[0].Metadata.ID,
				SemanticVersion:          "2.4.6",
				GPGASCIIArmor:            ptr.String("chain-mail-test-create"),
				GPGKeyID:                 ptr.Uint64(888222333444555666),
				Protocols:                []string{"protocol-42", "protocol-43"},
				SHASumsUploaded:          false,
				SHASumsSignatureUploaded: true,
				CreatedBy:                "TestCreateProviderVersion",
			},
		},

		{
			name: "duplicate provider ID and semantic version",
			toCreate: &models.TerraformProviderVersion{
				ProviderID:      warmupItems.terraformProviders[0].Metadata.ID,
				SemanticVersion: "2.4.6",
				CreatedBy:       "would-be-duplicate-provider-id-and-semantic-version",
			},
			expectMsg: ptr.String("terraform provider version 2.4.6 already exists"),
		},

		{
			name: "negative, non-existent provider ID",
			toCreate: &models.TerraformProviderVersion{
				ProviderID:      nonExistentID,
				SemanticVersion: "2.4.9",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_provider_versions\" violates foreign key constraint \"fk_provider_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid provider ID",
			toCreate: &models.TerraformProviderVersion{
				ProviderID:      invalidID,
				SemanticVersion: "2.5.9",
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformProviderVersions(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateProviderVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersions(ctx, testClient, warmupTerraformProviderVersions{
		groups:                    standardWarmupGroupsForTerraformProviderVersions,
		terraformProviders:        standardWarmupTerraformProvidersForTerraformProviderVersions,
		terraformProviderVersions: standardWarmupTerraformProviderVersions,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformProviderVersion
		expectUpdated *models.TerraformProviderVersion
		name          string
	}

	// Looks up by ID and version.
	// Updates GPGKeyID, GPGASCIIArmor, protocols, SHASumsUploaded, SHASumsSignatureUploaded.
	positiveTerraformProviderVersion := warmupItems.terraformProviderVersions[0]
	otherTerraformProviderVersion := warmupItems.terraformProviderVersions[1]
	now := time.Now()
	testCases := []testCase{

		{
			name: "positive",
			toUpdate: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProviderVersion.Metadata.ID,
					Version: initialResourceVersion,
				},
				GPGASCIIArmor:            ptr.String("chain-mail-test-update"),
				GPGKeyID:                 ptr.Uint64(999222333444555666),
				Protocols:                []string{"protocol-95", "protocol-96"},
				SHASumsUploaded:          true,
				SHASumsSignatureUploaded: true,
			},
			expectUpdated: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:                   positiveTerraformProviderVersion.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveTerraformProviderVersion.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ProviderID:               positiveTerraformProviderVersion.ProviderID,
				SemanticVersion:          "1.2.3",
				GPGASCIIArmor:            ptr.String("chain-mail-test-update"),
				GPGKeyID:                 ptr.Uint64(999222333444555666),
				Protocols:                []string{"protocol-95", "protocol-96"},
				SHASumsUploaded:          true,
				SHASumsSignatureUploaded: true,
				CreatedBy:                "someone-tpv0",
			},
		},

		{
			name: "would-be duplicate provider ID and semantic version",
			toUpdate: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProviderVersion.Metadata.ID,
					Version: initialResourceVersion,
				},
				// Would duplicate a different Terraform provider version.
				SemanticVersion: otherTerraformProviderVersion.SemanticVersion,
			},
			expectMsg: ptr.String("resource version does not match specified version"),
		},

		{
			name: "negative, non-existent Terraform provider version ID",
			toUpdate: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformProviderVersion{
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

			actualTerraformProviderVersion, err := testClient.client.TerraformProviderVersions.UpdateProviderVersion(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformProviderVersion)
				compareTerraformProviderVersions(t, test.expectUpdated, actualTerraformProviderVersion, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformProviderVersion)
			}
		})
	}
}

func TestDeleteProviderVersion(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersions(ctx, testClient, warmupTerraformProviderVersions{
		groups:                    standardWarmupGroupsForTerraformProviderVersions,
		terraformProviders:        standardWarmupTerraformProvidersForTerraformProviderVersions,
		terraformProviderVersions: standardWarmupTerraformProviderVersions,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformProviderVersion
		name      string
	}

	// Looks up by ID and version.
	positiveTerraformProviderVersion := warmupItems.terraformProviderVersions[0]
	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProviderVersion.Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform provider version ID",
			toDelete: &models.TerraformProviderVersion{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformProviderVersion{
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

			err := testClient.client.TerraformProviderVersions.DeleteProviderVersion(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformProviderVersions = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform provider version functions",
		FullPath:    "top-level-group-0-for-terraform-provider-versions",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup terraform providers for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformProvidersForTerraformProviderVersions = []models.TerraformProvider{
	{
		Name:        "terraform-provider-0",
		RootGroupID: "top-level-group-0-for-terraform-provider-versions",
		GroupID:     "top-level-group-0-for-terraform-provider-versions",
		Private:     false,
		CreatedBy:   "someone-tp0",
		// ResourcePath: "top-level-group-0-for-terraform-provider-versions/terraform-provider-0",
	},
	{
		Name:        "terraform-provider-1",
		RootGroupID: "top-level-group-0-for-terraform-provider-versions",
		GroupID:     "top-level-group-0-for-terraform-provider-versions",
		Private:     false,
		CreatedBy:   "someone-tp1",
		// ResourcePath: "top-level-group-0-for-terraform-provider-versions/terraform-provider-1",
	},
}

// Standard warmup terraform provider versions for tests in this module:
// The necessary ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformProviderVersions = []models.TerraformProviderVersion{
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-versions/terraform-provider-0",
		SemanticVersion:          "1.2.3",
		GPGASCIIArmor:            ptr.String("chain-mail-0"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-0", "protocol-1"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv0",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-versions/terraform-provider-0",
		SemanticVersion:          "2.3.4",
		GPGASCIIArmor:            ptr.String("chain-mail-1"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-2", "protocol-3"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: true,
		CreatedBy:                "someone-tpv1",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-versions/terraform-provider-1",
		SemanticVersion:          "3.4.5",
		GPGASCIIArmor:            ptr.String("chain-mail-2"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-4", "protocol-5"},
		SHASumsUploaded:          true,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv2",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-versions/terraform-provider-1",
		SemanticVersion:          "4.5.6",
		GPGASCIIArmor:            ptr.String("chain-mail-3"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-6", "protocol-7"},
		SHASumsUploaded:          true,
		SHASumsSignatureUploaded: true,
		CreatedBy:                "someone-tpv3",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-versions/terraform-provider-1",
		SemanticVersion:          "5.6.7",
		GPGASCIIArmor:            ptr.String("chain-mail-4"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-8", "protocol-9"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv4",
	},
}

// createWarmupTerraformProviderVersions creates some warmup terraform provider versions for a test
// The warmup terraform provider versions to create can be standard or otherwise.
func createWarmupTerraformProviderVersions(ctx context.Context, testClient *testClient,
	input warmupTerraformProviderVersions) (*warmupTerraformProviderVersions, error) {

	// It is necessary to create at least one group in order to
	// provide the necessary IDs for the terraform provider versions.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultTerraformProviders, providerResourcePath2ID, err := createInitialTerraformProviders(ctx, testClient,
		input.terraformProviders, parentPath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderVersions, _, err := createInitialTerraformProviderVersions(ctx, testClient,
		input.terraformProviderVersions, providerResourcePath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformProviderVersions{
		groups:                    resultGroups,
		terraformProviders:        resultTerraformProviders,
		terraformProviderVersions: resultTerraformProviderVersions,
	}, nil
}

func ptrTerraformProviderVersionSortableField(arg TerraformProviderVersionSortableField) *TerraformProviderVersionSortableField {
	return &arg
}

func (wis terraformProviderVersionInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderVersionInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderVersionInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformProviderVersionInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderVersionInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderVersionInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// terraformProviderVersionInfoFromTerraformProviderVersions returns a slice of terraformProviderVersionInfo, not necessarily sorted in any order.
func terraformProviderVersionInfoFromTerraformProviderVersions(terraformProviderVersions []models.TerraformProviderVersion) []terraformProviderVersionInfo {
	result := []terraformProviderVersionInfo{}

	for _, tp := range terraformProviderVersions {
		result = append(result, terraformProviderVersionInfo{
			id:         tp.Metadata.ID,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformProviderVersionIDsFromTerraformProviderVersionInfos preserves order
func terraformProviderVersionIDsFromTerraformProviderVersionInfos(terraformProviderVersionInfos []terraformProviderVersionInfo) []string {
	result := []string{}
	for _, terraformProviderVersionInfo := range terraformProviderVersionInfos {
		result = append(result, terraformProviderVersionInfo.id)
	}
	return result
}

// compareTerraformProviderVersions compares two terraform provider version objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformProviderVersions(t *testing.T, expected, actual *models.TerraformProviderVersion,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.ProviderID, actual.ProviderID)
	assert.Equal(t, expected.GPGASCIIArmor, actual.GPGASCIIArmor)
	assert.Equal(t, expected.GPGKeyID, actual.GPGKeyID)
	assert.Equal(t, expected.Protocols, actual.Protocols)
	assert.Equal(t, expected.SHASumsUploaded, actual.SHASumsUploaded)
	assert.Equal(t, expected.SHASumsSignatureUploaded, actual.SHASumsSignatureUploaded)
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

// The End.
