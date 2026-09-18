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
			// A freshly created role always gets the column default, since CreateRole never sets
			// sort_order — only the default-role seed migrations assign anything else.
			assert.Equal(t, 100, role.SortOrder)
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

func TestRoles_GetRolesDefaultSortOrder(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// The roles table is truncated per test, so seed rows with explicit sort_order values the same
	// way the production migration does, to exercise the default (no explicit Sort) ordering.
	seedRoles := []struct {
		name      string
		sortOrder int
	}{
		{name: "owner", sortOrder: 4},
		{name: "maintainer", sortOrder: 3},
		{name: "deployer", sortOrder: 2},
		{name: "publisher", sortOrder: 1},
		{name: "viewer", sortOrder: 0},
	}

	conn := testClient.client.getConnection(ctx)
	for _, seed := range seedRoles {
		_, err := conn.Exec(
			ctx,
			`INSERT INTO roles (id, version, created_at, updated_at, created_by, name, description, permissions, sort_order)
			 VALUES (gen_random_uuid(), 1, now(), now(), 'db-integration-tests', $1, $2, '[]', $3)`,
			seed.name, seed.name+" description", seed.sortOrder,
		)
		require.NoError(t, err)
	}

	// A custom role (left at the sort_order column default) should sort after every default role.
	customRole, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:        "custom-role-default-sort-test",
		Description: "custom role",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// No explicit sort: expect the fixed default role display order (viewer, publisher, deployer,
	// maintainer, owner), followed by the custom role.
	result, err := testClient.client.Roles.GetRoles(ctx, &GetRolesInput{})
	require.NoError(t, err)

	names := make([]string, len(result.Roles))
	sortOrders := make([]int, len(result.Roles))
	for i, role := range result.Roles {
		names[i] = role.Name
		sortOrders[i] = role.SortOrder
	}

	expectedOrder := []string{"viewer", "publisher", "deployer", "maintainer", "owner", customRole.Name}
	assert.Equal(t, expectedOrder, names)
	// Confirm SortOrder is actually scanned onto the model, not just used internally to order the query.
	assert.Equal(t, []int{0, 1, 2, 3, 4, 100}, sortOrders)

	// Paginate through the same default-sorted result two at a time, following the cursor each
	// page returns, to confirm the sort_order-based cursor (ResolveMetadata("sort_order")) actually
	// round-trips through Postgres correctly rather than only being verified as a single-page order.
	var (
		pagedNames []string
		after      *string
		pageSize   = int32(2)
	)
	for {
		page, pageErr := testClient.client.Roles.GetRoles(ctx, &GetRolesInput{
			PaginationOptions: &pagination.Options{First: &pageSize, After: after},
		})
		require.NoError(t, pageErr)

		for _, role := range page.Roles {
			pagedNames = append(pagedNames, role.Name)
		}

		if !page.PageInfo.HasNextPage {
			break
		}
		cursor, cursorErr := page.PageInfo.Cursor(&page.Roles[len(page.Roles)-1])
		require.NoError(t, cursorErr)
		after = cursor
	}

	assert.Equal(t, expectedOrder, pagedNames)
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
