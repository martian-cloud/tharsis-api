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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// getValue implements the sortableField interface for NamespaceMembershipSortableField
func (nm NamespaceMembershipSortableField) getValue() string {
	return string(nm)
}

func TestCreateNamespaceMembership(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-create-membership",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-create-membership",
		Email:    "test-user-create-membership@example.com",
	})
	require.Nil(t, err)

	serviceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:      "test-sa-create-membership",
		GroupID:   group.Metadata.ID,
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team-create-membership",
	})
	require.Nil(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-create-membership",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		input           *CreateNamespaceMembershipInput
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "create membership for a user",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				UserID:        &user.Metadata.ID,
				RoleID:        role.Metadata.ID,
			},
		},
		{
			name: "create membership for a service account",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    group.FullPath,
				ServiceAccountID: &serviceAccount.Metadata.ID,
				RoleID:           role.Metadata.ID,
			},
		},
		{
			name: "create membership for a team",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				TeamID:        &team.Metadata.ID,
				RoleID:        role.Metadata.ID,
			},
		},
		{
			name: "duplicate membership for a user is rejected",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				UserID:        &user.Metadata.ID,
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "duplicate membership for a service account is rejected",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    group.FullPath,
				ServiceAccountID: &serviceAccount.Metadata.ID,
				RoleID:           role.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "duplicate membership for a team is rejected",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				TeamID:        &team.Metadata.ID,
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "namespace does not exist",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "bogus-namespace",
				UserID:        &user.Metadata.ID,
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "user does not exist",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				UserID:        ptr.String(nonExistentID),
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "service account does not exist",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    group.FullPath,
				ServiceAccountID: ptr.String(nonExistentID),
				RoleID:           role.Metadata.ID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "team does not exist",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				TeamID:        ptr.String(nonExistentID),
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "invalid user id",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				UserID:        ptr.String(invalidID),
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid service account id",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    group.FullPath,
				ServiceAccountID: ptr.String(invalidID),
				RoleID:           role.Metadata.ID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid team id",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: group.FullPath,
				TeamID:        ptr.String(invalidID),
				RoleID:        role.Metadata.ID,
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, created)
				return
			}

			require.Nil(t, err)
			require.NotNil(t, created)

			assert.Equal(t, initialResourceVersion, created.Metadata.Version)
			assert.NotEmpty(t, created.Metadata.TRN)
			assert.Equal(t, test.input.RoleID, created.RoleID)
			assert.Equal(t, test.input.NamespacePath, created.Namespace.Path)
			assert.Equal(t, test.input.UserID, created.UserID)
			assert.Equal(t, test.input.ServiceAccountID, created.ServiceAccountID)
			assert.Equal(t, test.input.TeamID, created.TeamID)
		})
	}
}

func TestUpdateNamespaceMembership(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-update-membership",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-update-membership",
		Email:    "test-user-update-membership@example.com",
	})
	require.Nil(t, err)

	roleA, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-update-membership-a",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	roleB, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-update-membership-b",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	createdMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: group.FullPath,
		UserID:        &user.Metadata.ID,
		RoleID:        roleA.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		id              string
		version         int
		roleID          string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "update role",
			id:      createdMembership.Metadata.ID,
			version: createdMembership.Metadata.Version,
			roleID:  roleB.Metadata.ID,
		},
		{
			name:            "update will fail because resource version doesn't match",
			id:              createdMembership.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			roleID:          roleA.Metadata.ID,
		},
		{
			name:            "negative, invalid",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
			roleID:          roleA.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			updatedMembership, err := testClient.client.NamespaceMemberships.UpdateNamespaceMembership(ctx, &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
				RoleID: test.roleID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, updatedMembership)
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedMembership)

			assert.Equal(t, test.roleID, updatedMembership.RoleID)
			assert.Equal(t, createdMembership.Metadata.Version+1, updatedMembership.Metadata.Version)
		})
	}
}

func TestDeleteNamespaceMembership(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-delete-membership",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-delete-membership",
		Email:    "test-user-delete-membership@example.com",
	})
	require.Nil(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-delete-membership",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	createdMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: group.FullPath,
		UserID:        &user.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		id              string
		version         int
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "delete membership",
			id:      createdMembership.Metadata.ID,
			version: createdMembership.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdMembership.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "negative, does not exist",
			id:              nonExistentID,
			expectErrorCode: errors.EOptimisticLock,
		},
		{
			name:            "negative, invalid",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.NamespaceMemberships.DeleteNamespaceMembership(ctx, &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			membership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, test.id)
			assert.Nil(t, membership)
			assert.Nil(t, err)
		})
	}
}

func TestGetNamespaceMembershipByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-get-by-id",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-get-by-id",
		Email:    "test-user-get-by-id@example.com",
	})
	require.Nil(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-get-by-id",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	createdMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: group.FullPath,
		UserID:        &user.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode  errors.CodeType
		name             string
		searchID         string
		expectMembership bool
	}

	testCases := []testCase{
		{
			name:             "get resource by id",
			searchID:         createdMembership.Metadata.ID,
			expectMembership: true,
		},
		{
			name:     "resource with id not found",
			searchID: nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			searchID:        invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			membership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectMembership {
				require.NotNil(t, membership)
				assert.Equal(t, test.searchID, membership.Metadata.ID)
			} else {
				assert.Nil(t, membership)
			}
		})
	}
}

func TestGetNamespaceMembershipByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name: "test-role",
	})
	require.NoError(t, err)

	namespaceMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: group.FullPath,
		RoleID:        role.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name                      string
		trn                       string
		expectNamespaceMembership bool
		expectErrorCode           errors.CodeType
	}

	testCases := []testCase{
		{
			name:                      "get resource by TRN",
			trn:                       namespaceMembership.Metadata.TRN,
			expectNamespaceMembership: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeNamespaceMembership.Build(group.FullPath, nonExistentGlobalID),
		},
		{
			name:            "namespace membership TRN must not have less than two parts",
			trn:             trn.TypeNamespaceMembership.Build(nonExistentGlobalID),
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
			actualNamespaceMembership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectNamespaceMembership {
				require.NotNil(t, actualNamespaceMembership)
				assert.Equal(t, trn.TypeNamespaceMembership.Build(group.FullPath, namespaceMembership.GetGlobalID()), actualNamespaceMembership.Metadata.TRN)
			} else {
				assert.Nil(t, actualNamespaceMembership)
			}
		})
	}
}

func TestGetNamespaceMemberships(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	groupA, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-list-a",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	groupB, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-list-b",
		ParentID:  groupA.Metadata.ID,
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	workspaceA1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-ws-list-a1",
		GroupID:        groupA.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(int32(forTestMaxJobDuration.Minutes())),
	})
	require.Nil(t, err)

	user0, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-list-0",
		Email:    "test-user-list-0@example.com",
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-list-1",
		Email:    "test-user-list-1@example.com",
	})
	require.Nil(t, err)

	serviceAccount0, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:      "test-sa-list-0",
		GroupID:   groupA.Metadata.ID,
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	team, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "test-team-list",
	})
	require.Nil(t, err)

	_, err = testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID: user1.Metadata.ID,
		TeamID: team.Metadata.ID,
	})
	require.Nil(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-list",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	// groupA.FullPath is a prefix for groupB's and workspaceA1's paths, so this membership
	// exercises the plain lookup as well as the "no children" behavior of the filters.
	mUserGroupA, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: groupA.FullPath,
		UserID:        &user0.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.Nil(t, err)

	mSAGroupB, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath:    groupB.FullPath,
		ServiceAccountID: &serviceAccount0.Metadata.ID,
		RoleID:           role.Metadata.ID,
	})
	require.Nil(t, err)

	// This membership belongs to the team, so it is also the target for the indirect,
	// team-based membership filter on user1.
	mTeamWorkspaceA1, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: workspaceA1.FullPath,
		TeamID:        &team.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.Nil(t, err)

	allIDs := []string{mUserGroupA.Metadata.ID, mSAGroupB.Metadata.ID, mTeamWorkspaceA1.Metadata.ID}

	type testCase struct {
		name            string
		input           *GetNamespaceMembershipsInput
		expectErrorCode errors.CodeType
		expectIDs       []string
	}

	testCases := []testCase{
		{
			name:      "no filter returns all",
			input:     &GetNamespaceMembershipsInput{},
			expectIDs: allIDs,
		},
		{
			name: "filter, direct user membership",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{UserID: &user0.Metadata.ID},
			},
			expectIDs: []string{mUserGroupA.Metadata.ID},
		},
		{
			name: "filter, indirect user membership via team",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{UserID: &user1.Metadata.ID},
			},
			expectIDs: []string{mTeamWorkspaceA1.Metadata.ID},
		},
		{
			name: "filter, user id, non-existent",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{UserID: ptr.String(nonExistentID)},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, user id, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{UserID: ptr.String(invalidID)},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, service account id",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{ServiceAccountID: &serviceAccount0.Metadata.ID},
			},
			expectIDs: []string{mSAGroupB.Metadata.ID},
		},
		{
			name: "filter, service account id, non-existent",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{ServiceAccountID: ptr.String(nonExistentID)},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, service account id, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{ServiceAccountID: ptr.String(invalidID)},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, team id",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{TeamID: &team.Metadata.ID},
			},
			expectIDs: []string{mTeamWorkspaceA1.Metadata.ID},
		},
		{
			name: "filter, team id, non-existent",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{TeamID: ptr.String(nonExistentID)},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, team id, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{TeamID: ptr.String(invalidID)},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, group id",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{GroupID: &groupA.Metadata.ID},
			},
			expectIDs: []string{mUserGroupA.Metadata.ID},
		},
		{
			name: "filter, group id, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{GroupID: ptr.String(invalidID)},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, workspace id",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{WorkspaceID: &workspaceA1.Metadata.ID},
			},
			expectIDs: []string{mTeamWorkspaceA1.Metadata.ID},
		},
		{
			name: "filter, workspace id, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{WorkspaceID: ptr.String(invalidID)},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, namespace path prefix, matches group and its descendants",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePathPrefix: &groupA.FullPath},
			},
			expectIDs: allIDs,
		},
		{
			name: "filter, namespace path prefix, matches a descendant group only",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePathPrefix: &groupB.FullPath},
			},
			expectIDs: []string{mSAGroupB.Metadata.ID},
		},
		{
			name: "filter, namespace path prefix, no match",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePathPrefix: ptr.String("bogus-namespace")},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, empty slice of namespace paths matches everything",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePaths: []string{}},
			},
			expectIDs: allIDs,
		},
		{
			name: "filter, namespace paths, exact matches",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePaths: []string{groupA.FullPath, workspaceA1.FullPath}},
			},
			expectIDs: []string{mUserGroupA.Metadata.ID, mTeamWorkspaceA1.Metadata.ID},
		},
		{
			name: "filter, namespace paths, non-existent",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespacePaths: []string{"bogus-namespace"}},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, namespace membership ids",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespaceMembershipIDs: []string{mSAGroupB.Metadata.ID}},
			},
			expectIDs: []string{mSAGroupB.Metadata.ID},
		},
		{
			name: "filter, namespace membership ids, non-existent",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespaceMembershipIDs: []string{nonExistentID}},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, namespace membership ids, invalid",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{NamespaceMembershipIDs: []string{invalidID}},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "filter, combination user id and group id, match",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{UserID: &user0.Metadata.ID, GroupID: &groupA.Metadata.ID},
			},
			expectIDs: []string{mUserGroupA.Metadata.ID},
		},
		{
			name: "filter, combination group id and workspace id, contradictory",
			input: &GetNamespaceMembershipsInput{
				Filter: &NamespaceMembershipFilter{GroupID: &groupA.Metadata.ID, WorkspaceID: &workspaceA1.Metadata.ID},
			},
			expectIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.NamespaceMemberships.GetNamespaceMemberships(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, result)

			gotIDs := make([]string, len(result.NamespaceMemberships))
			for i, m := range result.NamespaceMemberships {
				gotIDs[i] = m.Metadata.ID
			}

			assert.ElementsMatch(t, test.expectIDs, gotIDs)

			totalCount, err := result.PageInfo.TotalCount(ctx)
			require.Nil(t, err)
			assert.Equal(t, int32(len(test.expectIDs)), totalCount)
		})
	}
}

func TestGetNamespaceMembershipsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-membership-pagination",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role-membership-pagination",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
			Name:           fmt.Sprintf("test-ws-membership-pagination-%d", i),
			GroupID:        group.Metadata.ID,
			CreatedBy:      "db-integration-tests",
			MaxJobDuration: ptr.Int32(int32(forTestMaxJobDuration.Minutes())),
		})
		require.Nil(t, err)

		user, err := testClient.client.Users.CreateUser(ctx, &models.User{
			Username: fmt.Sprintf("test-user-membership-pagination-%d", i),
			Email:    fmt.Sprintf("test-user-membership-pagination-%d@example.com", i),
		})
		require.Nil(t, err)

		_, err = testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
			NamespacePath: workspace.FullPath,
			UserID:        &user.Metadata.ID,
			RoleID:        role.Metadata.ID,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		NamespaceMembershipSortableFieldNamespacePathAsc,
		NamespaceMembershipSortableFieldNamespacePathDesc,
		NamespaceMembershipSortableFieldUpdatedAtAsc,
		NamespaceMembershipSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := NamespaceMembershipSortableField(sortByField.getValue())

		result, err := testClient.client.NamespaceMemberships.GetNamespaceMemberships(ctx, &GetNamespaceMembershipsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for i := range result.NamespaceMemberships {
			resources = append(resources, &result.NamespaceMemberships[i])
		}

		return result.PageInfo, resources, nil
	})
}
