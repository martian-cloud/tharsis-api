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

// roleInfo aids convenience in accessing the information TestGetRoles
// needs about the warmup roles.
type roleInfo struct {
	createTime time.Time
	updateTime time.Time
	roleID     string
	name       string
}

// roleInfoIDSlice makes a slice of roleInfo sortable by ID string
type roleInfoIDSlice []roleInfo

// roleInfoUpdateSlice makes a slice of roleInfo sortable by last updated time
type roleInfoUpdateSlice []roleInfo

// roleInfoNameSlice makes a slice of roleInfo sortable by name
type roleInfoNameSlice []roleInfo

// warmupRoles holds the outputs from createWarmupRoles.
type warmupRoles struct {
	roles []models.Role
}

func TestGetRoleByName(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupRoles(ctx, testClient, standardWarmupRoles)
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectMsg  *string
		expectRole *models.Role
		name       string
		search     string
	}

	positiveRole := warmupItems.roles[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:   "positive",
			search: positiveRole.Name,
			expectRole: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:                positiveRole.Metadata.ID,
					Version:           positiveRole.Metadata.Version,
					CreationTimestamp: &now,
				},
				Name:        positiveRole.Name,
				Description: positiveRole.Description,
				CreatedBy:   positiveRole.CreatedBy,
			},
		},

		{
			name:   "negative, non-existent role name",
			search: "name-does-not-exist",
			// expect role and error to be nil
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualRole, err := testClient.client.Roles.GetRoleByName(ctx, test.search)

			checkError(t, test.expectMsg, err)

			if test.expectRole != nil {
				require.NotNil(t, actualRole)
				compareRoles(t, test.expectRole, actualRole, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualRole)
			}
		})
	}
}

func TestGetRoleByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupRoles(ctx, testClient, standardWarmupRoles)
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectRole *models.Role
		expectMsg  *string
		name       string
		searchID   string
	}

	positiveRole := warmupItems.roles[0]
	testCases := []testCase{
		{
			name:       "positive",
			searchID:   positiveRole.Metadata.ID,
			expectRole: &positiveRole,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect role and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualRole, err := testClient.client.Roles.GetRoleByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectRole != nil {
				require.NotNil(t, actualRole)
				compareRoles(t, test.expectRole, actualRole, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualRole)
			}

		})
	}
}

func TestGetRoles(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRoles(ctx, testClient, standardWarmupRoles)
	require.Nil(t, err)

	allRoleInfos := roleInfoFromRoles(warmupItems.roles)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(roleInfoIDSlice(allRoleInfos))
	allRoleIDs := roleIDsFromRoleInfos(allRoleInfos)

	// Sort by last update times.
	sort.Sort(roleInfoUpdateSlice(allRoleInfos))
	allRoleIDsByUpdateTime := roleIDsFromRoleInfos(allRoleInfos)
	reverseRoleIDsByUpdateTime := reverseStringSlice(allRoleIDsByUpdateTime)

	// Sort by names.
	sort.Sort(roleInfoNameSlice(allRoleInfos))
	allRoleIDsByName := roleIDsFromRoleInfos(allRoleInfos)
	reverseRoleIDsByName := reverseStringSlice(allRoleIDsByName)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetRolesInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectRoleIDs               []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetRolesInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectRoleIDs               []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{

		// nil input likely causes a nil pointer dereference in GetRoles, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetRolesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectRoleIDs:        allRoleIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of name, nil filter",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectRoleIDs:        allRoleIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of name",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameDesc),
			},
			expectRoleIDs:        reverseRoleIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectRoleIDs:        allRoleIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtDesc),
			},
			expectRoleIDs:        reverseRoleIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectRoleIDs:        allRoleIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectRoleIDs: allRoleIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectRoleIDs:              allRoleIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectRoleIDs:              allRoleIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
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
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectRoleIDs: reverseRoleIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
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
			input: &GetRolesInput{
				Sort:              ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectRoleIDs:               []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
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
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &RoleFilter{
					Search: ptr.String(""),
				},
			},
			expectRoleIDs: allRoleIDsByName,
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allRoleIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, empty string",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String(""),
				},
			},
			expectRoleIDs:        allRoleIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allRoleIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 1",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String("1"),
				},
			},
			expectRoleIDs:        allRoleIDsByName[1:2],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 2",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String("2"),
				},
			},
			expectRoleIDs:        allRoleIDsByName[2:3],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 3",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String("3"),
				},
			},
			expectRoleIDs:        allRoleIDsByName[3:4],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 4",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String("4"),
				},
			},
			expectRoleIDs:        allRoleIDsByName[4:],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectRoleIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, role IDs, positive",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					RoleIDs: []string{
						allRoleIDsByName[0], allRoleIDsByName[1], allRoleIDsByName[3]},
				},
			},
			expectRoleIDs: []string{
				allRoleIDsByName[0], allRoleIDsByName[1], allRoleIDsByName[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, role IDs, non-existent",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					RoleIDs: []string{nonExistentID},
				},
			},
			expectRoleIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, role IDs, invalid ID",
			input: &GetRolesInput{
				Sort: ptrRoleSortableField(RoleSortableFieldNameAsc),
				Filter: &RoleFilter{
					RoleIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectRoleIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
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

			rolesActual, err := testClient.client.Roles.GetRoles(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, rolesActual.PageInfo)
				assert.NotNil(t, rolesActual.Roles)
				pageInfo := rolesActual.PageInfo
				roles := rolesActual.Roles

				// Check the roles result by comparing a list of the role IDs.
				actualRoleIDs := []string{}
				for _, role := range roles {
					actualRoleIDs = append(actualRoleIDs, role.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualRoleIDs)
				}

				assert.Equal(t, len(test.expectRoleIDs), len(actualRoleIDs))
				assert.Equal(t, test.expectRoleIDs, actualRoleIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one role returned.
				// If there are no roles returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(roles) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&roles[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&roles[len(roles)-1])
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

func TestCreateRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		toCreate      *models.Role
		expectCreated *models.Role
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{

		{
			name: "positive, nearly empty",
			toCreate: &models.Role{
				Name: "positive-create-role-nearly-empty",
			},
			expectCreated: &models.Role{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name: "positive-create-role-nearly-empty",
			},
		},

		{
			name: "positive full",
			toCreate: &models.Role{
				Name:        "positive-create-role-full",
				Description: "positive create role",
				CreatedBy:   "creator-of-roles",
			},
			expectCreated: &models.Role{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:        "positive-create-role-full",
				Description: "positive create role",
				CreatedBy:   "creator-of-roles",
			},
		},

		{
			name: "creating a duplicate role",
			toCreate: &models.Role{
				Name:        "positive-create-role-full",
				Description: "positive create role",
				CreatedBy:   "creator-of-roles",
			},
			expectMsg: ptr.String("role with name positive-create-role-full already exists"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// Set role permissions.
			perms, _ := models.ViewerRoleID.Permissions()
			test.toCreate.SetPermissions(perms)

			if test.expectCreated != nil {
				test.expectCreated.SetPermissions(perms)
			}

			actualCreated, err := testClient.client.Roles.CreateRole(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareRoles(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupRoles(ctx, testClient, standardWarmupRoles)
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		toUpdate   *models.Role
		expectRole *models.Role
		expectMsg  *string
		name       string
	}

	// Do only one positive test case, because the logic is theoretically the same for all roles.
	now := currentTime()
	positiveRole := warmupItems.roles[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:      positiveRole.Metadata.ID,
					Version: positiveRole.Metadata.Version,
				},
				Description: "updated description",
			},
			expectRole: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRole.Metadata.ID,
					Version:              positiveRole.Metadata.Version + 1,
					CreationTimestamp:    positiveRole.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Name:        positiveRole.Name,
				Description: "updated description",
				CreatedBy:   positiveRole.CreatedBy,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveRole.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveRole.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			perms, _ := models.ViewerRoleID.Permissions()
			test.toUpdate.SetPermissions(perms)

			if test.expectRole != nil {
				test.expectRole.SetPermissions(perms)
			}

			actualRole, err := testClient.client.Roles.UpdateRole(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := currentTime()
			if test.expectRole != nil {
				require.NotNil(t, actualRole)
				compareRoles(t, test.expectRole, actualRole, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualRole)
			}
		})
	}
}

func TestDeleteRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRoles(ctx, testClient, standardWarmupRoles)
	require.Nil(t, err)

	type testCase struct {
		toDelete  *models.Role
		expectMsg *string
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.Role{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.roles[0].Metadata.ID,
					Version: warmupItems.roles[0].Metadata.Version,
				},
			},
		},

		{
			name: "negative, non-existent ID",
			toDelete: &models.Role{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toDelete: &models.Role{
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
			err := testClient.client.Roles.DeleteRole(ctx, test.toDelete)
			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

var standardWarmupRoles = []models.Role{
	{
		Name:        "0-role-0",
		Description: "role 0 for testing roles",
		CreatedBy:   "someone-r0",
	},
	{
		Name:        "1-role-1",
		Description: "role 1 for testing roles",
		CreatedBy:   "someone-r1",
	},
	{
		Name:        "2-role-2",
		Description: "role 2 for testing roles",
		CreatedBy:   "someone-r3",
	},
	{
		Name:        "3-role-3",
		Description: "role 3 for testing roles",
		CreatedBy:   "someone-r4",
	},
	{
		Name:        "4-role-4",
		Description: "role 4 for testing roles",
		CreatedBy:   "someone-r4",
	},
}

func createWarmupRoles(ctx context.Context, testClient *testClient, input []models.Role) (*warmupRoles, error) {

	resultRoles, _, err := createInitialRoles(ctx, testClient, input)
	if err != nil {
		return nil, err
	}

	return &warmupRoles{
		roles: resultRoles,
	}, nil
}

// createInitialRoles creates some warmup roles for a test.
func createInitialRoles(ctx context.Context, testClient *testClient, toCreate []models.Role) (
	[]models.Role, map[string]string, error) {
	result := []models.Role{}
	roleName2ID := make(map[string]string)

	for _, input := range toCreate {
		created, err := testClient.client.Roles.CreateRole(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		roleName2ID[created.Name] = created.Metadata.ID
	}

	return result, roleName2ID, nil
}

func ptrRoleSortableField(arg RoleSortableField) *RoleSortableField {
	return &arg
}

func (ris roleInfoIDSlice) Len() int {
	return len(ris)
}

func (ris roleInfoIDSlice) Swap(i, j int) {
	ris[i], ris[j] = ris[j], ris[i]
}

func (ris roleInfoIDSlice) Less(i, j int) bool {
	return ris[i].roleID < ris[j].roleID
}

func (ris roleInfoUpdateSlice) Len() int {
	return len(ris)
}

func (ris roleInfoUpdateSlice) Swap(i, j int) {
	ris[i], ris[j] = ris[j], ris[i]
}

func (ris roleInfoUpdateSlice) Less(i, j int) bool {
	return ris[i].updateTime.Before(ris[j].updateTime)
}

func (ris roleInfoNameSlice) Len() int {
	return len(ris)
}

func (ris roleInfoNameSlice) Swap(i, j int) {
	ris[i], ris[j] = ris[j], ris[i]
}

func (ris roleInfoNameSlice) Less(i, j int) bool {
	return ris[i].name < ris[j].name
}

// roleInfoFromRoles returns a slice of roleInfo, not necessarily sorted in any order.
func roleInfoFromRoles(roles []models.Role) []roleInfo {
	result := []roleInfo{}

	for _, role := range roles {
		result = append(result, roleInfo{
			createTime: *role.Metadata.CreationTimestamp,
			updateTime: *role.Metadata.LastUpdatedTimestamp,
			roleID:     role.Metadata.ID,
			name:       role.Name,
		})
	}

	return result
}

// roleIDsFromRoleInfos preserves order
func roleIDsFromRoleInfos(roleInfos []roleInfo) []string {
	result := []string{}
	for _, roleInfo := range roleInfos {
		result = append(result, roleInfo.roleID)
	}
	return result
}

// compareRoles compares two role objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareRoles(t *testing.T, expected, actual *models.Role,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Len(t, actual.GetPermissions(), len(expected.GetPermissions()))

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
