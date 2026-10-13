package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSCIMCaller_GetSubject(t *testing.T) {
	caller := SCIMCaller{}
	assert.Equal(t, "scim", caller.GetSubject())
}

func TestSCIMCaller_IsAdmin(t *testing.T) {
	caller := SCIMCaller{}
	assert.False(t, caller.IsAdminModeActivated(t.Context()))
}

func TestSCIMCaller_GetNamespacePermissions(t *testing.T) {
	caller := SCIMCaller{}
	perms, err := caller.GetNamespacePermissions(context.Background(), "some/namespace")
	assert.NoError(t, err)
	// Not backed by namespace memberships, so it always reports an empty set.
	assert.Empty(t, perms)
}

func TestSCIMCaller_GetRootNamespaceMemberships(t *testing.T) {
	caller := SCIMCaller{}
	namespaces, err := caller.GetRootNamespaceMemberships(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
	// Must be a non-nil empty slice: a nil slice is treated as "no filter" by the membership
	// filter and would expose all resources. A SCIM caller must deny by default.
	assert.NotNil(t, namespaces)
	assert.Empty(t, namespaces)
}

func TestSCIMCaller_RequirePermissions(t *testing.T) {
	caller := SCIMCaller{idpIssuerURL: "https://idp.example.com"}
	ctx := WithCaller(context.Background(), &caller)

	testCases := []struct {
		expectErrorCode   errors.CodeType
		name              string
		perms             []models.Permission
		inMaintenanceMode bool
	}{
		{
			// The SCIM service checks that the caller's IDP manages the user or team.
			name: "managing SCIM users and teams; grant access",
			perms: []models.Permission{
				models.ManageSCIMTeamPermission,
				models.ManageSCIMUserPermission,
			},
		},
		{
			// The team and user services don't limit a SCIM caller to its IDP's resources, so the
			// generic permissions must never be granted.
			name: "generic team and user permissions; access denied",
			perms: []models.Permission{
				models.CreateTeamPermission,
				models.UpdateTeamPermission,
				models.DeleteTeamPermission,
				models.CreateUserPermission,
				models.UpdateUserPermission,
				models.DeleteUserPermission,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "access denied because permission is never available to caller",
			perms:           []models.Permission{models.CreateGroupPermission},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:              "cannot have write permission when server in maintenance mode",
			perms:             []models.Permission{models.ManageSCIMTeamPermission, models.ManageSCIMUserPermission},
			expectErrorCode:   errors.EServiceUnavailable,
			inMaintenanceMode: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(test.inMaintenanceMode, nil)

			caller.maintenanceMonitor = mockMaintenanceMonitor

			for _, perm := range test.perms {
				err := caller.RequirePermission(ctx, perm)
				if test.expectErrorCode != "" {
					require.NotNil(t, err, perm.String())
					assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err), perm.String())
					continue
				}
				require.Nil(t, err, perm.String())
			}
		})
	}
}

func TestSCIMCaller_RequireInheritedPermissions(t *testing.T) {
	caller := SCIMCaller{}
	err := caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), types.RunnerModelType, nil)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestSCIMCaller_Manages(t *testing.T) {
	issuer := "https://idp.example.com"

	testCases := []struct {
		name         string
		callerIssuer string
		objectIssuer string
		expect       bool
	}{
		{name: "object managed by the caller's IDP", callerIssuer: issuer, objectIssuer: issuer, expect: true},
		{name: "object managed by another IDP", callerIssuer: issuer, objectIssuer: "https://other-idp.example.com"},
		{name: "object not managed by SCIM", callerIssuer: issuer},
		// A caller without an issuer must never match unmanaged objects, whose issuer is also empty.
		{name: "caller without an issuer and an unmanaged object", callerIssuer: ""},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			caller := SCIMCaller{idpIssuerURL: test.callerIssuer}
			assert.Equal(t, test.expect, caller.Manages(test.objectIssuer))
		})
	}
}
