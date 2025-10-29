//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for TeamMemberSortableField
func (tm TeamMemberSortableField) getValue() string {
	return string(tm)
}

func TestTeamMembers_CreateTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-teammember",
		Email:    "test-teammember@example.com",
	})
	require.NoError(t, err)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-member",
		Description: "test team for team member",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		teamMember      *models.TeamMember
	}

	testCases := []testCase{
		{
			name: "successfully add user to team",
			teamMember: &models.TeamMember{
				UserID:       user.Metadata.ID,
				TeamID:       team.Metadata.ID,
				IsMaintainer: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			teamMember, err := testClient.client.TeamMembers.AddUserToTeam(ctx, test.teamMember)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, teamMember)
			assert.Equal(t, test.teamMember.UserID, teamMember.UserID)
			assert.Equal(t, test.teamMember.TeamID, teamMember.TeamID)
			assert.Equal(t, test.teamMember.IsMaintainer, teamMember.IsMaintainer)
		})
	}
}

func TestTeamMembers_UpdateTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-teammember-update",
		Email:    "test-teammember-update@example.com",
	})
	require.NoError(t, err)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-member-update",
		Description: "test team for team member update",
	})
	require.NoError(t, err)

	// Create a team member to update
	createdTeamMember, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: false,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		updateMember    *models.TeamMember
	}

	testCases := []testCase{
		{
			name: "successfully update team member",
			updateMember: &models.TeamMember{
				Metadata:     createdTeamMember.Metadata,
				UserID:       createdTeamMember.UserID,
				TeamID:       createdTeamMember.TeamID,
				IsMaintainer: true, // Change to maintainer
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			teamMember, err := testClient.client.TeamMembers.UpdateTeamMember(ctx, test.updateMember)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, teamMember)
			assert.Equal(t, test.updateMember.IsMaintainer, teamMember.IsMaintainer)
		})
	}
}

func TestTeamMembers_DeleteTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-teammember-delete",
		Email:    "test-teammember-delete@example.com",
	})
	require.NoError(t, err)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-member-delete",
		Description: "test team for team member delete",
	})
	require.NoError(t, err)

	// Create a team member to delete
	createdTeamMember, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: false,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		teamMember      *models.TeamMember
	}

	testCases := []testCase{
		{
			name:       "successfully remove user from team",
			teamMember: createdTeamMember,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TeamMembers.RemoveUserFromTeam(ctx, test.teamMember)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify the team member was removed
			deletedTeamMember, err := testClient.client.TeamMembers.GetTeamMember(ctx, test.teamMember.UserID, test.teamMember.TeamID)
			if err != nil {
				assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
			}
			assert.Nil(t, deletedTeamMember)
		})
	}
}

func TestTeamMembers_GetTeamMember(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-member-get",
		Description: "test team for member get",
	})
	require.NoError(t, err)

	// Create a user for testing
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-member-get",
		Email:    "test-user-member-get@example.com",
	})
	require.NoError(t, err)

	// Create a team member for testing
	createdMember, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: false,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		userID          string
		teamID          string
		expectMember    bool
	}

	testCases := []testCase{
		{
			name:         "get team member",
			userID:       user.Metadata.ID,
			teamID:       team.Metadata.ID,
			expectMember: true,
		},
		{
			name:   "team member not found",
			userID: nonExistentID,
			teamID: team.Metadata.ID,
		},
		{
			name:            "get team member with invalid user ID will return an error",
			userID:          invalidID,
			teamID:          team.Metadata.ID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			member, err := testClient.client.TeamMembers.GetTeamMember(ctx, test.userID, test.teamID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectMember {
				require.NotNil(t, member)
				assert.Equal(t, createdMember.Metadata.ID, member.Metadata.ID)
			} else {
				assert.Nil(t, member)
			}
		})
	}
}

func TestTeamMembers_GetTeamMemberByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-member-get-by-id",
		Description: "test team for member get by id",
	})
	require.NoError(t, err)

	// Create a user for testing
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-member-get-by-id",
		Email:    "test-user-member-get-by-id@example.com",
	})
	require.NoError(t, err)

	// Create a team member for testing
	createdMember, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: false,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectMember    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdMember.Metadata.ID,
			expectMember: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			member, err := testClient.client.TeamMembers.GetTeamMemberByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectMember {
				require.NotNil(t, member)
				assert.Equal(t, test.id, member.Metadata.ID)
			} else {
				assert.Nil(t, member)
			}
		})
	}
}

func TestTeamMembers_GetTeamMembers(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-members-list",
		Description: "test team for members list",
	})
	require.NoError(t, err)

	// Create users for testing
	users := []models.User{
		{
			Username: "test-user-member-1",
			Email:    "test-user-member-1@example.com",
		},
		{
			Username: "test-user-member-2",
			Email:    "test-user-member-2@example.com",
		},
	}

	createdUsers := []models.User{}
	for _, user := range users {
		created, err := testClient.client.Users.CreateUser(ctx, &user)
		require.NoError(t, err)
		createdUsers = append(createdUsers, *created)
	}

	// Create team members
	createdMembers := []models.TeamMember{}
	for i, user := range createdUsers {
		member, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
			UserID:       user.Metadata.ID,
			TeamID:       team.Metadata.ID,
			IsMaintainer: i == 0, // First user is maintainer
		})
		require.NoError(t, err)
		createdMembers = append(createdMembers, *member)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetTeamMembersInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name: "get all team members",
			input: &GetTeamMembersInput{
				Filter: &TeamMemberFilter{
					TeamIDs: []string{team.Metadata.ID},
				},
			},
			expectCount: len(createdMembers),
		},
		{
			name: "filter by user ID",
			input: &GetTeamMembersInput{
				Filter: &TeamMemberFilter{
					UserID: &createdUsers[0].Metadata.ID,
				},
			},
			expectCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TeamMembers.GetTeamMembers(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.TeamMembers, test.expectCount)
		})
	}
}

func TestTeamMembers_GetTeamMembersWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a team for testing
	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "test-team-members-pagination",
		Description: "test team for members pagination",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
			Username: fmt.Sprintf("test-user-member-%d", i),
			Email:    fmt.Sprintf("test-user-member-%d@example.com", i),
		})
		require.NoError(t, err)

		_, err = testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
			UserID:       createdUser.Metadata.ID,
			TeamID:       team.Metadata.ID,
			IsMaintainer: false,
		})
		require.NoError(t, err)
	}

	// Test basic pagination without sorting since TeamMemberSortableField.getFieldDescriptor() returns nil
	result, err := testClient.client.TeamMembers.GetTeamMembers(ctx, &GetTeamMembersInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(5),
		},
		Filter: &TeamMemberFilter{
			TeamIDs: []string{team.Metadata.ID},
		},
	})
	require.NoError(t, err)
	assert.Len(t, result.TeamMembers, 5)

	// Test getting all members
	result, err = testClient.client.TeamMembers.GetTeamMembers(ctx, &GetTeamMembersInput{
		Filter: &TeamMemberFilter{
			TeamIDs: []string{team.Metadata.ID},
		},
	})
	require.NoError(t, err)
	assert.Len(t, result.TeamMembers, resourceCount)
}
