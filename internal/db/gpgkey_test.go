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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// gpgKeyInfo aids convenience in accessing the information
// TestGetGPGKeys needs about the warmup GPG keys.
type gpgKeyInfo struct {
	updateTime time.Time
	id         string
}

// gpgKeyInfoIDSlice makes a slice of gpgKeyInfo sortable by ID string
type gpgKeyInfoIDSlice []gpgKeyInfo

// gpgKeyInfoUpdateTimeSlice makes a slice of gpgKeyInfo sortable by last updated time
type gpgKeyInfoUpdateTimeSlice []gpgKeyInfo

func TestGetGPGKeyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	_, warmupGPGKeys, err := createWarmupGPGKeys(ctx, testClient,
		standardWarmupGroupsForGPGKeys, standardWarmupGPGKeys)
	createdHigh := currentTime()
	require.Nil(t, err)

	type testCase struct {
		expectMsg    *string
		expectGPGKey *models.GPGKey
		name         string
		searchID     string
	}

	positiveGPGKey := warmupGPGKeys[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveGPGKey.Metadata.ID,
			expectGPGKey: &models.GPGKey{
				Metadata: models.ResourceMetadata{
					ID:                positiveGPGKey.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				GroupID:     positiveGPGKey.GroupID,
				CreatedBy:   positiveGPGKey.CreatedBy,
				ASCIIArmor:  positiveGPGKey.ASCIIArmor,
				Fingerprint: positiveGPGKey.Fingerprint,
				GPGKeyID:    positiveGPGKey.GPGKeyID,
			},
		},

		{
			name:     "negative, non-existent GPG key ID",
			searchID: nonExistentID,
			// expect GPG key and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualGPGKey, err := testClient.client.GPGKeys.GetGPGKeyByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectGPGKey != nil {
				require.NotNil(t, actualGPGKey)
				compareGPGKeys(t, test.expectGPGKey, actualGPGKey, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualGPGKey)
			}
		})
	}
}

func TestGetGPGKeys(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupGroups, warmupGPGKeys, err := createWarmupGPGKeys(ctx, testClient,
		standardWarmupGroupsForGPGKeys, standardWarmupGPGKeys)
	require.Nil(t, err)
	allGPGKeyInfos := gpgKeyInfoFromGPGKeys(warmupGPGKeys)

	// Sort by GPG key IDs.
	sort.Sort(gpgKeyInfoIDSlice(allGPGKeyInfos))
	allGPGKeyIDs := gpgKeyIDsFromGPGKeyInfos(allGPGKeyInfos)

	// Sort by update times.
	sort.Sort(gpgKeyInfoUpdateTimeSlice(allGPGKeyInfos))
	allGPGKeyIDsByUpdateTime := gpgKeyIDsFromGPGKeyInfos(allGPGKeyInfos)
	reverseGPGKeyIDsByUpdateTime := reverseStringSlice(allGPGKeyIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetGPGKeysInput
		expectMsg                   *string
		name                        string
		expectPageInfo              pagination.PageInfo
		expectGPGKeyIDs             []string
		getBeforeCursorFromPrevious bool
		sortedDescending            bool
		expectHasStartCursor        bool
		getAfterCursorFromPrevious  bool
		expectHasEndCursor          bool
		externalFilterHcl           bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetGPGKeysInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			externalFilterHcl            bool
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectGPGKeyIDs:             []string{},
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
		// nil input likely causes a nil pointer dereference in GetGPGKeys, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetGPGKeysInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectGPGKeyIDs:      allGPGKeyIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allGPGKeyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectGPGKeyIDs:      allGPGKeyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allGPGKeyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
			},
			expectGPGKeyIDs:      allGPGKeyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allGPGKeyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtDesc),
			},
			sortedDescending:     true,
			expectGPGKeyIDs:      reverseGPGKeyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allGPGKeyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectGPGKeyIDs:      allGPGKeyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allGPGKeyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectGPGKeyIDs: allGPGKeyIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allGPGKeyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGPGKeyIDs:            allGPGKeyIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allGPGKeyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGPGKeyIDs:            allGPGKeyIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allGPGKeyIDs)),
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
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending: true,
			expectGPGKeyIDs:  reverseGPGKeyIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allGPGKeyIDs)),
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
			input: &GetGPGKeysInput{
				Sort:              ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectGPGKeyIDs:             []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:       ptr.String("only first or last can be defined, not both"),
			expectGPGKeyIDs: allGPGKeyIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allGPGKeyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &GPGKeyFilter{
					GPGKeyID: ptr.Uint64(42),
					// Passing an empty slice to KeyIDs or NamespacePaths likely
					// causes an SQL syntax error ("... IN ()"), so don't try it.
					// KeyIDs: []string{},
					// NamespacePaths: []string{},
				},
			},
			expectGPGKeyIDs: []string{},
			expectPageInfo: pagination.PageInfo{
				TotalCount: 0,
				Cursor:     dummyCursorFunc,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, GPG key ID, positive",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					GPGKeyID: ptr.Uint64(warmupGPGKeys[0].GPGKeyID),
				},
			},
			expectGPGKeyIDs:      []string{allGPGKeyIDsByUpdateTime[0]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, GPG key ID, non-existent",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					GPGKeyID: ptr.Uint64(99),
				},
			},
			expectGPGKeyIDs:      []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, key IDs, positive",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					KeyIDs: []string{warmupGPGKeys[0].Metadata.ID, warmupGPGKeys[2].Metadata.ID, warmupGPGKeys[4].Metadata.ID},
				},
			},
			expectGPGKeyIDs: []string{
				allGPGKeyIDsByUpdateTime[0], allGPGKeyIDsByUpdateTime[2], allGPGKeyIDsByUpdateTime[4],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, key IDs, non-existent",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					KeyIDs: []string{nonExistentID},
				},
			},
			//			expectMsg:            ptr.String("Failed to scan query count result"),
			expectGPGKeyIDs:      []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, key IDs, invalid",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					KeyIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectGPGKeyIDs:      []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					NamespacePaths: []string{warmupGroups[0].FullPath},
				},
			},
			expectGPGKeyIDs:      allGPGKeyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, non-existent",
			input: &GetGPGKeysInput{
				Sort: ptrGPGKeySortableField(GPGKeySortableFieldUpdatedAtAsc),
				Filter: &GPGKeyFilter{
					NamespacePaths: []string{"this-path-does-not-exist"},
				},
			},
			expectGPGKeyIDs:      []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
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

			gpgKeysResult, err := testClient.client.GPGKeys.GetGPGKeys(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, gpgKeysResult.PageInfo)
				assert.NotNil(t, gpgKeysResult.GPGKeys)
				pageInfo := gpgKeysResult.PageInfo
				gpgKeys := gpgKeysResult.GPGKeys

				// Check the GPG keys result by comparing a list of the GPG key IDs.
				actualGPGKeyIDs := []string{}
				for _, gpgKey := range gpgKeys {
					actualGPGKeyIDs = append(actualGPGKeyIDs, gpgKey.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualGPGKeyIDs)
				}

				assert.Equal(t, len(test.expectGPGKeyIDs), len(actualGPGKeyIDs))
				assert.Equal(t, test.expectGPGKeyIDs, actualGPGKeyIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one GPG key returned.
				// If there are no GPG keys returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(gpgKeys) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&gpgKeys[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&gpgKeys[len(gpgKeys)-1])
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

func TestCreateGPGKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupGroups, _, err := createWarmupGPGKeys(ctx, testClient,
		standardWarmupGroupsForGPGKeys, []models.GPGKey{})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.GPGKey
		expectCreated *models.GPGKey
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.GPGKey{
				GroupID:     warmupGroups[0].Metadata.ID,
				CreatedBy:   "someone-k9",
				ASCIIArmor:  "armor-9",
				Fingerprint: "fingerprint-9",
				GPGKeyID:    111333555777999,
			},
			expectCreated: &models.GPGKey{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				GroupID:     warmupGroups[0].Metadata.ID,
				CreatedBy:   "someone-k9",
				ASCIIArmor:  "armor-9",
				Fingerprint: "fingerprint-9",
				GPGKeyID:    111333555777999,
			},
		},

		{
			name: "duplicate group ID and fingerprint",
			toCreate: &models.GPGKey{
				GroupID:     warmupGroups[0].Metadata.ID,
				CreatedBy:   "someone-k9",
				ASCIIArmor:  "armor-9",
				Fingerprint: "fingerprint-9",
				GPGKeyID:    111333555777999,
			},
			expectMsg: ptr.String("GPG key with key fingerprint fingerprint-9 already exists in group"),
		},

		{
			name: "negative, non-existent group ID",
			toCreate: &models.GPGKey{
				GroupID:     nonExistentID,
				CreatedBy:   "someone-k11",
				ASCIIArmor:  "armor-11",
				Fingerprint: "fingerprint-11",
				GPGKeyID:    222444666888000,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"gpg_keys\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid group ID",
			toCreate: &models.GPGKey{
				GroupID:     invalidID,
				CreatedBy:   "someone-k11",
				ASCIIArmor:  "armor-11",
				Fingerprint: "fingerprint-11",
				GPGKeyID:    222444666888000,
			},
			expectMsg: ptr.String("ERROR: invalid input syntax for type uuid: \"not-a-valid-uuid\" (SQLSTATE 22P02)"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.GPGKeys.CreateGPGKey(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareGPGKeys(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestDeleteGPGKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupGPGKeys, err := createWarmupGPGKeys(ctx, testClient,
		standardWarmupGroupsForGPGKeys, standardWarmupGPGKeys)
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.GPGKey
		name      string
	}

	// Looks up by ID and version.
	positiveGPGKey := warmupGPGKeys[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.GPGKey{
				Metadata: models.ResourceMetadata{
					ID:      positiveGPGKey.Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent GPG key ID",
			toDelete: &models.GPGKey{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.GPGKey{
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
			err := testClient.client.GPGKeys.DeleteGPGKey(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForGPGKeys = []models.Group{
	{
		Description: "top level group 0 for testing GPG key functions",
		FullPath:    "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup GPG keys for tests in this module:
// The create function will replace the group full path with the ID.
var standardWarmupGPGKeys = []models.GPGKey{
	{
		GroupID:     "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-k0",
		ASCIIArmor:  "armor-0",
		Fingerprint: "fingerprint-0",
		GPGKeyID:    111222333444555,
	},
	{
		GroupID:     "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-k1",
		ASCIIArmor:  "armor-1",
		Fingerprint: "fingerprint-1",
		GPGKeyID:    222333444555666,
	},
	{
		GroupID:     "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-k2",
		ASCIIArmor:  "armor-2",
		Fingerprint: "fingerprint-2",
		GPGKeyID:    333444555666777,
	},
	{
		GroupID:     "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-k3",
		ASCIIArmor:  "armor-3",
		Fingerprint: "fingerprint-3",
		GPGKeyID:    444555666777888,
	},
	{
		GroupID:     "top-level-group-0-for-gpg-keys",
		CreatedBy:   "someone-k4",
		ASCIIArmor:  "armor-4",
		Fingerprint: "fingerprint-5",
		GPGKeyID:    555666777888999,
	},
}

// createWarmupGPGKeys creates some warmup GPG keys for a test
// The warmup GPG keys to create can be standard or otherwise.
func createWarmupGPGKeys(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newGPGKeys []models.GPGKey) (
	[]models.Group,
	[]models.GPGKey,
	error,
) {
	// It is necessary to create at least one group
	// in order to provide the necessary IDs for the GPG key.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	resultGPGKeys, err := createInitialGPGKeys(ctx, testClient, newGPGKeys, parentPath2ID)
	if err != nil {
		return nil, nil, err
	}

	return resultGroups, resultGPGKeys, nil
}

func ptrGPGKeySortableField(arg GPGKeySortableField) *GPGKeySortableField {
	return &arg
}

func (wis gpgKeyInfoIDSlice) Len() int {
	return len(wis)
}

func (wis gpgKeyInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis gpgKeyInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis gpgKeyInfoUpdateTimeSlice) Len() int {
	return len(wis)
}

func (wis gpgKeyInfoUpdateTimeSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis gpgKeyInfoUpdateTimeSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// gpgKeyInfoFromGPGKeys returns a slice of gpgKeyInfo, not necessarily sorted in any order.
func gpgKeyInfoFromGPGKeys(gpgKeys []models.GPGKey) []gpgKeyInfo {
	result := []gpgKeyInfo{}

	for _, gpgKey := range gpgKeys {
		result = append(result, gpgKeyInfo{
			id:         gpgKey.Metadata.ID,
			updateTime: *gpgKey.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// gpgKeyIDsFromGPGKeyInfos preserves order
func gpgKeyIDsFromGPGKeyInfos(gpgKeyInfos []gpgKeyInfo) []string {
	result := []string{}
	for _, gpgKeyInfo := range gpgKeyInfos {
		result = append(result, gpgKeyInfo.id)
	}
	return result
}

// compareGPGKeys compares two GPG key objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareGPGKeys(t *testing.T, expected, actual *models.GPGKey,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.ASCIIArmor, actual.ASCIIArmor)
	assert.Equal(t, expected.Fingerprint, actual.Fingerprint)
	assert.Equal(t, expected.GPGKeyID, actual.GPGKeyID)

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
