package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestServiceAccountCaller_GetSubject(t *testing.T) {
	caller := ServiceAccountCaller{ServiceAccountPath: "group/service-account"}
	assert.Equal(t, "group/service-account", caller.GetSubject())
}

func TestServiceAccountCaller_IsAdmin(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.False(t, caller.IsAdmin())
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
		name              string
		expectErrorCode   errors.CodeType
		perm              models.Permission
		constraints       []func(*constraints)
		withAuthorizer    bool
		inMaintenanceMode bool
	}{
		{
			name:           "access is granted by the authorizer",
			perm:           models.ViewGroupPermission,
			constraints:    []func(*constraints){WithGroupID("group-1")},
			withAuthorizer: true,
		},
		{
			name:            "access is denied by the authorizer",
			perm:            models.CreateWorkspacePermission,
			constraints:     []func(*constraints){WithWorkspaceID("ws-1")},
			expectErrorCode: errors.ENotFound,
			withAuthorizer:  true,
		},
		{
			name:            "access denied because required constraints are not specified",
			perm:            models.ViewWorkspacePermission,
			expectErrorCode: errors.EInternal,
			withAuthorizer:  true,
		},
		{
			name:              "cannot have write permission when system is in maintenance mode",
			perm:              models.CreateWorkspacePermission,
			expectErrorCode:   errors.EServiceUnavailable,
			inMaintenanceMode: true,
		},
		{
			name:              "can have read permission when system is in maintenance mode",
			perm:              models.ViewWorkspacePermission,
			constraints:       []func(*constraints){WithWorkspaceID("ws-1")},
			withAuthorizer:    true,
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{test.perm}, mock.Anything).Return(requireAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer
			caller.maintenanceMonitor = mockMaintenanceMonitor

			err := caller.RequirePermission(ctx, test.perm, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestServiceAccountCaller_RequireInheritedAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		name            string
		expectErrorCode errors.CodeType
		modelType       types.ModelType
		constraints     []func(*constraints)
		withAuthorizer  bool
	}{
		{
			name:           "access is granted by the authorizer",
			modelType:      types.ManagedIdentityModelType,
			constraints:    []func(*constraints){WithGroupID("group-1")},
			withAuthorizer: true,
		},
		{
			name:            "access is denied by the authorizer",
			modelType:       types.ApplyModelType,
			constraints:     []func(*constraints){WithGroupID("group-1")},
			expectErrorCode: errors.ENotFound,
			withAuthorizer:  true,
		},
		{
			name:            "access denied because required constraints are not specified",
			modelType:       types.RunnerModelType,
			expectErrorCode: errors.EInternal,
			withAuthorizer:  true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockAuthorizer := NewMockAuthorizer(t)

			if test.withAuthorizer {
				mockAuthorizer.On("RequireAccessToInheritableResource", mock.Anything, []types.ModelType{test.modelType}, mock.Anything).Return(requireInheritedAccessAuthorizerFunc)
			}

			caller.authorizer = mockAuthorizer

			err := caller.RequireAccessToInheritableResource(ctx, test.modelType, test.constraints...)
			if test.expectErrorCode != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.Nil(t, err)
		})
	}
}

// requireAccessAuthorizerFunc is a helper function to mock the results returned from the Authorizer interface.
func requireAccessAuthorizerFunc(_ context.Context, perms []models.Permission, checks ...func(*constraints)) error {
	if len(perms) == 0 || len(checks) == 0 {
		return errMissingConstraints
	}

	for _, perm := range perms {
		if perm.Action != models.ViewAction {
			// Only grant viewer permissions for the sake of making testing easier.
			return errors.New("unauthorized", errors.WithErrorCode(errors.ENotFound))
		}
	}

	return nil
}

// requireInheritedAccessAuthorizerFunc is a helper function to mock the results returned from the Authorizer interface.
func requireInheritedAccessAuthorizerFunc(_ context.Context, modelTypes []types.ModelType, checks ...func(*constraints)) error {
	if len(modelTypes) == 0 || len(checks) == 0 {
		return errMissingConstraints
	}

	for _, mt := range modelTypes {
		if mt.Equals(types.ApplyModelType) {
			// Don't allow access to apply resource for sake of making testing easier.
			return errors.New("unauthorized", errors.WithErrorCode(errors.ENotFound))
		}
	}

	return nil
}
