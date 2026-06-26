package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

func TestSystemCaller_GetSubject(t *testing.T) {
	caller := SystemCaller{}
	assert.Equal(t, "system", caller.GetSubject())
}

func TestSystemCaller_GetRootNamespaceMemberships(t *testing.T) {
	caller := SystemCaller{}
	namespaces, err := caller.GetRootNamespaceMemberships(context.Background())
	assert.Nil(t, err)
	// The system caller is always an admin (IsAdminModeActivated returns true), so this is never
	// used for filtering; it returns nil and is gated on IsAdminModeActivated by consumers.
	assert.Nil(t, namespaces)
}

func TestSystemCaller_RequirePermissions(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequirePermission(WithCaller(context.Background(), &caller), models.Permission{}, nil))
}

func TestSystemCaller_RequireInheritedPermissions(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), types.RunModelType, nil))
}

func TestSystemCaller_RequireRole(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireRole(t.Context(), models.OwnerRoleID.String()))
}
