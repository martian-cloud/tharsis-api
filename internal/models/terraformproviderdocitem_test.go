package models

import (
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestTerraformProviderDocItem_Validate(t *testing.T) {
	tests := []struct {
		name            string
		docItem         *TerraformProviderDocItem
		expectErrorCode errors.CodeType
	}{
		{
			name: "valid doc item",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "instance.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
		},
		{
			name: "valid doc item with subcategory",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "virtual_machine.md",
				Category:          TerraformProviderDocItemCategoryResources,
				Subcategory:       ptr.String("Virtual Machine"),
			},
		},
		{
			name: "valid filename with dot, dash, underscore and mixed case",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "A.b-c_1.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
		},
		{
			name: "filename without a .md extension",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "instance",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename with an unexpected extension",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "instance.txt",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid category",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "instance.md",
				Category:          TerraformProviderDocItemCategory("bogus"),
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "missing filename",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename too long",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          strings.Repeat("a", maxTerraformProviderDocItemFilenameLength+1) + ".md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename starting with a dot",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          ".hidden.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename that is a parent-directory reference",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "..",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename with a space",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "aws instance.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename with a path separator",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "sub/instance.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "filename with a url-significant character",
			docItem: &TerraformProviderDocItem{
				ProviderVersionID: "version-1",
				Filename:          "aws#instance.md",
				Category:          TerraformProviderDocItemCategoryResources,
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.docItem.Validate()
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
