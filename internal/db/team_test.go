//go:build integration

package db

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// teamInfo aids convenience in accessing the information TestGetTeams
// needs about the warmup teams.
type teamInfo struct {
	createTime time.Time
	updateTime time.Time
	teamID     string
	name       string
}

// teamInfoIDSlice makes a slice of teamInfo sortable by ID string
type teamInfoIDSlice []teamInfo

// teamInfoUpdateSlice makes a slice of teamInfo sortable by last updated time
type teamInfoUpdateSlice []teamInfo

// teamInfoNameSlice makes a slice of teamInfo sortable by name
type teamInfoNameSlice []teamInfo

func TestGetTeamByID(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectTeam      bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:       "get team by ID",
			id:         team.Metadata.ID,
			expectTeam: true,
		},
		{
			name: "resource with ID not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid ID will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTeam, err := testClient.client.Teams.GetTeamByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectTeam {
				require.NotNil(t, actualTeam)
				assert.Equal(t, test.id, actualTeam.Metadata.ID)
			} else {
				assert.Nil(t, actualTeam)
			}
		})
	}
}

func TestGetTeamByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectTeam      bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:       "get team by TRN",
			trn:        team.Metadata.TRN,
			expectTeam: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TeamModelType.BuildTRN("unknown"),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTeam, err := testClient.client.Teams.GetTeamByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectTeam {
				require.NotNil(t, actualTeam)
				assert.Equal(t, test.trn, actualTeam.Metadata.TRN)
			} else {
				assert.Nil(t, actualTeam)
			}
		})
	}
}

func TestCreateTeams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		input         *models.Team
		expectMsg     *string
		expectCreated *models.Team
		name          string
	}

	/*
		template test case:

		{
			name          string
			input         *models.Team
			expectMsg     *string
			expectCreated *models.Team
		}
	*/

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive: create a simple team object",
			input: &models.Team{
				Name:        "simpleTeamObject-1",
				Description: "This is the first simple team object.",
			},
			expectCreated: &models.Team{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:        "simpleTeamObject-1",
				Description: "This is the first simple team object.",
			},
		},

		{
			name: "negative: duplicate",
			input: &models.Team{
				Name:        "simpleTeamObject-1",
				Description: "This would be a duplicate team object.",
			},
			expectMsg: ptr.String("team with name simpleTeamObject-1 already exists"),
		},

		// Negative test case for missing prerequisite is not applicable.

		// Currently, the DB layer does not check for a valid name: only alphanumeric, etc.
		// Also, there are no applicable negative tests for invalid UUID format, etc.

	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
			actualCreated, err := testClient.client.Teams.CreateTeam(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareTeams(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestGetTeamBySCIMExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupTeams, _, err := createInitialTeams(ctx, testClient, standardWarmupTeams)
	require.Nil(t, err)

	type testCase struct {
		expectMsg  *string
		name       string
		searchID   string
		expectTeam bool
	}

	testCases := []testCase{}
	for _, positiveTeam := range createdWarmupTeams {
		testCases = append(testCases, testCase{
			name:       "positive-" + positiveTeam.Name,
			searchID:   positiveTeam.SCIMExternalID,
			expectTeam: true,
		})
	}

	testCases = append(testCases,
		testCase{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect team and error to be nil
		},
		testCase{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			team, err := testClient.client.Teams.GetTeamBySCIMExternalID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTeam {
				// the positive case
				require.NotNil(t, team)
				require.NotNil(t, team.SCIMExternalID)
				assert.Equal(t, test.searchID, team.SCIMExternalID)
			} else {
				// the negative and defective cases
				assert.Nil(t, team)
			}
		})
	}
}

func TestGetTeams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupTeams, _, err := createInitialTeams(ctx, testClient, standardWarmupTeams)
	require.Nil(t, err)

	allTeamInfos := teamInfoFromTeams(createdWarmupTeams)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(teamInfoIDSlice(allTeamInfos))
	allTeamIDs := teamIDsFromTeamInfos(allTeamInfos)

	// Sort by last update times.
	sort.Sort(teamInfoUpdateSlice(allTeamInfos))
	allTeamIDsByUpdateTime := teamIDsFromTeamInfos(allTeamInfos)
	reverseTeamIDsByUpdateTime := reverseStringSlice(allTeamIDsByUpdateTime)

	// Sort by names.
	sort.Sort(teamInfoNameSlice(allTeamInfos))
	allTeamIDsByName := teamIDsFromTeamInfos(allTeamInfos)
	reverseTeamIDsByName := reverseStringSlice(allTeamIDsByName)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetTeamsInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectTeamIDs               []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	testCases := []testCase{
		{
			name: "non-nil but mostly empty input",
			input: &GetTeamsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTeamIDs:        allTeamIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of name, nil filter",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTeamIDs:        allTeamIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of name",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameDesc),
			},
			expectTeamIDs:        reverseTeamIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTeamIDs:        allTeamIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtDesc),
			},
			expectTeamIDs:        reverseTeamIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectTeamIDs:        allTeamIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectTeamIDs: allTeamIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTeamIDs:              allTeamIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTeamIDs:              allTeamIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
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
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectTeamIDs: reverseTeamIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination, before and after, expect error",
			input: &GetTeamsInput{
				Sort:              ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectTeamIDs:               []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
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
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String(""),
				},
			},
			expectTeamIDs: allTeamIDsByName,
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allTeamIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, empty string",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String(""),
				},
			},
			expectTeamIDs:        allTeamIDsByName,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allTeamIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 1",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String("1"),
				},
			},
			expectTeamIDs:        allTeamIDsByName[1:2],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 2",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String("2"),
				},
			},
			expectTeamIDs:        allTeamIDsByName[2:3],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 3",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String("3"),
				},
			},
			expectTeamIDs:        allTeamIDsByName[3:4],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 4",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String("4"),
				},
			},
			expectTeamIDs:        allTeamIDsByName[4:],
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamNamePrefix: ptr.String("bogus"),
				},
			},
			expectTeamIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, team IDs, positive",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamIDs: []string{
						allTeamIDsByName[0], allTeamIDsByName[1], allTeamIDsByName[3]},
				},
			},
			expectTeamIDs: []string{
				allTeamIDsByName[0], allTeamIDsByName[1], allTeamIDsByName[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, team IDs, non-existent",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamIDs: []string{nonExistentID},
				},
			},
			expectTeamIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, team IDs, invalid ID",
			input: &GetTeamsInput{
				Sort: ptrTeamSortableField(TeamSortableFieldNameAsc),
				Filter: &TeamFilter{
					TeamIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectTeamIDs:        []string{},
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

			teamsActual, err := testClient.client.Teams.GetTeams(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, teamsActual.PageInfo)
				assert.NotNil(t, teamsActual.Teams)
				pageInfo := teamsActual.PageInfo
				teams := teamsActual.Teams

				// Check the teams result by comparing a list of the team IDs.
				actualTeamIDs := []string{}
				for _, team := range teams {
					actualTeamIDs = append(actualTeamIDs, team.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTeamIDs)
				}

				assert.Equal(t, len(test.expectTeamIDs), len(actualTeamIDs))
				assert.Equal(t, test.expectTeamIDs, actualTeamIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one team returned.
				// If there are no teams returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(teams) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&teams[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&teams[len(teams)-1])
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

func TestUpdateTeams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupTeams, _, err := createInitialTeams(ctx, testClient, standardWarmupTeams)
	require.Nil(t, err)

	type testCase struct {
		input         *models.Team
		expectMsg     *string
		expectUpdated *models.Team
		name          string
	}

	/*
		template test case:

		{
		name        string
		input       *models.Team
		expectMsg   *string
		expectTeams []string
		}
	*/

	testCases := []testCase{}

	for _, preUpdate := range createdWarmupTeams {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive-" + preUpdate.Name,
			input: &models.Team{
				Metadata: models.ResourceMetadata{
					ID:      preUpdate.Metadata.ID,
					Version: preUpdate.Metadata.Version,
				},
				Name:        preUpdate.Name,
				Description: "updated-description: " + preUpdate.Description,
			},
			expectUpdated: &models.Team{
				Metadata: models.ResourceMetadata{
					ID:                   preUpdate.Metadata.ID,
					Version:              preUpdate.Metadata.Version + 1,
					CreationTimestamp:    preUpdate.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Name:        preUpdate.Name,
				Description: "updated-description: " + preUpdate.Description,
			},
		})
	}

	testCases = append(testCases, testCase{
		name: "negative, non-exist",
		input: &models.Team{
			Metadata: models.ResourceMetadata{
				ID:      nonExistentID,
				Version: 1,
			},
		},
		expectMsg: resourceVersionMismatch,
	},

	// No invalid test case is applicable.

	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualUpdated, err := testClient.client.Teams.UpdateTeam(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				compareTeams(t, test.expectUpdated, actualUpdated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualUpdated)
			}
		})
	}
}

func TestDeleteTeams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupTeams, _, err := createInitialTeams(ctx, testClient, standardWarmupTeams)
	require.Nil(t, err)

	warmupNames := []string{}
	for _, warmupTeam := range createdWarmupTeams {
		warmupNames = append(warmupNames, warmupTeam.Name)
	}

	type testCase struct {
		name            string
		input           []models.Team
		expectMsg       *string
		expectTeamNames []string // names of teams left after the delete operation
	}

	/*
		template test case:

		{
		name        string
		input       []models.Team
		expectMsg   *string
		expectTeamNames []string
		}
	*/

	// Because delete is destructive, start with negative cases and end with one
	// positive case per warmup group.  Alternatively, the warmup teams could be
	// created fresh for each test case.
	testCases := []testCase{
		{
			name: "negative: non-exist",
			input: []models.Team{
				{
					Metadata: models.ResourceMetadata{
						ID:      nonExistentID,
						Version: initialResourceVersion,
					},
					Name: "this-team-does-not-exist",
				},
			},
			expectMsg:       resourceVersionMismatch,
			expectTeamNames: warmupNames,
		},

		// No invalid test case is applicable.
	}

	// Positive case, one warmup team at a time.
	for ix, toDelete := range createdWarmupTeams {
		testCases = append(testCases, testCase{
			name:            "positive-" + toDelete.Name,
			input:           []models.Team{toDelete},
			expectTeamNames: copyStringSlice(warmupNames[ix+1:]), // sort-in-place was corrupting later test cases
		})
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Now, try to delete the teams in sequence.
			for _, teamToDelete := range test.input {

				err = testClient.client.Teams.DeleteTeam(ctx, &teamToDelete)

				checkError(t, test.expectMsg, err)

				// Get the names of the actual teams that remain.
				gotResult, err := testClient.client.Teams.GetTeams(ctx, &GetTeamsInput{
					Sort:              nil,
					PaginationOptions: nil,
					Filter:            nil,
				})
				require.Nil(t, err)

				actualNames := []string{}
				for _, gotTeam := range gotResult.Teams {
					actualNames = append(actualNames, gotTeam.Name)
				}

				// The sorting in place here is what makes the copying of the string slice necessary.
				sort.Strings(test.expectTeamNames)
				sort.Strings(actualNames)

				// Make sure the expected names remain.
				assert.Equal(t, test.expectTeamNames, actualNames)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup teams for tests in this module:
var standardWarmupTeams = []models.Team{
	{
		Name:           "0-team-0",
		Description:    "team 0 for testing teams",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "1-team-1",
		Description:    "team 1 for testing teams",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "2-team-2",
		Description:    "team 2 for testing teams",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "3-team-3",
		Description:    "team 3 for testing teams",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "4-team-4",
		Description:    "team 4 for testing teams",
		SCIMExternalID: uuid.New().String(),
	},
}

// createWarmupTeams creates some objects for a test
// The objects to create can be standard or otherwise.
func createWarmupTeams(ctx context.Context, testClient *testClient,
	input []models.Team,
) ([]models.Team, error) {
	resultTeams, _, err := createInitialTeams(ctx, testClient, input)
	if err != nil {
		return nil, err
	}

	return resultTeams, nil
}

func ptrTeamSortableField(arg TeamSortableField) *TeamSortableField {
	return &arg
}

func (t teamInfoIDSlice) Len() int {
	return len(t)
}

func (t teamInfoIDSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t teamInfoIDSlice) Less(i, j int) bool {
	return t[i].teamID < t[j].teamID
}

func (t teamInfoUpdateSlice) Len() int {
	return len(t)
}

func (t teamInfoUpdateSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t teamInfoUpdateSlice) Less(i, j int) bool {
	return t[i].updateTime.Before(t[j].updateTime)
}

func (t teamInfoNameSlice) Len() int {
	return len(t)
}

func (t teamInfoNameSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t teamInfoNameSlice) Less(i, j int) bool {
	return t[i].name < t[j].name
}

// teamInfoFromTeams returns a slice of teamInfo, not necessarily sorted in any order.
func teamInfoFromTeams(teams []models.Team) []teamInfo {
	result := []teamInfo{}

	for _, team := range teams {
		result = append(result, teamInfo{
			createTime: *team.Metadata.CreationTimestamp,
			updateTime: *team.Metadata.LastUpdatedTimestamp,
			teamID:     team.Metadata.ID,
			name:       team.Name,
		})
	}

	return result
}

// teamIDsFromTeamInfos preserves order
func teamIDsFromTeamInfos(teamInfos []teamInfo) []string {
	result := []string{}
	for _, teamInfo := range teamInfos {
		result = append(result, teamInfo.teamID)
	}

	return result
}

// compareTeams compares two team objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTeams(t *testing.T, expected, actual *models.Team,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)

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

// copyStringSlice makes a copy of a string slice.
func copyStringSlice(input []string) []string {
	result := []string{}
	result = append(result, input...)
	return result
}
