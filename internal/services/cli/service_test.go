package cli

import (
	"context"
	"net/url"
	"path"
	"strings"
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
	validInput := TerraformCLIVersionsInput{
		Version:      "1.2.2",
		OS:           "linux",
		Architecture: "amd64",
	}

	testCases := []struct {
		name            string
		expectErrorCode errors.CodeType
		mutateInput     func(*TerraformCLIVersionsInput)
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
		{
			name:            "version is not a semantic version",
			withCaller:      true,
			mutateInput:     func(i *TerraformCLIVersionsInput) { i.Version = "../../1.2.2" },
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "operating system contains a slash",
			withCaller:      true,
			mutateInput:     func(i *TerraformCLIVersionsInput) { i.OS = "linux/amd64" },
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "architecture is a dot dot segment",
			withCaller:      true,
			mutateInput:     func(i *TerraformCLIVersionsInput) { i.Architecture = ".." },
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "empty architecture",
			withCaller:      true,
			mutateInput:     func(i *TerraformCLIVersionsInput) { i.Architecture = "" },
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			input := validInput
			if test.mutateInput != nil {
				test.mutateInput(&input)
			}

			mockCLIStore := NewMockTerraformCLIStore(t)

			var caller auth.Caller
			if test.withCaller {
				caller = auth.NewMockCaller(t)
			}

			if test.withCaller && test.mutateInput == nil {

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

			url, err := service.CreateTerraformCLIDownloadURL(auth.WithCaller(context.TODO(), caller), &input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, url)
		})
	}
}

func FuzzGetTerraformCLIDownloadURL(f *testing.F) {
	f.Add("1.5.7", "linux", "amd64")
	f.Add("1.6.0-beta1", "darwin", "arm64")
	f.Add("", "", "")
	f.Add("..", "..", "..")
	f.Add("../../evil", "linux", "amd64")
	f.Add("1.0.0?x=1#frag", "linux/../..", "amd64%2F")
	f.Add("1.0.0@evil.example", "a b", "\x00")

	baseURL, err := url.Parse(hashicorpReleasesBaseURL)
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, version, os, architecture string) {
		input := &TerraformCLIVersionsInput{Version: version, OS: os, Architecture: architecture}
		downloadURL := getTerraformCLIDownloadURL(input)

		parsed, err := url.Parse(downloadURL)
		require.NoError(t, err)

		// User input can never change where the binary is downloaded from, nor add a query or fragment.
		assert.Equal(t, baseURL.Scheme, parsed.Scheme)
		assert.Equal(t, baseURL.Host, parsed.Host)
		assert.Nil(t, parsed.User)
		assert.Empty(t, parsed.RawQuery)
		assert.Empty(t, parsed.Fragment)

		// The path is always /terraform/<version>/<file name>, each a single escaped segment.
		segments := strings.Split(parsed.EscapedPath(), "/")
		require.Len(t, segments, 4)
		assert.Equal(t, []string{"", "terraform"}, segments[:2])
		unescapedVersion, err := url.PathUnescape(segments[2])
		require.NoError(t, err)
		assert.Equal(t, version, unescapedVersion)
		fileName, err := url.PathUnescape(segments[3])
		require.NoError(t, err)
		assert.Equal(t, "terraform_"+version+"_"+os+"_"+architecture+".zip", fileName)

		// CreateTerraformCLIDownloadURL only builds an object key for validated input, and for such
		// input the key is exactly terraform/<version>/<the downloaded file name>: no input can add a
		// path segment, so distinct inputs never share a key and cleaning leaves the key unchanged.
		if err := input.validate(); err != nil {
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			return
		}
		objectKey := getTerraformCLIBinaryObjectKey(version, os, architecture)
		assert.Equal(t, "terraform/"+version+"/"+fileName, objectKey)
		assert.Equal(t, "/"+objectKey, path.Clean("/"+objectKey))
		assert.Len(t, strings.Split(objectKey, "/"), 3)
	})
}
