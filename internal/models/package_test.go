package models

import (
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestPackage_Validate(t *testing.T) {
	tests := []struct {
		name            string
		pkg             *Package
		expectErrorCode errors.CodeType
	}{
		{
			name: "valid package",
			pkg: &Package{
				Name:        "my-package",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibilityPrivate,
			},
		},
		{
			name: "valid with description",
			pkg: &Package{
				Name:        "my-package",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Description: ptr.String("a description"),
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibilityGlobal,
			},
		},
		{
			name: "missing group owner",
			pkg: &Package{
				Name:        "my-package",
				RootGroupID: "root-1",
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibilityPrivate,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "missing root group",
			pkg: &Package{
				Name:       "my-package",
				GroupID:    "group-1",
				Kind:       PackageKindOPAPolicy,
				Visibility: PackageVisibilityPrivate,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid name",
			pkg: &Package{
				Name:        "Invalid Name!",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibilityPrivate,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "unsupported kind",
			pkg: &Package{
				Name:        "my-package",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Kind:        PackageKind("bogus"),
				Visibility:  PackageVisibilityPrivate,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "unsupported visibility",
			pkg: &Package{
				Name:        "my-package",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibility("bogus"),
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "description too long",
			pkg: &Package{
				Name:        "my-package",
				GroupID:     "group-1",
				RootGroupID: "root-1",
				Description: ptr.String(strings.Repeat("a", 1000)),
				Kind:        PackageKindOPAPolicy,
				Visibility:  PackageVisibilityPrivate,
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPackage_GetRootGroupPath(t *testing.T) {
	tests := []struct {
		name string
		trn  string
		want string
	}{
		{
			name: "nested group path returns first segment",
			trn:  "trn:package:acme/team/security/my-package",
			want: "acme",
		},
		{
			name: "single level group path",
			trn:  "trn:package:acme/my-package",
			want: "acme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &Package{Metadata: ResourceMetadata{TRN: tt.trn}}
			assert.Equal(t, tt.want, pkg.GetRootGroupPath())
		})
	}
}
