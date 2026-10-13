package models

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestWorkspaceVCSProviderLink_ValidateModuleDirectory(t *testing.T) {
	tests := []struct {
		moduleDirectory string
		valid           bool
	}{
		{moduleDirectory: "", valid: true},
		{moduleDirectory: ".", valid: true},
		{moduleDirectory: "modules/network", valid: true},
		{moduleDirectory: "modules/../network", valid: true},
		{moduleDirectory: "..", valid: false},
		{moduleDirectory: "../../etc", valid: false},
		{moduleDirectory: "modules/../../etc", valid: false},
		{moduleDirectory: "/etc", valid: false},
	}

	for _, test := range tests {
		t.Run(test.moduleDirectory, func(t *testing.T) {
			link := &WorkspaceVCSProviderLink{ModuleDirectory: ptr.String(test.moduleDirectory)}
			err := link.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			}
		})
	}
}
