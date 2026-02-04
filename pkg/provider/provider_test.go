package provider

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testHashicorpGPGKey used for testing in this module.
	// GPG will require a proper public key.
	testHashicorpGPGKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
ZF5q4h4I33PSGDdSvGXn9UMY5Isjpg==
=7pIB
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

type fakeDiscoverer struct{}

func (f *fakeDiscoverer) DiscoverServiceURL(_ svchost.Hostname, _ string) (*url.URL, error) {
	return &url.URL{Scheme: "https", Host: "test.io", Path: "/v1/providers/"}, nil
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
			resolver := &registryClient{httpClient: client, discovery: &fakeDiscoverer{}}

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
			resolver := &registryClient{httpClient: client, discovery: &fakeDiscoverer{}}

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
			resolver := &registryClient{httpClient: client, discovery: &fakeDiscoverer{}}

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
			resolver := &registryClient{httpClient: client, discovery: &fakeDiscoverer{}}

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
