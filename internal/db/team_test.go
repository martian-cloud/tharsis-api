//go:build integration

package db

import (
	"context"
	"sort"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// teamNameSlice makes a slice of models.Team sortable by name
type teamNameSlice []models.Team

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
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			team, err := testClient.client.Teams.GetTeamBySCIMExternalID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTeam {
				// the positive case
				require.NotNil(t, team)
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

	type testCase struct {
		name        string
		input       *GetTeamsInput
		expectMsg   *string
		expectTeams []models.Team
	}

	/*
		template test case:

		{
		name        string
		input       *GetTeamsInput
		expectMsg   *string
		expectTeams []models.Team
		}
	*/

	// TODO: Add more cases:
	testCases := []testCase{
		{
			name: "simple get teams test case",
			input: &GetTeamsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTeams: createdWarmupTeams,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// TODO: Add checks for pagination, cursors, etc.

			gotResult, err := testClient.client.Teams.GetTeams(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectTeams != nil {
				actualTeams := gotResult.Teams
				require.NotNil(t, actualTeams)
				assert.Equal(t, len(test.expectTeams), len(actualTeams))
				if len(test.expectTeams) == len(actualTeams) {

					// TODO: When implementing test cases that do sorting other than ascending order by name,
					// change this to handle all cases.

					// For now, sort both expected and actual by name in order to compare.
					sort.Sort(teamNameSlice(test.expectTeams))
					sort.Sort(teamNameSlice(actualTeams))

					// Compare the slices of teams, now that they should be sorted the same.
					for ix := 0; ix < len(test.expectTeams); ix++ {
						compareTeams(t, &test.expectTeams[ix], &actualTeams[ix], true, nil)
					}

				}
			}

			// TODO: Add code for pagination, cursors, etc.
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
		Name:           "team-a",
		Description:    "team a for team tests",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "team-b",
		Description:    "team b for team tests",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "team-c",
		Description:    "team c for team tests",
		SCIMExternalID: uuid.New().String(),
	},
	{
		Name:           "team-99",
		Description:    "team 99 for team tests",
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

func (tns teamNameSlice) Len() int {
	return len(tns)
}

func (tns teamNameSlice) Swap(i, j int) {
	tns[i], tns[j] = tns[j], tns[i]
}

func (tns teamNameSlice) Less(i, j int) bool {
	return tns[i].Name < tns[j].Name
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
