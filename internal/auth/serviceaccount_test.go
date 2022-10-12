package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

func TestServiceAccountCaller_RequireRunWriteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	err := caller.RequireRunWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestServiceAccountCaller_RequirePlanWriteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	err := caller.RequirePlanWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestServiceAccountCaller_RequireApplyWriteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	err := caller.RequireApplyWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestServiceAccountCaller_RequireJobWriteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	err := caller.RequireJobWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestServiceAccountCaller_RequireTeamCreateAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestServiceAccountCaller_RequireTeamUpdateAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireTeamUpdateAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

func TestServiceAccountCaller_RequireTeamDeleteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

func TestServiceAccountCaller_RequireUserCreateAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestServiceAccountCaller_RequireUserUpdateAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "a-fake-user-id"))
}

func TestServiceAccountCaller_RequireUserDeleteAccess(t *testing.T) {
	caller := ServiceAccountCaller{}
	assert.NotNil(t, caller.RequireUserDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-user-id"))
}
