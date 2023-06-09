package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestServiceAccountCaller_GetSubject(t *testing.T) {
	caller := ServiceAccountCaller{ServiceAccountPath: "group/service-account"}
	assert.Equal(t, "group/service-account", caller.GetSubject())
}

func TestServiceAccountCaller_GetNamespaceAccessPolicy(t *testing.T) {
	membershipNamespaceID := "nm-1"

	mockAuthorizer := NewMockAuthorizer(t)
	mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{{ID: membershipNamespaceID}}, nil)

	caller := ServiceAccountCaller{authorizer: mockAuthorizer}
	policy, err := caller.GetNamespaceAccessPolicy(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
	assert.Equal(t, &NamespaceAccessPolicy{AllowAll: false, RootNamespaceIDs: []string{membershipNamespaceID}}, policy)
}

func TestServiceAccountCaller_RequirePermissions(t *testing.T) {
	caller := ServiceAccountCaller{}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name           string
		expect         error
		perm           permissions.Permission
		constraints    []func(*constraints)
		withAuthorizer bool
	}{
		{
			name:           "access is granted by the authorizer",
			perm:           permissions.ViewGroupPermission,
			constraints:    []func(*constraints){WithGroupID("group-1")},
			withAuthorizer: true,
		},
		{
			name:           "access is denied by the authorizer",
			perm:           permissions.CreateWorkspacePermission,
			constraints:    []func(*constraints){WithWorkspaceID("ws-1")},
			expect:         authorizationError(ctx, false),
			withAuthorizer: true,
		},
		{
			name:           "access denied because required constraints are not specified",
			perm:           permissions.ViewWorkspacePermission,
			expect:         errMissingConstraints,
			withAuthorizer: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{test.perm}, mock.Anything).Return(requireAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer

			assert.Equal(t, test.expect, caller.RequirePermission(ctx, test.perm, test.constraints...))
		})
	}
}

func TestServiceAccountCaller_RequireInheritedAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name           string
		expect         error
		resourceType   permissions.ResourceType
		constraints    []func(*constraints)
		withAuthorizer bool
	}{
		{
			name:           "access is granted by the authorizer",
			resourceType:   permissions.ManagedIdentityResourceType,
			constraints:    []func(*constraints){WithGroupID("group-1")},
			withAuthorizer: true,
		},
		{
			name:           "access is denied by the authorizer",
			resourceType:   permissions.ApplyResourceType,
			constraints:    []func(*constraints){WithGroupID("group-1")},
			expect:         authorizationError(ctx, false),
			withAuthorizer: true,
		},
		{
			name:           "access denied because required constraints are not specified",
			resourceType:   permissions.RunnerResourceType,
			expect:         errMissingConstraints,
			withAuthorizer: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccessToInheritableResource", mock.Anything, []permissions.ResourceType{test.resourceType}, mock.Anything).Return(requireInheritedAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer

			assert.Equal(t, test.expect, caller.RequireAccessToInheritableResource(ctx, test.resourceType, test.constraints...))
		})
	}
}

// requireAccessAuthorizerFunc is a helper function to mock the results returned from the Authorizer interface.
func requireAccessAuthorizerFunc(ctx context.Context, perms []permissions.Permission, checks ...func(*constraints)) error {
	if len(perms) == 0 || len(checks) == 0 {
		return errMissingConstraints
	}

	for _, perm := range perms {
		if perm.Action != permissions.ViewAction {
			// Only grant viewer permissions for the sake of making testing easier.
			return authorizationError(ctx, false)
		}
	}

	return nil
}

// requireInheritedAccessAuthorizerFunc is a helper function to mock the results returned from the Authorizer interface.
func requireInheritedAccessAuthorizerFunc(ctx context.Context, resourceTypes []permissions.ResourceType, checks ...func(*constraints)) error {
	if len(resourceTypes) == 0 || len(checks) == 0 {
		return errMissingConstraints
	}

	for _, rt := range resourceTypes {
		if rt == permissions.ApplyResourceType {
			// Don't allow access to apply resource for sake of making testing easier.
			return authorizationError(ctx, false)
		}
	}

	return nil
}
