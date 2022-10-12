package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

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
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{ID: "ns1", Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{ID: "ns1/ns11", Path: "ns1/ns11"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{ID: "ns2/ns22/ns222", Path: "ns2/ns22/ns222"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{ID: "ns3", Path: "ns3"}},
			},
			expectedRootNamespaces: []models.MembershipNamespace{
				{ID: "ns1", Path: "ns1"},
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

func TestRequireAccessToGroup(t *testing.T) {
	userID := "user1"
	groupID := "group1"

	// Test cases
	tests := []struct {
		group                *models.Group
		name                 string
		requiredAccessLevel  models.Role
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.ViewerRole,
		},
		{
			name: "user does not have required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
		{
			name: "group does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
		// Need owner, have 3 namespaces, ensure highest wins no matter the order.
		{
			name: "need owner, have dov: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have dvo: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have odv: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have ovd: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have vdo: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have vod: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		// Need deployer, have 3 namespaces, ensure highest wins no matter the order.
		{
			name: "need deployer, have dov: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have dvo: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have odv: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have ovd: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have vdo: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have vod: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		// Need owner, have 2 namespaces, ensure highest wins no matter the order.
		{
			name: "need owner, have do: multiple namespaces, ensure highest wins, no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "need owner, have od: multiple namespaces, ensure highest wins, no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
		},
		// Need deployer, have 2 namespaces, ensure highest wins no matter the order.
		{
			name: "need deployer, have dv: multiple namespaces, ensure highest wins, no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		{
			name: "need deployer, have vd: multiple namespaces, ensure highest wins, no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
		},
		// Negative: Need owner, have 2 namespaces, ensure highest wins no matter the order.
		{
			name: "negative: need owner, have dv: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
		{
			name: "negative: need owner, have vd: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.OwnerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
		// Need deployer, have 1 namespace, ensure highest wins no matter the order.
		{
			name: "negative: need deployer, have v: multiple namespaces, ensure highest wins no matter the order",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

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

			mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Groups:               &mockGroups,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireAccessToGroup(WithCaller(ctx, &mockCaller), groupID, test.requiredAccessLevel)
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

	// Test cases
	tests := []struct {
		workspace            *models.Workspace
		name                 string
		requiredAccessLevel  models.Role
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.ViewerRole,
		},
		{
			name: "user does not have required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: workspaceID},
				FullPath: "ns1/ns11",
			},
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
		{
			name: "workspace does not exist",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockWorkspaces := db.MockWorkspaces{}
			mockWorkspaces.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

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

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Workspaces:           &mockWorkspaces,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireAccessToWorkspace(WithCaller(ctx, &mockCaller), workspaceID, test.requiredAccessLevel)
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

	// Test cases
	tests := []struct {
		userID               *string
		serviceAccountID     *string
		name                 string
		requiredNamespace    string
		requiredAccessLevel  models.Role
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required access level in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:              &userID,
			requiredNamespace:   "ns1",
			requiredAccessLevel: models.ViewerRole,
		},
		{
			name: "user has required access level in parent namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns2"}},
			},
			userID:              &userID,
			requiredNamespace:   "ns1/ns2/ns3",
			requiredAccessLevel: models.OwnerRole,
		},
		{
			name: "service account has required access level in namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			serviceAccountID:    &serviceAccountID,
			requiredNamespace:   "ns1",
			requiredAccessLevel: models.ViewerRole,
		},
		{
			name:                 "user doesn't have any namespace memberships",
			namespaceMemberships: []models.NamespaceMembership{},
			userID:               &userID,
			requiredNamespace:    "ns1",
			requiredAccessLevel:  models.ViewerRole,
			expectErrorMsg:       resourceNotFoundErrorMsg,
		},
		{
			name: "user has lower access level than required",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:              &userID,
			requiredNamespace:   "ns1",
			requiredAccessLevel: models.DeployerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
		{
			name: "user has lower access level than required in nested group",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			userID:              &userID,
			requiredNamespace:   "ns1/ns2/ns3",
			requiredAccessLevel: models.OwnerRole,
			expectErrorMsg:      "testsubject is not authorized to perform the requested operation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

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
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, test.userID, test.serviceAccountID, false)

			err := authorizer.RequireAccessToNamespace(WithCaller(ctx, &mockCaller), test.requiredNamespace, test.requiredAccessLevel)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireViewerAccessToGroups(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		name                 string
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
		groups               []models.Group
	}{
		{
			name: "user has viewer access to groups",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
			},
			groups: []models.Group{
				{FullPath: "ns1"},
				{FullPath: "ns2/ns22"},
			},
		},
		{
			name: "user does not have viewer access to all groups",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			groups: []models.Group{
				{FullPath: "ns1"},
				{FullPath: "ns2"},
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathAsc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort: &sortBy,
				Filter: &db.NamespaceMembershipFilter{
					UserID: &userID,
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireViewerAccessToGroups(WithCaller(ctx, &mockCaller), test.groups)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireViewerAccessToWorkspaces(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		name                 string
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
		workspaces           []models.Workspace
	}{
		{
			name: "user has viewer access to workspaces",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
			},
			workspaces: []models.Workspace{
				{FullPath: "ns1"},
				{FullPath: "ns2/ns22"},
			},
		},
		{
			name: "user does not have viewer access to all workspaces",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			workspaces: []models.Workspace{
				{FullPath: "ns1"},
				{FullPath: "ns2"},
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathAsc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort: &sortBy,
				Filter: &db.NamespaceMembershipFilter{
					UserID: &userID,
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireViewerAccessToWorkspaces(WithCaller(ctx, &mockCaller), test.workspaces)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireViewerAccessToNamespaces(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		name                 string
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
		requiredNamespaces   []string
	}{
		{
			name: "user has viewer access to namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222"},
		},
		{
			name: "user does not has viewer access to all namespaces",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns2/ns22"}},
			},
			requiredNamespaces: []string{"ns1", "ns1/ns11", "ns2/ns22/ns222", "ns2"},
			expectErrorMsg:     resourceNotFoundErrorMsg,
		},
		{
			name:                 "user does not have access to any namespaces",
			namespaceMemberships: []models.NamespaceMembership{},
			requiredNamespaces:   []string{"ns1"},
			expectErrorMsg:       resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathAsc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort: &sortBy,
				Filter: &db.NamespaceMembershipFilter{
					UserID: &userID,
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireViewerAccessToNamespaces(WithCaller(ctx, &mockCaller), test.requiredNamespaces)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequireAccessToInheritedGroupResources(t *testing.T) {
	userID := "user1"
	groupID := "group1"

	// Test cases
	tests := []struct {
		group                *models.Group
		name                 string
		expectErrorMsg       string
		namespaceMemberships []models.NamespaceMembership
	}{
		{
			name: "user has required access level to top-level group",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1",
			},
		},
		{
			name: "user has required access level to nested group",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns11/ns111"}},
			},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
		},
		{
			name:                 "user does not have required access level",
			namespaceMemberships: []models.NamespaceMembership{},
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: "ns1/ns11",
			},
			expectErrorMsg: resourceNotFoundErrorMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockCaller := MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("GetSubject").Return("testsubject")

			if test.group != nil {
				getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
					Filter: &db.NamespaceMembershipFilter{
						UserID:              &userID,
						NamespacePathPrefix: &strings.Split(test.group.FullPath, "/")[0],
					},
				}

				mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
					getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
					NamespaceMemberships: test.namespaceMemberships,
				}, nil)
			}

			mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(test.group, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Groups:               &mockGroups,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, false)

			err := authorizer.RequireAccessToInheritedGroupResource(WithCaller(ctx, &mockCaller), groupID)
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
		requiredAccessLevel  models.Role
		namespaceMemberships []models.NamespaceMembership
		expectCacheHit       bool
	}{
		{
			name: "cache hit on top level namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns2"}},
			},
			requiredAccessLevel: models.ViewerRole,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on multiple memberships in the same namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.DeployerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredAccessLevel: models.DeployerRole,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      true,
		},
		{
			name: "cache hit on nested namespace",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredAccessLevel: models.OwnerRole,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      true,
		},
		{
			name: "missing required access level",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
			},
			requiredAccessLevel: models.DeployerRole,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because namespace membership is for nested group only",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredAccessLevel: models.ViewerRole,
			key:                 cacheKey{path: ptr.String("ns1")},
			expectCacheHit:      false,
		},
		{
			name: "cache miss because nested namespace membership reduces scope of parent namespace membership",
			namespaceMemberships: []models.NamespaceMembership{
				{Role: models.OwnerRole, Namespace: models.MembershipNamespace{Path: "ns1"}},
				{Role: models.ViewerRole, Namespace: models.MembershipNamespace{Path: "ns1/ns11"}},
			},
			requiredAccessLevel: models.OwnerRole,
			key:                 cacheKey{path: ptr.String("ns1/ns11")},
			expectCacheHit:      false,
		},
		{
			name:                 "cache miss because cache is empty",
			namespaceMemberships: []models.NamespaceMembership{},
			requiredAccessLevel:  models.ViewerRole,
			key:                  cacheKey{path: ptr.String("ns1")},
			expectCacheHit:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			sortBy := db.NamespaceMembershipSortableFieldNamespacePathDesc
			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Sort: &sortBy,
				Filter: &db.NamespaceMembershipFilter{
					UserID: &userID,
				},
			}

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			authorizer := newNamespaceMembershipAuthorizer(&dbClient, &userID, nil, true)

			_, _ = authorizer.getNamespaceMemberships(ctx, getNamespaceMembershipsInput)

			cacheHit := authorizer.checkCache(&test.key, test.requiredAccessLevel)
			assert.Equal(t, test.expectCacheHit, cacheHit)
		})
	}
}
