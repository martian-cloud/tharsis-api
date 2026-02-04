package providermirror

import (
	context "context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
)

const (
	// hashicorpGPGKey used for testing in this module.
	// GPG will require a proper public key.
	hashicorpGPGKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
ZF5q4h4I33PSGDdSvGXn9UMY5Isjpg==
=7pIB
-----END PGP PUBLIC KEY BLOCK-----`

	// awsProviderChecksumsData contains actual signed checksums data
	// for the aws 5.2.0 provider as a base64 encoded string to ease
	// testing in this module.
	awsProviderChecksumsData = `MGU0ODQ0OWU5ZjI5YjY0NjYzZTdmZjY0MWEzYmExZGE0MzQ2MDg0NjBjMzNhMjBiZGI0NWVmZWIx
ZTA2N2Q0YSAgdGVycmFmb3JtLXByb3ZpZGVyLWF3c181LjIuMF9mcmVlYnNkX2FtZDY0LnppcAow
ZWM2NTdhMWU1ODYwODczNjhjYzMwNTFjY2I4YmRmNjdlODc2M2U1MGVlY2U3NmI4ZGM0Njk1Zjhk
MzQ5ZWJiICB0ZXJyYWZvcm0tcHJvdmlkZXItYXdzXzUuMi4wX3dpbmRvd3NfMzg2LnppcAoxY2Zm
NTQxZTc5MjQ3N2M0ZGM4Yjg0MDVhNmY3NmE1NmQxMjkyZTIzZDZmYzM2Nzk5M2VmZWQyYjM5ODhj
MjA4ICB0ZXJyYWZvcm0tcHJvdmlkZXItYXdzXzUuMi4wX29wZW5ic2RfYW1kNjQuemlwCjMxMmU3
YzBjNTY3NDJjYjgzZDE4Y2IyYTNmMmRhMzVmNGQ5MjQ4MTM1M2IwYWMyYjU0ZDYzMzNkZjkxYzIz
OTkgIHRlcnJhZm9ybS1wcm92aWRlci1hd3NfNS4yLjBfb3BlbmJzZF9hcm0uemlwCjM1MmJjOGE3
M2ZkYjgyOTMwNmMxNDFmZmZjYjE3MDE3MmM3ODYwYzlmZmYwZTAyMjM5ZWY4Yzc3MWIzNDY4YTMg
IHRlcnJhZm9ybS1wcm92aWRlci1hd3NfNS4yLjBfZGFyd2luX2FtZDY0LnppcAo0ZDU0MDNmOGQ1
YThkYTRkYjZiY2Y5ZDhiNjBmYzc5MGIyZjJlMWNmNDk0MzhiYjJjM2Y2YzJjY2JmYTY3MmNiICB0
ZXJyYWZvcm0tcHJvdmlkZXItYXdzXzUuMi4wX2ZyZWVic2RfYXJtLnppcAo0ZDkwYzlhNzU5Nzc4
YTU1Mjc0ZjA1ZjY5YzhiMzhhMmQwZDFkZWY3OTJiYjU1NmFjNjE5NGU1OTc5NjUzZjU1ICB0ZXJy
YWZvcm0tcHJvdmlkZXItYXdzXzUuMi4wX2xpbnV4X2FybTY0LnppcAo5MDMzZDAzZTA4OTY3YmYx
ZWJiNDIwY2IxZjZlNzc3NTBhN2FhNzUwMzZhNTNhMGI2NzA5ZmRjMTA3YTgyOWQ5ICB0ZXJyYWZv
cm0tcHJvdmlkZXItYXdzXzUuMi4wX2ZyZWVic2RfMzg2LnppcAo5YjEyYWY4NTQ4NmE5NmFlZGQ4
ZDc5ODRiMGZmODExYTRiNDJlM2Q4OGRhZDFhM2ZiNGMwYjU4MGQwNGZhNDI1ICB0ZXJyYWZvcm0t
cHJvdmlkZXItYXdzXzUuMi4wX21hbmlmZXN0Lmpzb24KYThjYmZmZWNmNWYxMjgwODFjZjYyZGJk
NmUyZDY4MTA1MDMxZDA4YzM0OTllMzRkY2I2NjkyY2MxNDdkZWM5ZCAgdGVycmFmb3JtLXByb3Zp
ZGVyLWF3c181LjIuMF9saW51eF8zODYuemlwCmJmMDc3OGFhMGI1M2UzZTY1YzI1ZmRjNGYxYjYz
OWY0MDgyYTAxMjI2NDAwOTU4ZTQxMzU2MzBlMmQxMzJkNTYgIHRlcnJhZm9ybS1wcm92aWRlci1h
d3NfNS4yLjBfbGludXhfYXJtLnppcApjOWRhMWFlNjhmYWE0YmM5ZjU4NWQxNzVlYjY2NmNiZmNk
ZGFmY2RjZmQ2YTE4ZjNlY2QwNzE2MzFmYjRkNGM1ICB0ZXJyYWZvcm0tcHJvdmlkZXItYXdzXzUu
Mi4wX29wZW5ic2RfMzg2LnppcApjYmMwNGVkZDdhODY4Y2YyNzEzZmQzMDFjYTM1N2FkYjk0MDAw
M2QwYzAyZjUzNTFjOTg3ZTJhYjZlYWQyYTQ2ICB0ZXJyYWZvcm0tcHJvdmlkZXItYXdzXzUuMi4w
X3dpbmRvd3NfYW1kNjQuemlwCmRhZTY0ZjMwNDdmMThmYWJlN2M1ZTlkNmJjY2ZiN2Y0YjQ3OTM5
OTRjZWEwOTQ3ZGMxYWMxN2UxOTIyYmY3MTUgIHRlcnJhZm9ybS1wcm92aWRlci1hd3NfNS4yLjBf
bGludXhfYW1kNjQuemlwCmY3YmFkMzk4MzQ3OWFmOTViNWIxY2JlNDMzMDVmYTY0OTIyMjJjYzRm
MTdhNTEyNjc3ZGZhYjdjOThjYzQ4MGMgIHRlcnJhZm9ybS1wcm92aWRlci1hd3NfNS4yLjBfZGFy
d2luX2FybTY0LnppcAo=`

	// awsProviderChecksumsSignature contains the actual detached signature
	// needed to verify the checksum data as a base64 encoded string.
	awsProviderChecksumsSignature = `wsFzBAABCAAnBQJkg4UzCRDIIMbVzSerhxYhBDdOx1tIWRNgSoMcx8ggxtXNJ6uHAABwow/8CK+K
FeWZemkJ2aC/RMZjlujvyd4vk/JGgK9dCnsvzTPcAqcbNSBRfzQFU6T54MZd3Hf1EcAaTqzpSSf7
7sFF6IQBGKEt5bBkqwGc36XX8+KUteq2ZO81Z+pDC4n3ODttUXODi0JUup6dKMMJris7bEwVzgXF
Ijp0sypILncmaVgScOF+U7IM4XDKveJuFQ+DwQCKA/64FB7qsd8pi8aUBtCThDxh0Z7xfEfyLULU
QIDpma59ve6XgNBkBZLA85ba+SoVn+Tnue3opIeNLxo0ffXckEGuKn5Ez4NJc87J8TwqIsPZZwuR
k2jNCQ//pauYUXzPFtM+j10aJyWwtV8GRnHQexMk1OfIT/d5o3lC3daYbcQoEzmMW7eB7bVQ0p7P
GuqzoeYFVWqDAIzLO2BJgZE3KB7oeeRjzoagbIaGnara3Q4Cukz0EpZ3uDmqrK/hYe20tbHA2LlE
x9d1qo6p7fkFE/ZEihlX2piV+9D6MrgvIthXHmHYiG9uD6KvA6UGXHRn6qmiBSBG8/3Rbh6GrTsx
Xn6MMb4aV5AsdW11GJX4SnH3LVLWO7i/em706ths0weld2KkXa1B/gUe6ucFfM0xh3JfIoD6mVh6
LISZ9LVjefyodjylJxUWlY8FMkg5UxM1o+SRmAIsxEHiaGrlEQ9HplZa7GxiQ2IQb2lL6NA=`
)

// roundTripFunc implements the RoundTripper interface.
type roundTripFunc func(r *http.Request) *http.Response

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

// newTestHTTPClient returns *http.Client with Transport replaced to avoid making real calls.
func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
	}
}

// fakeServiceDiscoverer implements serviceDiscoverer interface for service url discovery.
type fakeServiceDiscoverer struct{}

// DiscoverServiceURL returns hard-coded service URL for testing.
func (d *fakeServiceDiscoverer) DiscoverServiceURL(hostname svchost.Hostname, _ string) (*url.URL, error) {
	return &url.URL{
		Scheme: "https",
		Host:   hostname.String(),
		Path:   "/v1/providers/",
	}, nil
}

func TestGetProviderVersionMirrorByID(t *testing.T) {
	versionMirrorID := "version-mirror-1"
	groupID := "group-1"

	type testCase struct {
		expectMirror    *models.TerraformProviderVersionMirror
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return a version mirror by id",
			expectMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "1.0.0",
				GroupID:           groupID,
			},
		},
		{
			name:            "version mirror not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have access to view version mirror",
			expectMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "1.0.0",
				GroupID:           groupID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			if test.expectMirror != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)
			}

			mockVersionMirrors.On("GetVersionMirrorByID", mock.Anything, versionMirrorID).Return(test.expectMirror, nil)

			dbClient := &db.Client{
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			service := &service{dbClient: dbClient}

			actualMirror, err := service.GetProviderVersionMirrorByID(auth.WithCaller(ctx, mockCaller), versionMirrorID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectMirror, actualMirror)
		})
	}
}

func TestGetProviderVersionMirrorByTRN(t *testing.T) {
	sampleMirror := &models.TerraformProviderVersionMirror{
		Metadata: models.ResourceMetadata{
			ID:  "mirror-1",
			TRN: types.TerraformProviderVersionMirrorModelType.BuildTRN("group-1/mirror-1"),
		},
		GroupID:           "group-1",
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		SemanticVersion:   "1.0.0",
	}

	type testCase struct {
		name          string
		mirror        *models.TerraformProviderVersionMirror
		authError     error
		expectErrCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "get provider version mirror by TRN",
			mirror: sampleMirror,
		},
		{
			name:          "subject does not have access to provider version mirror",
			mirror:        sampleMirror,
			authError:     errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "provider version mirror not found",
			expectErrCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockVersionMirrors.On("GetVersionMirrorByTRN", mock.Anything, sampleMirror.Metadata.TRN).Return(test.mirror, nil)

			if test.mirror != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			service := &service{
				dbClient: dbClient,
			}

			mirror, err := service.GetProviderVersionMirrorByTRN(auth.WithCaller(ctx, mockCaller), sampleMirror.Metadata.TRN)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.mirror, mirror)
		})
	}
}

func TestGetProviderVersionMirrorsByIDs(t *testing.T) {
	versionMirrorID := "version-mirror-1"
	groupID := "group-1"

	type testCase struct {
		expectMirror    *models.TerraformProviderVersionMirror
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get version mirrors",
			expectMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "1.0.0",
				GroupID:           groupID,
			},
		},
		{
			name: "subject does not have access to version mirror",
			expectMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "1.0.0",
				GroupID:           groupID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "version mirrors not found",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			if test.expectMirror != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)
			}

			getVersionMirrorsResponse := db.ProviderVersionMirrorsResult{
				VersionMirrors: []models.TerraformProviderVersionMirror{},
			}

			if test.expectMirror != nil {
				getVersionMirrorsResponse.VersionMirrors = append(getVersionMirrorsResponse.VersionMirrors, *test.expectMirror)
			}

			mockVersionMirrors.On("GetVersionMirrors", mock.Anything, &db.GetProviderVersionMirrorsInput{
				Filter: &db.TerraformProviderVersionMirrorFilter{
					VersionMirrorIDs: []string{versionMirrorID},
				},
			}).Return(&getVersionMirrorsResponse, nil)

			dbClient := db.Client{
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			service := &service{dbClient: &dbClient}

			modules, err := service.GetProviderVersionMirrorsByIDs(auth.WithCaller(ctx, mockCaller), []string{versionMirrorID})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectMirror != nil {
				assert.Len(t, modules, 1)
				assert.Equal(t, test.expectMirror, &modules[0])
			} else {
				assert.Len(t, modules, 0)
			}
		})
	}
}

func TestGetProviderVersionMirrors(t *testing.T) {
	versionMirrorID := "version-mirror-1"
	groupID := "group-1"
	namespace := "some"

	type testCase struct {
		expectMirror    *models.TerraformProviderVersionMirror
		input           *GetProviderVersionMirrorsInput
		expectErrorCode errors.CodeType
		authError       error
		name            string
	}

	testCases := []testCase{
		{
			name: "successfully return a list of provider version mirrors",
			input: &GetProviderVersionMirrorsInput{
				NamespacePath: namespace,
			},
			expectMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "1.0.0",
				GroupID:           groupID,
			},
		},
		{
			name:            "subject does not have viewer access to namespace",
			input:           &GetProviderVersionMirrorsInput{},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "no version mirrors found",
			input: &GetProviderVersionMirrorsInput{
				NamespacePath: namespace,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)

			getVersionMirrorsResponse := db.ProviderVersionMirrorsResult{
				VersionMirrors: []models.TerraformProviderVersionMirror{},
			}

			if test.expectMirror != nil {
				getVersionMirrorsResponse.VersionMirrors = append(getVersionMirrorsResponse.VersionMirrors, *test.expectMirror)
			}

			if test.authError == nil {
				mockVersionMirrors.On("GetVersionMirrors", mock.Anything, &db.GetProviderVersionMirrorsInput{
					Filter: &db.TerraformProviderVersionMirrorFilter{
						NamespacePaths: []string{namespace},
					},
				}).Return(&getVersionMirrorsResponse, nil)
			}

			dbClient := db.Client{
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			service := &service{dbClient: &dbClient}

			result, err := service.GetProviderVersionMirrors(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectMirror != nil {
				assert.Len(t, result.VersionMirrors, 1)
				assert.Equal(t, *test.expectMirror, result.VersionMirrors[0])
			} else {
				assert.Len(t, result.VersionMirrors, 0)
			}
		})
	}
}

func TestCreateProviderVersionMirror(t *testing.T) {
	groupID := "group-1"
	namespace := "group-1"
	registryHostname := "registry.terraform.io"
	registryNamespace := "hashicorp"
	providerType := "aws"
	semanticVersion := "5.2.0"
	mockSubject := "test-subject"

	sampleCreatedMirror := &models.TerraformProviderVersionMirror{
		RegistryHostname:  registryHostname,
		RegistryNamespace: registryNamespace,
		Type:              providerType,
		SemanticVersion:   semanticVersion,
		GroupID:           groupID,
		CreatedBy:         mockSubject,
		Digests: provider.Checksums{
			"terraform-provider-aws_5.2.0_freebsd_amd64.zip": parseChecksumString(t, "0e48449e9f29b64663e7ff641a3ba1da434608460c33a20bdb45efeb1e067d4a"),
			"terraform-provider-aws_5.2.0_windows_386.zip":   parseChecksumString(t, "0ec657a1e586087368cc3051ccb8bdf67e8763e50eece76b8dc4695f8d349ebb"),
			"terraform-provider-aws_5.2.0_openbsd_amd64.zip": parseChecksumString(t, "1cff541e792477c4dc8b8405a6f76a56d1292e23d6fc367993efed2b3988c208"),
			"terraform-provider-aws_5.2.0_openbsd_arm.zip":   parseChecksumString(t, "312e7c0c56742cb83d18cb2a3f2da35f4d92481353b0ac2b54d6333df91c2399"),
			"terraform-provider-aws_5.2.0_darwin_amd64.zip":  parseChecksumString(t, "352bc8a73fdb829306c141fffcb170172c7860c9fff0e02239ef8c771b3468a3"),
			"terraform-provider-aws_5.2.0_freebsd_arm.zip":   parseChecksumString(t, "4d5403f8d5a8da4db6bcf9d8b60fc790b2f2e1cf49438bb2c3f6c2ccbfa672cb"),
			"terraform-provider-aws_5.2.0_linux_arm64.zip":   parseChecksumString(t, "4d90c9a759778a55274f05f69c8b38a2d0d1def792bb556ac6194e5979653f55"),
			"terraform-provider-aws_5.2.0_freebsd_386.zip":   parseChecksumString(t, "9033d03e08967bf1ebb420cb1f6e77750a7aa75036a53a0b6709fdc107a829d9"),
			"terraform-provider-aws_5.2.0_manifest.json":     parseChecksumString(t, "9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425"),
			"terraform-provider-aws_5.2.0_linux_386.zip":     parseChecksumString(t, "a8cbffecf5f128081cf62dbd6e2d68105031d08c3499e34dcb6692cc147dec9d"),
			"terraform-provider-aws_5.2.0_linux_arm.zip":     parseChecksumString(t, "bf0778aa0b53e3e65c25fdc4f1b639f4082a01226400958e4135630e2d132d56"),
			"terraform-provider-aws_5.2.0_openbsd_386.zip":   parseChecksumString(t, "c9da1ae68faa4bc9f585d175eb666cbfcddafcdcfd6a18f3ecd071631fb4d4c5"),
			"terraform-provider-aws_5.2.0_windows_amd64.zip": parseChecksumString(t, "cbc04edd7a868cf2713fd301ca357adb940003d0c02f5351c987e2ab6ead2a46"),
			"terraform-provider-aws_5.2.0_linux_amd64.zip":   parseChecksumString(t, "dae64f3047f18fabe7c5e9d6bccfb7f4b4793994cea0947dc1ac17e1922bf715"),
			"terraform-provider-aws_5.2.0_darwin_arm64.zip":  parseChecksumString(t, "f7bad3983479af95b5b1cbe43305fa6492222cc4f17a512677dfab7c98cc480c"),
		},
	}

	type testCase struct {
		group                 *models.Group
		expectCreated         *models.TerraformProviderVersionMirror
		input                 *CreateProviderVersionMirrorInput
		listVersions          []provider.VersionInfo
		packageInfo           *provider.PackageInfo
		checksums             provider.Checksums
		authError             error
		name                  string
		expectErrorCode       errors.CodeType
		limit                 int
		injectMirrorsPerGroup int32
	}

	testCases := []testCase{
		{
			name: "successfully create provider version in root group",
			input: &CreateProviderVersionMirrorInput{
				RegistryHostname:  registryHostname,
				RegistryNamespace: registryNamespace,
				Type:              providerType,
				SemanticVersion:   semanticVersion,
				GroupPath:         namespace,
			},
			listVersions: []provider.VersionInfo{
				{
					Version: semanticVersion,
					Platforms: []provider.Platform{
						{OS: "windows", Arch: "amd64"},
					},
				},
			},
			packageInfo: &provider.PackageInfo{
				SHASumsURL:          "https://registry.terraform.io/v1/providers/checksums",
				SHASumsSignatureURL: "https://registry.terraform.io/v1/providers/signatures",
				GPGASCIIArmors:      []string{hashicorpGPGKey},
			},
			checksums: sampleCreatedMirror.Digests,
			group: &models.Group{
				Metadata: models.ResourceMetadata{ID: groupID},
				FullPath: namespace,
			},
			expectCreated:         sampleCreatedMirror,
			limit:                 5,
			injectMirrorsPerGroup: 6,
			expectErrorCode:       errors.EInvalid,
		},
		{
			name: "group not found",
			input: &CreateProviderVersionMirrorInput{
				GroupPath: namespace,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "group is not a root",
			input: &CreateProviderVersionMirrorInput{
				GroupPath: namespace,
			},
			group: &models.Group{
				ParentID: "not-root",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "provider fully qualified name is invalid",
			input: &CreateProviderVersionMirrorInput{
				RegistryHostname:  "invalid-hostname",
				RegistryNamespace: registryHostname,
				Type:              providerType,
				SemanticVersion:   semanticVersion,
				GroupPath:         namespace,
			},
			group: &models.Group{
				FullPath: namespace,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "semantic version is invalid",
			input: &CreateProviderVersionMirrorInput{
				RegistryHostname:  registryHostname,
				RegistryNamespace: registryHostname,
				Type:              providerType,
				SemanticVersion:   "not-semantic",
				GroupPath:         namespace,
			},
			group: &models.Group{
				FullPath: namespace,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "unsupported provider version",
			input: &CreateProviderVersionMirrorInput{
				RegistryHostname:  registryHostname,
				RegistryNamespace: registryNamespace,
				Type:              providerType,
				SemanticVersion:   semanticVersion,
				GroupPath:         namespace,
			},
			group: &models.Group{
				FullPath: namespace,
			},
			listVersions: []provider.VersionInfo{
				{Version: "0.1.0"}, // Different from above.
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "subject does not have permissions to create version mirror",
			input: &CreateProviderVersionMirrorInput{
				GroupPath: namespace,
			},
			group: &models.Group{
				FullPath: namespace,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateTerraformProviderMirrorPermission, mock.Anything).Return(test.authError)

			if test.authError == nil {
				mockGroups.On("GetGroupByTRN", mock.Anything, mock.Anything).Return(test.group, nil)
			}

			if test.expectCreated != nil {
				mockCaller.On("GetSubject").Return(mockSubject)

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockVersionMirrors.On("CreateVersionMirror", mock.Anything, test.expectCreated).Return(test.expectCreated, nil)
				mockVersionMirrors.On("GetVersionMirrors", mock.Anything, &db.GetProviderVersionMirrorsInput{
					Filter: &db.TerraformProviderVersionMirrorFilter{
						GroupID: &groupID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(&db.ProviderVersionMirrorsResult{PageInfo: &pagination.PageInfo{TotalCount: test.injectMirrorsPerGroup}}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(&models.ResourceLimit{Value: test.limit}, nil)

				if test.injectMirrorsPerGroup <= int32(test.limit) {
					mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
						NamespacePath: &namespace,
						Action:        models.ActionCreate,
						TargetType:    models.TargetTerraformProviderVersionMirror,
					}).Return(nil, nil)

					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			dbClient := &db.Client{
				Groups:                          mockGroups,
				ResourceLimits:                  mockResourceLimits,
				Transactions:                    mockTransactions,
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			mockResolver := provider.NewMockRegistryProtocol(t)
			if test.listVersions != nil {
				mockResolver.On("ListVersions", mock.Anything, mock.Anything, mock.Anything).Return(test.listVersions, nil)
			}
			if test.packageInfo != nil {
				mockResolver.On("GetPackageInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.packageInfo, nil)
			}
			if test.checksums != nil {
				mockResolver.On("GetChecksums", mock.Anything, mock.Anything, mock.Anything).Return(test.checksums, nil)
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:          logger,
				dbClient:        dbClient,
				registryClient:  mockResolver,
				limitChecker:    limits.NewLimitChecker(dbClient),
				activityService: mockActivityEvents,
			}

			created, err := service.CreateProviderVersionMirror(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreated, created)
		})
	}
}

func TestDeleteProviderVersionMirror(t *testing.T) {
	versionMirrorID := "version-mirror-id"
	groupID := "group-1"
	namespace := "some/group"

	sampleGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: groupID},
		FullPath: namespace,
	}

	sampleVersionMirror := &models.TerraformProviderVersionMirror{
		Metadata:          models.ResourceMetadata{ID: versionMirrorID},
		RegistryHostname:  "registry.terraform.io",
		RegistryNamespace: "hashicorp",
		Type:              "aws",
		SemanticVersion:   "1.0.0",
		GroupID:           groupID,
	}

	type testCase struct {
		authError         error
		input             *DeleteProviderVersionMirrorInput
		name              string
		expectErrorCode   errors.CodeType
		mirroredPlatforms int32
	}

	testCases := []testCase{
		{
			name:  "delete a provider version mirror with no platform mirrors without force option",
			input: &DeleteProviderVersionMirrorInput{VersionMirror: sampleVersionMirror},
		},
		{
			name: "delete a provider version mirror with platform mirrors with force option",
			input: &DeleteProviderVersionMirrorInput{
				VersionMirror: sampleVersionMirror,
				Force:         true,
			},
		},
		{
			name:              "cannot delete a provider version mirror with platform mirrors without force option",
			input:             &DeleteProviderVersionMirrorInput{VersionMirror: sampleVersionMirror},
			mirroredPlatforms: 1,
			expectErrorCode:   errors.EConflict,
		},
		{
			name:            "subject does not have permissions to delete provider version mirror",
			input:           &DeleteProviderVersionMirrorInput{VersionMirror: sampleVersionMirror},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockCaller.On("RequirePermission", mock.Anything, models.DeleteTerraformProviderMirrorPermission, mock.Anything).Return(test.authError)

			if test.authError == nil && !test.input.Force {
				mockPlatformMirrors.On("GetPlatformMirrors", mock.Anything, &db.GetProviderPlatformMirrorsInput{
					Filter: &db.TerraformProviderPlatformMirrorFilter{
						VersionMirrorID: &versionMirrorID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(&db.ProviderPlatformMirrorsResult{PageInfo: &pagination.PageInfo{TotalCount: test.mirroredPlatforms}}, nil)
			}

			if test.expectErrorCode == "" {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockVersionMirrors.On("DeleteVersionMirror", mock.Anything, test.input.VersionMirror).Return(nil)

				mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(sampleGroup, nil)

				provider := &provider.Provider{
					Hostname:  test.input.VersionMirror.RegistryHostname,
					Namespace: test.input.VersionMirror.RegistryNamespace,
					Type:      test.input.VersionMirror.Type,
				}
				providerName := fmt.Sprintf("%s/%s", provider, test.input.VersionMirror.SemanticVersion)
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, &activityevent.CreateActivityEventInput{
					NamespacePath: &namespace,
					Action:        models.ActionDeleteChildResource,
					TargetType:    models.TargetGroup,
					TargetID:      groupID,
					Payload: &models.ActivityEventDeleteChildResourcePayload{
						Name: providerName,
						ID:   versionMirrorID,
						Type: string(models.TargetTerraformProviderVersionMirror),
					},
				}).Return(nil, nil)
			}

			dbClient := &db.Client{
				Groups:                           mockGroups,
				Transactions:                     mockTransactions,
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			logger, _ := logger.NewForTest()
			service := &service{logger: logger, dbClient: dbClient, activityService: mockActivityEvents}

			err := service.DeleteProviderVersionMirror(auth.WithCaller(ctx, mockCaller), test.input)

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

func TestGetProviderPlatformMirrorByID(t *testing.T) {
	versionMirrorID := "version-mirror-1"
	platformMirrorID := "platform-mirror-1"
	groupID := "group-1"

	type testCase struct {
		expectMirror    *models.TerraformProviderPlatformMirror
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return a platform mirror by id",
			expectMirror: &models.TerraformProviderPlatformMirror{
				Metadata:        models.ResourceMetadata{ID: platformMirrorID},
				VersionMirrorID: versionMirrorID,
			},
		},
		{
			name:            "platform mirror not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have access to view platform mirror",
			expectMirror: &models.TerraformProviderPlatformMirror{
				Metadata:        models.ResourceMetadata{ID: platformMirrorID},
				VersionMirrorID: versionMirrorID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)

			if test.expectMirror != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)

				mockVersionMirrors.On("GetVersionMirrorByID", mock.Anything, versionMirrorID).
					Return(&models.TerraformProviderVersionMirror{
						GroupID: groupID,
					}, nil)
			}

			mockPlatformMirrors.On("GetPlatformMirrorByID", mock.Anything, platformMirrorID).Return(test.expectMirror, nil)

			dbClient := &db.Client{
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			service := &service{dbClient: dbClient}

			actualMirror, err := service.GetProviderPlatformMirrorByID(auth.WithCaller(ctx, mockCaller), platformMirrorID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectMirror, actualMirror)
		})
	}
}

func TestGetProviderPlatformMirrorByTRN(t *testing.T) {
	sampleMirror := &models.TerraformProviderPlatformMirror{
		Metadata: models.ResourceMetadata{
			ID:  "mirror-1",
			TRN: types.TerraformProviderPlatformMirrorModelType.BuildTRN("group-1/mirror-1"),
		},
		VersionMirrorID: "version-1",
		OS:              "linux",
		Architecture:    "amd64",
	}

	type testCase struct {
		name          string
		mirror        *models.TerraformProviderPlatformMirror
		authError     error
		expectErrCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "get provider platform mirror by TRN",
			mirror: sampleMirror,
		},
		{
			name:          "subject does not have access to provider platform mirror",
			mirror:        sampleMirror,
			authError:     errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "provider platform mirror not found",
			expectErrCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVersionMirror := db.NewMockTerraformProviderVersionMirrors(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)

			mockPlatformMirrors.On("GetPlatformMirrorByTRN", mock.Anything, sampleMirror.Metadata.TRN).Return(test.mirror, nil)

			if test.mirror != nil {
				mockVersionMirror.On("GetVersionMirrorByID", mock.Anything, sampleMirror.VersionMirrorID).Return(&models.TerraformProviderVersionMirror{
					GroupID: "group-1",
				}, nil)

				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				TerraformProviderVersionMirrors:  mockVersionMirror,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			service := &service{
				dbClient: dbClient,
			}

			mirror, err := service.GetProviderPlatformMirrorByTRN(auth.WithCaller(ctx, mockCaller), sampleMirror.Metadata.TRN)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.mirror, mirror)
		})
	}
}

func TestGetProviderPlatformMirrors(t *testing.T) {
	groupID := "group-1"
	versionMirrorID := "version-mirror-1"
	platformMirrorID := "platform-mirror-1"

	sampleVersionMirror := &models.TerraformProviderVersionMirror{
		Metadata: models.ResourceMetadata{ID: versionMirrorID},
		GroupID:  groupID,
	}

	type testCase struct {
		expectMirror    *models.TerraformProviderPlatformMirror
		input           *GetProviderPlatformMirrorsInput
		versionMirror   *models.TerraformProviderVersionMirror
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return a list of provider platform mirrors",
			input: &GetProviderPlatformMirrorsInput{
				VersionMirrorID: versionMirrorID,
			},
			versionMirror: sampleVersionMirror,
			expectMirror: &models.TerraformProviderPlatformMirror{
				Metadata: models.ResourceMetadata{ID: platformMirrorID},
			},
		},
		{
			name: "subject does not have viewer access to namespace",
			input: &GetProviderPlatformMirrorsInput{
				VersionMirrorID: versionMirrorID,
			},
			versionMirror:   sampleVersionMirror,
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "version mirror does not exist",
			input: &GetProviderPlatformMirrorsInput{
				VersionMirrorID: versionMirrorID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "no platform mirrors found",
			input: &GetProviderPlatformMirrorsInput{
				VersionMirrorID: versionMirrorID,
			},
			versionMirror: sampleVersionMirror,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockVersionMirrors.On("GetVersionMirrorByID", mock.Anything, versionMirrorID).Return(test.versionMirror, nil)

			if test.versionMirror != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)

				getPlatformMirrorsResponse := db.ProviderPlatformMirrorsResult{
					PlatformMirrors: []models.TerraformProviderPlatformMirror{},
				}

				if test.expectMirror != nil {
					getPlatformMirrorsResponse.PlatformMirrors = append(getPlatformMirrorsResponse.PlatformMirrors, *test.expectMirror)
				}

				if test.authError == nil {
					mockPlatformMirrors.On("GetPlatformMirrors", mock.Anything, &db.GetProviderPlatformMirrorsInput{
						Filter: &db.TerraformProviderPlatformMirrorFilter{
							VersionMirrorID: &versionMirrorID,
						},
					}).Return(&getPlatformMirrorsResponse, nil)
				}
			}

			dbClient := db.Client{
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			service := &service{dbClient: &dbClient}

			result, err := service.GetProviderPlatformMirrors(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectMirror != nil {
				assert.Len(t, result.PlatformMirrors, 1)
				assert.Equal(t, *test.expectMirror, result.PlatformMirrors[0])
			} else {
				assert.Len(t, result.PlatformMirrors, 0)
			}
		})
	}
}

func TestDeleteProviderPlatformMirror(t *testing.T) {
	versionMirrorID := "version-mirror-id"
	groupID := "group-1"

	samplePlatformMirror := &models.TerraformProviderPlatformMirror{
		VersionMirrorID: versionMirrorID,
		OS:              "windows",
		Architecture:    "amd64",
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "delete a provider platform mirror",
		},
		{
			name:            "subject does not have permissions to delete provider platform mirror",
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockCaller.On("RequirePermission", mock.Anything, models.DeleteTerraformProviderMirrorPermission, mock.Anything).Return(test.authError)

			mockVersionMirrors.On("GetVersionMirrorByID", mock.Anything, versionMirrorID).
				Return(&models.TerraformProviderVersionMirror{
					Metadata: models.ResourceMetadata{ID: versionMirrorID},
					GroupID:  groupID,
				}, nil)

			if test.expectErrorCode == "" {
				mockCaller.On("GetSubject").Return("testSubject").Maybe()

				mockPlatformMirrors.On("DeletePlatformMirror", mock.Anything, samplePlatformMirror).Return(nil)
			}

			dbClient := &db.Client{
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			logger, _ := logger.NewForTest()
			service := &service{logger: logger, dbClient: dbClient}

			err := service.DeleteProviderPlatformMirror(auth.WithCaller(ctx, mockCaller), &DeleteProviderPlatformMirrorInput{samplePlatformMirror})

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

func TestUploadInstallationPackage(t *testing.T) {
	validPackageData := "package-data"
	validPackageChecksum := "5f0ba01fa9b567c766a6f60c1ea51c691cbf04fa4939a118e6cde5c475319311" // Sum of above string.
	versionMirrorID := "version-mirror-1"
	groupID := "group-1"

	type testCase struct {
		input                   *UploadInstallationPackageInput
		versionMirror           *models.TerraformProviderVersionMirror
		authError               error
		expectErrorCode         errors.CodeType
		name                    string
		platformAlreadyMirrored bool
	}

	testCases := []testCase{
		{
			name: "successfully upload provider package",
			input: &UploadInstallationPackageInput{
				Data:            strings.NewReader(validPackageData),
				VersionMirrorID: versionMirrorID,
				OS:              "windows",
				Architecture:    "amd64",
			},
			versionMirror: &models.TerraformProviderVersionMirror{
				Metadata:        models.ResourceMetadata{ID: versionMirrorID},
				Type:            "null",
				SemanticVersion: "0.1.0",
				GroupID:         groupID,
				Digests: provider.Checksums{
					provider.GetPackageName(
						"null",
						"0.1.0",
						"windows",
						"amd64",
					): parseChecksumString(t, validPackageChecksum),
				},
			},
		},
		{
			name: "version mirror does not exist",
			input: &UploadInstallationPackageInput{
				VersionMirrorID: versionMirrorID,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have permission to upload to provider mirror",
			input: &UploadInstallationPackageInput{
				VersionMirrorID: versionMirrorID,
			},
			versionMirror: &models.TerraformProviderVersionMirror{
				GroupID: groupID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "platform is already mirrored",
			input: &UploadInstallationPackageInput{
				VersionMirrorID: versionMirrorID,
				OS:              "linux",
				Architecture:    "arm",
			},
			versionMirror: &models.TerraformProviderVersionMirror{
				GroupID: groupID,
			},
			platformAlreadyMirrored: true,
			expectErrorCode:         errors.EConflict,
		},
		{
			// Shouldn't happen unless a new platform was introduced and our data is out-of-date.
			name: "platform digest not found",
			input: &UploadInstallationPackageInput{
				VersionMirrorID: versionMirrorID,
				OS:              "linux",
				Architecture:    "arm",
			},
			versionMirror: &models.TerraformProviderVersionMirror{
				GroupID: groupID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			input: &UploadInstallationPackageInput{
				Data:            strings.NewReader("invalid-data"),
				VersionMirrorID: versionMirrorID,
				OS:              "windows",
				Architecture:    "amd64",
			},
			versionMirror: &models.TerraformProviderVersionMirror{
				Metadata:        models.ResourceMetadata{ID: versionMirrorID},
				Type:            "null",
				SemanticVersion: "0.1.0",
				GroupID:         groupID,
				Digests: provider.Checksums{
					provider.GetPackageName(
						"null",
						"0.1.0",
						"windows",
						"amd64",
					): parseChecksumString(t, validPackageChecksum),
				},
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockTransactions := db.NewMockTransactions(t)
			mockStore := NewMockTerraformProviderMirrorStore(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockVersionMirrors.On("GetVersionMirrorByID", mock.Anything, versionMirrorID).Return(test.versionMirror, nil)

			if test.versionMirror != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.CreateTerraformProviderMirrorPermission, mock.Anything).Return(test.authError)

				if test.authError == nil {
					result := &db.ProviderPlatformMirrorsResult{
						PageInfo: &pagination.PageInfo{},
					}

					if test.platformAlreadyMirrored {
						// Simply increase the totalCount to verify the logic works.
						result.PageInfo.TotalCount = 1
					}

					mockPlatformMirrors.On("GetPlatformMirrors", mock.Anything, &db.GetProviderPlatformMirrorsInput{
						PaginationOptions: &pagination.Options{
							First: ptr.Int32(0),
						},
						Filter: &db.TerraformProviderPlatformMirrorFilter{
							VersionMirrorID: &versionMirrorID,
							OS:              &test.input.OS,
							Architecture:    &test.input.Architecture,
						},
					}).Return(result, nil)
				}
			}

			if test.expectErrorCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				toCreate := &models.TerraformProviderPlatformMirror{
					VersionMirrorID: versionMirrorID,
					OS:              test.input.OS,
					Architecture:    test.input.Architecture,
				}

				createdPlatformMirror := &models.TerraformProviderPlatformMirror{
					Metadata:        models.ResourceMetadata{ID: "platform-mirror-id"},
					VersionMirrorID: versionMirrorID,
					OS:              test.input.OS,
					Architecture:    test.input.Architecture,
				}

				mockPlatformMirrors.On("CreatePlatformMirror", mock.Anything, toCreate).Return(createdPlatformMirror, nil)

				mockStore.On("UploadProviderPlatformPackage", mock.Anything, "platform-mirror-id", mock.Anything).Return(nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			dbClient := &db.Client{
				Transactions:                     mockTransactions,
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			service := &service{dbClient: dbClient, mirrorStore: mockStore}

			err := service.UploadInstallationPackage(auth.WithCaller(ctx, mockCaller), test.input)

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

func TestGetAvailableProviderVersions(t *testing.T) {
	groupID := "group-1"
	namespace := "some/group"

	type testCase struct {
		input           *GetAvailableProviderVersionsInput
		expectVersion   *models.TerraformProviderVersionMirror
		group           *models.Group
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return available provider versions",
			input: &GetAvailableProviderVersionsInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
			},
			group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			expectVersion: &models.TerraformProviderVersionMirror{
				SemanticVersion: "0.1.0",
			},
		},
		{
			name: "group does not exist",
			input: &GetAvailableProviderVersionsInput{
				GroupPath: namespace,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have permissions to view provider version mirrors",
			input: &GetAvailableProviderVersionsInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "provider fully qualified name is invalid",
			input: &GetAvailableProviderVersionsInput{
				RegistryHostname:  "invalid/host",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "no versions are available for provider",
			input: &GetAvailableProviderVersionsInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(namespace)).Return(test.group, nil)

			if test.group != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)

				result := &db.ProviderVersionMirrorsResult{
					VersionMirrors: []models.TerraformProviderVersionMirror{},
					PageInfo:       &pagination.PageInfo{},
				}

				if test.expectVersion != nil {
					result.VersionMirrors = append(result.VersionMirrors, *test.expectVersion)
					result.PageInfo.TotalCount = 1
				}

				sort := db.TerraformProviderVersionMirrorSortableFieldCreatedAtAsc
				mockVersionMirrors.On("GetVersionMirrors", mock.Anything, &db.GetProviderVersionMirrorsInput{
					Sort: &sort,
					Filter: &db.TerraformProviderVersionMirrorFilter{
						GroupID:           &groupID,
						RegistryHostname:  &test.input.RegistryHostname,
						RegistryNamespace: &test.input.RegistryNamespace,
						Type:              &test.input.Type,
						HasPackages:       ptr.Bool(true),
					},
				}).Return(result, nil).Maybe()
			}

			dbClient := &db.Client{
				Groups:                          mockGroups,
				TerraformProviderVersionMirrors: mockVersionMirrors,
			}

			service := &service{dbClient: dbClient}

			availableVersions, err := service.GetAvailableProviderVersions(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Len(t, availableVersions, 1)

			for k := range availableVersions {
				// Ensure we have the semantic version in the response.
				assert.Equal(t, test.expectVersion.SemanticVersion, k)
			}
		})
	}
}

func TestGetAvailableInstallationPackages(t *testing.T) {
	groupID := "group-1"
	namespace := "some/group"
	versionMirrorID := "version-mirror-1"

	type testCase struct {
		input                *GetAvailableInstallationPackagesInput
		expectPlatformMirror *models.TerraformProviderPlatformMirror
		versionMirror        *models.TerraformProviderVersionMirror
		group                *models.Group
		authError            error
		name                 string
		expectErrorCode      errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully return a list of available installation packages",
			input: &GetAvailableInstallationPackagesInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
				SemanticVersion:   "0.1.0",
			},
			group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			versionMirror: &models.TerraformProviderVersionMirror{
				Metadata:          models.ResourceMetadata{ID: versionMirrorID},
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				SemanticVersion:   "0.1.0",
				Digests: provider.Checksums{
					provider.GetPackageName(
						"aws",
						"0.1.0",
						"windows",
						"amd64",
					): parseChecksumString(t, "cbc04edd7a868cf2713fd301ca357adb940003d0c02f5351c987e2ab6ead2a46"),
				},
			},
			expectPlatformMirror: &models.TerraformProviderPlatformMirror{
				Metadata:     models.ResourceMetadata{ID: "platform-mirror-id"},
				OS:           "windows",
				Architecture: "amd64",
			},
		},
		{
			name: "group does not exist",
			input: &GetAvailableInstallationPackagesInput{
				GroupPath: namespace,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject does not have permission to view version mirror",
			input: &GetAvailableInstallationPackagesInput{
				GroupPath: namespace,
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "could not parse provider",
			input: &GetAvailableInstallationPackagesInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "-invalid-",
				GroupPath:         namespace,
				SemanticVersion:   "0.1.0",
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "version mirror does not exist",
			input: &GetAvailableInstallationPackagesInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
				SemanticVersion:   "0.1.0",
			},
			group:           &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "no platforms are mirrored for version mirror",
			input: &GetAvailableInstallationPackagesInput{
				RegistryHostname:  "registry.terraform.io",
				RegistryNamespace: "hashicorp",
				Type:              "aws",
				GroupPath:         namespace,
				SemanticVersion:   "0.1.0",
			},
			group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}},
			versionMirror: &models.TerraformProviderVersionMirror{
				Metadata: models.ResourceMetadata{ID: versionMirrorID},
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockStore := NewMockTerraformProviderMirrorStore(t)
			mockGroups := db.NewMockGroups(t)
			mockPlatformMirrors := db.NewMockTerraformProviderPlatformMirrors(t)
			mockVersionMirrors := db.NewMockTerraformProviderVersionMirrors(t)

			mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(namespace)).Return(test.group, nil)

			if test.group != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.TerraformProviderMirrorModelType, mock.Anything).Return(test.authError)

				versionsResult := &db.ProviderVersionMirrorsResult{
					VersionMirrors: []models.TerraformProviderVersionMirror{},
					PageInfo:       &pagination.PageInfo{},
				}

				if test.versionMirror != nil {
					versionsResult.VersionMirrors = append(versionsResult.VersionMirrors, *test.versionMirror)
					versionsResult.PageInfo.TotalCount = 1
				}

				mockVersionMirrors.On("GetVersionMirrors", mock.Anything, &db.GetProviderVersionMirrorsInput{
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
					Filter: &db.TerraformProviderVersionMirrorFilter{
						GroupID:           &groupID,
						RegistryHostname:  &test.input.RegistryHostname,
						RegistryNamespace: &test.input.RegistryNamespace,
						Type:              &test.input.Type,
						SemanticVersion:   &test.input.SemanticVersion,
						HasPackages:       ptr.Bool(true),
					},
				}).Return(versionsResult, nil).Maybe()

				platformsResult := &db.ProviderPlatformMirrorsResult{
					PlatformMirrors: []models.TerraformProviderPlatformMirror{},
					PageInfo:        &pagination.PageInfo{},
				}

				if test.expectPlatformMirror != nil {
					platformsResult.PlatformMirrors = append(platformsResult.PlatformMirrors, *test.expectPlatformMirror)
					platformsResult.PageInfo.TotalCount = 1
				}

				mockPlatformMirrors.On("GetPlatformMirrors", mock.Anything, &db.GetProviderPlatformMirrorsInput{
					Filter: &db.TerraformProviderPlatformMirrorFilter{
						VersionMirrorID: &versionMirrorID,
					},
				}).Return(platformsResult, nil).Maybe()
			}

			if test.expectErrorCode == "" {
				mockStore.On("GetProviderPlatformPackagePresignedURL", mock.Anything, mock.Anything).Return("http://signed.url", nil)
			}

			dbClient := &db.Client{
				Groups:                           mockGroups,
				TerraformProviderVersionMirrors:  mockVersionMirrors,
				TerraformProviderPlatformMirrors: mockPlatformMirrors,
			}

			service := &service{dbClient: dbClient, mirrorStore: mockStore}

			availablePackages, err := service.GetAvailableInstallationPackages(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.Len(t, availablePackages, 1)

			for k, v := range availablePackages {
				// Ensure we have the platform.
				assert.Equal(t, fmt.Sprintf("%s_%s", test.expectPlatformMirror.OS, test.expectPlatformMirror.Architecture), k)

				m, ok := v.(map[string]any)
				if !ok {
					require.Fail(t, "unexpected type for map value")
				}

				// Expecting a presigned URL and a hash.
				assert.Len(t, m, 2)
			}
		})
	}
}

// parseChecksumString decodes a checksum string and requires a nil error.
func parseChecksumString(t *testing.T, checksum string) []byte {
	parsed, err := hex.DecodeString(checksum)
	require.Nil(t, err)

	return parsed
}
