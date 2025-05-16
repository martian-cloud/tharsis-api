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

// terraformProviderVersionMirrorInfo aids convenience in accessing the information
// TestGetTerraformProviderVersionMirrors needs about the warmup objects.
type terraformProviderVersionMirrorInfo struct {
	createdTime time.Time
	id          string
}

// terraformProviderVersionMirrorInfoIDSlice makes a slice of terraformProviderVersionMirrorInfo sortable by ID string
type terraformProviderVersionMirrorInfoIDSlice []terraformProviderVersionMirrorInfo

// terraformProviderVersionMirrorInfoCreatedSlice makes a slice of terraformProviderVersionMirrorInfo sortable by last updated time
type terraformProviderVersionMirrorInfoCreatedSlice []terraformProviderVersionMirrorInfo

// warmupTerraformProviderVersionMirrors holds the inputs to and outputs from createWarmupTerraformProviderVersionMirrors.
type warmupTerraformProviderVersionMirrors struct {
	groups                 []models.Group
	providerVersionMirrors []models.TerraformProviderVersionMirror
}

func TestGetVersionMirrorByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	warmupItems, err := createWarmupTerraformProviderVersionMirrors(ctx, testClient, warmupTerraformProviderVersionMirrors{
		groups:                 standardWarmupGroupsForTerraformProviderVersionMirrors,
		providerVersionMirrors: standardWarmupTerraformProviderVersionMirrors,
	})
	require.Nil(t, err)
	createdHigh := time.Now()

	type testCase struct {
		expectMsg                   *string
		expectProviderVersionMirror *models.TerraformProviderVersionMirror
		name                        string
		searchID                    string
	}

	positiveProviderVersionMirror := warmupItems.providerVersionMirrors[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveProviderVersionMirror.Metadata.ID,
			expectProviderVersionMirror: &models.TerraformProviderVersionMirror{
				Metadata: models.ResourceMetadata{
					ID:                positiveProviderVersionMirror.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				GroupID:           positiveProviderVersionMirror.GroupID,
				CreatedBy:         positiveProviderVersionMirror.CreatedBy,
				Type:              positiveProviderVersionMirror.Type,
				SemanticVersion:   positiveProviderVersionMirror.SemanticVersion,
				RegistryNamespace: positiveProviderVersionMirror.RegistryNamespace,
				RegistryHostname:  positiveProviderVersionMirror.RegistryHostname,
				Digests:           positiveProviderVersionMirror.Digests,
			},
		},
		{
			name:     "negative, non-existent Terraform provider version ID",
			searchID: nonExistentID,
			// expect terraform provider version mirror and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualProviderVersionMirror, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrorByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectProviderVersionMirror != nil {
				require.NotNil(t, actualProviderVersionMirror)
				compareTerraformProviderVersionMirrors(t, test.expectProviderVersionMirror, actualProviderVersionMirror, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualProviderVersionMirror)
			}
		})
	}
}

func TestGetVersionMirrorByTRN(t *testing.T) {
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

	type testCase struct {
		name            string
		trn             string
		expectMirror    bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "get version mirror by TRN",
			trn:          versionMirror.Metadata.TRN,
			expectMirror: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TerraformProviderVersionMirrorModelType.BuildTRN(group.FullPath, "registry.hashicorp.io", "hashicorp", "github", "0.3.0"),
		},
		{
			name:            "version mirror TRN has less than 5 parts",
			trn:             types.TerraformProviderVersionMirrorModelType.BuildTRN("registry.hashicorp.io", "hashicorp", "github"),
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
			actualMirror, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrorByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectMirror {
				require.NotNil(t, actualMirror)
				assert.Equal(t,
					types.TerraformProviderVersionMirrorModelType.BuildTRN(
						group.FullPath,
						versionMirror.RegistryHostname,
						versionMirror.RegistryNamespace,
						versionMirror.Type,
						versionMirror.SemanticVersion,
					),
					actualMirror.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualMirror)
			}
		})
	}
}

func TestGetVersionMirrors(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersionMirrors(ctx, testClient, warmupTerraformProviderVersionMirrors{
		groups:                 standardWarmupGroupsForTerraformProviderVersionMirrors,
		providerVersionMirrors: standardWarmupTerraformProviderVersionMirrors,
	})
	require.Nil(t, err)
	allProviderVersionMirrorInfos := terraformProviderVersionMirrorInfoFromTerraformProviderVersionMirrors(warmupItems.providerVersionMirrors)

	// Sort by Terraform provider version mirror IDs.
	sort.Sort(terraformProviderVersionMirrorInfoIDSlice(allProviderVersionMirrorInfos))
	allProviderVersionMirrorIDs := terraformProviderVersionMirrorIDsFromTerraformProviderVersionMirrorInfos(allProviderVersionMirrorInfos)

	// Sort by last update times.
	sort.Sort(terraformProviderVersionMirrorInfoCreatedSlice(allProviderVersionMirrorInfos))
	allProviderVersionMirrorIDsByTime := terraformProviderVersionMirrorIDsFromTerraformProviderVersionMirrorInfos(allProviderVersionMirrorInfos)
	reverseProviderVersionMirrorIDsByTime := reverseStringSlice(allProviderVersionMirrorIDsByTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError         error
		expectEndCursorError           error
		input                          *GetProviderVersionMirrorsInput
		expectMsg                      *string
		name                           string
		expectPageInfo                 pagination.PageInfo
		expectProviderVersionMirrorIDs []string
		getBeforeCursorFromPrevious    bool
		sortedDescending               bool
		expectHasStartCursor           bool
		getAfterCursorFromPrevious     bool
		expectHasEndCursor             bool
	}

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetProviderVersions, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetProviderVersionMirrorsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDs,
			expectPageInfo:                 pagination.PageInfo{TotalCount: int32(len(allProviderVersionMirrorIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: int32(len(allProviderVersionMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "sort in ascending order of time of creation time",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: int32(len(allProviderVersionMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "sort in descending order of time of creation time",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtDesc),
			},
			sortedDescending:               true,
			expectProviderVersionMirrorIDs: reverseProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: int32(len(reverseProviderVersionMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "pagination: everything at once",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: int32(len(allProviderVersionMirrorIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "pagination: first two",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allProviderVersionMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:     true,
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allProviderVersionMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:     true,
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allProviderVersionMirrorIDs)),
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
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:               true,
			expectProviderVersionMirrorIDs: reverseProviderVersionMirrorIDsByTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allProviderVersionMirrorIDs)),
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
			input: &GetProviderVersionMirrorsInput{
				Sort:              ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:     true,
			getBeforeCursorFromPrevious:    true,
			expectMsg:                      ptr.String("only before or after can be defined, not both"),
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                      ptr.String("only first or last can be defined, not both"),
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allProviderVersionMirrorIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryHostname:  ptr.String(""),
					RegistryNamespace: ptr.String(""),
					Type:              ptr.String(""),
					GroupID:           ptr.String(""),
					SemanticVersion:   ptr.String(""),
				},
			},
			expectMsg:                      emptyUUIDMsg2,
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{},
		},

		{
			name: "filter, group ID, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					GroupID: ptr.String(warmupItems.providerVersionMirrors[0].GroupID),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:                      invalidUUIDMsg2,
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{},
		},

		{
			name: "filter, registry hostname, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryHostname: ptr.String(warmupItems.providerVersionMirrors[0].RegistryHostname),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, registry hostname, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryHostname: ptr.String("invalid.tld"),
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, registry namespace, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryNamespace: ptr.String(warmupItems.providerVersionMirrors[0].RegistryNamespace),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, registry namespace, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					RegistryNamespace: ptr.String("invalid"),
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider type, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					Type: ptr.String(warmupItems.providerVersionMirrors[1].Type),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime[1:2],
			expectPageInfo:                 pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, provider type, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					Type: ptr.String("random"),
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, semantic version, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					SemanticVersion: ptr.String("0.2.0"),
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime[1:2],
			expectPageInfo:                 pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, semantic version, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					SemanticVersion: ptr.String("9.8.7"),
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, semantic version, invalid",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					SemanticVersion: ptr.String("this-is-not-a-valid-semantic-version"),
				},
			},
			// expect no error, just an empty return slice
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider version IDs, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					VersionMirrorIDs: []string{
						allProviderVersionMirrorIDsByTime[0],
						allProviderVersionMirrorIDsByTime[3],
					},
				},
			},
			expectProviderVersionMirrorIDs: []string{
				allProviderVersionMirrorIDsByTime[0],
				allProviderVersionMirrorIDsByTime[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, provider version IDs, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					VersionMirrorIDs: []string{nonExistentID},
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, provider version IDs, invalid",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					VersionMirrorIDs: []string{invalidID},
				},
			},
			expectMsg:                      invalidUUIDMsg2,
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{},
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					NamespacePaths: []string{
						warmupItems.groups[0].FullPath,
					},
				},
			},
			expectProviderVersionMirrorIDs: allProviderVersionMirrorIDsByTime,
			expectPageInfo:                 pagination.PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor:           true,
			expectHasEndCursor:             true,
		},

		{
			name: "filter, namespace paths, non-existent",
			input: &GetProviderVersionMirrorsInput{
				Sort: ptrTerraformProviderVersionMirrorSortableField(TerraformProviderVersionMirrorSortableFieldCreatedAtAsc),
				Filter: &TerraformProviderVersionMirrorFilter{
					NamespacePaths: []string{
						"invalid/path",
					},
				},
			},
			expectProviderVersionMirrorIDs: []string{},
			expectPageInfo:                 pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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

			providerVersionMirrorsResult, err := testClient.client.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, providerVersionMirrorsResult.PageInfo)
				assert.NotNil(t, providerVersionMirrorsResult.VersionMirrors)
				pageInfo := providerVersionMirrorsResult.PageInfo
				providerVersionMirrors := providerVersionMirrorsResult.VersionMirrors

				// Check the terraform provider version mirrors result by comparing a list of the terraform provider version mirror IDs.
				actualProviderVersionMirrorIDs := []string{}
				for _, terraformProviderVersion := range providerVersionMirrors {
					actualProviderVersionMirrorIDs = append(actualProviderVersionMirrorIDs, terraformProviderVersion.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualProviderVersionMirrorIDs)
				}

				assert.Equal(t, len(test.expectProviderVersionMirrorIDs), len(actualProviderVersionMirrorIDs))
				assert.Equal(t, test.expectProviderVersionMirrorIDs, actualProviderVersionMirrorIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one terraform provider version returned.
				// If there are no terraform provider versions returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(providerVersionMirrors) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&providerVersionMirrors[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&providerVersionMirrors[len(providerVersionMirrors)-1])
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

func TestCreateVersionMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersionMirrors(ctx, testClient, warmupTerraformProviderVersionMirrors{
		groups: standardWarmupGroupsForTerraformProviderVersionMirrors,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformProviderVersionMirror
		expectCreated *models.TerraformProviderVersionMirror
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformProviderVersionMirror{
				GroupID:           warmupItems.groups[0].Metadata.ID,
				SemanticVersion:   "2.4.6",
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				CreatedBy:         "TestCreateVersionMirror",
				Digests: map[string][]byte{
					"file": []byte("sum"),
				},
			},
			expectCreated: &models.TerraformProviderVersionMirror{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				GroupID:           warmupItems.groups[0].Metadata.ID,
				SemanticVersion:   "2.4.6",
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				CreatedBy:         "TestCreateVersionMirror",
				Digests: map[string][]byte{
					"file": []byte("sum"),
				},
			},
		},

		{
			name: "duplicate provider version mirror",
			toCreate: &models.TerraformProviderVersionMirror{
				GroupID:           warmupItems.groups[0].Metadata.ID,
				SemanticVersion:   "2.4.6",
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				CreatedBy:         "would-be-duplicate-provider-id-and-semantic-version",
			},
			expectMsg: ptr.String("terraform provider version is already mirrored"),
		},

		{
			name: "negative, non-existent group ID",
			toCreate: &models.TerraformProviderVersionMirror{
				GroupID:         nonExistentID,
				SemanticVersion: "2.4.9",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_provider_version_mirrors\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid group ID",
			toCreate: &models.TerraformProviderVersionMirror{
				GroupID:         invalidID,
				SemanticVersion: "2.5.9",
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformProviderVersionMirrors(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestDeleteVersionMirror(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformProviderVersionMirrors(ctx, testClient, warmupTerraformProviderVersionMirrors{
		groups:                 standardWarmupGroupsForTerraformProviderVersionMirrors,
		providerVersionMirrors: standardWarmupTerraformProviderVersionMirrors,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformProviderVersionMirror
		name      string
	}

	// Looks up by ID and version.
	positiveProviderVersionMirror := warmupItems.providerVersionMirrors[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.TerraformProviderVersionMirror{
				Metadata: models.ResourceMetadata{
					ID:      positiveProviderVersionMirror.Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform provider version ID",
			toDelete: &models.TerraformProviderVersionMirror{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformProviderVersionMirror{
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
			err := testClient.client.TerraformProviderVersionMirrors.DeleteVersionMirror(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformProviderVersionMirrors = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform provider version mirror functions",
		FullPath:    "top-level-group-0-for-terraform-provider-version-mirrors",
		CreatedBy:   "someone-g0",
	},
}

// standardWarmupTerraformProviderVersionMirrors for tests in this module.
var standardWarmupTerraformProviderVersionMirrors = []models.TerraformProviderVersionMirror{
	{
		GroupID:           "top-level-group-0-for-terraform-provider-version-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		CreatedBy:         "someone-vm0",
		SemanticVersion:   "0.1.0",
		Digests: map[string][]byte{ // This should help us verify JSON marshalling / un-marshalling is working.
			"file": []byte("sum"),
		},
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-version-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "null",
		CreatedBy:         "someone-vm1",
		SemanticVersion:   "0.2.0",
		Digests: map[string][]byte{
			"file": []byte("sum"),
		},
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-version-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "time",
		CreatedBy:         "someone-vm2",
		SemanticVersion:   "1.0.0",
		Digests: map[string][]byte{
			"file": []byte("sum"),
		},
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-version-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "azurerm",
		CreatedBy:         "someone-vm3",
		SemanticVersion:   "2.0.0",
		Digests: map[string][]byte{
			"file": []byte("sum"),
		},
	},
	{
		GroupID:           "top-level-group-0-for-terraform-provider-version-mirrors",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "kubernetes",
		CreatedBy:         "someone-vm4",
		SemanticVersion:   "0.5.0",
		Digests: map[string][]byte{
			"file": []byte("sum"),
		},
	},
}

// createWarmupTerraformProviderVersionMirrors creates some warmup terraform provider version mirrors for a test.
// The warmup terraform provider version mirrors to create can be standard or otherwise.
func createWarmupTerraformProviderVersionMirrors(
	ctx context.Context,
	testClient *testClient,
	input warmupTerraformProviderVersionMirrors,
) (*warmupTerraformProviderVersionMirrors, error) {
	// It is necessary to create at least one group in order to
	// provide the necessary IDs for the terraform provider version mirrors.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultVersionMirrors, _, err := createInitialTerraformProviderVersionMirrors(ctx, testClient,
		input.providerVersionMirrors, parentPath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformProviderVersionMirrors{
		groups:                 resultGroups,
		providerVersionMirrors: resultVersionMirrors,
	}, nil
}

// createInitialTerraformProviderVersionMirrors creates some warmup Terraform provider version mirrors for a test.
func createInitialTerraformProviderVersionMirrors(
	ctx context.Context,
	testClient *testClient,
	toCreate []models.TerraformProviderVersionMirror,
	groupPath2ID map[string]string,
) ([]models.TerraformProviderVersionMirror, map[string]string, error) {
	result := []models.TerraformProviderVersionMirror{}
	versionSpecs2ID := make(map[string]string)

	for _, input := range toCreate {
		groupPath := input.GroupID
		groupID, ok := groupPath2ID[groupPath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformProviderVersionMirrors failed to look up group path: %s", groupPath)
		}
		input.GroupID = groupID

		created, err := testClient.client.TerraformProviderVersionMirrors.CreateVersionMirror(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		key := fmt.Sprintf("%s/%s/%s/%s/%s", groupPath, input.RegistryHostname, input.RegistryNamespace, input.Type, input.SemanticVersion)
		versionSpecs2ID[key] = created.Metadata.ID
	}

	return result, versionSpecs2ID, nil
}

func ptrTerraformProviderVersionMirrorSortableField(arg TerraformProviderVersionMirrorSortableField) *TerraformProviderVersionMirrorSortableField {
	return &arg
}

func (s terraformProviderVersionMirrorInfoIDSlice) Len() int {
	return len(s)
}

func (s terraformProviderVersionMirrorInfoIDSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s terraformProviderVersionMirrorInfoIDSlice) Less(i, j int) bool {
	return s[i].id < s[j].id
}

func (s terraformProviderVersionMirrorInfoCreatedSlice) Len() int {
	return len(s)
}

func (s terraformProviderVersionMirrorInfoCreatedSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s terraformProviderVersionMirrorInfoCreatedSlice) Less(i, j int) bool {
	return s[i].createdTime.Before(s[j].createdTime)
}

// terraformProviderVersionMirrorInfoFromTerraformProviderVersionMirrors returns a slice of terraformProviderVersionMirrorInfo, not necessarily sorted in any order.
func terraformProviderVersionMirrorInfoFromTerraformProviderVersionMirrors(
	terraformProviderVersionMirrors []models.TerraformProviderVersionMirror,
) []terraformProviderVersionMirrorInfo {
	result := []terraformProviderVersionMirrorInfo{}

	for _, tp := range terraformProviderVersionMirrors {
		result = append(result, terraformProviderVersionMirrorInfo{
			id:          tp.Metadata.ID,
			createdTime: *tp.Metadata.CreationTimestamp,
		})
	}

	return result
}

// terraformProviderVersionMirrorIDsFromTerraformProviderVersionMirrorInfos preserves order
func terraformProviderVersionMirrorIDsFromTerraformProviderVersionMirrorInfos(
	terraformProviderVersionMirrorInfos []terraformProviderVersionMirrorInfo,
) []string {
	result := []string{}
	for _, versionMirrorInfo := range terraformProviderVersionMirrorInfos {
		result = append(result, versionMirrorInfo.id)
	}
	return result
}

// compareTerraformProviderVersionMirrors compares two terraform provider version mirror objects, including bounds
// for creation and updated times. If times is nil, it compares the exact metadata timestamps.
func compareTerraformProviderVersionMirrors(t *testing.T, expected, actual *models.TerraformProviderVersionMirror,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.RegistryHostname, actual.RegistryHostname)
	assert.Equal(t, expected.RegistryNamespace, actual.RegistryNamespace)
	assert.Equal(t, expected.SemanticVersion, actual.SemanticVersion)
	assert.Equal(t, expected.Type, actual.Type)
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
