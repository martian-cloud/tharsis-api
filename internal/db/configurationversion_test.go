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

// configurationVersionInfo aids convenience in accessing the information TestGetConfigurationVersions
// needs about the warmup configuration versions.
type configurationVersionInfo struct {
	updateTime             time.Time
	configurationVersionID string
}

// configurationVersionInfoIDSlice makes a slice of configurationVersionInfo sortable by ID string
type configurationVersionInfoIDSlice []configurationVersionInfo

// configurationVersionInfoUpdateSlice makes a slice of configurationVersionInfo sortable by last updated time
type configurationVersionInfoUpdateSlice []configurationVersionInfo

func TestGetConfigurationVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a configuration version with a specific ID without going into the really
	// low-level stuff, create the warmup configuration version(s) then find the relevant ID.
	createdLow := currentTime()
	_, createdWarmupConfigurationVersions, err := createWarmupConfigurationVersions(ctx, testClient,
		standardWarmupGroupsForConfigurationVersions,
		standardWarmupWorkspacesForConfigurationVersions,
		standardWarmupConfigurationVersions)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectConfigurationVersion *models.ConfigurationVersion
		expectMsg                  *string
		name                       string
		searchID                   string
	}

	// Do only one positive test case, because the logic is theoretically the same for all configuration versions.
	positiveConfigurationVersion := createdWarmupConfigurationVersions[0]
	testCases := []testCase{
		{
			name:                       "positive",
			searchID:                   positiveConfigurationVersion.Metadata.ID,
			expectConfigurationVersion: &positiveConfigurationVersion,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect configuration version to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configurationVersion, err := testClient.client.ConfigurationVersions.GetConfigurationVersionByID(ctx,
				test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectConfigurationVersion != nil {
				require.NotNil(t, configurationVersion)
				compareConfigurationVersions(t, test.expectConfigurationVersion, configurationVersion, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, configurationVersion)
			}
		})
	}
}

func TestGetConfigurationVersionByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(20),
	})
	require.NoError(t, err)

	cv, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectApply     bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "get resource by TRN",
			trn:         cv.Metadata.TRN,
			expectApply: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.ConfigurationVersionModelType.BuildTRN(workspace.FullPath, nonExistentGlobalID),
		},
		{
			name:            "a configuration version TRN cannot have less than two parts",
			trn:             types.ConfigurationVersionModelType.BuildTRN(nonExistentGlobalID),
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
			actualCV, err := testClient.client.ConfigurationVersions.GetConfigurationVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectApply {
				require.NotNil(t, actualCV)
				assert.Equal(t, types.ConfigurationVersionModelType.BuildTRN(workspace.FullPath, cv.GetGlobalID()), actualCV.Metadata.TRN)
			} else {
				assert.Nil(t, actualCV)
			}
		})
	}
}

func TestGetConfigurationVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, warmupConfigurationVersions, err := createWarmupConfigurationVersions(ctx, testClient,
		standardWarmupGroupsForConfigurationVersions,
		standardWarmupWorkspacesForConfigurationVersions,
		standardWarmupConfigurationVersions)
	require.Nil(t, err)
	allConfigurationVersionInfos := configurationVersionInfoFromConfigurationVersions(warmupConfigurationVersions)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(configurationVersionInfoIDSlice(allConfigurationVersionInfos))
	allConfigurationVersionIDs := configurationVersionIDsFromConfigurationVersionInfos(allConfigurationVersionInfos)

	// Sort by last update times.
	sort.Sort(configurationVersionInfoUpdateSlice(allConfigurationVersionInfos))
	allConfigurationVersionIDsByUpdateTime := configurationVersionIDsFromConfigurationVersionInfos(allConfigurationVersionInfos)
	reverseConfigurationVersionIDsByUpdateTime := reverseStringSlice(allConfigurationVersionIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError        error
		expectEndCursorError          error
		expectMsg                     *string
		input                         *GetConfigurationVersionsInput
		name                          string
		expectPageInfo                pagination.PageInfo
		expectConfigurationVersionIDs []string
		getBeforeCursorFromPrevious   bool
		getAfterCursorFromPrevious    bool
		expectHasStartCursor          bool
		expectHasEndCursor            bool
	}

	/*
		template test case:

		{
		name                          string
		input                         *GetConfigurationVersionsInput
		getAfterCursorFromPrevious    bool
		getBeforeCursorFromPrevious   bool
		expectMsg                     *string
		expectConfigurationVersionIDs []string
		expectPageInfo                pagination.PageInfo
		expectStartCursorError        error
		expectEndCursorError          error
		expectHasStartCursor          bool
		expectHasEndCursor            bool
		}
	*/

	testCases := []testCase{
		// nil input causes a nil pointer dereference in GetConfigurationVersions, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetConfigurationVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDs,
			expectPageInfo:                pagination.PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                pagination.PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtDesc),
			},
			expectConfigurationVersionIDs: reverseConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                pagination.PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "pagination: everything at once",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                pagination.PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "pagination: first two",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allConfigurationVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:    true,
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allConfigurationVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:    true,
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allConfigurationVersionIDs)),
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
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectConfigurationVersionIDs: reverseConfigurationVersionIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allConfigurationVersionIDs)),
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
			input: &GetConfigurationVersionsInput{
				Sort:              ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:    true,
			getBeforeCursorFromPrevious:   true,
			expectMsg:                     ptr.String("only before or after can be defined, not both"),
			expectConfigurationVersionIDs: []string{},
			expectPageInfo:                pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allConfigurationVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// The configuration version filter has only ConfigurationVersionIDs,
		// so there's no way to allow nothing through the filters.
		// Passing an empty slice to ConfigurationVersionIDs causes an SQL syntax error ("... IN ()"), so don't try it.

		{
			name: "filter, configuration version IDs",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				Filter: &ConfigurationVersionFilter{
					ConfigurationVersionIDs: []string{
						allConfigurationVersionIDsByUpdateTime[0],
						allConfigurationVersionIDsByUpdateTime[2],
						allConfigurationVersionIDsByUpdateTime[4],
					},
				},
			},
			expectConfigurationVersionIDs: []string{
				allConfigurationVersionIDsByUpdateTime[0],
				allConfigurationVersionIDsByUpdateTime[2],
				allConfigurationVersionIDsByUpdateTime[4],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, configuration version IDs, non-existent",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				Filter: &ConfigurationVersionFilter{
					ConfigurationVersionIDs: []string{nonExistentID},
				},
			},
			expectConfigurationVersionIDs: []string{},
			expectPageInfo:                pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "filter, configuration version IDs, invalid",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				Filter: &ConfigurationVersionFilter{
					ConfigurationVersionIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, positive",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				Filter: &ConfigurationVersionFilter{
					WorkspaceID: &warmupWorkspaces[1].Metadata.ID, // select odd-index configuration versions
				},
			},
			expectConfigurationVersionIDs: []string{
				allConfigurationVersionIDsByUpdateTime[1],
				allConfigurationVersionIDsByUpdateTime[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
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

			configurationVersionsActual, err := testClient.client.ConfigurationVersions.GetConfigurationVersions(ctx,
				test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, configurationVersionsActual.PageInfo)
				assert.NotNil(t, configurationVersionsActual.ConfigurationVersions)
				pageInfo := configurationVersionsActual.PageInfo
				configurationVersions := configurationVersionsActual.ConfigurationVersions

				// Check the configuration versions result by comparing a list of the configuration version IDs.
				actualConfigurationVersionIDs := []string{}
				for _, configurationVersion := range configurationVersions {
					actualConfigurationVersionIDs = append(actualConfigurationVersionIDs, configurationVersion.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualConfigurationVersionIDs)
				}

				assert.Equal(t, len(test.expectConfigurationVersionIDs), len(actualConfigurationVersionIDs))
				assert.Equal(t, test.expectConfigurationVersionIDs, actualConfigurationVersionIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one configuration version returned.
				// If there are no configuration versions returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(configurationVersions) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&configurationVersions[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&configurationVersions[len(configurationVersions)-1])
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

func TestCreateConfigurationVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, _, err := createWarmupConfigurationVersions(ctx, testClient,
		standardWarmupGroupsForConfigurationVersions,
		standardWarmupWorkspacesForConfigurationVersions,
		standardWarmupConfigurationVersions)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toCreate      *models.ConfigurationVersion
		expectCreated *models.ConfigurationVersion
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.ConfigurationVersion{
				WorkspaceID: warmupWorkspaceID,
			},
			expectCreated: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.ConfigurationVersion{
				Status:      models.ConfigurationPending,
				Speculative: true,
				WorkspaceID: warmupWorkspaceID,
				CreatedBy:   "tccv-pf",
			},
			expectCreated: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Status:      models.ConfigurationPending,
				Speculative: true,
				WorkspaceID: warmupWorkspaceID,
				CreatedBy:   "tccv-pf",
			},
		},

		// It does not make sense to try to create a duplicate configuration version,
		// because there is no unique name field to trigger an error.

		{
			name: "non-existent workspace ID",
			toCreate: &models.ConfigurationVersion{
				WorkspaceID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"configuration_versions\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.ConfigurationVersion{
				WorkspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, *test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareConfigurationVersions(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateConfigurationVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a configuration version with a specific ID without going into the really
	// low-level stuff, create the warmup configuration version(s) and then find the relevant ID.
	createdLow := currentTime()
	warmupWorkspaces, warmupConfigurationVersions, err := createWarmupConfigurationVersions(ctx, testClient,
		standardWarmupGroupsForConfigurationVersions,
		standardWarmupWorkspacesForConfigurationVersions,
		standardWarmupConfigurationVersions)
	require.Nil(t, err)
	createdHigh := currentTime()
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toUpdate                   *models.ConfigurationVersion
		expectConfigurationVersion *models.ConfigurationVersion
		expectMsg                  *string
		name                       string
	}

	// Do only one positive test case, because the logic is theoretically the same for all configuration versions.
	now := currentTime()
	positiveConfigurationVersion := warmupConfigurationVersions[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:      positiveConfigurationVersion.Metadata.ID,
					Version: positiveConfigurationVersion.Metadata.Version,
				},
				Status:      models.ConfigurationUploaded,
				Speculative: true,
				WorkspaceID: warmupWorkspaceID,
				// Cannot update CreatedBy.
			},
			expectConfigurationVersion: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:                   positiveConfigurationVersion.Metadata.ID,
					Version:              positiveConfigurationVersion.Metadata.Version + 1,
					CreationTimestamp:    positiveConfigurationVersion.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Status:      models.ConfigurationUploaded,
				Speculative: true,
				WorkspaceID: warmupWorkspaceID,
				CreatedBy:   "standard warmup configuration version 0",
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveConfigurationVersion.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
		{
			name: "defective-id",
			toUpdate: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveConfigurationVersion.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			configurationVersion, err := testClient.client.ConfigurationVersions.UpdateConfigurationVersion(ctx,
				*test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectConfigurationVersion != nil {
				require.NotNil(t, configurationVersion)
				now := currentTime()
				compareConfigurationVersions(t, test.expectConfigurationVersion, configurationVersion, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectConfigurationVersion.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, configurationVersion)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForConfigurationVersions = []models.Group{
	{
		Description: "top level group 0 for testing configuration version functions",
		FullPath:    "top-level-group-0-for-configuration-versions",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForConfigurationVersions = []models.Workspace{
	{
		Description: "workspace 0 for testing configuration version functions",
		FullPath:    "top-level-group-0-for-configuration-versions/workspace-0-for-configuration-versions",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing configuration version functions",
		FullPath:    "top-level-group-0-for-configuration-versions/workspace-1-for-configuration-versions",
		CreatedBy:   "someone-w1",
	},
}

// Standard warmup configuration version(s) for tests in this module:
var standardWarmupConfigurationVersions = []models.ConfigurationVersion{
	{
		CreatedBy: "standard warmup configuration version 0",
	},
	{
		CreatedBy: "standard warmup configuration version 1",
	},
	{
		CreatedBy: "standard warmup configuration version 2",
	},
	{
		CreatedBy: "standard warmup configuration version 3",
	},
	{
		CreatedBy: "standard warmup configuration version 4",
	},
}

// createWarmupConfigurationVersions creates some warmup configuration versions for a test
// The warmup configuration versions to create can be standard or otherwise.
func createWarmupConfigurationVersions(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newConfigurationVersions []models.ConfigurationVersion) (
	[]models.Workspace,
	[]models.ConfigurationVersion,
	error,
) {
	// It is necessary to create at least one group and workspace in order to provide the necessary IDs
	// for the configuration versions.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, err
	}

	// Set workspace IDs, alternating between workspace 0 and workspace 1.
	for ix := range newConfigurationVersions {
		newConfigurationVersions[ix].WorkspaceID = resultWorkspaces[ix&1].Metadata.ID
	}

	resultConfigurationVersions, err := createInitialConfigurationVersions(ctx, testClient,
		newConfigurationVersions)
	if err != nil {
		return nil, nil, err
	}

	return resultWorkspaces, resultConfigurationVersions, nil
}

func ptrConfigurationVersionSortableField(arg ConfigurationVersionSortableField) *ConfigurationVersionSortableField {
	return &arg
}

func (wis configurationVersionInfoIDSlice) Len() int {
	return len(wis)
}

func (wis configurationVersionInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis configurationVersionInfoIDSlice) Less(i, j int) bool {
	return wis[i].configurationVersionID < wis[j].configurationVersionID
}

func (wis configurationVersionInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis configurationVersionInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis configurationVersionInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// configurationVersionInfoFromConfigurationVersions returns a slice of configurationVersionInfo,
// not necessarily sorted in any order.
func configurationVersionInfoFromConfigurationVersions(
	configurationVersions []models.ConfigurationVersion,
) []configurationVersionInfo {
	result := []configurationVersionInfo{}

	for _, configurationVersion := range configurationVersions {
		result = append(result, configurationVersionInfo{
			configurationVersionID: configurationVersion.Metadata.ID,
			updateTime:             *configurationVersion.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// configurationVersionIDsFromConfigurationVersionInfos preserves order
func configurationVersionIDsFromConfigurationVersionInfos(
	configurationVersionInfos []configurationVersionInfo,
) []string {
	result := []string{}
	for _, configurationVersionInfo := range configurationVersionInfos {
		result = append(result, configurationVersionInfo.configurationVersionID)
	}
	return result
}

// compareConfigurationVersions compares two configuration version objects,
// including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareConfigurationVersions(t *testing.T, expected, actual *models.ConfigurationVersion,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Speculative, actual.Speculative)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
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
