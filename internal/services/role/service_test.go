package role

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetAvailablePermissions(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		expectErrorCode errors.CodeType
		expectPerms     []string
	}{
		{
			name:        "successfully retrieve all available permissions",
			caller:      &auth.SystemCaller{},
			expectPerms: models.GetAssignablePermissions(),
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
		expectErrorCode errors.CodeType
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

func TestGetRoleByTRN(t *testing.T) {
	sampleRole := &models.Role{
		Metadata: models.ResourceMetadata{
			ID:  "role-id-1",
			TRN: types.RoleModelType.BuildTRN("my-role/role-1"),
		},
		Name:        "role-1",
		Description: "Test role",
	}

	type testCase struct {
		caller          auth.Caller
		name            string
		role            *models.Role
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "successfully get role by trn",
			caller: &auth.SystemCaller{},
			role:   sampleRole,
		},
		{
			name:            "role not found",
			caller:          &auth.SystemCaller{},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRoles := db.NewMockRoles(t)

			if test.caller != nil {
				ctx = auth.WithCaller(ctx, test.caller)
				mockRoles.On("GetRoleByTRN", mock.Anything, sampleRole.Metadata.TRN).Return(test.role, nil)
			}

			dbClient := &db.Client{
				Roles: mockRoles,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualRole, err := service.GetRoleByTRN(ctx, sampleRole.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.role, actualRole)
		})
	}
}

func TestGetRolesByIDs(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		expectErrorCode errors.CodeType
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
		expectErrorCode errors.CodeType
	}{
		{
			name:   "successfully retrieve roles",
			caller: &auth.SystemCaller{},
			input: &GetRolesInput{
				Sort: &sort,
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
				Search: ptr.String("role"),
			},
			expectResult: &db.RolesResult{
				Roles: []models.Role{{Name: "role"}},
			},
		},
		{
			name:            "without caller",
			input:           &GetRolesInput{Search: ptr.String("role")},
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
						Search: test.input.Search,
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
		expectErrorCode errors.CodeType
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
				Permissions: []models.Permission{
					models.CreateConfigurationVersionPermission,
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
				Permissions: []models.Permission{models.UpdatePlanPermission},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "not a user caller",
			caller: &auth.SystemCaller{},
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []models.Permission{},
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
				Permissions: []models.Permission{},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "without caller",
			input: &CreateRoleInput{
				Name:        "role",
				Description: "Some new role.",
				Permissions: []models.Permission{},
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
		expectErrorCode errors.CodeType
		updatePerms     []models.Permission
		expectPerms     []models.Permission
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
			updatePerms: []models.Permission{
				models.CreateGroupPermission,
				models.CreateGroupPermission, // Should be deduplicated.
			},
			expectRole: &models.Role{
				Metadata: models.ResourceMetadata{
					ID: "role-1",
				},
				Name: "role",
			},
			expectPerms: []models.Permission{models.CreateGroupPermission},
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
			updatePerms: []models.Permission{
				models.UpdatePlanPermission,
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

			// Set models.
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
		expectErrorCode errors.CodeType
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
