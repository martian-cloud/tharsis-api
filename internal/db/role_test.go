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

// getValue implements the sortableField interface for RoleSortableField
func (r RoleSortableField) getValue() string {
	return string(r)
}

func TestRoles_CreateRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		roleName        string
		description     string
	}

	testCases := []testCase{
		{
			name:        "create role",
			roleName:    "test-role",
			description: "test role description",
		},
		{
			name:        "create role with invalid name",
			roleName:    "",
			description: "invalid role",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
				Name:        test.roleName,
				Description: test.description,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, role)

			assert.Equal(t, test.roleName, role.Name)
			assert.Equal(t, test.description, role.Description)
			assert.NotEmpty(t, role.Metadata.ID)
		})
	}
}

func TestRoles_UpdateRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a role for testing
	createdRole, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:        "test-role-update",
		Description: "original description",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		description     string
	}

	testCases := []testCase{
		{
			name:        "update role",
			version:     createdRole.Metadata.Version,
			description: "updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "should not update",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			roleToUpdate := *createdRole
			roleToUpdate.Metadata.Version = test.version
			roleToUpdate.Description = test.description

			updatedRole, err := testClient.client.Roles.UpdateRole(ctx, &roleToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedRole)

			assert.Equal(t, test.description, updatedRole.Description)
			assert.Equal(t, createdRole.Metadata.Version+1, updatedRole.Metadata.Version)
		})
	}
}

func TestRoles_DeleteRole(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a role for testing
	createdRole, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:        "test-role-delete",
		Description: "role to delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete role",
			id:      createdRole.Metadata.ID,
			version: createdRole.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdRole.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Roles.DeleteRole(ctx, &models.Role{
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

			// Verify role was deleted
			role, err := testClient.client.Roles.GetRoleByID(ctx, test.id)
			assert.Nil(t, role)
			assert.Nil(t, err)
		})
	}
}
func TestRoles_GetRoleByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a role for testing
	roleToCreate := &models.Role{
		Name:        "test-role-get-by-id",
		Description: "Test role for get by ID",
		CreatedBy:   "db-integration-tests",
	}
	roleToCreate.SetPermissions([]models.Permission{
		{Action: "read", ResourceType: "workspace"},
	})
	createdRole, err := testClient.client.Roles.CreateRole(ctx, roleToCreate)
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectRole      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			id:         createdRole.Metadata.ID,
			expectRole: true,
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
			role, err := testClient.client.Roles.GetRoleByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRole {
				require.NotNil(t, role)
				assert.Equal(t, test.id, role.Metadata.ID)
			} else {
				assert.Nil(t, role)
			}
		})
	}
}

func TestRoles_GetRoles(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test roles
	roles := []*models.Role{
		{
			Name:        "test-role-1",
			Description: "Test role 1",
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-role-2",
			Description: "Test role 2",
			CreatedBy:   "db-integration-tests",
		},
	}

	// Set permissions for each role
	roles[0].SetPermissions([]models.Permission{
		{Action: "read", ResourceType: "workspace"},
	})
	roles[1].SetPermissions([]models.Permission{
		{Action: "write", ResourceType: "workspace"},
	})

	createdRoles := []models.Role{}
	for _, role := range roles {
		created, err := testClient.client.Roles.CreateRole(ctx, role)
		require.NoError(t, err)
		createdRoles = append(createdRoles, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetRolesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all roles",
			input:       &GetRolesInput{},
			expectCount: len(createdRoles),
		},
		{
			name: "filter by search",
			input: &GetRolesInput{
				Filter: &RoleFilter{
					Search: ptr.String("test-role-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by role IDs",
			input: &GetRolesInput{
				Filter: &RoleFilter{
					RoleIDs: []string{createdRoles[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Roles.GetRoles(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Roles, test.expectCount)
		})
	}
}

func TestRoles_GetRolesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		role := &models.Role{
			Name:        fmt.Sprintf("role-%d", i),
			Description: fmt.Sprintf("Role %d", i),
			CreatedBy:   "db-integration-tests",
		}
		// Set permissions for each role
		role.SetPermissions([]models.Permission{
			{Action: "read", ResourceType: "workspace"},
		})
		_, err := testClient.client.Roles.CreateRole(ctx, role)
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		RoleSortableFieldNameAsc,
		RoleSortableFieldNameDesc,
		RoleSortableFieldUpdatedAtAsc,
		RoleSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := RoleSortableField(sortByField.getValue())

		result, err := testClient.client.Roles.GetRoles(ctx, &GetRolesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Roles {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestRoles_GetRoleByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a role for testing
	roleToCreate := &models.Role{
		Name:        "test-role-trn",
		Description: "Test role for TRN",
		CreatedBy:   "db-integration-tests",
	}
	roleToCreate.SetPermissions([]models.Permission{
		{Action: "read", ResourceType: "workspace"},
	})
	createdRole, err := testClient.client.Roles.CreateRole(ctx, roleToCreate)
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectRole      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by TRN",
			trn:        createdRole.Metadata.TRN,
			expectRole: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:role:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			role, err := testClient.client.Roles.GetRoleByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRole {
				require.NotNil(t, role)
				assert.Equal(t, test.trn, role.Metadata.TRN)
			} else {
				assert.Nil(t, role)
			}
		})
	}
}
