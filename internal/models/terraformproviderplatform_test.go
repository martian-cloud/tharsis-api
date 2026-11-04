package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestTerraformProviderPlatform_Validate(t *testing.T) {
	tests := []struct {
		name            string
		operatingSystem string
		architecture    string
		wantErr         bool
	}{
		{name: "linux amd64", operatingSystem: "linux", architecture: "amd64"},
		{name: "darwin arm64", operatingSystem: "darwin", architecture: "arm64"},
		{name: "windows 386", operatingSystem: "windows", architecture: "386"},
		{name: "freebsd arm", operatingSystem: "freebsd", architecture: "arm"},
		{name: "s390x", operatingSystem: "linux", architecture: "s390x"},
		{name: "underscore", operatingSystem: "linux_x", architecture: "amd64", wantErr: true},
		{name: "empty operating system", operatingSystem: "", architecture: "amd64", wantErr: true},
		{name: "empty architecture", operatingSystem: "linux", architecture: "", wantErr: true},
		{name: "dot dot operating system", operatingSystem: "..", architecture: "amd64", wantErr: true},
		{name: "path in architecture", operatingSystem: "linux", architecture: "../../x", wantErr: true},
		{name: "slash", operatingSystem: "linux/amd64", architecture: "amd64", wantErr: true},
		{name: "uppercase", operatingSystem: "Linux", architecture: "amd64", wantErr: true},
		{name: "hyphen", operatingSystem: "linux", architecture: "arm-64", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&TerraformProviderPlatform{OperatingSystem: tt.operatingSystem, Architecture: tt.architecture}).Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
