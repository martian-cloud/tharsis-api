package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestRegisterPermission(t *testing.T) {
	fakeModelType := types.ModelType{}

	t.Run("adds permission to registry", func(t *testing.T) {
		p := registerPermission(fakeModelType, "test_action_1", false)

		assert.Equal(t, fakeModelType.Name(), p.ResourceType)
		assert.Equal(t, Action("test_action_1"), p.Action)

		_, exists := permissionRegistry[p.String()]
		assert.True(t, exists)
	})

	t.Run("adds assignable permission to assignable map", func(t *testing.T) {
		p := registerPermission(fakeModelType, "test_action_2", true)

		_, exists := assignablePermissions[p]
		assert.True(t, exists)
	})

	t.Run("does not add non-assignable permission to assignable map", func(t *testing.T) {
		p := registerPermission(fakeModelType, "test_action_3", false)

		_, exists := assignablePermissions[p]
		assert.False(t, exists)
	})

	t.Run("panics on duplicate", func(t *testing.T) {
		assert.Panics(t, func() {
			registerPermission(fakeModelType, "test_action_1", true)
		})
	})

	t.Run("registry is populated", func(t *testing.T) {
		assert.NotEmpty(t, permissionRegistry)
		assert.NotEmpty(t, assignablePermissions)
	})
}

func TestHasViewerAccess(t *testing.T) {
	actions := []Action{
		ViewAction,
		CreateAction,
		UpdateAction,
		DeleteAction,
		ViewValueAction,
		IssueTokenAction,
	}

	// Positive.
	for _, action := range actions {
		assert.True(t, action.HasViewerAccess())
	}

	// Negative
	assert.False(t, Action("other").HasViewerAccess())
}

func TestString(t *testing.T) {
	assert.Equal(t, "gpg_key:view", ViewGPGKeyPermission.String())
}

func TestGTE(t *testing.T) {
	testCases := []struct {
		have   *Permission
		want   *Permission
		name   string
		expect bool
	}{
		{
			name:   "permissions are equal",
			have:   &CreateGPGKeyPermission,
			want:   &CreateGPGKeyPermission,
			expect: true,
		},
		{
			name:   "permissions have different actions",
			have:   &CreateGPGKeyPermission,
			want:   &DeleteGPGKeyPermission,
			expect: false,
		},
		{
			name:   "permissions are for different resource types",
			have:   &CreateGroupPermission,
			want:   &CreateWorkspacePermission,
			expect: false,
		},
		{
			name:   "permission grants view action",
			have:   &UpdateGroupPermission,
			want:   &ViewGroupPermission,
			expect: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, test.have.GTE(test.want))
		})
	}
}

func TestIsAssignable(t *testing.T) {
	// Positive
	for perm := range assignablePermissions {
		assert.True(t, perm.IsAssignable())
	}
	// Negative
	assert.False(t, CreateTeamPermission.IsAssignable())
}

func TestGetAssignablePermissions(t *testing.T) {
	assert.Equal(t, len(assignablePermissions), len(GetAssignablePermissions()))
}

func TestParsePermissions(t *testing.T) {
	testCases := []struct {
		name              string
		expectErrorCode   errors.CodeType
		input             []string
		expectPermissions []Permission
	}{
		{
			name: "successfully parse permissions",
			input: []string{
				"group:create",
				"gpg_key:view",
				"workspace : create",
				"  workspace : view  ", // Should parse these properly as well.
			},
			expectPermissions: []Permission{
				CreateGroupPermission,
				ViewGPGKeyPermission,
				CreateWorkspacePermission,
				ViewWorkspacePermission,
			},
		},
		{
			name: "permissions are not proper format",
			input: []string{
				"gpg_key.view",
				"invalid",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "permissions are empty",
			input: []string{
				"  ",
				"",
				"group:view",
			},
			expectPermissions: []Permission{
				ViewGroupPermission,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualParsed, err := ParsePermissions(test.input)

			assert.Equal(t, test.expectPermissions, actualParsed)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
