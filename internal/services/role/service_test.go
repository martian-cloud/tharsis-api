package role

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetAvailablePermissions(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		expectErrorCode string
		expectPerms     []string
	}{
		{
			name:        "successfully retrieve all available permissions",
			caller:      &auth.SystemCaller{},
			expectPerms: permissions.GetAssignablePermissions(),
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(nil, nil, nil)
			actualPerms, err := service.GetAvailablePermissions(auth.WithCaller(context.TODO(), test.caller))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectPerms, actualPerms)
		})
	}
}

func TestGetRoleByID(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		search          string
		expectedRole    *models.Role
		expectErrorCode string
	}{
		{
			name:         "Role was found",
			caller:       &auth.SystemCaller{},
			search:       "role-1",
			expectedRole: &models.Role{Name: "role"},
		},
		{
			name:            "role does not exist",
			caller:          &auth.SystemCaller{},
			search:          "role-2",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			search:          "role-3",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRoles := db.NewMockRoles(t)

			if test.caller != nil {
				mockRoles.On("GetRoleByID", mock.Anything, test.search).Return(test.expectedRole, nil)
			}

			dbClient := db.Client{
				Roles: mockRoles,
			}

			service := NewService(nil, &dbClient, nil)
			actualRole, err := service.GetRoleByID(auth.WithCaller(context.TODO(), test.caller), test.search)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectedRole, actualRole)
		})
	}
}

func TestGetRoleByName(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		search          string
		expectedRole    *models.Role
		expectErrorCode string
	}{
		{
			name:         "Role was found",
			caller:       &auth.SystemCaller{},
			search:       "role",
			expectedRole: &models.Role{Name: "role"},
		},
		{
			name:            "role does not exist",
			caller:          &auth.SystemCaller{},
			search:          "role-2",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			search:          "role-3",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRoles := db.NewMockRoles(t)

			if test.caller != nil {
				mockRoles.On("GetRoleByName", mock.Anything, test.search).Return(test.expectedRole, nil)
			}

			dbClient := db.Client{
				Roles: mockRoles,
			}

			service := NewService(nil, &dbClient, nil)
			actualRole, err := service.GetRoleByName(auth.WithCaller(context.TODO(), test.caller), test.search)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectedRole, actualRole)
		})
	}
}

func TestGetRolesByIDs(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		expectErrorCode string
		input           []string
		expectResult    []models.Role
	}{
		{
			name:         "successfully retrieve roles",
			caller:       &auth.SystemCaller{},
			input:        []string{"role-1", "role-2"},
			expectResult: []models.Role{{Name: "role"}, {Name: "another"}},
		},
		{
			name:            "without caller",
			input:           []string{"role-1", "role-2"},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRoles := db.NewMockRoles(t)

			if test.caller != nil {
				dbInput := &db.GetRolesInput{Filter: &db.RoleFilter{RoleIDs: test.input}}
				mockRoles.On("GetRoles", mock.Anything, dbInput).Return(&db.RolesResult{Roles: test.expectResult}, nil)
			}

			dbClient := db.Client{
				Roles: mockRoles,
			}

			service := NewService(nil, &dbClient, nil)
			actualRoles, err := service.GetRolesByIDs(auth.WithCaller(context.TODO(), test.caller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, actualRoles)
		})
	}
}

func TestGetRoles(t *testing.T) {
	sort := db.RoleSortableFieldNameAsc

	testCases := []struct {
		caller          auth.Caller
		input           *GetRolesInput
		expectResult    *db.RolesResult
		name            string
		expectErrorCode string
	}{
		{
			name:   "successfully retrieve roles",
			caller: &auth.SystemCaller{},
			input: &GetRolesInput{
				Sort: &sort,
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
				RoleNamePrefix: ptr.String("role"),
			},
			expectResult: &db.RolesResult{
				Roles: []models.Role{{Name: "role"}},
			},
		},
		{
			name:            "without caller",
			input:           &GetRolesInput{RoleNamePrefix: ptr.String("role")},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRoles := db.NewMockRoles(t)

			if test.caller != nil {
				dbInput := &db.GetRolesInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
					Filter: &db.RoleFilter{
						RoleNamePrefix: test.input.RoleNamePrefix,
					},
				}
				mockRoles.On("GetRoles", mock.Anything, dbInput).Return(test.expectResult, nil)
			}

			dbClient := db.Client{
				Roles: mockRoles,
			}

			service := NewService(nil, &dbClient, nil)
			actualRoles, err := service.GetRoles(auth.WithCaller(context.TODO(), test.caller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, actualRoles)
		})
	}
}

func TestCreateRole(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		input           *CreateRoleInput
		expectRole      *models.Role
		expectErrorCode string
	}{
		{
			name: "successfully create a role",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
					Email: "user@email",
				},
			},
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []permissions.Permission{
					permissions.CreateConfigurationVersionPermission,
				},
			},
			expectRole: &models.Role{
				Name:        "role",
				Description: "Some new role.",
				CreatedBy:   "user@email",
			},
		},
		{
			name: "permissions are not assignable",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
				},
			},
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []permissions.Permission{permissions.UpdatePlanPermission},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "not a user caller",
			caller: &auth.SystemCaller{},
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []permissions.Permission{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is not an admin",
			caller: &auth.UserCaller{
				User: &models.User{},
			},
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []permissions.Permission{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "without caller",
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []permissions.Permission{},
			},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.TODO(), test.caller)

			mockRoles := db.NewMockRoles(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)

			if test.expectRole != nil {
				eventsInput := &activityevent.CreateActivityEventInput{
					Action:     models.ActionCreate,
					TargetType: models.TargetRole,
					TargetID:   test.expectRole.Metadata.ID,
				}

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				test.expectRole.SetPermissions(test.input.Permissions)
				mockRoles.On("CreateRole", mock.Anything, test.expectRole).Return(test.expectRole, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, eventsInput).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := &db.Client{
				Roles:        mockRoles,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			actualRole, err := service.CreateRole(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectRole, actualRole)
			assert.Equal(t, test.expectRole.GetPermissions(), actualRole.GetPermissions())
		})
	}
}

func TestUpdateRole(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		input           *UpdateRoleInput
		expectRole      *models.Role
		name            string
		expectErrorCode string
		updatePerms     []permissions.Permission
		expectPerms     []permissions.Permission
	}{
		{
			name: "successfully update a role",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
					Email: "user@email",
				},
			},
			input: &UpdateRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: "role-1",
					},
					Name: "role",
				},
			},
			updatePerms: []permissions.Permission{
				permissions.CreateGroupPermission,
				permissions.CreateGroupPermission, // Should be deduplicated.
			},
			expectRole: &models.Role{
				Metadata: models.ResourceMetadata{
					ID: "role-1",
				},
				Name: "role",
			},
			expectPerms: []permissions.Permission{permissions.CreateGroupPermission},
		},
		{
			name: "permissions are not assignable",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
				},
			},
			input: &UpdateRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: "role-1",
					},
					Name: "role",
				},
			},
			updatePerms: []permissions.Permission{
				permissions.UpdatePlanPermission,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "updating a default role",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
				},
			},
			input: &UpdateRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: models.OwnerRoleID.String(),
					},
					Name:        "owner",
					Description: "updated description",
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:   "not a user caller",
			caller: &auth.SystemCaller{},
			input: &UpdateRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is not an admin",
			caller: &auth.UserCaller{
				User: &models.User{},
			},
			input: &UpdateRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "without caller",
			input: &UpdateRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.TODO(), test.caller)

			mockRoles := db.NewMockRoles(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)

			// Set permissions.
			test.input.Role.SetPermissions(test.updatePerms)

			if test.expectRole != nil {
				eventsInput := &activityevent.CreateActivityEventInput{
					Action:     models.ActionUpdate,
					TargetType: models.TargetRole,
					TargetID:   test.expectRole.Metadata.ID,
				}

				test.expectRole.SetPermissions(test.expectPerms)

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockRoles.On("UpdateRole", mock.Anything, test.expectRole).Return(test.expectRole, nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, eventsInput).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := &db.Client{
				Roles:        mockRoles,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockActivityEvents)

			actualRole, err := service.UpdateRole(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectRole, actualRole)
			assert.Equal(t, test.expectPerms, actualRole.GetPermissions())
		})
	}
}

func TestDeleteRole(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		input           *DeleteRoleInput
		name            string
		expectErrorCode string
		memberships     []models.NamespaceMembership
	}{
		{
			name: "successfully delete a role without force option",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
					Email: "user@email",
				},
			},
			input: &DeleteRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: "role-1",
					},
				},
			},
		},
		{
			name: "successfully deleting a role associated with namespace memberships with force option",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
					Email: "user@email",
				},
			},
			memberships: []models.NamespaceMembership{{}, {}},
			input: &DeleteRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: "role-1",
					},
				},
				Force: true,
			},
		},
		{
			name: "deleting a role associated with namespace memberships without force option",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
					Email: "user@email",
				},
			},
			memberships: []models.NamespaceMembership{{}, {}},
			input: &DeleteRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: "role-1",
					},
				},
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "deleting a default role",
			caller: &auth.UserCaller{
				User: &models.User{
					Admin: true,
				},
			},
			input: &DeleteRoleInput{
				Role: &models.Role{
					Metadata: models.ResourceMetadata{
						ID: models.OwnerRoleID.String(),
					},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:   "not a user caller",
			caller: &auth.SystemCaller{},
			input: &DeleteRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is not an admin",
			caller: &auth.UserCaller{
				User: &models.User{},
			},
			input: &DeleteRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "without caller",
			input: &DeleteRoleInput{
				Role: &models.Role{},
			},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRoles := db.NewMockRoles(t)
			mockMemberships := db.NewMockNamespaceMemberships(t)

			membershipsInput := &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					RoleID: &test.input.Role.Metadata.ID,
				},
			}
			membershipsResult := &db.NamespaceMembershipResult{NamespaceMemberships: test.memberships}
			mockMemberships.On("GetNamespaceMemberships", mock.Anything, membershipsInput).Return(membershipsResult, nil).Maybe()

			if test.expectErrorCode == "" {
				mockRoles.On("DeleteRole", mock.Anything, test.input.Role).Return(nil)
			}

			dbClient := db.Client{
				NamespaceMemberships: mockMemberships,
				Roles:                mockRoles,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, nil)

			err := service.DeleteRole(auth.WithCaller(context.TODO(), test.caller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
