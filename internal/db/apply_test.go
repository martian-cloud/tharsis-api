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

// applyInfo aids convenience in accessing the information TestGetApplies needs about the warmup applies.
type applyInfo struct {
	updateTime time.Time
	applyID    string
}

// applyInfoIDSlice makes a slice of applyInfo sortable by ID string
type applyInfoIDSlice []applyInfo

// applyInfoUpdateSlice makes a slice of applyInfo sortable by last updated time
type applyInfoUpdateSlice []applyInfo

func TestGetApplyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create an apply with a specific ID without going into the really
	// low-level stuff, create the warmup apply/applies then find the relevant ID.
	createdLow := currentTime()
	_, createdWarmupApplies, err := createWarmupApplies(ctx, testClient, standardWarmupGroupsForApplies,
		standardWarmupWorkspacesForApplies, standardWarmupApplies)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectApply *models.Apply
		expectMsg   *string
		name        string
		searchID    string
	}

	// Do only one positive test case, because the logic is theoretically the same for all applies.
	positiveApply := createdWarmupApplies[0]
	testCases := []testCase{
		{
			name:        "positive",
			searchID:    positiveApply.Metadata.ID,
			expectApply: &positiveApply,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect apply and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			apply, err := testClient.client.Applies.GetApplyByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectApply != nil {
				require.NotNil(t, apply)
				compareApplies(t, test.expectApply, apply, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, apply)
			}
		})
	}
}

func TestGetApplyByTRN(t *testing.T) {
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

	apply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
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
			trn:         apply.Metadata.TRN,
			expectApply: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.ApplyModelType.BuildTRN(workspace.FullPath, nonExistentGlobalID),
		},
		{
			name:            "apply trn cannot have than two parts",
			trn:             types.ApplyModelType.BuildTRN(nonExistentGlobalID),
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
			actualApply, err := testClient.client.Applies.GetApplyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectApply {
				require.NotNil(t, actualApply)
				assert.Equal(t, types.ApplyModelType.BuildTRN(workspace.FullPath, apply.GetGlobalID()), actualApply.Metadata.TRN)
			} else {
				assert.Nil(t, actualApply)
			}
		})
	}
}

func TestGetApplies(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupApplies, err := createWarmupApplies(ctx, testClient,
		standardWarmupGroupsForApplies, standardWarmupWorkspacesForApplies, standardWarmupApplies)
	require.Nil(t, err)
	allApplyInfos := applyInfoFromApplies(warmupApplies)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(applyInfoIDSlice(allApplyInfos))
	allApplyIDs := applyIDsFromApplyInfos(allApplyInfos)

	// Sort by last update times.
	sort.Sort(applyInfoUpdateSlice(allApplyInfos))
	allApplyIDsByUpdateTime := applyIDsFromApplyInfos(allApplyInfos)
	reverseApplyIDsByUpdateTime := reverseStringSlice(allApplyIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetAppliesInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectApplyIDs              []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetAppliesInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectApplyIDs              []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input causes a nil pointer dereference in GetApplies, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetAppliesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectApplyIDs:       allApplyIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allApplyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectApplyIDs:       allApplyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allApplyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtDesc),
			},
			expectApplyIDs:       reverseApplyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allApplyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectApplyIDs:       allApplyIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allApplyIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectApplyIDs: allApplyIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allApplyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectApplyIDs:             allApplyIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allApplyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectApplyIDs:             allApplyIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allApplyIDs)),
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
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectApplyIDs: reverseApplyIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allApplyIDs)),
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
			input: &GetAppliesInput{
				Sort:              ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectApplyIDs:              []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allApplyIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// The apply filter has only ApplyIDs, so there's no way to allow nothing through the filters.
		// Passing an empty slice to ApplyIDs causes an SQL syntax error ("... IN ()"), so don't try it.

		{
			name: "filter, apply IDs",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				Filter: &ApplyFilter{
					ApplyIDs: []string{allApplyIDsByUpdateTime[0], allApplyIDsByUpdateTime[2], allApplyIDsByUpdateTime[4]},
				},
			},
			expectApplyIDs:       []string{allApplyIDsByUpdateTime[0], allApplyIDsByUpdateTime[2], allApplyIDsByUpdateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, apply IDs, non-existent",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				Filter: &ApplyFilter{
					ApplyIDs: []string{nonExistentID},
				},
			},
			expectApplyIDs:       []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, apply IDs, invalid",
			input: &GetAppliesInput{
				Sort: ptrApplySortableField(ApplySortableFieldUpdatedAtAsc),
				Filter: &ApplyFilter{
					ApplyIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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

			appliesActual, err := testClient.client.Applies.GetApplies(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, appliesActual.PageInfo)
				assert.NotNil(t, appliesActual.Applies)
				pageInfo := appliesActual.PageInfo
				applies := appliesActual.Applies

				// Check the applies result by comparing a list of the apply IDs.
				actualApplyIDs := []string{}
				for _, apply := range applies {
					actualApplyIDs = append(actualApplyIDs, apply.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualApplyIDs)
				}

				assert.Equal(t, len(test.expectApplyIDs), len(actualApplyIDs))
				assert.Equal(t, test.expectApplyIDs, actualApplyIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one apply returned.
				// If there are no applies returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(applies) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&applies[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&applies[len(applies)-1])
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

func TestCreateApply(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, _, err := createWarmupApplies(ctx, testClient,
		standardWarmupGroupsForApplies, standardWarmupWorkspacesForApplies, standardWarmupApplies)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toCreate      *models.Apply
		expectCreated *models.Apply
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.Apply{
				WorkspaceID: warmupWorkspaceID,
			},
			expectCreated: &models.Apply{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.Apply{
				WorkspaceID: warmupWorkspaceID,
				Status:      models.ApplyFinished,
				TriggeredBy: "tca-pf",
				Comment:     "TestCreateApply, positive full",
			},
			expectCreated: &models.Apply{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.ApplyFinished,
				TriggeredBy: "tca-pf",
				Comment:     "TestCreateApply, positive full",
			},
		},

		// It does not make sense to try to create a duplicate apply,
		// because there is no unique name field to trigger an error.

		{
			name: "non-existent workspace ID",
			toCreate: &models.Apply{
				WorkspaceID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"applies\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.Apply{
				WorkspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.Applies.CreateApply(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareApplies(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateApply(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create an apply with a specific ID without going into the really
	// low-level stuff, create the warmup applies and then find the relevant ID.
	createdLow := currentTime()
	warmupWorkspaces, warmupApplies, err := createWarmupApplies(ctx, testClient,
		standardWarmupGroupsForApplies, standardWarmupWorkspacesForApplies, standardWarmupApplies)
	require.Nil(t, err)
	createdHigh := currentTime()
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toUpdate    *models.Apply
		expectApply *models.Apply
		expectMsg   *string
		name        string
	}

	// Do only one positive test case, because the logic is theoretically the same for all applies.
	now := currentTime()
	positiveApply := warmupApplies[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.Apply{
				Metadata: models.ResourceMetadata{
					ID:      positiveApply.Metadata.ID,
					Version: positiveApply.Metadata.Version,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.ApplyFinished,
				TriggeredBy: "tua-p",
				Comment:     "TestUpdateApply, positive",
			},
			expectApply: &models.Apply{
				Metadata: models.ResourceMetadata{
					ID:                   positiveApply.Metadata.ID,
					Version:              positiveApply.Metadata.Version + 1,
					CreationTimestamp:    positiveApply.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.ApplyFinished,
				TriggeredBy: "tua-p",
				Comment:     "TestUpdateApply, positive",
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.Apply{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveApply.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.Apply{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveApply.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			apply, err := testClient.client.Applies.UpdateApply(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectApply != nil {
				require.NotNil(t, apply)
				now := currentTime()
				compareApplies(t, test.expectApply, apply, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectApply.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, apply)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForApplies = []models.Group{
	{
		Description: "top level group 0 for testing apply functions",
		FullPath:    "top-level-group-0-for-applies",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForApplies = []models.Workspace{
	{
		Description: "workspace 0 for testing apply functions",
		FullPath:    "top-level-group-0-for-applies/workspace-0-for-applies",
		CreatedBy:   "someone-w0",
	},
}

// Standard warmup applies for tests in this module:
var standardWarmupApplies = []models.Apply{
	{
		Comment: "standard warmup apply 0",
	},
	{
		Comment: "standard warmup apply 1",
	},
	{
		Comment: "standard warmup apply 2",
	},
	{
		Comment: "standard warmup apply 3",
	},
	{
		Comment: "standard warmup apply 4",
	},
}

// createWarmupApplies creates some warmup applies for a test
// The warmup applies to create can be standard or otherwise.
func createWarmupApplies(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newApplies []models.Apply) (
	[]models.Workspace,
	[]models.Apply,
	error,
) {
	// It is necessary to create at least one group and workspace in order to provide the necessary IDs for the applies.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, err
	}
	workspaceID := resultWorkspaces[0].Metadata.ID

	resultApplies, err := createInitialApplies(ctx, testClient, newApplies, workspaceID)
	if err != nil {
		return nil, nil, err
	}

	return resultWorkspaces, resultApplies, nil
}

func ptrApplySortableField(arg ApplySortableField) *ApplySortableField {
	return &arg
}

func (wis applyInfoIDSlice) Len() int {
	return len(wis)
}

func (wis applyInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis applyInfoIDSlice) Less(i, j int) bool {
	return wis[i].applyID < wis[j].applyID
}

func (wis applyInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis applyInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis applyInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// applyInfoFromApplies returns a slice of applyInfo, not necessarily sorted in any order.
func applyInfoFromApplies(applies []models.Apply) []applyInfo {
	result := []applyInfo{}

	for _, apply := range applies {
		result = append(result, applyInfo{
			applyID:    apply.Metadata.ID,
			updateTime: *apply.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// applyIDsFromApplyInfos preserves order
func applyIDsFromApplyInfos(applyInfos []applyInfo) []string {
	result := []string{}
	for _, applyInfo := range applyInfos {
		result = append(result, applyInfo.applyID)
	}
	return result
}

// compareApplies compares two apply objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareApplies(t *testing.T, expected, actual *models.Apply,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.TriggeredBy, actual.TriggeredBy)
	assert.Equal(t, expected.Comment, actual.Comment)

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
