package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetTerraformCLIVersions(t *testing.T) {
	testCases := []struct {
		caller          auth.Caller
		name            string
		expectErrorCode errors.CodeType
	}{
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
		// The authorized happy path is not unit-testable here: after authorization,
		// GetTerraformCLIVersions calls terraform.GetCLIVersions, which performs live
		// network I/O against the HashiCorp releases API. That path is covered by
		// integration tests instead.
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			service := &service{}

			_, err := service.GetTerraformCLIVersions(auth.WithCaller(context.TODO(), test.caller))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateTerraformCLIDownloadURL(t *testing.T) {
	testCases := []struct {
		name            string
		expectErrorCode errors.CodeType
		withCaller      bool
		binaryExists    bool
	}{
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:         "authorized caller with existing binary returns a presigned URL",
			withCaller:   true,
			binaryExists: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			input := &TerraformCLIVersionsInput{
				Version:      "1.2.2",
				OS:           "linux",
				Architecture: "amd64",
			}

			mockCLIStore := NewMockTerraformCLIStore(t)

			var caller auth.Caller
			if test.withCaller {
				caller = auth.NewMockCaller(t)

				mockCLIStore.On("DoesTerraformCLIBinaryExist", mock.Anything, input.Version, input.OS, input.Architecture).
					Return(test.binaryExists, nil)

				if test.binaryExists {
					mockCLIStore.On("CreateTerraformCLIBinaryPresignedURL", mock.Anything, input.Version, input.OS, input.Architecture).
						Return("https://example.com/terraform_1.2.2_linux_amd64.zip", nil)
				}
			}

			service := &service{
				cliStore: mockCLIStore,
			}

			url, err := service.CreateTerraformCLIDownloadURL(auth.WithCaller(context.TODO(), caller), input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, url)
		})
	}
}
