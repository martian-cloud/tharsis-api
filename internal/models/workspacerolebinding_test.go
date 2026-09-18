package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestWorkspaceRoleBindingValidate(t *testing.T) {
	tests := []struct {
		name            string
		binding         WorkspaceRoleBinding
		expectErrorCode errors.CodeType
	}{
		{
			name:    "valid binding",
			binding: WorkspaceRoleBinding{WorkspaceID: "ws-1", RoleID: "role-1"},
		},
		{
			name:            "missing workspace ID",
			binding:         WorkspaceRoleBinding{RoleID: "role-1"},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "missing role ID",
			binding:         WorkspaceRoleBinding{WorkspaceID: "ws-1"},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bindingCopy := test.binding
			err := bindingCopy.Validate()

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestWorkspaceRoleBindingModelInterface(t *testing.T) {
	// ParseGlobalID validates that the embedded ID is a UUID, so use a real one.
	bindingID := "8d1c1f4e-0f1e-4a3b-9c2d-6f5a4b3c2d1e"

	binding := &WorkspaceRoleBinding{
		Metadata:    ResourceMetadata{ID: bindingID},
		WorkspaceID: "ws-1",
		RoleID:      "role-1",
	}

	assert.Equal(t, bindingID, binding.GetID())
	assert.Equal(t, types.WorkspaceRoleBindingModelType, binding.GetModelType())
	assert.NotEmpty(t, binding.GetGlobalID())

	// The GID must carry this model's own code so a binding GID cannot be mistaken for a
	// workspace GID, and must round-trip back to the underlying ID.
	parsed, err := gid.ParseGlobalID(binding.GetGlobalID())
	require.NoError(t, err)
	assert.Equal(t, types.WorkspaceRoleBindingModelType.GIDCode(), parsed.Code)
	assert.Equal(t, bindingID, gid.FromGlobalID(binding.GetGlobalID()))

	id, err := binding.ResolveMetadata("id")
	assert.NoError(t, err)
	assert.Equal(t, bindingID, *id)
}
