//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

type teamMemberWarmupsInput struct {
	users       []models.User
	teams       []models.Team
	teamMembers []models.TeamMember
}

type teamMemberWarmupsOutput struct {
	userIDs2Name map[string]string
	teamIDs2Name map[string]string
	users        []models.User
	teams        []models.Team
	teamMembers  []models.TeamMember
}

// teamMemberNameSlice makes a slice of models.TeamMember sortable by name
type teamMemberNameSlice struct {
	warmupOutput *teamMemberWarmupsOutput
	teamMembers  []models.TeamMember
}

func TestGetTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupTeamMembers(ctx, testClient, teamMemberWarmupsInput{
		teams:       standardWarmupTeamsForTeamMembers,
		users:       standardWarmupUsersForTeamMembers,
		teamMembers: standardWarmupTeamMembers,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg        *string
		expectTeamMember *models.TeamMember
		name             string
		userID           string
		teamID           string
	}

	/*
		template test case:

		{
		name             string
		userID           string
		teamID           string
		expectMsg        *string
		expectTeamMember *models.TeamMember
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup team at a time.
	for _, toGet := range createdWarmupOutput.teamMembers {
		copyToGet := toGet
		testCases = append(testCases, testCase{
			name:             "positive--" + buildTeamMemberName(createdWarmupOutput, toGet),
			userID:           toGet.UserID,
			teamID:           toGet.TeamID,
			expectTeamMember: &copyToGet,
		})
	}
	toGet0 := createdWarmupOutput.teamMembers[0]

	testCases = append(testCases,
		testCase{
			name:   "negative: non-exist user ID",
			userID: nonExistentID,
			teamID: toGet0.TeamID,
		},
		testCase{
			name:   "negative: non-exist team ID",
			userID: toGet0.UserID,
			teamID: nonExistentID,
		},
		testCase{
			name:      "negative: invalid user ID",
			userID:    invalidID,
			teamID:    toGet0.TeamID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
		testCase{
			name:      "negative: invalid team ID",
			userID:    toGet0.UserID,
			teamID:    invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotTeamMember, err := testClient.client.TeamMembers.GetTeamMember(ctx, test.userID, test.teamID)

			checkError(t, test.expectMsg, err)

			if test.expectTeamMember != nil {
				require.NotNil(t, gotTeamMember)
				compareTeamMembers(t, test.expectTeamMember, gotTeamMember, true, nil)
			} else {
				assert.Nil(t, gotTeamMember)
			}
		})
	}
}

func TestGetTeamMembers(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupTeamMembers(ctx, testClient, teamMemberWarmupsInput{
		teams:       standardWarmupTeamsForTeamMembers,
		users:       standardWarmupUsersForTeamMembers,
		teamMembers: standardWarmupTeamMembers,
	})
	require.Nil(t, err)

	type testCase struct {
		name              string
		input             *GetTeamMembersInput
		expectMsg         *string
		expectTeamMembers []models.TeamMember
	}

	/*
		template test case:

		{
		name              string
		input             *GetTeamMembersInput
		expectMsg         *string
		expectTeamMembers []models.TeamMember
		}
	*/

	// TODO: Add more cases:
	testCases := []testCase{
		{
			name: "simple get team members test case",
			input: &GetTeamMembersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTeamMembers: createdWarmupOutput.teamMembers,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// TODO: Add checks for pagination, cursors, etc.

			gotResult, err := testClient.client.TeamMembers.GetTeamMembers(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectTeamMembers != nil {
				actualTeamMembers := gotResult.TeamMembers
				require.NotNil(t, actualTeamMembers)
				require.Equal(t, len(test.expectTeamMembers), len(actualTeamMembers))

				// TODO: When implementing test cases that do sorting other than ascending order by name,
				// change this to handle all cases.

				// For now, sort both expected and actual by synthesized name in order to compare.
				sort.Sort(teamMemberNameSlice{
					warmupOutput: createdWarmupOutput,
					teamMembers:  test.expectTeamMembers,
				})
				sort.Sort(teamMemberNameSlice{
					warmupOutput: createdWarmupOutput,
					teamMembers:  actualTeamMembers,
				})

				// Compare the slices of teams, now that they should be sorted the same.
				for ix := 0; ix < len(test.expectTeamMembers); ix++ {
					compareTeamMembers(t, &test.expectTeamMembers[ix], &actualTeamMembers[ix], true, nil)
				}

			}

			// TODO: Add code for pagination, cursors, etc.
		})
	}
}

func TestAddUserToTeam(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create warmup teams and users but _NOT_ team members relationships.
	createdWarmupOutput, err := createWarmupTeamMembers(ctx, testClient, teamMemberWarmupsInput{
		teams:       standardWarmupTeamsForTeamMembers,
		users:       standardWarmupUsersForTeamMembers,
		teamMembers: []models.TeamMember{},
	})
	require.Nil(t, err)

	usernames2IDs := reverseMap(createdWarmupOutput.userIDs2Name)
	teamNames2IDs := reverseMap(createdWarmupOutput.teamIDs2Name)

	type testCase struct {
		input       *models.TeamMember
		expectMsg   *string
		expectAdded *models.TeamMember
		name        string
	}

	/*
		template test case:

		{
		name        string
		input       *models.TeamMember
		expectMsg   *string
		expectAdded *models.TeamMember
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup team member relationship at a time.
	for _, toAdd := range standardWarmupTeamMembers {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive: " + buildTeamMemberName(createdWarmupOutput, toAdd),
			input: &models.TeamMember{
				UserID:       usernames2IDs[toAdd.UserID],
				TeamID:       teamNames2IDs[toAdd.TeamID],
				IsMaintainer: toAdd.IsMaintainer,
			},
			expectAdded: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &now,
					Version:           initialResourceVersion,
				},
				UserID:       usernames2IDs[toAdd.UserID],
				TeamID:       teamNames2IDs[toAdd.TeamID],
				IsMaintainer: toAdd.IsMaintainer,
			},
		})
	}

	// Negative cases:
	input0 := standardWarmupTeamMembers[0]
	userID0 := usernames2IDs[input0.UserID]
	teamID0 := teamNames2IDs[input0.TeamID]
	testCases = append(testCases,

		testCase{
			name: "negative: duplicate",
			input: &models.TeamMember{
				UserID:       userID0,
				TeamID:       teamID0,
				IsMaintainer: input0.IsMaintainer,
			},
			expectMsg: ptr.String(fmt.Sprintf("team member of user %s in team %s already exists",
				input0.UserID, input0.TeamID)),
		},

		testCase{
			name: "negative: user ID does not exist",
			input: &models.TeamMember{
				UserID: nonExistentID,
				TeamID: teamID0,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"team_members\" violates foreign key constraint \"fk_team_members_user_id\" (SQLSTATE 23503)"),
		},

		testCase{
			name: "negative: team ID does not exist",
			input: &models.TeamMember{
				UserID: userID0,
				TeamID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"team_members\" violates foreign key constraint \"fk_team_members_team_id\" (SQLSTATE 23503)"),
		},

		testCase{
			name: "negative: invalid user ID",
			input: &models.TeamMember{
				UserID: invalidID,
				TeamID: teamID0,
			},
			expectMsg: invalidUUIDMsg,
		},

		testCase{
			name: "negative: invalid team ID",
			input: &models.TeamMember{
				UserID: userID0,
				TeamID: invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			claimedAdded, err := testClient.client.TeamMembers.AddUserToTeam(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectAdded != nil {
				require.NotNil(t, claimedAdded)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectAdded.Metadata.CreationTimestamp
				now := currentTime()

				compareTeamMembers(t, test.expectAdded, claimedAdded, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})

				// Verify that what the AddUserToTeam method claimed was added can fetched.
				fetched, err := testClient.client.TeamMembers.GetTeamMember(ctx, test.input.UserID, test.input.TeamID)
				assert.Nil(t, err)

				if test.expectAdded != nil {
					require.NotNil(t, fetched)
					compareTeamMembers(t, claimedAdded, fetched, true, nil)
				} else {
					assert.Nil(t, fetched)
				}
			} else {
				assert.Nil(t, claimedAdded)
			}
		})
	}
}

func TestUpdateTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupTeamMembers(ctx, testClient, teamMemberWarmupsInput{
		teams:       standardWarmupTeamsForTeamMembers,
		users:       standardWarmupUsersForTeamMembers,
		teamMembers: standardWarmupTeamMembers,
	})
	require.Nil(t, err)

	type testCase struct {
		input         *models.TeamMember
		expectMsg     *string
		expectUpdated *models.TeamMember
		name          string
	}

	/*
		template test case:

		{
		name        string
		input       *models.TeamMember
		expectMsg   *string
		expectUpdated *models.TeamMember
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup team member relationship at a time.
	// The only field that is modified is IsMaintainer.
	for _, toUpdate := range createdWarmupOutput.teamMembers {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive: " + buildTeamMemberName(createdWarmupOutput, toUpdate),
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      toUpdate.Metadata.ID,
					Version: toUpdate.Metadata.Version,
				},
				IsMaintainer: false,
			},
			expectUpdated: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    toUpdate.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				UserID:       toUpdate.UserID,
				TeamID:       toUpdate.TeamID,
				IsMaintainer: false,
			},
		})
	}

	// Negative cases:
	// Version number will have been incremented by the positive test cases.
	input0 := createdWarmupOutput.teamMembers[0]
	newVersion := input0.Metadata.Version + 1
	testCases = append(testCases,

		testCase{
			name: "negative: team member ID does not exist",
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: newVersion,
				},
				IsMaintainer: false,
			},
			expectMsg: resourceVersionMismatch,
		},

		testCase{
			name: "negative: invalid team member ID",
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: newVersion,
				},
				IsMaintainer: false,
			},
			expectMsg: invalidUUIDMsg,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualUpdated, err := testClient.client.TeamMembers.UpdateTeamMember(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				now := currentTime()

				compareTeamMembers(t, test.expectUpdated, actualUpdated, false, &timeBounds{
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

func TestRemoveUserFromTeam(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupTeamMembers(ctx, testClient, teamMemberWarmupsInput{
		teams:       standardWarmupTeamsForTeamMembers,
		users:       standardWarmupUsersForTeamMembers,
		teamMembers: standardWarmupTeamMembers,
	})
	require.Nil(t, err)

	warmupNames := []string{}
	for _, warmupTeamMember := range createdWarmupOutput.teamMembers {
		warmupNames = append(warmupNames, buildTeamMemberName(createdWarmupOutput, warmupTeamMember))
	}

	type testCase struct {
		name                  string
		input                 *models.TeamMember
		expectMsg             *string
		expectTeamMemberNames []string // names of teams left after the delete operation
	}

	/*
		template test case:

		{
		name                  string
		input                 *models.TeamMember
		expectMsg             *string
		expectTeamMemberNames []string // names of teams left after the delete operation
		}
	*/

	// Because delete is destructive, start with negative cases and end with one
	// positive case per warmup group.  Alternatively, the warmup team members could be
	// created fresh for each test case.
	testCases := []testCase{
		{
			name: "negative: team member ID does not exist",
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg:             resourceVersionMismatch,
			expectTeamMemberNames: warmupNames,
		},

		{
			name: "negative: invalid team member ID",
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg:             invalidUUIDMsg,
			expectTeamMemberNames: warmupNames,
		},
	}

	// Positive case, one warmup team member relationship at a time.
	for ix, toDelete := range createdWarmupOutput.teamMembers {
		testCases = append(testCases, testCase{
			name: "positive: " + buildTeamMemberName(createdWarmupOutput, toDelete),
			input: &models.TeamMember{
				Metadata: models.ResourceMetadata{
					ID:      toDelete.Metadata.ID,
					Version: toDelete.Metadata.Version,
				},
				IsMaintainer: false,
			},
			expectTeamMemberNames: copyStringSlice(warmupNames[ix+1:]), // sort-in-place was likely to corrupt later test cases
		})
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TeamMembers.RemoveUserFromTeam(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// Get the names of the actual team members that remain.
			gotResult, err := testClient.client.TeamMembers.GetTeamMembers(ctx, &GetTeamMembersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			})
			require.Nil(t, err)

			actualNames := []string{}
			for _, gotTeam := range gotResult.TeamMembers {
				actualNames = append(actualNames, buildTeamMemberName(createdWarmupOutput, gotTeam))
			}

			// The sorting in place here is what makes the copying of the string slice necessary.
			sort.Strings(test.expectTeamMemberNames)
			sort.Strings(actualNames)

			// Make sure the expected names remain.
			assert.Equal(t, test.expectTeamMemberNames, actualNames)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsersForTeamMembers = []models.User{
	{
		Username: "user-0",
		Email:    "user-0@example.com",
	},
	{
		Username: "user-1",
		Email:    "user-1@example.com",
	},
	{
		Username: "user-2",
		Email:    "user-2@example.com",
	},
	{
		Username: "user-99",
		Email:    "user-99@example.com",
	},
}

// Standard warmup teams for tests in this module:
var standardWarmupTeamsForTeamMembers = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for namespace membership tests",
	},
	{
		Name:        "team-b",
		Description: "team b for namespace membership tests",
	},
	{
		Name:        "team-c",
		Description: "team c for namespace membership tests",
	},
	{
		Name:        "team-99",
		Description: "team 99 for namespace membership tests",
	},
}

// Standard warmup team member objects for tests in this module:
// Please note that the ID fields contain names, not IDs.
var standardWarmupTeamMembers = []models.TeamMember{
	{
		UserID:       "user-0",
		TeamID:       "team-a",
		IsMaintainer: true,
	},
	{
		UserID:       "user-1",
		TeamID:       "team-b",
		IsMaintainer: true,
	},
	{
		UserID:       "user-2",
		TeamID:       "team-c",
		IsMaintainer: true,
	},
}

// createWarmupTeamMembers creates some objects for a test
// The objects to create can be standard or otherwise.
func createWarmupTeamMembers(ctx context.Context, testClient *testClient,
	input teamMemberWarmupsInput,
) (*teamMemberWarmupsOutput, error) {
	resultTeams, teamName2ID, err := createInitialTeams(ctx, testClient, input.teams)
	if err != nil {
		return nil, err
	}

	resultUsers, username2ID, err := createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, err
	}

	resultTeamMembers, err := createInitialTeamMembers(ctx, testClient, teamName2ID, username2ID, input.teamMembers)
	if err != nil {
		return nil, err
	}

	return &teamMemberWarmupsOutput{
		teams:        resultTeams,
		users:        resultUsers,
		teamMembers:  resultTeamMembers,
		userIDs2Name: reverseMap(username2ID),
		teamIDs2Name: reverseMap(teamName2ID),
	}, nil
}

func buildTeamMemberName(warmups *teamMemberWarmupsOutput, teamMember models.TeamMember) string {
	return fmt.Sprintf("%s--%s", warmups.teamIDs2Name[teamMember.TeamID], warmups.userIDs2Name[teamMember.UserID])
}

func (tmns teamMemberNameSlice) Len() int {
	return len(tmns.teamMembers)
}

func (tmns teamMemberNameSlice) Swap(i, j int) {
	tmns.teamMembers[i], tmns.teamMembers[j] = tmns.teamMembers[j], tmns.teamMembers[i]
}

func (tmns teamMemberNameSlice) Less(i, j int) bool {
	return buildTeamMemberName(tmns.warmupOutput, tmns.teamMembers[i]) <
		buildTeamMemberName(tmns.warmupOutput, tmns.teamMembers[j])
}

// compareTeamMembers compares two team member objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTeamMembers(t *testing.T, expected, actual *models.TeamMember,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.UserID, actual.UserID)
	assert.Equal(t, expected.TeamID, actual.TeamID)
	assert.Equal(t, expected.IsMaintainer, actual.IsMaintainer)

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
