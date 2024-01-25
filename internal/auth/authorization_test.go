package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

const forbiddenErrorMsg = "testsubject is not authorized to perform the requested operation"

func TestGetRootNamespaces(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		userID                 *string
		name                   string
		expectErrorMsg         string
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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		expectErrorMsg       string
		perms                []permissions.Permission
		customRolePerms      []permissions.Permission
		group                *models.Group
		workspace            *models.Workspace
		namespaceMemberships []models.NamespaceMembership
		constraints          []func(*constraints)
	}{
		{
			name:        "user access is granted",
			perms:       []permissions.Permission{permissions.ViewGroupPermission, permissions.CreateGroupPermission},
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
			perms:       []permissions.Permission{permissions.ViewGroupPermission, permissions.CreateGroupPermission},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1",
			},
			customRolePerms: []permissions.Permission{permissions.CreateGroupPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:        "multiple permissions and checks are granted access",
			perms:       []permissions.Permission{permissions.ViewGroupPermission, permissions.ViewWorkspacePermission},
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
			perms:       []permissions.Permission{permissions.ViewGroupPermission, permissions.CreateGroupPermission},
			constraints: []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1",
			},
			customRolePerms: []permissions.Permission{permissions.ViewGroupPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			expectErrorMsg: forbiddenErrorMsg,
		},
		{
			name:           "access denied because no permissions are specified",
			expectErrorMsg: errMissingConstraints.Error(),
		},
		{
			name:           "access denied because required constraints are missing",
			perms:          []permissions.Permission{permissions.ViewWorkspacePermission},
			expectErrorMsg: errMissingConstraints.Error(),
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

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		expectErrorMsg       string
		resourceTypes        []permissions.ResourceType
		customRolePerms      []permissions.Permission
		group                *models.Group
		workspace            *models.Workspace
		namespaceMemberships []models.NamespaceMembership
		constraints          []func(*constraints)
	}{
		{
			name:          "user access is granted",
			resourceTypes: []permissions.ResourceType{permissions.ManagedIdentityResourceType, permissions.RunnerResourceType},
			constraints:   []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:          "user with custom role is granted access",
			resourceTypes: []permissions.ResourceType{permissions.ManagedIdentityResourceType},
			constraints:   []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			customRolePerms: []permissions.Permission{permissions.CreateManagedIdentityPermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
		},
		{
			name:          "multiple permissions and constraints are granted access",
			resourceTypes: []permissions.ResourceType{permissions.RunnerResourceType, permissions.VCSProviderResourceType},
			constraints:   []func(*constraints){WithGroupID(groupID), WithNamespacePath("ns1/na/ws-1")},
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
			name:          "access denied because permission is not satisfied",
			resourceTypes: []permissions.ResourceType{permissions.TerraformModuleResourceType, permissions.TerraformProviderResourceType},
			constraints:   []func(*constraints){WithGroupID(groupID)},
			group: &models.Group{
				FullPath: "ns1/na",
			},
			customRolePerms: []permissions.Permission{permissions.ViewTerraformModulePermission},
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
		{
			name:           "access denied because no permissions are specified",
			expectErrorMsg: errMissingConstraints.Error(),
		},
		{
			name:           "access denied because required constraints are missing",
			resourceTypes:  []permissions.ResourceType{permissions.ServiceAccountResourceType},
			expectErrorMsg: errMissingConstraints.Error(),
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

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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

			err := authorizer.RequireAccessToInheritableResource(WithCaller(ctx, mockCaller), test.resourceTypes, test.constraints...)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		customRolePerms      []permissions.Permission
		group                *models.Group
		name                 string
		requiredPermission   *permissions.Permission
		expectErrorMsg       string
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
			requiredPermission: &permissions.ViewGroupPermission,
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
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
		},
		{
			name: "group does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), UserID: &userID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
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
			customRolePerms:    []permissions.Permission{permissions.UpdateGroupPermission},
			requiredPermission: &permissions.ViewGroupPermission,
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
			customRolePerms:    []permissions.Permission{permissions.CreateGPGKeyPermission},
			requiredPermission: &permissions.CreateGPGKeyPermission,
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
			customRolePerms: []permissions.Permission{
				permissions.UpdateManagedIdentityPermission,
				permissions.CreateRunPermission,
			},
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.ViewGPGKeyPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
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
			requiredPermission: &permissions.CreateGPGKeyPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.UpdateManagedIdentityPermission,
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
			requiredPermission: &permissions.CreateGroupPermission,
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
			requiredPermission: &permissions.UpdateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		customRolePerms      []permissions.Permission
		requiredPermission   *permissions.Permission
		expectErrorMsg       string
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
			requiredPermission: &permissions.ViewGroupPermission,
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
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			customRolePerms: []permissions.Permission{
				permissions.ViewGroupPermission,
				permissions.CreateManagedIdentityPermission,
			},
			requiredPermission: &permissions.CreateManagedIdentityPermission,
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
			customRolePerms: []permissions.Permission{
				permissions.ViewGroupPermission,
				permissions.CreateManagedIdentityPermission,
			},
			requiredPermission: &permissions.UpdateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			customRolePerms: []permissions.Permission{
				permissions.CreateManagedIdentityPermission,
			},
			requiredPermission: &permissions.ViewManagedIdentityPermission,
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
			requiredPermission: &permissions.UpdateManagedIdentityPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name: "workspace does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermission: &permissions.ViewWorkspacePermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
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

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		expectErrorMsg       string
		customRolePerms      []permissions.Permission
		requiredPermission   *permissions.Permission
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &permissions.ViewGroupPermission,
		},
		{
			name: "user has required permission in parent namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			requiredPermission: &permissions.ViewGroupPermission,
		},
		{
			name: "user with custom role has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			customRolePerms:    []permissions.Permission{permissions.CreateGroupPermission},
			requiredPermission: &permissions.CreateGroupPermission,
		},
		{
			name: "user with custom role has required permission in parent namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []permissions.Permission{permissions.CreateGroupPermission},
			requiredPermission: &permissions.CreateGroupPermission,
		},
		{
			name: "user with custom role can view a resource because of a higher permission action",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []permissions.Permission{permissions.CreateGroupPermission},
			requiredPermission: &permissions.ViewGroupPermission,
		},
		{
			name: "user with custom role does not have required permission",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			customRolePerms:    []permissions.Permission{permissions.ViewGroupPermission},
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
		},
		{
			name: "service account has required permission in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			serviceAccountID:   &serviceAccountID,
			requiredNamespace:  "ns1",
			requiredPermission: &permissions.ViewGPGKeyPermission,
		},
		{
			name:                 "user doesn't have any namespace memberships",
			namespaceMemberships: []models.NamespaceMembership{},
			userID:               &userID,
			requiredNamespace:    "ns1",
			requiredPermission:   &permissions.ViewGroupPermission,
			expectErrorMsg:       resourceNotFoundErrorMsg,
		},
		{
			name: "user has lower access level than required",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
		},
		{
			name: "user has lower access level than required in nested group",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1/ns2/ns3",
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
		},
		{
			name: "custom role does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:             &userID,
			requiredNamespace:  "ns1",
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
		customRolePerms      []permissions.Permission
		requiredPermission   *permissions.Permission
		requiredNamespaces   []string
	}{
		{
			name: "user has permissions for namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222"},
			requiredPermission: &permissions.ViewGPGKeyPermission,
		},
		{
			name: "user does not have permissions for all namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     forbiddenErrorMsg,
		},
		{
			name: "user with custom role has permissions for namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			customRolePerms: []permissions.Permission{
				permissions.CreateGroupPermission, // View should be granted since the action here is greater.
				permissions.ViewWorkspacePermission,
			},
			requiredPermission: &permissions.ViewGroupPermission,
		},
		{
			name: "user with custom role does not have permissions for all namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: customRoleID, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			customRolePerms: []permissions.Permission{
				permissions.CreateGroupPermission, // View should be granted since the action here is greater.
				permissions.ViewWorkspacePermission,
			},
			requiredPermission: &permissions.CreateManagedIdentityPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name:               "user does not have access to any namespaces",
			requiredNamespaces: []string{"ns3"},
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockCaller := NewMockCaller(t)
			mockRoles := db.NewMockRoles(t)

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		expectErrorMsg       string
		customRolePerms      []permissions.Permission
		requiredPermission   *permissions.Permission
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
			requiredPermission: &permissions.ViewGPGKeyPermission,
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
			requiredPermission: &permissions.ViewManagedIdentityPermission,
		},
		{
			name:                 "user does not have any namespace memberships",
			namespaceMemberships: []models.NamespaceMembership{},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
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
			expectErrorMsg: resourceNotFoundErrorMsg,
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
			requiredPermission: &permissions.CreateGroupPermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			customRolePerms:    []permissions.Permission{permissions.CreateTerraformModulePermission},
			requiredPermission: &permissions.CreateTerraformModulePermission,
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
			customRolePerms:    []permissions.Permission{permissions.UpdateTerraformModulePermission},
			requiredPermission: &permissions.CreateTerraformModulePermission,
			expectErrorMsg:     forbiddenErrorMsg,
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
			requiredPermission: &permissions.CreateServiceAccountPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name:               "group does not exist",
			requiredPermission: &permissions.CreateServiceAccountPermission,
			expectErrorMsg:     resourceNotFoundErrorMsg,
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

			if test.expectErrorMsg == forbiddenErrorMsg {
				mockCaller.On("GetSubject").Return("testsubject")
			}

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
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
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
		requiredPermissions  *permissions.Permission
		namespaceMemberships []models.NamespaceMembership
		expectCacheHit       bool
	}{
		{
			name: "cache hit on top level namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns2"}},
			},
			requiredPermissions: &permissions.ViewGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on multiple memberships in the same namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.DeployerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermissions: &permissions.CreateGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on nested namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &permissions.CreateManagedIdentityPermission,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      true,
		},
		{
			name: "missing required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredPermissions: &permissions.CreateGPGKeyPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because namespace membership is for nested group only",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &permissions.ViewGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because nested namespace membership reduces scope of parent namespace membership",
			namespaceMemberships: []models.NamespaceMembership{
				{RoleID: models.OwnerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1"}},
				{RoleID: models.ViewerRoleID.String(), Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredPermissions: &permissions.CreateGroupPermission,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      false,
		},
		{
			name:                 "cache miss because cache is empty",
			namespaceMemberships: []models.NamespaceMembership{},
			requiredPermissions:  &permissions.ViewGroupPermission,
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
