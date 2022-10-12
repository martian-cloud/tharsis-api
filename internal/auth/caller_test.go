package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemCaller_RequireTeamCreateAccess(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller)))
}

func TestSystemCaller_RequireTeamUpdateAccess(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireTeamUpdateAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

func TestSystemCaller_RequireTeamDeleteAccess(t *testing.T) {
	caller := SystemCaller{}
	assert.Nil(t, caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-team-id"))
}

// The End.
