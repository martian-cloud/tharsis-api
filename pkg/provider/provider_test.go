package provider

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	// testHashicorpGPGKey used for testing in this module.
	// GPG will require a proper public key.
	testHashicorpGPGKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GCHx/5nGBdDwoxEsXqpI3IUqPb6FDg==
=wtp+
-----END PGP PUBLIC KEY BLOCK-----`

	// testAWSProviderChecksumsData contains actual signed checksums data
	// for the aws 5.2.0 provider as a base64 encoded string to ease
	// testing in this module.
	testAWSProviderChecksumsData = `MGU0ODQ0OWU5ZjI5YjY0NjYzZTdmZjY0MWEzYmExZGE0MzQ2MDg0NjBjMzNhMjBiZGI0NWVmZWIx
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

	// testAWSProviderChecksumsSignature contains the actual detached signature
	// needed to verify the checksum data as a base64 encoded string.
	testAWSProviderChecksumsSignature = `wsFzBAABCAAnBQJkg4UzCRDIIMbVzSerhxYhBDdOx1tIWRNgSoMcx8ggxtXNJ6uHAABwow/8CK+K
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

const testBadData = "YmFkIGRhdGE=" // "bad data" in base64

type roundTripFunc func(r *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestProviderString(t *testing.T) {
	t.Run("public registry omits hostname", func(t *testing.T) {
		p := &Provider{Hostname: TerraformPublicRegistryHost, Namespace: "hashicorp", Type: "aws"}
		assert.Equal(t, "hashicorp/aws", p.String())
	})

	t.Run("private registry includes hostname", func(t *testing.T) {
		p := &Provider{Hostname: "private.registry.io", Namespace: "myorg", Type: "custom"}
		assert.Equal(t, "private.registry.io/myorg/custom", p.String())
	})
}

func TestWithToken(t *testing.T) {
	opt := WithToken("test-token")
	opts := &requestOptions{}
	opt(opts)
	assert.Equal(t, "test-token", opts.token)
}

func TestNewRegistryClient(t *testing.T) {
	client := &http.Client{}
	resolver := NewRegistryClient(client)
	assert.NotNil(t, resolver)
}

func TestListVersions(t *testing.T) {
	provider := &Provider{Hostname: "test.io", Namespace: "hashicorp", Type: "aws"}

	type testCase struct {
		name           string
		responseBody   string
		responseStatus int
		expectError    string
		expectCount    int
	}

	testCases := []testCase{
		{
			name:           "success",
			responseBody:   `{"versions":[{"version":"1.0.0","platforms":[{"os":"linux","arch":"amd64"}]}]}`,
			responseStatus: http.StatusOK,
			expectCount:    1,
		},
		{
			name:           "error status",
			responseBody:   "error",
			responseStatus: http.StatusBadRequest,
			expectError:    "400",
		},
		{
			name:           "no versions",
			responseBody:   `{"versions":[]}`,
			responseStatus: http.StatusOK,
			expectError:    "no versions found",
		},
		{
			name:           "warnings",
			responseBody:   `{"versions":[],"warnings":["deprecated"]}`,
			responseStatus: http.StatusOK,
			expectError:    "warnings",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestHTTPClient(func(_ *http.Request) *http.Response {
				return &http.Response{StatusCode: tc.responseStatus, Body: io.NopCloser(bytes.NewReader([]byte(tc.responseBody)))}
			})

			mockDiscovery := NewMockServiceDiscoverer(t)
			mockDiscovery.On("DiscoverTFEServices", mock.Anything, provider.Hostname).Return(&TFEServices{
				Services: map[ServiceID]*url.URL{
					ProvidersServiceID: &url.URL{
						Scheme: "https",
						Host:   "test.io",
						Path:   "/v1/tfe",
					},
				},
			}, nil)

			resolver := &registryClient{httpClient: client, discovery: mockDiscovery}

			versions, err := resolver.ListVersions(t.Context(), provider)
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
				assert.Len(t, versions, tc.expectCount)
			}
		})
	}
}

func TestGetPackageInfo(t *testing.T) {
	provider := &Provider{Hostname: "test.io", Namespace: "hashicorp", Type: "aws"}

	type testCase struct {
		name           string
		responseBody   string
		responseStatus int
		expectError    string
	}

	testCases := []testCase{
		{
			name:           "success",
			responseBody:   `{"shasums_url":"http://test/sums","shasums_signature_url":"http://test/sig","signing_keys":{"gpg_public_keys":[{"ascii_armor":"key"}]}}`,
			responseStatus: http.StatusOK,
		},
		{
			name:           "error status",
			responseBody:   "not found",
			responseStatus: http.StatusNotFound,
			expectError:    "404",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestHTTPClient(func(_ *http.Request) *http.Response {
				return &http.Response{StatusCode: tc.responseStatus, Body: io.NopCloser(bytes.NewReader([]byte(tc.responseBody)))}
			})

			mockDiscovery := NewMockServiceDiscoverer(t)
			mockDiscovery.On("DiscoverTFEServices", mock.Anything, provider.Hostname).Return(&TFEServices{
				Services: map[ServiceID]*url.URL{
					ProvidersServiceID: &url.URL{
						Scheme: "https",
						Host:   "test.io",
						Path:   "/v1/tfe",
					},
				},
			}, nil)

			resolver := &registryClient{httpClient: client, discovery: mockDiscovery}

			info, err := resolver.GetPackageInfo(t.Context(), provider, "1.0.0", "linux", "amd64")
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "http://test/sums", info.SHASumsURL)
				assert.Len(t, info.GPGASCIIArmors, 1)
			}
		})
	}
}

func TestDownloadPackage(t *testing.T) {
	type testCase struct {
		name           string
		responseStatus int
		expectError    string
	}

	testCases := []testCase{
		{
			name:           "success",
			responseStatus: http.StatusOK,
		},
		{
			name:           "error status",
			responseStatus: http.StatusForbidden,
			expectError:    "403",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte("package-content")
			client := newTestHTTPClient(func(_ *http.Request) *http.Response {
				return &http.Response{StatusCode: tc.responseStatus, Body: io.NopCloser(bytes.NewReader(content)), ContentLength: int64(len(content))}
			})
			resolver := &registryClient{httpClient: client}

			body, size, err := resolver.DownloadPackage(t.Context(), "http://test/pkg.zip")
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
				defer body.Close()
				assert.Equal(t, int64(len(content)), size)
			}
		})
	}
}

func TestGetChecksums(t *testing.T) {
	type testCase struct {
		name            string
		checksumData    string
		signatureData   string
		gpgKey          string
		checksumStatus  int
		signatureStatus int
		expectError     string
		expectCount     int
	}

	testCases := []testCase{
		{
			name:            "success",
			checksumData:    testAWSProviderChecksumsData,
			signatureData:   testAWSProviderChecksumsSignature,
			gpgKey:          testHashicorpGPGKey,
			checksumStatus:  http.StatusOK,
			signatureStatus: http.StatusOK,
			expectCount:     15,
		},
		{
			name:            "checksum fetch error",
			checksumStatus:  http.StatusBadRequest,
			signatureStatus: http.StatusOK,
			expectError:     "400",
		},
		{
			name:            "signature fetch error",
			checksumData:    testAWSProviderChecksumsData,
			checksumStatus:  http.StatusOK,
			signatureStatus: http.StatusBadRequest,
			expectError:     "400",
		},
		{
			name:            "signature mismatch",
			checksumData:    testBadData,
			signatureData:   testAWSProviderChecksumsSignature,
			gpgKey:          testHashicorpGPGKey,
			checksumStatus:  http.StatusOK,
			signatureStatus: http.StatusOK,
			expectError:     "failed to verify checksum signature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callCount := 0
			client := newTestHTTPClient(func(_ *http.Request) *http.Response {
				callCount++
				if callCount == 1 {
					data, _ := base64.StdEncoding.DecodeString(tc.checksumData)
					return &http.Response{StatusCode: tc.checksumStatus, Body: io.NopCloser(bytes.NewReader(data))}
				}
				data, _ := base64.StdEncoding.DecodeString(tc.signatureData)
				return &http.Response{StatusCode: tc.signatureStatus, Body: io.NopCloser(bytes.NewReader(data))}
			})
			resolver := &registryClient{httpClient: client}

			packageInfo := &PackageInfo{
				SHASumsURL:          "http://test/sums",
				SHASumsSignatureURL: "http://test/sig",
				GPGASCIIArmors:      []string{tc.gpgKey},
			}

			checksums, err := resolver.GetChecksums(t.Context(), packageInfo)
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tc.expectError), "expected %q in error: %v", tc.expectError, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, checksums, tc.expectCount)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	type testCase struct {
		name        string
		hostname    string
		namespace   string
		pType       string
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid provider",
			hostname:    TerraformPublicRegistryHost,
			namespace:   "hashicorp",
			pType:       "aws",
			expectError: false,
		},
		{
			name:        "invalid namespace",
			hostname:    TerraformPublicRegistryHost,
			namespace:   "INVALID!",
			pType:       "aws",
			expectError: true,
		},
		{
			name:        "invalid type",
			hostname:    TerraformPublicRegistryHost,
			namespace:   "hashicorp",
			pType:       "INVALID!",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewProvider(tc.hostname, tc.namespace, tc.pType)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.hostname, p.Hostname)
				assert.Equal(t, tc.namespace, p.Namespace)
				assert.Equal(t, tc.pType, p.Type)
			}
		})
	}
}

func TestGetPlatformForVersion(t *testing.T) {
	versions := []VersionInfo{
		{Version: "1.0.0", Platforms: []Platform{{OS: "linux", Arch: "amd64"}}},
		{Version: "2.0.0", Platforms: []Platform{{OS: "darwin", Arch: "arm64"}}},
	}

	type testCase struct {
		name        string
		version     string
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "found",
			version:     "1.0.0",
			expectError: false,
		},
		{
			name:        "version not found",
			version:     "3.0.0",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := GetPlatformForVersion(tc.version, versions)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "linux", p.OS)
			}
		})
	}
}

func TestGetPackageName(t *testing.T) {
	name := GetPackageName("aws", "5.0.0", "linux", "amd64")
	assert.Equal(t, "terraform-provider-aws_5.0.0_linux_amd64.zip", name)
}

func TestFindLatestVersion(t *testing.T) {
	testCases := []struct {
		name          string
		versions      []VersionInfo
		expectVersion string
		expectError   string
	}{
		{
			name: "found latest version 1.5.0",
			versions: []VersionInfo{
				{Version: "0.2.0"},
				{Version: "1.4.0"},
				{Version: "1.5.0-pre"},
				{Version: "1.5.0"},
				{Version: "1.5.0-rc"},
			},
			expectVersion: "1.5.0",
		},
		{
			name:        "empty versions list",
			versions:    []VersionInfo{},
			expectError: "no versions provided",
		},
		{
			name: "cannot parse version",
			versions: []VersionInfo{
				{Version: "0.1.0"},
				{Version: "invalid-version"},
			},
			expectError: "failed to parse provider version",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FindLatestVersion(tc.versions)

			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectVersion, result)
		})
	}
}

func TestChecksums_ZipHash(t *testing.T) {
	checksums := Checksums{
		"terraform-provider-aws_5.0.0_linux_amd64.zip": {0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a},
	}

	t.Run("returns hash for existing file", func(t *testing.T) {
		hash, ok := checksums.GetZipHash("terraform-provider-aws_5.0.0_linux_amd64.zip")
		assert.True(t, ok)
		assert.Equal(t, "zh:abcdef123456789abcdef0123456789aabcdef123456789abcdef0123456789a", hash)
	})

	t.Run("returns false for missing file", func(t *testing.T) {
		hash, ok := checksums.GetZipHash("nonexistent.zip")
		assert.False(t, ok)
		assert.Empty(t, hash)
	})
}
