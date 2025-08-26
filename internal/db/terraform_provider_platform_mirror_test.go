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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// terraformProviderPlatformMirrorInfo aids convenience in accessing the information
// TestGetTerraformProviderPlatformMirrors needs about the warmup objects.
type terraformProviderPlatformMirrorInfo struct {
	createTime time.Time
	id         string
}

// terraformProviderPlatformMirrorInfoIDSlice makes a slice of terraformProviderPlatformMirrorInfo sortable by ID string
type terraformProviderPlatformMirrorInfoIDSlice []terraformProviderPlatformMirrorInfo

// terraformProviderPlatformMirrorInfoCreateSlice makes a slice of terraformProviderPlatformMirrorInfo sortable by last updated time
type terraformProviderPlatformMirrorInfoCreateSlice []terraformProviderPlatformMirrorInfo

// warmupTerraformProviderPlatformMirrors holds the inputs to and outputs from createWarmupTerraformProviderPlatformMirrors.
type warmupTerraformProviderPlatformMirrors struct {
	groups                           []models.Group
	terraformProviderVersionMirrors  []models.TerraformProviderVersionMirror
	terraformProviderPlatformMirrors []models.TerraformProviderPlatformMirror
}

func TestGetPlatformMirrorByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupItems, err := createWarmupTerraformProviderPlatformMirrors(ctx, testClient, warmupTerraformProviderPlatformMirrors{
		groups:                           standardWarmupGroupsForTerraformProviderPlatformMirrors,
		terraformProviderVersionMirrors:  standardWarmupTerraformProviderVersionMirrorsForTerraformProviderPlatformMirrors,
		terraformProviderPlatformMirrors: standardWarmupTerraformProviderPlatformMirrors,
	})
	require.Nil(t, err)
	createdHigh := time.Now()

	type testCase struct {
		expectMsg                    *string
		expectProviderPlatformMirror *models.TerraformProviderPlatformMirror
		name                         string
		searchID                     string
	}

	positiveProviderPlatformMirror := warmupItems.terraformProviderPlatformMirrors[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveProviderPlatformMirror.Metadata.ID,
			expectProviderPlatformMirror: &models.TerraformProviderPlatformMirror{
				Metadata: models.ResourceMetadata{
					ID:                positiveProviderPlatformMirror.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				OS:              positiveProviderPlatformMirror.OS,
				VersionMirrorID: positiveProviderPlatformMirror.VersionMirrorID,
				Architecture:    positiveProviderPlatformMirror.Architecture,
			},
		},

		{
			name:     "negative, non-existent Terraform provider platform mirror ID",
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
			actualProviderPlatformMirror, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrorByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectProviderPlatformMirror != nil {
				require.NotNil(t, actualProviderPlatformMirror)
				compareTerraformProviderPlatformMirrors(t, test.expectProviderPlatformMirror, actualProviderPlatformMirror, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualProviderPlatformMirror)
			}
		})
	}
}

func TestGetPlatformMirrorByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	versionMirror, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &models.TerraformProviderVersionMirror{
		RegistryHostname:  "registry.hashicorp.io",
		RegistryNamespace: "hashicorp",
		Type:              "github",
		SemanticVersion:   "0.2.0",
		GroupID:           group.Metadata.ID,
	})
	require.NoError(t, err)

	platformMirror, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &models.TerraformProviderPlatformMirror{
		VersionMirrorID: versionMirror.Metadata.ID,
		OS:              "linux",
		Architecture:    "amd64",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectMirror    bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "get platform mirror by TRN",
			trn:          platformMirror.Metadata.TRN,
			expectMirror: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TerraformProviderPlatformMirrorModelType.BuildTRN(group.FullPath, "registry.hashicorp.io", "hashicorp", "github", "0.2.0", "darwin", "amd64"),
		},
		{
			name:            "platform mirror TRN has less than 7 parts",
			trn:             types.TerraformProviderPlatformMirrorModelType.BuildTRN("registry.hashicorp.io", "hashicorp", "github", "0.2.0"),
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
			actualMirror, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrorByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectMirror {
				require.NotNil(t, actualMirror)
				assert.Equal(t,
					types.TerraformProviderPlatformMirrorModelType.BuildTRN(
						group.FullPath,
						versionMirror.RegistryHostname,
						versionMirror.RegistryNamespace,
						versionMirror.Type,
						versionMirror.SemanticVersion,
						platformMirror.OS,
						platformMirror.Architecture,
					),
					actualMirror.Metadata.TRN)
			} else {
				assert.Nil(t, actualMirror)
			}
		})
	}
}

func TestGetPlatformMirrors(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderPlatformMirrors(ctx, testClient, warmupTerraformProviderPlatformMirrors{
		groups:                           standardWarmupGroupsForTerraformProviderPlatformMirrors,
		terraformProviderVersionMirrors:  standardWarmupTerraformProviderVersionMirrorsForTerraformProviderPlatformMirrors,
		terraformProviderPlatformMirrors: standardWarmupTerraformProviderPlatformMirrors,
	})
	require.Nil(t, err)
	allTerraformProviderPlatformMirrorInfos := terraformProviderPlatformMirrorInfoFromTerraformProviderPlatformMirrors(
		warmupItems.terraformProviderPlatformMirrors)

	// Sort by Terraform provider platform IDs.
	sort.Sort(terraformProviderPlatformMirrorInfoIDSlice(allTerraformProviderPlatformMirrorInfos))
	allTerraformProviderPlatformMirrorIDs := terraformProviderPlatformMirrorIDsFromTerraformProviderPlatformMirrorInfos(allTerraformProviderPlatformMirrorInfos)

	// Sort by last update times.
	sort.Sort(terraformProviderPlatformMirrorInfoCreateSlice(allTerraformProviderPlatformMirrorInfos))
	allTerraformProviderPlatformMirrorIDsByTime := terraformProviderPlatformMirrorIDsFromTerraformProviderPlatformMirrorInfos(allTerraformProviderPlatformMirrorInfos)
	reverseTerraformProviderPlatformMirrorIDsByTime := reverseStringSlice(allTerraformProviderPlatformMirrorIDsByTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError             error
		expectEndCursorError               error
		input                              *GetProviderPlatformMirrorsInput
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

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetPlatformMirrors, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetProviderPlatformMirrorsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDs,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformMirrorIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformMirrorIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "sort in ascending order of time of creation",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "sort in descending order of time of creation",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtDesc),
			},
			sortedDescending:                   true,
			expectTerraformProviderPlatformIDs: reverseTerraformProviderPlatformMirrorIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "pagination: everything at once",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime,
			expectPageInfo:                     pagination.PageInfo{TotalCount: int32(len(allTerraformProviderPlatformMirrorIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "pagination: first two",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:         true,
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:         true,
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformMirrorIDs)),
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
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:                   true,
			expectTerraformProviderPlatformIDs: reverseTerraformProviderPlatformMirrorIDsByTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformMirrorIDs)),
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
			input: &GetProviderPlatformMirrorsInput{
				Sort:              ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
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
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                          ptr.String("only first or last can be defined, not both"),
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTerraformProviderPlatformMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &TerraformProviderPlatformMirrorFilter{
					VersionMirrorID: ptr.String(""),
					OS:              ptr.String(""),
					Architecture:    ptr.String(""),
				},
			},
			expectMsg:                          invalidUUIDMsg,
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "filter, version mirror ID, positive",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					VersionMirrorID: ptr.String(warmupItems.terraformProviderVersionMirrors[0].Metadata.ID),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime[:2],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, version mirror ID, non-existent",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					VersionMirrorID: ptr.String(nonExistentID),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, version mirror ID, invalid",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					VersionMirrorID: ptr.String(invalidID),
				},
			},
			expectMsg:                          invalidUUIDMsg,
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{},
		},

		{
			name: "filter, operating system, positive",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					OS: ptr.String(warmupItems.terraformProviderPlatformMirrors[0].OS),
				},
			},
			expectTerraformProviderPlatformIDs: allTerraformProviderPlatformMirrorIDsByTime[:1],
			expectPageInfo:                     pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:               true,
			expectHasEndCursor:                 true,
		},

		{
			name: "filter, operating system, non-existent",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					OS: ptr.String("this-operating-system-does-not-exist"),
				},
			},
			expectTerraformProviderPlatformIDs: []string{},
			expectPageInfo:                     pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, architecture, positive",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
					Architecture: ptr.String(warmupItems.terraformProviderPlatformMirrors[0].Architecture),
				},
			},
			expectTerraformProviderPlatformIDs: []string{
				allTerraformProviderPlatformMirrorIDsByTime[0],
				allTerraformProviderPlatformMirrorIDsByTime[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, architecture, non-existent",
			input: &GetProviderPlatformMirrorsInput{
				Sort: ptrTerraformProviderPlatformMirrorSortableField(TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderPlatformMirrorFilter{
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

			providerPlatformMirrorsResult, err := testClient.client.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, providerPlatformMirrorsResult.PageInfo)
				assert.NotNil(t, providerPlatformMirrorsResult.PlatformMirrors)
				pageInfo := providerPlatformMirrorsResult.PageInfo
				providerPlatformMirrors := providerPlatformMirrorsResult.PlatformMirrors

				// Check the terraform provider platform mirrors result by comparing a list of the terraform provider platform mirror IDs.
				actualTerraformProviderPlatformIDs := []string{}
				for _, terraformProviderPlatform := range providerPlatformMirrors {
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
				if len(providerPlatformMirrors) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&providerPlatformMirrors[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&providerPlatformMirrors[len(providerPlatformMirrors)-1])
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

func TestCreatePlatformMirror(t *testing.T) {

}

func TestDeletePlatformMirror(t *testing.T) {

}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformProviderPlatformMirrors = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform provider platform mirror functions",
		FullPath:    "top-level-group-0-for-terraform-provider-platform-mirrors",
		CreatedBy:   "someone-g0",
	},
}

// standardWarmupTerraformProviderVersionMirrorsForTerraformProviderPlatformMirrors for tests in this module.
var standardWarmupTerraformProviderVersionMirrorsForTerraformProviderPlatformMirrors = []models.TerraformProviderVersionMirror{
	{
		GroupID:           "top-level-group-0-for-terraform-provider-platform-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		CreatedBy:         "someone-vm0",
		SemanticVersion:   "0.1.0",
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-platform-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "null",
		CreatedBy:         "someone-vm1",
		SemanticVersion:   "0.2.0",
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-platform-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "time",
		CreatedBy:         "someone-vm2",
		SemanticVersion:   "1.0.0",
	},
}

// standardWarmupTerraformProviderPlatformMirrors for tests in this module.
var standardWarmupTerraformProviderPlatformMirrors = []models.TerraformProviderPlatformMirror{
	{
		VersionMirrorID: "top-level-group-0-for-terraform-provider-platform-mirrors/registry.terraform.io/hashicorp/aws/0.1.0",
		OS:              "windows",
		Architecture:    "amd64",
	},
	{
		VersionMirrorID: "top-level-group-0-for-terraform-provider-platform-mirrors/registry.terraform.io/hashicorp/aws/0.1.0",
		OS:              "linux",
		Architecture:    "386",
	},
	{
		VersionMirrorID: "top-level-group-0-for-terraform-provider-platform-mirrors/registry.terraform.io/hashicorp/null/0.2.0",
		OS:              "freebsd",
		Architecture:    "arm",
	},
	{
		VersionMirrorID: "top-level-group-0-for-terraform-provider-platform-mirrors/registry.terraform.io/hashicorp/time/1.0.0",
		OS:              "darwin",
		Architecture:    "amd64",
	},
	{
		VersionMirrorID: "top-level-group-0-for-terraform-provider-platform-mirrors/registry.terraform.io/hashicorp/time/1.0.0",
		OS:              "darwin",
		Architecture:    "arm64",
	},
}

// createWarmupTerraformProviderPlatformMirrors creates some warmup terraform provider platform mirrors for a test
// The warmup terraform provider platform mirrors to create can be standard or otherwise.
func createWarmupTerraformProviderPlatformMirrors(
	ctx context.Context,
	testClient *testClient,
	input warmupTerraformProviderPlatformMirrors,
) (*warmupTerraformProviderPlatformMirrors, error) {
	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultProviderVersionMirrors, providerResourcePath2ID, err := createInitialTerraformProviderVersionMirrors(ctx, testClient,
		input.terraformProviderVersionMirrors, parentPath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderPlatforms, err := createInitialTerraformProviderPlatformMirrors(ctx, testClient,
		input.terraformProviderPlatformMirrors, providerResourcePath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformProviderPlatformMirrors{
		groups:                           resultGroups,
		terraformProviderVersionMirrors:  resultProviderVersionMirrors,
		terraformProviderPlatformMirrors: resultTerraformProviderPlatforms,
	}, nil
}

// createInitialTerraformProviderPlatformMirrors creates some warmup Terraform provider platform mirrors for a test.
func createInitialTerraformProviderPlatformMirrors(
	ctx context.Context,
	testClient *testClient,
	toCreate []models.TerraformProviderPlatformMirror,
	versionSpecs2ID map[string]string,
) ([]models.TerraformProviderPlatformMirror, error) {
	result := []models.TerraformProviderPlatformMirror{}

	for _, input := range toCreate {
		versionSpecs := input.VersionMirrorID
		versionID, ok := versionSpecs2ID[versionSpecs]
		if !ok {
			return nil,
				fmt.Errorf("createInitialTerraformProviderPlatformMirrors failed to look up version specs: %s", versionSpecs)
		}
		input.VersionMirrorID = versionID

		created, err := testClient.client.TerraformProviderPlatformMirrors.CreatePlatformMirror(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

func ptrTerraformProviderPlatformMirrorSortableField(arg TerraformProviderPlatformMirrorSortableField) *TerraformProviderPlatformMirrorSortableField {
	return &arg
}

func (s terraformProviderPlatformMirrorInfoIDSlice) Len() int {
	return len(s)
}

func (s terraformProviderPlatformMirrorInfoIDSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s terraformProviderPlatformMirrorInfoIDSlice) Less(i, j int) bool {
	return s[i].id < s[j].id
}

func (s terraformProviderPlatformMirrorInfoCreateSlice) Len() int {
	return len(s)
}

func (s terraformProviderPlatformMirrorInfoCreateSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s terraformProviderPlatformMirrorInfoCreateSlice) Less(i, j int) bool {
	return s[i].createTime.Before(s[j].createTime)
}

// terraformProviderPlatformMirrorInfoFromTerraformProviderPlatformMirrors returns a slice of terraformProviderPlatformMirrorInfo, not necessarily sorted in any order.
func terraformProviderPlatformMirrorInfoFromTerraformProviderPlatformMirrors(
	providerPlatformMirrors []models.TerraformProviderPlatformMirror,
) []terraformProviderPlatformMirrorInfo {
	result := []terraformProviderPlatformMirrorInfo{}

	for _, tp := range providerPlatformMirrors {
		result = append(result, terraformProviderPlatformMirrorInfo{
			id:         tp.Metadata.ID,
			createTime: *tp.Metadata.CreationTimestamp,
		})
	}

	return result
}

// terraformProviderPlatformMirrorIDsFromTerraformProviderPlatformMirrorInfos preserves order
func terraformProviderPlatformMirrorIDsFromTerraformProviderPlatformMirrorInfos(
	providerPlatformMirrorInfos []terraformProviderPlatformMirrorInfo,
) []string {
	result := []string{}
	for _, info := range providerPlatformMirrorInfos {
		result = append(result, info.id)
	}
	return result
}

// compareTerraformProviderPlatformMirrors compares two terraform provider platform mirror objects, including bounds
// for creation and updated times. If times is nil, it compares the exact metadata timestamps.
func compareTerraformProviderPlatformMirrors(t *testing.T, expected, actual *models.TerraformProviderPlatformMirror,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.VersionMirrorID, actual.VersionMirrorID)
	assert.Equal(t, expected.OS, actual.OS)
	assert.Equal(t, expected.Architecture, actual.Architecture)

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
