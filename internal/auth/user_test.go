package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestUserCaller_RequireRunWriteAccess(t *testing.T) {
	caller := UserCaller{}
	err := caller.RequireRunWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestUserCaller_RequirePlanWriteAccess(t *testing.T) {
	caller := UserCaller{}
	err := caller.RequirePlanWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestUserCaller_RequireApplyWriteAccess(t *testing.T) {
	caller := UserCaller{}
	err := caller.RequireApplyWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestUserCaller_RequireJobWriteAccess(t *testing.T) {
	caller := UserCaller{}
	err := caller.RequireJobWriteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestUserCaller_RequireTeamCreateAccess(t *testing.T) {

	// Non-admin case:
	caller := UserCaller{User: &models.User{}}
	err := caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller))
	assert.NotNil(t, err)

	// Admin case:
	caller.User.Admin = true
	err = caller.RequireTeamCreateAccess(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
}

func TestUserCaller_RequireTeamUpdateAccess(t *testing.T) {

	// Preliminary setup:
	caller := UserCaller{User: &models.User{}}
	ctx := WithCaller(context.Background(), &caller)

	// Test cases:
	tests := []struct {
		getError     error
		expect       error
		name         string
		isAdmin      bool
		isMember     bool
		isMaintainer bool
	}{
		{
			name:    "admin",
			isAdmin: true,
		},
		{
			name:     "not admin, get error",
			getError: fmt.Errorf("GetTeamMember mock error"),
			isMember: true,
			expect:   fmt.Errorf("GetTeamMember mock error"),
		},
		{
			name:     "not admin, member, not maintainer",
			isMember: true,
			expect:   authorizationError(ctx, true),
		},
		{
			name:         "not admin, member, is maintainer",
			isMember:     true,
			isMaintainer: true,
		},
		{
			name:   "not admin, not member",
			expect: authorizationError(ctx, true),
		},
	}

	// Run the tests:
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = t

			// Mock out the u.dbClient.TeamMembers.GetTeamMember method.
			mockResult0 := func(ctx context.Context, userID, teamID string) *models.TeamMember {
				_ = ctx
				_ = userID
				_ = teamID

				if test.getError != nil {
					// Return an error trying to get team members.
					return nil
				}

				if !test.isMember {
					// Return that the caller is not a member.
					return nil
				}

				// Return that the caller is a member and maybe a maintainer.
				return &models.TeamMember{IsMaintainer: test.isMaintainer}
			}
			mockResult1 := func(ctx context.Context, userID, teamID string) error {
				_ = ctx
				_ = userID
				_ = teamID

				if test.getError != nil {
					// Return an error trying to get team members.
					return test.getError
				}

				if !test.isMember {
					// Return that the caller is not a member.
					return nil
				}

				// Return that the caller is a member and maybe a maintainer.
				return nil
			}
			mockTeamMembers := db.MockTeamMembers{}
			mockTeamMembers.On("GetTeamMember", mock.Anything, mock.Anything, mock.Anything).Return(mockResult0, mockResult1)
			caller.dbClient = &db.Client{TeamMembers: &mockTeamMembers}

			// Run the test:
			caller.User.Admin = test.isAdmin
			gotErr := caller.RequireTeamUpdateAccess(ctx, "a-fake-team-ID")
			assert.Equal(t, test.expect, gotErr)
		})
	}
}

func TestUserCaller_RequireTeamDeleteAccess(t *testing.T) {

	// Non-admin case:
	caller := UserCaller{User: &models.User{}}
	err := caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-team-id")
	assert.NotNil(t, err)

	// Admin case:
	caller.User.Admin = true
	err = caller.RequireTeamDeleteAccess(WithCaller(context.Background(), &caller), "a-fake-team-id")
	assert.Nil(t, err)
}

func TestUserCaller_RequireUserCreateAccess(t *testing.T) {
	// Non-admin case:
	caller := UserCaller{User: &models.User{}}
	err := caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller))
	assert.NotNil(t, err)

	// Admin case:
	caller.User.Admin = true
	err = caller.RequireUserCreateAccess(WithCaller(context.Background(), &caller))
	assert.Nil(t, err)
}

func TestUserCaller_RequireUserUpdateAccess(t *testing.T) {
	// Non-admin case:
	caller := UserCaller{User: &models.User{}}
	err := caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "1")
	assert.NotNil(t, err)

	// Admin case:
	caller.User.Admin = true
	err = caller.RequireUserUpdateAccess(WithCaller(context.Background(), &caller), "1")
	assert.Nil(t, err)
}

func TestUserCaller_RequireUserDeleteAccess(t *testing.T) {
	// Non-admin case:
	caller := UserCaller{User: &models.User{}}
	err := caller.RequireUserDeleteAccess(WithCaller(context.Background(), &caller), "1")
	assert.NotNil(t, err)

	// Admin case:
	caller.User.Admin = true
	err = caller.RequireUserDeleteAccess(WithCaller(context.Background(), &caller), "1")
	assert.Nil(t, err)
}

// The End.
