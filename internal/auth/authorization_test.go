package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetRootNamespaces(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		userID                 *string
		name                   string
		expectErrorCode        errors.CodeType
		namespaceMemberships   []models.NamespaceMembership
		expectedRootNamespaces []models.MembershipNamespace
	}{
		{
			name: "multiple top level namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{ID: "ns1", Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{ID: "ns11", Path: "ns11"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{ID: "ns1/ns11", Path: "ns1/ns11"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{ID: "ns2/ns22/ns222", Path: "ns2/ns22/ns222"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{ID: "ns3", Path: "ns3"}},
			},
			expectedRootNamespaces: []models.MembershipNamespace{
				{ID: "ns1", Path: "ns1"},
				{ID: "ns11", Path: "ns11"},
				{ID: "ns2/ns22/ns222", Path: "ns2/ns22/ns222"},
				{ID: "ns3", Path: "ns3"},
			},
			userID: &userID,
		},
		{
			name:                   "no namespaces",
			namespaceMemberships:   []models.NamespaceMembership{},
			expectedRootNamespaces: []models.MembershipNamespace{},
			userID:                 &userID,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathAsc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort: &sortBy,
				Filter: &db.NamespaceMembershipFilter{
					UserID: test.userID,
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, test.userID, nil, false)

			namespaces, err := authorizer.GetRootNamespaces(ctx)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, len(test.expectedRootNamespaces), len(namespaces))
			for _, ns := range test.expectedRootNamespaces {
				assert.Contains(t, namespaces, ns)
			}
		})
	}
}

func TestRequireAccess(t *testing.T) {
	userID := "user-1"
	customRoleID := "custom-role-1"
	groupID := "group-1"

	tests := []struct {
		name                 string
		expectErrorCode      errors.CodeType
		perms                []models.Permission
		customRolePerms      []models.Permission
		group                *models.Group
		workspace            *models.Workspace
		namespaceMemberships []models.NamespaceMembership
		constraints          []func(*constraints)
	}{
		{
			name:        "user access is granted",
			perms:       []models.Permission{models.ViewGroupPermission, models.CreateGroupPermission},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1",
			},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "user with custom role is granted access",
			perms:       []models.Permission{models.ViewGroupPermission, models.CreateGroupPermission},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1",
			},
			customRolePerms: []models.Permission{models.CreateGroupPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "multiple permissions and checks are granted access",
			perms:       []models.Permission{models.ViewGroupPermission, models.ViewWorkspacePermission},
			constraints: []func(*constraints){WithGroupID(groupID), WithWorkspaceID("ws-1")},
			group: &models.Group{
				FullPath: "ns1",
			},
			workspace: &models.Workspace{
				FullPath: "ns1/ws1",
			},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "access denied because permission is not satisfied",
			perms:       []models.Permission{models.ViewGroupPermission, models.CreateGroupPermission},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1",
			},
			customRolePerms: []models.Permission{models.ViewGroupPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "access denied because no permissions are specified",
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "access denied because required constraints are missing",
			perms:           []models.Permission{models.ViewWorkspacePermission},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockGroups := db.NewMockGroups(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			checks := getConstraints(test.constraints...)

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
			if checks.groupID != nil {
				mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Sort:              &sortBy,
					PaginationOptions: nil,
					Filter: &db.NamespaceMembershipFilter{
						UserID:         &userID,
						NamespacePaths: expandNamespaceDescOrder(test.group.FullPath),
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			if checks.workspaceID != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(test.workspace, nil)

				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Sort:              &sortBy,
					PaginationOptions: nil,
					Filter: &db.NamespaceMembershipFilter{
						UserID:         &userID,
						NamespacePaths: expandNamespaceDescOrder(test.workspace.FullPath),
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{Metadata: models.ResourceMetadata{ID: nm.RoleID}}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil)
				}
			}

			dbClient := db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
				Groups:               mockGroups,
				Roles:                mockRoles,
				Workspaces:           mockWorkspaces,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireAccess(WithCaller(ctx, mockCaller), test.perms, test.constraints...)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireInheritedAccess(t *testing.T) {
	userID := "user-1"
	customRoleID := "custom-role-1"
	groupID := "group-1"

	tests := []struct {
		name                 string
		expectErrorCode      errors.CodeType
		modelTypes           []types.ModelType
		customRolePerms      []models.Permission
		group                *models.Group
		workspace            *models.Workspace
		namespaceMemberships []models.NamespaceMembership
		constraints          []func(*constraints)
	}{
		{
			name:        "user access is granted",
			modelTypes:  []types.ModelType{types.ManagedIdentityModelType, types.RunnerModelType},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "user with custom role is granted access",
			modelTypes:  []types.ModelType{types.ManagedIdentityModelType},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			customRolePerms: []models.Permission{models.CreateManagedIdentityPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "multiple permissions and constraints are granted access",
			modelTypes:  []types.ModelType{types.RunnerModelType, types.VCSProviderModelType},
			constraints: []func(*constraints){WithGroupID(groupID), WithNamespacePath("ns1/na/ws-1")},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			workspace: &models.Workspace{
				FullPath: "ns1/na/ws-1",
			},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "access denied because permission is not satisfied",
			modelTypes:  []types.ModelType{types.TerraformModuleModelType, types.TerraformProviderModelType},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			customRolePerms: []models.Permission{models.ViewTerraformModulePermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because no permissions are specified",
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "access denied because required constraints are missing",
			modelTypes:      []types.ModelType{types.ServiceAccountModelType},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockGroups := db.NewMockGroups(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			checks := getConstraints(test.constraints...)

			if checks.groupID != nil {
				mockGroups.On("GetGroupByID", mock.Anything, *checks.groupID).Return(test.group, nil)

				namespaceParts := strings.Split(test.group.FullPath, "/")
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						UserID:              &userID,
						NamespacePathPrefix: &namespaceParts[0],
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			for _, np := range checks.namespacePaths {
				namespaceParts := strings.Split(np, "/")
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						UserID:              &userID,
						NamespacePathPrefix: &namespaceParts[0],
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil)
				}
			}

			dbClient := db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
				Groups:               mockGroups,
				Roles:                mockRoles,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireAccessToInheritableResource(WithCaller(ctx, mockCaller), test.modelTypes, test.constraints...)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToGroup(t *testing.T) {
	userID := "user1"
	groupID := "group1"
	customRoleID := "role1"

	// Test cases
	tests := []struct {
		customRolePerms      []models.Permission
		group                *models.Group
		name                 string
		requiredPermission   *models.Permission
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user does not have required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "group does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.ENotFound,
		},
		{
			name: "user with custom role has viewer permission available through a greater permission action",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			customRolePerms:    []models.Permission{models.UpdateGroupPermission},
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user with custom role has required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			customRolePerms:    []models.Permission{models.CreateGPGKeyPermission},
			requiredPermission: &models.CreateGPGKeyPermission,
		},
		{
			name: "user with custom role does not have required permissions",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			customRolePerms: []models.Permission{
				models.UpdateManagedIdentityPermission,
				models.CreateRunPermission,
			},
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "custom role does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.ViewGPGKeyPermission,
			expectErrorCode:    errors.ENotFound,
		},
		{
			name: "need CreateGPGKeyPermission, have dov: multiple namespaces, ensure lowest namespace membership wins",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1/ns2/ns11"}}, // This should win.
				{RoleID: models.DeployerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
				{RoleID: models.OwnerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns2/ns11",
			},
			requiredPermission: &models.CreateGPGKeyPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "need CreateManagedIdentityPermission, have do: multiple namespaces, ensure lowest membership wins",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
				{RoleID: models.OwnerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "need UpdateManagedIdentityPermission, have od: multiple namespaces, ensure lowest membership wins",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
				{RoleID: models.DeployerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.UpdateManagedIdentityPermission,
		},
		// Need CreateGroupPermission, have 2 namespaces, ensure lowest membership wins.
		{
			name: "need CreateGroupPermission, have dv: multiple namespaces, ensure lowest membership wins",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateGroupPermission,
		},
		{
			name: "negative: need UpdateManagedIdentityPermission, have deployer",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.UpdateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "negative: need CreateGroupPermission, have viewer",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
		// Ensure higher permission can't be granted if they only have View action.
		{
			name: "need CreateGroupPermission, have ViewGroupPermission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockGroups := db.NewMockGroups(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			if test.group != nil {
				sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Sort:              &sortBy,
					PaginationOptions: nil,
					Filter: &db.NamespaceMembershipFilter{
						UserID:         &userID,
						NamespacePaths: expandNamespaceDescOrder(test.group.FullPath),
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil).Maybe() // Depending on the order this may never be called for a test case.
				}
			}

			mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

			dbClient := db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
				Groups:               mockGroups,
				Roles:                mockRoles,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.requireAccessToGroup(WithCaller(ctx, mockCaller), groupID, test.requiredPermission)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToWorkspace(t *testing.T) {
	userID := "user1"
	workspaceID := "ws1"
	customRoleID := "role1"

	// Test cases
	tests := []struct {
		workspace            *models.Workspace
		name                 string
		customRolePerms      []models.Permission
		requiredPermission   *models.Permission
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required permissions",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user does not have required permissions",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "user with custom role has required permissions",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			customRolePerms: []models.Permission{
				models.ViewGroupPermission,
				models.CreateManagedIdentityPermission,
			},
			requiredPermission: &models.CreateManagedIdentityPermission,
		},
		{
			name: "user with custom role does not have required permissions",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			customRolePerms: []models.Permission{
				models.ViewGroupPermission,
				models.CreateManagedIdentityPermission,
			},
			requiredPermission: &models.UpdateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "user with custom role has viewer access from a greater action",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			customRolePerms: []models.Permission{
				models.CreateManagedIdentityPermission,
			},
			requiredPermission: &models.ViewManagedIdentityPermission,
		},
		{
			name: "custom role does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.UpdateManagedIdentityPermission,
			expectErrorCode:    errors.ENotFound,
		},
		{
			name: "workspace does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermission: &models.ViewWorkspacePermission,
			expectErrorCode:    errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			if test.workspace != nil {
				sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Sort:              &sortBy,
					PaginationOptions: nil,
					Filter: &db.NamespaceMembershipFilter{
						UserID:         &userID,
						NamespacePaths: expandNamespaceDescOrder(test.workspace.FullPath),
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(test.workspace, nil)

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, customRoleID).Return(role, nil)
				}
			}

			dbClient := db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
				Workspaces:           mockWorkspaces,
				Roles:                mockRoles,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.requireAccessToWorkspace(WithCaller(ctx, mockCaller), workspaceID, test.requiredPermission)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToNamespace(t *testing.T) {
	userID := "user1"
	serviceAccountID := "serviceAccount1"
	customRoleID := "custom-role-1"

	// Test cases
	tests := []struct {
		userID               *string
		serviceAccountID     *string
		name                 string
		requiredNamespace    string
		expectErrorCode      errors.CodeType
		customRolePerms      []models.Permission
		requiredPermission   *models.Permission
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user has required permission in parent namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user with custom role has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			customRolePerms:    []models.Permission{models.CreateGroupPermission},
			requiredPermission: &models.CreateGroupPermission,
		},
		{
			name: "user with custom role has required permission in parent namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []models.Permission{models.CreateGroupPermission},
			requiredPermission: &models.CreateGroupPermission,
		},
		{
			name: "user with custom role can view a resource because of a higher permission action",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []models.Permission{models.CreateGroupPermission},
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user with custom role does not have required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []models.Permission{models.ViewGroupPermission},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "service account has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			serviceAccountID:   &serviceAccountID,
			requiredNamespace:  "ns1",
			requiredPermission: &models.ViewGPGKeyPermission,
		},
		{
			name:                 "user doesn't have any namespace memberships",
			namespaceMemberships: []models.NamespaceMembership{},
			userID:               &userID,
			requiredNamespace:    "ns1",
			requiredPermission:   &models.ViewGroupPermission,
			expectErrorCode:      errors.ENotFound,
		},
		{
			name: "user has lower access level than required",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "user has lower access level than required in nested group",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "custom role does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil)
				}
			}

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort:              &sortBy,
				PaginationOptions: nil,
				Filter: &db.NamespaceMembershipFilter{
					UserID:           test.userID,
					ServiceAccountID: test.serviceAccountID,
					NamespacePaths:   expandNamespaceDescOrder(test.requiredNamespace),
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				Roles:                mockRoles,
				NamespaceMemberships: mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, test.userID, test.serviceAccountID, false)

			err := authorizer.requireAccessToNamespace(WithCaller(ctx, mockCaller), test.requiredNamespace, test.requiredPermission)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToNamespaces(t *testing.T) {
	userID := "user1"
	customRoleID := "custom-role-1"

	// Test cases
	tests := []struct {
		name                 string
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
		customRolePerms      []models.Permission
		requiredPermission   *models.Permission
		requiredNamespaces   []string
	}{
		{
			name: "user has permissions for namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222"},
			requiredPermission: &models.ViewGPGKeyPermission,
		},
		{
			name: "user does not have permissions for all namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "user with custom role has permissions for namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			customRolePerms: []models.Permission{
				models.CreateGroupPermission, // View should be granted since the action here is greater.
				models.ViewWorkspacePermission,
			},
			requiredPermission: &models.ViewGroupPermission,
		},
		{
			name: "user with custom role does not have permissions for all namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			customRolePerms: []models.Permission{
				models.CreateGroupPermission, // View should be granted since the action here is greater.
				models.ViewWorkspacePermission,
			},
			requiredPermission: &models.CreateManagedIdentityPermission,
			expectErrorCode:    errors.ENotFound,
		},
		{
			name:               "user does not have access to any namespaces",
			requiredNamespaces: []string{"ns3"},
			expectErrorCode:    errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil)
				}
			}

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
			for _, rn := range test.requiredNamespaces {
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Sort: &sortBy,
					Filter: &db.NamespaceMembershipFilter{
						UserID:         &userID,
						NamespacePaths: expandNamespaceDescOrder(rn),
					},
				}

				var memberships []models.NamespaceMembership
				if strings.HasPrefix(rn, "ns1") {
					memberships = append(memberships, test.namespaceMemberships[0])
				} else if strings.HasPrefix(rn, "ns2") {
					memberships = append(memberships, test.namespaceMemberships[1])
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: memberships,
				}, nil).Maybe()
			}

			dbClient := db.Client{
				Roles:                mockRoles,
				NamespaceMemberships: mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.requireAccessToNamespaces(WithCaller(ctx, mockCaller), test.requiredNamespaces, test.requiredPermission)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToInheritedGroupResource(t *testing.T) {
	userID := "user1"
	groupID := "group1"
	customRoleID := "custom-role-1"

	// Test cases
	tests := []struct {
		group                *models.Group
		name                 string
		expectErrorCode      errors.CodeType
		customRolePerms      []models.Permission
		requiredPermission   *models.Permission
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required permission for top-level group",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1",
			},
			requiredPermission: &models.ViewGPGKeyPermission,
		},
		{
			name: "user has required permission for nested group",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.ViewManagedIdentityPermission,
		},
		{
			name:                 "user does not have any namespace memberships",
			namespaceMemberships: []models.NamespaceMembership{},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "user does not have a membership in requested namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns20",
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "user does not have required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateGroupPermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "user with custom role has required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			customRolePerms:    []models.Permission{models.CreateTerraformModulePermission},
			requiredPermission: &models.CreateTerraformModulePermission,
		},
		{
			name: "user with custom role does not have required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			customRolePerms:    []models.Permission{models.UpdateTerraformModulePermission},
			requiredPermission: &models.CreateTerraformModulePermission,
			expectErrorCode:    errors.EForbidden,
		},
		{
			name: "custom role does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredPermission: &models.CreateServiceAccountPermission,
			expectErrorCode:    errors.ENotFound,
		},
		{
			name:               "group does not exist",
			requiredPermission: &models.CreateServiceAccountPermission,
			expectErrorCode:    errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockGroups := db.NewMockGroups(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()
			mockCaller.On("UnauthorizedError", mock.Anything, mock.Anything).Return(func(_ context.Context, hasViewerAccess bool) error {
				if hasViewerAccess {
					return errors.New("forbidden", errors.WithErrorCode(errors.EForbidden))
				}
				return errors.New("not found", errors.WithErrorCode(errors.ENotFound))
			}).Maybe()

			for _, nm := range test.namespaceMemberships {
				if nm.RoleID == customRoleID {
					role := &models.Role{}
					role.SetPermissions(test.customRolePerms)
					mockRoles.On("GetRoleByID", mock.Anything, nm.RoleID).Return(role, nil)
				}
			}

			if test.group != nil {
				namespaceParts := strings.Split(test.group.FullPath, "/")
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						UserID:              &userID,
						NamespacePathPrefix: &namespaceParts[0],
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

			dbClient := db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
				Groups:               mockGroups,
				Roles:                mockRoles,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.requireAccessToInheritedGroupResource(WithCaller(ctx, mockCaller), groupID, test.requiredPermission)
			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckCache(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		key                  cacheKey
		name                 string
		requiredPermissions  *models.Permission
		namespaceMemberships []models.NamespaceMembership
		expectCacheHit       bool
	}{
		{
			name: "cache hit on top level namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2"}},
			},
			requiredPermissions: &models.ViewGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on multiple memberships in the same namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermissions: &models.CreateGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on nested namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &models.CreateManagedIdentityPermission,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      true,
		},
		{
			name: "missing required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermissions: &models.CreateGPGKeyPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because namespace membership is for nested group only",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &models.ViewGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because nested namespace membership reduces scope of parent namespace membership",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &models.CreateGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      false,
		},
		{
			name:                 "cache miss because cache is empty",
			namespaceMemberships: []models.NamespaceMembership{},
			requiredPermissions:  &models.ViewGroupPermission,
			key:                  cacheKey{path: ptr.String("ns1")},
			expectCacheHit:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRoles := db.NewMockRoles(t)

			dbClient := db.Client{
				Roles: mockRoles,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, true)

			// Add all membership permissions to cache.
			for _, nm := range test.namespaceMemberships {
				nmCopy := nm
				_, _ = authorizer.getPermissionsFromMembership(ctx, &nmCopy)
			}

			cacheHit := authorizer.checkCache(&test.key, test.requiredPermissions)
			assert.Equal(t, test.expectCacheHit, cacheHit)
		})
	}
}
