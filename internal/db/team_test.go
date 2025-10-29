//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for TeamSortableField
func (t TeamSortableField) getValue() string {
	return string(t)
}

func TestTeams_GetTeamByID(t *testing.T) {
	ctx := context.Background()
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

func TestTeams_GetTeamByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectTeam      bool
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

func TestTeams_CreateTeam(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		teamName        string
		description     string
	}

	testCases := []testCase{
		{
			name:        "successfully create team",
			teamName:    "test-team",
			description: "This is a test team",
		},
		{
			name:            "create will fail because team name already exists",
			teamName:        "test-team",
			description:     "This would be a duplicate team",
			expectErrorCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTeam, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
				Name:        test.teamName,
				Description: test.description,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualTeam)
			assert.Equal(t, test.teamName, actualTeam.Name)
			assert.Equal(t, test.description, actualTeam.Description)
		})
	}
}

func TestTeams_GetTeamBySCIMExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	externalID := uuid.New().String()
	_, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:           "test-team",
		SCIMExternalID: externalID,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		searchID        string
		expectTeam      bool
	}

	testCases := []testCase{
		{
			name:       "get team by SCIM external ID",
			searchID:   externalID,
			expectTeam: true,
		},
		{
			name:     "resource with ID not found",
			searchID: nonExistentID,
		},
		{
			name:            "get resource with invalid ID will return an error",
			searchID:        invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTeam, err := testClient.client.Teams.GetTeamBySCIMExternalID(ctx, test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectTeam {
				require.NotNil(t, actualTeam)
				assert.Equal(t, test.searchID, actualTeam.SCIMExternalID)
			} else {
				assert.Nil(t, actualTeam)
			}
		})
	}
}

func TestTeams_GetTeams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test teams
	teams := []*models.Team{}
	for i := 0; i < 3; i++ {
		team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
			Name:        fmt.Sprintf("test-team-%d", i),
			Description: fmt.Sprintf("Test team %d", i),
		})
		require.NoError(t, err)
		teams = append(teams, team)
	}

	type testCase struct {
		name              string
		filter            *TeamFilter
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return all teams when no filter is provided",
			filter:            nil,
			expectResultCount: 3,
		},
		{
			name: "filter by team name prefix",
			filter: &TeamFilter{
				TeamNamePrefix: ptr.String("test-team-1"),
			},
			expectResultCount: 1,
		},
		{
			name: "filter by team IDs",
			filter: &TeamFilter{
				TeamIDs: []string{teams[0].Metadata.ID, teams[2].Metadata.ID},
			},
			expectResultCount: 2,
		},
		{
			name: "filter with non-existent team ID",
			filter: &TeamFilter{
				TeamIDs: []string{nonExistentID},
			},
			expectResultCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Teams.GetTeams(ctx, &GetTeamsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.Teams))
		})
	}
}

func TestTeams_UpdateTeam(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team",
		Description: "Original description",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		description     string
	}

	testCases := []testCase{
		{
			name:        "successfully update team",
			version:     1,
			description: "Updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "This should fail",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualTeam, err := testClient.client.Teams.UpdateTeam(ctx, &models.Team{
				Metadata: models.ResourceMetadata{
					ID:      team.Metadata.ID,
					Version: test.version,
				},
				Name:        team.Name,
				Description: test.description,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualTeam)
			assert.Equal(t, test.description, actualTeam.Description)
		})
	}
}

func TestTeams_DeleteTeam(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              team.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete team",
			id:      team.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Teams.DeleteTeam(ctx, &models.Team{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestTeams_GetTeamsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
			Name:        fmt.Sprintf("test-team-%d", i),
			Description: fmt.Sprintf("test team %d", i),
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TeamSortableFieldNameAsc,
		TeamSortableFieldNameDesc,
		TeamSortableFieldUpdatedAtAsc,
		TeamSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TeamSortableField(sortByField.getValue())

		result, err := testClient.client.Teams.GetTeams(ctx, &GetTeamsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Teams {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
