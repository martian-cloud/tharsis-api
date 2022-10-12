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

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// serviceAccountInfo aids convenience in accessing the information TestGetServiceAccounts
// needs about the warmup service accounts.
type serviceAccountInfo struct {
	createTime       time.Time
	updateTime       time.Time
	serviceAccountID string
	name             string
}

// serviceAccountInfoIDSlice makes a slice of serviceAccountInfo sortable by ID string
type serviceAccountInfoIDSlice []serviceAccountInfo

// serviceAccountInfoCreateSlice makes a slice of serviceAccountInfo sortable by creation time
type serviceAccountInfoCreateSlice []serviceAccountInfo

// serviceAccountInfoUpdateSlice makes a slice of serviceAccountInfo sortable by last updated time
type serviceAccountInfoUpdateSlice []serviceAccountInfo

// serviceAccountInfoNameSlice makes a slice of serviceAccountInfo sortable by name
type serviceAccountInfoNameSlice []serviceAccountInfo

func TestGetServiceAccountByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	_, warmupServiceAccounts, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, standardWarmupServiceAccounts,
		standardWarmupOIDCTrustPoliciesForServiceAccounts)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := time.Now()

	type testCase struct {
		expectMsg            *string
		expectServiceAccount *models.ServiceAccount
		name                 string
		searchID             string
	}

	positiveServiceAccount := warmupServiceAccounts[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveServiceAccount.Metadata.ID,
			expectServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:                positiveServiceAccount.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ResourcePath:      positiveServiceAccount.ResourcePath,
				Name:              positiveServiceAccount.Name,
				Description:       positiveServiceAccount.Description,
				GroupID:           positiveServiceAccount.GroupID,
				CreatedBy:         positiveServiceAccount.CreatedBy,
				OIDCTrustPolicies: positiveServiceAccount.OIDCTrustPolicies,
			},
		},

		{
			name:     "negative, non-existent service account ID",
			searchID: nonExistentID,
			// expect service account and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualServiceAccount, err := testClient.client.ServiceAccounts.GetServiceAccountByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectServiceAccount != nil {
				require.NotNil(t, actualServiceAccount)
				compareServiceAccounts(t, test.expectServiceAccount, actualServiceAccount, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualServiceAccount)
			}
		})
	}
}

func TestGetServiceAccountByPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupGroups, warmupServiceAccounts, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, standardWarmupServiceAccounts,
		standardWarmupOIDCTrustPoliciesForServiceAccounts)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := time.Now()

	type testCase struct {
		expectMsg            *string
		expectServiceAccount *models.ServiceAccount
		name                 string
		searchPath           string
	}

	positiveServiceAccount := warmupServiceAccounts[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:       "positive",
			searchPath: warmupGroups[0].FullPath + "/" + positiveServiceAccount.Name,
			expectServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:                positiveServiceAccount.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ResourcePath:      positiveServiceAccount.ResourcePath,
				Name:              positiveServiceAccount.Name,
				Description:       positiveServiceAccount.Description,
				GroupID:           positiveServiceAccount.GroupID,
				CreatedBy:         positiveServiceAccount.CreatedBy,
				OIDCTrustPolicies: positiveServiceAccount.OIDCTrustPolicies,
			},
		},

		{
			name:       "negative, non-existent service account path",
			searchPath: "path-does-not-exist",
			// expect service account and error to be nil
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualServiceAccount, err := testClient.client.ServiceAccounts.GetServiceAccountByPath(ctx, test.searchPath)

			checkError(t, test.expectMsg, err)

			if test.expectServiceAccount != nil {
				require.NotNil(t, actualServiceAccount)
				compareServiceAccounts(t, test.expectServiceAccount, actualServiceAccount, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualServiceAccount)
			}
		})
	}
}

func TestCreateServiceAccount(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupGroups, _, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, []models.ServiceAccount{}, []models.OIDCTrustPolicy{})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	warmupGroup := warmupGroups[0]
	warmupGroupID := warmupGroup.Metadata.ID

	type testCase struct {
		toCreate      *models.ServiceAccount
		expectCreated *models.ServiceAccount
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{

		{
			name: "positive, nearly empty",
			toCreate: &models.ServiceAccount{
				Name:    "positive-create-service-account-nearly-empty",
				GroupID: warmupGroupID,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:              "positive-create-service-account-nearly-empty",
				GroupID:           warmupGroupID,
				ResourcePath:      warmupGroup.FullPath + "/positive-create-service-account-nearly-empty",
				OIDCTrustPolicies: []models.OIDCTrustPolicy{},
			},
		},

		{
			name: "positive full",
			toCreate: &models.ServiceAccount{
				Name:              "positive-create-service-account-full",
				Description:       "positive create service account",
				GroupID:           warmupGroupID,
				CreatedBy:         "creator-of-service-accounts",
				OIDCTrustPolicies: standardWarmupOIDCTrustPoliciesForServiceAccounts,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ResourcePath:      warmupGroup.FullPath + "/positive-create-service-account-full",
				Name:              "positive-create-service-account-full",
				Description:       "positive create service account",
				GroupID:           warmupGroupID,
				CreatedBy:         "creator-of-service-accounts",
				OIDCTrustPolicies: standardWarmupOIDCTrustPoliciesForServiceAccounts,
			},
		},

		{
			name: "duplicate name in same group",
			toCreate: &models.ServiceAccount{
				Name:    "positive-create-service-account-nearly-empty",
				GroupID: warmupGroupID,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectMsg: ptr.String(fmt.Sprintf("Service account with name %s already exists in group %s",
				"positive-create-service-account-nearly-empty", warmupGroupID)),
		},

		{
			name: "non-existent group ID",
			toCreate: &models.ServiceAccount{
				Name:    "non-existent-group-id",
				GroupID: nonExistentID,
			},
			expectMsg: ptr.String("invalid group: the specified group does not exist"),
		},

		{
			name: "defective group ID",
			toCreate: &models.ServiceAccount{
				Name:    "non-existent-group-id",
				GroupID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareServiceAccounts(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateServiceAccount(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupGroups, warmupServiceAccounts, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, standardWarmupServiceAccounts,
		standardWarmupOIDCTrustPoliciesForServiceAccounts)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := time.Now()

	type testCase struct {
		expectMsg            *string
		expectServiceAccount *models.ServiceAccount
		toUpdate             *models.ServiceAccount
		name                 string
	}

	positiveServiceAccount := warmupServiceAccounts[0]
	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:      positiveServiceAccount.Metadata.ID,
					Version: positiveServiceAccount.Metadata.Version,
				},
				Description: "updated description",
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:      "new-issuer",
						BoundClaims: map[string]string{"new-key": "new-value"},
					},
				},
			},
			// Only the description and trust policies get updated.
			expectServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:                   positiveServiceAccount.Metadata.ID,
					Version:              positiveServiceAccount.Metadata.Version + 1,
					CreationTimestamp:    positiveServiceAccount.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ResourcePath: warmupGroups[0].FullPath + "/" + positiveServiceAccount.Name,
				Name:         positiveServiceAccount.Name,
				Description:  "updated description",
				GroupID:      warmupGroups[0].Metadata.ID,
				CreatedBy:    positiveServiceAccount.CreatedBy,
				OIDCTrustPolicies: []models.OIDCTrustPolicy{
					{
						Issuer:      "new-issuer",
						BoundClaims: map[string]string{"new-key": "new-value"},
					},
				},
			},
		},

		{
			name: "negative, non-existent service account ID",
			toUpdate: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveServiceAccount.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toUpdate: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveServiceAccount.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualServiceAccount, err :=
				testClient.client.ServiceAccounts.UpdateServiceAccount(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := time.Now()
			if test.expectServiceAccount != nil {
				require.NotNil(t, actualServiceAccount)
				compareServiceAccounts(t, test.expectServiceAccount, actualServiceAccount, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualServiceAccount)
			}
		})
	}
}

func TestGetServiceAccounts(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupServiceAccounts, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, standardWarmupServiceAccounts,
		standardWarmupOIDCTrustPoliciesForServiceAccounts)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	allServiceAccountInfos := serviceAccountInfoFromServiceAccounts(warmupServiceAccounts)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(serviceAccountInfoIDSlice(allServiceAccountInfos))
	allServiceAccountIDs := serviceAccountIDsFromServiceAccountInfos(allServiceAccountInfos)

	// Sort by creation times.
	sort.Sort(serviceAccountInfoCreateSlice(allServiceAccountInfos))
	allServiceAccountIDsByCreateTime := serviceAccountIDsFromServiceAccountInfos(allServiceAccountInfos)
	reverseServiceAccountIDsByCreateTime := reverseStringSlice(allServiceAccountIDsByCreateTime)

	// Sort by last update times.
	sort.Sort(serviceAccountInfoUpdateSlice(allServiceAccountInfos))
	allServiceAccountIDsByUpdateTime := serviceAccountIDsFromServiceAccountInfos(allServiceAccountInfos)
	reverseServiceAccountIDsByUpdateTime := reverseStringSlice(allServiceAccountIDsByUpdateTime)

	// Sort by names.
	sort.Sort(serviceAccountInfoNameSlice(allServiceAccountInfos))
	allServiceAccountIDsByName := serviceAccountIDsFromServiceAccountInfos(allServiceAccountInfos)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetServiceAccountsInput
		name                        string
		expectPageInfo              PageInfo
		expectServiceAccountIDs     []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetServiceAccountsInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectServiceAccountIDs     []string
		expectPageInfo              PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{

		// nil input likely causes a nil pointer dereference in GetServiceAccounts, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetServiceAccountsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectServiceAccountIDs: allServiceAccountIDs,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectServiceAccountIDs: allServiceAccountIDsByCreateTime,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtDesc),
			},
			expectServiceAccountIDs: reverseServiceAccountIDsByCreateTime,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectServiceAccountIDs: allServiceAccountIDsByUpdateTime,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtDesc),
			},
			expectServiceAccountIDs: reverseServiceAccountIDsByUpdateTime,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "pagination: everything at once",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByUpdateTime,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "pagination: first two",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByUpdateTime[:2],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectServiceAccountIDs:    allServiceAccountIDsByUpdateTime[2:4],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectServiceAccountIDs:    allServiceAccountIDsByUpdateTime[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
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
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectServiceAccountIDs: reverseServiceAccountIDsByUpdateTime[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
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
			input: &GetServiceAccountsInput{
				Sort:              ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectServiceAccountIDs:     []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
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
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &ServiceAccountFilter{
					Search: ptr.String(""),
					// Passing an empty slice to ServiceAccountIDs or NamespacePaths likely
					// causes an SQL syntax error ("... IN ()"), so don't try it.
					// ServiceAccountIDs: []string{},
					// NamespacePaths: []string{},
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByCreateTime,
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allServiceAccountIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, empty string",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					Search: ptr.String(""),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByName,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, search field, 1",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					Search: ptr.String("1"),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByName[0:2],
			expectPageInfo:          PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, search field, 2",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					Search: ptr.String("2"),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByName[2:4],
			expectPageInfo:          PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, search field, 5",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					Search: ptr.String("5"),
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByName[4:],
			expectPageInfo:          PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectServiceAccountIDs: []string{},
			expectPageInfo:          PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, service account IDs, positive",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					ServiceAccountIDs: []string{
						allServiceAccountIDsByName[0], allServiceAccountIDsByName[1], allServiceAccountIDsByName[3]},
				},
			},
			expectServiceAccountIDs: []string{
				allServiceAccountIDsByName[0], allServiceAccountIDsByName[1], allServiceAccountIDsByName[3],
			},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account IDs, non-existent",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					ServiceAccountIDs: []string{nonExistentID},
				},
			},
			expectServiceAccountIDs: []string{},
			expectPageInfo:          PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, service account IDs, invalid ID",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					ServiceAccountIDs: []string{invalidID},
				},
			},
			expectMsg:               invalidUUIDMsg2,
			expectServiceAccountIDs: []string{},
			expectPageInfo:          PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					NamespacePaths: []string{"top-level-group-0-for-service-accounts"},
				},
			},
			expectServiceAccountIDs: allServiceAccountIDsByName,
			expectPageInfo:          PageInfo{TotalCount: int32(len(allServiceAccountIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
		},

		{
			name: "filter, namespace paths, negative",
			input: &GetServiceAccountsInput{
				Sort: ptrServiceAccountSortableField(ServiceAccountSortableFieldCreatedAtAsc),
				Filter: &ServiceAccountFilter{
					NamespacePaths: []string{"top-level-group-9-for-service-accounts"},
				},
			},
			expectServiceAccountIDs: []string{},
			expectPageInfo:          PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:    true,
			expectHasEndCursor:      true,
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

			serviceAccountsActual, err := testClient.client.ServiceAccounts.GetServiceAccounts(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, serviceAccountsActual.PageInfo)
				assert.NotNil(t, serviceAccountsActual.ServiceAccounts)
				pageInfo := serviceAccountsActual.PageInfo
				serviceAccounts := serviceAccountsActual.ServiceAccounts

				// Check the service accounts result by comparing a list of the service account IDs.
				actualServiceAccountIDs := []string{}
				for _, serviceAccount := range serviceAccounts {
					actualServiceAccountIDs = append(actualServiceAccountIDs, serviceAccount.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualServiceAccountIDs)
				}

				assert.Equal(t, len(test.expectServiceAccountIDs), len(actualServiceAccountIDs))
				assert.Equal(t, test.expectServiceAccountIDs, actualServiceAccountIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one service account returned.
				// If there are no service accounts returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(serviceAccounts) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&serviceAccounts[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&serviceAccounts[len(serviceAccounts)-1])
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

func TestDeleteServiceAccount(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupServiceAccounts, err := createWarmupServiceAccounts(ctx, testClient,
		standardWarmupGroupsForServiceAccounts, standardWarmupServiceAccounts,
		standardWarmupOIDCTrustPoliciesForServiceAccounts)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toDelete  *models.ServiceAccount
		expectMsg *string
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:      warmupServiceAccounts[0].Metadata.ID,
					Version: warmupServiceAccounts[0].Metadata.Version,
				},
			},
		},

		{
			name: "negative, non-existent ID",
			toDelete: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toDelete: &models.ServiceAccount{
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

			err := testClient.client.ServiceAccounts.DeleteServiceAccount(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)

		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForServiceAccounts = []models.Group{
	{
		Description: "top level group 0 for testing service account functions",
		FullPath:    "top-level-group-0-for-service-accounts",
		CreatedBy:   "someone-g0",
	},
}

// Standard service account(s) for tests in this module:
// The create function will convert the group name to group ID.
var standardWarmupServiceAccounts = []models.ServiceAccount{
	{
		ResourcePath:      "sa-resource-path-0",
		Name:              "1-service-account-0",
		Description:       "service account 0",
		GroupID:           "top-level-group-0-for-service-accounts", // will be fixed later
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-1",
		Name:              "1-service-account-1",
		Description:       "service account 1",
		GroupID:           "top-level-group-0-for-service-accounts", // will be fixed later
		CreatedBy:         "someone-sa1",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-2",
		Name:              "2-service-account-2",
		Description:       "service account 2",
		GroupID:           "top-level-group-0-for-service-accounts", // will be fixed later
		CreatedBy:         "someone-sa2",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-3",
		Name:              "2-service-account-3",
		Description:       "service account 3",
		GroupID:           "top-level-group-0-for-service-accounts", // will be fixed later
		CreatedBy:         "someone-sa3",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-4",
		Name:              "5-service-account-4",
		Description:       "service account 4",
		GroupID:           "top-level-group-0-for-service-accounts", // will be fixed later
		CreatedBy:         "someone-sa4",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

// Standard OIDC trust policy/policies for tests in this module:
var standardWarmupOIDCTrustPoliciesForServiceAccounts = []models.OIDCTrustPolicy{
	{
		Issuer:      "issuer-0",
		BoundClaims: map[string]string{"bc1-k1": "bc1-v1", "bc1-k2": "bc1-v2"},
	},
	{
		Issuer:      "issuer-1",
		BoundClaims: map[string]string{"bc2-k1": "bc2-v1", "bc2-k2": "bc2-v2"},
	},
}

// createWarmupServiceAccounts creates some warmup service accounts for a test
// The warmup service accounts to create can be standard or otherwise.
func createWarmupServiceAccounts(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newServiceAccounts []models.ServiceAccount,
	newTrustPolicies []models.OIDCTrustPolicy) (
	[]models.Group, []models.ServiceAccount, error) {

	// It is necessary to create at least one group
	// in order to provide the necessary IDs for the service accounts.

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	// It is necessary to add the OIDC trust policies to the service account before
	// creating the service account in the DB.
	for _, sa := range newServiceAccounts {
		for _, tp := range newTrustPolicies {
			sa.OIDCTrustPolicies = append(sa.OIDCTrustPolicies, models.OIDCTrustPolicy{
				Issuer:      tp.Issuer + "-for-" + sa.Name,
				BoundClaims: tp.BoundClaims,
			})
		}
	}

	resultServiceAccounts, _, err := createInitialServiceAccounts(ctx, testClient,
		groupPath2ID, newServiceAccounts)
	if err != nil {
		return nil, nil, err
	}

	return resultGroups, resultServiceAccounts, nil
}

func ptrServiceAccountSortableField(arg ServiceAccountSortableField) *ServiceAccountSortableField {
	return &arg
}

func (sais serviceAccountInfoIDSlice) Len() int {
	return len(sais)
}

func (sais serviceAccountInfoIDSlice) Swap(i, j int) {
	sais[i], sais[j] = sais[j], sais[i]
}

func (sais serviceAccountInfoIDSlice) Less(i, j int) bool {
	return sais[i].serviceAccountID < sais[j].serviceAccountID
}

func (sais serviceAccountInfoCreateSlice) Len() int {
	return len(sais)
}

func (sais serviceAccountInfoCreateSlice) Swap(i, j int) {
	sais[i], sais[j] = sais[j], sais[i]
}

func (sais serviceAccountInfoCreateSlice) Less(i, j int) bool {
	return sais[i].createTime.Before(sais[j].createTime)
}

func (sais serviceAccountInfoUpdateSlice) Len() int {
	return len(sais)
}

func (sais serviceAccountInfoUpdateSlice) Swap(i, j int) {
	sais[i], sais[j] = sais[j], sais[i]
}

func (sais serviceAccountInfoUpdateSlice) Less(i, j int) bool {
	return sais[i].updateTime.Before(sais[j].updateTime)
}

func (sais serviceAccountInfoNameSlice) Len() int {
	return len(sais)
}

func (sais serviceAccountInfoNameSlice) Swap(i, j int) {
	sais[i], sais[j] = sais[j], sais[i]
}

func (sais serviceAccountInfoNameSlice) Less(i, j int) bool {
	return sais[i].name < sais[j].name
}

// serviceAccountInfoFromServiceAccounts returns a slice of serviceAccountInfo, not necessarily sorted in any order.
func serviceAccountInfoFromServiceAccounts(serviceAccounts []models.ServiceAccount) []serviceAccountInfo {
	result := []serviceAccountInfo{}

	for _, serviceAccount := range serviceAccounts {
		result = append(result, serviceAccountInfo{
			createTime:       *serviceAccount.Metadata.CreationTimestamp,
			updateTime:       *serviceAccount.Metadata.LastUpdatedTimestamp,
			serviceAccountID: serviceAccount.Metadata.ID,
			name:             serviceAccount.Name,
		})
	}

	return result
}

// serviceAccountIDsFromServiceAccountInfos preserves order
func serviceAccountIDsFromServiceAccountInfos(serviceAccountInfos []serviceAccountInfo) []string {
	result := []string{}
	for _, serviceAccountInfo := range serviceAccountInfos {
		result = append(result, serviceAccountInfo.serviceAccountID)
	}
	return result
}

// compareServiceAccounts compares two service account objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareServiceAccounts(t *testing.T, expected, actual *models.ServiceAccount,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.ResourcePath, actual.ResourcePath)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	compareOIDCTrustPolicies(t, expected.OIDCTrustPolicies, actual.OIDCTrustPolicies)

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

// compareOIDCTrustPolicies compares two slices of OIDC trust policies,
func compareOIDCTrustPolicies(t *testing.T, expected, actual []models.OIDCTrustPolicy) {
	require.Equal(t, len(expected), len(actual))

	for ix := range expected {
		assert.Equal(t, expected[ix].Issuer, actual[ix].Issuer)
		assert.Equal(t, expected[ix].BoundClaims, actual[ix].BoundClaims)
	}
}

// The End.
