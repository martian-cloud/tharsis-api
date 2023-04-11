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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// userInfo aids convenience in accessing the information TestGetUsers
// needs about the warmup users.
type userInfo struct {
	updateTime time.Time
	userID     string
	name       string
	active     bool
}

// userInfoIDSlice makes a slice of userInfo sortable by ID string
type userInfoIDSlice []userInfo

// userInfoUpdateSlice makes a slice of userInfo sortable by last updated time
type userInfoUpdateSlice []userInfo

// userInfoNameSlice makes a slice of userInfo sortable by name
type userInfoNameSlice []userInfo

type linkExternalIDInput struct {
	issuer     string
	externalID string
	userID     string
}

type getExternalIDInput struct {
	issuer     string
	externalID string
}

// Because no other functions interact with external IDs, testing for
// GetUserByExternalID and LinkUserWithExternalID are combined.
func TestGetUserByLinkUserWithExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		linkInput     *linkExternalIDInput
		getInput      *getExternalIDInput
		expectLinkMsg *string
		expectGetMsg  *string
		expectUser    *models.User
		name          string
	}

	/*
		template test case:

		{
		name          string
		linkInput     *linkExternalIDInput
		getInput      *getExternalIDInput
		expectLinkMsg *string
		expectGetMsg  *string
		expectUser    *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one linkage per user.
	for _, user := range createdWarmupUsers {
		copyUser := user
		testCases = append(testCases, testCase{
			name: "positive--" + user.Username,
			linkInput: &linkExternalIDInput{
				issuer:     "issuer-" + user.Username,
				externalID: "external-ID-" + user.Username,
				userID:     user.Metadata.ID,
			},
			getInput: &getExternalIDInput{
				issuer:     "issuer-" + user.Username,
				externalID: "external-ID-" + user.Username,
			},
			expectUser: &copyUser,
		})
	}

	// Shortcut to the first positive test case.
	testCase0 := testCases[0]

	testCases = append(testCases,
		testCase{
			name:      "negative: link duplicate",
			linkInput: testCase0.linkInput,
			expectLinkMsg: ptr.String(fmt.Sprintf("user with external id %s already exists for issuer %s",
				testCase0.linkInput.externalID, testCase0.linkInput.issuer)),
		},
		testCase{
			name: "negative: link non-exist",
			linkInput: &linkExternalIDInput{
				issuer:     "issuer-for-negative-link-non-exist",
				externalID: "issuer-for-negative-link-non-exist",
				userID:     nonExistentID,
			},
			expectLinkMsg: ptr.String("ERROR: insert or update on table \"user_external_identities\" violates foreign key constraint \"fk_user_id\" (SQLSTATE 23503)"),
		},
		testCase{
			name: "negative: link invalid",
			linkInput: &linkExternalIDInput{
				issuer:     "issuer-for-negative-link-invalid",
				externalID: "issuer-for-negative-link-invalid",
				userID:     invalidID,
			},
			expectLinkMsg: invalidUUIDMsg1,
		},
		testCase{
			name: "negative: get issuer non-exist",
			getInput: &getExternalIDInput{
				issuer:     "issuer-does-not-exist",
				externalID: testCase0.linkInput.externalID,
			},
		},
		testCase{
			name: "negative: get external ID non-exist",
			getInput: &getExternalIDInput{
				issuer:     testCase0.linkInput.issuer,
				externalID: "external-ID-does-not-exist",
			},
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.linkInput != nil {

				err := testClient.client.Users.LinkUserWithExternalID(ctx,
					test.linkInput.issuer, test.linkInput.externalID, test.linkInput.userID)

				checkError(t, test.expectLinkMsg, err)

			}

			if test.getInput != nil {

				gotUser, err := testClient.client.Users.GetUserByExternalID(ctx,
					test.getInput.issuer, test.getInput.externalID)

				checkError(t, test.expectGetMsg, err)

				if test.expectUser != nil {
					require.NotNil(t, gotUser)
					compareUsers(t, test.expectUser, gotUser, true, nil)
				} else {
					assert.Nil(t, gotUser)
				}
			}
		})
	}
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		expectMsg  *string
		expectUser *models.User
		name       string
		input      string
	}

	/*
		template test case:

		{
		name       string
		input      string
		expectMsg  *string
		expectUser *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one linkage per user.
	for _, user := range createdWarmupUsers {
		copyUser := user
		testCases = append(testCases, testCase{
			name:       "positive--" + user.Username,
			input:      user.Metadata.ID,
			expectUser: &copyUser,
		})
	}

	testCases = append(testCases,
		testCase{
			name:  "negative: non-exist",
			input: nonExistentID,
		},
		testCase{
			name:      "negative: invalid",
			input:     invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotUser, err := testClient.client.Users.GetUserByID(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUser != nil {
				require.NotNil(t, gotUser)
				compareUsers(t, test.expectUser, gotUser, true, nil)
			} else {
				assert.Nil(t, gotUser)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		expectMsg  *string
		expectUser *models.User
		name       string
		input      string
	}

	/*
		template test case:

		{
		name       string
		input      string
		expectMsg  *string
		expectUser *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one linkage per user.
	for _, user := range createdWarmupUsers {
		copyUser := user
		testCases = append(testCases, testCase{
			name:       "positive--" + user.Email,
			input:      user.Email,
			expectUser: &copyUser,
		})
	}

	testCases = append(testCases,
		testCase{
			name:  "negative: non-exist",
			input: "nobody@nowhere.example.com",
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotUser, err := testClient.client.Users.GetUserByEmail(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUser != nil {
				require.NotNil(t, gotUser)
				compareUsers(t, test.expectUser, gotUser, true, nil)
			} else {
				assert.Nil(t, gotUser)
			}
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		expectMsg  *string
		expectUser *models.User
		name       string
		input      string
	}

	/*
		template test case:

		{
		name       string
		input      string
		expectMsg  *string
		expectUser *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one linkage per user.
	for _, user := range createdWarmupUsers {
		copyUser := user
		testCases = append(testCases, testCase{
			name:       "positive--" + user.Username,
			input:      user.Username,
			expectUser: &copyUser,
		})
	}

	testCases = append(testCases,
		testCase{
			name:  "negative: non-exist",
			input: "nobody",
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotUser, err := testClient.client.Users.GetUserByUsername(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUser != nil {
				require.NotNil(t, gotUser)
				compareUsers(t, test.expectUser, gotUser, true, nil)
			} else {
				assert.Nil(t, gotUser)
			}
		})
	}
}

func TestGetUsers(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)
	allUserInfos := userInfoFromUsers(warmupUsers)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(userInfoIDSlice(allUserInfos))
	allUserIDs := userIDsFromUserInfos(allUserInfos)

	// Sort by last update times.
	sort.Sort(userInfoUpdateSlice(allUserInfos))
	allUserIDsByUpdateTime := userIDsFromUserInfos(allUserInfos)
	reverseUserIDsByUpdateTime := reverseStringSlice(allUserIDsByUpdateTime)

	// Sort by names.
	sort.Sort(userInfoNameSlice(allUserInfos))
	allUserIDsByName := userIDsFromUserInfos(allUserInfos)

	allActiveUsers := activeUsersFromUserInfos(allUserInfos)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetUsersInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectUserIDs               []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetUsersInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectUserIDs               []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetUsers, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetUsersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectUserIDs:        allUserIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectUserIDs:        allUserIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtDesc),
			},
			expectUserIDs:        reverseUserIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectUserIDs:        allUserIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectUserIDs: allUserIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectUserIDs:              allUserIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectUserIDs:              allUserIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
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
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectUserIDs: reverseUserIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
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
			input: &GetUsersInput{
				Sort:              ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectUserIDs:               []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
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
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &UserFilter{
					UsernamePrefix: ptr.String(""),
					// Passing an empty slice to UserIDs likely
					// causes an SQL syntax error ("... IN ()"), so don't try it.
					// UserIDs: []string{},
				},
			},
			expectUserIDs: allUserIDsByUpdateTime,
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allUserIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user IDs, positive",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UserIDs: []string{
						allUserIDsByName[0], allUserIDsByName[1], allUserIDsByName[3],
					},
				},
			},
			expectUserIDs: []string{
				allUserIDsByName[0], allUserIDsByName[1], allUserIDsByName[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user IDs, non-existent",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UserIDs: []string{nonExistentID},
				},
			},
			expectUserIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user IDs, invalid ID",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UserIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectUserIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, username prefix, positive, u",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UsernamePrefix: ptr.String("u"),
				},
			},
			expectUserIDs:        allUserIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, username prefix, positive, user",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UsernamePrefix: ptr.String("user"),
				},
			},
			expectUserIDs:        allUserIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allUserIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, active, positive",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					Active: true,
				},
			},
			expectUserIDs:        allActiveUsers,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allActiveUsers)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, username prefix, negative, user-9",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UsernamePrefix: ptr.String("user-9"),
				},
			},
			expectUserIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, username prefix, negative, bogus",
			input: &GetUsersInput{
				Sort: ptrUserSortableField(UserSortableFieldUpdatedAtAsc),
				Filter: &UserFilter{
					UsernamePrefix: ptr.String("bogus"),
				},
			},
			expectUserIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// The username prefix is not required to be a UUID, so no check for UUID format can be done.

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

			usersActual, err := testClient.client.Users.GetUsers(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, usersActual.PageInfo)
				assert.NotNil(t, usersActual.Users)
				pageInfo := usersActual.PageInfo
				users := usersActual.Users

				// Check the users result by comparing a list of the user IDs.
				actualUserIDs := []string{}
				for _, user := range users {
					actualUserIDs = append(actualUserIDs, user.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualUserIDs)
				}

				assert.Equal(t, len(test.expectUserIDs), len(actualUserIDs))
				assert.Equal(t, test.expectUserIDs, actualUserIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one user returned.
				// If there are no users returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(users) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&users[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&users[len(users)-1])
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

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		input         *models.User
		expectMsg     *string
		expectUpdated *models.User
		name          string
	}

	/*
		template test case:

		{
		name          string
		input         *models.User
		expectMsg     *string
		expectUpdated *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup user at a time.
	// Only the username and email address fields can be updated.
	// Curiously, the admin field cannot be updated.
	for _, toUpdate := range createdWarmupUsers {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive: " + toUpdate.Username,
			input: &models.User{
				Metadata: models.ResourceMetadata{
					ID:      toUpdate.Metadata.ID,
					Version: toUpdate.Metadata.Version,
				},
				Username: "updated-" + toUpdate.Username,
				Email:    "updated-" + toUpdate.Email,
			},
			expectUpdated: &models.User{
				Metadata: models.ResourceMetadata{
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    toUpdate.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Username: "updated-" + toUpdate.Username,
				Email:    "updated-" + toUpdate.Email,
				Admin:    toUpdate.Admin,
			},
		})
	}

	// Negative cases:
	// Version number will have been incremented by the positive test cases.
	input0 := createdWarmupUsers[0]
	newVersion := input0.Metadata.Version + 1
	testCases = append(testCases,

		testCase{
			name: "negative: user ID does not exist",
			input: &models.User{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: newVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		testCase{
			name: "negative: invalid user ID",
			input: &models.User{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: newVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualUpdated, err := testClient.client.Users.UpdateUser(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				now := currentTime()

				compareUsers(t, test.expectUpdated, actualUpdated, false, &timeBounds{
					createLow:  test.expectUpdated.Metadata.CreationTimestamp,
					createHigh: test.expectUpdated.Metadata.CreationTimestamp,
					updateLow:  test.expectUpdated.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualUpdated)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Don't create any warmup users.

	type testCase struct {
		input       *models.User
		expectMsg   *string
		expectAdded *models.User
		name        string
	}

	/*
		template test case:

		{
		name        string
		input       *models.User
		expectMsg   *string
		expectAdded *models.User
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup user at a time.
	for _, toAdd := range standardWarmupUsers {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive: " + toAdd.Username,
			input: &models.User{
				Username: toAdd.Username,
				Email:    toAdd.Email,
				Admin:    toAdd.Admin,
			},
			expectAdded: &models.User{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &now,
					Version:           initialResourceVersion,
				},
				Username: toAdd.Username,
				Email:    toAdd.Email,
				Admin:    toAdd.Admin,
			},
		})
	}

	// Negative case:
	input0 := standardWarmupUsers[0]
	testCases = append(testCases,

		testCase{
			name: "negative: duplicate",
			input: &models.User{
				Username: input0.Username,
				Email:    input0.Email,
				Admin:    input0.Admin,
			},
			expectMsg: ptr.String(fmt.Sprintf("user with username %s already exists", input0.Username)),
		},

		// No does-not-exist or invalid test case is applicable.

	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			claimedAdded, err := testClient.client.Users.CreateUser(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectAdded != nil {
				require.NotNil(t, claimedAdded)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectAdded.Metadata.CreationTimestamp
				now := currentTime()

				compareUsers(t, test.expectAdded, claimedAdded, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})

				// Verify that what the CreateUser method claimed was added can fetched.
				fetched, err := testClient.client.Users.GetUserByEmail(ctx, test.input.Email)
				assert.Nil(t, err)

				if test.expectAdded != nil {
					require.NotNil(t, fetched)
					compareUsers(t, claimedAdded, fetched, true, nil)
				} else {
					assert.Nil(t, fetched)
				}
			} else {
				assert.Nil(t, claimedAdded)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupUsers, _, err := createInitialUsers(ctx, testClient, standardWarmupUsers)
	require.Nil(t, err)

	type testCase struct {
		toDelete  *models.User
		expectMsg *string
		name      string
	}

	testCases := []testCase{}
	for _, positiveUser := range createdWarmupUsers {
		testCases = append(testCases, testCase{
			name: "positive-" + positiveUser.Username,
			toDelete: &models.User{
				Metadata: models.ResourceMetadata{
					ID:      positiveUser.Metadata.ID,
					Version: positiveUser.Metadata.Version,
				},
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name: "negative, non-existent ID",
			toDelete: &models.User{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		testCase{
			name: "defective-id",
			toDelete: &models.User{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Users.DeleteUser(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsers = []models.User{
	{
		Username: "user-0",
		Email:    "user-0@example.com",
		Active:   true,
	},
	{
		Username: "user-1",
		Email:    "user-1@example.com",
		Active:   true,
	},
	{
		Username: "user-2",
		Email:    "user-2@example.com",
		Active:   true,
	},
	{
		Username: "user-3",
		Email:    "user-3@example.com",
		Active:   false,
	},
	{
		Username: "user-4",
		Email:    "user-4@example.com",
		Active:   false,
	},
}

func ptrUserSortableField(arg UserSortableField) *UserSortableField {
	return &arg
}

func (uis userInfoIDSlice) Len() int {
	return len(uis)
}

func (uis userInfoIDSlice) Swap(i, j int) {
	uis[i], uis[j] = uis[j], uis[i]
}

func (uis userInfoIDSlice) Less(i, j int) bool {
	return uis[i].userID < uis[j].userID
}

func (uus userInfoUpdateSlice) Len() int {
	return len(uus)
}

func (uus userInfoUpdateSlice) Swap(i, j int) {
	uus[i], uus[j] = uus[j], uus[i]
}

func (uus userInfoUpdateSlice) Less(i, j int) bool {
	return uus[i].updateTime.Before(uus[j].updateTime)
}

func (uns userInfoNameSlice) Len() int {
	return len(uns)
}

func (uns userInfoNameSlice) Swap(i, j int) {
	uns[i], uns[j] = uns[j], uns[i]
}

func (uns userInfoNameSlice) Less(i, j int) bool {
	return uns[i].name < uns[j].name
}

// userInfoFromUsers returns a slice of userInfo, not necessarily sorted in any order.
func userInfoFromUsers(users []models.User) []userInfo {
	result := []userInfo{}

	for _, user := range users {
		result = append(result, userInfo{
			updateTime: *user.Metadata.LastUpdatedTimestamp,
			userID:     user.Metadata.ID,
			name:       user.Username,
			active:     user.Active,
		})
	}

	return result
}

// userIDsFromUserInfos preserves order
func userIDsFromUserInfos(userInfos []userInfo) []string {
	result := []string{}
	for _, userInfo := range userInfos {
		result = append(result, userInfo.userID)
	}
	return result
}

// activeUsersFromUserInfos returns only active users.
func activeUsersFromUserInfos(userInfos []userInfo) []string {
	result := []string{}
	for _, userInfo := range userInfos {
		if userInfo.active {
			result = append(result, userInfo.userID)
		}
	}
	return result
}

// compareUsers compares two user objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareUsers(t *testing.T, expected, actual *models.User,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Username, actual.Username)
	assert.Equal(t, expected.Email, actual.Email)
	assert.Equal(t, expected.Admin, actual.Admin)

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
