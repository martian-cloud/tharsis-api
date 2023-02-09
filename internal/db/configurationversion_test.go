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

func TestGetConfigurationVersion(t *testing.T) {

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
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup configuration versions weren't all created.
		return
	}
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
			name:      "negative, non-existent ID",
			searchID:  nonExistentID,
			expectMsg: ptr.String("no rows in result set"),
			// expect configuration version to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			configurationVersion, err := testClient.client.ConfigurationVersions.GetConfigurationVersion(ctx,
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

func TestGetConfigurationVersions(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupConfigurationVersions, err := createWarmupConfigurationVersions(ctx, testClient,
		standardWarmupGroupsForConfigurationVersions,
		standardWarmupWorkspacesForConfigurationVersions,
		standardWarmupConfigurationVersions)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup configuration versions weren't all created.
		return
	}
	allConfigurationVersionInfos := configurationVersionInfoFromConfigurationVersions(warmupConfigurationVersions)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(configurationVersionInfoIDSlice(allConfigurationVersionInfos))
	allConfigurationVersionIDs := configurationVersionIDsFromConfigurationVersionInfos(allConfigurationVersionInfos)

	// Sort by last update times.
	sort.Sort(configurationVersionInfoUpdateSlice(allConfigurationVersionInfos))
	allConfigurationVersionIDsByUpdateTime := configurationVersionIDsFromConfigurationVersionInfos(allConfigurationVersionInfos)
	reverseConfigurationVersionIDsByUpdateTime := reverseStringSlice(allConfigurationVersionIDsByUpdateTime)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError        error
		expectEndCursorError          error
		expectMsg                     *string
		input                         *GetConfigurationVersionsInput
		name                          string
		expectPageInfo                PageInfo
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
		expectPageInfo                PageInfo
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
			expectPageInfo:                PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtDesc),
			},
			expectConfigurationVersionIDs: reverseConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "pagination: everything at once",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime,
			expectPageInfo:                PageInfo{TotalCount: int32(len(allConfigurationVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:          true,
			expectHasEndCursor:            true,
		},

		{
			name: "pagination: first two",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[:2],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious:    true,
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[2:4],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious:    true,
			expectConfigurationVersionIDs: allConfigurationVersionIDsByUpdateTime[4:],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectConfigurationVersionIDs: reverseConfigurationVersionIDsByUpdateTime[:3],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:    true,
			getBeforeCursorFromPrevious:   true,
			expectMsg:                     ptr.String("only before or after can be defined, not both"),
			expectConfigurationVersionIDs: []string{},
			expectPageInfo:                PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetConfigurationVersionsInput{
				Sort: ptrConfigurationVersionSortableField(ConfigurationVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: PageInfo{
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
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
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
			expectPageInfo:                PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup configuration versions weren't all created.
		return
	}
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
			expectMsg: invalidUUIDMsg1,
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
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup configuration versions weren't all created.
		return
	}
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
			expectMsg: invalidUUIDMsg4,
		},
		{
			name: "defective-id",
			toUpdate: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveConfigurationVersion.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg4,
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
	error) {

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
	workspaceID := resultWorkspaces[0].Metadata.ID

	resultConfigurationVersions, err := createInitialConfigurationVersions(ctx, testClient,
		newConfigurationVersions, workspaceID)
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
	configurationVersions []models.ConfigurationVersion) []configurationVersionInfo {
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
	configurationVersionInfos []configurationVersionInfo) []string {
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
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Speculative, actual.Speculative)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
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
