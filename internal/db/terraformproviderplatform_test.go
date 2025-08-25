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

// terraformProviderPlatformInfo aids convenience in accessing the information
// TestGetTerraformProviderPlatforms needs about the warmup objects.
type terraformProviderPlatformInfo struct {
	updateTime time.Time
	id         string
}

// terraformProviderPlatformInfoIDSlice makes a slice of terraformProviderPlatformInfo sortable by ID string
type terraformProviderPlatformInfoIDSlice []terraformProviderPlatformInfo

// terraformProviderPlatformInfoUpdateSlice makes a slice of terraformProviderPlatformInfo sortable by last updated time
type terraformProviderPlatformInfoUpdateSlice []terraformProviderPlatformInfo

// warmupTerraformProviderPlatforms holds the inputs to and outputs from createWarmupTerraformProviderPlatforms.
type warmupTerraformProviderPlatforms struct {
	groups                     []models.Group
	workspaces                 []models.Workspace
	teams                      []models.Team
	users                      []models.User
	teamMembers                []models.TeamMember
	serviceAccounts            []models.ServiceAccount
	namespaceMembershipsIn     []CreateNamespaceMembershipInput
	namespaceMembershipsOut    []models.NamespaceMembership
	terraformProviders         []models.TerraformProvider
	terraformProviderVersions  []models.TerraformProviderVersion
	terraformProviderPlatforms []models.TerraformProviderPlatform
}

func TestGetProviderPlatformByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupItems, err := createWarmupTerraformProviderPlatforms(ctx, testClient, warmupTerraformProviderPlatforms{
		groups:                     standardWarmupGroupsForTerraformProviderPlatforms,
		terraformProviders:         standardWarmupTerraformProvidersForTerraformProviderPlatforms,
		terraformProviderVersions:  standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms,
		terraformProviderPlatforms: standardWarmupTerraformProviderPlatforms,
	})
	require.Nil(t, err)
	createdHigh := time.Now()

	type testCase struct {
		expectMsg                       *string
		expectTerraformProviderPlatform *models.TerraformProviderPlatform
		name                            string
		searchID                        string
	}

	positiveTerraformProviderPlatform := warmupItems.terraformProviderPlatforms[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveTerraformProviderPlatform.Metadata.ID,
			expectTerraformProviderPlatform: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:                positiveTerraformProviderPlatform.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},

				ProviderVersionID: positiveTerraformProviderPlatform.ProviderVersionID,
				OperatingSystem:   positiveTerraformProviderPlatform.OperatingSystem,
				Architecture:      positiveTerraformProviderPlatform.Architecture,
				SHASum:            positiveTerraformProviderPlatform.SHASum,
				Filename:          positiveTerraformProviderPlatform.Filename,
				CreatedBy:         positiveTerraformProviderPlatform.CreatedBy,
				BinaryUploaded:    positiveTerraformProviderPlatform.BinaryUploaded,
			},
		},

		{
			name:     "negative, non-existent Terraform provider platform ID",
			searchID: nonExistentID,
			// expect terraform provider platform and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTerraformProviderPlatform, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatformByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformProviderPlatform != nil {
				require.NotNil(t, actualTerraformProviderPlatform)
				compareTerraformProviderPlatforms(t, test.expectTerraformProviderPlatform, actualTerraformProviderPlatform, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualTerraformProviderPlatform)
			}
		})
	}
}

func TestGetProviderPlatformByTRN(t *testing.T) {
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
	})
	require.NoError(t, err)

	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		ProviderID:      provider.Metadata.ID,
		SemanticVersion: "1.0.0",
	})
	require.NoError(t, err)

	platform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		ProviderVersionID: providerVersion.Metadata.ID,
		OperatingSystem:   "linux",
		Architecture:      "amd64",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectPlatform  bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:           "get provider platform by TRN",
			trn:            platform.Metadata.TRN,
			expectPlatform: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TerraformProviderPlatformModelType.BuildTRN(group.FullPath, provider.Name, providerVersion.SemanticVersion, "windows", "arm"),
		},
		{
			name:            "provider platform TRN has less than 5 parts",
			trn:             types.TerraformProviderPlatformModelType.BuildTRN("test-group"),
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
			actualPlatform, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatformByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectPlatform {
				require.NotNil(t, actualPlatform)
				assert.Equal(t,
					types.TerraformProviderPlatformModelType.BuildTRN(
						group.FullPath,
						provider.Name,
						providerVersion.SemanticVersion,
						platform.OperatingSystem,
						platform.Architecture,
					),
					actualPlatform.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualPlatform)
			}
		})
	}
}

func TestGetProviderPlatforms(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderPlatforms(ctx, testClient, warmupTerraformProviderPlatforms{
		groups:                     standardWarmupGroupsForTerraformProviderPlatforms,
		terraformProviders:         standardWarmupTerraformProvidersForTerraformProviderPlatforms,
		terraformProviderVersions:  standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms,
		terraformProviderPlatforms: standardWarmupTerraformProviderPlatforms,
	})
	require.Nil(t, err)
	allTerraformProviderPlatformInfos := terraformProviderPlatformInfoFromTerraformProviderPlatforms(
		warmupItems.terraformProviderPlatforms)

	// Sort by Terraform provider platform IDs.
	sort.Sort(terraformProviderPlatformInfoIDSlice(allTerraformProviderPlatformInfos))
	allTerraformProviderPlatformIDs := terraformProviderPlatformIDsFromTerraformProviderPlatformInfos(allTerraformProviderPlatformInfos)

	// Sort by last update times.
	sort.Sort(terraformProviderPlatformInfoUpdateSlice(allTerraformProviderPlatformInfos))
	allTerraformProviderPlatformIDsByTime := terraformProviderPlatformIDsFromTerraformProviderPlatformInfos(allTerraformProviderPlatformInfos)
	reverseTerraformProviderPlatformIDsByTime := reverseStringSlice(allTerraformProviderPlatformIDsByTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError             error
		expectEndCursorError               error
		input                              *GetProviderPlatformsInput
		expectMsg                          *string
		name                               string
		expectPageInfo                     pagination.PageInfo
		expectTerraformProviderPlatformIDs []string
		getBeforeCursorFromPrevious        bool
		sortedDescending                   bool
		expectHasStartCursor               bool
		getAfterCursorFromPrevious         bool
		expectHasEndCursor                 bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetProviderPlatformsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectTerraformProviderPlatformIDs:  []string{},
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
		// nil input likely causes a nil pointer dereference in GetProviderPlatforms, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetProviderPlatformsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDs,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtDesc),
			},
			sortedDescending:                   true,
			expectTerraformProviderPlatformIDs: reverseTerraformProviderPlatformIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "pagination: everything at once",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "pagination: first two",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:         true,
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:         true,
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformIDs)),
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
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:                   true,
			expectTerraformProviderPlatformIDs: reverseTerraformProviderPlatformIDsByTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformIDs)),
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
			input: &GetProviderPlatformsInput{
				Sort:              ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:         true,
			getBeforeCursorFromPrevious:        true,
			expectMsg:                          ptr.String("only before or after can be defined, not both"),
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                          ptr.String("only first or last can be defined, not both"),
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &TerraformProviderPlatformFilter{
					ProviderID:        ptr.String(""),
					ProviderVersionID: ptr.String(""),
					BinaryUploaded:    ptr.Bool(false),
					OperatingSystem:   ptr.String(""),
					Architecture:      ptr.String(""),
				},
			},
			expectMsg:                          invalidUUIDMsg,
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "filter, provider ID, positive",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderID: ptr.String(warmupItems.terraformProviderVersions[0].ProviderID),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[:3],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, provider ID, non-existent",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider ID, invalid",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderID: ptr.String(invalidID),
				},
			},
			expectMsg:                          invalidUUIDMsg,
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "filter, provider version ID, positive",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderVersionID: ptr.String(warmupItems.terraformProviderVersions[0].Metadata.ID),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[:2],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, provider version ID, non-existent",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderVersionID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider version ID, invalid",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					ProviderVersionID: ptr.String(invalidID),
				},
			},
			expectMsg:                          invalidUUIDMsg,
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "filter, binary uploaded, true",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					BinaryUploaded: ptr.Bool(true),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[2:],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, binary uploaded, false",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					BinaryUploaded: ptr.Bool(false),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[:2],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, operating system, positive",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					OperatingSystem: ptr.String(warmupItems.terraformProviderPlatforms[0].OperatingSystem),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformIDsByTime[:2],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, operating system, non-existent",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					OperatingSystem: ptr.String("this-operating-system-does-not-exist"),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, architecture, positive",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					Architecture: ptr.String(warmupItems.terraformProviderPlatforms[0].Architecture),
				},
			},
			expectTerraformProviderPlatformIDs: []string{
				allTerraformProviderPlatformIDsByTime[0],
				allTerraformProviderPlatformIDsByTime[2],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, architecture, non-existent",
			input: &GetProviderPlatformsInput{
				Sort: ptrTerraformProviderPlatformSortableField(TerraformProviderPlatformSortableFieldUpdatedAtAsc),
				Filter: &TerraformProviderPlatformFilter{
					Architecture: ptr.String("this-architecture-does-not-exist"),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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

			terraformProviderPlatformsResult, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatforms(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, terraformProviderPlatformsResult.PageInfo)
				assert.NotNil(t, terraformProviderPlatformsResult.ProviderPlatforms)
				pageInfo := terraformProviderPlatformsResult.PageInfo
				terraformProviderPlatforms := terraformProviderPlatformsResult.ProviderPlatforms

				// Check the terraform provider platforms result by comparing a list of the terraform provider platform IDs.
				actualTerraformProviderPlatformIDs := []string{}
				for _, terraformProviderPlatform := range terraformProviderPlatforms {
					actualTerraformProviderPlatformIDs = append(actualTerraformProviderPlatformIDs, terraformProviderPlatform.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformProviderPlatformIDs)
				}

				assert.Equal(t, len(test.expectTerraformProviderPlatformIDs), len(actualTerraformProviderPlatformIDs))
				assert.Equal(t, test.expectTerraformProviderPlatformIDs, actualTerraformProviderPlatformIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one terraform provider platform returned.
				// If there are no terraform provider platforms returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(terraformProviderPlatforms) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&terraformProviderPlatforms[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&terraformProviderPlatforms[len(terraformProviderPlatforms)-1])
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

func TestCreateProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderPlatforms(ctx, testClient, warmupTerraformProviderPlatforms{
		groups:                    standardWarmupGroupsForTerraformProviderPlatforms,
		terraformProviders:        standardWarmupTerraformProvidersForTerraformProviderPlatforms,
		terraformProviderVersions: standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformProviderPlatform
		expectCreated *models.TerraformProviderPlatform
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformProviderPlatform{
				ProviderVersionID: warmupItems.terraformProviderVersions[0].Metadata.ID,
				OperatingSystem:   "os-y",
				Architecture:      "arch-y",
				SHASum:            "sha-sum-y",
				Filename:          "filename-y",
				CreatedBy:         "TestCreateProviderPlatform",
				BinaryUploaded:    true,
			},
			expectCreated: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ProviderVersionID: warmupItems.terraformProviderVersions[0].Metadata.ID,
				OperatingSystem:   "os-y",
				Architecture:      "arch-y",
				SHASum:            "sha-sum-y",
				Filename:          "filename-y",
				CreatedBy:         "TestCreateProviderPlatform",
				BinaryUploaded:    true,
			},
		},

		{
			name: "duplicate provider, operating system, and architecture",
			toCreate: &models.TerraformProviderPlatform{
				ProviderVersionID: warmupItems.terraformProviderVersions[0].Metadata.ID,
				OperatingSystem:   "os-y",
				Architecture:      "arch-y",
			},
			expectMsg: ptr.String("terraform provider platform os-y_arch-y already exists"),
		},

		{
			name: "negative, non-existent provider version ID",
			toCreate: &models.TerraformProviderPlatform{
				ProviderVersionID: nonExistentID,
				OperatingSystem:   "os-z1",
				Architecture:      "arch-z1",
				SHASum:            "sha-sum-z1",
				Filename:          "filename-z1",
				CreatedBy:         "TestCreateProviderPlatform",
				BinaryUploaded:    true,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_provider_platforms\" violates foreign key constraint \"fk_provider_version_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid provider version ID",
			toCreate: &models.TerraformProviderPlatform{
				ProviderVersionID: invalidID,
				OperatingSystem:   "os-z2",
				Architecture:      "arch-z2",
				SHASum:            "sha-sum-z2",
				Filename:          "filename-z2",
				CreatedBy:         "TestCreateProviderPlatform",
				BinaryUploaded:    true,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformProviderPlatforms(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderPlatforms(ctx, testClient, warmupTerraformProviderPlatforms{
		groups:                     standardWarmupGroupsForTerraformProviderPlatforms,
		terraformProviders:         standardWarmupTerraformProvidersForTerraformProviderPlatforms,
		terraformProviderVersions:  standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms,
		terraformProviderPlatforms: standardWarmupTerraformProviderPlatforms,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformProviderPlatform
		expectUpdated *models.TerraformProviderPlatform
		name          string
	}

	// Looks up by ID and version.
	// Updates the BinaryUploaded field.
	positiveTerraformProviderPlatform := warmupItems.terraformProviderPlatforms[0]
	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProviderPlatform.Metadata.ID,
					Version: initialResourceVersion,
				},
				BinaryUploaded: !positiveTerraformProviderPlatform.BinaryUploaded,
			},
			expectUpdated: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:                   positiveTerraformProviderPlatform.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveTerraformProviderPlatform.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ProviderVersionID: positiveTerraformProviderPlatform.ProviderVersionID,
				OperatingSystem:   positiveTerraformProviderPlatform.OperatingSystem,
				Architecture:      positiveTerraformProviderPlatform.Architecture,
				SHASum:            positiveTerraformProviderPlatform.SHASum,
				Filename:          positiveTerraformProviderPlatform.Filename,
				CreatedBy:         positiveTerraformProviderPlatform.CreatedBy,
				BinaryUploaded:    !positiveTerraformProviderPlatform.BinaryUploaded,
			},
		},

		// Because the BinaryUploaded field is the only field that gets updated,
		// It's not possible to make a test case that would duplicate another record.

		{
			name: "negative, non-existent Terraform provider platform ID",
			toUpdate: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformProviderPlatform{
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
			actualTerraformProviderPlatform, err := testClient.client.TerraformProviderPlatforms.UpdateProviderPlatform(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformProviderPlatform)
				compareTerraformProviderPlatforms(t, test.expectUpdated, actualTerraformProviderPlatform, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformProviderPlatform)
			}
		})
	}
}

func TestDeleteProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderPlatforms(ctx, testClient, warmupTerraformProviderPlatforms{
		groups:                     standardWarmupGroupsForTerraformProviderPlatforms,
		terraformProviders:         standardWarmupTerraformProvidersForTerraformProviderPlatforms,
		terraformProviderVersions:  standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms,
		terraformProviderPlatforms: standardWarmupTerraformProviderPlatforms,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformProviderPlatform
		name      string
	}

	// Looks up by ID and version.
	positiveTerraformProviderPlatform := warmupItems.terraformProviderPlatforms[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:      positiveTerraformProviderPlatform.Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform provider platform ID",
			toDelete: &models.TerraformProviderPlatform{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformProviderPlatform{
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
			err := testClient.client.TerraformProviderPlatforms.DeleteProviderPlatform(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformProviderPlatforms = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform provider platform functions",
		FullPath:    "top-level-group-0-for-terraform-provider-platforms",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup terraform providers for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformProvidersForTerraformProviderPlatforms = []models.TerraformProvider{
	{
		Name:        "terraform-provider-0",
		RootGroupID: "top-level-group-0-for-terraform-provider-platforms",
		GroupID:     "top-level-group-0-for-terraform-provider-platforms",
		Private:     false,
		CreatedBy:   "someone-tp0",
	},
	{
		Name:        "terraform-provider-1",
		RootGroupID: "top-level-group-0-for-terraform-provider-platforms",
		GroupID:     "top-level-group-0-for-terraform-provider-platforms",
		Private:     false,
		CreatedBy:   "someone-tp1",
	},
}

// Standard warmup terraform provider versions for tests in this module:
// The necessary ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformProviderVersionsForTerraformProviderPlatforms = []models.TerraformProviderVersion{
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-platforms/terraform-provider-0",
		SemanticVersion:          "1.2.3",
		GPGASCIIArmor:            ptr.String("chain-mail-0"),
		GPGKeyID:                 ptr.Uint64(111222333444555666),
		Protocols:                []string{"protocol-0", "protocol-1"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv0",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-platforms/terraform-provider-0",
		SemanticVersion:          "4.5.6",
		GPGASCIIArmor:            ptr.String("chain-mail-1"),
		GPGKeyID:                 ptr.Uint64(777222333444555666),
		Protocols:                []string{"protocol-2", "protocol-3"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv1",
	},
	{
		ProviderID:               "top-level-group-0-for-terraform-provider-platforms/terraform-provider-1",
		SemanticVersion:          "1.2.3",
		GPGASCIIArmor:            ptr.String("chain-mail-2"),
		GPGKeyID:                 ptr.Uint64(777222333444555666),
		Protocols:                []string{"protocol-4", "protocol-5"},
		SHASumsUploaded:          false,
		SHASumsSignatureUploaded: false,
		CreatedBy:                "someone-tpv2",
	},
}

// Standard warmup terraform provider platforms for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
// Here, the ID field has the provider path, a colon, and the semantic version
var standardWarmupTerraformProviderPlatforms = []models.TerraformProviderPlatform{
	{
		ProviderVersionID: "top-level-group-0-for-terraform-provider-platforms/terraform-provider-0:1.2.3",
		OperatingSystem:   "os-a",
		Architecture:      "arch-n",
		SHASum:            "sha-sum-0",
		Filename:          "filename-0",
		CreatedBy:         "someone-tpp0",
		BinaryUploaded:    false,
	},
	{
		ProviderVersionID: "top-level-group-0-for-terraform-provider-platforms/terraform-provider-0:1.2.3",
		OperatingSystem:   "os-a",
		Architecture:      "arch-o",
		SHASum:            "sha-sum-1",
		Filename:          "filename-1",
		CreatedBy:         "someone-tpp1",
		BinaryUploaded:    false,
	},
	{
		ProviderVersionID: "top-level-group-0-for-terraform-provider-platforms/terraform-provider-0:4.5.6",
		OperatingSystem:   "os-b",
		Architecture:      "arch-n",
		SHASum:            "sha-sum-2",
		Filename:          "filename-2",
		CreatedBy:         "someone-tpp2",
		BinaryUploaded:    true,
	},
	{
		ProviderVersionID: "top-level-group-0-for-terraform-provider-platforms/terraform-provider-1:1.2.3",
		OperatingSystem:   "os-b",
		Architecture:      "arch-o",
		SHASum:            "sha-sum-3",
		Filename:          "filename-3",
		CreatedBy:         "someone-tpp3",
		BinaryUploaded:    true,
	},
	{
		ProviderVersionID: "top-level-group-0-for-terraform-provider-platforms/terraform-provider-1:1.2.3",
		OperatingSystem:   "os-c",
		Architecture:      "arch-p",
		SHASum:            "sha-sum-4",
		Filename:          "filename-4",
		CreatedBy:         "someone-tpp4",
		BinaryUploaded:    true,
	},
}

// createWarmupTerraformProviderPlatforms creates some warmup terraform provider platforms for a test
// The warmup terraform provider platforms to create can be standard or otherwise.
func createWarmupTerraformProviderPlatforms(ctx context.Context, testClient *testClient,
	input warmupTerraformProviderPlatforms,
) (*warmupTerraformProviderPlatforms, error) {
	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultTerraformProviders, providerResourcePath2ID, err := createInitialTerraformProviders(ctx, testClient,
		input.terraformProviders, parentPath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderVersions, providerVersion2ID, err := createInitialTerraformProviderVersions(ctx, testClient,
		input.terraformProviderVersions, providerResourcePath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderPlatforms, err := createInitialTerraformProviderPlatforms(ctx, testClient,
		input.terraformProviderPlatforms, providerVersion2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformProviderPlatforms{
		groups:                     resultGroups,
		terraformProviders:         resultTerraformProviders,
		terraformProviderVersions:  resultTerraformProviderVersions,
		terraformProviderPlatforms: resultTerraformProviderPlatforms,
	}, nil
}

func ptrTerraformProviderPlatformSortableField(arg TerraformProviderPlatformSortableField) *TerraformProviderPlatformSortableField {
	return &arg
}

func (wis terraformProviderPlatformInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderPlatformInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderPlatformInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformProviderPlatformInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformProviderPlatformInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformProviderPlatformInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// terraformProviderPlatformInfoFromTerraformProviderPlatforms returns a slice of terraformProviderPlatformInfo, not necessarily sorted in any order.
func terraformProviderPlatformInfoFromTerraformProviderPlatforms(terraformProviderPlatforms []models.TerraformProviderPlatform) []terraformProviderPlatformInfo {
	result := []terraformProviderPlatformInfo{}

	for _, tp := range terraformProviderPlatforms {
		result = append(result, terraformProviderPlatformInfo{
			id:         tp.Metadata.ID,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformProviderPlatformIDsFromTerraformProviderPlatformInfos preserves order
func terraformProviderPlatformIDsFromTerraformProviderPlatformInfos(terraformProviderPlatformInfos []terraformProviderPlatformInfo) []string {
	result := []string{}
	for _, terraformProviderPlatformInfo := range terraformProviderPlatformInfos {
		result = append(result, terraformProviderPlatformInfo.id)
	}
	return result
}

// compareTerraformProviderPlatforms compares two terraform provider platform objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformProviderPlatforms(t *testing.T, expected, actual *models.TerraformProviderPlatform,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.ProviderVersionID, actual.ProviderVersionID)
	assert.Equal(t, expected.OperatingSystem, actual.OperatingSystem)
	assert.Equal(t, expected.Architecture, actual.Architecture)
	assert.Equal(t, expected.SHASum, actual.SHASum)
	assert.Equal(t, expected.Filename, actual.Filename)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.BinaryUploaded, actual.BinaryUploaded)

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
