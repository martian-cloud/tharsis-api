package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMembershipChangeEmail_Subject(t *testing.T) {
	tests := []struct {
		name     string
		email    MembershipChangeEmail
		expected string
	}{
		// Service account variants (workspace).
		{
			name:     "service account created in workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionCreated, ServiceAccountPath: "sa/path", IsWorkspace: true},
			expected: "Service account added to Workspace",
		},
		{
			name:     "service account role changed in workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRoleChanged, ServiceAccountPath: "sa/path", IsWorkspace: true},
			expected: "Service account role updated in Workspace",
		},
		{
			name:     "service account removed from workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved, ServiceAccountPath: "sa/path", IsWorkspace: true},
			expected: "Service account removed from Workspace",
		},
		// Service account variants (group).
		{
			name:     "service account created in group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionCreated, ServiceAccountPath: "sa/path"},
			expected: "Service account added to Group",
		},
		{
			name:     "service account role changed in group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRoleChanged, ServiceAccountPath: "sa/path"},
			expected: "Service account role updated in Group",
		},
		{
			name:     "service account removed from group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved, ServiceAccountPath: "sa/path"},
			expected: "Service account removed from Group",
		},
		// Member variants (workspace).
		{
			name:     "member created in workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionCreated, IsWorkspace: true},
			expected: "Workspace access granted",
		},
		{
			name:     "member role changed in workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRoleChanged, IsWorkspace: true},
			expected: "Workspace role updated",
		},
		{
			name:     "member removed from workspace",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved, IsWorkspace: true},
			expected: "Workspace access revoked",
		},
		// Member variants (group).
		{
			name:     "member created in group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionCreated},
			expected: "Group access granted",
		},
		{
			name:     "member role changed in group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRoleChanged},
			expected: "Group role updated",
		},
		{
			name:     "member removed from group",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved},
			expected: "Group access revoked",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := test.email
			assert.Equal(t, test.expected, email.Subject())
		})
	}
}

func TestMembershipChangeEmail_ShowCTA(t *testing.T) {
	tests := []struct {
		name     string
		email    MembershipChangeEmail
		expected bool
	}{
		{
			name:     "service account path set always shows CTA even when removed",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved, ServiceAccountPath: "sa/path"},
			expected: true,
		},
		{
			name:     "created member shows CTA",
			email:    MembershipChangeEmail{Action: MembershipChangeActionCreated},
			expected: true,
		},
		{
			name:     "role changed member shows CTA",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRoleChanged},
			expected: true,
		},
		{
			name:     "removed member hides CTA",
			email:    MembershipChangeEmail{Action: MembershipChangeActionRemoved},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := test.email
			assert.Equal(t, test.expected, email.ShowCTA())
		})
	}
}

func TestMembershipChangeEmail_namespaceType(t *testing.T) {
	tests := []struct {
		name     string
		email    MembershipChangeEmail
		expected string
	}{
		{
			name:     "workspace",
			email:    MembershipChangeEmail{IsWorkspace: true},
			expected: "Workspace",
		},
		{
			name:     "group",
			email:    MembershipChangeEmail{IsWorkspace: false},
			expected: "Group",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := test.email
			assert.Equal(t, test.expected, email.namespaceType())
		})
	}
}

func TestMembershipChangeEmail_InitFromData(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
		expected    MembershipChangeEmail
	}{
		{
			name: "valid JSON",
			data: []byte(`{"Action":"created","NamespacePath":"group/workspace","RoleName":"owner","IsWorkspace":true}`),
			expected: MembershipChangeEmail{
				Action:        MembershipChangeActionCreated,
				NamespacePath: "group/workspace",
				RoleName:      "owner",
				IsWorkspace:   true,
			},
		},
		{
			name:     "empty data is a no-op",
			data:     []byte{},
			expected: MembershipChangeEmail{},
		},
		{
			name:        "invalid JSON returns error",
			data:        []byte(`{not valid json`),
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := &MembershipChangeEmail{}
			err := email.InitFromData(test.data)
			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, *email)
		})
	}
}

func TestMembershipChangeEmail_Type(t *testing.T) {
	email := &MembershipChangeEmail{}
	assert.Equal(t, MembershipChangeEmailType, email.Type())
}
