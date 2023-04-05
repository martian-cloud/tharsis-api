package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
)

func TestSystemCaller_GetSubject(t *testing.T) {
	caller := SystemCaller{}
	assert.Equal(t, "system", caller.GetSubject())
}

func TestSystemCaller_GetNamespaceAccessPolicy(t *testing.T) {
	caller := SystemCaller{}
	policy, err := caller.GetNamespaceAccessPolicy(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, &NamespaceAccessPolicy{AllowAll: true}, policy)
}

func TestSystemCaller_RequirePermissions(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequirePermission(WithCaller(context.Background(), &caller), permissions.Permission{}, nil))
}

func TestSystemCaller_RequireInheritedPermissions(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireAccessToInheritableResource(WithCaller(context.Background(), &caller), permissions.RunResourceType, nil))
}

// The End.
